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
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func checkIfFileReadSucceeded(filePath string, expectedContent string, t *testing.T) {
	content, err := operations.ReadFile(filePath)
	if err != nil {
		t.Errorf("ReadAll: %v", err)
	}
	if got, want := string(content), expectedContent; got != want {
		t.Errorf("File content %q not match %q", got, want)
	}
}

// testBucket/Test1.txt
func TestReadFile(t *testing.T) {
	filePath := path.Join(setup.MntDir(), FileNameInTestBucket)

	checkIfFileReadSucceeded(filePath, ContentInFileInTestBucket, t)
}

// testBucket/Test/a.txt
func TestReadFileFromBucketDirectory(t *testing.T) {
	filePath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, FileNameInDirectoryTestBucket)

	checkIfFileReadSucceeded(filePath, ContentInFileInDirectoryTestBucket, t)
}

// testBucket/Test/b/b.txt
func TestReadFileFromBucketSubDirectory(t *testing.T) {
	filePath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, SubDirectoryNameInTestBucket, FileNameInSubDirectoryTestBucket)

	checkIfFileReadSucceeded(filePath, ContentInFileInSubDirectoryTestBucket, t)
}

func checkIfNonExistentFileFailedToOpen(filePath string, t *testing.T) {
	file, err := os.OpenFile(filePath, os.O_RDONLY|syscall.O_DIRECT, setup.FilePermission_0600)

	checkErrorForObjectNotExist(err, t)

	if err == nil {
		t.Errorf("Nonexistent file opened to read.")
	}
	defer file.Close()
}

func TestReadNonExistentFile(t *testing.T) {
	filePath := path.Join(setup.MntDir(), FileNotExist)

	checkIfNonExistentFileFailedToOpen(filePath, t)
}

func TestReadNonExistentFileFromBucketDirectory(t *testing.T) {
	filePath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, FileNotExist)

	checkIfNonExistentFileFailedToOpen(filePath, t)
}

func TestReadNonExistentFileFromBucketSubDirectory(t *testing.T) {
	filePath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, SubDirectoryNameInTestBucket, FileNotExist)

	checkIfNonExistentFileFailedToOpen(filePath, t)
}
