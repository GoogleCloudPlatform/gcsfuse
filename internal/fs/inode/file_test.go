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

package inode

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/storageutil"
	"github.com/jacobsa/syncutil"
	"golang.org/x/net/context"

	"github.com/googlecloudplatform/gcsfuse/internal/contentcache"
	"github.com/googlecloudplatform/gcsfuse/internal/gcsx"
	"github.com/jacobsa/fuse/fuseops"
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
const fileName = "foo/bar"
const fileMode os.FileMode = 0641
const Delta = 30 * time.Minute

type FileTest struct {
	ctx    context.Context
	bucket gcs.Bucket
	clock  timeutil.SimulatedClock

	initialContents string
	backingObj      *gcs.Object

	in *FileInode
}

var _ SetUpInterface = &FileTest{}
var _ TearDownInterface = &FileTest{}

func init() { RegisterTestSuite(&FileTest{}) }

func (t *FileTest) SetUp(ti *TestInfo) {
	// Enabling invariant check for all tests.
	syncutil.EnableInvariantChecking()
	t.ctx = ti.Ctx
	t.clock.SetTime(time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local))
	t.bucket = fake.NewFakeBucket(&t.clock, "some_bucket")

	// Set up the backing object.
	var err error

	t.initialContents = "taco"
	t.backingObj, err = storageutil.CreateObject(
		t.ctx,
		t.bucket,
		fileName,
		[]byte(t.initialContents))

	AssertEq(nil, err)

	// Create the inode.
	t.createInode()
}

func (t *FileTest) TearDown() {
	t.in.Unlock()
}

func (t *FileTest) createInode() {
	t.createInodeWithLocalParam(fileName, false)
}
func (t *FileTest) createInodeWithLocalParam(fileName string, local bool) {
	if t.in != nil {
		t.in.Unlock()
	}

	name := NewFileName(
		NewRootName(""),
		fileName,
	)
	syncerBucket := gcsx.NewSyncerBucket(
		1, // Append threshold
		".gcsfuse_tmp/",
		t.bucket)

	if local {
		t.backingObj = nil
	}
	t.in = NewFileInode(
		fileInodeID,
		name,
		t.backingObj,
		fuseops.InodeAttributes{
			Uid:  uid,
			Gid:  gid,
			Mode: fileMode,
		},
		&syncerBucket,
		false, // localFileCache
		contentcache.New("", &t.clock),
		&t.clock,
		local)

	t.in.Lock()
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *FileTest) ID() {
	ExpectEq(fileInodeID, t.in.ID())
}

func (t *FileTest) Name() {
	ExpectEq(fileName, t.in.Name().GcsObjectName())
}

func (t *FileTest) InitialSourceGeneration() {
	sg := t.in.SourceGeneration()
	ExpectEq(t.backingObj.Generation, sg.Object)
	ExpectEq(t.backingObj.MetaGeneration, sg.Metadata)
}

func (t *FileTest) InitialAttributes() {
	attrs, err := t.in.Attributes(t.ctx)
	AssertEq(nil, err)

	ExpectEq(len(t.initialContents), attrs.Size)
	ExpectEq(1, attrs.Nlink)
	ExpectEq(uid, attrs.Uid)
	ExpectEq(gid, attrs.Gid)
	ExpectEq(fileMode, attrs.Mode)
	ExpectThat(attrs.Atime, timeutil.TimeEq(t.backingObj.Updated))
	ExpectThat(attrs.Ctime, timeutil.TimeEq(t.backingObj.Updated))
	ExpectThat(attrs.Mtime, timeutil.TimeEq(t.backingObj.Updated))
}

func (t *FileTest) InitialAttributes_MtimeFromObjectMetadata_Gcsfuse() {
	// Set up an explicit mtime on the backing object and re-create the inode.
	if t.backingObj.Metadata == nil {
		t.backingObj.Metadata = make(map[string]string)
	}

	mtime := time.Now().Add(123*time.Second).UTC().AddDate(0, 0, 0)
	t.backingObj.Metadata["gcsfuse_mtime"] = mtime.Format(time.RFC3339Nano)

	t.createInode()

	// Ask it for its attributes.
	attrs, err := t.in.Attributes(t.ctx)
	AssertEq(nil, err)

	ExpectThat(attrs.Mtime, timeutil.TimeEq(mtime))
}

func (t *FileTest) InitialAttributes_MtimeFromObjectMetadata_Gsutil() {
	// Set up an explicit mtime on the backing object and re-create the inode.
	if t.backingObj.Metadata == nil {
		t.backingObj.Metadata = make(map[string]string)
	}

	mtime := time.Now().Add(123*time.Second).UTC().AddDate(0, 0, 0).Round(time.Second)
	t.backingObj.Metadata["goog-reserved-file-mtime"] = strconv.FormatInt(mtime.Unix(), 10)

	t.createInode()

	// Ask it for its attributes.
	attrs, err := t.in.Attributes(t.ctx)
	AssertEq(nil, err)

	ExpectThat(attrs.Mtime.UTC(), timeutil.TimeEq(mtime))
}

func (t *FileTest) InitialAttributes_MtimeFromObjectMetadata_GcsfuseOutranksGsutil() {
	// Set up an explicit mtime on the backing object and re-create the inode.
	if t.backingObj.Metadata == nil {
		t.backingObj.Metadata = make(map[string]string)
	}

	gsutilMtime := time.Now().Add(123*time.Second).UTC().AddDate(0, 0, 0).Round(time.Second)
	t.backingObj.Metadata["goog-reserved-file-mtime"] = strconv.FormatInt(gsutilMtime.Unix(), 10)

	canonicalMtime := time.Now().Add(456*time.Second).UTC().AddDate(0, 0, 0)
	t.backingObj.Metadata["gcsfuse_mtime"] = canonicalMtime.Format(time.RFC3339Nano)

	t.createInode()

	// Ask it for its attributes.
	attrs, err := t.in.Attributes(t.ctx)
	AssertEq(nil, err)

	ExpectThat(attrs.Mtime, timeutil.TimeEq(canonicalMtime))
}

func (t *FileTest) Read() {
	AssertEq("taco", t.initialContents)

	// Make several reads, checking the expected contents.
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

		// Ignore EOF.
		if err == io.EOF {
			err = nil
		}

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

	if err == io.EOF {
		err = nil
	}

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

	if err == io.EOF {
		err = nil
	}

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
	t.clock.AdvanceTime(time.Second)
	writeTime := t.clock.Now()

	err = t.in.Write(t.ctx, []byte("p"), 0)
	AssertEq(nil, err)

	t.clock.AdvanceTime(time.Second)

	// Sync.
	err = t.in.Sync(t.ctx)
	AssertEq(nil, err)

	// The generation should have advanced.
	ExpectLt(t.backingObj.Generation, t.in.SourceGeneration().Object)

	// Stat the current object in the bucket.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
	o, err := t.bucket.StatObject(t.ctx, statReq)

	AssertEq(nil, err)
	ExpectEq(t.in.SourceGeneration().Object, o.Generation)
	ExpectEq(t.in.SourceGeneration().Metadata, o.MetaGeneration)
	ExpectEq(len("paco"), o.Size)
	ExpectEq(
		writeTime.UTC().Format(time.RFC3339Nano),
		o.Metadata["gcsfuse_mtime"])

	// Read the object's contents.
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, t.in.Name().GcsObjectName())

	AssertEq(nil, err)
	ExpectEq("paco", string(contents))

	// Check attributes.
	attrs, err = t.in.Attributes(t.ctx)
	AssertEq(nil, err)

	ExpectEq(len("paco"), attrs.Size)
	ExpectThat(attrs.Mtime, timeutil.TimeEq(writeTime.UTC()))
}

func (t *FileTest) WriteToLocalFileThenSync() {
	var attrs fuseops.InodeAttributes
	var err error
	// Create a local file inode.
	t.createInodeWithLocalParam("test", true)
	// Create a temp file for the local inode created above.
	err = t.in.CreateEmptyTempFile()
	AssertEq(nil, err)
	// Write some content to temp file.
	t.clock.AdvanceTime(time.Second)
	writeTime := t.clock.Now()
	err = t.in.Write(t.ctx, []byte("tacos"), 0)
	AssertEq(nil, err)
	t.clock.AdvanceTime(time.Second)

	// Sync.
	err = t.in.Sync(t.ctx)

	AssertEq(nil, err)
	// Verify that fileInode is no more local
	AssertFalse(t.in.IsLocal())
	// Stat the current object in the bucket.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
	o, err := t.bucket.StatObject(t.ctx, statReq)
	AssertEq(nil, err)
	ExpectEq(t.in.SourceGeneration().Object, o.Generation)
	ExpectEq(t.in.SourceGeneration().Metadata, o.MetaGeneration)
	ExpectEq(len("tacos"), o.Size)
	ExpectEq(
		writeTime.UTC().Format(time.RFC3339Nano),
		o.Metadata["gcsfuse_mtime"])
	// Read the object's contents.
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, t.in.Name().GcsObjectName())
	AssertEq(nil, err)
	ExpectEq("tacos", string(contents))
	// Check attributes.
	attrs, err = t.in.Attributes(t.ctx)
	AssertEq(nil, err)
	ExpectEq(len("tacos"), attrs.Size)
	ExpectThat(attrs.Mtime, timeutil.TimeEq(writeTime.UTC()))
}

func (t *FileTest) SyncEmptyLocalFile() {
	var attrs fuseops.InodeAttributes
	var err error
	// Create a local file inode.
	t.createInodeWithLocalParam("test", true)
	creationTime := t.clock.Now()
	// Create a temp file for the local inode created above.
	err = t.in.CreateEmptyTempFile()
	AssertEq(nil, err)

	// Sync.
	err = t.in.Sync(t.ctx)

	AssertEq(nil, err)
	// Verify that fileInode is no more local
	AssertFalse(t.in.IsLocal())
	// Stat the current object in the bucket.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
	o, err := t.bucket.StatObject(t.ctx, statReq)
	AssertEq(nil, err)
	ExpectEq(t.in.SourceGeneration().Object, o.Generation)
	ExpectEq(t.in.SourceGeneration().Metadata, o.MetaGeneration)
	ExpectEq(0, o.Size)
	// Validate the mtime.
	mtimeInBucket, ok := o.Metadata["gcsfuse_mtime"]
	AssertTrue(ok)
	mtime, _ := time.Parse(time.RFC3339Nano, mtimeInBucket)
	ExpectThat(mtime, timeutil.TimeNear(creationTime, Delta))
	// Read the object's contents.
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, t.in.Name().GcsObjectName())
	AssertEq(nil, err)
	ExpectEq("", string(contents))
	// Check attributes.
	attrs, err = t.in.Attributes(t.ctx)
	AssertEq(nil, err)
	ExpectEq(0, attrs.Size)
}

func (t *FileTest) AppendThenSync() {
	var attrs fuseops.InodeAttributes
	var err error

	AssertEq("taco", t.initialContents)

	// Append some data.
	t.clock.AdvanceTime(time.Second)
	writeTime := t.clock.Now()

	err = t.in.Write(t.ctx, []byte("burrito"), int64(len("taco")))
	AssertEq(nil, err)

	t.clock.AdvanceTime(time.Second)

	// Sync.
	err = t.in.Sync(t.ctx)
	AssertEq(nil, err)

	// The generation should have advanced.
	ExpectLt(t.backingObj.Generation, t.in.SourceGeneration().Object)

	// Stat the current object in the bucket.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
	o, err := t.bucket.StatObject(t.ctx, statReq)

	AssertEq(nil, err)
	ExpectEq(t.in.SourceGeneration().Object, o.Generation)
	ExpectEq(t.in.SourceGeneration().Metadata, o.MetaGeneration)
	ExpectEq(len("tacoburrito"), o.Size)
	ExpectEq(
		writeTime.UTC().Format(time.RFC3339Nano),
		o.Metadata["gcsfuse_mtime"])

	// Read the object's contents.
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, t.in.Name().GcsObjectName())

	AssertEq(nil, err)
	ExpectEq("tacoburrito", string(contents))

	// Check attributes.
	attrs, err = t.in.Attributes(t.ctx)
	AssertEq(nil, err)

	ExpectEq(len("tacoburrito"), attrs.Size)
	ExpectThat(attrs.Mtime, timeutil.TimeEq(writeTime.UTC()))
}

func (t *FileTest) TruncateDownwardThenSync() {
	var attrs fuseops.InodeAttributes
	var err error

	// Truncate downward.
	t.clock.AdvanceTime(time.Second)
	truncateTime := t.clock.Now()

	err = t.in.Truncate(t.ctx, 2)
	AssertEq(nil, err)

	t.clock.AdvanceTime(time.Second)

	// Sync.
	err = t.in.Sync(t.ctx)
	AssertEq(nil, err)

	// The generation should have advanced.
	ExpectLt(t.backingObj.Generation, t.in.SourceGeneration().Object)

	// Stat the current object in the bucket.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
	o, err := t.bucket.StatObject(t.ctx, statReq)

	AssertEq(nil, err)
	ExpectEq(t.in.SourceGeneration().Object, o.Generation)
	ExpectEq(t.in.SourceGeneration().Metadata, o.MetaGeneration)
	ExpectEq(2, o.Size)
	ExpectEq(
		truncateTime.UTC().Format(time.RFC3339Nano),
		o.Metadata["gcsfuse_mtime"])

	// Check attributes.
	attrs, err = t.in.Attributes(t.ctx)
	AssertEq(nil, err)

	ExpectEq(2, attrs.Size)
	ExpectThat(attrs.Mtime, timeutil.TimeEq(truncateTime.UTC()))
}

func (t *FileTest) TruncateUpwardThenSync() {
	var attrs fuseops.InodeAttributes
	var err error

	AssertEq(4, len(t.initialContents))

	// Truncate upward.
	t.clock.AdvanceTime(time.Second)
	truncateTime := t.clock.Now()

	err = t.in.Truncate(t.ctx, 6)
	AssertEq(nil, err)

	t.clock.AdvanceTime(time.Second)

	// Sync.
	err = t.in.Sync(t.ctx)
	AssertEq(nil, err)

	// The generation should have advanced.
	ExpectLt(t.backingObj.Generation, t.in.SourceGeneration().Object)

	// Stat the current object in the bucket.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
	o, err := t.bucket.StatObject(t.ctx, statReq)
	ExpectEq(
		truncateTime.UTC().Format(time.RFC3339Nano),
		o.Metadata["gcsfuse_mtime"])

	AssertEq(nil, err)
	ExpectEq(t.in.SourceGeneration().Object, o.Generation)
	ExpectEq(t.in.SourceGeneration().Metadata, o.MetaGeneration)
	ExpectEq(6, o.Size)

	// Check attributes.
	attrs, err = t.in.Attributes(t.ctx)
	AssertEq(nil, err)

	ExpectEq(6, attrs.Size)
	ExpectThat(attrs.Mtime, timeutil.TimeEq(truncateTime.UTC()))
}

func (t *FileTest) TestTruncateUpwardForLocalFileShouldUpdateLocalFileAttributes() {
	var err error
	var attrs fuseops.InodeAttributes
	// Create a local file inode.
	t.createInodeWithLocalParam("test", true)
	err = t.in.CreateEmptyTempFile()
	AssertEq(nil, err)
	// Fetch the attributes and check if the file is empty.
	attrs, err = t.in.Attributes(t.ctx)
	AssertEq(nil, err)
	AssertEq(0, attrs.Size)

	err = t.in.Truncate(t.ctx, 6)

	AssertEq(nil, err)
	// The inode should return the new size.
	attrs, err = t.in.Attributes(t.ctx)
	AssertEq(nil, err)
	AssertEq(6, attrs.Size)
	// Data shouldn't be updated to GCS.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
	_, err = t.bucket.StatObject(t.ctx, statReq)
	AssertNe(nil, err)
	AssertEq("gcs.NotFoundError: Object test not found", err.Error())
}

func (t *FileTest) TestTruncateDownwardForLocalFileShouldUpdateLocalFileAttributes() {
	var err error
	var attrs fuseops.InodeAttributes
	// Create a local file inode.
	t.createInodeWithLocalParam("test", true)
	err = t.in.CreateEmptyTempFile()
	AssertEq(nil, err)
	// Write some data to the local file.
	err = t.in.Write(t.ctx, []byte("burrito"), 0)
	AssertEq(nil, err)
	// Validate the new data is written correctly.
	attrs, err = t.in.Attributes(t.ctx)
	AssertEq(nil, err)
	AssertEq(7, attrs.Size)

	err = t.in.Truncate(t.ctx, 2)

	AssertEq(nil, err)
	// The inode should return the new size.
	attrs, err = t.in.Attributes(t.ctx)
	AssertEq(nil, err)
	AssertEq(2, attrs.Size)
	// Data shouldn't be updated to GCS.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
	_, err = t.bucket.StatObject(t.ctx, statReq)
	AssertNe(nil, err)
	AssertEq("gcs.NotFoundError: Object test not found", err.Error())
}

func (t *FileTest) Sync_Clobbered() {
	var err error

	// Truncate downward.
	err = t.in.Truncate(t.ctx, 2)
	AssertEq(nil, err)

	// Clobber the backing object.
	newObj, err := storageutil.CreateObject(
		t.ctx,
		t.bucket,
		t.in.Name().GcsObjectName(),
		[]byte("burrito"))

	AssertEq(nil, err)

	// Sync. The call should succeed, but nothing should change.
	err = t.in.Sync(t.ctx)

	AssertEq(nil, err)
	ExpectEq(t.backingObj.Generation, t.in.SourceGeneration().Object)
	ExpectEq(t.backingObj.MetaGeneration, t.in.SourceGeneration().Metadata)

	// The object in the bucket should not have been changed.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
	o, err := t.bucket.StatObject(t.ctx, statReq)

	AssertEq(nil, err)
	ExpectEq(newObj.Generation, o.Generation)
	ExpectEq(newObj.Size, o.Size)
}

func (t *FileTest) SetMtime_ContentNotFaultedIn() {
	var err error
	var attrs fuseops.InodeAttributes

	// Set mtime.
	mtime := time.Now().UTC().Add(123*time.Second).AddDate(0, 0, 0)

	err = t.in.SetMtime(t.ctx, mtime)
	AssertEq(nil, err)

	// The inode should agree about the new mtime.
	attrs, err = t.in.Attributes(t.ctx)

	AssertEq(nil, err)
	ExpectThat(attrs.Mtime, timeutil.TimeEq(mtime))

	// The inode should have added the mtime to the backing object's metadata.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
	o, err := t.bucket.StatObject(t.ctx, statReq)

	AssertEq(nil, err)
	ExpectEq(
		mtime.UTC().Format(time.RFC3339Nano),
		o.Metadata["gcsfuse_mtime"])
}

func (t *FileTest) SetMtime_ContentClean() {
	var err error
	var attrs fuseops.InodeAttributes

	// Cause the content to be faulted in.
	_, err = t.in.Read(t.ctx, make([]byte, 1), 0)
	AssertEq(nil, err)

	// Set mtime.
	mtime := time.Now().UTC().Add(123*time.Second).AddDate(0, 0, 0)

	err = t.in.SetMtime(t.ctx, mtime)
	AssertEq(nil, err)

	// The inode should agree about the new mtime.
	attrs, err = t.in.Attributes(t.ctx)

	AssertEq(nil, err)
	ExpectThat(attrs.Mtime, timeutil.TimeEq(mtime))

	// The inode should have added the mtime to the backing object's metadata.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
	o, err := t.bucket.StatObject(t.ctx, statReq)

	AssertEq(nil, err)
	ExpectEq(
		mtime.UTC().Format(time.RFC3339Nano),
		o.Metadata["gcsfuse_mtime"])
}

func (t *FileTest) SetMtime_ContentDirty() {
	var err error
	var attrs fuseops.InodeAttributes

	// Dirty the content.
	err = t.in.Write(t.ctx, []byte("a"), 0)
	AssertEq(nil, err)

	// Set mtime.
	mtime := time.Now().UTC().Add(123 * time.Second)

	err = t.in.SetMtime(t.ctx, mtime)
	AssertEq(nil, err)

	// The inode should agree about the new mtime.
	attrs, err = t.in.Attributes(t.ctx)

	AssertEq(nil, err)
	ExpectThat(attrs.Mtime, timeutil.TimeEq(mtime))

	// Sync.
	err = t.in.Sync(t.ctx)
	AssertEq(nil, err)

	// Now the object in the bucket should have the appropriate mtime.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
	o, err := t.bucket.StatObject(t.ctx, statReq)

	AssertEq(nil, err)
	ExpectEq(
		mtime.UTC().Format(time.RFC3339Nano),
		o.Metadata["gcsfuse_mtime"])
}

func (t *FileTest) SetMtime_SourceObjectGenerationChanged() {
	var err error

	// Clobber the backing object.
	newObj, err := storageutil.CreateObject(
		t.ctx,
		t.bucket,
		t.in.Name().GcsObjectName(),
		[]byte("burrito"))

	AssertEq(nil, err)

	// Set mtime.
	mtime := time.Now().UTC().Add(123 * time.Second)
	err = t.in.SetMtime(t.ctx, mtime)
	AssertEq(nil, err)

	// The object in the bucket should not have been changed.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
	o, err := t.bucket.StatObject(t.ctx, statReq)

	AssertEq(nil, err)
	ExpectEq(newObj.Generation, o.Generation)
	ExpectEq(0, len(o.Metadata))
}

func (t *FileTest) SetMtime_SourceObjectMetaGenerationChanged() {
	var err error

	// Update the backing object.
	lang := "fr"
	newObj, err := t.bucket.UpdateObject(
		t.ctx,
		&gcs.UpdateObjectRequest{
			Name:            t.in.Name().GcsObjectName(),
			ContentLanguage: &lang,
		})

	AssertEq(nil, err)

	// Set mtime.
	mtime := time.Now().UTC().Add(123 * time.Second)
	err = t.in.SetMtime(t.ctx, mtime)
	AssertEq(nil, err)

	// The object in the bucket should not have been changed.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
	o, err := t.bucket.StatObject(t.ctx, statReq)

	AssertEq(nil, err)
	ExpectEq(newObj.Generation, o.Generation)
	ExpectEq(newObj.MetaGeneration, o.MetaGeneration)
}

func (t *FileTest) TestSetMtimeForLocalFileShouldUpdateLocalFileAttributes() {
	var err error
	var attrs fuseops.InodeAttributes
	// Create a local file inode.
	t.createInodeWithLocalParam("test", true)
	createTime := t.in.mtimeClock.Now()
	err = t.in.CreateEmptyTempFile()
	AssertEq(nil, err)
	// Validate the attributes on an empty file.
	attrs, err = t.in.Attributes(t.ctx)
	AssertEq(nil, err)
	ExpectThat(attrs.Mtime, timeutil.TimeNear(createTime, Delta))

	// Set mtime.
	mtime := time.Now().UTC().Add(123 * time.Second)
	err = t.in.SetMtime(t.ctx, mtime)

	AssertEq(nil, err)
	// The inode should agree about the new mtime.
	attrs, err = t.in.Attributes(t.ctx)
	AssertEq(nil, err)
	ExpectThat(attrs.Mtime, timeutil.TimeEq(mtime))
	ExpectThat(attrs.Ctime, timeutil.TimeEq(mtime))
	ExpectThat(attrs.Atime, timeutil.TimeEq(mtime))
	// Data shouldn't be updated to GCS.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
	_, err = t.bucket.StatObject(t.ctx, statReq)
	AssertNe(nil, err)
	AssertEq("gcs.NotFoundError: Object test not found", err.Error())
}

func (t *FileTest) ContentEncodingGzip() {
	// Set up an explicit content-encoding on the backing object and re-create the inode.
	contentEncoding := "gzip"
	t.backingObj.ContentEncoding = contentEncoding

	t.createInode()

	AssertEq(contentEncoding, t.in.Source().ContentEncoding)
	AssertTrue(t.in.Source().HasContentEncodingGzip())
}

func (t *FileTest) ContentEncodingNone() {
	// Set up an explicit content-encoding on the backing object and re-create the inode.
	contentEncoding := ""
	t.backingObj.ContentEncoding = contentEncoding

	t.createInode()

	AssertEq(contentEncoding, t.in.Source().ContentEncoding)
	AssertFalse(t.in.Source().HasContentEncodingGzip())
}

func (t *FileTest) ContentEncodingOther() {
	// Set up an explicit content-encoding on the backing object and re-create the inode.
	contentEncoding := "other"
	t.backingObj.ContentEncoding = contentEncoding

	t.createInode()

	AssertEq(contentEncoding, t.in.Source().ContentEncoding)
	AssertFalse(t.in.Source().HasContentEncodingGzip())
}

func (t *FileTest) TestCheckInvariantsShouldNotThrowExceptionForLocalFiles() {
	t.createInodeWithLocalParam("test", true)

	AssertNe(nil, t.in)
}

func (t *FileTest) TestCreateEmptyTempFileShouldCreateEmptyFile() {
	err := t.in.CreateEmptyTempFile()

	AssertEq(nil, err)
	AssertNe(nil, t.in.content)
	// Validate that file size is 0.
	sr, err := t.in.content.Stat()
	AssertEq(nil, err)
	AssertEq(0, sr.Size)
}

func (t *FileTest) UnlinkLocalFile() {
	var err error
	// Create a local file inode.
	t.createInodeWithLocalParam("test", true)
	// Create a temp file for the local inode created above.
	err = t.in.CreateEmptyTempFile()
	AssertEq(nil, err)

	// Unlink.
	t.in.Unlink()

	// Verify that fileInode is now unlinked
	AssertTrue(t.in.IsUnlinked())
	// Data shouldn't be updated to GCS.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
	_, err = t.bucket.StatObject(t.ctx, statReq)
	AssertNe(nil, err)
	AssertEq("gcs.NotFoundError: Object test not found", err.Error())
}
