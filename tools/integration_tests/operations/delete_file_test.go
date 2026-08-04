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

// Provides integration tests for delete files.
package operations_test

import (
	"os"
	"path"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

const DirNameInTestBucket = "A"               // testBucket/A
const FileNameInTestBucket = "A.txt"          // testBucket/A.txt
const FileNameInDirectoryTestBucket = "a.txt" // testBucket/A/a.txt

func (s *operationsTestSuite) checkIfFileDeletionSucceeded(filePath string) {
	err := os.Remove(filePath)

	if err != nil {
		s.T().Errorf("File deletion failed: %v", err)
	}

	file, err := os.Stat(filePath)
	if err == nil && file.IsDir() == false {
		s.T().Errorf("File is not deleted.")
	}
}

func (s *operationsTestSuite) createFile(filePath string) {
	file, err := os.Create(filePath)
	if err != nil {
		s.T().Errorf("Error in creating file: %v", err)
	}

	// Closing file at the end
	operations.CloseFileShouldNotThrowError(s.T(), file)
}

// Remove testBucket/A.txt
func (s *operationsTestSuite) TestDeleteFileFromBucket() {
	testDir := setup.SetupTestDirectory(DirForOperationTests)

	filePath := path.Join(testDir, FileNameInTestBucket)

	s.createFile(filePath)

	s.checkIfFileDeletionSucceeded(filePath)
}

// Remove testBucket/A/a.txt
func (s *operationsTestSuite) TestDeleteFileFromBucketDirectory() {
	testDir := setup.SetupTestDirectory(DirForOperationTests)

	dirPath := path.Join(testDir, DirNameInTestBucket)
	err := os.Mkdir(dirPath, setup.FilePermission_0600)
	if err != nil {
		s.T().Errorf("Error in creating directory: %v", err)
	}

	filePath := path.Join(dirPath, FileNameInDirectoryTestBucket)
	s.createFile(filePath)

	s.checkIfFileDeletionSucceeded(filePath)
}
