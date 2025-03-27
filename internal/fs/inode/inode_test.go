package inode_test

import (
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/fs/inode"
	"github.com/stretchr/testify/assert"
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
			expected: 1, // change in object generation should always be treated like a new object.
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
