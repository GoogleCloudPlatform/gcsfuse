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

func TestSyncer(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// fakeObjectCreator
////////////////////////////////////////////////////////////////////////

// An objectCreator that records the arguments it is called with, returning
// canned results.
type fakeObjectCreator struct {
	called bool

	// Supplied arguments
	srcObject *gcs.Object
	mtime     time.Time
	contents  []byte

	// Canned results
	o   *gcs.Object
	err error
}

func (oc *fakeObjectCreator) Create(
	ctx context.Context,
	srcObject *gcs.Object,
	mtime time.Time,
	r io.Reader) (o *gcs.Object, err error) {
	// Have we been called more than once?
	AssertFalse(oc.called)
	oc.called = true

	// Record args.
	oc.srcObject = srcObject
	oc.mtime = mtime
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

type SyncerTest struct {
	ctx context.Context

	fullCreator   fakeObjectCreator
	appendCreator fakeObjectCreator

	bucket gcs.Bucket
	syncer Syncer
	clock  timeutil.SimulatedClock

	srcObject *gcs.Object
	content   TempFile
}

var _ SetUpInterface = &SyncerTest{}

func init() { RegisterTestSuite(&SyncerTest{}) }

func (t *SyncerTest) SetUp(ti *TestInfo) {
	var err error
	t.ctx = ti.Ctx

	// Set up dependencies.
	t.bucket = gcsfake.NewFakeBucket(&t.clock, "some_bucket")
	t.syncer = newSyncer(
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
		dummyReadCloser{strings.NewReader(srcObjectContents)},
		"",
		&t.clock)

	AssertEq(nil, err)

	// Return errors from the fakes by default.
	t.fullCreator.err = errors.New("Fake error")
	t.appendCreator.err = errors.New("Fake error")
}

func (t *SyncerTest) call() (o *gcs.Object, err error) {
	o, err = t.syncer.SyncObject(t.ctx, t.srcObject, t.content)
	return
}

type dummyReadCloser struct {
	io.Reader
}

func (rc dummyReadCloser) Close() error {
	return nil
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *SyncerTest) NotDirty() {
	// Call
	o, err := t.call()

	AssertEq(nil, err)
	ExpectEq(nil, o)

	// Neither creater should have been called.
	ExpectFalse(t.fullCreator.called)
	ExpectFalse(t.appendCreator.called)
}

func (t *SyncerTest) SmallerThanSource() {
	// Truncate downward.
	err := t.content.Truncate(int64(len(srcObjectContents) - 1))
	AssertEq(nil, err)

	// The full creator should be called.
	t.call()

	ExpectTrue(t.fullCreator.called)
	ExpectFalse(t.appendCreator.called)
}

func (t *SyncerTest) SameSizeAsSource() {
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

func (t *SyncerTest) LargerThanSource_ThresholdInSource() {
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

func (t *SyncerTest) SourceTooShortForAppend() {
	var err error

	// Recreate the syncer with a higher append threshold.
	t.syncer = newSyncer(
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

func (t *SyncerTest) SourceComponentCountTooHigh() {
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

func (t *SyncerTest) LargerThanSource_ThresholdAtEndOfSource() {
	var err error

	// Extend the length of the content.
	err = t.content.Truncate(int64(len(srcObjectContents) + 1))
	AssertEq(nil, err)

	// The append creator should be called.
	t.call()

	ExpectFalse(t.fullCreator.called)
	ExpectTrue(t.appendCreator.called)
}

func (t *SyncerTest) CallsFullCreator() {
	var err error
	AssertLt(2, t.srcObject.Size)

	// Ready the content.
	err = t.content.Truncate(2)
	AssertEq(nil, err)

	mtime := time.Now().Add(123 * time.Second)
	t.content.SetMtime(mtime)

	// Call
	t.call()

	AssertTrue(t.fullCreator.called)
	ExpectEq(t.srcObject, t.fullCreator.srcObject)
	ExpectThat(t.fullCreator.mtime, timeutil.TimeEq(mtime.UTC()))
	ExpectEq(srcObjectContents[:2], string(t.fullCreator.contents))
}

func (t *SyncerTest) FullCreatorFails() {
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

func (t *SyncerTest) FullCreatorReturnsPreconditionError() {
	var err error
	t.fullCreator.err = &gcs.PreconditionError{}

	// Truncate downward.
	err = t.content.Truncate(2)
	AssertEq(nil, err)

	// Call
	_, err = t.call()

	ExpectEq(t.fullCreator.err, err)
}

func (t *SyncerTest) FullCreatorSucceeds() {
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

func (t *SyncerTest) CallsAppendCreator() {
	var err error

	// Append some data.
	_, err = t.content.WriteAt([]byte("burrito"), int64(t.srcObject.Size))
	AssertEq(nil, err)

	// Set up an expected mtime.
	mtime := time.Now().Add(123 * time.Second)
	t.content.SetMtime(mtime)

	// Call
	t.call()

	AssertTrue(t.appendCreator.called)
	ExpectEq(t.srcObject, t.appendCreator.srcObject)
	ExpectThat(t.appendCreator.mtime, timeutil.TimeEq(mtime.UTC()))
	ExpectEq("burrito", string(t.appendCreator.contents))
}

func (t *SyncerTest) AppendCreatorFails() {
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

func (t *SyncerTest) AppendCreatorReturnsPreconditionError() {
	var err error
	t.appendCreator.err = &gcs.PreconditionError{}

	// Append some data.
	_, err = t.content.WriteAt([]byte("burrito"), int64(t.srcObject.Size))
	AssertEq(nil, err)

	// Call
	_, err = t.call()

	ExpectEq(t.appendCreator.err, err)
}

func (t *SyncerTest) AppendCreatorSucceeds() {
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
