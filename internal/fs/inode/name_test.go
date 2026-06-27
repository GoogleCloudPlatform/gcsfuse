// Copyright 2020 Google LLC
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

	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/inode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestName(t *testing.T) {
	for _, bucketName := range []string{"", "bucketx"} {
		mountPoint := ""
		if bucketName != "" {
			mountPoint = bucketName + "/"
		}

		root := inode.NewRootName(bucketName) // ""
		assert.True(t, root.IsBucketRoot())
		assert.True(t, root.IsDir())
		assert.False(t, root.IsFile())
		assert.Equal(t, "", root.GcsObjectName())
		assert.Equal(t, mountPoint+"", root.LocalName())
		assert.False(t, root.IsDirectChildOf(root))

		anotherRoot := inode.NewRootName("bucket-y") // ""
		assert.False(t, root.IsDirectChildOf(anotherRoot))
		assert.False(t, anotherRoot.IsDirectChildOf(root))

		foo := inode.NewDirName(root, "foo") // "foo"
		assert.False(t, foo.IsBucketRoot())
		assert.True(t, foo.IsDir())
		assert.False(t, foo.IsFile())
		assert.Equal(t, "foo/", foo.GcsObjectName())
		assert.Equal(t, mountPoint+"foo/", foo.LocalName())
		assert.False(t, foo.IsDirectChildOf(foo))
		assert.False(t, foo.IsDirectChildOf(anotherRoot))
		assert.True(t, foo.IsDirectChildOf(root))

		bar := inode.NewDirName(foo, "bar") // "foo/bar"
		assert.False(t, bar.IsBucketRoot())
		assert.True(t, bar.IsDir())
		assert.False(t, bar.IsFile())
		assert.Equal(t, "foo/bar/", bar.GcsObjectName())
		assert.Equal(t, mountPoint+"foo/bar/", bar.LocalName())
		assert.False(t, bar.IsDirectChildOf(bar))
		assert.False(t, bar.IsDirectChildOf(root))
		assert.True(t, bar.IsDirectChildOf(foo))
		assert.False(t, foo.IsDirectChildOf(bar))

		baz := inode.NewFileName(root, "baz") // "baz"
		assert.False(t, baz.IsBucketRoot())
		assert.False(t, baz.IsDir())
		assert.True(t, baz.IsFile())
		assert.Equal(t, "baz", baz.GcsObjectName())
		assert.Equal(t, mountPoint+"baz", baz.LocalName())
		assert.False(t, baz.IsDirectChildOf(foo))
		assert.False(t, baz.IsDirectChildOf(bar))
		assert.True(t, baz.IsDirectChildOf(root))

		qux := inode.NewFileName(bar, "qux") // "foo/bar/qux"
		assert.False(t, qux.IsBucketRoot())
		assert.False(t, qux.IsDir())
		assert.True(t, qux.IsFile())
		assert.Equal(t, "foo/bar/qux", qux.GcsObjectName())
		assert.Equal(t, mountPoint+"foo/bar/qux", qux.LocalName())
		assert.False(t, qux.IsDirectChildOf(root))
		assert.False(t, baz.IsDirectChildOf(baz))
		assert.True(t, qux.IsDirectChildOf(bar))

		qux = inode.NewDescendantName(foo, "foo/bar/qux") // "foo/bar/qux"
		assert.False(t, qux.IsBucketRoot())
		assert.False(t, qux.IsDir())
		assert.True(t, qux.IsFile())
		assert.Equal(t, "foo/bar/qux", qux.GcsObjectName())
		assert.Equal(t, mountPoint+"foo/bar/qux", qux.LocalName())
	}
}

func TestNameAsMapKey(t *testing.T) {
	root := inode.NewRootName("bucketx")
	foo := inode.NewDirName(root, "foo")
	foo2 := inode.NewDirName(root, "foo")
	bar := inode.NewDirName(root, "bar")

	count := make(map[inode.Name]int)
	count[root]++
	count[foo]++
	count[foo2]++

	assert.Equal(t, 1, count[root])
	assert.Equal(t, 2, count[foo])
	_, ok := count[bar]
	assert.False(t, ok)
}

func TestParentName(t *testing.T) {
	for _, bucketName := range []string{"", "bucketx"} {
		// Setup
		root := inode.NewRootName(bucketName) // ""
		foo := inode.NewDirName(root, "foo")  // "foo/"
		bar := inode.NewDirName(foo, "bar")   // "foo/bar/"
		baz := inode.NewFileName(root, "baz") // "baz"
		qux := inode.NewFileName(bar, "qux")  // "foo/bar/qux"
		// Test cases
		testCases := []struct {
			name               inode.Name
			expectedParentName inode.Name
		}{
			{name: foo, expectedParentName: root},
			{name: bar, expectedParentName: foo},
			{name: baz, expectedParentName: root},
			{name: qux, expectedParentName: bar},
		}

		for _, tc := range testCases {
			parent, err := tc.name.ParentName()

			assert.Nil(t, err)
			assert.Equal(t, tc.expectedParentName.GcsObjectName(), parent.GcsObjectName())
			assert.Equal(t, tc.expectedParentName.LocalName(), parent.LocalName())
		}
	}
}

func TestParentNameReturnsErrorOnBucketRoot(t *testing.T) {
	for _, bucketName := range []string{"", "bucketx"} {
		// Setup
		root := inode.NewRootName(bucketName) // ""

		// Call ParentName on bucket root
		_, err := root.ParentName()

		// Expect an error
		require.NotNil(t, err)
		assert.ErrorContains(t, err, "root has no parent")
	}
}
