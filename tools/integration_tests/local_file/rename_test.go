// Copyright 2023 Google Inc. All Rights Reserved.
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
package local_file_test

import (
	"os"
	"path"
	"strings"
	"testing"

	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
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

func TestRenameOfLocalFileFails(t *testing.T) {
	testDirPath = setup.SetupTestDirectory(testDirName)
	// Create local file with some content.
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath, FileName1, t)
	WritingToLocalFileShouldNotWriteToGCS(ctx, storageClient, fh, testDirName, FileName1, t)

	// Attempt to rename local file.
	err := os.Rename(
		path.Join(testDirPath, FileName1),
		path.Join(testDirPath, NewFileName))

	// Verify rename operation fails.
	verifyRenameOperationNotSupported(err, t)
	// write more content to local file.
	WritingToLocalFileShouldNotWriteToGCS(ctx, storageClient, fh, testDirName, FileName1, t)
	// Close the local file.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName,
		FileName1, FileContents+FileContents, t)
}

func TestRenameOfDirectoryWithLocalFileFails(t *testing.T) {
	testDirPath = setup.SetupTestDirectory(testDirName)
	//Create directory with 1 synced and 1 local file.
	operations.CreateDirectory(path.Join(testDirPath, ExplicitDirName), t)
	// Create synced file.
	CreateObjectInGCSTestDir(ctx, storageClient, testDirName,
		path.Join(ExplicitDirName, FileName1), GCSFileContent, t)
	// Create local file with some content.
	_, fh := CreateLocalFileInTestDir(ctx, storageClient, testDirPath,
		path.Join(ExplicitDirName, FileName2), t)
	WritingToLocalFileShouldNotWriteToGCS(ctx, storageClient, fh, testDirName,
		path.Join(ExplicitDirName, FileName2), t)

	// Attempt to rename directory containing local file.
	err := os.Rename(
		path.Join(testDirPath, ExplicitDirName),
		path.Join(testDirPath, NewDirName))

	// Verify rename operation fails.
	verifyRenameOperationNotSupported(err, t)
	// Write more content to local file.
	WritingToLocalFileShouldNotWriteToGCS(ctx, storageClient, fh, testDirName, FileName2, t)
	// Close the local file.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName,
		path.Join(ExplicitDirName, FileName2), FileContents+FileContents, t)
}

func TestRenameOfLocalFileSucceedsAfterSync(t *testing.T) {
	TestRenameOfLocalFileFails(t)

	// Attempt to Rename synced file.
	err := os.Rename(
		path.Join(testDirPath, FileName1),
		path.Join(testDirPath, NewFileName))

	// Validate.
	if err != nil {
		t.Fatalf("os.Rename() failed on synced file: %v", err)
	}
	ValidateObjectContentsFromGCS(ctx, storageClient, testDirName, NewFileName,
		FileContents+FileContents, t)
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, FileName1, t)
}

func TestRenameOfDirectoryWithLocalFileSucceedsAfterSync(t *testing.T) {
	TestRenameOfDirectoryWithLocalFileFails(t)

	// Attempt to rename directory again after sync.
	err := os.Rename(
		path.Join(testDirPath, ExplicitDirName),
		path.Join(testDirPath, NewDirName))

	// Validate.
	if err != nil {
		t.Fatalf("os.Rename() failed on directory containing synced files: %v", err)
	}
	ValidateObjectContentsFromGCS(ctx, storageClient, testDirName,
		path.Join(NewDirName, FileName1), GCSFileContent, t)
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName,
		path.Join(ExplicitDirName, FileName1), t)
	ValidateObjectContentsFromGCS(ctx, storageClient, testDirName,
		path.Join(NewDirName, FileName2), FileContents+FileContents, t)
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName,
		path.Join(ExplicitDirName, FileName2), t)
}
