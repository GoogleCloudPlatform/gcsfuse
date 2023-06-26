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

package fs_test

import (
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/fs"
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

type DirHandleTest struct {
	ctx    context.Context
	bucket gcsx.SyncerBucket
	clock  timeutil.SimulatedClock
	dh     *fs.DirHandle
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
	t.dh = fs.NewDirHandle(in, implicitDirs)
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
// fetchEntriesAsync will return 0 entries for empty directory
func (t *DirHandleTest) FetchAsyncEntries_EmptyDir() {
	t.createDirHandle(false, false, dirInodeName)
	t.dh.FetchEntriesAsync(fuseops.RootInodeID, true)

	AssertEq(0, len(t.dh.Entries))
	AssertEq(true, t.dh.EntriesValid)
	t.resetDirHandle()
}

// Directory Structure Used
// foo       --Directory
// foo/bar   --File
// fetchEntriesAsync will return 1 entry for directory with 1 file
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
	t.dh.FetchEntriesAsync(fuseops.RootInodeID, true)

	AssertEq(1, len(t.dh.Entries))
	AssertEq(fileUnderDir, t.dh.Entries[0].Name)
	AssertEq(true, t.dh.EntriesValid)
	t.resetDirHandle()
}

// Directory Structure Used
// foo              --Directory
// foo/baz          --Implicit Directory
// foo/baz/bar      --file
// fetchEntriesAsync will return 1 entry for implicit directory if flag is set to true
func (t *DirHandleTest) FetchAsyncEntries_ImplicitDir_FlagTrue() {
	err := t.createImplicitDirDefinedByFile()
	AssertEq(nil, err)

	//implicit-dirs flag set to true
	t.createDirHandle(true, false, dirInodeName)
	t.dh.FetchEntriesAsync(fuseops.RootInodeID, true)

	AssertEq(1, len(t.dh.Entries))
	AssertEq(implicitDirName, t.dh.Entries[0].Name)
	AssertEq(true, t.dh.EntriesValid)
	t.resetDirHandle()
}

// Same directory structure as above.
// fetchEntriesAsync will return 0 entry for implicit directory if flag is set to false
func (t *DirHandleTest) FetchAsyncEntries_ImplicitDir_FlagFalse() {
	err := t.createImplicitDirDefinedByFile()
	AssertEq(nil, err)

	//implicit-dirs flag set to false
	t.createDirHandle(false, false, dirInodeName)
	t.dh.FetchEntriesAsync(fuseops.RootInodeID, true)

	AssertEq(0, len(t.dh.Entries))
	AssertEq(true, t.dh.EntriesValid)
	t.resetDirHandle()
}
