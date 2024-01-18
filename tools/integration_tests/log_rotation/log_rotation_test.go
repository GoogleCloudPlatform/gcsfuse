// Copyright 2023 Google Inc. All Rights Reserved.
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

// Provides integration tests for log rotation of gcsfuse logs.

package log_rotation

import (
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/internal/config"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

const (
	testDirName        = "LogRotationTest"
	logFileName        = "log.txt"
	logDirName         = "gcsfuse_integration_test_logs"
	maxFileSizeMB      = 2
	activeLogFileCount = 1
	backupLogFileCount = 2
	logFileCount       = activeLogFileCount + backupLogFileCount
)

var logDirPath string
var logFilePath string

func getMountConfigForLogRotation(maxFileSizeMB, backupFileCount int, compress bool,
	logFilePath string) config.MountConfig {
	mountConfig := config.MountConfig{
		LogConfig: config.LogConfig{
			Severity: config.TRACE,
			FilePath: logFilePath,
			LogRotateConfig: config.LogRotateConfig{
				MaxFileSizeMB:   maxFileSizeMB,
				BackupFileCount: backupFileCount,
				Compress:        compress,
			},
		},
	}
	return mountConfig
}

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	// Run tests for mountedDirectory only if --mountedDirectory flag is set.

	logDirPath = setup.ValidateLogDirForMountedDirTests(logDirName)
	logFilePath = path.Join(logDirPath, logFileName)
	setup.RunTestsForMountedDirectoryFlag(m)

	// Else run tests for testBucket.
	// Set up test directory.
	setup.SetUpTestDirForTestBucketFlag()

	// Set up directory for logs.
	logDirPath = setup.SetUpLogDirForTestDirTests(logDirName)
	logFilePath = path.Join(logDirPath, logFileName)

	// Set up config files.
	// TODO: add tests for backupLogFileCount = 0.
	configFile1 := setup.YAMLConfigFile(
		getMountConfigForLogRotation(maxFileSizeMB, backupLogFileCount, true, logFilePath),
		"config1.yaml")
	configFile2 := setup.YAMLConfigFile(
		getMountConfigForLogRotation(maxFileSizeMB, backupLogFileCount, false, logFilePath),
		"config2.yaml")

	// Set up flags to run tests on.
	// Not setting config file explicitly with 'create-empty-file: false' as it is default.
	flags := [][]string{
		{"--config-file=" + configFile1},
		{"--config-file=" + configFile2},
	}

	successCode := static_mounting.RunTests(flags, m)

	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(path.Join(setup.TestBucket(), testDirName))
	setup.RemoveBinFileCopiedForTesting()
	os.Exit(successCode)
}
