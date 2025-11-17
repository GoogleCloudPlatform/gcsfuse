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

package implicit_and_explicit_dir_setup

import (
	"context"
	"log"
	"os"
	"path"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/stretchr/testify/require"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/mounting/persistent_mounting"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

const ExplicitDirectory = "explicitDirectory"
const ExplicitFile = "explicitFile"
const ImplicitDirectory = "implicitDirectory"
const ImplicitSubDirectory = "implicitSubDirectory"
const NumberOfExplicitObjects = 2
const NumberOfTotalObjects = 3
const NumberOfFilesInExplicitDirectory = 2
const NumberOfFilesInImplicitDirectory = 2
const NumberOfFilesInImplicitSubDirectory = 1
const PrefixFileInExplicitDirectory = "fileInExplicitDir"
const FirstFileInExplicitDirectory = "fileInExplicitDir1"
const SecondFileInExplicitDirectory = "fileInExplicitDir2"
const FileInImplicitDirectory = "fileInImplicitDir1"
const FileInImplicitSubDirectory = "fileInImplicitDir2"

func RunTestsForExplicitAndImplicitDir(config *test_suite.TestConfig, flags [][]string, m *testing.M) int {
	if config == nil {
		log.Println("config is nil")
		return 1
	}

	if len(flags) == 0 {
		log.Println("flags empty: no tests to run")
		return 0
	}

	if config.GKEMountedDirectory != "" && config.TestBucket != "" {
		successCode := setup.RunTestsForMountedDirectory(config.GKEMountedDirectory, m)
		return successCode
	}

	// Run tests for testBucket only if --testbucket flag is set.
	if config.TestBucket == "" {
		log.Print("pass test bucket to run the tests")
		return 1
	}
	setup.SetUpTestDirForTestBucket(config)

	successCode := static_mounting.RunTestsWithConfigFile(config, flags, m)

	if successCode == 0 {
		successCode = persistent_mounting.RunTestsWithConfigFile(config, flags, m)
	}
	return successCode
}

func RemoveAndCheckIfDirIsDeleted(dirPath string, dirName string, t *testing.T) {
	err := os.RemoveAll(dirPath)
	require.Nil(t, err)

	dir, err := os.Stat(dirPath)
	if err == nil && dir.Name() == dirName && dir.IsDir() {
		t.Errorf("Directory is not deleted.")
	}
}

// testdataCreateObjects is equivalent of the script tools/integration_tests/util/setup/implicit_and_explicit_dir_setup/testdata/create_objects.sh .
// That script uses gcloud, but this function instead uses go client library.
func testdataCreateObjects(ctx context.Context, t *testing.T, storageClient *storage.Client, testDirWithoutBucketName string) {
	t.Helper()
	// Following is needed for error log, and because
	// TestBucket can be of the form <bucket>/<onlydir> .
	bucketName, _, _ := strings.Cut(setup.TestBucket(), "/")

	objectName := path.Join(testDirWithoutBucketName, ImplicitDirectory, FileInImplicitDirectory)
	err := client.CreateObjectOnGCS(ctx, storageClient, objectName, "This is from directory fileInImplicitDir1 file implicitDirectory")
	if err != nil {
		t.Fatalf("Failed to create GCS object %q in bucket %q: %v", objectName, bucketName, err)
	}

	objectName = path.Join(testDirWithoutBucketName, path.Join(ImplicitDirectory, ImplicitSubDirectory), FileInImplicitSubDirectory)
	err = client.CreateObjectOnGCS(ctx, storageClient, objectName, "This is from directory implicitDirectory/implicitSubDirectory file fileInImplicitDir2")
	if err != nil {
		t.Fatalf("Failed to create GCS object %q in bucket %q: %v", objectName, bucketName, err)
	}
}

func CreateImplicitDirectoryStructureUsingStorageClient(ctx context.Context, t *testing.T, storageClient *storage.Client, testDir string) {
	t.Helper()

	// Implicit Directory Structure
	// testBucket/testDir/implicitDirectory                                                  -- Dir
	// testBucket/testDir/implicitDirectory/fileInImplicitDir1                               -- File
	// testBucket/testDir/implicitDirectory/implicitSubDirectory                             -- Dir
	// testBucket/testDir/implicitDirectory/implicitSubDirectory/fileInImplicitDir2          -- File

	// Create implicit directory in bucket for testing.
	testdataCreateObjects(ctx, t, storageClient, testDir)
}

func CreateImplicitDirectoryStructure(testDir string) {
	// Implicit Directory Structure
	// testBucket/testDir/implicitDirectory                                                  -- Dir
	// testBucket/testDir/implicitDirectory/fileInImplicitDir1                               -- File
	// testBucket/testDir/implicitDirectory/implicitSubDirectory                             -- Dir
	// testBucket/testDir/implicitDirectory/implicitSubDirectory/fileInImplicitDir2          -- File

	// Create implicit directory in bucket for testing.
	setup.RunScriptForTestData("../util/setup/implicit_and_explicit_dir_setup/testdata/create_objects.sh", path.Join(setup.TestBucket(), testDir))
}

func CreateExplicitDirectoryStructure(testDir string, t *testing.T) {
	// Explicit Directory structure
	// testBucket/testDir/explicitDirectory                            -- Dir
	// testBucket/testDir/explictFile                                  -- File
	// testBucket/testDir/explicitDirectory/fileInExplicitDir1         -- File
	// testBucket/testDir/explicitDirectory/fileInExplicitDir2         -- File

	dirPath := path.Join(setup.MntDir(), testDir, ExplicitDirectory)
	operations.CreateDirectoryWithNFiles(NumberOfFilesInExplicitDirectory, dirPath, PrefixFileInExplicitDirectory, t)
	filePath := path.Join(setup.MntDir(), testDir, ExplicitFile)
	file, err := os.Create(filePath)
	if err != nil {
		t.Errorf("Create file at %q: %v", setup.MntDir(), err)
	}

	// Closing file at the end.
	defer operations.CloseFileShouldNotThrowError(t, file)
}

func CreateImplicitDirectoryInExplicitDirectoryStructure(testDir string, t *testing.T) {
	// testBucket/testDir/explicitDirectory                                                                   -- Dir
	// testBucket/testDir/explictFile                                                                         -- File
	// testBucket/testDir/explicitDirectory/fileInExplicitDir1                                                -- File
	// testBucket/testDir/explicitDirectory/fileInExplicitDir2                                                -- File
	// testBucket/testDir/explicitDirectory/implicitDirectory                                                 -- Dir
	// testBucket/testDir/explicitDirectory/implicitDirectory/fileInImplicitDir1                              -- File
	// testBucket/testDir/explicitDirectory/implicitDirectory/implicitSubDirectory                            -- Dir
	// testBucket/testDir/explicitDirectory/implicitDirectory/implicitSubDirectory/fileInImplicitDir2         -- File

	// CreateExplicitDirectoryStructure writes files using GCSFuse.
	CreateExplicitDirectoryStructure(testDir, t)

	dirPathInBucket := path.Join(setup.TestBucket(), testDir, ExplicitDirectory)
	setup.RunScriptForTestData("../util/setup/implicit_and_explicit_dir_setup/testdata/create_objects.sh", dirPathInBucket)
}

func CreateImplicitDirectoryInExplicitDirectoryStructureUsingStorageClient(ctx context.Context, t *testing.T, storageClient *storage.Client, testDir string) {
	t.Helper()

	// testBucket/testDir/explicitDirectory                                                                   -- Dir
	// testBucket/testDir/explictFile                                                                         -- File
	// testBucket/testDir/explicitDirectory/fileInExplicitDir1                                                -- File
	// testBucket/testDir/explicitDirectory/fileInExplicitDir2                                                -- File
	// testBucket/testDir/explicitDirectory/implicitDirectory                                                 -- Dir
	// testBucket/testDir/explicitDirectory/implicitDirectory/fileInImplicitDir1                              -- File
	// testBucket/testDir/explicitDirectory/implicitDirectory/implicitSubDirectory                            -- Dir
	// testBucket/testDir/explicitDirectory/implicitDirectory/implicitSubDirectory/fileInImplicitDir2         -- File

	// CreateExplicitDirectoryStructure writes files using GCSFuse.
	CreateExplicitDirectoryStructure(testDir, t)

	dirPathInBucket := path.Join(testDir, ExplicitDirectory)
	testdataCreateObjects(ctx, t, storageClient, dirPathInBucket)
}
