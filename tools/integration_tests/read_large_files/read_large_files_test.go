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

// Provides integration tests for read large files sequentially and randomly.
package read_large_files

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

const OneMB = 1024 * 1024
const FiveHundredMB = 500 * OneMB
const ChunkSize = 200 * OneMB
const NumberOfRandomReadCalls = 200
const MinReadableByteFromFile = 0
const MaxReadableByteFromFile = 500 * OneMB
const DirForReadLargeFilesTests = "dirForReadLargeFilesTests"

var (
	storageClient     *storage.Client
	ctx               context.Context
	FiveHundredMBFile = "fiveHundredMBFile" + setup.GenerateRandomString(5) + ".txt"
)

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// 1. Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.ReadLargeFiles) == 0 {
		log.Println("No configuration found for read large files tests in config. Using flags instead.")
		// Populate the config manually.
		cfg.ReadLargeFiles = make([]test_suite.TestConfig, 1)
		cfg.ReadLargeFiles[0].TestBucket = setup.TestBucket()
		cfg.ReadLargeFiles[0].GKEMountedDirectory = setup.MountedDirectory()
		cfg.ReadLargeFiles[0].Configs = make([]test_suite.ConfigItem, 1)
		cfg.ReadLargeFiles[0].Configs[0].Flags = []string{
			"--implicit-dirs",
			"--implicit-dirs --client-protocol=grpc",
			fmt.Sprintf("--implicit-dirs=true --file-cache-max-size-mb=700 --file-cache-cache-file-for-range-read=true --cache-dir=%s/cache-dir-read-large-files-%s", os.TempDir(), setup.GenerateRandomString(4)),
			fmt.Sprintf("--implicit-dirs=true --file-cache-max-size-mb=700 --file-cache-cache-file-for-range-read=true --client-protocol=grpc --cache-dir=%s/cache-dir-read-large-files-%s", os.TempDir(), setup.GenerateRandomString(4)),
			fmt.Sprintf("--implicit-dirs=true --file-cache-max-size-mb=-1 --file-cache-cache-file-for-range-read=false --cache-dir=%s/cache-dir-read-large-files-%s", os.TempDir(), setup.GenerateRandomString(4)),
			fmt.Sprintf("--implicit-dirs=true --file-cache-max-size-mb=-1 --file-cache-cache-file-for-range-read=false --client-protocol=grpc --cache-dir=%s/cache-dir-read-large-files-%s", os.TempDir(), setup.GenerateRandomString(4)),
		}
		cfg.ReadLargeFiles[0].Configs[0].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
	}

	ctx = context.Background()
	bucketType := setup.TestEnvironment(ctx, &cfg.ReadLargeFiles[0])

	// 2. Create storage client before running tests.
	closeStorageClient := client.CreateStorageClientWithCancel(&ctx, &storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	// 3. To run mountedDirectory tests, we need both testBucket and mountedDirectory
	// flags to be set, as ReadLargeFiles tests validates content from the bucket.
	if cfg.ReadLargeFiles[0].GKEMountedDirectory != "" && cfg.ReadLargeFiles[0].TestBucket != "" {
		os.Exit(setup.RunTestsForMountedDirectory(cfg.ReadLargeFiles[0].GKEMountedDirectory, m))
	}

	// Run tests for testBucket.
	// 4. Build the flag sets dynamically from the config.
	flags := setup.BuildFlagSets(cfg.ReadLargeFiles[0], bucketType, "")
	flags = setup.AddCacheDirToFlags(flags, "read-large-files")

	setup.SetUpTestDirForTestBucket(&cfg.ReadLargeFiles[0])

	successCode := static_mounting.RunTestsWithConfigFile(&cfg.ReadLargeFiles[0], flags, m)

	os.Exit(successCode)
}
