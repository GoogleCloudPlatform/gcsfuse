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
	. "github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (t *defaultMountCommonTest) TestTruncate() {
	truncateSize := 2 * 1024 * 1024

	err := t.f1.Truncate(int64(truncateSize))

	assert.NoError(t.T(), err)
	data := make([]byte, truncateSize)
	// Close the file and validate that the file is created on GCS.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, t.f1, testDirName, t.fileName, string(data[:]), t.T())
}

func (t *defaultMountCommonTest) TestWriteAfterTruncate() {
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

			// Triggers writes after truncate.
			newData := []byte("hi")
			_, err = t.f1.WriteAt(newData, tc.offset)

			require.NoError(t.T(), err)
			data[tc.offset] = newData[0]
			data[tc.offset+1] = newData[1]
			// Close the file and validate that the file is created on GCS.
			CloseFileAndValidateContentFromGCS(ctx, storageClient, t.f1, testDirName, t.fileName, string(data[:]), t.T())
		})
	}

}

func (t *defaultMountCommonTest) TestWriteAndTruncate() {
	truncateSize := 20
	operations.WriteWithoutClose(t.f1, FileContents, t.T())

	err := t.f1.Truncate(int64(truncateSize))

	require.NoError(t.T(), err)
	data := make([]byte, 10)
	// Close the file and validate that the file is created on GCS.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, t.f1, testDirName, t.fileName, FileContents+string(data[:]), t.T())
}

func (t *defaultMountCommonTest) TestWriteTruncateWrite() {
	truncateSize := 30

	// Write
	operations.WriteWithoutClose(t.f1, FileContents, t.T())
	// Perform truncate
	err := t.f1.Truncate(int64(truncateSize))
	require.NoError(t.T(), err)
	// Write
	operations.WriteWithoutClose(t.f1, FileContents, t.T())

	data := make([]byte, 10)
	// Close the file and validate that the file is created on GCS.
	CloseFileAndValidateContentFromGCS(ctx, storageClient, t.f1, testDirName, t.fileName, FileContents+FileContents+string(data[:]), t.T())
}

func (t *defaultMountCommonTest) TestTruncateToLowerSizeAfterWrite() {
	// Write
	operations.WriteWithoutClose(t.f1, FileContents+FileContents, t.T())
	// Perform truncate
	err := t.f1.Truncate(int64(5))

	// Truncating to lower size after writes are not allowed.
	require.Error(t.T(), err)
}
