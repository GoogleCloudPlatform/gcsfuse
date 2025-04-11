// Copyright 2021 Google LLC
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
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/jacobsa/fuse/fuseops"
	. "github.com/jacobsa/ogletest"
	"golang.org/x/net/context"
)

func TestSymlink(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const uid = 123
const gid = 456

const fileInodeID = 17

type SymlinkTest struct {
	ctx context.Context
}

var _ SetUpInterface = &CoreTest{}
var _ TearDownInterface = &CoreTest{}

func init() { RegisterTestSuite(&SymlinkTest{}) }

func (t *SymlinkTest) SetupSubTest() {
	t.SetupTest()
}

func (t *SymlinkTest) SetupTest() {
	t.ctx = context.Background()
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *SymlinkTest) TestIsSymLinkWhenMetadataKeyIsPresent() {
	metadata := map[string]string{
		inode.SymlinkMetadataKey: "target",
	}
	m := gcs.MinObject{
		Name:     "test",
		Metadata: metadata,
	}

	AssertEq(true, inode.IsSymlink(&m))
}

func (t *SymlinkTest) TestIsSymLinkWhenMetadataKeyIsNotPresent() {
	m := gcs.MinObject{
		Name: "test",
	}

	AssertEq(false, inode.IsSymlink(&m))
}

func (t *SymlinkTest) TestIsSymLinkForNilObject() {
	AssertEq(false, inode.IsSymlink(nil))
}

func (t *SymlinkTest) TestSymlinkSize() {
	metadata := map[string]string{
		inode.SymlinkMetadataKey: "target",
	}
	m := gcs.MinObject{
		Name:     "test",
		Metadata: metadata,
	}
	name := inode.NewFileName(inode.NewRootName(""), "test")
	inode := inode.NewSymlinkInode(
		fileInodeID,
		name,
		&m,
		fuseops.InodeAttributes{
			Uid:  uid,
			Gid:  gid,
			Mode: 0o777,
		},
	)

	attrs, err := inode.Attributes(t.ctx)
	AssertEq(nil, err)
	AssertEq(attrs.Size, uint64(len("target")))
}
