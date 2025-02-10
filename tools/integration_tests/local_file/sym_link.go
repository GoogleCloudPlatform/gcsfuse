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
	"strings"

	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
)

func (t *localFileTestSuite) createAndVerifySymLink() (symlink string) {
	WritingToLocalFileShouldNotWriteToGCS(t.ctx, t.storageClient, t.fh, t.testDirName, FileName1, t.T())

	// Create the symlink.
	symlink = path.Join(t.testDirPath, "bar")
	operations.CreateSymLink(t.filePath, symlink, t.T())

	// Read the link.
	operations.VerifyReadLink(t.filePath, symlink, t.T())
	operations.VerifyReadFile(symlink, FileContents, t.T())
	return
}

func (t *localFileTestSuite) TestCreateSymlinkForLocalFile() {
	_ = t.createAndVerifySymLink()
	CloseFileAndValidateContentFromGCS(t.ctx, t.storageClient, t.fh, t.testDirName,
		FileName1, FileContents, t.T())
}

func (t *localFileTestSuite) TestReadSymlinkForDeletedLocalFile() {
	symlink := t.createAndVerifySymLink()
	// Remove filePath and then close the fileHandle to avoid syncing to GCS.
	operations.RemoveFile(t.filePath)
	operations.CloseFileShouldNotThrowError(t.fh, t.T())
	ValidateObjectNotFoundErrOnGCS(t.ctx, t.storageClient, t.testDirName, FileName1, t.T())

	// Reading symlink should fail.
	_, err := os.Stat(symlink)
	if err == nil || !strings.Contains(err.Error(), "no such file or directory") {
		t.T().Fatalf("Reading symlink for deleted local file did not fail.")
	}
}
