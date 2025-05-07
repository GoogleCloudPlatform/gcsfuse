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

// Provides integration tests for rename file.
package operations_test

import (
	"path"
	"strings"

	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/all_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
)

func (s *OperationSuite) TestRenameFile() {
	testDir := setup.SetupTestDirectoryOnMntDir(s.mountConfiguration.MntDir(), TestDirName(s.T()))
	fileName := path.Join(testDir, tempFileName)

	operations.CreateFileWithContent(fileName, setup.FilePermission_0600, Content, s.T())

	content, err := operations.ReadFile(fileName)
	if err != nil {
		s.T().Errorf("Read: %v", err)
	}

	newFileName := fileName + "Rename"

	err = operations.RenameFile(fileName, newFileName)
	if err != nil {
		s.T().Errorf("Error in file renaming: %v", err)
	}
	// Check if the data in the file is the same after renaming.
	setup.CompareFileContents(s.T(), newFileName, string(content))
}

func (s *OperationSuite) TestRenameFileWithSrcFileDoesNoExist() {
	// Set up the test directory.
	testDir := setup.SetupTestDirectoryOnMntDir(s.mountConfiguration.MntDir(), TestDirName(s.T()))
	// Define source and destination file names.
	srcFilePath := path.Join(testDir, "move1.txt") // This file does not exist.
	destFilePath := path.Join(testDir, "move2.txt")

	// Attempt to rename the non-existent file.
	err := operations.RenameFile(srcFilePath, destFilePath)

	// Assert that an error occurred.
	assert.Error(s.T(), err)
	assert.True(s.T(), strings.Contains(err.Error(), "no such file or directory"))
}
