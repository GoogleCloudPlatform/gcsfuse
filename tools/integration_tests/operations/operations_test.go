// Copyright 2024 Google LLC
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

// Provides integration tests for file and directory operations.
package operations_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/creds_tests"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/dynamic_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/only_dir_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/persistent_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

const DirForOperationTests = "dirForOperationsTest"
const MoveFile = "move.txt"
const MoveFileContent = "This is from move file in Test directory.\n"
const SrcCopyDirectory = "srcCopyDir"
const SubSrcCopyDirectory = "subSrcCopyDir"
const SrcCopyFile = "copy.txt"
const SrcCopyFileContent = "This is from copy file in srcCopy directory.\n"
const DestCopyDirectory = "destCopyDir"
const DestNonEmptyCopyDirectory = "destNonEmptyCopyDirectory"
const SubDirInNonEmptyDestCopyDirectory = "subDestCopyDir"
const DestCopyDirectoryNotExist = "notExist"
const NumberOfObjectsInSrcCopyDirectory = 2
const NumberOfObjectsInNonEmptyDestCopyDirectory = 2
const DestEmptyCopyDirectory = "destEmptyCopyDirectory"
const EmptySrcDirectoryCopyTest = "emptySrcDirectoryCopyTest"
const NumberOfObjectsInEmptyDestCopyDirectory = 1
const NumberOfObjectsInBucketDirectoryListTest = 1
const DirectoryForListTest = "directoryForListTest"
const NumberOfObjectsInDirectoryForListTest = 4
const NumberOfFilesInDirectoryForListTest = 1
const EmptySubDirInDirectoryForListTest = "emptySubDirInDirectoryForListTest"
const NumberOfObjectsInEmptySubDirInDirectoryForListTest = 0
const NumberOfFilesInEmptySubDirInDirectoryForListTest = 0
const FirstSubDirectoryForListTest = "firstSubDirectoryForListTest"
const NumberOfObjectsInFirstSubDirectoryForListTest = 1
const NumberOfFilesInFirstSubDirectoryForListTest = 1
const PrefixFileInDirectoryForListTest = "fileInDirectoryForListTest"
const FileInDirectoryForListTest = "fileInDirectoryForListTest1"
const NumberOfObjectsInSecondSubDirectoryForListTest = 2
const NumberOfFilesInSecondSubDirectoryForListTest = 2
const PrefixFileInFirstSubDirectoryForListTest = "fileInFirstSubDirectoryForListTest"
const FileInFirstSubDirectoryForListTest = "fileInFirstSubDirectoryForListTest1"
const SecondSubDirectoryForListTest = "secondSubDirectoryForListTest"
const PrefixFileInSecondSubDirectoryForListTest = "fileInSecondSubDirectoryForListTest"
const FirstFileInSecondSubDirectoryForListTest = "fileInSecondSubDirectoryForListTest1"
const SecondFileInSecondSubDirectoryForListTest = "fileInSecondSubDirectoryForListTest2"
const EmptyExplicitDirectoryForDeleteTest = "emptyExplicitDirectoryForDeleteTest"
const NonEmptyExplicitDirectoryForDeleteTest = "nonEmptyExplicitDirectoryForDeleteTest"
const NonEmptyExplicitSubDirectoryForDeleteTest = "nonEmptyExplicitSubDirectoryForDeleteTest"
const NumberOfFilesInNonEmptyExplicitDirectoryForDeleteTest = 2
const PrefixFilesInNonEmptyExplicitDirectoryForDeleteTest = "filesInNonEmptyExplicitDirectoryForDeleteTest"
const NumberOfFilesInNonEmptyExplicitSubDirectoryForDeleteTest = 1
const PrefixFilesInNonEmptyExplicitSubDirectoryForDeleteTest = "filesInNonEmptyExplicitSubDirectoryForDeleteTest"
const DirOneInCreateThreeLevelDirTest = "dirOneInCreateThreeLevelDirTest"
const DirTwoInCreateThreeLevelDirTest = "dirTwoInCreateThreeLevelDirTest"
const DirThreeInCreateThreeLevelDirTest = "dirThreeInCreateThreeLevelDirTest"
const NumberOfObjectsInBucketDirectoryCreateTest = 1
const NumberOfObjectsInDirOneInCreateThreeLevelDirTest = 1
const NumberOfObjectsInDirTwoInCreateThreeLevelDirTest = 1
const NumberOfObjectsInDirThreeInCreateThreeLevelDirTest = 1
const PrefixFileInDirThreeInCreateThreeLevelDirTest = "fileInDirThreeInCreateThreeLevelDirTest"
const FileInDirThreeInCreateThreeLevelDirTest = "fileInDirThreeInCreateThreeLevelDirTest1"
const ContentInFileInDirThreeInCreateThreeLevelDirTest = "Hello world!!"
const Content = "line 1\nline 2\n"
const onlyDirMounted = "OnlyDirMountOperations"

var (
	cacheDir      string
	storageClient *storage.Client
	ctx           context.Context
	err           error
)

func RunTestOnTPCEndPoint(cfg test_suite.Config, m *testing.M) int {
	ctx = context.Background()
	if storageClient, err = client.CreateStorageClient(ctx); err != nil {
		log.Fatalf("Error creating storage client: %v\n", err)
	}
	cfg.Operations = make([]test_suite.TestConfig, 1)
	cfg.Operations[0].TestBucket = setup.TestBucket()
	cfg.Operations[0].MountedDirectory = setup.MountedDirectory()
	cfg.Operations[0].Configs = make([]test_suite.ConfigItem, 1)
	cfg.Operations[0].Configs[0].Flags = []string{
		"--enable-atomic-rename-object=true",
		"--experimental-enable-json-read=true",
		"--create-empty-file=true --enable-atomic-rename-object=true",
		"--metadata-cache-ttl-secs=0 --enable-streaming-writes=false",
		"--kernel-list-cache-ttl-secs=-1 --implicit-dirs=true",
	}
	cacheDirFlag := fmt.Sprintf("--file-cache-max-size-mb=2 --cache-dir=%s/cache-dir-operations-hns", os.TempDir())
	cfg.Operations[0].Configs[0].Flags = append(cfg.Operations[0].Configs[0].Flags, cacheDirFlag)
	cfg.Operations[0].Configs[0].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": false}
	var flags [][]string

	// Iterate over the original flags and split each string by spaces
	for _, flagSet := range cfg.Operations[0].Configs[0].Flags {
		splitFlags := strings.Fields(flagSet)
		flags = append(flags, splitFlags)
	}
	setup.SetUpTestDirForTestBucket(cfg.Operations[0].TestBucket)
	successCodeTPC := static_mounting.RunTestsWithConfigFile(&cfg.Operations[0], flags, m)
	return successCodeTPC
}
func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// 1. Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())

	if setup.TestOnTPCEndPoint() {
		log.Println("Running TPC tests without config file.")
		successCodeTPC := RunTestOnTPCEndPoint(cfg, m)
		os.Exit(successCodeTPC)
	}

	var successCode int
	if len(cfg.Operations) != 0 {
		var tpcCfg test_suite.Config
		setup.SetTestBucket(cfg.Operations[0].TestBucket)
		var remainingConfigs []test_suite.ConfigItem

		for _, config := range cfg.Operations[0].Configs {
			// Check if the Tpc field is true.
			if config.Tpc {
				log.Println("Running TPC test")
				successCode = RunTestOnTPCEndPoint(tpcCfg, m)
			} else {
				// If the Tpc field is false, add it to the remainingConfigs slice.
				remainingConfigs = append(remainingConfigs, config)
			}
		}
		cfg.Operations[0].Configs = remainingConfigs
	}

	if len(cfg.Operations) == 0 {
		log.Println("No configuration found for operations tests in config. Using flags instead.")
		// Populate the config manually.
		cfg.Operations = make([]test_suite.TestConfig, 1)
		cfg.Operations[0].TestBucket = setup.TestBucket()
		cfg.Operations[0].MountedDirectory = setup.MountedDirectory()
		cfg.Operations[0].Configs = make([]test_suite.ConfigItem, 2)
		cfg.Operations[0].Configs[0].Flags = []string{
			"--enable-atomic-rename-object=true",
			"--experimental-enable-json-read=true",
			"--client-protocol=grpc --implicit-dirs=true --enable-atomic-rename-object=true",
			"--create-empty-file=true --enable-atomic-rename-object=true",
			"--metadata-cache-ttl-secs=0 --enable-streaming-writes=false",
			"--kernel-list-cache-ttl-secs=-1 --implicit-dirs=true",
		}
		cfg.Operations[0].Configs[1].Flags = []string{
			"--experimental-enable-json-read=true --enable-atomic-rename-object=true",
			"--client-protocol=grpc --implicit-dirs=true --enable-atomic-rename-object=true",
			"--create-empty-file=true --enable-atomic-rename-object=true",
			"--metadata-cache-ttl-secs=0 --enable-streaming-writes=false",
			"--kernel-list-cache-ttl-secs=-1 --implicit-dirs=true",
		}
		cacheDirFlag := fmt.Sprintf("--file-cache-max-size-mb=2 --cache-dir=%s/cache-dir-operations-hns", os.TempDir())
		cfg.Operations[0].Configs[0].Flags = append(cfg.Operations[0].Configs[0].Flags, cacheDirFlag)
		cfg.Operations[0].Configs[1].Flags = append(cfg.Operations[0].Configs[1].Flags, cacheDirFlag)
		cfg.Operations[0].Configs[0].Compatible = map[string]bool{"flat": true, "hns": false, "zonal": true}
		cfg.Operations[0].Configs[1].Compatible = map[string]bool{"flat": false, "hns": true, "zonal": true}
	}

	// 2. Create storage client before running tests.
	setup.SetTestBucket(cfg.Operations[0].TestBucket)
	ctx = context.Background()
	closeStorageClient := client.CreateStorageClientWithCancel(&ctx, &storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Printf("closeStorageClient failed: %v", err)
		}
	}()

	// 3. To run mountedDirectory tests, we need both testBucket and mountedDirectory
	// flags to be set, as operations tests validates content from the bucket.
	if cfg.Operations[0].MountedDirectory != "" && cfg.Operations[0].TestBucket != "" {
		os.Exit(setup.RunTestsForMountedDirectory(cfg.Operations[0].MountedDirectory, m))
	}

	// Run tests for testBucket
	// 4. Build the flag sets dynamically from the config.
	bucketType, err := setup.BucketType(ctx, cfg.Operations[0].TestBucket)
	if err != nil {
		log.Fatalf("BucketType failed: %v", err)
	}
	if bucketType == setup.ZonalBucket {
		setup.SetIsZonalBucketRun(true)
	}
	flags := setup.BuildFlagSets(cfg.Operations[0], bucketType)

	// Set up test directory.
	setup.SetUpTestDirForTestBucket(cfg.Operations[0].TestBucket)
	if successCode == 0 {
		successCode = static_mounting.RunTestsWithConfigFile(&cfg.Operations[0], flags, m)
	}

	if successCode == 0 {
		successCode = only_dir_mounting.RunTestsWithConfigFile(&cfg.Operations[0], flags, onlyDirMounted, m)
	}

	if successCode == 0 {
		successCode = persistent_mounting.RunTestsWithConfigFile(&cfg.Operations[0], flags, m)
	}

	if successCode == 0 {
		successCode = dynamic_mounting.RunTests(ctx, storageClient, flags, m)
	}

	if successCode == 0 {
		// Test for admin permission on test bucket.
		successCode = creds_tests.RunTestsForKeyFileAndGoogleApplicationCredentialsEnvVarSetWithConfigFile(&cfg.Operations[0], ctx, storageClient, flags, "objectAdmin", m)
	}

	os.Exit(successCode)
}
