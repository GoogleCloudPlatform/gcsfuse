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
	"log"
	"os"
	"path"
	"testing"

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

	setup.RemoveBinFileCopiedForTesting()
	return successCode
}

func RemoveAndCheckIfDirIsDeleted(dirPath string, dirName string, t *testing.T) {
	operations.RemoveDir(dirPath)

	dir, err := os.Stat(dirPath)
	if err == nil && dir.Name() == dirName && dir.IsDir() {
		t.Errorf("Directory is not deleted.")
	}
}

func CreateImplicitDirectoryStructure() {
	// Implicit Directory Structure
	// testBucket/implicitDirectory                                                  -- Dir
	// testBucket/implicitDirectory/fileInImplicitDir1                               -- File
	// testBucket/implicitDirectory/implicitSubDirectory                             -- Dir
	// testBucket/implicitDirectory/implicitSubDirectory/fileInImplicitDir2          -- File

	// Clean the bucket.
	setup.RunScriptForTestData("../util/setup/implicit_and_explicit_dir_setup/testdata/delete_objects.sh", setup.TestBucket())

	// Create implicit directory in bucket for testing.
	setup.RunScriptForTestData("../util/setup/implicit_and_explicit_dir_setup/testdata/create_objects.sh", setup.TestBucket())
}

func CreateExplicitDirectoryStructure(t *testing.T) {
	// Explicit Directory structure
	// testBucket/explicitDirectory                            -- Dir
	// testBucket/explictFile                                  -- File
	// testBucket/explicitDirectory/fileInExplicitDir1         -- File
	// testBucket/explicitDirectory/fileInExplicitDir2         -- File

	dirPath := path.Join(setup.MntDir(), ExplicitDirectory)
	operations.CreateDirectoryWithNFiles(NumberOfFilesInExplicitDirectory, dirPath, PrefixFileInExplicitDirectory, t)
	filePath := path.Join(setup.MntDir(), ExplicitFile)
	file, err := os.Create(filePath)
	if err != nil {
		t.Errorf("Create file at %q: %v", setup.MntDir(), err)
	}

	// Closing file at the end.
	defer operations.CloseFile(file)
}

func CreateImplicitDirectoryInExplicitDirectoryStructure(t *testing.T) {
	// testBucket/explicitDirectory                                                                   -- Dir
	// testBucket/explictFile                                                                         -- File
	// testBucket/explicitDirectory/fileInExplicitDir1                                                -- File
	// testBucket/explicitDirectory/fileInExplicitDir2                                                -- File
	// testBucket/explicitDirectory/implicitDirectory                                                 -- Dir
	// testBucket/explicitDirectory/implicitDirectory/fileInImplicitDir1                              -- File
	// testBucket/explicitDirectory/implicitDirectory/implicitSubDirectory                            -- Dir
	// testBucket/explicitDirectory/implicitDirectory/implicitSubDirectory/fileInImplicitDir2         -- File

	CreateExplicitDirectoryStructure(t)
	dirPathInBucket := path.Join(setup.TestBucket(), ExplicitDirectory)
	setup.RunScriptForTestData("../util/setup/implicit_and_explicit_dir_setup/testdata/create_objects.sh", dirPathInBucket)
}
