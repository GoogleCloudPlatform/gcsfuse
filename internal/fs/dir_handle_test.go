// Copyright 2023 Google Inc. All Rights Reserved.
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

package fs

import (
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/internal/gcsx"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsfake"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
	"golang.org/x/net/context"
)

const dirInodeID = 17
const dirInodeName = "foo/"
const dirMode os.FileMode = 0712 | os.ModeDir
const fileUnderDir = "bar"
const implicitDirName = "baz"
const typeCacheTTL = time.Second
const uid = 123
const gid = 456
const tmpObjectPrefix = ".gcsfuse_tmp/"
const appendThreshold = 1
const fakeBucketName = "some_bucket"
const dummyInodeId = fuseops.InodeID(2)
const dummyfuseid = uint64(4)
const dummypid = uint32(6)
const dummyHandleId = fuseops.HandleID(8)

type DirHandleTest struct {
	ctx    context.Context
	bucket gcsx.SyncerBucket
	clock  timeutil.SimulatedClock
	dh     *dirHandle
}

var _ SetUpInterface = &DirHandleTest{}
var _ TearDownInterface = &DirHandleTest{}

func init() {
	RegisterTestSuite(&DirHandleTest{})
}

func (t *DirHandleTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
	t.clock.SetTime(time.Date(2023, 6, 26, 15, 55, 0, 0, time.Local))
	bucket := gcsfake.NewFakeBucket(&t.clock, fakeBucketName)
	t.bucket = gcsx.NewSyncerBucket(
		int64(appendThreshold), // Append threshold
		tmpObjectPrefix,
		bucket)
}

func TestDirHandle(t *testing.T) {
	RunTests(t)
}

func (t *DirHandleTest) TearDown() {
}

// Create the inode and directory handle. No implicit dirs by default.
func (t *DirHandleTest) createDirHandle(implicitDirs bool, enableNonexistentTypecache bool, dirInodeName string) {
	in := inode.NewDirInode(
		dirInodeID,
		inode.NewDirName(inode.NewRootName(""), dirInodeName),
		fuseops.InodeAttributes{
			Uid:  uid,
			Gid:  gid,
			Mode: dirMode,
		},
		implicitDirs,
		enableNonexistentTypecache,
		typeCacheTTL,
		&t.bucket,
		&t.clock,
		&t.clock)
	t.dh = newDirHandle(in, implicitDirs)
}

func (t *DirHandleTest) resetDirHandle() {
	t.dh = nil
}

func (t *DirHandleTest) createImplicitDirDefinedByFile() (err error) {
	contents := "Implicit dir"
	filePath := path.Join(dirInodeName, path.Join(implicitDirName, fileUnderDir))
	_, err = t.bucket.CreateObject(
		t.ctx,
		&gcs.CreateObjectRequest{
			Name:     filePath,
			Contents: strings.NewReader(contents),
		})
	return err
}

// Directory Structure Used
// foo   --Directory
// fetchEntriesAsync will return 0 entries for empty directory.
func (t *DirHandleTest) FetchAsyncEntries_EmptyDir() {
	t.createDirHandle(false, false, dirInodeName)
	t.dh.FetchEntriesAsync(fuseops.RootInodeID)

	AssertEq(0, len(t.dh.entries))
	AssertEq(true, t.dh.entriesValid)
	t.resetDirHandle()
}

// Directory Structure Used
// foo       --Directory
// foo/bar   --File
// fetchEntriesAsync will return 1 entry for directory with 1 file.
func (t *DirHandleTest) FetchAsyncEntries_NonEmptyDir() {
	contents := "Non-empty dir"
	filePath := path.Join(dirInodeName, fileUnderDir)
	_, err := t.bucket.CreateObject(
		t.ctx,
		&gcs.CreateObjectRequest{
			Name:     filePath,
			Contents: strings.NewReader(contents),
		})
	AssertEq(nil, err)

	t.createDirHandle(false, false, dirInodeName)
	t.dh.FetchEntriesAsync(fuseops.RootInodeID)
	entries := t.dh.entries

	AssertEq(1, len(entries))
	AssertEq(fileUnderDir, entries[0].Name)
	AssertEq(true, t.dh.entriesValid)
	t.resetDirHandle()
}

// Directory Structure Used
// foo              --Directory
// foo/baz          --Implicit Directory
// foo/baz/bar      --file
// fetchEntriesAsync will return 1 entry for implicit directory if flag is set to true.
func (t *DirHandleTest) FetchAsyncEntries_ImplicitDir_FlagTrue() {
	err := t.createImplicitDirDefinedByFile()
	AssertEq(nil, err)

	//implicit-dirs flag set to true
	t.createDirHandle(true, false, dirInodeName)
	t.dh.FetchEntriesAsync(fuseops.RootInodeID)
	entries := t.dh.entries

	AssertEq(1, len(entries))
	AssertEq(implicitDirName, entries[0].Name)
	AssertEq(true, t.dh.entriesValid)
	t.resetDirHandle()
}

// Same directory structure as above.
// fetchEntriesAsync will return 0 entry for implicit directory if flag is set to false.
func (t *DirHandleTest) FetchAsyncEntries_ImplicitDir_FlagFalse() {
	err := t.createImplicitDirDefinedByFile()
	AssertEq(nil, err)

	//implicit-dirs flag set to false
	t.createDirHandle(false, false, dirInodeName)
	t.dh.FetchEntriesAsync(fuseops.RootInodeID)

	AssertEq(0, len(t.dh.entries))
	AssertEq(true, t.dh.entriesValid)
	t.resetDirHandle()
}

// Directory Structure Used
// foo       --Directory
// foo/bar   --File
// Offset for first read dir operation is zero.Hence, entries for dir will
// be fetched. Bytes read for this operation will be 32(size of 1 dirent)
// as according to the directory structure created.
func (t *DirHandleTest) Readdir_OffsetZero() {
	contents := "read dir at zero dirOffset"
	filePath := path.Join(dirInodeName, fileUnderDir)
	_, err := t.bucket.CreateObject(
		t.ctx,
		&gcs.CreateObjectRequest{
			Name:     filePath,
			Contents: strings.NewReader(contents),
		})
	AssertEq(nil, err)
	t.createDirHandle(false, false, dirInodeName)
	op := &fuseops.ReadDirOp{
		Inode:  fuseops.InodeID(dummyInodeId),
		Handle: fuseops.HandleID(dummyHandleId),
		Offset: fuseops.DirOffset(0),
		OpContext: fuseops.OpContext{
			FuseID: dummyfuseid,
			Pid:    dummypid,
		},
		Dst: make([]byte, 1024),
	}

	err = t.dh.ReadDir(t.ctx, op)

	AssertEq(nil, err)
	AssertEq(1024, len(op.Dst))
	AssertEq(32, op.BytesRead)
}

// Directory Structure Used
// foo       --Directory
// foo/bar   --File
// Offset for first read dir operation is non-zero.Hence, no entries are fetched.
// Bytes read for this operation will be zero.
func (t *DirHandleTest) Readdir_OffsetNonZero() {
	contents := "read dir at non-zero dirOffset"
	filePath := path.Join(dirInodeName, fileUnderDir)
	_, err := t.bucket.CreateObject(
		t.ctx,
		&gcs.CreateObjectRequest{
			Name:     filePath,
			Contents: strings.NewReader(contents),
		})
	AssertEq(nil, err)
	t.createDirHandle(false, false, dirInodeName)
	op := &fuseops.ReadDirOp{
		Inode:  fuseops.InodeID(dummyInodeId),
		Handle: fuseops.HandleID(dummyHandleId),
		Offset: fuseops.DirOffset(1),
		OpContext: fuseops.OpContext{
			FuseID: dummyfuseid,
			Pid:    dummypid,
		},
		Dst: make([]byte, 1024),
	}

	err = t.dh.ReadDir(t.ctx, op)

	AssertEq(nil, err)
	AssertEq(1024, len(op.Dst))
	AssertEq(0, op.BytesRead)
}

// Directory Structure Used
// foo       --Directory
// foo/bar   --File
// Offset for first read dir operation is zero.However, the size of destination buffer
// is less than the size of a single dirent here.Hence, even though the entry is fetched,
// the entry cannot be copied into the buffer and bytes read is 0.
func (t *DirHandleTest) Readdir_InsufficientBufferSize() {
	contents := "read dir with insufficient buffer size"
	filePath := path.Join(dirInodeName, fileUnderDir)
	_, err := t.bucket.CreateObject(
		t.ctx,
		&gcs.CreateObjectRequest{
			Name:     filePath,
			Contents: strings.NewReader(contents),
		})
	AssertEq(nil, err)
	t.createDirHandle(false, false, dirInodeName)
	op := &fuseops.ReadDirOp{
		Inode:  fuseops.InodeID(dummyInodeId),
		Handle: fuseops.HandleID(dummyHandleId),
		Offset: fuseops.DirOffset(0),
		OpContext: fuseops.OpContext{
			FuseID: dummyfuseid,
			Pid:    dummypid,
		},
		Dst: make([]byte, 24),
	}

	err = t.dh.ReadDir(t.ctx, op)

	AssertEq(nil, err)
	AssertEq(24, len(op.Dst))
	AssertEq(0, op.BytesRead)
}
