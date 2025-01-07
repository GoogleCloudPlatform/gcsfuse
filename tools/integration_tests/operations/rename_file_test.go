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
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
)

func TestRenameFile(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForOperationTests)
	fileName := path.Join(testDir, tempFileName)

	operations.CreateFileWithContent(fileName, setup.FilePermission_0600, Content, t)

	content, err := operations.ReadFile(fileName)
	if err != nil {
		t.Errorf("Read: %v", err)
	}

	newFileName := fileName + "Rename"

	err = operations.RenameFile(fileName, newFileName)
	if err != nil {
		t.Errorf("Error in file renaming: %v", err)
	}
	// Check if the data in the file is the same after renaming.
	setup.CompareFileContents(t, newFileName, string(content))
}

// Rename file from Test/move1.txt to Test/move2.txt
func TestRenameFileWithDstDestFileExist(t *testing.T) {
	// Set up the test directory.
	testDir := setup.SetupTestDirectory(DirForOperationTests)
	// Define source and destination file names.
	fileName := path.Join(testDir, "move1.txt")
	destFileName := path.Join(testDir, "move2.txt")
	// Create the source file with content.
	operations.CreateFileWithContent(fileName, setup.FilePermission_0600, Content, t)

	// Rename the file.
	err := operations.RenameFile(fileName, destFileName)

	assert.NoError(t, err, "error in file renaming")
	// Verify the file was renamed and content is preserved.
	setup.CompareFileContents(t, destFileName, Content)
}

func TestRenameFileWithSrcFileDoesNoExist(t *testing.T) {
	// Set up the test directory.
	testDir := setup.SetupTestDirectory(DirForOperationTests)
	// Define source and destination file names.
	fileName := path.Join(testDir, "move1.txt") // This file does not exist.
	destFileName := path.Join(testDir, "move2.txt")

	// Attempt to rename the non-existent file.
	err := operations.RenameFile(fileName, destFileName)

	// Assert that an error occurred.
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "no such file or directory"))
}
