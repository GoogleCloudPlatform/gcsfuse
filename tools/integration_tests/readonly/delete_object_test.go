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

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func checkIfObjDeletionFailed(objPath string, t *testing.T) {
	err := os.RemoveAll(objPath)

	if err == nil {
		t.Errorf("Objects are deleted in read-only file system.")
	}

	checkErrorForReadOnlyFileSystem(err, t)
}

func TestDeleteDir(t *testing.T) {
	objPath := path.Join(setup.MntDir(), DirectoryNameInTestBucket)

	checkIfObjDeletionFailed(objPath, t)
}

func TestDeleteFile(t *testing.T) {
	objPath := path.Join(setup.MntDir(), FileNameInTestBucket)

	checkIfObjDeletionFailed(objPath, t)
}

func TestDeleteSubDirectory(t *testing.T) {
	objPath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, SubDirectoryNameInTestBucket)

	checkIfObjDeletionFailed(objPath, t)
}

func TestDeleteFileInDirectory(t *testing.T) {
	objPath := path.Join(setup.MntDir(), DirectoryNameInTestBucket, FileNameInDirectoryTestBucket)

	checkIfObjDeletionFailed(objPath, t)
}

func TestDeleteAllObjectsInBucket(t *testing.T) {
	checkIfObjDeletionFailed(setup.MntDir(), t)
}
