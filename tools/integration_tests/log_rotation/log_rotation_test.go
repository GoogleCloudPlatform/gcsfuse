// Copyright 2023 Google LLC
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
	"context"
	"log"
	"os"
	"path"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

const (
	testDirName        = "LogRotationTest"
	logFileName        = "log.txt"
	maxFileSizeMB      = 2
	activeLogFileCount = 1
	stderrLogFileCount = 1
	backupLogFileCount = 2
	logFileCount       = activeLogFileCount + backupLogFileCount + stderrLogFileCount // Adding 1 for stderr logs file
)

var (
	logDirPath    string
	logFilePath   string
	storageClient *storage.Client
	ctx           context.Context
)

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// Set up log file path.
	logDirPath = path.Join(setup.TestDir(), testDirName)
	logFilePath = path.Join(logDirPath, logFileName)
	setup.SetLogFile(logFilePath)

	// 1. Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.LogRotation) == 0 {
		log.Println("No configuration found for log rotation tests in config. Using flags instead.")
		// Populate the config manually.
		cfg.LogRotation = make([]test_suite.TestConfig, 1)
		cfg.LogRotation[0].TestBucket = setup.TestBucket()
		cfg.LogRotation[0].GKEMountedDirectory = setup.MountedDirectory()
		cfg.LogRotation[0].LogFile = setup.LogFile()
		cfg.LogRotation[0].Configs = make([]test_suite.ConfigItem, 1)
		cfg.LogRotation[0].Configs[0].Flags = []string{
			"---log-severity=TRACE --log-file=/gcsfuse-tmp/LogRotationTest/log.txt --log-rotate-max-file-size-mb=2 --log-rotate-backup-file-count=2 --log-rotate-compress=false",
			"---log-severity=TRACE --log-file=/gcsfuse-tmp/LogRotationTest/log.txt --log-rotate-max-file-size-mb=2 --log-rotate-backup-file-count=2 --log-rotate-compress=true",
		}
		cfg.LogRotation[0].Configs[0].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
	}

	ctx = context.Background()
	bucketType := setup.TestEnvironment(ctx, &cfg.LogRotation[0])

	// 2. Create storage client before running tests.
	var err error
	storageClient, err = client.CreateStorageClient(ctx)
	if err != nil {
		log.Printf("Error creating storage client: %v\n", err)
		os.Exit(1)
	}
	defer storageClient.Close()

	// 3. To run mountedDirectory tests, we need both testBucket and mountedDirectory
	if cfg.LogRotation[0].GKEMountedDirectory != "" && cfg.LogRotation[0].TestBucket != "" {
		os.Exit(setup.RunTestsForMountedDirectory(cfg.LogRotation[0].GKEMountedDirectory, m))
	}

	// Set up log file path.
	logDirPath = path.Join(setup.TestDir(), testDirName)
	logFilePath = path.Join(logDirPath, logFileName)
	setup.SetLogFile(logFilePath)

	// Run tests for testBucket
	// 4. Build the flag sets dynamically from the config.
	flags := setup.BuildFlagSets(cfg.LogRotation[0], bucketType, "")
	setup.SetUpTestDirForTestBucket(&cfg.LogRotation[0])

	successCode := static_mounting.RunTestsWithConfigFile(&cfg.LogRotation[0], flags, m)

	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(ctx, storageClient, path.Join(setup.TestBucket(), "/gcsfuse-tmp/LogRotationTest"))
	os.Exit(successCode)
}
