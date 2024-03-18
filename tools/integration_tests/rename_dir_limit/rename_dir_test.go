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

// Provides integration tests when --rename-dir-limit flag is set.
package rename_dir_limit_test

import (
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

// As --rename-directory-limit = 3, and the number of objects in the directory is three,
// which is equal to the limit, the operation should get successful.
func TestRenameDirectoryWithThreeFiles(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	// Create directory structure
	// testBucket/directoryWithThreeFiles               -- Dir
	// testBucket/directoryWithThreeFiles/temp1.txt     -- File
	// testBucket/directoryWithThreeFiles/temp2.txt     -- File
	// testBucket/directoryWithThreeFiles/temp3.txt     -- File
	dirPath := path.Join(setup.MntDir(), DirectoryWithThreeFiles)
	operations.CreateDirectoryWithNFiles(3, dirPath, PrefixTempFile, t)

	oldDirPath := path.Join(setup.MntDir(), DirectoryWithThreeFiles)
	newDirPath := path.Join(setup.MntDir(), RenamedDirectory)

	//  Cleaning the directory before renaming.
	operations.RemoveDir(newDirPath)

	err := os.Rename(oldDirPath, newDirPath)

	if err != nil {
		t.Errorf("Error in renaming directory: %v", err)
	}
}

// As --rename-directory-limit = 3, and the number of objects in the directory is two,
// which is less than the limit, the operation should get successful.
func TestRenameDirectoryWithTwoFiles(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	// Create directory structure
	// testBucket/directoryWithTwoFiles              -- Dir
	// testBucket/directoryWithTwoFiles/temp1.txt    -- File
	// testBucket/directoryWithTwoFiles/temp2.txt    -- File
	dirPath := path.Join(setup.MntDir(), DirectoryWithTwoFiles)

	operations.CreateDirectoryWithNFiles(2, dirPath, PrefixTempFile, t)

	oldDirPath := path.Join(setup.MntDir(), DirectoryWithTwoFiles)
	newDirPath := path.Join(setup.MntDir(), RenamedDirectory)

	//  Cleaning the directory before renaming.
	operations.RemoveDir(newDirPath)

	err := os.Rename(oldDirPath, newDirPath)

	if err != nil {
		t.Errorf("Error in renaming directory: %v", err)
	}
}

// As --rename-directory-limit = 3, and the number of objects in the directory is two,
// which is greater than the limit, the operation should get fail.
func TestRenameDirectoryWithFourFiles(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	// Creating directory structure
	// testBucket/directoryWithFourFiles              -- Dir
	// testBucket/directoryWithFourFiles/temp1.txt    -- File
	// testBucket/directoryWithFourFiles/temp2.txt    -- File
	// testBucket/directoryWithFourFiles/temp3.txt    -- File
	// testBucket/directoryWithFourFiles/temp4.txt    -- File
	dirPath := path.Join(setup.MntDir(), DirectoryWithFourFiles)

	operations.CreateDirectoryWithNFiles(4, dirPath, PrefixTempFile, t)

	oldDirPath := path.Join(setup.MntDir(), DirectoryWithFourFiles)
	newDirPath := path.Join(setup.MntDir(), RenamedDirectory)

	//  Cleaning the directory before renaming.
	operations.RemoveDir(newDirPath)

	err := os.Rename(oldDirPath, newDirPath)

	if err == nil {
		t.Errorf("Renaming directory succeeded with objects greater than rename-dir-limit.")
	}
}

// As --rename-directory-limit = 3, and the number of objects in the directory is three,
// which is equal to limit, the operation should get successful.
func TestRenameDirectoryWithTwoFilesAndOneEmptyDirectory(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	// Creating directory structure
	// testBucket/directoryWithTwoFilesOneEmptyDirectory                       -- Dir
	// testBucket/directoryWithTwoFilesOneEmptyDirectory/a.txt                 -- File
	// testBucket/directoryWithTwoFilesOneEmptyDirectory/b.txt                 -- File
	// testBucket/directoryWithTwoFilesOneEmptyDirectory/emptySubDirectory     -- Dir
	dirPath := path.Join(setup.MntDir(), DirectoryWithTwoFilesOneEmptyDirectory)
	subDirPath := path.Join(setup.MntDir(), DirectoryWithTwoFilesOneEmptyDirectory, EmptySubDirectory)

	operations.CreateDirectoryWithNFiles(2, dirPath, PrefixTempFile, t)
	operations.CreateDirectoryWithNFiles(0, subDirPath, PrefixTempFile, t)

	oldDirPath := path.Join(setup.MntDir(), DirectoryWithTwoFilesOneEmptyDirectory)
	newDirPath := path.Join(setup.MntDir(), RenamedDirectory)

	//  Cleaning the directory before renaming.
	operations.RemoveDir(newDirPath)

	err := os.Rename(oldDirPath, newDirPath)

	if err != nil {
		t.Errorf("Error in renaming directory: %v", err)
	}
}

// As --rename-directory-limit = 3, and the number of objects in the directory is Four,
// which is greater than the limit, the operation should get fail.
func TestRenameDirectoryWithTwoFilesAndOneNonEmptyDirectory(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	// Creating directory structure
	// testBucket/directoryWithTwoFilesOneNonEmptyDirectory                                      -- Dir
	// testBucket/directoryWithTwoFilesOneNonEmptyDirectory/temp1.txt                            -- File
	// testBucket/directoryWithTwoFilesOneNonEmptyDirectory/temp2.txt                            -- File
	// testBucket/directoryWithTwoFilesOneNonEmptyDirectory/NonEmptySubDirectory                 -- Dir
	// testBucket/directoryWithTwoFilesOneNonEmptyDirectory/NonEmptySubDirectory/temp3.txt   		 -- File

	dirPath := path.Join(setup.MntDir(), DirectoryWithTwoFilesOneNonEmptyDirectory)
	subDirPath := path.Join(dirPath, NonEmptySubDirectory)

	operations.CreateDirectoryWithNFiles(2, dirPath, PrefixTempFile, t)
	operations.CreateDirectoryWithNFiles(1, subDirPath, PrefixTempFile, t)

	oldDirPath := path.Join(setup.MntDir(), DirectoryWithTwoFilesOneNonEmptyDirectory)
	newDirPath := path.Join(setup.MntDir(), RenamedDirectory)

	//  Cleaning the directory before renaming.
	operations.RemoveDir(newDirPath)

	err := os.Rename(oldDirPath, newDirPath)

	if err == nil {
		t.Errorf("Renaming directory succeeded with objects greater than rename-dir-limit.")
	}
}
