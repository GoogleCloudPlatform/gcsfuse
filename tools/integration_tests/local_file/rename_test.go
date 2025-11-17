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

	"github.com/stretchr/testify/require"
	. "github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
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
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create local file with some content.
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t.T())
	defer operations.CloseFileShouldNotThrowError(t.T(), fh)
	WritingToLocalFileShouldNotWriteToGCS(ctx, storageClient, fh, testDirName, FileName1, t.T())

	// Attempt to rename local file.
	err := os.Rename(
		path.Join(testDirPath, FileName1),
		path.Join(testDirPath, NewFileName))

	// Validate that move didn't throw any error.
	require.NoError(t.T(), err)
	// Verify the new object contents.
	ValidateObjectContentsFromGCS(ctx, storageClient, testDirName, NewFileName, FileContents, t.T())
	// Validate old object is deleted.
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, FileName1, t.T())
}

func (t *LocalFileTestSuite) TestRenameOfDirectoryWithLocalFileFails() {
	testDirPath = setup.SetupTestDirectory(testDirName)
	//Create directory with 1 synced and 1 local file.
	operations.CreateDirectory(path.Join(testDirPath, ExplicitDirName), t.T())
	// Create synced file.
	CreateObjectInGCSTestDir(ctx, storageClient, testDirName,
		path.Join(ExplicitDirName, FileName1), GCSFileContent, t.T())
	// Create local file with some content.
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath,
		path.Join(ExplicitDirName, FileName2), t.T())
	WritingToLocalFileShouldNotWriteToGCS(ctx, storageClient, fh, testDirName,
		path.Join(ExplicitDirName, FileName2), t.T())

	// Attempt to rename directory containing local file.
	err := os.Rename(
		path.Join(testDirPath, ExplicitDirName),
		path.Join(testDirPath, NewDirName))

	// Verify rename operation fails.
	verifyRenameOperationNotSupported(err, t.T())
	// Write more content to local file.
	WritingToLocalFileShouldNotWriteToGCS(ctx, storageClient, fh, testDirName, FileName2, t.T())
	// Close the local file.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName,
		path.Join(ExplicitDirName, FileName2), FileContents+FileContents, t.T())
}

func (t *LocalFileTestSuite) TestRenameOfLocalFileSucceedsAfterSync() {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create local file with some content.
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t.T())
	WritingToLocalFileShouldNotWriteToGCS(ctx, storageClient, fh, testDirName, FileName1, t.T())
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName, FileName1, FileContents, t.T())

	// Attempt to Rename synced file.
	err := os.Rename(
		path.Join(testDirPath, FileName1),
		path.Join(testDirPath, NewFileName))

	// Validate.
	if err != nil {
		t.T().Fatalf("os.Rename() failed on synced file: %v", err)
	}
	ValidateObjectContentsFromGCS(ctx, storageClient, testDirName, NewFileName, FileContents, t.T())
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, FileName1, t.T())
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
	ValidateObjectContentsFromGCS(ctx, storageClient, testDirName,
		path.Join(NewDirName, FileName1), GCSFileContent, t.T())
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName,
		path.Join(ExplicitDirName, FileName1), t.T())
	ValidateObjectContentsFromGCS(ctx, storageClient, testDirName,
		path.Join(NewDirName, FileName2), FileContents+FileContents, t.T())
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName,
		path.Join(ExplicitDirName, FileName2), t.T())
}
