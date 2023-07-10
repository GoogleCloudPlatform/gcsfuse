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

// Provides integration tests for delete files.
package operations

import (
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

const DirNameInTestBucket = "A"               // testBucket/A
const FileNameInTestBucket = "A.txt"          // testBucket/A.txt
const FileNameInDirectoryTestBucket = "a.txt" // testBucket/A/a.txt

func checkIfFileDeletionSucceeded(filePath string, t *testing.T) {
	err := os.Remove(filePath)

	if err != nil {
		t.Errorf("File deletion failed: %v", err)
	}

	file, err := os.Stat(filePath)
	if err == nil && file.IsDir() == false {
		t.Errorf("File is not deleted.")
	}
}

func createFile(filePath string, t *testing.T) {
	file, err := os.Create(filePath)
	if err != nil {
		t.Errorf("Error in creating file: %v", err)
	}

	// Closing file at the end
	operations.CloseFile(file)
}

// Remove testBucket/A.txt
func TestDeleteFileFromBucket(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	filePath := path.Join(setup.MntDir(), FileNameInTestBucket)

	createFile(filePath, t)

	checkIfFileDeletionSucceeded(filePath, t)
}

// Remove testBucket/A/a.txt
func TestDeleteFileFromBucketDirectory(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()

	dirPath := path.Join(setup.MntDir(), DirNameInTestBucket)
	err := os.Mkdir(dirPath, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("Error in creating directory: %v", err)
	}

	filePath := path.Join(dirPath, FileNameInDirectoryTestBucket)
	createFile(filePath, t)

	checkIfFileDeletionSucceeded(filePath, t)
}
