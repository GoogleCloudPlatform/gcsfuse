// Copyright 2015 Google Inc. All Rights Reserved.
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
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/googlecloudplatform/gcsfuse/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/gcsproxy"
	"github.com/googlecloudplatform/gcsfuse/lease"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsfake"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
)

func TestFile(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const uid = 123
const gid = 456

const fileInodeID = 17
const fileInodeName = "foo/bar"
const fileMode os.FileMode = 0641

type FileTest struct {
	ctx    context.Context
	bucket gcs.Bucket
	leaser lease.FileLeaser
	clock  timeutil.SimulatedClock

	initialContents string
	backingObj      *gcs.Object

	in *inode.FileInode
}

var _ SetUpInterface = &FileTest{}
var _ TearDownInterface = &FileTest{}

func init() { RegisterTestSuite(&FileTest{}) }

func (t *FileTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
	t.clock.SetTime(time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local))
	t.leaser = lease.NewFileLeaser("", math.MaxInt32, math.MaxInt64)
	t.bucket = gcsfake.NewFakeBucket(&t.clock, "some_bucket")

	// Set up the backing object.
	var err error

	t.initialContents = "taco"
	t.backingObj, err = gcsutil.CreateObject(
		t.ctx,
		t.bucket,
		fileInodeName,
		[]byte(t.initialContents))

	AssertEq(nil, err)

	// Create the inode.
	t.in = inode.NewFileInode(
		fileInodeID,
		t.backingObj,
		fuseops.InodeAttributes{
			Uid:  uid,
			Gid:  gid,
			Mode: fileMode,
		},
		math.MaxUint64, // GCS chunk size
		t.bucket,
		t.leaser,
		gcsproxy.NewObjectSyncer(
			1, // Append threshold
			".gcsfuse_tmp/",
			t.bucket),
		&t.clock)

	t.in.Lock()
}

func (t *FileTest) TearDown() {
	t.in.Unlock()
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *FileTest) ID() {
	ExpectEq(fileInodeID, t.in.ID())
}

func (t *FileTest) Name() {
	ExpectEq(fileInodeName, t.in.Name())
}

func (t *FileTest) InitialSourceGeneration() {
	ExpectEq(t.backingObj.Generation, t.in.SourceGeneration())
}

func (t *FileTest) InitialAttributes() {
	attrs, err := t.in.Attributes(t.ctx)
	AssertEq(nil, err)

	ExpectEq(len(t.initialContents), attrs.Size)
	ExpectEq(1, attrs.Nlink)
	ExpectEq(uid, attrs.Uid)
	ExpectEq(gid, attrs.Gid)
	ExpectEq(fileMode, attrs.Mode)
	ExpectThat(attrs.Mtime, timeutil.TimeEq(t.backingObj.Updated))
}

func (t *FileTest) Read() {
	AssertEq("taco", t.initialContents)

	// Make several reads, checking the expected contents. We should never get an
	// EOF error, since fuseops.ReadFileOp is not supposed to see those.
	testCases := []struct {
		offset   int64
		size     int
		expected string
	}{
		{0, 1, "t"},
		{0, 2, "ta"},
		{0, 3, "tac"},
		{0, 4, "taco"},
		{0, 5, "taco"},

		{1, 1, "a"},
		{1, 2, "ac"},
		{1, 3, "aco"},
		{1, 4, "aco"},

		{3, 1, "o"},
		{3, 2, "o"},

		// Empty ranges
		{0, 0, ""},
		{3, 0, ""},
		{4, 0, ""},
		{4, 1, ""},
		{5, 0, ""},
		{5, 1, ""},
		{5, 2, ""},
	}

	for _, tc := range testCases {
		desc := fmt.Sprintf("offset: %d, size: %d", tc.offset, tc.size)

		data := make([]byte, tc.size)
		n, err := t.in.Read(t.ctx, data, tc.offset)
		data = data[:n]

		AssertEq(nil, err, "%s", desc)
		ExpectEq(tc.expected, string(data), "%s", desc)
	}
}

func (t *FileTest) Write() {
	var err error

	AssertEq("taco", t.initialContents)

	// Overwite a byte.
	err = t.in.Write(t.ctx, []byte("p"), 0)
	AssertEq(nil, err)

	// Add some data at the end.
	t.clock.AdvanceTime(time.Second)
	writeTime := t.clock.Now()

	err = t.in.Write(t.ctx, []byte("burrito"), 4)
	AssertEq(nil, err)

	t.clock.AdvanceTime(time.Second)

	// Read back the content.
	var buf [1024]byte
	n, err := t.in.Read(t.ctx, buf[:], 0)
	AssertEq(nil, err)
	ExpectEq("pacoburrito", string(buf[:n]))

	// Check attributes.
	attrs, err := t.in.Attributes(t.ctx)
	AssertEq(nil, err)

	ExpectEq(len("pacoburrito"), attrs.Size)
	ExpectThat(attrs.Mtime, timeutil.TimeEq(writeTime))
}

func (t *FileTest) Truncate() {
	var attrs fuseops.InodeAttributes
	var err error

	AssertEq("taco", t.initialContents)

	// Truncate downward.
	t.clock.AdvanceTime(time.Second)
	truncateTime := t.clock.Now()

	err = t.in.Truncate(t.ctx, 2)
	AssertEq(nil, err)

	t.clock.AdvanceTime(time.Second)

	// Read the contents.
	var buf [1024]byte
	n, err := t.in.Read(t.ctx, buf[:], 0)
	AssertEq(nil, err)
	ExpectEq("ta", string(buf[:n]))

	// Check attributes.
	attrs, err = t.in.Attributes(t.ctx)
	AssertEq(nil, err)

	ExpectEq(len("ta"), attrs.Size)
	ExpectThat(attrs.Mtime, timeutil.TimeEq(truncateTime))
}

func (t *FileTest) WriteThenSync() {
	var attrs fuseops.InodeAttributes
	var err error

	AssertEq("taco", t.initialContents)

	// Overwite a byte.
	err = t.in.Write(t.ctx, []byte("p"), 0)
	AssertEq(nil, err)

	t.clock.AdvanceTime(time.Second)

	// Sync.
	err = t.in.Sync(t.ctx)
	AssertEq(nil, err)

	// The generation should have advanced.
	ExpectLt(t.backingObj.Generation, t.in.SourceGeneration())

	// Stat the current object in the bucket.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name()}
	o, err := t.bucket.StatObject(t.ctx, statReq)

	AssertEq(nil, err)
	ExpectEq(t.in.SourceGeneration(), o.Generation)
	ExpectEq(len("paco"), o.Size)

	// Read the object's contents.
	contents, err := gcsutil.ReadObject(t.ctx, t.bucket, t.in.Name())

	AssertEq(nil, err)
	ExpectEq("paco", string(contents))

	// Check attributes.
	attrs, err = t.in.Attributes(t.ctx)
	AssertEq(nil, err)

	ExpectEq(len("paco"), attrs.Size)
	ExpectThat(attrs.Mtime, timeutil.TimeEq(o.Updated))
}

func (t *FileTest) AppendThenSync() {
	var attrs fuseops.InodeAttributes
	var err error

	AssertEq("taco", t.initialContents)

	// Append some data.
	err = t.in.Write(t.ctx, []byte("burrito"), int64(len("taco")))
	AssertEq(nil, err)

	t.clock.AdvanceTime(time.Second)

	// Sync.
	err = t.in.Sync(t.ctx)
	AssertEq(nil, err)

	// The generation should have advanced.
	ExpectLt(t.backingObj.Generation, t.in.SourceGeneration())

	// Stat the current object in the bucket.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name()}
	o, err := t.bucket.StatObject(t.ctx, statReq)

	AssertEq(nil, err)
	ExpectEq(t.in.SourceGeneration(), o.Generation)
	ExpectEq(len("tacoburrito"), o.Size)

	// Read the object's contents.
	contents, err := gcsutil.ReadObject(t.ctx, t.bucket, t.in.Name())

	AssertEq(nil, err)
	ExpectEq("tacoburrito", string(contents))

	// Check attributes.
	attrs, err = t.in.Attributes(t.ctx)
	AssertEq(nil, err)

	ExpectEq(len("tacoburrito"), attrs.Size)
	ExpectThat(attrs.Mtime, timeutil.TimeEq(o.Updated))
}

func (t *FileTest) TruncateDownwardThenSync() {
	var attrs fuseops.InodeAttributes
	var err error

	// Truncate downward.
	err = t.in.Truncate(t.ctx, 2)
	AssertEq(nil, err)

	t.clock.AdvanceTime(time.Second)

	// Sync.
	err = t.in.Sync(t.ctx)
	AssertEq(nil, err)

	// The generation should have advanced.
	ExpectLt(t.backingObj.Generation, t.in.SourceGeneration())

	// Stat the current object in the bucket.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name()}
	o, err := t.bucket.StatObject(t.ctx, statReq)

	AssertEq(nil, err)
	ExpectEq(t.in.SourceGeneration(), o.Generation)
	ExpectEq(2, o.Size)

	// Check attributes.
	attrs, err = t.in.Attributes(t.ctx)
	AssertEq(nil, err)

	ExpectEq(2, attrs.Size)
	ExpectThat(attrs.Mtime, timeutil.TimeEq(o.Updated))
}

func (t *FileTest) TruncateUpwardThenSync() {
	var attrs fuseops.InodeAttributes
	var err error

	AssertEq(4, len(t.initialContents))

	// Truncate upward.
	err = t.in.Truncate(t.ctx, 6)
	AssertEq(nil, err)

	t.clock.AdvanceTime(time.Second)

	// Sync.
	err = t.in.Sync(t.ctx)
	AssertEq(nil, err)

	// The generation should have advanced.
	ExpectLt(t.backingObj.Generation, t.in.SourceGeneration())

	// Stat the current object in the bucket.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name()}
	o, err := t.bucket.StatObject(t.ctx, statReq)

	AssertEq(nil, err)
	ExpectEq(t.in.SourceGeneration(), o.Generation)
	ExpectEq(6, o.Size)

	// Check attributes.
	attrs, err = t.in.Attributes(t.ctx)
	AssertEq(nil, err)

	ExpectEq(6, attrs.Size)
	ExpectThat(attrs.Mtime, timeutil.TimeEq(o.Updated))
}

func (t *FileTest) Sync_Clobbered() {
	var err error

	// Truncate downward.
	err = t.in.Truncate(t.ctx, 2)
	AssertEq(nil, err)

	// Clobber the backing object.
	newObj, err := gcsutil.CreateObject(t.ctx, t.bucket, t.in.Name(), []byte("burrito"))
	AssertEq(nil, err)

	// Sync. The call should succeed, but nothing should change.
	err = t.in.Sync(t.ctx)

	AssertEq(nil, err)
	ExpectEq(t.backingObj.Generation, t.in.SourceGeneration())

	// The object in the bucket should not have been changed.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name()}
	o, err := t.bucket.StatObject(t.ctx, statReq)

	AssertEq(nil, err)
	ExpectEq(newObj.Generation, o.Generation)
	ExpectEq(newObj.Size, o.Size)
}
