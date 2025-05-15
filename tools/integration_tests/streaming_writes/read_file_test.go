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

	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)



func (t *defaultMountCommonTest) TestReadFileAfterSync() {
	// Write some content to the file.
	operations.WriteAt(t.data, 0, t.f1, t.T())
	// Sync File to ensure buffers are flushed to GCS.
	operations.SyncFile(t.f1, t.T())
	buf := make([]byte, len(t.data))

	n, err := t.f1.Read(buf)

	require.NoError(t.T(), err)
	assert.Equal(t.T(), n, len(t.data))
	assert.Equal(t.T(), string(buf), t.data)
	// Close the file and validate that the file is created on GCS.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, t.f1, testDirName, t.fileName, t.data, t.T())
}

func (t *defaultMountCommonTest) TestReadBeforeFileIsFlushed() {
	// Write data to file.
	operations.WriteAt(t.data, 0, t.f1, t.T())
	// Try to read the file.
	buf := make([]byte, len(t.data))

	n, err := t.f1.Read(buf)

	require.NoError(t.T(), err)
	assert.Equal(t.T(), n, len(t.data))
	assert.Equal(t.T(), string(buf), t.data)
	// Validate if correct content is uploaded to GCS after read error.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, t.f1, testDirName, t.fileName, t.data, t.T())
}

func (t *defaultMountCommonTest) TestReadBeforeSyncThenWriteAgainAndRead() {
	// Write data to file.
	operations.WriteAt(t.data, 0, t.f1, t.T())
	buf := make([]byte, len(t.data))
	// Read the file.
	_, err := t.f1.Read(buf)
	require.NoError(t.T(), err)
	operations.WriteAt(t.data, int64(len(t.data)), t.f1, t.T())

	// Read the file again.
	n, err := t.f1.ReadAt(buf, int64(len(t.data)))

	require.NoError(t.T(), err)
	assert.Equal(t.T(), n, len(t.data))
	assert.Equal(t.T(), string(buf), t.data)
	// Validate if correct content is uploaded to GCS after read error.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, t.f1, testDirName, t.fileName, t.data+t.data, t.T())
}

func (t *defaultMountCommonTest) TestReadAfterFlush() {
	// Write data to file and flush.
	operations.WriteAt(t.data, 0, t.f1, t.T())
	CloseFileAndValidateContentFromGCS(ctx, storageClient, t.f1, testDirName, t.fileName, t.data, t.T())

	// Perform read and validate the contents.
	var err error
	t.f1, err = operations.OpenFileAsReadonly(path.Join(testDirPath, t.fileName))
	require.NoError(t.T(), err)
	buf := make([]byte, len(t.data))
	_, err = t.f1.Read(buf)

	require.NoError(t.T(), err)
	require.Equal(t.T(), string(buf), t.data)
}
