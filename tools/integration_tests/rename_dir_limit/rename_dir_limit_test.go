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

// Provides integration tests when --rename-dir-limit flag is set.
package rename_dir_limit_test

import (
	"context"
	"log"
	"os"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/only_dir_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/persistent_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

const DirForRenameDirLimitTests = "dirForRenameDirLimitTests"
const DirectoryWithThreeFiles = "directoryWithThreeFiles"
const DirectoryWithTwoFiles = "directoryWithTwoFiles"
const DirectoryWithFourFiles = "directoryWithFourFiles"
const DirectoryWithTwoFilesOneEmptyDirectory = "directoryWithTwoFilesOneEmptyDirectory"
const DirectoryWithTwoFilesOneNonEmptyDirectory = "directoryWithTwoFilesOneNonEmptyDirectory"
const EmptySubDirectory = "emptySubDirectory"
const NonEmptySubDirectory = "nonEmptySubDirectory"
const RenamedDirectory = "renamedDirectory"
const SrcDirectory = "srcDirectory"
const EmptyDestDirectory = "emptyDestDirectory"
const PrefixTempFile = "temp"
const onlyDirMounted = "OnlyDirMountRenameDirLimit"

var (
	storageClient *storage.Client
	ctx           context.Context
)

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// 1. Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.RenameDirLimit) == 0 {
		log.Println("No configuration found for rename dir limit tests in config. Using flags instead.")
		// Populate the config manually.
		cfg.RenameDirLimit = make([]test_suite.TestConfig, 1)
		cfg.RenameDirLimit[0].TestBucket = setup.TestBucket()
		cfg.RenameDirLimit[0].GKEMountedDirectory = setup.MountedDirectory()
		cfg.RenameDirLimit[0].Configs = make([]test_suite.ConfigItem, 2)
		cfg.RenameDirLimit[0].Configs[0].Flags = []string{
			"--rename-dir-limit=3 --implicit-dirs --client-protocol=grpc",
			"--rename-dir-limit=3",
			"--rename-dir-limit=3 --client-protocol=grpc",
		}
		cfg.RenameDirLimit[0].Configs[0].Compatible = map[string]bool{"flat": true, "hns": false, "zonal": false}
		cfg.RenameDirLimit[0].Configs[1].Flags = []string{
			"",
		}
		cfg.RenameDirLimit[0].Configs[1].Compatible = map[string]bool{"flat": false, "hns": true, "zonal": true}
	}

	ctx = context.Background()
	bucketType := setup.TestEnvironment(ctx, &cfg.RenameDirLimit[0])

	// 2. Create storage client before running tests.
	var err error
	storageClient, err = client.CreateStorageClient(ctx)
	if err != nil {
		log.Printf("Error creating storage client: %v\n", err)
		os.Exit(1)
	}
	defer storageClient.Close()

	// 3. To run mountedDirectory tests, we need both testBucket and mountedDirectory
	if cfg.RenameDirLimit[0].GKEMountedDirectory != "" && cfg.RenameDirLimit[0].TestBucket != "" {
		os.Exit(setup.RunTestsForMountedDirectory(cfg.RenameDirLimit[0].GKEMountedDirectory, m))
	}

	// Run tests for testBucket
	// 4. Build the flag sets dynamically from the config.
	flags := setup.BuildFlagSets(cfg.RenameDirLimit[0], bucketType, "")
	setup.SetUpTestDirForTestBucket(&cfg.RenameDirLimit[0])

	successCode := static_mounting.RunTestsWithConfigFile(&cfg.RenameDirLimit[0], flags, m)

	if successCode == 0 {
		successCode = only_dir_mounting.RunTestsWithConfigFile(&cfg.RenameDirLimit[0], flags, onlyDirMounted, m)
	}

	if successCode == 0 {
		successCode = persistent_mounting.RunTestsWithConfigFile(&cfg.RenameDirLimit[0], flags, m)
	}

	os.Exit(successCode)
}
