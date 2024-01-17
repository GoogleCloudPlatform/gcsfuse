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

package inode

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/storage/fake"
	"golang.org/x/net/context"

	"github.com/googlecloudplatform/gcsfuse/internal/gcsx"
	"github.com/jacobsa/fuse/fuseops"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
)

func TestBaseDir(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type BaseDirTest struct {
	ctx   context.Context
	clock timeutil.SimulatedClock
	bm    *fakeBucketManager
	in    DirInode
}

var _ SetUpInterface = &BaseDirTest{}
var _ TearDownInterface = &BaseDirTest{}

func init() { RegisterTestSuite(&BaseDirTest{}) }

func (t *BaseDirTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
	t.clock.SetTime(time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local))

	// Create a bucket manager for 2 buckets: bucketA and bucketB
	t.bm = &fakeBucketManager{
		buckets: make(map[string]gcsx.SyncerBucket),
	}
	t.bm.buckets["bucketA"] = gcsx.NewSyncerBucket(
		1, // Append threshold
		".gcsfuse_tmp/",
		fake.NewFakeBucket(&t.clock, "bucketA"),
	)
	t.bm.buckets["bucketB"] = gcsx.NewSyncerBucket(
		1, // Append threshold
		".gcsfuse_tmp/",
		fake.NewFakeBucket(&t.clock, "bucketB"),
	)

	// Create the inode. No implicit dirs by default.
	t.resetInode()
}

func (t *BaseDirTest) TearDown() {
	t.in.Unlock()
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type fakeBucketManager struct {
	buckets    map[string]gcsx.SyncerBucket
	setupTimes int
}

func (bm *fakeBucketManager) SetUpBucket(
	ctx context.Context,
	name string, isMultibucketMount bool) (sb gcsx.SyncerBucket, err error) {
	bm.setupTimes++

	var ok bool
	sb, ok = bm.buckets[name]
	if ok {
		return
	}
	err = fmt.Errorf("Cannot open bucket %q", name)
	return
}

func (bm *fakeBucketManager) ShutDown() {}

func (bm *fakeBucketManager) SetUpTimes() int {
	return bm.setupTimes
}

func (t *BaseDirTest) resetInode() {
	if t.in != nil {
		t.in.Unlock()
	}

	t.in = NewBaseDirInode(
		dirInodeID,
		NewRootName(""),
		fuseops.InodeAttributes{
			Uid:  uid,
			Gid:  gid,
			Mode: dirMode,
		},
		t.bm)

	t.in.Lock()
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *BaseDirTest) ID() {
	ExpectEq(dirInodeID, t.in.ID())
}

func (t *BaseDirTest) Name() {
	ExpectEq("", t.in.Name().LocalName())
}

func (t *BaseDirTest) LookupCount() {
	// Increment thrice. The count should now be three.
	t.in.IncrementLookupCount()
	t.in.IncrementLookupCount()
	t.in.IncrementLookupCount()

	// Decrementing twice shouldn't cause destruction. But one more should.
	AssertFalse(t.in.DecrementLookupCount(2))
	ExpectTrue(t.in.DecrementLookupCount(1))
}

func (t *BaseDirTest) Attributes() {
	attrs, err := t.in.Attributes(t.ctx)
	AssertEq(nil, err)
	ExpectEq(uid, attrs.Uid)
	ExpectEq(gid, attrs.Gid)
	ExpectEq(dirMode|os.ModeDir, attrs.Mode)
}

func (t *BaseDirTest) IsBaseDirInode() {
	ExpectEq(true, t.in.IsBaseDirInode())
}

func (t *BaseDirTest) LookUpChild_NonExistent() {
	result, err := t.in.LookUpChild(t.ctx, "missing_bucket")

	ExpectNe(nil, err)
	ExpectEq(nil, result)
	ExpectEq(1, t.bm.SetUpTimes())
}

func (t *BaseDirTest) LookUpChild_BucketFound() {
	result, err := t.in.LookUpChild(t.ctx, "bucketA")

	AssertEq(nil, err)
	AssertNe(nil, result)

	ExpectEq("bucketA", result.Bucket.Name())
	ExpectTrue(result.FullName.IsBucketRoot())
	ExpectEq("bucketA/", result.FullName.LocalName())
	ExpectEq("", result.FullName.GcsObjectName())
	ExpectEq(nil, result.Object)
	ExpectEq(ImplicitDirType, result.Type())

	result, err = t.in.LookUpChild(t.ctx, "bucketB")

	AssertEq(nil, err)
	AssertNe(nil, result)

	ExpectEq("bucketB", result.Bucket.Name())
	ExpectTrue(result.FullName.IsBucketRoot())
	ExpectEq("bucketB/", result.FullName.LocalName())
	ExpectEq("", result.FullName.GcsObjectName())
	ExpectEq(nil, result.Object)
	ExpectEq(ImplicitDirType, result.Type())
}

func (t *BaseDirTest) LookUpChild_BucketCached() {
	_, _ = t.in.LookUpChild(t.ctx, "bucketA")
	ExpectEq(1, t.bm.SetUpTimes())
	_, _ = t.in.LookUpChild(t.ctx, "bucketA")
	ExpectEq(1, t.bm.SetUpTimes())
	_, _ = t.in.LookUpChild(t.ctx, "bucketB")
	ExpectEq(2, t.bm.SetUpTimes())
	_, _ = t.in.LookUpChild(t.ctx, "bucketB")
	ExpectEq(2, t.bm.SetUpTimes())
	_, _ = t.in.LookUpChild(t.ctx, "missing_bucket")
	ExpectEq(3, t.bm.SetUpTimes())
}
