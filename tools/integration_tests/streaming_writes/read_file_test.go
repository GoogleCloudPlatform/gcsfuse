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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (t *StreamingWritesSuite) TestReadFileAfterSync() {
	// Write some content to the file.
	_, err := t.f1.WriteAt([]byte(t.data), 0)
	assert.NoError(t.T(), err)
	// Sync File to ensure buffers are flushed to GCS.
	operations.SyncFile(t.f1, t.T())

	t.validateReadCall(t.f1, t.data)

	// Close the file and validate that the file is created on GCS.
	CloseFileAndValidateContentFromGCS(testEnv.ctx, testEnv.storageClient, t.f1, testDirName, t.fileName, t.data, t.T())
}

func (t *StreamingWritesSuite) TestReadBeforeFileIsFlushed() {
	// Write data to file.
	operations.WriteAt(t.data, 0, t.f1, t.T())

	// Try to read the file.
	t.validateReadCall(t.f1, t.data)

	// Validate if correct content is uploaded to GCS after read error.
	CloseFileAndValidateContentFromGCS(testEnv.ctx, testEnv.storageClient, t.f1, testDirName, t.fileName, t.data, t.T())
}

func (t *StreamingWritesSuite) TestReadBeforeSyncThenWriteAgainAndRead() {
	// Write data to file.
	operations.WriteAt(t.data, 0, t.f1, t.T())

	t.validateReadCall(t.f1, t.data)

	operations.WriteAt(t.data, int64(len(t.data)), t.f1, t.T())
	t.validateReadCall(t.f1, t.data+t.data)
	// Validate if correct content is uploaded to GCS after read.
	CloseFileAndValidateContentFromGCS(testEnv.ctx, testEnv.storageClient, t.f1, testDirName, t.fileName, t.data+t.data, t.T())
}

func (t *StreamingWritesSuite) TestReadAfterFlush() {
	// Write data to file and flush.
	operations.WriteAt(t.data, 0, t.f1, t.T())
	CloseFileAndValidateContentFromGCS(testEnv.ctx, testEnv.storageClient, t.f1, testDirName, t.fileName, t.data, t.T())

	// Perform read and validate the contents.
	var err error
	t.f1, err = operations.OpenFileAsReadonly(path.Join(testEnv.testDirPath, t.fileName))
	require.NoError(t.T(), err)
	buf := make([]byte, len(t.data))
	_, err = t.f1.Read(buf)

	require.NoError(t.T(), err)
	require.Equal(t.T(), string(buf), t.data)
}
