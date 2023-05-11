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

// Copy srcFile in testBucket/Test/b/b.txt destination.
func checkIfFileCopyFailed(srcFilePath string, t *testing.T) {
	// cp without destination file creates a destination file and create workflow is already covered separately.
	copyFile := path.Join(setup.MntDir(), DirectoryNameInTestBucket, SubDirectoryNameInTestBucket, FileNameInSubDirectoryTestBucket)

	// cp without destination file creates a destination file and create workflow is already covered separately.
	// Checking if destination object exist.
	if _, err := os.Stat(copyFile); err != nil {
		t.Errorf("Copied file %s is not present", copyFile)
	}

	err := operations.CopyFile(srcFilePath, copyFile)
	if err == nil {
		t.Errorf("File copied in read-only file system: %v", err)
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
	// In the read-only filesystem, It is Throwing an exit status 1.
	err := operations.CopyDir(srcDirPath, destDirPath)
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
