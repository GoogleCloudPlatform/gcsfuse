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
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
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

func TestRenameFileWithSrcFileDoesNoExist(t *testing.T) {
	// Set up the test directory.
	testDir := setup.SetupTestDirectory(DirForOperationTests)
	// Define source and destination file names.
	srcFilePath := path.Join(testDir, "move1.txt") // This file does not exist.
	destFilePath := path.Join(testDir, "move2.txt")

	// Attempt to rename the non-existent file.
	err := operations.RenameFile(srcFilePath, destFilePath)

	// Assert that an error occurred.
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "no such file or directory"))
}

func TestRenameSymlinkToFile(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForOperationTests)
	targetName := "target.txt"
	targetPath := path.Join(testDir, targetName)
	err := os.WriteFile(targetPath, []byte("taco"), setup.FilePermission_0600)
	require.NoError(t, err)
	oldSymlinkPath := path.Join(testDir, "symlink_old")
	err = os.Symlink(targetPath, oldSymlinkPath)
	require.NoError(t, err)
	newSymlinkPath := path.Join(testDir, "symlink_new")

	err = os.Rename(oldSymlinkPath, newSymlinkPath)

	require.NoError(t, err)
	_, err = os.Lstat(oldSymlinkPath)
	require.Error(t, err)
	assert.True(t, os.IsNotExist(err))
	fi, err := os.Lstat(newSymlinkPath)
	require.NoError(t, err)
	assert.Equal(t, os.ModeSymlink, fi.Mode()&os.ModeType)
	targetRead, err := os.Readlink(newSymlinkPath)
	require.NoError(t, err)
	assert.Equal(t, targetPath, targetRead)
	content, err := operations.ReadFile(newSymlinkPath)
	require.NoError(t, err)
	assert.Equal(t, "taco", string(content))
}
