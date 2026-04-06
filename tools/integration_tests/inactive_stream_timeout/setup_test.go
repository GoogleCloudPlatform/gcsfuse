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
	"bufio"
	"context"
	"fmt"
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
)

const (
	kTestDirName                         = "inactiveReadTimeout"
	kOnlyDirMounted                      = "onlyDirInactiveReadTimeout"
	kFileSize                            = 10 * 1024 * 1024 // 10 MiB
	kChunkSizeToRead                     = 128 * 1024       // 128 KiB
	kDefaultInactiveReadTimeoutInSeconds = 1                // A short timeout for testing
	GKETempDir                           = "/gcsfuse-tmp"
	OldGKElogFilePath                    = "/tmp/inactive_stream_timeout_logs/log.json"
	retryFrequency                       = 1 * time.Second
	retryDuration                        = 30 * time.Second
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

func mountGCSFuseAndSetupTestDir(flags []string, ctx context.Context, storageClient *storage.Client) {
	setup.MountGCSFuseWithGivenMountWithConfigFunc(testEnv.cfg, flags, mountFunc)
	// In case of GKE, the mount directory is not created by the test.
	if testEnv.cfg.GKEMountedDirectory != "" {
		setup.SetMntDir(testEnv.cfg.GKEMountedDirectory)
	}
	testEnv.testDirPath = client.SetupTestDirectory(ctx, storageClient, kTestDirName)
}

// doesNotHaveInactiveReaderClosedLogLineInLogFile checks if the "Closing reader for object ... due to inactivity"
// log message is absent for the given objectName, b/w the [startTime, endTime] interval.
// It sleeps for 5 seconds before checking to allow logs to be flushed.
func doesNotHaveInactiveReaderClosedLogLineInLogFile(t *testing.T, objectName, logFile string, startTime, endTime time.Time) {
	t.Helper()
	time.Sleep(5 * time.Second)

	_, err := hasInactiveReaderClosedLogLineInLogFile(t, objectName, logFile, startTime, endTime)
	if err == nil {
		t.Fatalf("Unexpected 'Inactive Reader Closed' log message found in log file %s for object %s", logFile, objectName)
	}
}

// hasInactiveReaderClosedLogInLogFile checks if the "Closing reader for object ... due to inactivity"
// log message is present for the given objectName, b/w the [startTime, endTime] interval.
func hasInactiveReaderClosedLogLineInLogFile(t *testing.T, objectName, logFile string, startTime, endTime time.Time) (string, error) {
	t.Helper()
	expectedMsgSubstring := fmt.Sprintf("Closing reader for object %q due to inactivity.", objectName)

	file, err := os.Open(logFile)
	if err != nil {
		return "", fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Pre-filter: skip lines that do not contain the core parts of the expected message.
		// We cannot use `expectedMsgSubstring` directly for the pre-filter because the double
		// quotes around the object name will be escaped as `\"` in the raw JSON string.
		if !strings.Contains(line, "Closing reader for object") || !strings.Contains(line, objectName) {
			continue
		}

		logEntry, err := read_logs.ParseJsonLogLineIntoLogEntryStruct(line)
		if err == nil && logEntry != nil {
			if (logEntry.Timestamp.After(startTime) || logEntry.Timestamp.Equal(startTime)) &&
				(logEntry.Timestamp.Before(endTime) || logEntry.Timestamp.Equal(endTime)) {
				if strings.Contains(logEntry.Message, expectedMsgSubstring) {
					return line, nil
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to read log file: %w", err)
	}

	return "", fmt.Errorf("expected log message substring %q not found between %s and %s", expectedMsgSubstring, startTime, endTime)
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
		setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(setup.TestBucket(), setup.OnlyDirMounted()))
	}

	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(setup.TestBucket(), kTestDirName))
	os.Exit(successCode)
}
