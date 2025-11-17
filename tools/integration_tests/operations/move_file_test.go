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

// Provides integration tests for move file.
package operations_test

import (
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
)

// Create below directory and file.
// Test               -- Directory
// Test/move.txt      -- File
func createSrcDirectoryAndFile(dirPath string, filePath string, t *testing.T) {
	err := os.Mkdir(dirPath, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("Mkdir at %q: %v", dirPath, err)
		return
	}

	file, err := os.Create(filePath)
	if err != nil {
		t.Errorf("Error in creating file %v:", err)
	}

	// Closing file at the end.
	defer operations.CloseFileShouldNotThrowError(t, file)

	err = operations.WriteFile(file.Name(), MoveFileContent)
	if err != nil {
		t.Errorf("File at %v", err)
	}
}

func checkIfFileMoveOperationSucceeded(srcFilePath string, destDirPath string, t *testing.T) {
	// Move file from Test/move.txt to destination.
	err := operations.Move(srcFilePath, destDirPath)
	if err != nil {
		t.Errorf("Error in moving file: %v", err)
	}

	// Check if the file content matches.
	moveFilePath := path.Join(destDirPath, MoveFile)
	content, err := operations.ReadFile(moveFilePath)
	if err != nil {
		t.Errorf("ReadAll: %v", err)
	}

	if got, want := string(content), MoveFileContent; got != want {
		t.Errorf("File content %q not match %q", got, want)
	}
}

// Move file from Test/move.txt to Test/a/move.txt
func TestMoveFileWithinSameDirectory(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForOperationTests)
	dirPath := path.Join(testDir, "Test")
	filePath := path.Join(dirPath, MoveFile)

	createSrcDirectoryAndFile(dirPath, filePath, t)

	destDirPath := path.Join(dirPath, "a")
	err := os.Mkdir(destDirPath, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("Mkdir at %q: %v", destDirPath, err)
	}

	checkIfFileMoveOperationSucceeded(filePath, destDirPath, t)
}

// Move file from Test/move.txt to Test1/move.txt
func TestMoveFileWithinDifferentDirectory(t *testing.T) {
	testDir := setup.SetupTestDirectory(DirForOperationTests)
	dirPath := path.Join(testDir, "Test")
	filePath := path.Join(dirPath, MoveFile)

	createSrcDirectoryAndFile(dirPath, filePath, t)

	destDirPath := path.Join(testDir, "Test2")
	err := os.Mkdir(destDirPath, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("Mkdir at %q: %v", destDirPath, err)
	}

	checkIfFileMoveOperationSucceeded(filePath, destDirPath, t)
}

// Rename file from Test/move1.txt to Test/move2.txt
func TestMoveFileWithDestFileExist(t *testing.T) {
	// Set up the test directory.
	testDir := setup.SetupTestDirectory(DirForOperationTests)
	// Define source and destination file names.
	srcFilePath := path.Join(testDir, "move1.txt")
	destFilePath := path.Join(testDir, "move2.txt")
	// Create the source and dest file with some content.
	operations.CreateFileWithContent(srcFilePath, setup.FilePermission_0600, Content, t)
	operations.CreateFileWithContent(destFilePath, setup.FilePermission_0600, "Hello from dest file", t)

	// Move the file.
	err := operations.Move(srcFilePath, destFilePath)

	assert.NoError(t, err, "error in file moving")
	// Verify the file was renamed and content is preserved.
	setup.CompareFileContents(t, destFilePath, Content)
	// Verify the old file is removed.
	_, err = os.Stat(srcFilePath)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "no such file or directory"))
}
