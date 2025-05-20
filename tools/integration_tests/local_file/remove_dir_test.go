// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Provides integration tests for removeDir operation on directories containing local files.
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

func (t *LocalFileTestSuite) TestRmDirOfDirectoryContainingGCSAndLocalFiles() {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create explicit directory with one synced and one local file.
	operations.CreateDirectory(path.Join(testDirPath, ExplicitDirName), t.T())
	syncedFile := path.Join(ExplicitDirName, FileName1)
	localFile := path.Join(ExplicitDirName, FileName2)
	_, fh1 := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, syncedFile, t.T())
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh1, testDirName, syncedFile, "", t.T())
	_, fh2 := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, localFile, t.T())

	// Attempt to remove explicit directory.
	operations.RemoveDir(path.Join(testDirPath, ExplicitDirName))

	// Verify that directory is removed.
	operations.ValidateNoFileOrDirError(t.T(), path.Join(testDirPath, ExplicitDirName))
	// Validate writing content to unlinked local file does not throw error.
	operations.WriteWithoutClose(fh2, FileContents, t.T())
	// Validate flush file does not throw error and does not create object on GCS.
	operations.CloseFileShouldNotThrowError(t.T(), fh2)
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, localFile, t.T())
	// Validate synced files are also deleted.
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, syncedFile, t.T())
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, ExplicitDirName, t.T())
}

func (t *LocalFileTestSuite) TestRmDirOfDirectoryContainingOnlyLocalFiles() {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create a directory with two local files.
	operations.CreateDirectory(path.Join(testDirPath, ExplicitDirName), t.T())
	localFile1 := path.Join(ExplicitDirName, FileName1)
	localFile2 := path.Join(ExplicitDirName, FileName2)
	_, fh1 := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, localFile1, t.T())
	_, fh2 := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, localFile2, t.T())

	// Attempt to remove explicit directory.
	operations.RemoveDir(path.Join(testDirPath, ExplicitDirName))

	// Verify rmDir operation succeeds.
	operations.ValidateNoFileOrDirError(t.T(), path.Join(testDirPath, ExplicitDirName))
	// Close the local files and validate they are not present on GCS.
	operations.CloseFileShouldNotThrowError(t.T(), fh1)
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, localFile1, t.T())
	operations.CloseFileShouldNotThrowError(t.T(), fh2)
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, localFile2, t.T())
	// Validate directory is also deleted.
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, ExplicitDirName, t.T())
}
