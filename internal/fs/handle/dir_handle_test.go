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

package handle

import (
	"context"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/metadata"
	"math"
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/contentcache"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
	"golang.org/x/sync/semaphore"
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
		1, 10, ".gcsfuse_tmp/", fake.NewFakeBucket(&t.clock, "some_bucket", gcs.BucketType{}))
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
		false, // implicitDirs,
		true,  // enableManagedFoldersListing
		false, // enableNonExistentTypeCache
		0,     // typeCacheTTL
		&t.bucket,
		&t.clock,
		&t.clock,
		0,
		false)

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
		true, // localFile
		&cfg.Config{},
		semaphore.NewWeighted(math.MaxInt64))
	return
}

func (t *DirHandleTest) validateEntry(entry fuseutil.Dirent, name string, filetype fuseutil.DirentType) {
	AssertEq(name, entry.Name)
	AssertEq(filetype, entry.Type)
}

func (t *DirHandleTest) validateCore(core *inode.Core, name string, filetype metadata.Type, minObjectName string) {
	AssertNe(nil, core)
	AssertNe(nil, core.MinObject)
	AssertEq(name, path.Base(core.FullName.LocalName()))
	AssertEq(minObjectName, core.MinObject.Name)
	AssertEq(filetype, core.Type())
}

func (t *DirHandleTest) createTestDirentPlus(name string, dtype fuseutil.DirentType, childInodeID fuseops.InodeID, size uint64) fuseutil.DirentPlus {
	attrs := fuseops.InodeAttributes{
		Size:  size,
		Mode:  0777,
		Nlink: 1,
		Uid:   123,
		Gid:   456,
	}
	if dtype != fuseutil.DT_Directory {
		attrs.Mode = 0666
	}

	return fuseutil.DirentPlus{
		Dirent: fuseutil.Dirent{
			Name: name,
			Type: dtype,
			// Offset and Inode will be set by ReadDirPlus
		},
		Entry: fuseops.ChildInodeEntry{
			Child:      childInodeID,
			Attributes: attrs,
			// EntryValid and AttrValid can be set if needed
		},
	}
}

func (t *DirHandleTest) validateEntryPlus(entry fuseutil.DirentPlus, name string, fileType fuseutil.DirentType, childInodeID fuseops.InodeID) {
	AssertEq(name, entry.Dirent.Name)
	AssertEq(fileType, entry.Dirent.Type)
	AssertEq(childInodeID, entry.Entry.Child)
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
	localFileName1 := "localFile1"
	localFileName2 := "localFile2"
	// Setup localFileEntries.
	localFileEntries := map[string]fuseutil.Dirent{
		localFileName1: {Offset: 0, Inode: 10, Name: localFileName1, Type: fuseutil.DT_File},
		localFileName2: {Offset: 0, Inode: 20, Name: localFileName2, Type: fuseutil.DT_File},
	}

	// Ensure entries.
	err = t.dh.ensureEntries(t.ctx, localFileEntries)

	// Validations
	AssertEq(nil, err)
	AssertEq(4, len(t.dh.entries))
	t.validateEntry(t.dh.entries[0], "gcsObject1", fuseutil.DT_File)
	t.validateEntry(t.dh.entries[1], "gcsObject2", fuseutil.DT_File)
	t.validateEntry(t.dh.entries[2], localFileName1, fuseutil.DT_File)
	t.validateEntry(t.dh.entries[3], localFileName2, fuseutil.DT_File)
}

func (t *DirHandleTest) EnsureEntriesWithOnlyGCSFiles() {
	var err error
	// Set up empty GCS objects.
	// DirHandle holds a DirInode pointing to "testDir".
	_, err = storageutil.CreateObject(t.ctx, t.bucket, "testDir/gcsObject1", nil)
	AssertEq(nil, err)
	_, err = storageutil.CreateObject(t.ctx, t.bucket, "testDir/gcsObject2", nil)
	AssertEq(nil, err)
	// Setup empty localFileEntries.
	var localFileEntries map[string]fuseutil.Dirent

	// Ensure entries.
	err = t.dh.ensureEntries(t.ctx, localFileEntries)

	// Validations
	AssertEq(nil, err)
	AssertEq(2, len(t.dh.entries))
	t.validateEntry(t.dh.entries[0], "gcsObject1", fuseutil.DT_File)
	t.validateEntry(t.dh.entries[1], "gcsObject2", fuseutil.DT_File)
}

func (t *DirHandleTest) EnsureEntriesWithOnlyLocalFiles() {
	var err error
	localFileName1 := "localFile1"
	localFileName2 := "localFile2"
	// Setup localFileEntries.
	localFileEntries := map[string]fuseutil.Dirent{
		localFileName1: {Offset: 0, Inode: 10, Name: localFileName1, Type: fuseutil.DT_File},
		localFileName2: {Offset: 0, Inode: 20, Name: localFileName2, Type: fuseutil.DT_File},
	}

	// Ensure entries.
	err = t.dh.ensureEntries(t.ctx, localFileEntries)

	// Validations
	AssertEq(nil, err)
	AssertEq(2, len(t.dh.entries))
	t.validateEntry(t.dh.entries[0], localFileName1, fuseutil.DT_File)
	t.validateEntry(t.dh.entries[1], localFileName2, fuseutil.DT_File)
}

func (t *DirHandleTest) EnsureEntriesWithSameNameLocalAndGCSFile() {
	var err error
	// Set up empty GCS objects.
	// DirHandle holds a DirInode pointing to "testDir".
	_, err = storageutil.CreateObject(t.ctx, t.bucket, "testDir/file1", nil)
	AssertEq(nil, err)
	localFileName := "file1"
	// Setup localFileEntries.
	localFileEntries := map[string]fuseutil.Dirent{
		localFileName: {Offset: 0, Inode: 10, Name: localFileName, Type: fuseutil.DT_File},
	}

	// Ensure entries.
	err = t.dh.ensureEntries(t.ctx, localFileEntries)

	// Validations
	AssertEq(nil, err)
	AssertEq(1, len(t.dh.entries))
	t.validateEntry(t.dh.entries[0], localFileName, fuseutil.DT_File)
}

func (t *DirHandleTest) EnsureEntriesWithSameNameLocalFileAndGCSDirectory() {
	var err error
	// Set up empty GCS objects.
	// DirHandle holds a DirInode pointing to "testDir".
	_, err = storageutil.CreateObject(t.ctx, t.bucket, "testDir/file1/", nil)
	AssertEq(nil, err)
	localFileName := "file1"
	// Setup localFileEntries.
	localFileEntries := map[string]fuseutil.Dirent{
		localFileName: {Offset: 0, Inode: 10, Name: localFileName, Type: fuseutil.DT_File},
	}

	// Ensure entries.
	err = t.dh.ensureEntries(t.ctx, localFileEntries)

	// Validations
	AssertEq(nil, err)
	AssertEq(2, len(t.dh.entries))
	t.validateEntry(t.dh.entries[0], localFileName, fuseutil.DT_Directory)
	t.validateEntry(t.dh.entries[1], localFileName+inode.ConflictingFileNameSuffix, fuseutil.DT_File)
}

func (t *DirHandleTest) EnsureEntriesWithNoFiles() {
	// Setup localFileEntries.
	localFileEntries := map[string]fuseutil.Dirent{}

	// Ensure entries.
	err := t.dh.ensureEntries(t.ctx, localFileEntries)

	// Validations
	AssertEq(nil, err)
	AssertEq(0, len(t.dh.entries))
}

func (t *DirHandleTest) EnsureEntriesWithOneGCSFile() {
	var err error
	// Set up empty GCS objects.
	// DirHandle holds a DirInode pointing to "testDir".
	_, err = storageutil.CreateObject(t.ctx, t.bucket, "testDir/gcsObject1", nil)
	AssertEq(nil, err)
	// Setup empty localFileEntries.
	var localFileEntries map[string]fuseutil.Dirent

	// Ensure entries.
	err = t.dh.ensureEntries(t.ctx, localFileEntries)

	// Validations
	AssertEq(nil, err)
	AssertEq(1, len(t.dh.entries))
	t.validateEntry(t.dh.entries[0], "gcsObject1", fuseutil.DT_File)
}

func (t *DirHandleTest) EnsureEntriesWithOneLocalFile() {
	var err error
	localFileName1 := "localFile1"
	// Setup localFileEntries.
	localFileEntries := map[string]fuseutil.Dirent{
		localFileName1: {Offset: 0, Inode: 10, Name: localFileName1, Type: fuseutil.DT_File},
	}

	// Ensure entries.
	err = t.dh.ensureEntries(t.ctx, localFileEntries)

	// Validations
	AssertEq(nil, err)
	AssertEq(1, len(t.dh.entries))
	t.validateEntry(t.dh.entries[0], localFileName1, fuseutil.DT_File)
}

////////////////////////////////////////////////////////////////////////
// ensureEntriesPlus Tests (for readAllEntriesPlus)
////////////////////////////////////////////////////////////////////////

func (t *DirHandleTest) EnsureEntriesPlusWithNoFiles() {
	cores, err := t.dh.ensureEntriesPlus(t.ctx)

	AssertEq(nil, err)
	AssertEq(0, len(cores))
}

func (t *DirHandleTest) EnsureEntriesPlusWithOnlyGCSFiles() {
	// Setup GCS objects
	_, err := storageutil.CreateObject(t.ctx, t.bucket, "testDir/gcsObject1", nil)
	AssertEq(nil, err)
	_, err = storageutil.CreateObject(t.ctx, t.bucket, "testDir/gcsObject2", nil)
	AssertEq(nil, err)

	// ensure entries
	cores, err := t.dh.ensureEntriesPlus(t.ctx)

	// validations
	AssertEq(nil, err)
	AssertEq(2, len(cores))

	coreFile1, ok := cores[inode.NewFileName(t.dh.in.Name(), "gcsObject1")]
	AssertTrue(ok, "Core for gcsObject1 not found")
	t.validateCore(coreFile1, "gcsObject1", metadata.RegularFileType, "testDir/gcsObject1")

	coreFile2, ok := cores[inode.NewFileName(t.dh.in.Name(), "gcsObject2")]
	AssertTrue(ok, "Core for gcsObject2 not found")
	t.validateCore(coreFile2, "gcsObject2", metadata.RegularFileType, "testDir/gcsObject2")
}

////////////////////////////////////////////////////////////////////////
// ReadDirPlusHelper Tests
////////////////////////////////////////////////////////////////////////

func (t *DirHandleTest) ReadDirPlusHelperPopulatesCores() {
	_, err := storageutil.CreateObject(t.ctx, t.bucket, "testDir/testFile", nil)
	AssertEq(nil, err)
	op := &fuseops.ReadDirPlusOp{
		ReadDirOp: fuseops.ReadDirOp{Offset: 0},
	}
	t.dh.entriesPlusValid = false

	cores, err := t.dh.ReadDirPlusHelper(t.ctx, op)

	AssertEq(nil, err)
	AssertEq(1, len(cores))

	coreFile, ok := cores[inode.NewFileName(t.dh.in.Name(), "testFile")]
	AssertTrue(ok, "Core for gcsFile1 not found")
	t.validateCore(coreFile, "testFile", metadata.RegularFileType, "testDir/testFile")
}

func (t *DirHandleTest) ReadDirPlusHelperNonZeroOffsetNoFetchIfCacheValid() {
	t.dh.entriesPlus = []fuseutil.DirentPlus{{}}
	t.dh.entriesPlusValid = true
	op := &fuseops.ReadDirPlusOp{
		ReadDirOp: fuseops.ReadDirOp{Offset: 1},
	}

	cores, err := t.dh.ReadDirPlusHelper(t.ctx, op)

	AssertEq(nil, err)
	AssertEq(nil, cores)
	AssertTrue(t.dh.entriesPlusValid)
}

func (t *DirHandleTest) ReadDirPlusHelperNonZeroOffsetFetchesIfCacheInvalid() {
	t.dh.entriesPlusValid = false
	_, err := storageutil.CreateObject(t.ctx, t.bucket, "testDir/fetchThis", nil)
	AssertEq(nil, err)
	op := &fuseops.ReadDirPlusOp{
		ReadDirOp: fuseops.ReadDirOp{Offset: 1},
	}

	cores, err := t.dh.ReadDirPlusHelper(t.ctx, op)

	AssertEq(nil, err)
	AssertEq(1, len(cores))
}

////////////////////////////////////////////////////////////////////////
// ReadDirPlus Tests
////////////////////////////////////////////////////////////////////////

func (t *DirHandleTest) ReadDirPlusResponseForNoFile() {
	op := &fuseops.ReadDirPlusOp{
		ReadDirOp: fuseops.ReadDirOp{Dst: make([]byte, 1024)},
	}
	var gcsEntriesPlus []fuseutil.DirentPlus
	localFileEntriesPlus := make(map[string]fuseutil.DirentPlus)

	err := t.dh.ReadDirPlus(op, gcsEntriesPlus, localFileEntriesPlus)

	AssertEq(nil, err)
	AssertEq(0, op.BytesRead)
	AssertTrue(t.dh.entriesPlusValid)
	AssertEq(0, len(t.dh.entriesPlus))
}

func (t *DirHandleTest) ReadDirPlusSameNameLocalAndGCSFile() {
	op := &fuseops.ReadDirPlusOp{
		ReadDirOp: fuseops.ReadDirOp{Dst: make([]byte, 1024)},
	}
	gcsFile := t.createTestDirentPlus("sameName", fuseutil.DT_File, 1001, 10)
	localFile := t.createTestDirentPlus("sameName", fuseutil.DT_File, 1002, 0)

	gcsEntriesPlus := []fuseutil.DirentPlus{gcsFile}
	localFileEntriesPlus := map[string]fuseutil.DirentPlus{"sameName": localFile}

	err := t.dh.ReadDirPlus(op, gcsEntriesPlus, localFileEntriesPlus)
	AssertEq(nil, err)
	AssertEq(1, len(t.dh.entriesPlus))

	t.validateEntryPlus(t.dh.entriesPlus[0], "sameName", fuseutil.DT_File, 1001)
}

func (t *DirHandleTest) ReadDirPlusSameNameLocalFileAndGCSDirectory() {
	op := &fuseops.ReadDirPlusOp{
		ReadDirOp: fuseops.ReadDirOp{Dst: make([]byte, 1024)},
	}
	gcsDir := t.createTestDirentPlus("sameName", fuseutil.DT_Directory, 1001, 0)
	gcsEntriesPlus := []fuseutil.DirentPlus{gcsDir}

	localFile := t.createTestDirentPlus("sameName", fuseutil.DT_File, 2001, 20)
	localFileEntriesPlus := map[string]fuseutil.DirentPlus{"sameName": localFile}

	err := t.dh.ReadDirPlus(op, gcsEntriesPlus, localFileEntriesPlus)
	AssertEq(nil, err)

	AssertEq(2, len(t.dh.entriesPlus))

	t.validateEntryPlus(t.dh.entriesPlus[0], "sameName", fuseutil.DT_Directory, 1001)
	t.validateEntryPlus(t.dh.entriesPlus[1], "sameName"+inode.ConflictingFileNameSuffix, fuseutil.DT_File, 2001)
}
