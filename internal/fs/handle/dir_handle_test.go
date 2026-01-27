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
	"path"
	"testing"
	"time"

	cfg2 "github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"golang.org/x/sync/semaphore"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
)

const testDirentName = "sameName"

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
	cfg := &cfg2.Config{
		List:                         cfg2.ListConfig{EnableEmptyManagedFolders: true},
		MetadataCache:                cfg2.MetadataCacheConfig{TypeCacheMaxSizeMb: 0},
		EnableHns:                    false,
		EnableUnsupportedPathSupport: true,
		EnableTypeCacheDeprecation:   isTypeCacheDeprecationEnabled,
	}
	dirInode := inode.NewDirInode(
		17,
		inode.NewDirName(inode.NewRootName(""), "testDir"),
		fuseops.InodeAttributes{
			Uid:  123,
			Gid:  456,
			Mode: 0712,
		},
		false, // implicitDirs,
		false, // enableNonExistentTypeCache
		0,     // typeCacheTTL
		&t.bucket,
		&t.clock,
		&t.clock,
		semaphore.NewWeighted(10),
		cfg)

	t.dh = NewDirHandle(
		dirInode,
		true,
	)
}

func (t *DirHandleTest) validateEntry(entry fuseutil.Dirent, name string, filetype fuseutil.DirentType) {
	AssertEq(name, entry.Name)
	AssertEq(filetype, entry.Type)
}

func (t *DirHandleTest) createTestDirentPlus(dtype fuseutil.DirentType, childInodeID fuseops.InodeID, size uint64) fuseutil.DirentPlus {
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
			Name: testDirentName,
			Type: dtype,
		},
		Entry: fuseops.ChildInodeEntry{
			Child:      childInodeID,
			Attributes: attrs,
		},
	}
}

func (t *DirHandleTest) validateEntryPlus(entry fuseutil.DirentPlus, expectedName string, expectedType fuseutil.DirentType, expectedChildInodeID fuseops.InodeID) {
	AssertEq(expectedName, entry.Dirent.Name)
	AssertEq(expectedType, entry.Dirent.Type)
	AssertEq(expectedChildInodeID, entry.Entry.Child)
}

func (t *DirHandleTest) validateFileType(core *inode.Core, expectedName string, expectedMinObjectName string) {
	AssertNe(nil, core)
	AssertNe(nil, core.MinObject)
	AssertEq(expectedName, path.Base(core.FullName.LocalName()))
	AssertEq(expectedMinObjectName, core.MinObject.Name)
	AssertEq(metadata.RegularFileType, core.Type())
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

func (t *DirHandleTest) ReadAllEntryCoresWithNoEntry() {
	cores, err := readAllEntryCores(t.ctx, t.dh.in)

	AssertEq(nil, err)
	AssertEq(0, len(cores))
}

func (t *DirHandleTest) ReadAllEntryCoresReturnsAllEntryCores() {
	// Setup GCS objects
	_, err := storageutil.CreateObject(t.ctx, t.bucket, "testDir/gcsObject1", nil)
	AssertEq(nil, err)
	_, err = storageutil.CreateObject(t.ctx, t.bucket, "testDir/gcsObject2", nil)
	AssertEq(nil, err)

	// read all entry cores
	cores, err := readAllEntryCores(t.ctx, t.dh.in)

	// validations
	AssertEq(nil, err)
	AssertEq(2, len(cores))
	entry1, ok := cores[inode.NewFileName(t.dh.in.Name(), "gcsObject1")]
	AssertTrue(ok, "Core for gcsObject1 not found")
	t.validateFileType(entry1, "gcsObject1", "testDir/gcsObject1")
	entry2, ok := cores[inode.NewFileName(t.dh.in.Name(), "gcsObject2")]
	AssertTrue(ok, "Core for gcsObject2 not found")
	t.validateFileType(entry2, "gcsObject2", "testDir/gcsObject2")
}

func (t *DirHandleTest) FetchEntryCoresFetchesCores() {
	_, err := storageutil.CreateObject(t.ctx, t.bucket, "testDir/testFile", nil)
	AssertEq(nil, err)
	op := &fuseops.ReadDirPlusOp{
		ReadDirOp: fuseops.ReadDirOp{Offset: 0},
	}
	t.dh.entriesPlusValid = false

	cores, err := t.dh.FetchEntryCores(t.ctx, op)

	AssertEq(nil, err)
	AssertEq(1, len(cores))
	entry, ok := cores[inode.NewFileName(t.dh.in.Name(), "testFile")]
	AssertTrue(ok, "Core for gcsFile1 not found")
	t.validateFileType(entry, "testFile", "testDir/testFile")
}

func (t *DirHandleTest) FetchEntryCoresNonZeroOffsetNoFetchIfCacheValid() {
	t.dh.entriesPlus = []fuseutil.DirentPlus{{}}
	t.dh.entriesPlusValid = true
	op := &fuseops.ReadDirPlusOp{
		ReadDirOp: fuseops.ReadDirOp{Offset: 1},
	}

	cores, err := t.dh.FetchEntryCores(t.ctx, op)

	AssertEq(nil, err)
	AssertEq(nil, cores)
	AssertTrue(t.dh.entriesPlusValid)
}

func (t *DirHandleTest) FetchEntryCoresNonZeroOffsetFetchesIfCacheInvalid() {
	t.dh.entriesPlusValid = false
	_, err := storageutil.CreateObject(t.ctx, t.bucket, "testDir/fetchThis", nil)
	AssertEq(nil, err)
	op := &fuseops.ReadDirPlusOp{
		ReadDirOp: fuseops.ReadDirOp{Offset: 1},
	}

	cores, err := t.dh.FetchEntryCores(t.ctx, op)

	AssertEq(nil, err)
	AssertEq(1, len(cores))
	entry, ok := cores[inode.NewFileName(t.dh.in.Name(), "fetchThis")]
	AssertTrue(ok, "Core for fetchThis not found")
	t.validateFileType(entry, "fetchThis", "testDir/fetchThis")
}

func (t *DirHandleTest) ReadDirPlusResponseForNoFile() {
	op := &fuseops.ReadDirPlusOp{
		ReadDirOp: fuseops.ReadDirOp{Dst: make([]byte, 1024)},
	}
	var gcsEntries []fuseutil.DirentPlus
	localFileEntries := make(map[string]fuseutil.DirentPlus)

	err := t.dh.ReadDirPlus(op, gcsEntries, localFileEntries)

	AssertEq(nil, err)
	AssertEq(0, op.BytesRead)
	AssertTrue(t.dh.entriesPlusValid)
	AssertEq(0, len(t.dh.entriesPlus))
}

func (t *DirHandleTest) ReadDirPlusSameNameLocalAndGCSFile() {
	op := &fuseops.ReadDirPlusOp{
		ReadDirOp: fuseops.ReadDirOp{Dst: make([]byte, 1024)},
	}
	gcsFile := t.createTestDirentPlus(fuseutil.DT_File, 1001, 10)
	localFile := t.createTestDirentPlus(fuseutil.DT_File, 1002, 0)
	gcsEntriesPlus := []fuseutil.DirentPlus{gcsFile}
	localFileEntriesPlus := map[string]fuseutil.DirentPlus{testDirentName: localFile}

	err := t.dh.ReadDirPlus(op, gcsEntriesPlus, localFileEntriesPlus)

	AssertEq(nil, err)
	AssertEq(1, len(t.dh.entriesPlus))
	t.validateEntryPlus(t.dh.entriesPlus[0], testDirentName, fuseutil.DT_File, 1001)
}

func (t *DirHandleTest) ReadDirPlusSameNameLocalFileAndGCSDirectory() {
	op := &fuseops.ReadDirPlusOp{
		ReadDirOp: fuseops.ReadDirOp{Dst: make([]byte, 1024)},
	}
	gcsDir := t.createTestDirentPlus(fuseutil.DT_Directory, 1001, 0)
	gcsEntriesPlus := []fuseutil.DirentPlus{gcsDir}
	localFile := t.createTestDirentPlus(fuseutil.DT_File, 2001, 20)
	localFileEntriesPlus := map[string]fuseutil.DirentPlus{testDirentName: localFile}

	err := t.dh.ReadDirPlus(op, gcsEntriesPlus, localFileEntriesPlus)

	AssertEq(nil, err)
	AssertEq(2, len(t.dh.entriesPlus))
	t.validateEntryPlus(t.dh.entriesPlus[0], testDirentName, fuseutil.DT_Directory, 1001)
	t.validateEntryPlus(t.dh.entriesPlus[1], testDirentName+inode.ConflictingFileNameSuffix, fuseutil.DT_File, 2001)
	AssertEq(t.dh.entriesPlus[1].Dirent.Offset, t.dh.entriesPlus[0].Dirent.Offset+1)
}
