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

// import (
// 	"os"
// 	"path"
// 	"strings"
// 	"testing"

// 	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
// 	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
// 	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
// )

// ////////////////////////////////////////////////////////////////////////
// // Helpers
// ////////////////////////////////////////////////////////////////////////

// func verifyRenameOperationNotSupported(err error, t *testing.T) {
// 	if err == nil || !strings.Contains(err.Error(), "operation not supported") {
// 		t.Fatalf("os.Rename(), expected err: %s, got err: %v",
// 			"operation not supported", err)
// 	}
// }

// ////////////////////////////////////////////////////////////////////////
// // Tests
// ////////////////////////////////////////////////////////////////////////

// func (t *CommonLocalFileTestSuite) TestRenameOfLocalFileFails() {
// 	t.testDirPath = setup.SetupTestDirectory(t.testDirName)
// 	// Create local file with some content.
// 	_, fh := CreateLocalFileInTestDir(t.ctx, t.storageClient, t.testDirPath, FileName1, t.T())
// 	WritingToLocalFileShouldNotWriteToGCS(t.ctx, t.storageClient, fh, t.testDirName, FileName1, t.T())

// 	// Attempt to rename local file.
// 	err := os.Rename(
// 		path.Join(t.testDirPath, FileName1),
// 		path.Join(t.testDirPath, NewFileName))

// 	// Verify rename operation fails.
// 	verifyRenameOperationNotSupported(err, t.T())
// 	// write more content to local file.
// 	WritingToLocalFileShouldNotWriteToGCS(t.ctx, t.storageClient, fh, t.testDirName, FileName1, t.T())
// 	// Close the local file.
// 	CloseFileAndValidateContentFromGCS(t.ctx, t.storageClient, fh, t.testDirName,
// 		FileName1, FileContents+FileContents, t.T())
// }

// func (t *CommonLocalFileTestSuite) TestRenameOfDirectoryWithLocalFileFails() {
// 	t.testDirPath = setup.SetupTestDirectory(t.testDirName)
// 	//Create directory with 1 synced and 1 local file.
// 	operations.CreateDirectory(path.Join(t.testDirPath, ExplicitDirName), t.T())
// 	// Create synced file.
// 	CreateObjectInGCSTestDir(t.ctx, t.storageClient, t.testDirName,
// 		path.Join(ExplicitDirName, FileName1), GCSFileContent, t.T())
// 	// Create local file with some content.
// 	_, fh := CreateLocalFileInTestDir(t.ctx, t.storageClient, t.testDirPath,
// 		path.Join(ExplicitDirName, FileName2), t.T())
// 	WritingToLocalFileShouldNotWriteToGCS(t.ctx, t.storageClient, fh, t.testDirName,
// 		path.Join(ExplicitDirName, FileName2), t.T())

// 	// Attempt to rename directory containing local file.
// 	err := os.Rename(
// 		path.Join(t.testDirPath, ExplicitDirName),
// 		path.Join(t.testDirPath, NewDirName))

// 	// Verify rename operation fails.
// 	verifyRenameOperationNotSupported(err, t.T())
// 	// Write more content to local file.
// 	WritingToLocalFileShouldNotWriteToGCS(t.ctx, t.storageClient, fh, t.testDirName, FileName2, t.T())
// 	// Close the local file.
// 	CloseFileAndValidateContentFromGCS(t.ctx, t.storageClient, fh, t.testDirName,
// 		path.Join(ExplicitDirName, FileName2), FileContents+FileContents, t.T())
// }

// func (t *CommonLocalFileTestSuite) TestRenameOfLocalFileSucceedsAfterSync() {
// 	t.TestRenameOfLocalFileFails()

// 	// Attempt to Rename synced file.
// 	err := os.Rename(
// 		path.Join(t.testDirPath, FileName1),
// 		path.Join(t.testDirPath, NewFileName))

// 	// Validate.
// 	if err != nil {
// 		t.T().Fatalf("os.Rename() failed on synced file: %v", err)
// 	}
// 	ValidateObjectContentsFromGCS(t.ctx, t.storageClient, t.testDirName, NewFileName,
// 		FileContents+FileContents, t.T())
// 	ValidateObjectNotFoundErrOnGCS(t.ctx, t.storageClient, t.testDirName, FileName1, t.T())
// }

// func (t *CommonLocalFileTestSuite) TestRenameOfDirectoryWithLocalFileSucceedsAfterSync() {
// 	t.TestRenameOfDirectoryWithLocalFileFails()

// 	// Attempt to rename directory again after sync.
// 	err := os.Rename(
// 		path.Join(t.testDirPath, ExplicitDirName),
// 		path.Join(t.testDirPath, NewDirName))

// 	// Validate.
// 	if err != nil {
// 		t.T().Fatalf("os.Rename() failed on directory containing synced files: %v", err)
// 	}
// 	ValidateObjectContentsFromGCS(t.ctx, t.storageClient, t.testDirName,
// 		path.Join(NewDirName, FileName1), GCSFileContent, t.T())
// 	ValidateObjectNotFoundErrOnGCS(t.ctx, t.storageClient, t.testDirName,
// 		path.Join(ExplicitDirName, FileName1), t.T())
// 	ValidateObjectContentsFromGCS(t.ctx, t.storageClient, t.testDirName,
// 		path.Join(NewDirName, FileName2), FileContents+FileContents, t.T())
// 	ValidateObjectNotFoundErrOnGCS(t.ctx, t.storageClient, t.testDirName,
// 		path.Join(ExplicitDirName, FileName2), t.T())
// }
