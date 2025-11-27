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
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
	"github.com/stretchr/testify/require"
)

const (
	kTestDirName                         = "inactiveReadTimeout"
	kOnlyDirMounted                      = "onlyDirInactiveReadTimeout"
	kFileSize                            = 10 * 1024 * 1024 // 10 MiB
	kChunkSizeToRead                     = 128 * 1024       // 128 KiB
	kTestFileName                        = "foo"
	kDefaultInactiveReadTimeoutInSeconds = 1 // A short timeout for testing
	GKETempDir                           = "/gcsfuse-tmp"
	OldGKElogFilePath                    = "/tmp/inactive_stream_timeout_logs/log.json"
)

type env struct {
	storageClient *storage.Client
	ctx           context.Context
	testDirPath   string
	cfg           *test_suite.TestConfig
	bucketType    string
}

var (
	testEnv   env
	mountFunc func(*test_suite.TestConfig, []string) error
	// mount directory is where our tests run.
	mountDir string
	// root directory is the directory to be unmounted.
	rootDir string
)

func SetupNestedTestDir(path string, permission os.FileMode, t *testing.T) {
	err := os.MkdirAll(path, permission)
	require.NoError(t, err)
}

func mountGCSFuseAndSetupTestDir(flags []string, ctx context.Context, storageClient *storage.Client) {
	setup.MountGCSFuseWithGivenMountWithConfigFunc(testEnv.cfg, flags, mountFunc)
	// In case of GKE, the test directory is not created by the test.
	if testEnv.cfg.GKEMountedDirectory != "" {
		setup.SetMntDir(testEnv.cfg.GKEMountedDirectory)
	}
	testEnv.testDirPath = client.SetupTestDirectory(ctx, storageClient, kTestDirName)
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

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// 1. read config file
	configFile := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(configFile.InactiveStreamTimeout) == 0 {
		log.Println("No configuration found for inactive_stream_timeout tests in config. Using default flags.")
		configFile.InactiveStreamTimeout = make([]test_suite.TestConfig, 1)
		testEnv.cfg = &configFile.InactiveStreamTimeout[0]
		testEnv.cfg.TestBucket = setup.TestBucket()
		testEnv.cfg.LogFile = setup.LogFile()
		testEnv.cfg.GKEMountedDirectory = setup.MountedDirectory()

		testEnv.cfg.Configs = make([]test_suite.ConfigItem, 2)
		testEnv.cfg.Configs[0].Flags = []string{
			"--read-inactive-stream-timeout=1s --client-protocol=http1 --log-format=json --log-file=/gcsfuse-tmp/TestTimeoutEnabledSuite.log",
			"--read-inactive-stream-timeout=1s --client-protocol=grpc --log-format=json --log-file=/gcsfuse-tmp/TestTimeoutEnabledSuite.log",
		}
		testEnv.cfg.Configs[0].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		testEnv.cfg.Configs[0].Run = "TestTimeoutEnabledSuite"

		testEnv.cfg.Configs[1].Flags = []string{
			"--read-inactive-stream-timeout=0s --client-protocol=http1 --log-format=json --log-file=/gcsfuse-tmp/TestTimeoutDisabledSuite.log",
		}
		testEnv.cfg.Configs[1].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		testEnv.cfg.Configs[1].Run = "TestTimeoutDisabledSuite"
	}
	testEnv.cfg = &configFile.InactiveStreamTimeout[0]
	testEnv.ctx = context.Background()
	testEnv.bucketType = setup.TestEnvironment(testEnv.ctx, testEnv.cfg)

	// 2. Create common storage client to be used in test.
	var err error
	testEnv.storageClient, err = client.CreateStorageClient(testEnv.ctx)
	if err != nil {
		log.Fatalf("client.CreateStorageClient: %v", err)
	}
	defer testEnv.storageClient.Close()

	// 3. To run mountedDirectory tests, we need both testBucket and mountedDirectory
	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		mountDir = testEnv.cfg.GKEMountedDirectory
		os.Exit(setup.RunTestsForMountedDirectory(testEnv.cfg.GKEMountedDirectory, m))
	}

	// Run tests for testBucket
	// Set up test directory.
	setup.SetUpTestDirForTestBucket(testEnv.cfg)
	// Override GKE specific paths with GCSFuse paths if running in GCE environment.
	setup.OverrideFilePathsInFlagSet(testEnv.cfg, setup.TestDir())
	// Save mount and root directory variables.
	mountDir, rootDir = setup.MntDir(), setup.MntDir()
	log.Println("Running static mounting tests...")
	mountFunc = static_mounting.MountGcsfuseWithStaticMountingWithConfigFile
	successCode := m.Run()

	if successCode == 0 {
		log.Println("Running dynamic mounting tests...")
		// Save mount directory variable to have path of bucket to run tests.
		mountDir = path.Join(setup.MntDir(), setup.TestBucket())
		mountFunc = dynamic_mounting.MountGcsfuseWithDynamicMountingWithConfig
		successCode = m.Run()
	}

	if successCode == 0 {
		log.Println("Running only dir mounting tests...")
		setup.SetOnlyDirMounted(kOnlyDirMounted + "/")
		mountDir = rootDir
		mountFunc = only_dir_mounting.MountGcsfuseWithOnlyDirWithConfigFile
		successCode = m.Run()
		setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(setup.TestBucket(), setup.OnlyDirMounted(), "TestTimeoutEnabledSuite"))
		setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(setup.TestBucket(), setup.OnlyDirMounted(), "TestTimeoutDisabledSuite"))
	}

	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(setup.TestBucket(), kTestDirName))
	os.Exit(successCode)
}
