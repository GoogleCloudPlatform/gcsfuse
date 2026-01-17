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

	. "github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (t *StreamingWritesSuite) TestTruncate() {
	truncateSize := 2 * 1024 * 1024

	err := t.f1.Truncate(int64(truncateSize))

	assert.NoError(t.T(), err)
	data := make([]byte, truncateSize)
	// Verify that GCSFuse is returning correct file size before the file is uploaded.
	operations.VerifyStatFile(t.filePath, int64(truncateSize), FilePerms, t.T())
	// Close the file and validate that the file is created on GCS.
	CloseFileAndValidateContentFromGCS(testEnv.ctx, testEnv.storageClient, t.f1, t.dirName, t.fileName, string(data[:]), t.T())
}

func (t *StreamingWritesSuite) TestTruncateNegative() {
	err := t.f1.Truncate(-1)

	assert.Error(t.T(), err)
}

func (t *StreamingWritesSuite) TestWriteAfterTruncate() {
	truncateSize := 10

	testCases := []struct {
		name     string
		offset   int64
		fileSize int64
	}{
		{
			name:     "ZeroOffset",
			offset:   0,
			fileSize: 10,
		},
		{
			name:     "RandomOffset",
			offset:   5,
			fileSize: 10,
		},
		{
			name:     "Append",
			offset:   10,
			fileSize: 12,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			data := make([]byte, tc.fileSize)
			// Perform truncate.
			err := t.f1.Truncate(int64(truncateSize))
			require.NoError(t.T(), err)
			operations.VerifyStatFile(t.filePath, int64(truncateSize), FilePerms, t.T())

			// Triggers writes after truncate.
			newData := []byte("hi")
			_, err = t.f1.WriteAt(newData, tc.offset)

			require.NoError(t.T(), err)
			// Verify that GCSFuse is returning correct file size before the file is uploaded.
			operations.VerifyStatFile(t.filePath, tc.fileSize, FilePerms, t.T())
			data[tc.offset] = newData[0]
			data[tc.offset+1] = newData[1]
			// Close the file and validate that the file is created on GCS.
			CloseFileAndValidateContentFromGCS(testEnv.ctx, testEnv.storageClient, t.f1, t.dirName, t.fileName, string(data[:]), t.T())
		})
	}

}

func (t *StreamingWritesSuite) TestWriteAndTruncate() {
	testCases := []struct {
		name            string
		intitialContent string
		truncateSize    int64
		finalContent    string
	}{
		{
			name:            "WriteTruncateToUpper",
			intitialContent: "foobar",
			truncateSize:    9,
			finalContent:    "foobar\x00\x00\x00",
		},
		{
			name:            "WriteTruncateToLower",
			intitialContent: "foobar",
			truncateSize:    3,
			finalContent:    "foo",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func() {
			// Write
			operations.WriteWithoutClose(t.f1, tc.intitialContent, t.T())
			operations.VerifyStatFile(t.filePath, int64(len(tc.intitialContent)), FilePerms, t.T())

			// Perform truncate
			err := t.f1.Truncate(tc.truncateSize)

			require.NoError(t.T(), err)
			operations.VerifyStatFile(t.filePath, tc.truncateSize, FilePerms, t.T())
			// Close the file and validate that the file is created on GCS.
			CloseFileAndValidateContentFromGCS(testEnv.ctx, testEnv.storageClient, t.f1, t.dirName, t.fileName, tc.finalContent, t.T())
		})
	}
}

func (t *StreamingWritesSuite) TestWriteTruncateWrite() {
	testCases := []struct {
		name           string
		truncateSize   int64
		initialContent string
		writeContent   string
		finalContent   string
	}{
		{
			name:           "WriteTruncateToUpperWrite",
			truncateSize:   12,
			initialContent: "foobar",
			writeContent:   "foo",
			finalContent:   "foobarfoo\x00\x00\x00",
		},
		{
			name:           "WriteTruncateToLowerWrite",
			truncateSize:   3,
			initialContent: "foobar",
			writeContent:   "foo",
			finalContent:   "foo\x00\x00\x00foo",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func() {
			// Write
			operations.WriteWithoutClose(t.f1, tc.initialContent, t.T())
			operations.VerifyStatFile(t.filePath, int64(len(tc.initialContent)), FilePerms, t.T())
			// Perform truncate
			// Note: truncate operation does not change the position of the file pointer of file handle.
			err := t.f1.Truncate(tc.truncateSize)
			require.NoError(t.T(), err)
			operations.VerifyStatFile(t.filePath, tc.truncateSize, FilePerms, t.T())

			// Write again
			operations.WriteWithoutClose(t.f1, tc.writeContent, t.T())

			operations.VerifyStatFile(t.filePath, int64(len(tc.finalContent)), FilePerms, t.T())
			// Close the file and validate that the file is created on GCS.
			CloseFileAndValidateContentFromGCS(testEnv.ctx, testEnv.storageClient, t.f1, t.dirName, t.fileName, tc.finalContent, t.T())
		})
	}
}

func (t *StreamingWritesSuite) TestTruncateDownAndDeleteFile() {
	// Write
	operations.WriteWithoutClose(t.f1, "foobar", t.T())
	operations.VerifyStatFile(t.filePath, int64(len("foobar")), FilePerms, t.T())
	// Perform truncate
	err := t.f1.Truncate(3)
	require.NoError(t.T(), err)
	operations.VerifyStatFile(t.filePath, 3, FilePerms, t.T())
	ValidateObjectContentsFromGCS(testEnv.ctx, testEnv.storageClient, t.dirName, t.fileName, "foobar", t.T())

	err = os.Remove(t.filePath)

	require.NoError(t.T(), err)
	ValidateObjectNotFoundErrOnGCS(testEnv.ctx, testEnv.storageClient, t.dirName, t.fileName, t.T())
}
