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

// Provides integration tests for operation on unlinked local files.
package local_file_test

import (
	"path"
	"testing"

	. "github.com/googlecloudplatform/gcsfuse/tools/integration_tests/local_file/helpers"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func TestStatOnUnlinkedLocalFile(t *testing.T) {
	setup.SetupTestDirectory(testDirPath)
	// Create a local file.
	filePath, fh := CreateLocalFileInTestDir(testDirPath, FileName1, t)
	// unlink the local file.
	UnlinkShouldNotThrowError(filePath, t)

	// Stat the local file and validate error.
	ValidateNoFileOrDirError(path.Join(testDirPath, FileName1), t)

	// Close the file and validate that file is not created on GCS.
	CloseFile(fh, FileName1, t)
	ValidateObjectNotFoundErrOnGCS(FileName1, t)
}

func TestReadDirContainingUnlinkedLocalFiles(t *testing.T) {
	setup.SetupTestDirectory(testDirPath)
	// Create local files.
	_, fh1 := CreateLocalFileInTestDir(testDirPath, FileName1, t)
	_, fh2 := CreateLocalFileInTestDir(testDirPath, FileName2, t)
	filepath3, fh3 := CreateLocalFileInTestDir(testDirPath, FileName3, t)
	// Unlink local file 3.
	UnlinkShouldNotThrowError(filepath3, t)

	// Attempt to list mntDir.
	entries := ReadDirectory(testDirPath, t)

	// Verify unlinked entries are not listed.
	VerifyCountOfEntries(2, len(entries), t)
	VerifyLocalFileEntry(entries[0], FileName1, 0, t)
	VerifyLocalFileEntry(entries[1], FileName2, 0, t)
	// Close the local files.
	CloseFileAndValidateObjectContents(fh1, FileName1, "", t)
	CloseFileAndValidateObjectContents(fh2, FileName2, "", t)
	// Verify unlinked file is not written to GCS.
	CloseFile(fh3, FileName3, t)
	ValidateObjectNotFoundErrOnGCS(FileName3, t)
}
func TestUnlinkOfLocalFile(t *testing.T) {
	setup.SetupTestDirectory(testDirPath)
	// Create empty local file.
	filePath, fh := CreateLocalFileInTestDir(testDirPath, FileName1, t)

	// Attempt to unlink local file.
	UnlinkShouldNotThrowError(filePath, t)

	// Verify unlink operation succeeds.
	ValidateNoFileOrDirError(path.Join(testDirPath, FileName1), t)
	CloseFile(fh, FileName1, t)
	// Validate file it is not present on GCS.
	ValidateObjectNotFoundErrOnGCS(FileName1, t)
}

func TestWriteOnUnlinkedLocalFileSucceeds(t *testing.T) {
	setup.SetupTestDirectory(testDirPath)
	// Create local file.
	filepath, fh := CreateLocalFileInTestDir(testDirPath, FileName1, t)
	// Verify unlink operation succeeds.
	UnlinkShouldNotThrowError(filepath, t)
	ValidateNoFileOrDirError(path.Join(testDirPath, FileName1), t)

	// Write to unlinked local file.
	WritingToFileSHouldNotThrowError(fh, FileContents, t)

	// Validate flush file does not throw error.
	CloseFile(fh, FileName1, t)
	// Validate unlinked file is not written to GCS.
	ValidateObjectNotFoundErrOnGCS(FileName1, t)
}

func TestSyncOnUnlinkedLocalFile(t *testing.T) {
	setup.SetupTestDirectory(testDirPath)
	// Create local file.
	filepath, fh := CreateLocalFileInTestDir(testDirPath, FileName1, t)

	// Attempt to unlink local file.
	UnlinkShouldNotThrowError(filepath, t)

	// Verify unlink operation succeeds.
	ValidateNoFileOrDirError(path.Join(testDirPath, FileName1), t)
	// Validate sync operation does not write to GCS after unlink.
	SyncOnLocalFileShouldNotThrowError(fh, FileName1, t)
	ValidateObjectNotFoundErrOnGCS(FileName1, t)
	// Close the local file and validate it is not present on GCS.
	CloseFile(fh, FileName1, t)
	ValidateObjectNotFoundErrOnGCS(FileName1, t)
}

func TestUnlinkOfSyncedLocalFile(t *testing.T) {
	setup.SetupTestDirectory(testDirPath)
	// Create local file and sync to GCS.
	filePath, fh := CreateLocalFileInTestDir(testDirPath, FileName1, t)
	CloseFileAndValidateObjectContents(fh, FileName1, "", t)

	// Attempt to unlink synced local file.
	UnlinkShouldNotThrowError(filePath, t)

	// Verify unlink operation succeeds.
	ValidateNoFileOrDirError(path.Join(testDirPath, FileName1), t)
	ValidateObjectNotFoundErrOnGCS(FileName1, t)
}
