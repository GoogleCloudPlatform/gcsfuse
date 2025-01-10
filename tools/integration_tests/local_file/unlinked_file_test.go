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
	operations.ValidateNoFileOrDirError(t, path.Join(testDirPath, FileName1))

	// Validate closing local file throws error and does not create file on GCS.
	err := fh.Close()
	operations.ValidateStaleNFSFileHandleError(t, err)
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
	// Verify closing unlinked local file throws error and does not write to GCS.
	err := fh3.Close()
	operations.ValidateStaleNFSFileHandleError(t, err)
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, FileName3, t)
}

func TestWriteOnUnlinkedLocalFileSucceeds(t *testing.T) {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create local file.
	filepath, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t)
	// Verify unlink operation succeeds.
	operations.RemoveFile(filepath)
	operations.ValidateNoFileOrDirError(t, path.Join(testDirPath, FileName1))

	// Write to unlinked local file.
	operations.WriteWithoutClose(fh, FileContents, t)

	// Validate flush file throws error.
	err := fh.Close()
	operations.ValidateStaleNFSFileHandleError(t, err)
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
	operations.ValidateNoFileOrDirError(t, path.Join(testDirPath, FileName1))
	// Validate sync and close operations throws error and do not write to GCS after unlink.
	err := fh.Sync()
	operations.ValidateStaleNFSFileHandleError(t, err)
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, FileName1, t)
	err = fh.Close()
	operations.ValidateStaleNFSFileHandleError(t, err)
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, FileName1, t)
}

func TestFileWithSameNameCanBeCreatedWhenDeletedBeforeSync(t *testing.T) {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create a local file.
	filePath, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t)
	// Write some content.
	operations.WriteWithoutClose(fh, FileContents, t)
	// Remove and close the file.
	operations.RemoveFile(filePath)
	// Currently flush calls returns error if unlinked. Ignoring that error here.
	_ = fh.Close()
	// Validate that file is not created on  GCS
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, FileName1, t)
	// Verify unlink operation succeeds.
	operations.ValidateNoFileOrDirError(t, path.Join(testDirPath, FileName1))

	// Create a local file.
	_, fh = CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t)

	newContents := "newContents"
	operations.WriteWithoutClose(fh, newContents, t)
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName, FileName1, newContents, t)
}

func TestFileWithSameNameCanBeCreatedAfterDelete(t *testing.T) {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create a local file.
	filePath, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t)
	// Write some content.
	operations.WriteWithoutClose(fh, FileContents, t)
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName,
		FileName1, FileContents, t)
	// Remove  the file.
	operations.RemoveFile(filePath)
	// Validate that file id deleted from GCS
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, FileName1, t)
	// Verify unlink operation succeeds.
	operations.ValidateNoFileOrDirError(t, path.Join(testDirPath, FileName1))

	// Create a local file.
	_, fh = CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t)

	newContents := "newContents"
	operations.WriteWithoutClose(fh, newContents, t)
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName, FileName1, newContents, t)
}
