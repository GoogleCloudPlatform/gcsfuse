// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Provides integration tests for Readdirplus
package readdirplus

import (
	"context"
	"io"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/storage"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/require"
)

const (
	testDirName                         = "dirForReaddirplusTest"
	targetDirName                       = "target_dir"
	logFileNameForMountedDirectoryTests = "/tmp/readdirplus_logs/log.json"
)

var (
	storageClient *storage.Client
	ctx           context.Context
	testDirPath   string
	mountFunc     func([]string) error
	// mount directory is where our tests run.
	mountDir string
	// root directory is the directory to be unmounted.
	rootDir string
)

func loadLogLines(reader io.Reader) ([]string, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return strings.Split(string(content), "\n"), nil
}

// validateLogsForReaddirplus checks that ReadDirPlus was called and ReadDir was not.
// It also checks that LookUpInode is not called when dentry cache is enabled.
func validateLogsForReaddirplus(t *testing.T, logFile string, dentryCacheEnabled bool, startTime, endTime time.Time) {
	t.Helper()

	logForReadDirPlus := "ReadDirPlus ("
	logForReadDir := "ReadDir ("
	logForLookUpInode := "LookUpInode ("

	file, err := os.Open(logFile)
	require.NoError(t, err, "Failed to open log file")
	defer file.Close()

	logLines, err := loadLogLines(file)
	require.NoError(t, err, "Failed to read log file")

	foundReadDirPlus := false
	foundReadDir := false
	foundLookUpInode := false
	for _, line := range logLines {
		logEntry, err := read_logs.ParseJsonLogLineIntoLogEntryStruct(line) // Assuming read_logs can parse general log lines too or a more generic parser is available.
		// If parsing fails, it might be a non-JSON line or a different structured log.
		// For this specific message, we expect it to be in the "Message" field of a structured log.

		if err == nil && logEntry != nil {
			// Check if the log entry's timestamp is within the expected window.
			if (logEntry.Timestamp.After(startTime) || logEntry.Timestamp.Equal(startTime)) &&
				(logEntry.Timestamp.Before(endTime) || logEntry.Timestamp.Equal(endTime)) {
				if strings.Contains(logEntry.Message, logForReadDirPlus) {
					foundReadDirPlus = true
				}
				if strings.Contains(logEntry.Message, logForReadDir) {
					foundReadDir = true
				}
				if strings.Contains(logEntry.Message, logForLookUpInode) {
					foundLookUpInode = true
				}
			}
		}
	}

	require.True(t, foundReadDirPlus, "ReadDirPlus not called")
	require.False(t, foundReadDir, "ReadDir called unexpectedly")
	if dentryCacheEnabled {
		require.False(t, foundLookUpInode, "LookUpInode called unexpectedly")
	} else {
		require.True(t, foundLookUpInode, "LookUpInode not called")
	}
}

func mountGCSFuseAndSetupTestDir(t *testing.T, flags []string, testDirName string) {
	setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)
	setup.SetMntDir(mountDir)
	testDirPath = setup.SetupTestDirectory(testDirName)
}

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()
	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	// Create common storage client to be used in test.
	ctx = context.Background()
	closeStorageClient := client.CreateStorageClientWithCancel(&ctx, &storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	if setup.MountedDirectory() != "" {
		mountDir = setup.MountedDirectory()
		setup.SetLogFile(logFileNameForMountedDirectoryTests)
		// Run tests for mounted directory if the flag is set.
		os.Exit(m.Run())
	}
	// Else run tests for testBucket.
	// Set up test directory.
	setup.SetUpTestDirForTestBucketFlag()

	// Save mount and root directory variables.
	mountDir, rootDir = setup.MntDir(), setup.MntDir()

	log.Println("Running static mounting tests...")
	mountFunc = static_mounting.MountGcsfuseWithStaticMounting
	successCode := m.Run()

	os.Exit(successCode)
}
