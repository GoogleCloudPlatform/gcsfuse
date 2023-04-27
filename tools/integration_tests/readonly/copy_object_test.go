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
	"io"
	"os"
	"os/exec"
	"path"
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

	// Checking if destination object exist.
	copyFile := path.Join(setup.MntDir(), DirectoryNameInTestBucket, SubDirectoryNameInTestBucket, FileNameInSubDirectoryTestBucket)
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
func TestCopyFileFromBucketDirectory(t *testing.T) {
	file := path.Join(setup.MntDir(), DirectoryNameInTestBucket, FileNameInDirectoryTestBucket)

	checkIfFileCopyFailed(file, t)
}

func checkIfDirCopyFailed(srcDirPath string, destDirPath string, t *testing.T) {
	cmd := exec.Command("cp", "--recursive", srcDirPath, destDirPath)
	err := cmd.Run()

	// Throwing an  exit status 1
	if err == nil {
		t.Errorf("Directory copied in read-only file system.")
	}
}

// Copy testBucket/Test to testBucket/Test/b
func TestCopyDirectory(t *testing.T) {
	srcDir := path.Join(setup.MntDir(), DirectoryNameInTestBucket)
	destDir := path.Join(setup.MntDir(), DirectoryNameInTestBucket, SubDirectoryNameInTestBucket)

	checkIfDirCopyFailed(srcDir, destDir, t)
}

// Copy testBucket/Test/b to testBucket
func TestCopySubDirectory(t *testing.T) {
	srcDir := path.Join(setup.MntDir(), DirectoryNameInTestBucket, SubDirectoryNameInTestBucket)
	destDir := path.Join(setup.MntDir())

	checkIfDirCopyFailed(srcDir, destDir, t)
}
