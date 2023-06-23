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

package fs_test

import (
	"os"
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
const typeCacheTTL = time.Second
const uid = 123
const gid = 456

type AsyncFetchTest struct {
	ctx    context.Context
	bucket gcsx.SyncerBucket
	clock  timeutil.SimulatedClock
	dh     *fs.DirHandle
}

var _ SetUpInterface = &AsyncFetchTest{}
var _ TearDownInterface = &AsyncFetchTest{}

func init() {
	RegisterTestSuite(&AsyncFetchTest{})
}

func (t *AsyncFetchTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
	t.clock.SetTime(time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local))
	bucket := gcsfake.NewFakeBucket(&t.clock, "some_bucket")
	t.bucket = gcsx.NewSyncerBucket(
		1, // Append threshold
		".gcsfuse_tmp/",
		bucket)
	// Create the inode and directory handle. No implicit dirs by default.
}

func TestAsync(t *testing.T) {
	RunTests(t)
}

func (t *AsyncFetchTest) TearDown() {
}

func (t *AsyncFetchTest) createDirHandle(implicitDirs bool, enableNonexistentTypecache bool, dirInodeName string) {
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

func (t *AsyncFetchTest) resetDirHandle() {
	t.dh = nil
}

func (t *AsyncFetchTest) FetchAsyncEntries_EmptyDir() {
	//directory structure : foo/ readir should return 0 entries
	t.createDirHandle(false, false, dirInodeName)
	t.dh.FetchEntriesAsync(fuseops.RootInodeID, true)
	AssertEq(len(t.dh.Entries), 0)
	AssertEq(t.dh.EntriesValid, true)
	t.resetDirHandle()
}

func (t *AsyncFetchTest) FetchAsyncEntries_NonEmptyDir() {
	//directory structure : foo/bar fetchEntriesAsync should return 1 entry
	contents := "Non-empty dir"
	_, err := t.bucket.CreateObject(
		t.ctx,
		&gcs.CreateObjectRequest{
			Name:     "foo/bar",
			Contents: strings.NewReader(contents),
		})
	AssertEq(err, nil)
	t.createDirHandle(false, false, dirInodeName)
	t.dh.FetchEntriesAsync(fuseops.RootInodeID, true)
	AssertEq(len(t.dh.Entries), 1)
	AssertEq(t.dh.Entries[0].Name, "bar")
	AssertEq(t.dh.EntriesValid, true)
	t.resetDirHandle()
}

func (t *AsyncFetchTest) FetchAsyncEntries_ImplicitDir() {
	//directory structure : foo/bar/lorem.txt
	contents := "Implicit dir"
	_, err := t.bucket.CreateObject(
		t.ctx,
		&gcs.CreateObjectRequest{
			Name:     "foo/bar/lorem.txt",
			Contents: strings.NewReader(contents),
		})
	AssertEq(err, nil)
	//implicit dir set to true. entry for bar is returned
	t.createDirHandle(true, false, "foo/")
	t.dh.FetchEntriesAsync(fuseops.RootInodeID, true)
	AssertEq(len(t.dh.Entries), 1)
	AssertEq(t.dh.Entries[0].Name, "bar")
	AssertEq(true, t.dh.EntriesValid)
	t.resetDirHandle()
	//implicit dir flag set to false. entry for bar will not be returned
	t.createDirHandle(false, false, "foo/")
	t.dh.FetchEntriesAsync(fuseops.RootInodeID, true)
	AssertEq(len(t.dh.Entries), 0)
	AssertEq(true, t.dh.EntriesValid)
	t.resetDirHandle()
}
