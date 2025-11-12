// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
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
	"path"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
	"github.com/stretchr/testify/require"
)

const (
	testDirName   = "dirForReaddirplusTest"
	targetDirName = "target_dir"
	GKETempDir    = "/gcsfuse-tmp"
	// // TODO: clean this up when GKE test migration completes.
	OldGKElogFilePath = "/tmp/readdirplus_logs/log.json"
)

var (
	testEnv   env
	mountFunc func(*test_suite.TestConfig, []string) error
)

type env struct {
	storageClient *storage.Client
	ctx           context.Context
	testDirPath   string
	cfg           *test_suite.TestConfig
	bucketType    string
}

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

func mountGCSFuseAndSetupTestDir(flags []string, ctx context.Context, storageClient *storage.Client) {
	setup.MountGCSFuseWithGivenMountWithConfigFunc(testEnv.cfg, flags, mountFunc)
	testEnv.testDirPath = client.SetupTestDirectory(ctx, storageClient, testDirName)
}

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// 1. Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.ReadDirPlus) == 0 {
		log.Println("No configuration found for readdirplus tests in config. Using flags instead.")
		// Populate the config manually.
		cfg.ReadDirPlus = make([]test_suite.TestConfig, 1)
		cfg.ReadDirPlus[0].TestBucket = setup.TestBucket()
		cfg.ReadDirPlus[0].GKEMountedDirectory = setup.MountedDirectory()
		cfg.ReadDirPlus[0].LogFile = setup.LogFile()
		cfg.ReadDirPlus[0].Configs = make([]test_suite.ConfigItem, 2)
		cfg.ReadDirPlus[0].Configs[0].Flags = []string{
			"--implicit-dirs --experimental-enable-readdirplus --experimental-enable-dentry-cache --log-file=/gcsfuse-tmp/TestReaddirplusWithDentryCacheTest.log --log-severity=TRACE",
		}
		cfg.ReadDirPlus[0].Configs[0].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.ReadDirPlus[0].Configs[0].Run = "TestReaddirplusWithDentryCacheTest"

		cfg.ReadDirPlus[0].Configs[1].Flags = []string{
			"--implicit-dirs --experimental-enable-readdirplus --log-file=/gcsfuse-tmp/TestReaddirplusWithoutDentryCacheTest.log --log-severity=TRACE",
		}
		cfg.ReadDirPlus[0].Configs[1].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.ReadDirPlus[0].Configs[1].Run = "TestReaddirplusWithoutDentryCacheTest"
	}

	testEnv.ctx = context.Background()
	testEnv.cfg = &cfg.ReadDirPlus[0]
	testEnv.bucketType = setup.TestEnvironment(testEnv.ctx, testEnv.cfg)

	// 2. Create storage client before running tests.
	var err error
	testEnv.storageClient, err = client.CreateStorageClient(testEnv.ctx)
	if err != nil {
		log.Printf("Error creating storage client: %v\n", err)
		os.Exit(1)
	}
	defer testEnv.storageClient.Close()

	// 3. To run mountedDirectory tests, we need both testBucket and mountedDirectory
	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		os.Exit(setup.RunTestsForMountedDirectory(testEnv.cfg.GKEMountedDirectory, m))
	}

	// Run tests for testBucket
	// Set up test directory.
	setup.SetUpTestDirForTestBucket(testEnv.cfg)
	// Override GKE specific paths with GCSFuse paths if running in GCE environment.
	setup.OverrideFilePathsInFlagSet(testEnv.cfg, setup.TestDir())

	log.Println("Running static mounting tests...")
	mountFunc = static_mounting.MountGcsfuseWithStaticMountingWithConfigFile
	successCode := m.Run()

	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(cfg.ReadDirPlus[0].TestBucket, testDirName))
	os.Exit(successCode)
}
