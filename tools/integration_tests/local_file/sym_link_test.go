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
// Provides integration tests for symlink operation on local files.
package local_file

import (
	"os"
	"path"
	"testing"

	. "github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createAndVerifySymLink(t *testing.T) (filePath, symlink string, fh *os.File) {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create a local file.
	filePath, fh = CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t)
	WritingToLocalFileShouldNotWriteToGCS(ctx, storageClient, fh, testDirName, FileName1, t)

	// Create the symlink.
	symlink = path.Join(testDirPath, "bar")
	operations.CreateSymLink(filePath, symlink, t)

	// Read the link.
	operations.VerifyReadLink(filePath, symlink, t)
	operations.VerifyReadFile(symlink, FileContents, t)
	return
}

func (t *LocalFileTestSuite) TestCreateSymlinkForLocalFile() {
	_, _, fh := createAndVerifySymLink(t.T())
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName, FileName1, FileContents, t.T())
}

func (t *LocalFileTestSuite) TestReadSymlinkForDeletedLocalFile() {
	filePath, symlink, fh := createAndVerifySymLink(t.T())
	// Remove filePath and then close the fileHandle to avoid syncing to GCS.
	operations.RemoveFile(filePath)
	operations.CloseFileShouldNotThrowError(t.T(), fh)
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, FileName1, t.T())

	// Reading symlink should fail.
	_, err := os.Stat(symlink)

	require.Error(t.T(), err)
	assert.True(t.T(), os.IsNotExist(err), "Reading symlink for deleted local file should have failed with 'no such file or directory'. Got: %v", err)
}

func (t *LocalFileTestSuite) TestRenameSymlinkForLocalFile() {
	filePath, symlinkPath, fh := createAndVerifySymLink(t.T())
	newSymlinkPath := path.Join(testDirPath, "newSymlink")

	err := os.Rename(symlinkPath, newSymlinkPath)

	require.NoError(t.T(), err, "os.Rename failed for symlink")
	_, err = os.Lstat(symlinkPath)
	require.Error(t.T(), err)
	assert.True(t.T(), os.IsNotExist(err), "Old symlink should not exist after rename. err: %v", err)
	operations.VerifyReadLink(filePath, newSymlinkPath, t.T())
	operations.VerifyReadFile(newSymlinkPath, FileContents, t.T())
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName, FileName1, FileContents, t.T())
}
