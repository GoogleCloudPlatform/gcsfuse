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

	. "github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
)

func (t *StreamingWritesSuite) TestCreateSymlinkForLocalFileAndReadFromSymlink() {
	// Create Symlink.
	symlink := path.Join(testEnv.testDirPath, setup.GenerateRandomString(5))
	operations.CreateSymLink(t.filePath, symlink, t.T())
	_, err := t.f1.WriteAt([]byte(t.data), 0)
	assert.NoError(t.T(), err)
	// Verify read link.
	operations.VerifyReadLink(t.filePath, symlink, t.T())

	// Validate read file from symlink.
	symlink_fh := operations.OpenFile(symlink, t.T())
	defer operations.CloseFileShouldNotThrowError(t.T(), symlink_fh)
	t.validateReadCall(symlink_fh, t.data)

	// Close the file and validate that the file is created on GCS.
	CloseFileAndValidateContentFromGCS(testEnv.ctx, testEnv.storageClient, t.f1, t.dirName, t.fileName, t.data, t.T())
}

func (t *StreamingWritesSuite) TestReadingFromSymlinkForDeletedLocalFile() {
	// Create Symlink.
	symlink := path.Join(testEnv.testDirPath, setup.GenerateRandomString(5))
	operations.CreateSymLink(t.filePath, symlink, t.T())
	_, err := t.f1.WriteAt([]byte(t.data), 0)
	assert.NoError(t.T(), err)
	// Verify read link.
	operations.VerifyReadLink(t.filePath, symlink, t.T())

	// Validate read from symlink.
	symlink_fh := operations.OpenFile(symlink, t.T())
	defer operations.CloseFileShouldNotThrowError(t.T(), symlink_fh)
	t.validateReadCall(symlink_fh, t.data)

	// Remove filePath and then close the fileHandle to avoid syncing to GCS.
	operations.RemoveFile(t.filePath)
	operations.CloseFileShouldNotThrowError(t.T(), t.f1)
	ValidateObjectNotFoundErrOnGCS(testEnv.ctx, testEnv.storageClient, t.dirName, t.fileName, t.T())
	// Reading symlink should fail.
	_, err = os.Stat(symlink)
	assert.Error(t.T(), err)
}
