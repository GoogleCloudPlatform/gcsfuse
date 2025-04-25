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

	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/all_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

func (s *OperationSuite) TestDeleteEmptyExplicitDir() {
	testDir := setup.SetupTestDirectoryOnMntDir(s.mountConfiguration.MntDir(), TestDirName(s.T()))

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

func (s *OperationSuite) TestDeleteNonEmptyExplicitDir() {
	testDir := setup.SetupTestDirectoryOnMntDir(s.mountConfiguration.MntDir(), TestDirName(s.T()))

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
