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
	"log"
	"os"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

const FiveHundredMB = 500 * operations.MiB
const ChunkSize = 200 * operations.MiB
const RandomReadChunkSize = operations.MiB
const NumberOfRandomReadCalls = 200
const MinReadableByteFromFile = 0
const MaxReadableByteFromFile = 500 * operations.MiB
const DirForReadLargeFilesTests = "dirForReadLargeFilesTests"

var (
	storageClient *storage.Client
	ctx           context.Context
)

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// 1. Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.ReadLargeFiles) == 0 {
		log.Fatal("No configuration found for ReadLargeFiles in config file.")
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

	setup.SetUpTestDirForTestBucket(&cfg.ReadLargeFiles[0])
	setup.OverrideFilePathsInFlagSet(&cfg.ReadLargeFiles[0], setup.TestDir())

	// 4. Build the flag sets dynamically from the modified config.
	flags := setup.BuildFlagSets(cfg.ReadLargeFiles[0], bucketType, "")

	successCode := static_mounting.RunTestsWithConfigFile(&cfg.ReadLargeFiles[0], flags, m)

	os.Exit(successCode)
}
