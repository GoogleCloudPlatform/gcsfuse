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

	. "github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

func (t *LocalFileTestSuite) TestStatOnUnlinkedLocalFile() {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create a local file.
	filePath, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t.T())
	// Unlink the local file.
	operations.RemoveFile(filePath)

	// Stat the local file and validate error.
	operations.ValidateNoFileOrDirError(t.T(), path.Join(testDirPath, FileName1))

	// Close the file and validate that file is not created on GCS.
	operations.CloseFileShouldNotThrowError(t.T(), fh)
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, FileName1, t.T())
}

func (t *LocalFileTestSuite) TestReadDirContainingUnlinkedLocalFiles() {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create local files.
	_, fh1 := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t.T())
	_, fh2 := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName2, t.T())
	filepath3, fh3 := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName3, t.T())
	// Unlink local file 3.
	operations.RemoveFile(filepath3)

	// Attempt to list testDir.
	entries := operations.ReadDirectory(testDirPath, t.T())

	// Verify unlinked entries are not listed.
	operations.VerifyCountOfDirectoryEntries(2, len(entries), t.T())
	operations.VerifyFileEntry(entries[0], FileName1, 0, t.T())
	operations.VerifyFileEntry(entries[1], FileName2, 0, t.T())
	// Close the local files and validate they are written to GCS.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh1, testDirName,
		FileName1, "", t.T())
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh2, testDirName,
		FileName2, "", t.T())
	// Verify unlinked file is not written to GCS.
	operations.CloseFileShouldNotThrowError(t.T(), fh3)
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, FileName3, t.T())
}

func (t *LocalFileTestSuite) TestWriteOnUnlinkedLocalFileSucceeds() {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create local file.
	filepath, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t.T())
	// Verify unlink operation succeeds.
	operations.RemoveFile(filepath)
	operations.ValidateNoFileOrDirError(t.T(), path.Join(testDirPath, FileName1))

	// Write to unlinked local file.
	operations.WriteWithoutClose(fh, FileContents, t.T())

	// Validate flush file does not throw error.
	operations.CloseFileShouldNotThrowError(t.T(), fh)
	// Validate unlinked file is not written to GCS.
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, FileName1, t.T())
}

func (t *LocalFileTestSuite) TestSyncOnUnlinkedLocalFile() {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create local file.
	filepath, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t.T())

	// Attempt to unlink local file.
	operations.RemoveFile(filepath)

	// Verify unlink operation succeeds.
	operations.ValidateNoFileOrDirError(t.T(), path.Join(testDirPath, FileName1))
	// Validate sync operation does not write to GCS after unlink.
	operations.SyncFile(fh, t.T())
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, FileName1, t.T())
	// Close the local file and validate it is not present on GCS.
	operations.CloseFileShouldNotThrowError(t.T(), fh)
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, FileName1, t.T())
}

func (t *LocalFileTestSuite) TestFileWithSameNameCanBeCreatedWhenDeletedBeforeSync() {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create a local file.
	filePath, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t.T())
	// Write some content.
	operations.WriteWithoutClose(fh, FileContents, t.T())
	// Remove and close the file.
	operations.RemoveFile(filePath)
	// Currently flush calls returns error if unlinked. Ignoring that error here.
	_ = fh.Close()
	// Validate that file is not created on  GCS
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, FileName1, t.T())
	// Verify unlink operation succeeds.
	operations.ValidateNoFileOrDirError(t.T(), path.Join(testDirPath, FileName1))

	// Create a local file.
	_, fh = CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t.T())

	newContents := "newContents"
	operations.WriteWithoutClose(fh, newContents, t.T())
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName, FileName1, newContents, t.T())
}

func (t *LocalFileTestSuite) TestFileWithSameNameCanBeCreatedAfterDelete() {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create a local file.
	filePath, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t.T())
	// Write some content.
	operations.WriteWithoutClose(fh, FileContents, t.T())
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName,
		FileName1, FileContents, t.T())
	// Remove  the file.
	operations.RemoveFile(filePath)
	// Validate that file id deleted from GCS
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, FileName1, t.T())
	// Verify unlink operation succeeds.
	operations.ValidateNoFileOrDirError(t.T(), path.Join(testDirPath, FileName1))

	// Create a local file.
	_, fh = CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t.T())

	newContents := "newContents"
	operations.WriteWithoutClose(fh, newContents, t.T())
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName, FileName1, newContents, t.T())
}
