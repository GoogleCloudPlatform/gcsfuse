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
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

const (
	testDirName        = "TestLogRotation"
	maxFileSizeMB      = 2
	activeLogFileCount = 1
	stderrLogFileCount = 1
	backupLogFileCount = 2
	logFileCount       = activeLogFileCount + backupLogFileCount + stderrLogFileCount // Adding 1 for stderr logs file
	GKETempDir         = "/gcsfuse-tmp"
)

var (
	storageClient *storage.Client
	ctx           context.Context
	cfg           *test_suite.TestConfig
)

func setupLogFilePath(testName string) {
	var logFilePath = path.Join(setup.TestDir(), GKETempDir, testName) + ".log"
	cfg.LogFile = logFilePath
}

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// 1. Load and parse the common configuration.
	config := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(config.LogRotation) == 0 {
		log.Println("No configuration found for log rotation tests in config. Using flags instead.")
		// Populate the config manually.
		config.LogRotation = make([]test_suite.TestConfig, 1)
		config.LogRotation[0].TestBucket = setup.TestBucket()
		config.LogRotation[0].GKEMountedDirectory = setup.MountedDirectory()
		config.LogRotation[0].LogFile = setup.LogFile()
		config.LogRotation[0].Configs = make([]test_suite.ConfigItem, 1)
		config.LogRotation[0].Configs[0].Flags = []string{
			"--log-file=/gcsfuse-tmp/TestLogRotation.log --log-rotate-max-file-size-mb=2 --log-rotate-backup-file-count=2 --log-rotate-compress=false --log-severity=trace",
			"--log-file=/gcsfuse-tmp/TestLogRotation.log --log-rotate-max-file-size-mb=2 --log-rotate-backup-file-count=2 --log-rotate-compress=true --log-severity=trace",
		}
		config.LogRotation[0].Configs[0].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
	}

	cfg = &config.LogRotation[0]
	ctx = context.Background()
	bucketType := setup.TestEnvironment(ctx, cfg)

	// 2. Create storage client before running tests.
	closeStorageClient := client.CreateStorageClientWithCancel(&ctx, &storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	// 3. To run mountedDirectory tests, we need both testBucket and mountedDirectory
	if cfg.GKEMountedDirectory != "" && cfg.TestBucket != "" {
		log.Println("These tests will not run with mounted directory..")
		return
	}

	// 4. Build the flag sets dynamically from the config.
	setup.SetUpTestDirForTestBucket(cfg)

	// 5. Create the temporary directory for log rotation logs for GCE environment.
	if err := os.MkdirAll(path.Join(setup.TestDir(), GKETempDir), 0755); err != nil {
		log.Fatalf("Failed to create log directory: %v", err)
	}

	// 6. Override GKE specific paths with GCSFuse paths if running in GCE environment.
	overrideFilePathsInFlagSet(cfg, setup.TestDir())

	flags := setup.BuildFlagSets(*cfg, bucketType, "")
	setupLogFilePath(testDirName)

	successCode := static_mounting.RunTestsWithConfigFile(cfg, flags, m)

	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(ctx, storageClient, path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)
}

func overrideFilePathsInFlagSet(t *test_suite.TestConfig, GCSFuseTempDirPath string) {
	for _, flags := range t.Configs {
		for i := range flags.Flags {
			// Iterate over the indices of the flags slice
			flags.Flags[i] = strings.ReplaceAll(flags.Flags[i], "/gcsfuse-tmp", path.Join(GCSFuseTempDirPath, "gcsfuse-tmp"))
		}
	}
}
