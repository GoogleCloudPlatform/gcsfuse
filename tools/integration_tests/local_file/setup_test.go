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

// Provides integration tests for file and directory operations.

package local_file

import (
	"context"
	"log"
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/dynamic_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/only_dir_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// 1. Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.LocalFile) == 0 {
		log.Println("No configuration found for LocalFile tests in config. Using flags instead.")
		// Populate the config manually.
		cfg.LocalFile = make([]test_suite.TestConfig, 1)
		cfg.LocalFile[0].TestBucket = setup.TestBucket()
		cfg.LocalFile[0].MountedDirectory = setup.MountedDirectory()
		cfg.LocalFile[0].Configs = make([]test_suite.ConfigItem, 1)
		cfg.LocalFile[0].Configs[0].Flags = []string{
			// Set up flags to run tests on local file test suite.
			// Not setting config file explicitly with 'create-empty-file: false' as it is default.
			// Running these tests with streaming writes disabled because local file tests are already running in streaming_writes test package.
			"--implicit-dirs=true --rename-dir-limit=3 --enable-streaming-writes=false",
			"--implicit-dirs=true --rename-dir-limit=3 --enable-streaming-writes=false  --client-protocol=grpc",
			"--implicit-dirs=false --rename-dir-limit=3 --enable-streaming-writes=false",
			"--implicit-dirs=false --rename-dir-limit=3 --enable-streaming-writes=false --client-protocol=grpc",
		}
		cfg.LocalFile[0].Configs[0].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
	}
	// set the test dir to local file test
	testDirName = testDirLocalFileTest
	var err error

	setup.SetBucketFromConfigFile(cfg.LocalFile[0].TestBucket)
	ctx = context.Background()
	bucketType, err := setup.BucketType(ctx, cfg.LocalFile[0].TestBucket)
	if err != nil {
		log.Fatalf("BucketType failed: %v", err)
	}
	if bucketType == setup.ZonalBucket {
		setup.SetIsZonalBucketRun(true)
	}

	// 2. Create storage client before running tests.
	closeStorageClient := client.CreateStorageClientWithCancel(&ctx, &storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	// 3. To run mountedDirectory tests, we need both testBucket and mountedDirectory
	// flags to be set, as LocalFile tests validates content from the bucket.
	if cfg.LocalFile[0].MountedDirectory != "" && cfg.LocalFile[0].TestBucket != "" {
		os.Exit(setup.RunTestsForMountedDirectory(cfg.LocalFile[0].MountedDirectory, m))
	}

	// Run tests for testBucket
	// 4. Build the flag sets dynamically from the config.
	flags := setup.BuildFlagSets(cfg.LocalFile[0], bucketType)
	setup.SetUpTestDirForTestBucket(cfg.LocalFile[0].TestBucket)

	successCode := static_mounting.RunTestsWithConfigFile(&cfg.LocalFile[0], flags, m)

	if successCode == 0 {
		successCode = only_dir_mounting.RunTestsWithConfigFile(&cfg.LocalFile[0], onlyDirMounted, m)
	}

	// Dynamic mounting tests create a bucket and perform tests on that bucket,
	// which is not a hierarchical bucket. So we are not running those tests with
	// hierarchical bucket.
	if successCode == 0 && !setup.ResolveIsHierarchicalBucket(ctx, cfg.LocalFile[0].TestBucket, storageClient) {
		successCode = dynamic_mounting.RunTests(ctx, storageClient, flags, m)
	}

	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(ctx, storageClient, path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)
}

func TestLocalFileTestSuite(t *testing.T) {
	s := new(localFileTestSuite)
	s.CommonLocalFileTestSuite.TestifySuite = &s.Suite
	suite.Run(t, s)
}
