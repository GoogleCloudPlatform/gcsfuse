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

package inactive_stream_timeout

import (
	"context"
	"fmt"
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
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/dynamic_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/only_dir_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/require"
)

const (
	kTestDirName                         = "inactiveReadTimeoutTest"
	kOnlyDirMounted                      = "onlyDirInactiveReadTimeout"
	kFileSize                            = 10 * 1024 * 1024 // 10 MiB
	kChunkSizeToRead                     = 128 * 1024       // 64 KiB
	kTestFileName                        = "foo"
	kDefaultInactiveReadTimeoutInSeconds = 1 // A short timeout for testing
	kLogFileNameForMountedDirectoryTests = "/tmp/inactive_stream_timeout_logs/log.json"
	kHTTP1ClientProtocol                 = "http1"
	kGRPCClientProtocol                  = "grpc"
)

var (
	// Stores test directory path in the mounted path.
	gTestDirPath string

	// Used to run the test for different types of mount by modify this function.
	gMountFunc func([]string) error

	// Actual mounted directory, for dynamic mount it becomes gRootDir/<bucket_name>
	gMountDir string

	// Root directory which is mounted by gcsfuse.
	gRootDir string

	// Clients to create the object in GCS.
	gStorageClient *storage.Client
	gCtx           context.Context
)

type gcsfuseTestFlags struct {
	cliFlags            []string
	inactiveReadTimeout time.Duration
	fileName            string
	clientProtocol      string
}

func setupFile(ctx context.Context, storageClient *storage.Client, fileName string, fileSize int, t *testing.T) string {
	t.Helper()
	client.SetupFileInTestDirectory(ctx, storageClient, kTestDirName, fileName, int64(fileSize), t)
	return path.Join(gTestDirPath, fileName)
}

func loadLogLines(reader io.Reader) ([]string, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return strings.Split(string(content), "\n"), nil
}

// validateInactiveReaderClosedLog checks if the "Closing reader for object ... due to inactivity"
// log message is present (or absent) for the given objectName, b/w the [startTime, endTime] interval.
// Also expects based on the shouldBePresent value.
func validateInactiveReaderClosedLog(t *testing.T, logFile, objectName string, shouldBePresent bool, startTime, endTime time.Time) {
	t.Helper()
	// Be specific about the object name in the expected message.
	expectedMsgSubstring := fmt.Sprintf("Closing reader for object %q due to inactivity.", objectName)

	file, err := os.Open(logFile)
	require.NoError(t, err, "Failed to open log file")
	defer file.Close()

	logLines, err := loadLogLines(file)
	require.NoError(t, err, "Failed to read log file")

	found := false
	for _, line := range logLines {
		logEntry, err := read_logs.ParseJsonLogLineIntoLogEntryStruct(line) // Assuming read_logs can parse general log lines too or a more generic parser is available.
		// If parsing fails, it might be a non-JSON line or a different structured log.
		// For this specific message, we expect it to be in the "Message" field of a structured log.

		if err == nil && logEntry != nil {
			// Check if the log entry's timestamp is within the expected window.
			if (logEntry.Timestamp.After(startTime) || logEntry.Timestamp.Equal(startTime)) &&
				(logEntry.Timestamp.Before(endTime) || logEntry.Timestamp.Equal(endTime)) {
				if strings.Contains(logEntry.Message, expectedMsgSubstring) {
					found = true
					break
				}
			}
		}
	}

	if shouldBePresent {
		require.True(t, found, "Expected log message substring '%s' not found between %v and %v", expectedMsgSubstring, startTime, endTime)
	} else {
		require.False(t, found, "Unexpected log message substring '%s' found between %v and %v", expectedMsgSubstring, startTime, endTime)
	}
}

func mountGCSFuseAndSetupTestDir(ctx context.Context, flags []string, storageClient *storage.Client, testDirName string) {
	setup.MountGCSFuseWithGivenMountFunc(flags, gMountFunc)
	setup.SetMntDir(gMountDir)
	gTestDirPath = client.SetupTestDirectory(ctx, storageClient, testDirName)
}

// createConfigFile generate mount config.yaml.
func createConfigFile(flags *gcsfuseTestFlags) string {
	mountConfig := map[string]any{
		"read": map[string]any{
			"inactive-stream-timeout": flags.inactiveReadTimeout.String(),
		},
		"gcs-connection": map[string]any{
			"client-protocol": flags.clientProtocol,
		},
		"logging": map[string]any{
			"file-path": setup.LogFile(),
			"format":    "json", // Ensure JSON logs for easier parsing
		},
	}
	return setup.YAMLConfigFile(mountConfig, flags.fileName)
}

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// Create common storage client to be used in test.
	gCtx = context.Background()
	closeStorageClient := client.CreateStorageClientWithCancel(&gCtx, &gStorageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	if setup.MountedDirectory() != "" {
		gMountDir = setup.MountedDirectory()
		setup.SetLogFile(kLogFileNameForMountedDirectoryTests)
		// Run tests for mounted directory if the flag is set.
		os.Exit(m.Run())
	}

	// Else run tests for testBucket.
	setup.SetUpTestDirForTestBucketFlag()

	log.Println("Running static mounting tests...")
	// MountDir and RootDir will be same for static mount.
	gMountDir, gRootDir = setup.MntDir(), setup.MntDir()
	gMountFunc = static_mounting.MountGcsfuseWithStaticMounting
	successCode := m.Run()

	if successCode == 0 {
		log.Println("Running dynamic mounting tests...")
		// For dyanamic mount - gMountDir = gRootDir + <bucket_name>
		gMountDir = path.Join(setup.MntDir(), setup.TestBucket())
		gMountFunc = dynamic_mounting.MountGcsfuseWithDynamicMounting
		successCode = m.Run()
	}

	if successCode == 0 {
		log.Println("Running only dir mounting tests...")
		setup.SetOnlyDirMounted(kOnlyDirMounted + "/")
		gMountDir = gRootDir
		gMountFunc = only_dir_mounting.MountGcsfuseWithOnlyDir
		successCode = m.Run()
		setup.CleanupDirectoryOnGCS(gCtx, gStorageClient, path.Join(setup.TestBucket(), setup.OnlyDirMounted(), kTestDirName))
	}

	setup.CleanupDirectoryOnGCS(gCtx, gStorageClient, path.Join(setup.TestBucket(), kTestDirName))
	os.Exit(successCode)
}
