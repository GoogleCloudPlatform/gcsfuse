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

// Provides integration tests for file operations with --o=ro flag set.
package readonly_test

import (
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

// Rename oldFile to newFile
func checkIfRenameFileFailed(oldFilePath string, newFilePath string, t *testing.T) {
	err := operations.RenameFile(oldFilePath, newFilePath)
	if err == nil {
		t.Errorf("File renamed in read-only file system.")
	}

	checkErrorForReadOnlyFileSystem(err, t)

	if _, err := os.Stat(oldFilePath); err != nil {
		t.Errorf("Old file is deleted in read-only file system.")
	}
	if _, err := os.Stat(newFilePath); err == nil {
		t.Errorf("Renamed file found in read-only file system.")
	}
}

// Rename oldDir to newDir
func checkIfRenameDirFailed(oldDirPath string, newDirPath string, t *testing.T) {
	err := operations.RenameDir(oldDirPath, newDirPath)
	if err == nil {
		t.Errorf("Directory renamed in read-only file system.")
	}

	checkErrorForReadOnlyFileSystem(err, t)

	if _, err := os.Stat(oldDirPath); err != nil {
		t.Errorf("Old directory is deleted in read-only file system.")
	}
	if _, err := os.Stat(newDirPath); err == nil {
		t.Errorf("Renamed directory found in read-only file system.")
	}
}

// Rename testBucket/Test1.txt to testBucket/Rename.txt
func TestRenameFile(t *testing.T) {
	oldFilePath := path.Join(setup.MntDir(), FileNameInTestBucket)
	newFilePath := path.Join(setup.MntDir(), RenameFile)

	checkIfRenameFileFailed(oldFilePath, newFilePath, t)
}

// Rename testBucket/Test/a.txt to testBucket/Test/Rename.txt
func TestRenameFileFromBucketDirectory(t *testing.T) {
	oldFilePath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, FileNameInDirectoryTestBucket)
	newFilePath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, RenameFile)

	checkIfRenameFileFailed(oldFilePath, newFilePath, t)
}

// Rename testBucket/Test/b/b.txt to testBucket/Test/b/Rename.txt
func TestRenameFileFromBucketSubDirectory(t *testing.T) {
	oldFilePath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, SubDirectoryNameInTestBucket, FileNameInSubDirectoryTestBucket)
	newFilePath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, SubDirectoryNameInTestBucket, RenameFile)

	checkIfRenameFileFailed(oldFilePath, newFilePath, t)
}

// Rename testBucket/Test to testBucket/Rename
func TestRenameDir(t *testing.T) {
	oldDirPath := path.Join(setup.MntDir(), DirectoryNameInTestBucket)
	newDirPath := path.Join(setup.MntDir(), RenameDir)

	checkIfRenameDirFailed(oldDirPath, newDirPath, t)

	// Ensure none of the child is deleted during the directory rename test.
	// ** OldDirectory structure **
	// Test
	// Test/b      -- Dir
	// Test/a.txt  -- File

	obj, err := os.ReadDir(oldDirPath)
	if err != nil {
		t.Errorf("Error in reading directory %v ,", err.Error())
	}

	// Comparing number of objects in the oldDirectory - 2
	if len(obj) != NumberOfObjectsInDirectoryTestBucket {
		t.Errorf("The number of objects in the current directory doesn't match.")
	}

	// Comparing first object name and type
	// Name - Test/a.txt, Type - File
	if obj[0].Name() != FileNameInDirectoryTestBucket || obj[0].IsDir() != false {
		t.Errorf("Object Listed for file in bucket is incorrect.")
	}

	// Comparing second object name and type
	// Name - Test/b , Type - Dir
	if obj[1].Name() != SubDirectoryNameInTestBucket || obj[1].IsDir() != true {
		t.Errorf("Object Listed for bucket directory is incorrect.")
	}
}

// Rename testBucket/Test/b to testBucket/Test/Rename
func TestRenameSubDirectory(t *testing.T) {
	oldDirPath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, SubDirectoryNameInTestBucket)
	newDirPath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, RenameDir)

	checkIfRenameDirFailed(oldDirPath, newDirPath, t)

	// Ensure none of the child is deleted during the directory rename test.
	// ** OldDirectory structure **
	// b
	// b/b.txt   -- File

	obj, err := os.ReadDir(oldDirPath)
	if err != nil {
		t.Errorf("Error in reading directory %v ,", err.Error())
	}

	// Comparing number of objects in the oldDirectory - 1
	if len(obj) != NumberOfObjectsInSubDirectoryTestBucket {
		t.Errorf("The number of objects in the current directory doesn't match.")
	}

	// Comparing object name and type
	// Name - b/b.txt, Type - File
	if obj[0].Name() != FileNameInSubDirectoryTestBucket || obj[0].IsDir() != false {
		t.Errorf("Object Listed for file in bucket SubDirectory is incorrect.")
	}
}
