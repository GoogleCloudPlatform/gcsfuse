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
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

const Content = "Testing"

func checkIfFileFailedToOpenForWrite(filePath string, t *testing.T) {
	err := operations.WriteFile(filePath, Content)

	if err == nil {
		t.Errorf("File opened for writing in read-only mount.")
	}

	checkErrorForReadOnlyFileSystem(err, t)
}

// testBucket/Test1.txt
func TestOpenFileWithReadWriteAccess(t *testing.T) {
	filePath := path.Join(setup.MntDir(), FileNameInTestBucket)

	checkIfFileFailedToOpenForWrite(filePath, t)
}

// testBucket/Test/a.txt
func TestOpenFileFromBucketDirectoryWithReadWriteAccess(t *testing.T) {
	filePath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, FileNameInDirectoryTestBucket)

	checkIfFileFailedToOpenForWrite(filePath, t)
}

// testBucket/Test/b/b.txt
func TestOpenFileFromBucketSubDirectoryWithReadWriteAccess(t *testing.T) {
	filePath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, SubDirectoryNameInTestBucket, FileNameInSubDirectoryTestBucket)

	checkIfFileFailedToOpenForWrite(filePath, t)
}

func checkIfNonExistentFileFailedToOpenForWrite(filePath string, t *testing.T) {
	err := operations.WriteFile(filePath, Content)

	if err == nil {
		t.Errorf("NonExist file opened for writing in read-only mount.")
	}

	checkErrorForObjectNotExist(err, t)
}

func TestOpenNonExistentFileWithReadWriteAccess(t *testing.T) {
	filePath := path.Join(setup.MntDir(), FileNotExist)

	checkIfNonExistentFileFailedToOpenForWrite(filePath, t)
}

func TestOpenNonExistentFileFromBucketDirectoryWithReadWriteAccess(t *testing.T) {
	filePath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, FileNotExist)

	checkIfNonExistentFileFailedToOpenForWrite(filePath, t)
}

func TestOpenNonExistentFileFromBucketSubDirectoryWithReadWriteAccess(t *testing.T) {
	filePath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, SubDirectoryNameInTestBucket, FileNotExist)

	checkIfNonExistentFileFailedToOpenForWrite(filePath, t)
}

func checkIfFileFailedToOpenForAppend(filePath string, t *testing.T) {
	err := operations.WriteFileInAppendMode(filePath, Content)

	if err == nil {
		t.Errorf("File opened for appending content in read-only mount.")
	}

	checkErrorForReadOnlyFileSystem(err, t)
}

func TestOpenFileWithAppendAccess(t *testing.T) {
	filePath := path.Join(setup.MntDir(), FileNameInTestBucket)

	checkIfFileFailedToOpenForAppend(filePath, t)
}

func TestOpenFileFromBucketDirectoryWithAppendAccess(t *testing.T) {
	filePath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, FileNameInDirectoryTestBucket)

	checkIfFileFailedToOpenForAppend(filePath, t)
}

func TestOpenFileFromBucketSubDirectoryWithAppendAccess(t *testing.T) {
	filePath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, SubDirectoryNameInTestBucket, FileNameInSubDirectoryTestBucket)

	checkIfFileFailedToOpenForAppend(filePath, t)
}

func checkIfNonExistentFileFailedToOpenForAppend(filePath string, t *testing.T) {
	err := operations.WriteFileInAppendMode(filePath, Content)

	if err == nil {
		t.Errorf("File opened for appending content in read-only mount.")
	}

	checkErrorForObjectNotExist(err, t)
}

func TestOpenNonExistentFileWithAppendAccess(t *testing.T) {
	filePath := path.Join(setup.MntDir(), FileNotExist)

	checkIfNonExistentFileFailedToOpenForAppend(filePath, t)
}

func TestOpenNonExistentFileFromBucketDirectoryWithAppendAccess(t *testing.T) {
	filePath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, FileNotExist)

	checkIfNonExistentFileFailedToOpenForAppend(filePath, t)
}

func TestOpenNonExistentFileFromBucketSubDirectoryWithAppendAccess(t *testing.T) {
	filePath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, SubDirectoryNameInTestBucket, FileNotExist)

	checkIfNonExistentFileFailedToOpenForAppend(filePath, t)
}
