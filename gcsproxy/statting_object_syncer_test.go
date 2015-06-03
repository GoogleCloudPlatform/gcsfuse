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

package gcsproxy

import (
	"errors"
	"io"
	"io/ioutil"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/lease"
	"github.com/googlecloudplatform/gcsfuse/mutable"
	"github.com/googlecloudplatform/gcsfuse/timeutil"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsfake"
	. "github.com/jacobsa/ogletest"
	"golang.org/x/net/context"
)

func TestStattingObjectSyncer(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// fakeObjectCreator
////////////////////////////////////////////////////////////////////////

// An objectCreator that records the arguments it is called with, returning
// canned results.
type fakeObjectCreator struct {
	called bool

	// Supplied arguments
	srcObject *gcs.Object
	contents  []byte

	// Canned results
	o   *gcs.Object
	err error
}

func (oc *fakeObjectCreator) Create(
	ctx context.Context,
	srcObject *gcs.Object,
	r io.Reader) (o *gcs.Object, err error) {
	// Have we been called more than once?
	AssertFalse(oc.called)
	oc.called = true

	// Record args.
	oc.srcObject = srcObject
	oc.contents, err = ioutil.ReadAll(r)
	AssertEq(nil, err)

	// Return results.
	o, err = oc.o, oc.err
	return
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const srcObjectContents = "taco"

type StattingObjectSyncerTest struct {
	ctx context.Context

	fullCreator   fakeObjectCreator
	appendCreator fakeObjectCreator

	bucket gcs.Bucket
	leaser lease.FileLeaser
	syncer ObjectSyncer
	clock  timeutil.SimulatedClock

	srcObject *gcs.Object
	content   mutable.Content
}

var _ SetUpInterface = &StattingObjectSyncerTest{}

func init() { RegisterTestSuite(&StattingObjectSyncerTest{}) }

func (t *StattingObjectSyncerTest) SetUp(ti *TestInfo) {
	var err error
	t.ctx = ti.Ctx

	// Set up dependencies.
	t.bucket = gcsfake.NewFakeBucket(&t.clock, "some_bucket")
	t.leaser = lease.NewFileLeaser("", math.MaxInt32, math.MaxInt32)
	t.syncer = createStattingObjectSyncer(&t.fullCreator, &t.appendCreator)
	t.clock.SetTime(time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local))

	// Set up a source object.
	t.srcObject, err = t.bucket.CreateObject(
		t.ctx,
		&gcs.CreateObjectRequest{
			Name:     "foo",
			Contents: strings.NewReader(srcObjectContents),
		})

	AssertEq(nil, err)

	// Wrap a mutable.Content around it.
	t.content = mutable.NewContent(
		NewReadProxy(
			t.srcObject,
			nil,            // Initial read lease
			math.MaxUint64, // Chunk size
			t.leaser,
			t.bucket),
		&t.clock)
}

func (t *StattingObjectSyncerTest) call() (
	rl lease.ReadLease, o *gcs.Object, err error) {
	err = errors.New("TODO")
	return
}

func (t *StattingObjectSyncerTest) serveSyncFull(
	ctx context.Context,
	srcObject *gcs.Object,
	r io.Reader) (o *gcs.Object, err error) {
	err = errors.New("TODO: serveSyncFull")
	return
}

func (t *StattingObjectSyncerTest) serveSyncAppend(
	ctx context.Context,
	srcObject *gcs.Object,
	r io.Reader) (o *gcs.Object, err error) {
	err = errors.New("TODO: serveSyncAppend")
	return
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *StattingObjectSyncerTest) NotDirty() {
	// Call
	rl, o, err := t.call()

	AssertEq(nil, err)
	ExpectEq(nil, rl)
	ExpectEq(nil, o)
}

func (t *StattingObjectSyncerTest) SmallerThanSource() {
	AssertTrue(false, "TODO")
}

func (t *StattingObjectSyncerTest) SameSizeAsSource() {
	AssertTrue(false, "TODO")
}

func (t *StattingObjectSyncerTest) LargerThanSource_ThresholdInSource() {
	AssertTrue(false, "TODO")
}

func (t *StattingObjectSyncerTest) LargerThanSource_ThresholdAtEndOfSource() {
	AssertTrue(false, "TODO")
}

func (t *StattingObjectSyncerTest) SyncFullFails() {
	AssertTrue(false, "TODO")
}

func (t *StattingObjectSyncerTest) SyncFullSucceeds() {
	AssertTrue(false, "TODO")
}

func (t *StattingObjectSyncerTest) SyncAppendFails() {
	AssertTrue(false, "TODO")
}

func (t *StattingObjectSyncerTest) SyncAppendSucceeds() {
	AssertTrue(false, "TODO")
}
