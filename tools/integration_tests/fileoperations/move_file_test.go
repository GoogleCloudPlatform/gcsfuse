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

// Provides integration tests for move a file within same directory and from current directory to different directory.
package fileoperations_test

import (
	"os"
	"os/exec"
	"path"
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/setup"
)

// Move file from src directory to destination.
func checkIfFileMoveSucceeded(srcDirPath string, destDirPath string, t *testing.T) {
	// io packages do not have a method to copy the directory.
	cmd := exec.Command("mv", srcDirPath, destDirPath)

	err := cmd.Run()
	if err != nil {
		t.Errorf("Moving file operation is failed.")
	}
}

// Create below directory and file.
// Test               -- Directory
// Test/move.txt      -- File
func createFileAndDirectory(dirPath string, filePath string, t *testing.T) {
	err := os.Mkdir(dirPath, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("Mkdir at %q: %v", setup.MntDir(), err)
		return
	}

	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC|syscall.O_DIRECT, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("Error in the opening the file %v", err)
	}
	defer file.Close()

	_, err = file.WriteString(moveFileContent)
	if err != nil {
		t.Errorf("Temporary file at %v", err)
	}
}

// Move file from Test/move.txt to Test/a/move.txt
func TestMoveFileWithinSameDirectory(t *testing.T) {
	dirPath := path.Join(setup.MntDir(), "Test")
	filePath := path.Join(dirPath, moveFile)

	createFileAndDirectory(dirPath, filePath, t)

	subDirPath := path.Join(dirPath, "a")

	// Create directory at testBucket/Test/a
	err := os.Mkdir(subDirPath, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("Mkdir at %q: %v", dirPath, err)
		return
	}

	movePath := path.Join(subDirPath, moveFile)

	// Move file from Test/move.txt to Test/a/move.txt.
	checkIfFileMoveSucceeded(filePath, movePath, t)

	content, err := os.ReadFile(movePath)
	if err != nil {
		t.Errorf("ReadAll: %v", err)
	}

	if got, want := string(content), moveFileContent; got != want {
		t.Errorf("File content %q not match %q", got, want)
	}

	os.RemoveAll(dirPath)
}

// Move file from Test/move.txt to Test1/move.txt
func TestMoveFileWithinDifferentDirectory(t *testing.T) {
	dirPath := path.Join(setup.MntDir(), "Test")
	filePath := path.Join(dirPath, moveFile)

	createFileAndDirectory(dirPath, filePath, t)

	dirPath2 := path.Join(setup.MntDir(), "Test2")

	// Create directory at testBucket/Test2
	err := os.Mkdir(dirPath2, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("Mkdir at %q: %v", dirPath2, err)
		return
	}

	movePath := path.Join(dirPath2, "move.txt")

	checkIfFileMoveSucceeded(filePath, movePath, t)

	content, err := os.ReadFile(movePath)
	if err != nil {
		t.Errorf("ReadAll: %v", err)
	}

	if got, want := string(content), moveFileContent; got != want {
		t.Errorf("File content %q not match %q", got, want)
	}

	os.RemoveAll(dirPath)
}
