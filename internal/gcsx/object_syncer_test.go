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

package gcsx

import (
	"errors"
	"io"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsfake"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
	"golang.org/x/net/context"
)

func TestObjectSyncer(t *testing.T) { RunTests(t) }

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
const appendThreshold = int64(len(srcObjectContents))

type ObjectSyncerTest struct {
	ctx context.Context

	fullCreator   fakeObjectCreator
	appendCreator fakeObjectCreator

	bucket gcs.Bucket
	syncer ObjectSyncer
	clock  timeutil.SimulatedClock

	srcObject *gcs.Object
	content   TempFile
}

var _ SetUpInterface = &ObjectSyncerTest{}

func init() { RegisterTestSuite(&ObjectSyncerTest{}) }

func (t *ObjectSyncerTest) SetUp(ti *TestInfo) {
	var err error
	t.ctx = ti.Ctx

	// Set up dependencies.
	t.bucket = gcsfake.NewFakeBucket(&t.clock, "some_bucket")
	t.syncer = newObjectSyncer(
		appendThreshold,
		&t.fullCreator,
		&t.appendCreator)

	t.clock.SetTime(time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local))

	// Set up a source object.
	t.srcObject, err = t.bucket.CreateObject(
		t.ctx,
		&gcs.CreateObjectRequest{
			Name:     "foo",
			Contents: strings.NewReader(srcObjectContents),
		})

	AssertEq(nil, err)

	// Wrap a TempFile around it.
	t.content, err = NewTempFile(
		strings.NewReader(srcObjectContents),
		&t.clock)

	AssertEq(nil, err)

	// Return errors from the fakes by default.
	t.fullCreator.err = errors.New("Fake error")
	t.appendCreator.err = errors.New("Fake error")
}

func (t *ObjectSyncerTest) call() (o *gcs.Object, err error) {
	o, err = t.syncer.SyncObject(t.ctx, t.srcObject, t.content)
	return
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ObjectSyncerTest) NotDirty() {
	// Call
	o, err := t.call()

	AssertEq(nil, err)
	ExpectEq(nil, o)

	// Neither creater should have been called.
	ExpectFalse(t.fullCreator.called)
	ExpectFalse(t.appendCreator.called)
}

func (t *ObjectSyncerTest) SmallerThanSource() {
	// Truncate downward.
	err := t.content.Truncate(int64(len(srcObjectContents) - 1))
	AssertEq(nil, err)

	// The full creator should be called.
	t.call()

	ExpectTrue(t.fullCreator.called)
	ExpectFalse(t.appendCreator.called)
}

func (t *ObjectSyncerTest) SameSizeAsSource() {
	// Dirty a byte without changing the length.
	_, err := t.content.WriteAt(
		[]byte("a"),
		int64(len(srcObjectContents)-1))

	AssertEq(nil, err)

	// The full creator should be called.
	t.call()

	ExpectTrue(t.fullCreator.called)
	ExpectFalse(t.appendCreator.called)
}

func (t *ObjectSyncerTest) LargerThanSource_ThresholdInSource() {
	var err error

	// Extend the length of the content.
	err = t.content.Truncate(int64(len(srcObjectContents) + 100))
	AssertEq(nil, err)

	// But dirty a byte within the initial content.
	_, err = t.content.WriteAt(
		[]byte("a"),
		int64(len(srcObjectContents)-1))

	AssertEq(nil, err)

	// The full creator should be called.
	t.call()

	ExpectTrue(t.fullCreator.called)
	ExpectFalse(t.appendCreator.called)
}

func (t *ObjectSyncerTest) SourceTooShortForAppend() {
	var err error

	// Recreate the syncer with a higher append threshold.
	t.syncer = newObjectSyncer(
		int64(len(srcObjectContents)+1),
		&t.fullCreator,
		&t.appendCreator)

	// Extend the length of the content.
	err = t.content.Truncate(int64(len(srcObjectContents) + 1))
	AssertEq(nil, err)

	// The full creator should be called.
	t.call()

	ExpectTrue(t.fullCreator.called)
	ExpectFalse(t.appendCreator.called)
}

func (t *ObjectSyncerTest) SourceComponentCountTooHigh() {
	var err error

	// Simulate a large component count.
	t.srcObject.ComponentCount = gcs.MaxComponentCount

	// Extend the length of the content.
	err = t.content.Truncate(int64(len(srcObjectContents) + 1))
	AssertEq(nil, err)

	// The full creator should be called.
	t.call()

	ExpectTrue(t.fullCreator.called)
	ExpectFalse(t.appendCreator.called)
}

func (t *ObjectSyncerTest) LargerThanSource_ThresholdAtEndOfSource() {
	var err error

	// Extend the length of the content.
	err = t.content.Truncate(int64(len(srcObjectContents) + 1))
	AssertEq(nil, err)

	// The append creator should be called.
	t.call()

	ExpectFalse(t.fullCreator.called)
	ExpectTrue(t.appendCreator.called)
}

func (t *ObjectSyncerTest) CallsFullCreator() {
	var err error
	AssertLt(2, t.srcObject.Size)

	// Truncate downward.
	err = t.content.Truncate(2)
	AssertEq(nil, err)

	// Call
	t.call()

	AssertTrue(t.fullCreator.called)
	ExpectEq(t.srcObject, t.fullCreator.srcObject)
	ExpectEq(srcObjectContents[:2], string(t.fullCreator.contents))
}

func (t *ObjectSyncerTest) FullCreatorFails() {
	var err error
	t.fullCreator.err = errors.New("taco")

	// Truncate downward.
	err = t.content.Truncate(2)
	AssertEq(nil, err)

	// Call
	_, err = t.call()

	ExpectThat(err, Error(HasSubstr("Create")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *ObjectSyncerTest) FullCreatorReturnsPreconditionError() {
	var err error
	t.fullCreator.err = &gcs.PreconditionError{}

	// Truncate downward.
	err = t.content.Truncate(2)
	AssertEq(nil, err)

	// Call
	_, err = t.call()

	ExpectEq(t.fullCreator.err, err)
}

func (t *ObjectSyncerTest) FullCreatorSucceeds() {
	var err error
	t.fullCreator.o = &gcs.Object{}
	t.fullCreator.err = nil

	// Truncate downward.
	err = t.content.Truncate(2)
	AssertEq(nil, err)

	// Call
	o, err := t.call()

	AssertEq(nil, err)
	ExpectEq(t.fullCreator.o, o)
}

func (t *ObjectSyncerTest) CallsAppendCreator() {
	var err error

	// Append some data.
	_, err = t.content.WriteAt([]byte("burrito"), int64(t.srcObject.Size))
	AssertEq(nil, err)

	// Call
	t.call()

	AssertTrue(t.appendCreator.called)
	ExpectEq(t.srcObject, t.appendCreator.srcObject)
	ExpectEq("burrito", string(t.appendCreator.contents))
}

func (t *ObjectSyncerTest) AppendCreatorFails() {
	var err error
	t.appendCreator.err = errors.New("taco")

	// Append some data.
	_, err = t.content.WriteAt([]byte("burrito"), int64(t.srcObject.Size))
	AssertEq(nil, err)

	// Call
	_, err = t.call()

	ExpectThat(err, Error(HasSubstr("Create")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *ObjectSyncerTest) AppendCreatorReturnsPreconditionError() {
	var err error
	t.appendCreator.err = &gcs.PreconditionError{}

	// Append some data.
	_, err = t.content.WriteAt([]byte("burrito"), int64(t.srcObject.Size))
	AssertEq(nil, err)

	// Call
	_, err = t.call()

	ExpectEq(t.appendCreator.err, err)
}

func (t *ObjectSyncerTest) AppendCreatorSucceeds() {
	var err error
	t.appendCreator.o = &gcs.Object{}
	t.appendCreator.err = nil

	// Append some data.
	_, err = t.content.WriteAt([]byte("burrito"), int64(t.srcObject.Size))
	AssertEq(nil, err)

	// Call
	o, err := t.call()

	AssertEq(nil, err)
	ExpectEq(t.appendCreator.o, o)
}
