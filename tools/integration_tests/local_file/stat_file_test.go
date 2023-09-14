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
	"testing"

	"github.com/googlecloudplatform/gcsfuse/internal/fs/inode"
	. "github.com/googlecloudplatform/gcsfuse/tools/integration_tests/local_file/helpers"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func TestStatOnLocalFile(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()
	// Create a local file.
	filePath, fh := CreateLocalFile(FileName1, t)

	// Stat the local file.
	VerifyStatOnLocalFile(filePath, 0, t)

	// Writing contents to local file shouldn't create file on GCS.
	WritingToLocalFileShouldNotWriteToGCS(fh, FileName1, t)

	// Stat the local file again to check if new content is written.
	VerifyStatOnLocalFile(filePath, 10, t)

	// Close the file and validate if the file is created on GCS.
	CloseFileAndValidateObjectContents(fh, FileName1, FileContents, t)
}

func TestStatOnLocalFileWithConflictingFileNameSuffix(t *testing.T) {
	// Clean the mountedDirectory before running test.
	setup.CleanMntDir()
	// Create a local file.
	filePath, fh := CreateLocalFile(FileName1, t)

	// Stat the local file.
	VerifyStatOnLocalFile(filePath+inode.ConflictingFileNameSuffix, 0, t)

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
	VerifyStatOnLocalFile(filePath, 10, t)

	// Truncate the file to update the file size.
	err := os.Truncate(filePath, 5)
	if err != nil {
		t.Fatalf("os.Truncate err: %v", err)
	}
	ValidateObjectNotFoundErrOnGCS(FileName1, t)

	// Stat the file to validate if file is truncated correctly.
	VerifyStatOnLocalFile(filePath, 5, t)

	// Close the file and validate if the file is created on GCS.
	CloseFileAndValidateObjectContents(fh, FileName1, "tests", t)
}
