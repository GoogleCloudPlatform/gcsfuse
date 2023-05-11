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
package rename_directory_test

import (
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/setup"
)

func createDir(dirPath string, t *testing.T) {
	err := os.Mkdir(dirPath, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("Error in creating directory: %v", err)
	}
}

func createFile(filePath string, t *testing.T) {
	_, err := os.Create(filePath)
	if err != nil {
		t.Errorf("Error in creating file: %v", err)
	}
}

// Directory structure
// testBucket/directoryWithThreeFiles           -- Dir
// testBucket/directoryWithThreeFiles/a.txt     -- File
// testBucket/directoryWithThreeFiles/b.txt     -- File
// testBucket/directoryWithThreeFiles/c.txt     -- File
func createDirectoryWithThreeFiles(t *testing.T) {
	dirPath := path.Join(setup.MntDir(), DirectoryWithThreeFiles)

	createDir(dirPath, t)

	filePath1 := path.Join(dirPath, "a.txt")
	filePath2 := path.Join(dirPath, "b.txt")
	filePath3 := path.Join(dirPath, "c.txt")

	createFile(filePath1, t)
	createFile(filePath2, t)
	createFile(filePath3, t)
}

// Directory structure
// testBucket/directoryWithTwoFiles          -- Dir
// testBucket/directoryWithTwoFiles/a.txt    -- File
// testBucket/directoryWithTwoFiles/b.txt    -- File
func createDirectoryWithTwoFiles(t *testing.T) {
	dirPath := path.Join(setup.MntDir(), DirectoryWithTwoFiles)

	createDir(dirPath, t)

	filePath1 := path.Join(dirPath, "a.txt")
	filePath2 := path.Join(dirPath, "b.txt")

	createFile(filePath1, t)
	createFile(filePath2, t)
}

// Directory structure
// testBucket/directoryWithFourFiles          -- Dir
// testBucket/directoryWithFourFiles/a.txt    -- File
// testBucket/directoryWithFourFiles/b.txt    -- File
// testBucket/directoryWithFourFiles/c.txt    -- File
// testBucket/directoryWithFourFiles/d.txt    -- File
func createDirectoryWithFourFiles(t *testing.T) {
	dirPath := path.Join(setup.MntDir(), DirectoryWithFourFiles)

	createDir(dirPath, t)

	filePath1 := path.Join(dirPath, "a.txt")
	filePath2 := path.Join(dirPath, "b.txt")
	filePath3 := path.Join(dirPath, "c.txt")
	filePath4 := path.Join(dirPath, "d.txt")

	createFile(filePath1, t)
	createFile(filePath2, t)
	createFile(filePath3, t)
	createFile(filePath4, t)
}

// Directory structure
// testBucket/directoryWithTwoFilesOneEmptyDirectory                       -- Dir
// testBucket/directoryWithTwoFilesOneEmptyDirectory/a.txt                 -- File
// testBucket/directoryWithTwoFilesOneEmptyDirectory/b.txt                 -- File
// testBucket/directoryWithTwoFilesOneEmptyDirectory/emptySubDirectory     -- Dir
func createDirectoryWithTwoFilesOneEmptyDirectory(t *testing.T) {
	dirPath := path.Join(setup.MntDir(), DirectoryWithTwoFilesOneEmptyDirectory)

	createDir(dirPath, t)

	subDir := path.Join(setup.MntDir(), DirectoryWithTwoFilesOneEmptyDirectory, EmptySubDirectory)

	createDir(subDir, t)

	filePath1 := path.Join(dirPath, "a.txt")
	filePath2 := path.Join(dirPath, "b.txt")

	createFile(filePath1, t)
	createFile(filePath2, t)
}

// Directory structure
// testBucket/directoryWithTwoFilesOneNonEmptyDirectory                                  -- Dir
// testBucket/directoryWithTwoFilesOneNonEmptyDirectory/a.txt                            -- File
// testBucket/directoryWithTwoFilesOneNonEmptyDirectory/b.txt                            -- File
// testBucket/directoryWithTwoFilesOneNonEmptyDirectory/NonEmptySubDirectory             -- Dir
// testBucket/directoryWithTwoFilesOneNonEmptyDirectory/NonEmptySubDirectory/c.txt   		 -- File
func createDirectoryWithTwoFilesOneNonEmptyDirectory(t *testing.T) {
	dirPath := path.Join(setup.MntDir(), DirectoryWithTwoFilesOneNonEmptyDirectory)

	createDir(dirPath, t)

	subDir := path.Join(dirPath, NonEmptySubDirectory)

	createDir(subDir, t)

	filePath1 := path.Join(dirPath, "a.txt")
	filePath2 := path.Join(dirPath, "b.txt")
	filePath3 := path.Join(subDir, "c.txt")

	createFile(filePath1, t)
	createFile(filePath2, t)
	createFile(filePath3, t)
}

// As --rename-directory-limit = 3, the operation should get successful.
func TestRenameDirectoryWithThreeFiles(t *testing.T) {
	createDirectoryWithThreeFiles(t)

	oldDirPath := path.Join(setup.MntDir(), DirectoryWithThreeFiles)
	newDirPath := path.Join(setup.MntDir(), RenamedDirectory)

	//  Cleaning the directory before renaming.
	os.RemoveAll(newDirPath)

	err := os.Rename(oldDirPath, newDirPath)

	if err != nil {
		t.Errorf("Error in renaming directory: %v", err)
	}
}

func TestRenameDirectoryWithTwoFiles(t *testing.T) {
	createDirectoryWithTwoFiles(t)

	oldDirPath := path.Join(setup.MntDir(), DirectoryWithTwoFiles)
	newDirPath := path.Join(setup.MntDir(), RenamedDirectory)

	//  Cleaning the directory before renaming.
	os.RemoveAll(newDirPath)

	err := os.Rename(oldDirPath, newDirPath)

	if err != nil {
		t.Errorf("Error in renaming directory: %v", err)
	}
}

func TestRenameDirectoryWithFourFiles(t *testing.T) {
	createDirectoryWithFourFiles(t)

	oldDirPath := path.Join(setup.MntDir(), DirectoryWithFourFiles)
	newDirPath := path.Join(setup.MntDir(), RenamedDirectory)

	//  Cleaning the directory before renaming.
	os.RemoveAll(newDirPath)

	err := os.Rename(oldDirPath, newDirPath)

	if err == nil {
		t.Errorf("Renaming directory succeeded with objects greater than rename-dir-limit.")
	}
}

func TestRenameDirectoryWithTwoFilesAndOneEmptyDirectory(t *testing.T) {
	createDirectoryWithTwoFilesOneEmptyDirectory(t)

	oldDirPath := path.Join(setup.MntDir(), DirectoryWithTwoFilesOneEmptyDirectory)
	newDirPath := path.Join(setup.MntDir(), RenamedDirectory)

	//  Cleaning the directory before renaming.
	os.RemoveAll(newDirPath)

	err := os.Rename(oldDirPath, newDirPath)

	if err != nil {
		t.Errorf("Error in renaming directory: %v", err)
	}
}

func TestRenameDirectoryWithTwoFilesAndOneNonEmptyDirectory(t *testing.T) {
	createDirectoryWithTwoFilesOneNonEmptyDirectory(t)

	oldDirPath := path.Join(setup.MntDir(), DirectoryWithTwoFilesOneNonEmptyDirectory)
	newDirPath := path.Join(setup.MntDir(), RenamedDirectory)

	//  Cleaning the directory before renaming.
	os.RemoveAll(newDirPath)

	err := os.Rename(oldDirPath, newDirPath)

	if err == nil {
		t.Errorf("Renaming directory succeeded with objects greater than rename-dir-limit.")
	}
}
