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

	"github.com/googlecloudplatform/gcsfuse/internal/fs/inode"
	. "github.com/jacobsa/ogletest"
)

func TestName(t *testing.T) {
	root := inode.NewRootName()
	ExpectTrue(root.IsDir())
	ExpectFalse(root.IsFile())
	ExpectEq("", root.GcsObjectName())
	ExpectEq("", root.LocalName())

	foo := inode.NewDirName(root, "foo")
	ExpectTrue(foo.IsDir())
	ExpectFalse(foo.IsFile())
	ExpectEq("foo/", foo.GcsObjectName())
	ExpectEq("foo/", foo.LocalName())

	bar := inode.NewDirName(foo, "bar")
	ExpectTrue(bar.IsDir())
	ExpectFalse(bar.IsFile())
	ExpectEq("foo/bar/", bar.GcsObjectName())
	ExpectEq("foo/bar/", bar.LocalName())

	baz := inode.NewFileName(root, "baz")
	ExpectFalse(baz.IsDir())
	ExpectTrue(baz.IsFile())
	ExpectEq("baz", baz.GcsObjectName())
	ExpectEq("baz", baz.LocalName())

	qux := inode.NewFileName(bar, "qux")
	ExpectFalse(qux.IsDir())
	ExpectTrue(qux.IsFile())
	ExpectEq("foo/bar/qux", qux.GcsObjectName())
	ExpectEq("foo/bar/qux", qux.LocalName())
}

func TestNameAsMapKey(t *testing.T) {
	root := inode.NewRootName()
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
