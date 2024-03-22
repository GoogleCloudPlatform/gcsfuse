// Copyright 2023 Google Inc. All Rights Reserved.
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

func TestCreateSymlinkForLocalFile(t *testing.T) {
	_, _, fh := createAndVerifySymLink(t)
	CloseFileAndValidateContentFromGCS(ctx, storageClient, fh, testDirName,
		FileName1, FileContents, t)
}

func TestReadSymlinkForDeletedLocalFile(t *testing.T) {
	filePath, symlink, fh := createAndVerifySymLink(t)
	// Remove filePath and then close the fileHandle to avoid syncing to GCS.
	operations.RemoveFile(filePath)
	operations.CloseFileShouldNotThrowError(fh, t)
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, FileName1, t)

	// Reading symlink should fail.
	_, err := os.Stat(symlink)
	if err == nil || !strings.Contains(err.Error(), "no such file or directory") {
		t.Fatalf("Reading symlink for deleted local file did not fail.")
	}
}
