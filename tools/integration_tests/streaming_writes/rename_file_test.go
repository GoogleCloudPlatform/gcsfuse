// Copyright 2025 Google LLC
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

package streaming_writes

import (
	"path"

	. "github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/stretchr/testify/require"
)

func (t *StreamingWritesSuite) TestRenameBeforeFileIsFlushed() {
	operations.WriteWithoutClose(t.f1, t.data, t.T())
	operations.WriteWithoutClose(t.f1, t.data, t.T())
	operations.VerifyStatFile(t.filePath, int64(2*len(t.data)), FilePerms, t.T())
	err := t.f1.Sync()
	require.NoError(t.T(), err)

	newFile := "new" + t.fileName
	destDirPath := path.Join(testEnv.testDirPath, newFile)
	err = operations.RenameFile(t.filePath, destDirPath)

	// Validate that move didn't throw any error.
	require.NoError(t.T(), err)
	// Verify the new object contents.
	ValidateObjectContentsFromGCS(testEnv.ctx, testEnv.storageClient, t.dirName, newFile, t.data+t.data, t.T())
	require.NoError(t.T(), t.f1.Close())
	// Check if old object is deleted.
	ValidateObjectNotFoundErrOnGCS(testEnv.ctx, testEnv.storageClient, t.dirName, t.fileName, t.T())
}

func (t *StreamingWritesSuite) TestSyncAfterRenameSucceeds() {
	_, err := t.f1.WriteAt([]byte(t.data), 0)
	require.NoError(t.T(), err)
	operations.VerifyStatFile(t.filePath, int64(len(t.data)), FilePerms, t.T())
	err = t.f1.Sync()
	require.NoError(t.T(), err)
	newFile := "new" + t.fileName
	err = operations.RenameFile(t.filePath, path.Join(testEnv.testDirPath, newFile))
	require.NoError(t.T(), err)

	err = t.f1.Sync()

	// Verify that sync succeeds after rename.
	require.NoError(t.T(), err)
	// Verify the new object contents.
	ValidateObjectContentsFromGCS(testEnv.ctx, testEnv.storageClient, t.dirName, newFile, string(t.data), t.T())
	require.NoError(t.T(), t.f1.Close())
	// Check if old object is deleted.
	ValidateObjectNotFoundErrOnGCS(testEnv.ctx, testEnv.storageClient, t.dirName, t.fileName, t.T())
}
