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

package handle

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/contentcache"
	"github.com/googlecloudplatform/gcsfuse/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/storageutil"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
)

func TestDirHandle(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type DirHandleTest struct {
	ctx    context.Context
	bucket gcsx.SyncerBucket
	clock  timeutil.SimulatedClock

	dh *DirHandle
}

var _ SetUpInterface = &DirHandleTest{}
var _ TearDownInterface = &DirHandleTest{}

func init() { RegisterTestSuite(&DirHandleTest{}) }

func (t *DirHandleTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
	t.bucket = gcsx.NewSyncerBucket(
		1, ".gcsfuse_tmp/", fake.NewFakeBucket(&t.clock, "some_bucket"))
	t.clock.SetTime(time.Date(2022, 8, 15, 22, 56, 0, 0, time.Local))
	t.resetDirHandle()
}

func (t *DirHandleTest) TearDown() {}

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////
func (t *DirHandleTest) resetDirHandle() {
	dirInode := inode.NewDirInode(
		17,
		inode.NewDirName(inode.NewRootName(""), "testDir"),
		fuseops.InodeAttributes{
			Uid:  123,
			Gid:  456,
			Mode: 0712,
		},
		false, // implicitDirs
		false, // enableNonExistentTypeCache
		0,     // typeCacheTTL
		&t.bucket,
		&t.clock,
		&t.clock,
		0,
	)

	t.dh = NewDirHandle(
		dirInode,
		true,
	)
}

func (t *DirHandleTest) createLocalFileInode(name string, id fuseops.InodeID) (in inode.Inode) {
	in = inode.NewFileInode(
		id,
		inode.NewFileName(t.dh.in.Name(), name),
		nil,
		fuseops.InodeAttributes{
			Uid:  123,
			Gid:  456,
			Mode: 0712,
		},
		&t.bucket,
		false, // localFileCache
		contentcache.New("", &t.clock),
		&t.clock,
		true) // localFile
	return
}

func (t *DirHandleTest) validateEntry(entry fuseutil.Dirent, name string, filetype fuseutil.DirentType) {
	AssertEq(name, entry.Name)
	AssertEq(filetype, entry.Type)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *DirHandleTest) EnsureEntriesWithLocalAndGCSFiles() {
	var err error
	// Set up empty GCS objects.
	// DirHandle holds a DirInode pointing to "testDir".
	_, err = storageutil.CreateObject(t.ctx, t.bucket, "testDir/gcsObject1", nil)
	AssertEq(nil, err)
	_, err = storageutil.CreateObject(t.ctx, t.bucket, "testDir/gcsObject2", nil)
	AssertEq(nil, err)
	// Set up local file inodes.
	localIn1 := t.createLocalFileInode("localFile1", 10)
	localIn2 := t.createLocalFileInode("localFile2", 20)
	// Setup localFileInodes Map.
	localFileInodes := map[inode.Name]inode.Inode{
		localIn1.Name(): localIn1,
		localIn2.Name(): localIn2,
	}

	// Ensure entries.
	err = t.dh.ensureEntries(t.ctx, localFileInodes)

	// Validations
	AssertEq(nil, err)
	AssertEq(4, len(t.dh.entries))
	t.validateEntry(t.dh.entries[0], "gcsObject1", fuseutil.DT_File)
	t.validateEntry(t.dh.entries[1], "gcsObject2", fuseutil.DT_File)
	t.validateEntry(t.dh.entries[2], "localFile1", fuseutil.DT_File)
	t.validateEntry(t.dh.entries[3], "localFile2", fuseutil.DT_File)
}

func (t *DirHandleTest) EnsureEntriesWithOnlyLocalFiles() {
	var err error
	// Set up empty GCS objects.
	// DirHandle holds a DirInode pointing to "testDir".
	_, err = storageutil.CreateObject(t.ctx, t.bucket, "testDir/gcsObject1", nil)
	AssertEq(nil, err)
	_, err = storageutil.CreateObject(t.ctx, t.bucket, "testDir/gcsObject2", nil)
	AssertEq(nil, err)
	// Setup empty localFileInodes Map.
	localFileInodes := map[inode.Name]inode.Inode{}

	// Ensure entries.
	err = t.dh.ensureEntries(t.ctx, localFileInodes)

	// Validations
	AssertEq(nil, err)
	AssertEq(2, len(t.dh.entries))
	t.validateEntry(t.dh.entries[0], "gcsObject1", fuseutil.DT_File)
	t.validateEntry(t.dh.entries[1], "gcsObject2", fuseutil.DT_File)
}

func (t *DirHandleTest) EnsureEntriesWithOnlyGCSFiles() {
	var err error
	// Set up local file inodes.
	localIn1 := t.createLocalFileInode("localFile1", 10)
	localIn2 := t.createLocalFileInode("localFile2", 20)
	// Setup localFileInodes Map.
	localFileInodes := map[inode.Name]inode.Inode{
		localIn1.Name(): localIn1,
		localIn2.Name(): localIn2,
	}

	// Ensure entries.
	err = t.dh.ensureEntries(t.ctx, localFileInodes)

	// Validations
	AssertEq(nil, err)
	AssertEq(2, len(t.dh.entries))
	t.validateEntry(t.dh.entries[0], "localFile1", fuseutil.DT_File)
	t.validateEntry(t.dh.entries[1], "localFile2", fuseutil.DT_File)
}

func (t *DirHandleTest) EnsureEntriesWithSameNameLocalAndGCSFile() {
	var err error
	// Set up empty GCS objects.
	// DirHandle holds a DirInode pointing to "testDir".
	_, err = storageutil.CreateObject(t.ctx, t.bucket, "testDir/file1", nil)
	AssertEq(nil, err)
	// Set up local file inodes.
	localIn := t.createLocalFileInode("file1", 10)
	// Setup localFileInodes Map.
	localFileInodes := map[inode.Name]inode.Inode{
		localIn.Name(): localIn,
	}

	// Ensure entries.
	err = t.dh.ensureEntries(t.ctx, localFileInodes)

	// Validations
	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), "readAllEntries: fixConflictingNames: "))
}

func (t *DirHandleTest) EnsureEntriesWithSameNameLocalFileAndGCSDirectory() {
	var err error
	// Set up empty GCS objects.
	// DirHandle holds a DirInode pointing to "testDir".
	_, err = storageutil.CreateObject(t.ctx, t.bucket, "testDir/file1/", nil)
	AssertEq(nil, err)
	// Set up local file inodes.
	localIn := t.createLocalFileInode("file1", 10)
	// Setup localFileInodes Map.
	localFileInodes := map[inode.Name]inode.Inode{
		localIn.Name(): localIn,
	}

	// Ensure entries.
	err = t.dh.ensureEntries(t.ctx, localFileInodes)

	// Validations
	AssertEq(nil, err)
	AssertEq(2, len(t.dh.entries))
	t.validateEntry(t.dh.entries[0], "file1", fuseutil.DT_Directory)
	t.validateEntry(t.dh.entries[1], "file1"+inode.ConflictingFileNameSuffix, fuseutil.DT_File)
}
