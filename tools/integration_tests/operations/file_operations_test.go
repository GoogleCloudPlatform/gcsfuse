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

// Provides integration tests for file operations.
package operations_test

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/file_operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func TestRenameFile(t *testing.T) {
	fileName := setup.CreateTempFile()

	content, err := file_operations.ReadFile(fileName)
	if err != nil {
		t.Errorf("Read: %v", err)
	}

	newFileName := fileName + "Rename"

	err = file_operations.RenameFile(fileName, newFileName)
	if err != nil {
		t.Errorf("Error in file copying: %v", err)
	}
	// Check if the data in the file is the same after renaming.
	setup.CompareFileContents(t, newFileName, string(content))
}

func TestFileAttributes(t *testing.T) {
	preCreateTime := time.Now()
	fileName := setup.CreateTempFile()
	postCreateTime := time.Now()

	fStat, err := os.Stat(fileName)

	if err != nil {
		t.Errorf("os.Stat error: %s, %v", fileName, err)
	}
	statFileName := path.Join(setup.MntDir(), fStat.Name())
	if fileName != statFileName {
		t.Errorf("File name not matched in os.Stat, found: %s, expected: %s", statFileName, fileName)
	}
	if (preCreateTime.After(fStat.ModTime())) || (postCreateTime.Before(fStat.ModTime())) {
		t.Errorf("File modification time not in the expected time-range")
	}
	// The file size in createTempFile() is 14 bytes
	if fStat.Size() != 14 {
		t.Errorf("File size is not 14 bytes, found size: %d bytes", fStat.Size())
	}
}

func TestCopyFile(t *testing.T) {
	fileName := setup.CreateTempFile()

	content, err := file_operations.ReadFile(fileName)
	if err != nil {
		t.Errorf("Read: %v", err)
	}

	newFileName := fileName + "Copy"
	if _, err := os.Stat(newFileName); err == nil {
		t.Errorf("Copied file %s already present", newFileName)
	}

	err = file_operations.CopyFile(fileName, newFileName)
	if err != nil {
		t.Errorf("Error : %v", err)
	}

	// Check if the data in the copied file matches the original file,
	// and the data in original file is unchanged.
	setup.CompareFileContents(t, newFileName, string(content))
	setup.CompareFileContents(t, fileName, string(content))
}
