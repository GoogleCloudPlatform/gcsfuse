// Copyright 2020 Google Inc. All Rights Reserved.
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

	"github.com/googlecloudplatform/gcsfuse/v2/internal/fs/inode"
	. "github.com/jacobsa/ogletest"
)

func TestName(t *testing.T) {
	for _, bucketName := range []string{"", "bucketx"} {
		mountPoint := ""
		if bucketName != "" {
			mountPoint = bucketName + "/"
		}

		root := inode.NewRootName(bucketName) // ""
		ExpectTrue(root.IsBucketRoot())
		ExpectTrue(root.IsDir())
		ExpectFalse(root.IsFile())
		ExpectEq("", root.GcsObjectName())
		ExpectEq(mountPoint+"", root.LocalName())
		ExpectFalse(root.IsDirectChildOf(root))

		anotherRoot := inode.NewRootName("bucket-y") // ""
		ExpectFalse(root.IsDirectChildOf(anotherRoot))
		ExpectFalse(anotherRoot.IsDirectChildOf(root))

		foo := inode.NewDirName(root, "foo") // "foo"
		ExpectFalse(foo.IsBucketRoot())
		ExpectTrue(foo.IsDir())
		ExpectFalse(foo.IsFile())
		ExpectEq("foo/", foo.GcsObjectName())
		ExpectEq(mountPoint+"foo/", foo.LocalName())
		ExpectFalse(foo.IsDirectChildOf(foo))
		ExpectFalse(foo.IsDirectChildOf(anotherRoot))
		ExpectTrue(foo.IsDirectChildOf(root))

		bar := inode.NewDirName(foo, "bar") // "foo/bar"
		ExpectFalse(bar.IsBucketRoot())
		ExpectTrue(bar.IsDir())
		ExpectFalse(bar.IsFile())
		ExpectEq("foo/bar/", bar.GcsObjectName())
		ExpectEq(mountPoint+"foo/bar/", bar.LocalName())
		ExpectFalse(bar.IsDirectChildOf(bar))
		ExpectFalse(bar.IsDirectChildOf(root))
		ExpectTrue(bar.IsDirectChildOf(foo))
		ExpectFalse(foo.IsDirectChildOf(bar))

		baz := inode.NewFileName(root, "baz") // "baz"
		ExpectFalse(baz.IsBucketRoot())
		ExpectFalse(baz.IsDir())
		ExpectTrue(baz.IsFile())
		ExpectEq("baz", baz.GcsObjectName())
		ExpectEq(mountPoint+"baz", baz.LocalName())
		ExpectFalse(baz.IsDirectChildOf(foo))
		ExpectFalse(baz.IsDirectChildOf(bar))
		ExpectTrue(baz.IsDirectChildOf(root))

		qux := inode.NewFileName(bar, "qux") // "foo/bar/qux"
		ExpectFalse(qux.IsBucketRoot())
		ExpectFalse(qux.IsDir())
		ExpectTrue(qux.IsFile())
		ExpectEq("foo/bar/qux", qux.GcsObjectName())
		ExpectEq(mountPoint+"foo/bar/qux", qux.LocalName())
		ExpectFalse(qux.IsDirectChildOf(root))
		ExpectFalse(baz.IsDirectChildOf(baz))
		ExpectTrue(qux.IsDirectChildOf(bar))

		qux = inode.NewDescendantName(foo, "foo/bar/qux") // "foo/bar/qux"
		ExpectFalse(qux.IsBucketRoot())
		ExpectFalse(qux.IsDir())
		ExpectTrue(qux.IsFile())
		ExpectEq("foo/bar/qux", qux.GcsObjectName())
		ExpectEq(mountPoint+"foo/bar/qux", qux.LocalName())
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

	ExpectEq(1, count[root])
	ExpectEq(2, count[foo])
	_, ok := count[bar]
	ExpectFalse(ok)
}
