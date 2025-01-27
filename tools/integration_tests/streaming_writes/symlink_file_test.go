// Copyright 2025 Google LLC
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

package streaming_writes

import (
	"os"
	"path"
	"strings"

	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
)

func (t *defaultMountCommonTest) TestCreateSymlinkForLocalFileReadFails() {
	// Create Symlink.
	symlink := path.Join(testDirPath, setup.GenerateRandomString(5))
	operations.CreateSymLink(t.filePath, symlink, t.T())
	_, err := t.f1.WriteAt([]byte(FileContents), 0)
	assert.NoError(t.T(), err)

	// Verify read link.
	operations.VerifyReadLink(t.filePath, symlink, t.T())

	// Reading file from symlink fails.
	_, err = os.ReadFile(symlink)
	assert.Error(t.T(), err)

	// Close the file and validate that the file is created on GCS.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, t.f1, testDirName,
		t.fileName, FileContents, t.T())
}

func (t *defaultMountCommonTest) TestReadSymlinkForDeletedLocalFileFails() {
	// Create Symlink.
	symlink := path.Join(testDirPath, setup.GenerateRandomString(5))
	operations.CreateSymLink(t.filePath, symlink, t.T())
	_, err := t.f1.WriteAt([]byte(FileContents), 0)
	assert.NoError(t.T(), err)

	// Verify read link.
	operations.VerifyReadLink(t.filePath, symlink, t.T())

	// Read the file from symlink fails
	_, err = os.ReadFile(symlink)
	assert.Error(t.T(), err)

	// Remove filePath and then close the fileHandle to avoid syncing to GCS.
	operations.RemoveFile(t.filePath)
	operations.CloseFileShouldNotThrowError(t.f1, t.T())
	ValidateObjectNotFoundErrOnGCS(ctx, storageClient, testDirName, t.fileName, t.T())

	// Reading symlink should fail.
	_, err = os.Stat(symlink)
	if err == nil || !strings.Contains(err.Error(), "no such file or directory") {
		t.T().Fatalf("Reading symlink for deleted local file did not fail.")
	}
}
