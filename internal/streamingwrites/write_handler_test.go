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

package streamingwrites

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSequentialWrite(t *testing.T) {
	testCases := []struct {
		name          string
		offset        int64
		totalSize     int64
		truncatedSize int64
		expected      bool
	}{
		// No pending truncate (truncatedSize is -1).
		{
			name:          "FirstWriteToNewFile",
			offset:        0,
			totalSize:     0,
			truncatedSize: -1,
			expected:      true,
		},
		{
			name:          "AtTotalSize",
			offset:        10,
			totalSize:     10,
			truncatedSize: -1,
			expected:      true,
		},
		{
			name:          "BeforeTotalSize",
			offset:        5,
			totalSize:     10,
			truncatedSize: -1,
			expected:      false,
		},
		{
			name:          "AfterTotalSize",
			offset:        15,
			totalSize:     10,
			truncatedSize: -1,
			expected:      false,
		},
		// Pending forward truncate: the file is smaller than truncatedSize.
		{
			name:          "AtTotalSizeWithPendingTruncate",
			offset:        10,
			totalSize:     10,
			truncatedSize: 20,
			expected:      true,
		},
		{
			name:          "AtPendingTruncatedSize",
			offset:        20,
			totalSize:     10,
			truncatedSize: 20,
			expected:      true,
		},
		{
			name:          "BeforeTotalSizeWithPendingTruncate",
			offset:        5,
			totalSize:     10,
			truncatedSize: 20,
			expected:      false,
		},
		{
			name:          "BetweenTotalAndPendingTruncatedSize",
			offset:        15,
			totalSize:     10,
			truncatedSize: 20,
			expected:      false,
		},
		{
			name:          "PastPendingTruncatedSize",
			offset:        25,
			totalSize:     10,
			truncatedSize: 20,
			expected:      false,
		},
		// The file has reached or passed truncatedSize.
		{
			name:          "AtTruncatedSizeWhenFileReachedIt",
			offset:        20,
			totalSize:     20,
			truncatedSize: 20,
			expected:      true,
		},
		{
			name:          "AtTruncatedSizeAfterWritingPastIt",
			offset:        20,
			totalSize:     25,
			truncatedSize: 20,
			expected:      false,
		},
		{
			name:          "AtTotalSizeAfterWritingPastTruncatedSize",
			offset:        25,
			totalSize:     25,
			truncatedSize: 20,
			expected:      true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := isSequentialWrite(tc.offset, tc.totalSize, tc.truncatedSize)

			assert.Equal(t, tc.expected, got)
		})
	}
}
