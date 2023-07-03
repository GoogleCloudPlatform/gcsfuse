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
const singleDirentSize = 32
const sufficientBufferSize = 1024
const insufficientBufferSize = 24

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
	now := time.Now()
	t.clock.SetTime(time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), time.Local))
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

func (t *DirHandleTest) createImplicitDirDefinedByFile() {
	contents := "Implicit dir"
	filePath := path.Join(dirInodeName, path.Join(implicitDirName, fileUnderDir))

	_, err := t.bucket.CreateObject(
		t.ctx,
		&gcs.CreateObjectRequest{
			Name:     filePath,
			Contents: strings.NewReader(contents),
		})

	AssertEq(nil, err)
}

// Directory Structure Created
// foo       --Directory
// foo/filename   --File
func (t *DirHandleTest) createSimpleDirectory(fileName string) {
	contents := "Simple File contents"
	filePath := path.Join(dirInodeName, fileName)

	_, err := t.bucket.CreateObject(
		t.ctx,
		&gcs.CreateObjectRequest{
			Name:     filePath,
			Contents: strings.NewReader(contents),
		})

	AssertEq(nil, err)
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
	t.createSimpleDirectory(fileUnderDir)
	t.createDirHandle(false, false, dirInodeName)

	t.dh.FetchEntriesAsync(fuseops.RootInodeID)

	AssertEq(1, len(t.dh.entries))
	AssertEq(fileUnderDir, t.dh.entries[0].Name)
	AssertEq(true, t.dh.entriesValid)

	t.resetDirHandle()
}

// Directory Structure Used
// foo              --Directory
// foo/baz          --Implicit Directory
// foo/baz/bar      --file
// fetchEntriesAsync will return 1 entry for implicit directory if flag is set to true.
func (t *DirHandleTest) FetchAsyncEntries_ImplicitDir_FlagTrue() {
	t.createImplicitDirDefinedByFile()
	//implicit-dirs flag set to true
	t.createDirHandle(true, false, dirInodeName)

	t.dh.FetchEntriesAsync(fuseops.RootInodeID)

	AssertEq(1, len(t.dh.entries))
	AssertEq(implicitDirName, t.dh.entries[0].Name)
	AssertEq(true, t.dh.entriesValid)

	t.resetDirHandle()
}

// Same directory structure as above.
// fetchEntriesAsync will return 0 entry for implicit directory if flag is set to false.
func (t *DirHandleTest) FetchAsyncEntries_ImplicitDir_FlagFalse() {
	t.createImplicitDirDefinedByFile()
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
	t.createSimpleDirectory(fileUnderDir)
	t.createDirHandle(false, false, dirInodeName)
	op := &fuseops.ReadDirOp{
		Inode:  fuseops.InodeID(dummyInodeId),
		Handle: fuseops.HandleID(dummyHandleId),
		Offset: fuseops.DirOffset(0),
		OpContext: fuseops.OpContext{
			FuseID: dummyfuseid,
			Pid:    dummypid,
		},
		Dst: make([]byte, sufficientBufferSize),
	}

	err := t.dh.ReadDir(t.ctx, op)

	AssertEq(nil, err)
	AssertEq(sufficientBufferSize, len(op.Dst))
	AssertEq(singleDirentSize, op.BytesRead)

	t.resetDirHandle()
}

// Directory Structure Used
// foo       --Directory
// foo/bar   --File
// Offset for first read dir operation is non-zero.Hence, no entries are fetched.
// Bytes read for this operation will be zero.
func (t *DirHandleTest) Readdir_OffsetNonZero() {
	t.createSimpleDirectory(fileUnderDir)
	t.createDirHandle(false, false, dirInodeName)
	op := &fuseops.ReadDirOp{
		Inode:  fuseops.InodeID(dummyInodeId),
		Handle: fuseops.HandleID(dummyHandleId),
		Offset: fuseops.DirOffset(1),
		OpContext: fuseops.OpContext{
			FuseID: dummyfuseid,
			Pid:    dummypid,
		},
		Dst: make([]byte, sufficientBufferSize),
	}

	err := t.dh.ReadDir(t.ctx, op)

	AssertEq(nil, err)
	AssertEq(sufficientBufferSize, len(op.Dst))
	AssertEq(0, op.BytesRead)

	t.resetDirHandle()
}

// Directory Structure Used
// foo       --Directory
// foo/bar   --File
// Offset for first read dir operation is zero.However, the size of destination buffer
// is less than the size of a single dirent here.Hence, even though the entry is fetched,
// the entry cannot be copied into the buffer and bytes read is 0.
func (t *DirHandleTest) Readdir_InsufficientBufferSize() {
	t.createSimpleDirectory(fileUnderDir)
	t.createDirHandle(false, false, dirInodeName)
	op := &fuseops.ReadDirOp{
		Inode:  fuseops.InodeID(dummyInodeId),
		Handle: fuseops.HandleID(dummyHandleId),
		Offset: fuseops.DirOffset(0),
		OpContext: fuseops.OpContext{
			FuseID: dummyfuseid,
			Pid:    dummypid,
		},
		Dst: make([]byte, insufficientBufferSize),
	}

	err := t.dh.ReadDir(t.ctx, op)

	AssertEq(nil, err)
	AssertEq(insufficientBufferSize, len(op.Dst))
	AssertEq(0, op.BytesRead)

	t.resetDirHandle()
}

// Test to check if the second readdir() call fetches the next entry.
// Test SetUp: Creating 2 files under the same directory.
// The size of buffer is kept to be the size of dirent , so that only
// 1 entry can be read into the buffer after each readdir() call
// when the first readdir is called at offset 0, 2 entries are fetched
// into dh.entries .However, only the first entry is written into the
// op.Dst buffer.After resetting the values of op.Bytesread to 0 (so
// that the same buffer is reused which also is similar to how a fresh
// buffer is made available with each readdir operation in real setting.
// Another readdir call is made at offset 1 and the second entry from
// dh.entries is buffered into op.Dst .
func (t *DirHandleTest) Readdir_verifySecondCall() {
	t.createSimpleDirectory(fileUnderDir)
	t.createSimpleDirectory(fileUnderDir + "2")
	t.createDirHandle(false, false, dirInodeName)
	op := &fuseops.ReadDirOp{
		Inode:  fuseops.InodeID(dummyInodeId),
		Handle: fuseops.HandleID(dummyHandleId),
		Offset: fuseops.DirOffset(0),
		OpContext: fuseops.OpContext{
			FuseID: dummyfuseid,
			Pid:    dummypid,
		},
		Dst: make([]byte, singleDirentSize),
	}

	errFirstFetch := t.dh.ReadDir(t.ctx, op)
	bytesReadFirstFetch := op.BytesRead
	op.Offset += 1
	op.BytesRead = 0
	errSecondFetch := t.dh.ReadDir(t.ctx, op)
	bytesReadSecFetch := op.BytesRead

	AssertEq(nil, errFirstFetch)
	AssertEq(nil, errSecondFetch)
	AssertEq(2, len(t.dh.entries))
	AssertEq(singleDirentSize, bytesReadFirstFetch)
	AssertEq(singleDirentSize, bytesReadSecFetch)

	t.resetDirHandle()
}
