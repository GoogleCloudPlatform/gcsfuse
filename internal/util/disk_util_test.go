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

package util

import (
	"testing"
)

func TestGetSpeculativeFileSizeOnDisk(t *testing.T) {
	tests := []struct {
		name            string
		fileContentSize uint64
		volumeBlockSize uint64
		expectedSize    uint64
	}{
		{
			name:            "Zero Block Size",
			fileContentSize: 100,
			volumeBlockSize: 0,
			expectedSize:    0,
		},
		{
			name:            "Zero File Size",
			fileContentSize: 0,
			volumeBlockSize: 4096,
			expectedSize:    0,
		},
		{
			name:            "File Size Less Than Block Size",
			fileContentSize: 1,
			volumeBlockSize: 4096,
			expectedSize:    4096,
		},
		{
			name:            "File Size Equal To Block Size",
			fileContentSize: 4096,
			volumeBlockSize: 4096,
			expectedSize:    4096,
		},
		{
			name:            "File Size Greater Than Block Size",
			fileContentSize: 4097,
			volumeBlockSize: 4096,
			expectedSize:    8192,
		},
		{
			name:            "File Size Much Greater Than Block Size",
			fileContentSize: 10000,
			volumeBlockSize: 4096,
			expectedSize:    12288,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actualSize := GetSpeculativeFileSizeOnDisk(tc.fileContentSize, tc.volumeBlockSize)
			if actualSize != tc.expectedSize {
				t.Errorf("GetSpeculativeFileSizeOnDisk(%d, %d) = %d; expected %d", tc.fileContentSize, tc.volumeBlockSize, actualSize, tc.expectedSize)
			}
		})
	}
}

func TestGetVolumeBlockSize(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	blockSize, err := GetVolumeBlockSize(tempDir)
	if err != nil {
		t.Fatalf("GetVolumeBlockSize failed: %v", err)
	}

	// Assuming a standard block size like 4096 for most modern filesystems
	// This might fail on specific edge-case filesystems, but should work for common CI environments
	if blockSize <= 0 {
		t.Errorf("Expected positive block size, got: %d", blockSize)
	}
}
