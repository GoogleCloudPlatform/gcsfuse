// Copyright 2023 Google LLC
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
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

// Copy srcFile in testBucket/testDirForReadOnlyTest/Test/b/b.txt destination.
func checkIfFileCopyFailed(srcFilePath string, t *testing.T) {

	destFileName := FileNameInSubDirectoryTestBucket + setup.GenerateRandomString(5)
	destFile := path.Join(setup.MntDir(), TestDirForReadOnlyTest, DirectoryNameInTestBucket, SubDirectoryNameInTestBucket, destFileName)

	err := operations.CopyFile(srcFilePath, destFile)
	if err == nil {
		t.Errorf("File copied in read-only file system: %v", err)
	}
}

// Copy testBucket/testDirForReadOnlyTest/Test1.txt to testBucket/testDirForReadOnlyTest/Test/b/b.txt
func TestCopyFile(t *testing.T) {
	file := path.Join(setup.MntDir(), TestDirForReadOnlyTest, FileNameInTestBucket)

	checkIfFileCopyFailed(file, t)
}

// Copy testBucket/testDirForReadOnlyTest/Test/a.txt to testBucket/testDirForReadOnlyTest/Test/b/b.txt
func TestCopyFileFromBucketDirectory(t *testing.T) {
	file := path.Join(setup.MntDir(), TestDirForReadOnlyTest, DirectoryNameInTestBucket, FileNameInDirectoryTestBucket)

	checkIfFileCopyFailed(file, t)
}

func checkIfDirCopyFailed(srcDirPath string, destDirPath string, t *testing.T) {
	// In the read-only filesystem, It is Throwing an exit status 1.
	err := operations.CopyDir(srcDirPath, destDirPath)
	if err == nil {
		t.Errorf("Directory copied in read-only file system.")
	}
}

// Copy testBucket/testDirForReadOnlyTest/Test to testBucket/testDirForReadOnlyTest/Test/b
func TestCopyDirectory(t *testing.T) {
	srcDir := path.Join(setup.MntDir(), TestDirForReadOnlyTest, DirectoryNameInTestBucket)
	destDir := path.Join(setup.MntDir(), TestDirForReadOnlyTest, DirectoryNameInTestBucket, SubDirectoryNameInTestBucket)

	checkIfDirCopyFailed(srcDir, destDir, t)
}

// Copy testBucket/testDirForReadOnlyTest/Test/b to testBucket
func TestCopySubDirectory(t *testing.T) {
	srcDir := path.Join(setup.MntDir(), TestDirForReadOnlyTest, DirectoryNameInTestBucket, SubDirectoryNameInTestBucket)
	destDir := path.Join(setup.MntDir(), TestDirForReadOnlyTest)

	checkIfDirCopyFailed(srcDir, destDir, t)
}
