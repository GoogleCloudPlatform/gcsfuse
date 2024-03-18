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

	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

func TestStatOnUnlinkedLocalFile(t *testing.T) {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create a local file.
	filePath, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t)
	// Unlink the local file.
	operations.RemoveFile(filePath)

	// Stat the local file and validate error.
	operations.ValidateNoFileOrDirError(path.Join(testDirPath, FileName1), t)

	// Close the file and validate that file is not created on GCS.
	operations.CloseFileShouldNotThrowError(fh, t)
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, FileName1, t)
}

func TestReadDirContainingUnlinkedLocalFiles(t *testing.T) {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create local files.
	_, fh1 := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t)
	_, fh2 := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName2, t)
	filepath3, fh3 := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName3, t)
	// Unlink local file 3.
	operations.RemoveFile(filepath3)

	// Attempt to list testDir.
	entries := operations.ReadDirectory(testDirPath, t)

	// Verify unlinked entries are not listed.
	operations.VerifyCountOfDirectoryEntries(2, len(entries), t)
	operations.VerifyFileEntry(entries[0], FileName1, 0, t)
	operations.VerifyFileEntry(entries[1], FileName2, 0, t)
	// Close the local files and validate they are written to GCS.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh1, testDirName,
		FileName1, "", t)
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh2, testDirName,
		FileName2, "", t)
	// Verify unlinked file is not written to GCS.
	operations.CloseFileShouldNotThrowError(fh3, t)
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, FileName3, t)
}

func TestWriteOnUnlinkedLocalFileSucceeds(t *testing.T) {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create local file.
	filepath, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t)
	// Verify unlink operation succeeds.
	operations.RemoveFile(filepath)
	operations.ValidateNoFileOrDirError(path.Join(testDirPath, FileName1), t)

	// Write to unlinked local file.
	operations.WriteWithoutClose(fh, FileContents, t)

	// Validate flush file does not throw error.
	operations.CloseFileShouldNotThrowError(fh, t)
	// Validate unlinked file is not written to GCS.
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, FileName1, t)
}

func TestSyncOnUnlinkedLocalFile(t *testing.T) {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create local file.
	filepath, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t)

	// Attempt to unlink local file.
	operations.RemoveFile(filepath)

	// Verify unlink operation succeeds.
	operations.ValidateNoFileOrDirError(path.Join(testDirPath, FileName1), t)
	// Validate sync operation does not write to GCS after unlink.
	operations.SyncFile(fh, t)
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, FileName1, t)
	// Close the local file and validate it is not present on GCS.
	operations.CloseFileShouldNotThrowError(fh, t)
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, FileName1, t)
}
