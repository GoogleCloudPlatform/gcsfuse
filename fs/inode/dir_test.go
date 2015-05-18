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

package inode_test

import (
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/googlecloudplatform/gcsfuse/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/timeutil"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsfake"
	. "github.com/jacobsa/ogletest"
)

func TestDir(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const inodeID = 17
const inodeName = "foo/bar/"
const typeCacheTTL = time.Second

type DirTest struct {
	ctx    context.Context
	bucket gcs.Bucket
	clock  timeutil.SimulatedClock

	in *inode.DirInode
}

var _ SetUpInterface = &DirTest{}

func init() { RegisterTestSuite(&DirTest{}) }

func (t *DirTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
	t.clock.SetTime(time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local))
	t.bucket = gcsfake.NewFakeBucket(&t.clock, "some_bucket")

	// Create the inode. No implicit dirs by default.
	t.resetInode(false)
}

func (t *DirTest) resetInode(implicitDirs bool) {
	t.in = inode.NewDirInode(
		inodeID,
		inodeName,
		implicitDirs,
		typeCacheTTL,
		t.bucket,
		&t.clock)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *DirTest) DoesFoo() {
	AssertTrue(false, "TODO")
}
