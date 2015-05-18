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
var _ TearDownInterface = &DirTest{}

func init() { RegisterTestSuite(&DirTest{}) }

func (t *DirTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
	t.clock.SetTime(time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local))
	t.bucket = gcsfake.NewFakeBucket(&t.clock, "some_bucket")

	// Create the inode. No implicit dirs by default.
	t.resetInode(false)
}

func (t *DirTest) TearDown() {
	t.in.Unlock()
}

func (t *DirTest) resetInode(implicitDirs bool) {
	if t.in != nil {
		t.in.Unlock()
	}

	t.in = inode.NewDirInode(
		inodeID,
		inodeName,
		implicitDirs,
		typeCacheTTL,
		t.bucket,
		&t.clock)

	t.in.Lock()
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *DirTest) ID() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) Name() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) LookupCount() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) Attributes() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) LookUpChild_NonExistent() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) LookUpChild_FileOnly() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) LookUpChild_DirOnly() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) LookUpChild_ImplicitDirOnly_Disabled() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) LookUpChild_ImplicitDirOnly_Enabled() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) LookUpChild_FileAndDir() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) LookUpChild_FileAndDirAndImplicitDir_Disabled() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) LookUpChild_FileAndDirAndImplicitDir_Enabled() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) LookUpChild_TypeCaching() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) ReadEntries_Empty() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) ReadEntries_NonEmpty_ImplicitDirsDisabled() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) ReadEntries_NonEmpty_ImplicitDirsEnabled() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) ReadEntries_LotsOfEntries() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) ReadEntries_TypeCaching() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) CreateChildFile_DoesntExist() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) CreateChildFile_Exists() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) CreateChildFile_TypeCaching() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) CreateChildDir_DoesntExist() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) CreateChildDir_Exists() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) CreateChildDir_TypeCaching() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) DeleteChildFile_DoesntExist() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) DeleteChildFile_Exists() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) DeleteChildFile_TypeCaching() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) DeleteChildDir_DoesntExist() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) DeleteChildDir_Exists() {
	AssertTrue(false, "TODO")
}

func (t *DirTest) DeleteChildDir_TypeCaching() {
	AssertTrue(false, "TODO")
}
