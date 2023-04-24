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
	"path"
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/setup"
)

// Copy srcFile in testBucket/Test/b/b.txt destination.
func ensureFileSystemLockedForFileCopy(srcFilePath string, t *testing.T) {
	_, err := os.OpenFile(srcFilePath, syscall.O_DIRECT, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("Error in the opening file: %v", err)
	}

	copyFile := path.Join(setup.MntDir(), DirectoryNameInTestBucket, SubDirectoryNameInTestBucket, "b.txt")
	if _, err := os.Stat(copyFile); err != nil {
		t.Errorf("Copied file %s is not present", copyFile)
	}

	source, err := os.OpenFile(srcFilePath, syscall.O_DIRECT, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("File %s opening error: %v", source.Name(), err)
	}
	defer source.Close()

	destination, err := os.OpenFile(copyFile, syscall.O_DIRECT, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("File %s opening error: %v", destination.Name(), err)
	}
	defer destination.Close()

	// File copying with io.Copy() utility.
	_, err = io.Copy(destination, source)
	if err == nil {
		t.Errorf("File copied in read-only file system.")
	}
}

// Copy testBucket/Test1.txt to testBucket/Test/b/b.txt
func TestCopyFile(t *testing.T) {
	file := path.Join(setup.MntDir(), FileNameInTestBucket)

	ensureFileSystemLockedForFileCopy(file, t)
}

// Copy testBucket/Test/a.txt to testBucket/Test/b/b.txt
func TestCopyFileFromSubDirectory(t *testing.T) {
	file := path.Join(setup.MntDir(), DirectoryNameInTestBucket, FileInSubDirectoryNameInTestBucket)

	ensureFileSystemLockedForFileCopy(file, t)
}

// Rename srcFile to Rename.txt
func ensureFileSystemLockedForFileRename(filePath string, dirPath string, t *testing.T) {
	file, err := os.OpenFile(filePath, syscall.O_DIRECT, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("Error in the opening file: %v", err)
	}
	defer file.Close()

	newFileName := path.Join(dirPath, "Rename.txt")
	if _, err := os.Stat(newFileName); err == nil {
		t.Errorf("Renamed file %s already present", newFileName)
	}

	if err := os.Rename(file.Name(), newFileName); err == nil {
		t.Errorf("File renamed in read-only file system.")
	}

	if _, err := os.Stat(file.Name()); err != nil {
		t.Errorf("SrcFile is deleted in read-only file system.")
	}
	if _, err := os.Stat(newFileName); err == nil {
		t.Errorf("Renamed file found in read-only file system.")
	}
}

// Rename testBucket/Test1.txt to testBucket/Rename.txt
func TestRenameFile(t *testing.T) {
	filePath := path.Join(setup.MntDir(), FileNameInTestBucket)

	ensureFileSystemLockedForFileRename(filePath, setup.MntDir(), t)
}

// Rename testBucket/Test/a.txt to testBucket/Test/Rename.txt
func TestRenameFileFromSubDirectory(t *testing.T) {
	filePath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, FileInSubDirectoryNameInTestBucket)
	dirPath := path.Join(setup.MntDir(), DirectoryNameInTestBucket)

	ensureFileSystemLockedForFileRename(filePath, dirPath, t)
}

func ensureFileSystemLockedToWriteOrUpdateContent(filePath string, t *testing.T) {
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT, setup.FilePermission_0600)
	if err == nil {
		t.Errorf("File opened for writing in read-only mount.")
	}
	defer file.Close()
}

func TestOpenFileToWriteOrUpdateContent(t *testing.T) {
	filePath := path.Join(setup.MntDir(), FileNameInTestBucket)

	ensureFileSystemLockedToWriteOrUpdateContent(filePath, t)
}

func TestOpenFileFromSubDirectoryToWriteOrUpdateContent(t *testing.T) {
	filePath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, FileInSubDirectoryNameInTestBucket)

	ensureFileSystemLockedToWriteOrUpdateContent(filePath, t)
}

func TestOpenNonExistentFileToWriteOrUpdateContent(t *testing.T) {
	filePath := path.Join(setup.MntDir(), FileNotExist)

	ensureFileSystemLockedToWriteOrUpdateContent(filePath, t)
}

func TestOpenNonExistentFileFromSubDirectoryToWriteOrUpdateContent(t *testing.T) {
	filePath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, FileNotExist)

	ensureFileSystemLockedToWriteOrUpdateContent(filePath, t)
}

func ensureFileSystemLockedToAppendContent(filePath string, t *testing.T) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT, setup.FilePermission_0600)
	if err == nil {
		t.Errorf("File opened for appending Content in read-only mount.")
	}
	defer file.Close()
}

func TestOpenFileToAppendContent(t *testing.T) {
	filePath := path.Join(setup.MntDir(), FileNameInTestBucket)

	ensureFileSystemLockedToAppendContent(filePath, t)
}

func TestOpenFileFromSubDirectoryToAppendContent(t *testing.T) {
	filePath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, FileInSubDirectoryNameInTestBucket)

	ensureFileSystemLockedToAppendContent(filePath, t)
}

func TestOpenNonExistentFileToAppendContent(t *testing.T) {
	filePath := path.Join(setup.MntDir(), FileNotExist)

	ensureFileSystemLockedToAppendContent(filePath, t)
}

func TestOpenNonExistentFileFromSubDirectoryToAppendContent(t *testing.T) {
	filePath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, FileNotExist)

	ensureFileSystemLockedToAppendContent(filePath, t)
}
