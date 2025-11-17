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

package inode_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vipnydav/gcsfuse/v3/internal/fs/inode"
)

func TestGenerationCompare(t *testing.T) {
	testCases := []struct {
		name     string
		latest   inode.Generation
		current  inode.Generation
		expected int
	}{
		{
			name:     "latest.Object > current.Object",
			latest:   inode.Generation{Object: 2, Metadata: 1, Size: 100},
			current:  inode.Generation{Object: 1, Metadata: 1, Size: 100},
			expected: 1,
		},
		{
			name:     "latest.Object < current.Object",
			latest:   inode.Generation{Object: 1, Metadata: 1, Size: 100},
			current:  inode.Generation{Object: 2, Metadata: 1, Size: 100},
			expected: -1,
		},
		{
			name:     "latest.Object == current.Object, latest.Metadata < current.Metadata",
			latest:   inode.Generation{Object: 1, Metadata: 1, Size: 100},
			current:  inode.Generation{Object: 1, Metadata: 2, Size: 100},
			expected: -1,
		},
		{
			name:     "latest.Object == current.Object, latest.Metadata > current.Metadata",
			latest:   inode.Generation{Object: 1, Metadata: 2, Size: 100},
			current:  inode.Generation{Object: 1, Metadata: 1, Size: 100},
			expected: 1,
		},
		{
			name:     "latest.Object == current.Object, latest.Metadata < current.Metadata",
			latest:   inode.Generation{Object: 1, Metadata: 1, Size: 100},
			current:  inode.Generation{Object: 1, Metadata: 2, Size: 100},
			expected: -1,
		},
		{
			name:     "latest.Object == current.Object, latest.Metadata == current.Metadata, latest.Size > current.Size",
			latest:   inode.Generation{Object: 1, Metadata: 1, Size: 200},
			current:  inode.Generation{Object: 1, Metadata: 1, Size: 100},
			expected: 1,
		},
		{
			name:     "latest.Object == current.Object, latest.Metadata == current.Metadata, latest.Size < current.Size",
			latest:   inode.Generation{Object: 1, Metadata: 1, Size: 100},
			current:  inode.Generation{Object: 1, Metadata: 1, Size: 200},
			expected: 0,
		},
		{
			name:     "latest.Object == current.Object, latest.Metadata == current.Metadata, latest.Size == current.Size",
			latest:   inode.Generation{Object: 1, Metadata: 1, Size: 100},
			current:  inode.Generation{Object: 1, Metadata: 1, Size: 100},
			expected: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.latest.Compare(tc.current)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
