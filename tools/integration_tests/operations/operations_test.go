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
	"github.com/stretchr/testify/suite"
)

type operationsTestSuite struct {
	suite.Suite
}

func (s *operationsTestSuite) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
}

func runOperationsSuite(t *testing.T, runSuiteFunc func()) {
	// Run tests for mounted directory if the flag is set. This assumes that run flag is properly passed by GKE team as per the config.
	if operationsConfig.GKEMountedDirectory != "" && operationsConfig.TestBucket != "" {
		runSuiteFunc()
		return
	}

	// Run tests for GCE environment otherwise.
	flagsSet := setup.BuildFlagSets(*operationsConfig, bucketType, t.Name())
	for _, flags := range flagsSet {
		t.Run(strings.Join(flags, "_"), func(t *testing.T) {
			// 1. Static mounting
			t.Run("Static", func(t *testing.T) {
				static_mounting.RunSuiteForStaticMounting(operationsConfig, flags, t, runSuiteFunc)
			})

			if setup.TestOnTPCEndPoint() {
				return
			}

			// 2. Only-dir mounting
			t.Run("OnlyDir", func(t *testing.T) {
				only_dir_mounting.RunSuiteForOnlyDirMounting(ctx, storageClient, operationsConfig, flags, onlyDirMounted, t, runSuiteFunc)
			})

			// 3. Persistent mounting
			t.Run("Persistent", func(t *testing.T) {
				persistent_mounting.RunSuiteForPersistentMounting(operationsConfig, flags, t, runSuiteFunc)
			})

			// 4. Dynamic mounting
			t.Run("Dynamic", func(t *testing.T) {
				dynamic_mounting.RunSuiteForDynamicMounting(operationsConfig, flags, t, runSuiteFunc)
			})

			// 5. Creds tests (runs for different auth methods)
			t.Run("Creds", func(t *testing.T) {
				creds_tests.RunSuiteForDifferentAuthMethods(ctx, operationsConfig, storageClient, flags, "objectAdmin", t, runSuiteFunc)
			})
		})
	}
}

func TestOperationsBase(t *testing.T) {
	runOperationsSuite(t, func() {
		suite.Run(t, new(operationsTestSuite))
		suite.Run(t, &writeOperationsTest{isRapidWritesEnabled: false})
	})
}

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
	storageClient    *storage.Client
	ctx              context.Context
	operationsConfig *test_suite.TestConfig
	bucketType       string
)

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// 1. Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.Operations) == 0 {
		log.Fatal("No configuration found for operations tests in config.")
	}

	ctx = context.Background()
	operationsConfig = &cfg.Operations[0]
	bucketType = setup.TestEnvironment(ctx, operationsConfig)

	// 2. Create storage client before running tests.
	var err error
	storageClient, err = client.CreateStorageClient(ctx)
	if err != nil {
		log.Printf("Error creating storage client: %v\n", err)
		os.Exit(1)
	}
	defer storageClient.Close()

	// 3. To run mountedDirectory tests, we need both testBucket and mountedDirectory
	if operationsConfig.GKEMountedDirectory != "" && operationsConfig.TestBucket != "" {
		os.Exit(setup.RunTestsForMountedDirectory(operationsConfig.GKEMountedDirectory, m))
	}

	// 4. Override GKE specific paths with GCSFuse paths if running in GCE environment.
	setup.OverrideFilePathsInFlagSet(operationsConfig, setup.TestDir())
	setup.SetUpTestDirForTestBucket(operationsConfig)

	successCode := m.Run()
	setup.SaveLogFileInCaseOfFailure(successCode)
	os.Exit(successCode)
}
