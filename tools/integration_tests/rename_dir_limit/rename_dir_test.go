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

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func createDirectoryWithNFiles(numberOfFiles int, dirPath string, t *testing.T) {
	err := os.Mkdir(dirPath, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("Error in creating directory: %v", err)
	}

	for i := 0; i < numberOfFiles; i++ {
		_, err := os.CreateTemp(dirPath, "tmpFile")
		if err != nil {
			t.Errorf("Create file at %q: %v", dirPath, err)
		}
	}
}

// As --rename-directory-limit = 3, and the number of objects in the directory is three,
// which is equal to the limit, the operation should get successful.
func TestRenameDirectoryWithThreeFiles(t *testing.T) {
	// Create directory structure
	// testBucket/directoryWithThreeFiles               -- Dir
	// testBucket/directoryWithThreeFiles/temp1.txt     -- File
	// testBucket/directoryWithThreeFiles/temp2.txt     -- File
	// testBucket/directoryWithThreeFiles/temp3.txt     -- File
	dirPath := path.Join(setup.MntDir(), DirectoryWithThreeFiles)
	createDirectoryWithNFiles(3, dirPath, t)

	oldDirPath := path.Join(setup.MntDir(), DirectoryWithThreeFiles)
	newDirPath := path.Join(setup.MntDir(), RenamedDirectory)

	//  Cleaning the directory before renaming.
	os.RemoveAll(newDirPath)

	err := os.Rename(oldDirPath, newDirPath)

	if err != nil {
		t.Errorf("Error in renaming directory: %v", err)
	}
}

// As --rename-directory-limit = 3, and the number of objects in the directory is two,
// which is less than the limit, the operation should get successful.
func TestRenameDirectoryWithTwoFiles(t *testing.T) {
	// Create directory structure
	// testBucket/directoryWithTwoFiles              -- Dir
	// testBucket/directoryWithTwoFiles/temp1.txt    -- File
	// testBucket/directoryWithTwoFiles/temp2.txt    -- File
	dirPath := path.Join(setup.MntDir(), DirectoryWithTwoFiles)

	createDirectoryWithNFiles(2, dirPath, t)

	oldDirPath := path.Join(setup.MntDir(), DirectoryWithTwoFiles)
	newDirPath := path.Join(setup.MntDir(), RenamedDirectory)

	//  Cleaning the directory before renaming.
	os.RemoveAll(newDirPath)

	err := os.Rename(oldDirPath, newDirPath)

	if err != nil {
		t.Errorf("Error in renaming directory: %v", err)
	}
}

// As --rename-directory-limit = 3, and the number of objects in the directory is two,
// which is greater than the limit, the operation should get fail.
func TestRenameDirectoryWithFourFiles(t *testing.T) {
	// Creating directory structure
	// testBucket/directoryWithFourFiles              -- Dir
	// testBucket/directoryWithFourFiles/temp1.txt    -- File
	// testBucket/directoryWithFourFiles/temp2.txt    -- File
	// testBucket/directoryWithFourFiles/temp3.txt    -- File
	// testBucket/directoryWithFourFiles/temp4.txt    -- File
	dirPath := path.Join(setup.MntDir(), DirectoryWithFourFiles)

	createDirectoryWithNFiles(4, dirPath, t)

	oldDirPath := path.Join(setup.MntDir(), DirectoryWithFourFiles)
	newDirPath := path.Join(setup.MntDir(), RenamedDirectory)

	//  Cleaning the directory before renaming.
	os.RemoveAll(newDirPath)

	err := os.Rename(oldDirPath, newDirPath)

	if err == nil {
		t.Errorf("Renaming directory succeeded with objects greater than rename-dir-limit.")
	}
}

// As --rename-directory-limit = 3, and the number of objects in the directory is three,
// which is equal to limit, the operation should get successful.
func TestRenameDirectoryWithTwoFilesAndOneEmptyDirectory(t *testing.T) {
	// Creating directory structure
	// testBucket/directoryWithTwoFilesOneEmptyDirectory                       -- Dir
	// testBucket/directoryWithTwoFilesOneEmptyDirectory/a.txt                 -- File
	// testBucket/directoryWithTwoFilesOneEmptyDirectory/b.txt                 -- File
	// testBucket/directoryWithTwoFilesOneEmptyDirectory/emptySubDirectory     -- Dir
	dirPath := path.Join(setup.MntDir(), DirectoryWithTwoFilesOneEmptyDirectory)
	subDirPath := path.Join(setup.MntDir(), DirectoryWithTwoFilesOneEmptyDirectory, EmptySubDirectory)

	createDirectoryWithNFiles(2, dirPath, t)
	createDirectoryWithNFiles(0, subDirPath, t)

	oldDirPath := path.Join(setup.MntDir(), DirectoryWithTwoFilesOneEmptyDirectory)
	newDirPath := path.Join(setup.MntDir(), RenamedDirectory)

	//  Cleaning the directory before renaming.
	os.RemoveAll(newDirPath)

	err := os.Rename(oldDirPath, newDirPath)

	if err != nil {
		t.Errorf("Error in renaming directory: %v", err)
	}
}

// As --rename-directory-limit = 3, and the number of objects in the directory is Four,
// which is greater than the limit, the operation should get fail.
func TestRenameDirectoryWithTwoFilesAndOneNonEmptyDirectory(t *testing.T) {
	// Creating directory structure
	// testBucket/directoryWithTwoFilesOneNonEmptyDirectory                                      -- Dir
	// testBucket/directoryWithTwoFilesOneNonEmptyDirectory/temp1.txt                            -- File
	// testBucket/directoryWithTwoFilesOneNonEmptyDirectory/temp2.txt                            -- File
	// testBucket/directoryWithTwoFilesOneNonEmptyDirectory/NonEmptySubDirectory                 -- Dir
	// testBucket/directoryWithTwoFilesOneNonEmptyDirectory/NonEmptySubDirectory/temp3.txt   		 -- File

	dirPath := path.Join(setup.MntDir(), DirectoryWithTwoFilesOneNonEmptyDirectory)
	subDirPath := path.Join(dirPath, NonEmptySubDirectory)

	createDirectoryWithNFiles(2, dirPath, t)
	createDirectoryWithNFiles(1, subDirPath, t)

	oldDirPath := path.Join(setup.MntDir(), DirectoryWithTwoFilesOneNonEmptyDirectory)
	newDirPath := path.Join(setup.MntDir(), RenamedDirectory)

	//  Cleaning the directory before renaming.
	os.RemoveAll(newDirPath)

	err := os.Rename(oldDirPath, newDirPath)

	if err == nil {
		t.Errorf("Renaming directory succeeded with objects greater than rename-dir-limit.")
	}
}
