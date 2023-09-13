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

// Provides integration tests for stat operation on local files.
package local_file_test

import (
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/internal/fs/inode"
	. "github.com/googlecloudplatform/gcsfuse/tools/integration_tests/local_file/helpers"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////
func assertStatCallErrorIsNil(err error, t *testing.T) {
	if err != nil {
		t.Errorf("os.Stat err: %v", err)
	}
}

func assertStatCallFileName(expected, got string, t *testing.T) {
	if got != expected {
		t.Errorf("File name mismatch in stat call. Expected: %s, Got: %s", expected, got)
	}
}

func assertStatCallFileSize(expected, got int64, t *testing.T) {
	if got != expected {
		t.Errorf("File size mismatch in stat call. Expected: %d, Got: %d", expected, got)
	}
}

func assertStatCallFileMode(got os.FileMode, t *testing.T) {
	if got != FilePerms {
		t.Errorf("File permissions mismatch in stat call. Expected: %v, Got: %v", FilePerms, got)
	}
}

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////
func TestStatOnLocalFile(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()
	// Create a local file.
	filePath, fh := CreateLocalFile(FileName1, t)

	// Stat the local file.
	fi, err := os.Stat(filePath)
	assertStatCallErrorIsNil(err, t)
	assertStatCallFileName(path.Base(filePath), fi.Name(), t)
	assertStatCallFileSize(0, fi.Size(), t)
	assertStatCallFileMode(fi.Mode(), t)

	// Writing contents to local file shouldn't create file on GCS.
	WritingToLocalFileShouldNotWriteToGCS(fh, FileName1, t)

	// Stat the local file again to check if new content is written.
	fi, err = os.Stat(filePath)
	assertStatCallErrorIsNil(err, t)
	assertStatCallFileName(path.Base(filePath), fi.Name(), t)
	assertStatCallFileSize(10, fi.Size(), t)
	assertStatCallFileMode(fi.Mode(), t)

	// Close the file and validate if the file is created on GCS.
	CloseFileAndValidateObjectContents(fh, FileName1, FileContents, t)
}

func TestStatOnLocalFileWithConflictingFileNameSuffix(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()
	// Create a local file.
	filePath, fh := CreateLocalFile(FileName1, t)
	// Stat the local file.
	fi, err := os.Stat(filePath + inode.ConflictingFileNameSuffix)
	assertStatCallErrorIsNil(err, t)
	assertStatCallFileName(path.Base(filePath)+inode.ConflictingFileNameSuffix, fi.Name(), t)
	assertStatCallFileSize(0, fi.Size(), t)
	assertStatCallFileMode(fi.Mode(), t)

	// Close the file and validate if the file is created on GCS.
	CloseFileAndValidateObjectContents(fh, FileName1, "", t)
}

func TestTruncateLocalFile(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()
	// Create a local file.
	filePath, fh := CreateLocalFile(FileName1, t)
	// Writing contents to local file .
	WritingToLocalFileShouldNotWriteToGCS(fh, FileName1, t)

	// Stat the file to validate if new contents are written.
	fi, err := os.Stat(filePath)
	assertStatCallErrorIsNil(err, t)
	assertStatCallFileName(path.Base(filePath), fi.Name(), t)
	assertStatCallFileSize(10, fi.Size(), t)
	assertStatCallFileMode(fi.Mode(), t)

	// Truncate the file to update the file size.
	err = os.Truncate(filePath, 5)
	if err != nil {
		t.Errorf("os.Truncate err: %v", err)
	}
	ValidateObjectNotFoundErr(FileName1, t)

	// Stat the file to validate if file is truncated correctly.
	fi, err = os.Stat(filePath)
	assertStatCallErrorIsNil(err, t)
	assertStatCallFileName(path.Base(filePath), fi.Name(), t)
	assertStatCallFileSize(5, fi.Size(), t)
	assertStatCallFileMode(fi.Mode(), t)

	// Close the file and validate if the file is created on GCS.
	CloseFileAndValidateObjectContents(fh, FileName1, "tests", t)
}
