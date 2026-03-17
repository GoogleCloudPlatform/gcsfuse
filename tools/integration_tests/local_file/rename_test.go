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

// Provides integration tests for rename operation on local files.
package local_file

import (
	"os"
	"path"
	"strings"
	"testing"

	. "github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/require"
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func verifyRenameOperationNotSupported(err error, t *testing.T) {
	if err == nil || !strings.Contains(err.Error(), "operation not supported") {
		t.Fatalf("os.Rename(), expected err: %s, got err: %v",
			"operation not supported", err)
	}
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *LocalFileTestSuite) TestRenameOfLocalFile() {
	fileName := path.Base(t.T().Name())
	newFileName := fileName + "new"
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create local file with some content.
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, fileName, t.T())
	defer operations.CloseFileShouldNotThrowError(t.T(), fh)
	WritingToLocalFileShouldNotWriteToGCS(ctx, storageClient, fh, testDirName, fileName, t.T())

	// Attempt to rename local file.
	err := os.Rename(
		path.Join(testDirPath, fileName),
		path.Join(testDirPath, newFileName))

	// Validate that move didn't throw any error.
	require.NoError(t.T(), err)
	// Verify the new object contents.
	ValidateObjectContentsFromGCS(ctx, storageClient, testDirName, newFileName, FileContents, t.T())
	// Validate old object is deleted.
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, fileName, t.T())
}

func (t *LocalFileTestSuite) TestRenameOfDirectoryWithLocalFileFails() {
	fileName1 := path.Base(t.T().Name()) + "1"
	fileName2 := path.Base(t.T().Name()) + "2"
	testDirPath = setup.SetupTestDirectory(testDirName)
	//Create directory with 1 synced and 1 local file.
	operations.CreateDirectory(path.Join(testDirPath, ExplicitDirName), t.T())
	// Create synced file.
	CreateObjectInGCSTestDir(ctx, storageClient, testDirName, path.Join(ExplicitDirName, fileName1), GCSFileContent, t.T())
	// Create local file with some content.
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, path.Join(ExplicitDirName, fileName2), t.T())
	WritingToLocalFileShouldNotWriteToGCS(ctx, storageClient, fh, testDirName, path.Join(ExplicitDirName, fileName2), t.T())

	// Attempt to rename directory containing local file.
	err := os.Rename(
		path.Join(testDirPath, ExplicitDirName),
		path.Join(testDirPath, NewDirName))

	// Verify rename operation fails.
	verifyRenameOperationNotSupported(err, t.T())
	// Write more content to local file.
	WritingToLocalFileShouldNotWriteToGCS(ctx, storageClient, fh, testDirName, fileName2, t.T())
	// Close the local file.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName, path.Join(ExplicitDirName, fileName2), FileContents+FileContents, t.T())
}

func (t *LocalFileTestSuite) TestRenameOfLocalFileSucceedsAfterSync() {
	fileName := path.Base(t.T().Name())
	newFileName := fileName + "new"
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create local file with some content.
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, fileName, t.T())
	WritingToLocalFileShouldNotWriteToGCS(ctx, storageClient, fh, testDirName, fileName, t.T())
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName, fileName, FileContents, t.T())

	// Attempt to Rename synced file.
	err := os.Rename(
		path.Join(testDirPath, fileName),
		path.Join(testDirPath, newFileName))

	// Validate.
	if err != nil {
		t.T().Fatalf("os.Rename() failed on synced file: %v", err)
	}
	ValidateObjectContentsFromGCS(ctx, storageClient, testDirName, newFileName, FileContents, t.T())
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, fileName, t.T())
}

func (t *LocalFileTestSuite) TestRenameOfDirectoryWithLocalFileSucceedsAfterSync() {
	t.TestRenameOfDirectoryWithLocalFileFails()

	// Attempt to rename directory again after sync.
	err := os.Rename(
		path.Join(testDirPath, ExplicitDirName),
		path.Join(testDirPath, NewDirName))

	// Validate.
	if err != nil {
		t.T().Fatalf("os.Rename() failed on directory containing synced files: %v", err)
	}
	fileName1 := path.Base(t.T().Name()) + "1"
	fileName2 := path.Base(t.T().Name()) + "2"
	ValidateObjectContentsFromGCS(ctx, storageClient, testDirName, path.Join(NewDirName, fileName1), GCSFileContent, t.T())
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, path.Join(ExplicitDirName, fileName1), t.T())
	ValidateObjectContentsFromGCS(ctx, storageClient, testDirName, path.Join(NewDirName, fileName2), FileContents+FileContents, t.T())
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, path.Join(ExplicitDirName, fileName2), t.T())
}
