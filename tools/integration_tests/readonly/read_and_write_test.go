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
	"fmt"
	"os"
	"path"
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/setup"
)

func readFile(filePath string, t *testing.T) (content []byte) {
	file, err := os.OpenFile(filePath, os.O_RDONLY|syscall.O_DIRECT, setup.FilePermission_0600)
	if err != nil {
		setup.LogAndExit(fmt.Sprintf("Error in the opening the file %v", err))
	}
	defer file.Close()

	content, err = os.ReadFile(file.Name())
	if err != nil {
		t.Errorf("ReadAll: %v", err)
	}
	return content
}

func TestReadFile(t *testing.T) {
	filePath := path.Join(setup.MntDir(), "Test1.txt")
	content := readFile(filePath, t)
	if got, want := string(content), "This is from file Test1\n"; got != want {
		t.Errorf("File content %q not match %q", got, want)
	}
}

func TestReadFileFromSubDirectory(t *testing.T) {
	filePath := path.Join(setup.MntDir(), "Test", "a.txt")
	content := readFile(filePath, t)
	if got, want := string(content), "This is from directory Test file a\n"; got != want {
		t.Errorf("File content %q not match %q", got, want)
	}
}

func openFileToWrite(filePath string, t *testing.T) {
	file, err := os.OpenFile(filePath, os.O_RDWR|syscall.O_DIRECT, setup.FilePermission_0600)
	if err == nil {
		t.Errorf("File opened for writing in read-only mount.")
	}
	defer file.Close()
}

func TestOpenFileToWrite(t *testing.T) {
	filePath := path.Join(setup.MntDir(), "Test1.txt")
	openFileToWrite(filePath, t)
}

func TestOpenFileToWriteFromSubDirectory(t *testing.T) {
	filePath := path.Join(setup.MntDir(), "Test", "a.txt")
	openFileToWrite(filePath, t)
}

func openFileToAppendData(filePath string, t *testing.T) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|syscall.O_DIRECT, setup.FilePermission_0600)
	if err == nil {
		t.Errorf("File opened for appending data in read-only mount.")
	}
	defer file.Close()
}

func TestOpenFileForAppendData(t *testing.T) {
	filePath := path.Join(setup.MntDir(), "Test1.txt")
	openFileToAppendData(filePath, t)
}

func TestOpenFileForAppendDataFromSubDirectory(t *testing.T) {
	filePath := path.Join(setup.MntDir(), "Test", "a.txt")
	openFileToAppendData(filePath, t)
}
