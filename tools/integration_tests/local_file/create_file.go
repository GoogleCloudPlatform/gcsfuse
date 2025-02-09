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
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *CommonLocalFileTestSuite) TestNewFileShouldNotGetSyncedToGCSTillClose() {
	// Writing to local file should not write to GCS.
	WritingToLocalFileShouldNotWriteToGCS(t.ctx, t.storageClient, t.fh, t.testDirName, FileName1, t.T())
	// Close and validate file content from GCS.
	CloseFileAndValidateContentFromGCS(t.ctx, t.storageClient, t.fh, t.testDirName, FileName1, FileContents, t.T())
}

func (t *CommonLocalFileTestSuite) TestNewFileUnderExplicitDirectoryShouldNotGetSyncedToGCSTillClose() {

	// Make explicit directory.
	operations.CreateDirectory(path.Join(t.testDirPath, ExplicitDirName), t.T())

	_, t.fh = CreateLocalFileInTestDir(t.ctx, t.storageClient, t.testDirPath, path.Join(ExplicitDirName, ExplicitFileName1), t.T())
	// Validate.
	NewFileShouldGetSyncedToGCSAtClose(t.ctx, t.storageClient, t.testDirPath, path.Join(ExplicitDirName, ExplicitFileName1), t.T())
}

func (t *CommonLocalFileTestSuite) TestCreateNewFileWhenSameFileExistsOnGCS() {
	t.testDirPath = setup.SetupTestDirectory(t.testDirName)
	// Create a local file.
	_, fh := CreateLocalFileInTestDir(t.ctx, t.storageClient, t.testDirPath, FileName1, t.T())

	// Create a file on GCS with the same name.
	CreateObjectInGCSTestDir(t.ctx, t.storageClient, t.testDirName, FileName1, GCSFileContent, t.T())

	// Write to local file.
	operations.WriteWithoutClose(fh, FileContents, t.T())
	// Validate closing local file throws error.
	err := fh.Close()
	operations.ValidateStaleNFSFileHandleError(t.T(), err)
	//  Ensure that the content on GCS is not overwritten.
	ValidateObjectContentsFromGCS(t.ctx, t.storageClient, t.testDirName, FileName1, GCSFileContent, t.T())
}

func (t *CommonLocalFileTestSuite) TestEmptyFileCreation() {
	t.testDirPath = setup.SetupTestDirectory(t.testDirName)
	// Create a local file.
	_, fh := CreateLocalFileInTestDir(t.ctx, t.storageClient, t.testDirPath, FileName1, t.T())

	// Close the file and validate that the file is created on GCS.
	CloseFileAndValidateContentFromGCS(t.ctx, t.storageClient, fh, t.testDirName, FileName1, "", t.T())
}
