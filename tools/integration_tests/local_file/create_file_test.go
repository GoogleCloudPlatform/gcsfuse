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

	. "github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *LocalFileTestSuite) TestNewFileShouldNotGetSyncedToGCSTillClose() {
	testDirPath = setup.SetupTestDirectory(testDirName)

	// Validate.
	NewFileShouldGetSyncedToGCSAtClose(ctx, storageClient, testDirPath, FileName1, t.T())
}

func (t *LocalFileTestSuite) TestNewFileUnderExplicitDirectoryShouldNotGetSyncedToGCSTillClose() {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Make explicit directory.
	operations.CreateDirectory(path.Join(testDirPath, ExplicitDirName), t.T())

	// Validate.
	NewFileShouldGetSyncedToGCSAtClose(ctx, storageClient, testDirPath, path.Join(ExplicitDirName, ExplicitFileName1), t.T())
}

func (t *LocalFileTestSuite) TestCreateNewFileWhenSameFileExistsOnGCS() {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create a local file.
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t.T())

	// Create a file on GCS with the same name.
	CreateObjectInGCSTestDir(ctx, storageClient, testDirName, FileName1, GCSFileContent, t.T())

	// Write to local file.
	operations.WriteWithoutClose(fh, FileContents, t.T())
	// Validate closing local file throws error.
	err := fh.Close()
	operations.ValidateESTALEError(t.T(), err)
	//  Ensure that the content on GCS is not overwritten.
	ValidateObjectContentsFromGCS(ctx, storageClient, testDirName, FileName1, GCSFileContent, t.T())
}

func (t *LocalFileTestSuite) TestEmptyFileCreation() {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create a local file.
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t.T())

	// Close the file and validate that the file is created on GCS.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName, FileName1, "", t.T())
}
