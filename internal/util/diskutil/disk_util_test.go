// Copyright 2026 Google LLC
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

package diskutil_test

import (
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/util/diskutil"
	"github.com/stretchr/testify/assert"
)

func TestGetSpeculativeFileSizeOnDisk(t *testing.T) {
	testcases := []struct {
		name            string
		fileContentSize uint64
		volumeBlockSize uint64
		expectedSize    uint64
	}{
		{
			name:            "Zero_Block_Size",
			fileContentSize: 100,
			volumeBlockSize: 0,
			expectedSize:    100,
		},
		{
			name:            "Block_Size_One",
			fileContentSize: 100,
			volumeBlockSize: 1,
			expectedSize:    100,
		},
		{
			name:            "Zero_File_Size",
			fileContentSize: 0,
			volumeBlockSize: 4096,
			expectedSize:    0,
		},
		{
			name:            "File_Size_Less_Than_Block_Size",
			fileContentSize: 1,
			volumeBlockSize: 4096,
			expectedSize:    4096,
		},
		{
			name:            "File_Size_Equal_To_Block_Size",
			fileContentSize: 4096,
			volumeBlockSize: 4096,
			expectedSize:    4096,
		},
		{
			name:            "File_Size_Greater_Than_Block_Size",
			fileContentSize: 4097,
			volumeBlockSize: 4096,
			expectedSize:    8192,
		},
		{
			name:            "File_Size_Much_Greater_Than_Block_Size",
			fileContentSize: 10000,
			volumeBlockSize: 4096,
			expectedSize:    12288,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			speculativeSize := diskutil.GetSpeculativeFileSizeOnDisk(tc.fileContentSize, tc.volumeBlockSize)

			assert.Equal(t, tc.expectedSize, speculativeSize)
		})
	}
}

func TestGetVolumeBlockSize_ProperDir(t *testing.T) {
	dir := t.TempDir()

	blockSize := diskutil.GetVolumeBlockSize(dir)

	// expect actual block-size (which is positive and power of 2) if directory exists and is proper.
	assert.True(t, blockSize > 0 && ((blockSize&(blockSize-1)) == 0), "Block-size of a directory should be a power of 2. Got: %d", blockSize)
}

func TestGetVolumeBlockSize_InvalidDir(t *testing.T) {
	dir := "/path/that/does/not/exist"
	expectedVolumeBlockSize := uint64(4096)

	blockSize := diskutil.GetVolumeBlockSize(dir)

	// expect default value if directory doesn't exist.
	assert.Equal(t, expectedVolumeBlockSize, blockSize)
}
