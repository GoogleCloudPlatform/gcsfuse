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
	"context"
	"os"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/jacobsa/fuse/fuseops"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
)

func TestSymlink(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type SymlinkTest struct {
	bucket *gcsx.SyncerBucket
}

var _ SetUpInterface = &CoreTest{}
var _ TearDownInterface = &CoreTest{}

func init() { RegisterTestSuite(&SymlinkTest{}) }

func (t *SymlinkTest) SetUp(ti *TestInfo) {
	bucket := gcsx.NewSyncerBucket(
		1,
		10, // ChunkTransferTimeoutSecs
		".gcsfuse_tmp/",
		fake.NewFakeBucket(timeutil.RealClock(), "some-bucket", gcs.BucketType{}),
	)
	t.bucket = &bucket
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

func (t *SymlinkTest) TestAttributes() {
	metadata := map[string]string{
		inode.SymlinkMetadataKey: "target",
	}
	m := &gcs.MinObject{
		Name:     "test",
		Metadata: metadata,
	}
	attrs := fuseops.InodeAttributes{
		Uid:  1001,
		Gid:  1002,
		Mode: 0777 | os.ModeSymlink,
	}
	name := inode.NewFileName(inode.NewRootName("some-bucket"), m.Name)
	s := inode.NewSymlinkInode(fuseops.InodeID(42), name, t.bucket, m, attrs)
	tests := []struct {
		name           string
		clobberedCheck bool
	}{
		{"WithClobberedCheckFalse", false},
		{"WithClobberedCheckTrue", true},
	}

	for _, tt := range tests {
		// Call Attributes
		extracted, err := s.Attributes(context.TODO(), tt.clobberedCheck)

		// Check expected values
		AssertEq(nil, err)
		ExpectEq(uint32(1), extracted.Nlink)
		ExpectEq(attrs.Uid, extracted.Uid)
		ExpectEq(attrs.Gid, extracted.Gid)
		ExpectEq(attrs.Mode, extracted.Mode)
	}
}

func (t *SymlinkTest) TestUpdateSize() {
	m := &gcs.MinObject{
		Name:           "test",
		Generation:     1,
		MetaGeneration: 2,
		Size:           100,
		Metadata:       map[string]string{inode.SymlinkMetadataKey: "target"},
	}
	attrs := fuseops.InodeAttributes{}
	name := inode.NewFileName(inode.NewRootName("some-bucket"), m.Name)
	s := inode.NewSymlinkInode(fuseops.InodeID(42), name, t.bucket, m, attrs)

	s.UpdateSize(200)

	AssertEq(uint64(200), s.SourceGeneration().Size)
}
