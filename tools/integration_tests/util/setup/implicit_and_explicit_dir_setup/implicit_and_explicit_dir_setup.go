// Copyright 2023 Google Inc. All Rights Reserved.
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
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/persistent_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
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
const FileInUnsupportedImplicitDirectory1 = "fileInUnsupportedImplicitDir1"
const FileInUnsupportedImplicitDirectory2 = "fileInUnsupportedImplicitDir2"
const FileInUnsupportedImplicitDirectory3 = "fileInUnsupportedImplicitDir3"
const FileInUnsupportedPathInRootDirectory = "fileInUnsupportedPathInRootDirectory"
const FileInUnsupportedPathInRootDirectory2 = "fileInUnsupportedPathInRootDirectory2"
const FileInUnsupportedPathInRootDirectory3 = "fileInUnsupportedPathInRootDirectory3"

func RunTestsForImplicitDirAndExplicitDir(flags [][]string, m *testing.M) int {
	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	if setup.TestBucket() == "" && setup.MountedDirectory() != "" {
		log.Print("Please pass the name of bucket mounted at mountedDirectory to --testBucket flag.")
		os.Exit(1)
	}

	// Run tests for mountedDirectory only if --mountedDirectory and --testbucket flag is set.
	setup.RunTestsForMountedDirectoryFlag(m)

	// Run tests for testBucket only if --testbucket flag is set.
	setup.SetUpTestDirForTestBucketFlag()

	successCode := static_mounting.RunTests(flags, m)

	if successCode == 0 {
		successCode = persistent_mounting.RunTests(flags, m)
	}
	return successCode
}

func RemoveAndCheckIfDirIsDeleted(dirPath string, dirName string, t *testing.T) {
	operations.RemoveDir(dirPath)

	dir, err := os.Stat(dirPath)
	if err == nil && dir.Name() == dirName && dir.IsDir() {
		t.Errorf("Directory is not deleted.")
	}
}

type objectCreationMetadata struct{ content, completeObjectName string }

func createObjectsOnGcs(ctx context.Context, objects []objectCreationMetadata, sgClient *storage.Client, t *testing.T) {
	for _, object := range objects {
		if err := client.CreateObjectOnGCS(ctx, sgClient, object.completeObjectName, object.content); err != nil {
			t.Fatalf("Failed to create object %s: %v", object.completeObjectName, err)
		}
	}
}

func createObjectsInImplicitDir(ctx context.Context, completeTestDirName string, storageClient *storage.Client, t *testing.T) {
	implicitDirName := path.Join(completeTestDirName, ImplicitDirectory)
	createObjectsOnGcs(ctx,
		[]objectCreationMetadata{
			{"This is from directory fileInImplicitDir1 file implicitDirectory", path.Join(implicitDirName, FileInImplicitDirectory)},
			{"This is from directory implicitDirectory/implicitSubDirectory file fileInImplicitDir2", path.Join(implicitDirName, ImplicitSubDirectory, FileInImplicitSubDirectory)},
		},
		storageClient, t)
}

func CreateImplicitDirectoryStructure(ctx context.Context, testDir string, storageClient *storage.Client, t *testing.T) {
	// Implicit Directory Structure
	// testBucket/testDir/implicitDirectory                                                  -- Dir
	// testBucket/testDir/implicitDirectory/fileInImplicitDir1                               -- File
	// testBucket/testDir/implicitDirectory/implicitSubDirectory                             -- Dir
	// testBucket/testDir/implicitDirectory/implicitSubDirectory/fileInImplicitDir2          -- File

	// Create implicit directory in bucket for testing.
	createObjectsInImplicitDir(ctx, testDir, storageClient, t)
}

func CreateUnsupportedImplicitDirectoryStructure(ctx context.Context, testDir string, storageClient *storage.Client, t *testing.T) {
	// Unsupported Implicit Directory Structure
	// testBucket/testDir/implicitDirectory                                                 -- Dir
	// testBucket/testDir/implicitDirectory//fileInUnsupportedImplicitDir1                  -- File
	// testBucket/testDir/implicitDirectory/./fileInUnsupportedImplicitDir2                 -- File
	// testBucket/testDir/implicitDirectory/../fileInUnsupportedImplicitDir3                -- File
	// testBucket//FileInUnsupportedPathInRootDirectory                                     -- File
	// testBucket/./FileInUnsupportedPathInRootDirectory2                                   -- File
	// testBucket/../FileInUnsupportedPathInRootDirectory3                                  -- File

	completeGcsTestDirName := path.Join(testDir, ImplicitDirectory)
	createObjectsOnGcs(ctx,
		[]objectCreationMetadata{
			{
				"This is testBucket/testDir//fileInUnsupportedImplicitDir1", completeGcsTestDirName + "//" + FileInUnsupportedImplicitDirectory1,
			}, 
			{
				"This is testBucket/testDir/./fileInUnsupportedImplicitDir2", completeGcsTestDirName + "/./" + FileInUnsupportedImplicitDirectory2,
			},
			{
				"This is testBucket/testDir/../fileInUnsupportedImplicitDir3", completeGcsTestDirName + "/../" + FileInUnsupportedImplicitDirectory3,
			},
			{
				"This is testBucket//fileInUnsupportedPathInRootDirectory", "/" + FileInUnsupportedPathInRootDirectory,
			}, 
			{
				"This is testBucket/./fileInUnsupportedPathInRootDirectory2", "./" + FileInUnsupportedPathInRootDirectory2,
			},
			{
				"This is testBucket/../fileInUnsupportedPathInRootDirectory3", "../" + FileInUnsupportedPathInRootDirectory3,
			},
		},
		storageClient, t)
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
	defer operations.CloseFile(file)
}

func CreateImplicitDirectoryInExplicitDirectoryStructure(ctx context.Context, testDir string, storageClient *storage.Client, t *testing.T) {
	// testBucket/testDir/explicitDirectory                                                                   -- Dir
	// testBucket/testDir/explictFile                                                                         -- File
	// testBucket/testDir/explicitDirectory/fileInExplicitDir1                                                -- File
	// testBucket/testDir/explicitDirectory/fileInExplicitDir2                                                -- File
	// testBucket/testDir/explicitDirectory/implicitDirectory                                                 -- Dir
	// testBucket/testDir/explicitDirectory/implicitDirectory/fileInImplicitDir1                              -- File
	// testBucket/testDir/explicitDirectory/implicitDirectory/implicitSubDirectory                            -- Dir
	// testBucket/testDir/explicitDirectory/implicitDirectory/implicitSubDirectory/fileInImplicitDir2         -- File

	CreateExplicitDirectoryStructure(testDir, t)
	createObjectsInImplicitDir(ctx, path.Join(testDir, ExplicitDirectory), storageClient, t)
}
