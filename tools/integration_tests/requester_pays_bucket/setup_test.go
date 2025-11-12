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

// Provide tests for cases where bucket with requester-pays feature is
// mounted and used through gcsfuse.
package requester_pays_bucket

import (
	"context"
	"log"
	"os"
	"path"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/only_dir_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

const (
	testDirName        = "RequesterPaysBucketTests"
	onlyDirTestDirName = "OnlyDirRequesterPaysBucketTests"
)

// To prevent global variable pollution, enhance code clarity,
// and avoid inadvertent errors. We strongly suggest that, all new package-level
// variables (which would otherwise be declared with `var` at the package root) should
// be added as fields to this 'env' struct instead.
type env struct {
	testDirPath   string
	storageClient *storage.Client
	ctx           context.Context
	bucketName    string
}

var (
	testEnv env
)

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	if setup.IsZonalBucketRun() {
		log.Fatal("Test not supported for zonal bucket as they don't support requester-pays feature")
	}

	// Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.RequesterPaysBucket) == 0 {
		log.Println("No configuration found for requester pays bucket tests in config. Using flags instead.")
		// Populate the config manually.
		cfg.RequesterPaysBucket = make([]test_suite.TestConfig, 1)
		cfg.RequesterPaysBucket[0].TestBucket = setup.TestBucket()
		cfg.RequesterPaysBucket[0].GKEMountedDirectory = setup.MountedDirectory()
		cfg.RequesterPaysBucket[0].Configs = make([]test_suite.ConfigItem, 1)
		cfg.RequesterPaysBucket[0].Configs[0].Flags = []string{
			"--billing-project=gcs-fuse-test-ml",
		}
		cfg.RequesterPaysBucket[0].Configs[0].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": false}
	}

	testEnv.ctx = context.Background()
	bucketType := setup.TestEnvironment(testEnv.ctx, &cfg.RequesterPaysBucket[0])

	// Create storage client before running tests.
	closeStorageClient := client.CreateStorageClientWithCancel(&testEnv.ctx, &testEnv.storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	// Temporarily enable --requester-pays metadata flag for the test bucket.
	testEnv.bucketName = strings.Split(cfg.RequesterPaysBucket[0].TestBucket, "/")[0]
	client.MustEnableRequesterPays(testEnv.storageClient, testEnv.ctx, testEnv.bucketName)
	defer client.MustDisableRequesterPays(testEnv.storageClient, testEnv.ctx, testEnv.bucketName)

	// To run mountedDirectory tests, we need both testBucket and mountedDirectory
	// flags to be set, as RequesterPaysBucket tests validates content from the bucket.
	if cfg.RequesterPaysBucket[0].GKEMountedDirectory != "" && cfg.RequesterPaysBucket[0].TestBucket != "" {
		os.Exit(setup.RunTestsForMountedDirectory(cfg.RequesterPaysBucket[0].GKEMountedDirectory, m))
	}

	// Run tests for testBucket
	// Build the flag sets dynamically from the config.
	flags := setup.BuildFlagSets(cfg.RequesterPaysBucket[0], bucketType, "")

	setup.SetUpTestDirForTestBucket(&cfg.RequesterPaysBucket[0])

	log.Println("Running static mounting tests...")
	successCode := static_mounting.RunTestsWithConfigFile(&cfg.RequesterPaysBucket[0], flags, m)

	if successCode == 0 {
		log.Printf("Running only-dir mounting tests ...")
		successCode = only_dir_mounting.RunTestsWithConfigFile(&cfg.RequesterPaysBucket[0], flags, onlyDirTestDirName, m)
	}

	// If failed, then save the gcsfuse log file(s).
	setup.SaveLogFileInCaseOfFailure(successCode)

	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)
}
