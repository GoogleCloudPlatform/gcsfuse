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

// Provides integration tests for create local file.
package local_file

import (
	"path"

	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
)

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *CommonLocalFileTestSuite) TestNewFileShouldNotGetSyncedToGCSTillClose() {
	testDirName := GetDirName(t.testDirPath)

	// Writing contents to local file shouldn't create file on GCS.
	operations.WriteWithoutClose(t.fh, FileContents, t.T())

	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, t.fileName, t.T())
	// Close the file and validate if the file is created on GCS.
	operations.CloseFileShouldNotThrowError(t.fh, t.T())
	ValidateObjectContentsFromGCS(ctx, storageClient, testDirName, t.fileName, FileContents, t.T())
}

func (t *CommonLocalFileTestSuite) TestNewFileUnderExplicitDirectoryShouldNotGetSyncedToGCSTillClose() {
	// Make explicit directory.
	t.testDirName = path.Join(t.testDirName, ExplicitDirName)
	t.testDirPath = path.Join(t.testDirPath, ExplicitDirName)
	operations.CreateDirectory(t.testDirPath, t.T())
	_, t.fh = CreateLocalFileInTestDir(ctx, storageClient, t.testDirPath, t.fileName, t.T())

	t.WritingToLocalFileShouldNotWriteToGCS(ctx, storageClient, t.testDirName)

	// Close the file and validate if the file is created on GCS.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, t.fh, t.testDirName, t.fileName, FileContents, t.T())
}

func (t *CommonLocalFileTestSuite) TestCreateNewFileWhenSameFileExistsOnGCS() {
	// Create a file on GCS with the same name.
	CreateObjectInGCSTestDir(ctx, storageClient, t.testDirName, t.fileName, GCSFileContent, t.T())

	// Write to local file.
	operations.WriteWithoutClose(t.fh, FileContents, t.T())
	// Validate closing local file throws error.
	err := t.fh.Close()
	operations.ValidateStaleNFSFileHandleError(t.T(), err)
	//  Ensure that the content on GCS is not overwritten.
	ValidateObjectContentsFromGCS(ctx, storageClient, t.testDirName, t.fileName, GCSFileContent, t.T())
}

func (t *CommonLocalFileTestSuite) TestEmptyFileCreation() {
	// Close the file and validate that the file is created on GCS.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, t.fh, t.testDirName, t.fileName, "", t.T())
}
