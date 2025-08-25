// Copyright 2023 Google LLC
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

// Provides integration tests for write large files sequentially and randomly.
package write_large_files

import (
	"context"
	"log"
	"os"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

const (
	TmpDir               = "/tmp"
	OneMiB               = 1024 * 1024
	WritePermission_0200 = 0200
)

var (
	storageClient *storage.Client
	ctx           context.Context
)

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// 1. Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.WriteLargeFiles) == 0 {
		log.Println("No configuration found for write large files tests in config. Using flags instead.")
		// Populate the config manually.
		cfg.WriteLargeFiles = make([]test_suite.TestConfig, 1)
		cfg.WriteLargeFiles[0].TestBucket = setup.TestBucket()
		cfg.WriteLargeFiles[0].MountedDirectory = setup.MountedDirectory()
		cfg.WriteLargeFiles[0].Configs = make([]test_suite.ConfigItem, 1)
		cfg.WriteLargeFiles[0].Configs[0].Flags = []string{
			"--enable-streaming-writes=false",
			"--write-max-blocks-per-file=2 --write-global-max-blocks=5",
		}
		cfg.WriteLargeFiles[0].Configs[0].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
	}

	setup.SetTestBucket(cfg.WriteLargeFiles[0].TestBucket)
	ctx = context.Background()
	bucketType, err := setup.BucketType(ctx, cfg.WriteLargeFiles[0].TestBucket)
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
	// flags to be set, as WriteLargeFiles tests validates content from the bucket.
	if cfg.WriteLargeFiles[0].MountedDirectory != "" && cfg.WriteLargeFiles[0].TestBucket != "" {
		os.Exit(setup.RunTestsForMountedDirectory(cfg.WriteLargeFiles[0].MountedDirectory, m))
	}

	// Run tests for testBucket// Run tests for testBucket
	// 4. Build the flag sets dynamically from the config.
	flags := setup.BuildFlagSets(cfg.WriteLargeFiles[0], bucketType)

	setup.SetUpTestDirForTestBucket(cfg.WriteLargeFiles[0].TestBucket)

	successCode := static_mounting.RunTestsWithConfigFile(&cfg.WriteLargeFiles[0], flags, m)

	os.Exit(successCode)
}
