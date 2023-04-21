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
	"io/fs"
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/setup"
)

func ensureFileSystemLockedForFileCreation(filePath string, t *testing.T) {
	file, err := os.OpenFile(filePath, os.O_CREATE, setup.FilePermission_0600)

	// It will throw an error read-only file system or permission denied.
	if err == nil {
		t.Errorf("File is created in read-only file system.")
	}

	defer file.Close()
}

func TestCreateFile(t *testing.T) {
	filePath := path.Join(setup.MntDir(), "testFile.txt")
	ensureFileSystemLockedForFileCreation(filePath, t)
}

func TestCreateFileInSubDirectory(t *testing.T) {
	filePath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, "testFile.txt")
	ensureFileSystemLockedForFileCreation(filePath, t)
}

func ensureFileSystemLockedForDirCreation(dirPath string, t *testing.T) {
	err := os.Mkdir(dirPath, fs.ModeDir)

	// It will throw an error read-only file system or permission denied.
	if err == nil {
		t.Errorf("Directory is created in read-only file system.")
	}
}

func TestCreateDir(t *testing.T) {
	dirPath := path.Join(setup.MntDir(), "test")
	ensureFileSystemLockedForDirCreation(dirPath, t)
}

func TestCreateDirInSubDirectory(t *testing.T) {
	dirPath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, "test")
	ensureFileSystemLockedForDirCreation(dirPath, t)
}

func ensureFileSystemLockedForDeletion(objPath string, t *testing.T) {
	err := os.RemoveAll(objPath)

	// It will throw an error read-only file system or permission denied.
	if err == nil {
		t.Errorf("Objects are deleted in read-only file system.")
	}
}

func TestDeleteDir(t *testing.T) {
	objPath := path.Join(setup.MntDir(), DirectoryNameInTestBucket)
	ensureFileSystemLockedForDeletion(objPath, t)
}

func TestDeleteFile(t *testing.T) {
	objPath := path.Join(setup.MntDir(), FileInSubDirectoryNameInTestBucket)
	ensureFileSystemLockedForDeletion(objPath, t)
}

func TestDeleteSubDirectory(t *testing.T) {
	objPath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, SubDirectoryNameInTestBucket)
	ensureFileSystemLockedForDeletion(objPath, t)
}

func TestDeleteFileInSubDirectory(t *testing.T) {
	objPath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, FileInSubDirectoryNameInTestBucket)
	ensureFileSystemLockedForDeletion(objPath, t)
}

func TestDeleteAllObjectsInBucket(t *testing.T) {
	ensureFileSystemLockedForDeletion(setup.MntDir(), t)
}
