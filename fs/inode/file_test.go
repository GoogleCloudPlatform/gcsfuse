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
	"fmt"
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
	AssertEq("taco", t.initialContents)

	// Make several reads, checking the expected contents. We should never get an
	// EOF error, since fuseops.ReadFileOp is not supposed to see those.
	testCases := []struct {
		offset   int64
		size     int
		expected string
	}{
		{0, 1, "t"},
		{0, 2, "ta"},
		{0, 3, "tac"},
		{0, 4, "taco"},
		{0, 5, "taco"},

		{1, 1, "a"},
		{1, 2, "ac"},
		{1, 3, "aco"},
		{1, 4, "aco"},

		{3, 1, "o"},
		{3, 2, "o"},

		// Empty ranges
		{0, 0, ""},
		{3, 0, ""},
		{4, 0, ""},
		{4, 1, ""},
		{5, 0, ""},
		{5, 1, ""},
		{5, 2, ""},
	}

	for _, tc := range testCases {
		desc := fmt.Sprintf("offset: %d, size: %d", tc.offset, tc.size)

		data, err := t.in.Read(t.ctx, tc.offset, tc.size)
		AssertEq(nil, err, "%s", desc)
		ExpectEq(tc.expected, string(data), "%s", desc)
	}
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
