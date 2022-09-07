// Copyright 2021 Google Inc. All Rights Reserved.
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

// Provides integration tests for file operations with implicit_dir flag set.
package implicitdir_test

import (
	"os"
	"path"
	"testing"
	"time"
)

func TestRenameFile(t *testing.T) {
	fileName := createTempFile()
	err := clearKernelCache()
	if err != nil {
		t.Errorf("Clear Kernel Cache: %v", err)
	}
	content, err := os.ReadFile(fileName)
	if err != nil {
		t.Errorf("Read: %v", err)
	}
	newFileName := fileName + "Rename"
	if _, err := os.Stat(newFileName); err == nil {
		t.Errorf("Renamed file %s already present", newFileName)
	}

	if err := os.Rename(fileName, newFileName); err != nil {
		t.Errorf("Rename unsuccessful: %v", err)
	}

	if _, err := os.Stat(fileName); err == nil {
		t.Errorf("File %s still exists", fileName)
	}
	if _, err := os.Stat(newFileName); err != nil {
		t.Errorf("Renamed file %s not found", newFileName)
	}
	// Check if the data in the file is the same after renaming.
	compareFileContents(t, newFileName, string(content))
}

func TestFileAttributes(t *testing.T) {
	preCreateTime := time.Now()
	fileName := createTempFile()
	postCreateTime := time.Now()

	fStat, err := os.Stat(fileName)

	if err != nil {
		t.Errorf("os.Stat error: %s, %v", fileName, err)
	}
	statFileName := path.Join(tmpDir, fStat.Name())
	if fileName != statFileName {
		t.Errorf("File name not matched in os.Stat, found: %s, expected: %s", statFileName, fileName)
	}
	if (preCreateTime.After(fStat.ModTime())) || (postCreateTime.Before(fStat.ModTime())) {
		t.Errorf("File modification time not in the required time-range")
	}
	// The file size in createTempFile() is 14 bytes
	if fStat.Size() != 14 {
		t.Errorf("File size is not 14 bytes, found size: %d bytes", fStat.Size())
	}
}
