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
	"math"
	"os"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/googlecloudplatform/gcsfuse/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/lease"
	"github.com/googlecloudplatform/gcsfuse/timeutil"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsfake"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	. "github.com/jacobsa/ogletest"
)

func TestFile(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const fileInodeID = 17
const fileInodeName = "foo/bar"

type FileTest struct {
	ctx    context.Context
	bucket gcs.Bucket
	leaser lease.FileLeaser
	clock  timeutil.SimulatedClock

	initialContents string
	backingObj      *gcs.Object

	in *inode.FileInode
}

var _ SetUpInterface = &FileTest{}
var _ TearDownInterface = &FileTest{}

func init() { RegisterTestSuite(&FileTest{}) }

func (t *FileTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
	t.clock.SetTime(time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local))
	t.leaser = lease.NewFileLeaser("", math.MaxInt64)
	t.bucket = gcsfake.NewFakeBucket(&t.clock, "some_bucket")

	// Set up the backing object.
	var err error

	t.initialContents = "taco"
	t.backingObj, err = gcsutil.CreateObject(
		t.ctx,
		t.bucket,
		fileInodeName,
		t.initialContents)

	AssertEq(nil, err)

	// Create the inode.
	t.in = inode.NewFileInode(
		fileInodeID,
		t.backingObj,
		math.MaxUint64, // GCS chunk size
		false,          // Support nlink
		t.bucket,
		t.leaser,
		&t.clock)

	t.in.Lock()
}

func (t *FileTest) TearDown() {
	t.in.Unlock()
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *FileTest) ID() {
	ExpectEq(fileInodeID, t.in.ID())
}

func (t *FileTest) Name() {
	ExpectEq(fileInodeName, t.in.Name())
}

func (t *FileTest) InitialSourceGeneration() {
	ExpectEq(t.backingObj.Generation, t.in.SourceGeneration())
}

func (t *FileTest) InitialAttributes() {
	attrs, err := t.in.Attributes(t.ctx)
	AssertEq(nil, err)

	ExpectEq(len(t.initialContents), attrs.Size)
	ExpectEq(1, attrs.Nlink)
	ExpectEq(os.FileMode(0700), attrs.Mode)
	ExpectThat(attrs.Mtime, timeutil.TimeEq(t.backingObj.Updated))
}

func (t *FileTest) Read() {
	// TODO(jacobsa): Test various ranges in a table-driven test. Make sure no
	// EOF.
	AssertTrue(false, "TODO")
}

func (t *FileTest) Write() {
	// TODO(jacobsa): Check attributes and read afterward.
	AssertTrue(false, "TODO")
}

func (t *FileTest) Truncate() {
	// TODO(jacobsa): Check attributes and read afterward.
	AssertTrue(false, "TODO")
}

func (t *FileTest) Sync_NotClobbered() {
	// TODO(jacobsa): Check generation and bucket afterward.
	AssertTrue(false, "TODO")
}

func (t *FileTest) Sync_Clobbered() {
	AssertTrue(false, "TODO")
}
