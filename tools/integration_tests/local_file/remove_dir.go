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

	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *CommonLocalFileTestSuite) TestRmDirOfDirectoryContainingGCSAndLocalFiles() {
	t.testDirPath = setup.SetupTestDirectory(t.testDirName)
	// Create explicit directory with one synced and one local file.
	operations.CreateDirectory(path.Join(t.testDirPath, ExplicitDirName), t.T())
	syncedFile := path.Join(ExplicitDirName, FileName1)
	localFile := path.Join(ExplicitDirName, FileName2)
	_, fh1 := CreateLocalFileInTestDir(t.ctx, t.storageClient, t.testDirPath, syncedFile, t.T())
	CloseFileAndValidateContentFromGCS(t.ctx, t.storageClient, fh1, t.testDirName, syncedFile, "", t.T())
	_, fh2 := CreateLocalFileInTestDir(t.ctx, t.storageClient, t.testDirPath, localFile, t.T())

	// Attempt to remove explicit directory.
	operations.RemoveDir(path.Join(t.testDirPath, ExplicitDirName))

	// Verify that directory is removed.
	operations.ValidateNoFileOrDirError(t.T(), path.Join(t.testDirPath, ExplicitDirName))
	// Validate writing content to unlinked local file does not throw error.
	operations.WriteWithoutClose(fh2, FileContents, t.T())
	// Validate flush file does not throw error and does not create object on GCS.
	operations.CloseFileShouldNotThrowError(fh2, t.T())
	ValidateObjectNotFoundErrOnGCS(t.ctx, t.storageClient, t.testDirName, localFile, t.T())
	// Validate synced files are also deleted.
	ValidateObjectNotFoundErrOnGCS(t.ctx, t.storageClient, t.testDirName, syncedFile, t.T())
	ValidateObjectNotFoundErrOnGCS(t.ctx, t.storageClient, t.testDirName, ExplicitDirName, t.T())
}

func (t *CommonLocalFileTestSuite) TestRmDirOfDirectoryContainingOnlyLocalFiles() {
	t.testDirPath = setup.SetupTestDirectory(t.testDirName)
	// Create a directory with two local files.
	operations.CreateDirectory(path.Join(t.testDirPath, ExplicitDirName), t.T())
	localFile1 := path.Join(ExplicitDirName, FileName1)
	localFile2 := path.Join(ExplicitDirName, FileName2)
	_, fh1 := CreateLocalFileInTestDir(t.ctx, t.storageClient, t.testDirPath, localFile1, t.T())
	_, fh2 := CreateLocalFileInTestDir(t.ctx, t.storageClient, t.testDirPath, localFile2, t.T())

	// Attempt to remove explicit directory.
	operations.RemoveDir(path.Join(t.testDirPath, ExplicitDirName))

	// Verify rmDir operation succeeds.
	operations.ValidateNoFileOrDirError(t.T(), path.Join(t.testDirPath, ExplicitDirName))
	// Close the local files and validate they are not present on GCS.
	operations.CloseFileShouldNotThrowError(fh1, t.T())
	ValidateObjectNotFoundErrOnGCS(t.ctx, t.storageClient, t.testDirName, localFile1, t.T())
	operations.CloseFileShouldNotThrowError(fh2, t.T())
	ValidateObjectNotFoundErrOnGCS(t.ctx, t.storageClient, t.testDirName, localFile2, t.T())
	// Validate directory is also deleted.
	ValidateObjectNotFoundErrOnGCS(t.ctx, t.storageClient, t.testDirName, ExplicitDirName, t.T())
}
