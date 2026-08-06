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

// Provides integration tests for delete directory.
package operations_test

import (
	"os"
	"path"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

func (s *operationsTestSuite) TestDeleteEmptyExplicitDir() {
	testDir := setup.SetupTestDirectory(DirForOperationTests)

	dirPath := path.Join(testDir, EmptyExplicitDirectoryForDeleteTest)
	operations.CreateDirectoryWithNFiles(0, dirPath, "", s.T())

	err := os.RemoveAll(dirPath)
	if err != nil {
		s.T().Errorf("Error in deleting empty explicit directory.")
	}

	dir, err := os.Stat(dirPath)
	if err == nil && dir.Name() == EmptyExplicitDirectoryForDeleteTest && dir.IsDir() {
		s.T().Errorf("Directory is not deleted.")
	}
}

func (s *operationsTestSuite) TestDeleteNonEmptyExplicitDir() {
	testDir := setup.SetupTestDirectory(DirForOperationTests)

	dirPath := path.Join(testDir, NonEmptyExplicitDirectoryForDeleteTest)
	operations.CreateDirectoryWithNFiles(NumberOfFilesInNonEmptyExplicitDirectoryForDeleteTest, dirPath, PrefixFilesInNonEmptyExplicitDirectoryForDeleteTest, s.T())

	subDirPath := path.Join(dirPath, NonEmptyExplicitSubDirectoryForDeleteTest)
	operations.CreateDirectoryWithNFiles(NumberOfFilesInNonEmptyExplicitSubDirectoryForDeleteTest, subDirPath, PrefixFilesInNonEmptyExplicitSubDirectoryForDeleteTest, s.T())

	err := os.RemoveAll(dirPath)
	if err != nil {
		s.T().Errorf("Error in deleting empty explicit directory.")
	}

	dir, err := os.Stat(dirPath)
	if err == nil && dir.Name() == NonEmptyExplicitDirectoryForDeleteTest && dir.IsDir() {
		s.T().Errorf("Directory is not deleted.")
	}
}

func (s *operationsTestSuite) TestRmDirAlreadyDeletedExplicitDirReturnsENOENT() {
	testDir := setup.SetupTestDirectory(DirForOperationTests)
	dirPath := path.Join(testDir, "explicit_dir_double_delete")
	operations.CreateDirectoryWithNFiles(0, dirPath, "", s.T())
	// Delete explicit directory first time.
	err := os.Remove(dirPath)
	if err != nil {
		s.T().Fatalf("Error in deleting empty explicit directory: %v", err)
	}

	// Delete it again via RmDir, which must return os.ErrNotExist (ENOENT).
	err = os.Remove(dirPath)

	if err == nil {
		s.T().Errorf("Expected ENOENT error when deleting non-existent explicit directory, got nil")
	}
	if !os.IsNotExist(err) {
		s.T().Errorf("Expected os.ErrNotExist (ENOENT) error when deleting non-existent explicit directory, got: %v", err)
	}
}

func (s *operationsTestSuite) TestRmDirNonExistentDirReturnsENOENT() {
	testDir := setup.SetupTestDirectory(DirForOperationTests)
	dirPath := path.Join(testDir, "non_existent_dir_rmdir_enoent")

	err := os.Remove(dirPath)

	if err == nil {
		s.T().Errorf("Expected ENOENT error when calling RmDir on non-existent path, got nil")
	}
	if !os.IsNotExist(err) {
		s.T().Errorf("Expected os.ErrNotExist (ENOENT) error when calling RmDir on non-existent path, got: %v", err)
	}
}

func (s *operationsTestSuite) TestRmDirOutOfBandDeletedExplicitDirReturnsENOENT() {
	testDir := setup.SetupTestDirectory(DirForOperationTests)
	dirName := "oob_deleted_explicit_dir"
	dirPath := path.Join(testDir, dirName)
	operations.CreateDirectoryWithNFiles(0, dirPath, "", s.T())
	if _, err := os.Stat(dirPath); err != nil {
		s.T().Fatalf("Error statting explicit directory: %v", err)
	}
	client.DeleteDirOnGCS(ctx, storageClient, path.Join(DirForOperationTests, dirName))

	err := os.Remove(dirPath)

	if err == nil {
		s.T().Errorf("Expected ENOENT error when deleting out-of-band deleted explicit directory, got nil")
	}
	if !os.IsNotExist(err) {
		s.T().Errorf("Expected os.ErrNotExist (ENOENT) error, got: %v", err)
	}
}
