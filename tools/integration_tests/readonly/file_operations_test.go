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
// copy, rename, open file operations
package readonly_test

import (
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/setup"
)

// Copy srcFile in testBucket/Test/b/b.txt destination.
func checkIfFileCopyFailed(srcFilePath string, t *testing.T) {
	source, err := os.OpenFile(srcFilePath, syscall.O_DIRECT, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("Error in the opening file: %v", err)
	}

	copyFile := path.Join(setup.MntDir(), DirectoryNameInTestBucket, SubDirectoryNameInTestBucket, "b.txt")
	if _, err := os.Stat(copyFile); err != nil {
		t.Errorf("Copied file %s is not present", copyFile)
	}

	destination, err := os.OpenFile(copyFile, syscall.O_DIRECT, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("File %s opening error: %v", destination.Name(), err)
	}
	defer destination.Close()

	// File copying with io.Copy() utility.
	_, err = io.Copy(destination, source)
	// Throwing an error "copy_file_range: bad file descriptor"
	if err == nil {
		t.Errorf("File copied in read-only file system.")
	}
}

// Copy testBucket/Test1.txt to testBucket/Test/b/b.txt
func TestCopyFile(t *testing.T) {
	file := path.Join(setup.MntDir(), FileNameInTestBucket)

	checkIfFileCopyFailed(file, t)
}

// Copy testBucket/Test/a.txt to testBucket/Test/b/b.txt
func TestCopyFileFromSubDirectory(t *testing.T) {
	file := path.Join(setup.MntDir(), DirectoryNameInTestBucket, FileInSubDirectoryNameInTestBucket)

	checkIfFileCopyFailed(file, t)
}

func checkIfDirCopyFailed(srcDirPath string, newDir string, t *testing.T) {
	cmd := exec.Command("cp", "--recursive", srcDirPath, newDir)
	err := cmd.Run()

	// Throwing an  exit status 1
	if err == nil {
		t.Errorf("Dir copied in read-only file system.")
	}
}

// Copy testBucket/Test to testBucket/Test/b
func TestCopyDir(t *testing.T) {
	srcDir := path.Join(setup.MntDir(), DirectoryNameInTestBucket)
	destDir := path.Join(setup.MntDir(), DirectoryNameInTestBucket, SubDirectoryNameInTestBucket)

	checkIfDirCopyFailed(srcDir, destDir, t)
}

// Copy testBucket/Test/b to testBucket/Test
func TestCopySubDirectory(t *testing.T) {
	srcDir := path.Join(setup.MntDir(), DirectoryNameInTestBucket, SubDirectoryNameInTestBucket)
	destDir := path.Join(setup.MntDir(), DirectoryNameInTestBucket)

	checkIfDirCopyFailed(srcDir, destDir, t)
}

// Rename srcFile to Rename.txt
func checkIfFileRenameFailed(oldObjPath string, newObjPath string, t *testing.T) {
	_, err := os.Stat(oldObjPath)
	if err != nil {
		t.Errorf("Error in the stating object: %v", err)
	}

	if _, err := os.Stat(newObjPath); err == nil {
		t.Errorf("Renamed file %s already present", newObjPath)
	}

	err = os.Rename(oldObjPath, newObjPath)

	if err == nil {
		t.Errorf("File renamed in read-only file system.")
	}

	// It will throw an error read-only file system or permission denied.
	if !strings.Contains(err.Error(), "read-only file system") && !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("Throwing incorrect error.")
	}

	if _, err := os.Stat(oldObjPath); err != nil {
		t.Errorf("SrcFile is deleted in read-only file system.")
	}
	if _, err := os.Stat(newObjPath); err == nil {
		t.Errorf("Renamed file found in read-only file system.")
	}
}

// Rename testBucket/Test1.txt to testBucket/Rename.txt
func TestRenameFile(t *testing.T) {
	oldFilePath := path.Join(setup.MntDir(), FileNameInTestBucket)
	newFilePath := path.Join(setup.MntDir(), "Rename.txt")

	checkIfFileRenameFailed(oldFilePath, newFilePath, t)
}

// Rename testBucket/Test/a.txt to testBucket/Test/Rename.txt
func TestRenameFileFromSubDirectory(t *testing.T) {
	oldFilePath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, FileInSubDirectoryNameInTestBucket)
	newFilePath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, "Rename.txt")

	checkIfFileRenameFailed(oldFilePath, newFilePath, t)
}

// Rename testBucket/Test to testBucket/Rename
func TestRenameDir(t *testing.T) {
	oldDirPath := path.Join(setup.MntDir(), DirectoryNameInTestBucket)
	newDirPath := path.Join(setup.MntDir(), "Rename")

	checkIfFileRenameFailed(oldDirPath, newDirPath, t)
}

// Rename testBucket/Test/b to testBucket/Test/Rename
func TestRenameSubDirectory(t *testing.T) {
	oldDirPath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, SubDirectoryNameInTestBucket)
	newDirPath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, "Rename")

	checkIfFileRenameFailed(oldDirPath, newDirPath, t)
}
