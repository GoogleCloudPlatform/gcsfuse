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
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/mounting/dynamic_mounting"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/mounting/only_dir_mounting"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// set the test dir to local file test
	testDirName = testDirLocalFileTest

	// 1. Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.LocalFile) == 0 {
		log.Println("No configuration found for LocalFile tests in config. Using flags instead.")
		// Populate the config manually.
		cfg.LocalFile = make([]test_suite.TestConfig, 1)
		cfg.LocalFile[0].TestBucket = setup.TestBucket()
		cfg.LocalFile[0].GKEMountedDirectory = setup.MountedDirectory()
		cfg.LocalFile[0].Configs = make([]test_suite.ConfigItem, 2)
		cfg.LocalFile[0].Configs[0].Flags = []string{
			"--implicit-dirs=true --rename-dir-limit=3 --enable-streaming-writes=false",
			"--implicit-dirs=false --rename-dir-limit=3 --enable-streaming-writes=false --client-protocol=grpc",
			"--rename-dir-limit=3 --write-block-size-mb=1 --write-max-blocks-per-file=2 --write-global-max-blocks=0",
		}
		cfg.LocalFile[0].Configs[0].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.LocalFile[0].Configs[1].Flags = []string{
			"--rename-dir-limit=3 --write-block-size-mb=1 --write-max-blocks-per-file=2 --write-global-max-blocks=-1 --client-protocol=grpc",
			"--rename-dir-limit=3 --write-block-size-mb=1 --write-max-blocks-per-file=2 --write-global-max-blocks=-1",
		}
		cfg.LocalFile[0].Configs[1].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": false}
	}

	// 2. Create storage client before running tests.
	ctx = context.Background()
	bucketType := setup.TestEnvironment(ctx, &cfg.LocalFile[0])
	closeStorageClient := client.CreateStorageClientWithCancel(&ctx, &storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	// 3. To run mountedDirectory tests, we need both testBucket and mountedDirectory
	// flags to be set, as LocalFile tests validates content from the bucket.
	if cfg.LocalFile[0].GKEMountedDirectory != "" && cfg.LocalFile[0].TestBucket != "" {
		os.Exit(setup.RunTestsForMountedDirectory(cfg.LocalFile[0].GKEMountedDirectory, m))
	}

	// Run tests for testBucket.
	// 4. Build the flag sets dynamically from the config.
	flags := setup.BuildFlagSets(cfg.LocalFile[0], bucketType, "")

	setup.SetUpTestDirForTestBucket(&cfg.LocalFile[0])

	successCode := static_mounting.RunTestsWithConfigFile(&cfg.LocalFile[0], flags, m)

	if successCode == 0 {
		successCode = only_dir_mounting.RunTestsWithConfigFile(&cfg.LocalFile[0], flags, onlyDirMounted, m)
	}

	// Dynamic mounting tests.
	if successCode == 0 {
		successCode = dynamic_mounting.RunTestsWithConfigFile(&cfg.LocalFile[0], flags, m)
	}

	os.Exit(successCode)
}

type LocalFileTestSuite struct {
	suite.Suite
}

func TestLocalFileTestSuite(t *testing.T) {
	s := new(LocalFileTestSuite)
	suite.Run(t, s)
}
