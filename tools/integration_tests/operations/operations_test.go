// Copyright 2024 Google LLC
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
package operations_test

import (
	"context"
	"log"
	"os"
	"strconv"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/creds_tests"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/dynamic_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/only_dir_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/persistent_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
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
)

func createMountConfigsAndEquivalentFlags() (flags [][]string) {
	mountConfig1 := map[string]any{
		"metadata-cache": map[string]any{
			"ttl-secs": 0,
		},
		"write": map[string]any{
			"enable-streaming-writes": false,
		},
	}
	filePath1 := setup.YAMLConfigFile(mountConfig1, "config1.yaml")
	flags = append(flags, []string{"--config-file=" + filePath1})

	mountConfig2 := map[string]any{
		"file-system": map[string]any{
			"kernel-list-cache-ttl-secs": -1,
		},
	}
	filePath2 := setup.YAMLConfigFile(mountConfig2, "config2.yaml")
	flags = append(flags, []string{"--config-file=" + filePath2, "--implicit-dirs=true"})
	return flags
}

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	// Create storage client before running tests.
	var err error
	ctx = context.Background()
	storageClient, err = client.CreateStorageClient(ctx)
	if err != nil {
		log.Printf("Error creating storage client: %v\n", err)
		os.Exit(1)
	}
	defer storageClient.Close()

	cacheDir = "cache-dir-operations-hns-" + strconv.FormatBool(setup.IsHierarchicalBucket(ctx, storageClient))

	// To run mountedDirectory tests, we need both testBucket and mountedDirectory
	// flags to be set, as operations tests validates content from the bucket.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		setup.RunTestsForMountedDirectoryFlag(m)
	}

	// Run tests for testBucket
	// Set up test directory.
	setup.SetUpTestDirForTestBucketFlag()
	// Set up flags to run tests on.
	// Note: GRPC related tests will work only if you have allow-list bucket.
	// Note: We are not testing specifically for implicit-dirs because they are covered as part of the other flags.
	flagsSet := [][]string{{"--enable-atomic-rename-object=true"}}

	// Enable experimental-enable-json-read=true case, but for non-presubmit runs only.
	if !setup.IsPresubmitRun() {
		flagsSet = append(flagsSet, []string{
			// By default, creating emptyFile is disabled.
			"--experimental-enable-json-read=true"})
	}

	// gRPC tests will not run in TPC environment
	if !testing.Short() && !setup.TestOnTPCEndPoint() {
		flagsSet = append(flagsSet, []string{"--client-protocol=grpc", "--implicit-dirs=true", "--enable-atomic-rename-object=true"})
	}

	// HNS tests utilize the gRPC protocol, which is not supported by TPC.
	if !setup.TestOnTPCEndPoint() {
		if setup.IsHierarchicalBucket(ctx, storageClient) {
			flagsSet = [][]string{{"--experimental-enable-json-read=true", "--enable-atomic-rename-object=true"}}
		}
	}

	mountConfigFlags := createMountConfigsAndEquivalentFlags()
	flagsSet = append(flagsSet, mountConfigFlags...)

	// Only running static_mounting test for TPC.
	if setup.TestOnTPCEndPoint() {
		successCodeTPC := static_mounting.RunTests(flagsSet, m)
		os.Exit(successCodeTPC)
	}

	successCode := static_mounting.RunTests(flagsSet, m)

	if successCode == 0 {
		successCode = only_dir_mounting.RunTests(flagsSet, onlyDirMounted, m)
	}

	if successCode == 0 {
		successCode = persistent_mounting.RunTests(flagsSet, m)
	}

	if successCode == 0 {
		successCode = dynamic_mounting.RunTests(ctx, storageClient, flagsSet, m)
	}

	if successCode == 0 {
		// Test for admin permission on test bucket.
		successCode = creds_tests.RunTestsForKeyFileAndGoogleApplicationCredentialsEnvVarSet(ctx, storageClient, flagsSet, "objectAdmin", m)
	}

	os.Exit(successCode)
}
