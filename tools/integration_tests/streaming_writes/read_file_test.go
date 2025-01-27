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
	"github.com/stretchr/testify/assert"
)

func (t *defaultMountCommonTest) TestReadLocalFileFails() {
	// Write some content to local file.
	_, err := t.f1.WriteAt([]byte(FileContents), 0)
	assert.NoError(t.T(), err)

	// Reading the local file content fails.
	buf := make([]byte, len(FileContents))
	_, err = t.f1.ReadAt(buf, 0)
	assert.Error(t.T(), err)

	// Close the file and validate that the file is created on GCS.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, t.f1, testDirName, t.fileName, FileContents, t.T())
}

func (t *defaultMountCommonTest) TestReadBeforeFileIsFlushed() {
	testContent := "testContent"
	// Write data to file.
	operations.WriteAt(testContent, 0, t.f1, t.T())

	// Try to read the file.
	_, err := t.f1.Seek(0, 0)
	require.NoError(t.T(), err)
	buf := make([]byte, 10)
	_, err = t.f1.Read(buf)

	require.Error(t.T(), err, "input/output error")
	// Validate if correct content is uploaded to GCS after read error.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, t.f1, testDirName, t.fileName, testContent, t.T())
}

func (t *defaultMountCommonTest) TestReadAfterFlush() {
	testContent := "testContent"
	// Write data to file and flush.
	operations.WriteAt(testContent, 0, t.f1, t.T())
	CloseFileAndValidateContentFromGCS(ctx, storageClient, t.f1, testDirName, t.fileName, testContent, t.T())

	// Perform read and validate the contents.
	var err error
	t.f1, err = operations.OpenFileAsReadonly(path.Join(testDirPath, t.fileName))
	require.NoError(t.T(), err)
	buf := make([]byte, len(testContent))
	_, err = t.f1.Read(buf)

	require.NoError(t.T(), err)
	require.Equal(t.T(), string(buf), testContent)
}

func (t *defaultMountLocalFile) TestReadLocalFileFails() {
	// Write some content to local file.
	t.f1.WriteAt([]byte(FileContents), 0)

	// Reading the local file content fails.
	buf := make([]byte, len(FileContents))
	_, err := t.f1.ReadAt(buf, 0)
	assert.Error(t.T(), err)

	// Close the file and validate that the file is created on GCS.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, t.f1, testDirName, t.fileName, FileContents, t.T())
}
