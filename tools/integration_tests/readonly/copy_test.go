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
	"io"
	"os"
	"path"
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/setup"
)

func ensureFileSystemLockedForFileCopy(srcFilePath string, t *testing.T) {
	file, err := os.OpenFile(srcFilePath, syscall.O_DIRECT, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("Error in the opening file: %v", err)
	}

	content := make([]byte, setup.BufferSize)
	_, err = file.Read(content)
	if err != nil {
		t.Errorf("Read: %v", err)
	}

	copyFile := path.Join(setup.MntDir(), "Test", "b", "b.txt")
	if _, err := os.Stat(copyFile); err != nil {
		t.Errorf("Copied file %s is not present", copyFile)
	}

	// File copying with io.Copy() utility.
	source, err := os.OpenFile(srcFilePath, syscall.O_DIRECT, setup.FilePermission_0600)
	if err != nil {
		t.Errorf("File %s opening error: %v", "Test1.txt", err)
	}
	defer source.Close()

	destination, err := os.OpenFile(copyFile, os.O_WRONLY|os.O_CREATE|syscall.O_DIRECT, setup.FilePermission_0600)
	if err == nil {
		t.Errorf("File %s opening error: %v", "b.txt", err)
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	if err == nil {
		t.Errorf("File copied in read-only system.")
	}
}

func TestCopyFileInExistingFile(t *testing.T) {
	srcFile := path.Join(setup.MntDir(), "Test1.txt")
	ensureFileSystemLockedForFileCopy(srcFile, t)
}

func TestCopySubDirectoryFileInExistingFile(t *testing.T) {
	srcFile := path.Join(setup.MntDir(), "Test", "a.txt")
	ensureFileSystemLockedForFileCopy(srcFile, t)
}
