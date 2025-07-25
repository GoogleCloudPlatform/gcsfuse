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

package inode

import (
	"fmt"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"golang.org/x/net/context"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/jacobsa/fuse/fuseops"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
)

const ChunkTransferTimeoutSecs = 10

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
		ChunkTransferTimeoutSecs,
		".gcsfuse_tmp/",
		fake.NewFakeBucket(&t.clock, "bucketA", gcs.BucketType{}),
	)
	t.bm.buckets["bucketB"] = gcsx.NewSyncerBucket(
		1, // Append threshold
		ChunkTransferTimeoutSecs,
		".gcsfuse_tmp/",
		fake.NewFakeBucket(&t.clock, "bucketB", gcs.BucketType{}),
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
	name string, isMultibucketMount bool, _ metrics.MetricHandle) (sb gcsx.SyncerBucket, err error) {
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
		t.bm,
		metrics.NewNoopMetrics())

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

func (t *BaseDirTest) Attributes_ClobberedCheckTrue() {
	attrs, err := t.in.Attributes(t.ctx, true)

	AssertEq(nil, err)
	ExpectEq(uid, attrs.Uid)
	ExpectEq(gid, attrs.Gid)
	ExpectEq(dirMode|os.ModeDir, attrs.Mode)
}

func (t *BaseDirTest) Attributes_ClobberedCheckFalse() {
	attrs, err := t.in.Attributes(t.ctx, false)

	AssertEq(nil, err)
	ExpectEq(uid, attrs.Uid)
	ExpectEq(gid, attrs.Gid)
	ExpectEq(dirMode|os.ModeDir, attrs.Mode)
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
	ExpectEq(nil, result.MinObject)
	ExpectEq(metadata.ImplicitDirType, result.Type())

	result, err = t.in.LookUpChild(t.ctx, "bucketB")

	AssertEq(nil, err)
	AssertNe(nil, result)

	ExpectEq("bucketB", result.Bucket.Name())
	ExpectTrue(result.FullName.IsBucketRoot())
	ExpectEq("bucketB/", result.FullName.LocalName())
	ExpectEq("", result.FullName.GcsObjectName())
	ExpectEq(nil, result.MinObject)
	ExpectEq(metadata.ImplicitDirType, result.Type())
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

func (t *BaseDirTest) Test_ShouldInvalidateKernelListCache() {
	ttl := time.Second
	AssertEq(true, t.in.ShouldInvalidateKernelListCache(ttl))
}

func (t *BaseDirTest) Test_ShouldInvalidateKernelListCache_TtlExpired() {
	ttl := time.Second
	t.clock.AdvanceTime(10 * time.Second)

	AssertEq(true, t.in.ShouldInvalidateKernelListCache(ttl))
}

func (t *BaseDirTest) TestReadEntryCores() {
	cores, newTok, err := t.in.ReadEntryCores(t.ctx, "")

	// Should return ENOTSUP because listing is unsupported.
	ExpectEq(nil, cores)
	ExpectEq("", newTok)
	ExpectEq(syscall.ENOTSUP, err)
}
