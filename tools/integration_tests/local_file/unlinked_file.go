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
package local_file

import (
	"path"

	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
)

func (t *CommonLocalFileTestSuite) TestStatOnUnlinkedLocalFile() {
	// Unlink the local file.
	operations.RemoveFile(t.filePath)

	// Stat the local file and validate error.
	operations.ValidateNoFileOrDirError(t.T(), path.Join(t.testDirPath, FileName1))

	// Close the file and validate that file is not created on GCS.
	operations.CloseFileShouldNotThrowError(t.fh, t.T())
	ValidateObjectNotFoundErrOnGCS(t.ctx, t.storageClient, t.testDirName, FileName1, t.T())
}

func (t *CommonLocalFileTestSuite) TestReadDirContainingUnlinkedLocalFiles() {
	// Create more local files.
	_, fh2 := CreateLocalFileInTestDir(t.ctx, t.storageClient, t.testDirPath, FileName2, t.T())
	filepath3, fh3 := CreateLocalFileInTestDir(t.ctx, t.storageClient, t.testDirPath, FileName3, t.T())
	// Unlink local file 2.
	operations.RemoveFile(filepath3)

	// Attempt to list testDir.
	entries := operations.ReadDirectory(t.testDirPath, t.T())

	// Verify unlinked entries are not listed.
	operations.VerifyCountOfDirectoryEntries(2, len(entries), t.T())
	operations.VerifyFileEntry(entries[0], FileName1, 0, t.T())
	operations.VerifyFileEntry(entries[1], FileName2, 0, t.T())
	// Close the local files and validate they are written to GCS.
	CloseFileAndValidateContentFromGCS(t.ctx, t.storageClient, t.fh, t.testDirName,
		FileName1, "", t.T())
	CloseFileAndValidateContentFromGCS(t.ctx, t.storageClient, fh2, t.testDirName,
		FileName2, "", t.T())
	// Verify unlinked file is not written to GCS.
	operations.CloseFileShouldNotThrowError(fh3, t.T())
	ValidateObjectNotFoundErrOnGCS(t.ctx, t.storageClient, t.testDirName, FileName3, t.T())
}

func (t *CommonLocalFileTestSuite) TestWriteOnUnlinkedLocalFileSucceeds() {
	// Verify unlink operation succeeds.
	operations.RemoveFile(t.filePath)
	operations.ValidateNoFileOrDirError(t.T(), path.Join(t.testDirPath, FileName1))

	// Write to unlinked local file.
	operations.WriteWithoutClose(t.fh, FileContents, t.T())

	// Validate flush file does not throw error.
	operations.CloseFileShouldNotThrowError(t.fh, t.T())
	// Validate unlinked file is not written to GCS.
	ValidateObjectNotFoundErrOnGCS(t.ctx, t.storageClient, t.testDirName, FileName1, t.T())
}

func (t *CommonLocalFileTestSuite) TestSyncOnUnlinkedLocalFile() {
	// Attempt to unlink local file.
	operations.RemoveFile(t.filePath)
	// Verify unlink operation succeeds.
	operations.ValidateNoFileOrDirError(t.T(), path.Join(t.testDirPath, FileName1))

	// Validate sync operation does not write to GCS after unlink.
	operations.SyncFile(t.fh, t.T())

	ValidateObjectNotFoundErrOnGCS(t.ctx, t.storageClient, t.testDirName, FileName1, t.T())
	// Close the local file and validate it is not present on GCS.
	operations.CloseFileShouldNotThrowError(t.fh, t.T())
	ValidateObjectNotFoundErrOnGCS(t.ctx, t.storageClient, t.testDirName, FileName1, t.T())
}

func (t *CommonLocalFileTestSuite) TestFileWithSameNameCanBeCreatedWhenDeletedBeforeSync() {
	// Write some content.
	operations.WriteWithoutClose(t.fh, FileContents, t.T())
	// Remove and close the file.
	operations.RemoveFile(t.filePath)
	// Currently flush calls returns error if unlinked. Ignoring that error here.
	_ = t.fh.Close()
	// Validate that file is not created on  GCS
	ValidateObjectNotFoundErrOnGCS(t.ctx, t.storageClient, t.testDirName, FileName1, t.T())
	// Verify unlink operation succeeds.
	operations.ValidateNoFileOrDirError(t.T(), path.Join(t.testDirPath, FileName1))

	// Create a local file.
	_, t.fh = CreateLocalFileInTestDir(t.ctx, t.storageClient, t.testDirPath, FileName1, t.T())

	newContents := "newContents"
	operations.WriteWithoutClose(t.fh, newContents, t.T())
	CloseFileAndValidateContentFromGCS(t.ctx, t.storageClient, t.fh, t.testDirName, FileName1, newContents, t.T())
}

func (t *CommonLocalFileTestSuite) TestFileWithSameNameCanBeCreatedAfterDelete() {
	// Write some content.
	operations.WriteWithoutClose(t.fh, FileContents, t.T())
	CloseFileAndValidateContentFromGCS(t.ctx, t.storageClient, t.fh, t.testDirName,
		FileName1, FileContents, t.T())
	// Remove  the file.
	operations.RemoveFile(t.filePath)
	// Validate that file id deleted from GCS
	ValidateObjectNotFoundErrOnGCS(t.ctx, t.storageClient, t.testDirName, FileName1, t.T())
	// Verify unlink operation succeeds.
	operations.ValidateNoFileOrDirError(t.T(), path.Join(t.testDirPath, FileName1))

	// Create a local file.
	_, t.fh = CreateLocalFileInTestDir(t.ctx, t.storageClient, t.testDirPath, FileName1, t.T())

	newContents := "newContents"
	operations.WriteWithoutClose(t.fh, newContents, t.T())
	CloseFileAndValidateContentFromGCS(t.ctx, t.storageClient, t.fh, t.testDirName, FileName1, newContents, t.T())
}
