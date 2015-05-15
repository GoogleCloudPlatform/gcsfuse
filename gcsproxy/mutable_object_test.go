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

package gcsproxy_test

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/googlecloudplatform/gcsfuse/gcsproxy"
	"github.com/googlecloudplatform/gcsfuse/lease"
	"github.com/googlecloudplatform/gcsfuse/timeutil"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/mock_gcs"
	. "github.com/jacobsa/oglematchers"
	"github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"golang.org/x/net/context"
)

func TestMutableObject(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func nameIs(name string) Matcher {
	return NewMatcher(
		func(candidate interface{}) error {
			var actual string
			switch typed := candidate.(type) {
			case *gcs.CreateObjectRequest:
				actual = typed.Name

			case *gcs.StatObjectRequest:
				actual = typed.Name

			case *gcs.ReadObjectRequest:
				actual = typed.Name

			default:
				panic(fmt.Sprintf("Unhandled type: %v", reflect.TypeOf(candidate)))
			}

			if actual != name {
				return fmt.Errorf("which has name %v", actual)
			}

			return nil
		},
		fmt.Sprintf("Name is: %s", name))
}

func contentsAre(s string) Matcher {
	return NewMatcher(
		func(candidate interface{}) error {
			// Snarf the contents.
			req := candidate.(*gcs.CreateObjectRequest)
			contents, err := ioutil.ReadAll(req.Contents)
			if err != nil {
				panic(err)
			}

			// Compare
			if string(contents) != s {
				return errors.New("")
			}

			return nil
		},
		fmt.Sprintf("Object contents are: %s", s))
}

func generationIs(g int64) Matcher {
	pred := func(c interface{}) error {
		switch req := c.(type) {
		case *gcs.CreateObjectRequest:
			if req.GenerationPrecondition == nil {
				return errors.New("which has a nil GenerationPrecondition field.")
			}

			if *req.GenerationPrecondition != g {
				return fmt.Errorf(
					"which has *GenerationPrecondition == %v",
					*req.GenerationPrecondition)
			}

		case *gcs.ReadObjectRequest:
			if req.Generation != g {
				return fmt.Errorf("which has Generation == %v", req.Generation)
			}

		default:
			panic(fmt.Sprintf("Unknown type: %v", reflect.TypeOf(c)))
		}

		return nil
	}

	return NewMatcher(
		pred,
		fmt.Sprintf("*GenerationPrecondition == %v", g))
}

type errorReadCloser struct {
	wrapped io.Reader
	err     error
}

func (ec *errorReadCloser) Read(p []byte) (n int, err error) {
	return ec.wrapped.Read(p)
}

func (ec *errorReadCloser) Close() error {
	return ec.err
}

////////////////////////////////////////////////////////////////////////
// Invariant-checking mutable object
////////////////////////////////////////////////////////////////////////

// A wrapper around MutableObject that calls CheckInvariants whenever
// invariants should hold. For catching logic errors early in the test.
type checkingMutableObject struct {
	ctx     context.Context
	wrapped *gcsproxy.MutableObject
}

func (mo *checkingMutableObject) SourceGeneration() int64 {
	mo.wrapped.CheckInvariants()
	defer mo.wrapped.CheckInvariants()
	return mo.wrapped.SourceGeneration()
}

func (mo *checkingMutableObject) Stat(
	needClobbered bool) (gcsproxy.StatResult, error) {
	mo.wrapped.CheckInvariants()
	defer mo.wrapped.CheckInvariants()
	return mo.wrapped.Stat(mo.ctx, needClobbered)
}

func (mo *checkingMutableObject) ReadAt(b []byte, o int64) (int, error) {
	mo.wrapped.CheckInvariants()
	defer mo.wrapped.CheckInvariants()
	return mo.wrapped.ReadAt(mo.ctx, b, o)
}

func (mo *checkingMutableObject) WriteAt(b []byte, o int64) (int, error) {
	mo.wrapped.CheckInvariants()
	defer mo.wrapped.CheckInvariants()
	return mo.wrapped.WriteAt(mo.ctx, b, o)
}

func (mo *checkingMutableObject) Truncate(n int64) error {
	mo.wrapped.CheckInvariants()
	defer mo.wrapped.CheckInvariants()
	return mo.wrapped.Truncate(mo.ctx, n)
}

func (mo *checkingMutableObject) Sync() error {
	mo.wrapped.CheckInvariants()
	defer mo.wrapped.CheckInvariants()
	return mo.wrapped.Sync(mo.ctx)
}

func (mo *checkingMutableObject) Destroy() {
	mo.wrapped.CheckInvariants()
	defer mo.wrapped.CheckInvariants()
	mo.wrapped.Destroy()
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const initialContentsLen = 11

var initialContents = strings.Repeat("a", initialContentsLen)

type MutableObjectTest struct {
	src    gcs.Object
	clock  timeutil.SimulatedClock
	bucket mock_gcs.MockBucket
	mo     checkingMutableObject
}

var _ SetUpInterface = &MutableObjectTest{}

func init() { RegisterTestSuite(&MutableObjectTest{}) }

func (t *MutableObjectTest) SetUp(ti *TestInfo) {
	t.src = gcs.Object{
		Name:       "some/object",
		Generation: 123,
		Size:       uint64(initialContentsLen),
		Updated:    time.Date(2001, 2, 3, 4, 5, 0, 0, time.Local),
	}

	t.bucket = mock_gcs.NewMockBucket(ti.MockController, "bucket")

	// Set up a fixed, non-zero time.
	t.clock.SetTime(time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local))

	t.mo.ctx = ti.Ctx
	t.mo.wrapped = gcsproxy.NewMutableObject(
		math.MaxUint64, // Disable chunking
		&t.src,
		t.bucket,
		lease.NewFileLeaser("", math.MaxInt64),
		&t.clock)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *MutableObjectTest) InitialSourceGeneration() {
	ExpectEq(t.src.Generation, t.mo.SourceGeneration())
}

func (t *MutableObjectTest) Read_CallsNewReader() {
	// NewReader
	ExpectCall(t.bucket, "NewReader")(
		Any(),
		AllOf(nameIs(t.src.Name), generationIs(t.src.Generation))).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// ReadAt
	t.mo.ReadAt([]byte{}, 0)
}

func (t *MutableObjectTest) Read_NewReaderFails() {
	// NewReader
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// ReadAt
	_, err := t.mo.ReadAt([]byte{}, 0)

	ExpectThat(err, Error(HasSubstr("NewReader")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *MutableObjectTest) Read_ReadError() {
	// NewReader -- return a reader that returns an error after the first byte.
	rc := ioutil.NopCloser(
		iotest.TimeoutReader(
			iotest.OneByteReader(
				strings.NewReader("aaa"))))

	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(rc, nil))

	// ReadAt
	_, err := t.mo.ReadAt([]byte{}, 0)

	ExpectThat(err, Error(HasSubstr("Copy:")))
	ExpectThat(err, Error(HasSubstr("timeout")))
}

func (t *MutableObjectTest) Read_CloseError() {
	// NewReader -- return a ReadCloser that will fail to close.
	rc := &errorReadCloser{
		wrapped: strings.NewReader(initialContents),
		err:     errors.New("taco"),
	}

	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(rc, nil))

	// ReadAt
	_, err := t.mo.ReadAt([]byte{}, 0)

	ExpectThat(err, Error(HasSubstr("Close:")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *MutableObjectTest) Read_NewReaderSucceeds() {
	contents := "tacoburrito" + initialContents[len("tacoburrito"):]
	buf := make([]byte, 1024)
	var n int
	var err error

	// NewReader
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader(contents)), nil))

	// Read once.
	n, err = t.mo.ReadAt(buf[:4], 0)

	AssertEq(nil, err)
	AssertEq(4, n)
	ExpectEq("taco", string(buf[:n]))

	// The second read should work without calling NewReader again.
	n, err = t.mo.ReadAt(buf[:4], 2)

	AssertEq(nil, err)
	AssertEq(4, n)
	ExpectEq("cobu", string(buf[:n]))
}

func (t *MutableObjectTest) Write_CallsNewReader() {
	// NewReader
	ExpectCall(t.bucket, "NewReader")(
		Any(),
		AllOf(nameIs(t.src.Name), generationIs(t.src.Generation))).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// WriteAt
	t.mo.WriteAt([]byte{}, 0)
}

func (t *MutableObjectTest) WriteToEndOfObjectThenRead() {
	var buf []byte
	var n int
	var err error

	// NewReader
	s := initialContents
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader(s)), nil))

	// Extend the object by writing twice.
	n, err = t.mo.WriteAt([]byte("taco"), 0)
	AssertEq(nil, err)
	AssertEq(len("taco"), n)

	n, err = t.mo.WriteAt([]byte("burrito"), int64(len("taco")))
	AssertEq(nil, err)
	AssertEq(len("burrito"), n)

	// Read the whole thing.
	buf = make([]byte, 1024)
	n, err = t.mo.ReadAt(buf, 0)

	AssertEq(io.EOF, err)
	ExpectEq(len("tacoburrito"), n)
	ExpectEq("tacoburrito", string(buf[:n]))

	// Read a range in the middle.
	buf = make([]byte, 4)
	n, err = t.mo.ReadAt(buf, 3)

	AssertEq(nil, err)
	ExpectEq(4, n)
	ExpectEq("obur", string(buf[:n]))
}

func (t *MutableObjectTest) WritePastEndOfObjectThenRead() {
	var n int
	var err error
	var buf []byte

	// NewReader
	s := initialContents
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader(s)), nil))

	// Extend the object by writing past its end.
	n, err = t.mo.WriteAt([]byte("taco"), initialContentsLen+2)
	AssertEq(nil, err)
	AssertEq(len("taco"), n)

	// Read the whole thing.
	buf = make([]byte, 1024)
	n, err = t.mo.ReadAt(buf, 0)

	AssertEq(io.EOF, err)
	ExpectEq(initialContentsLen+2+len("taco"), n)
	ExpectEq(initialContents+"\x00\x00taco", string(buf[:n]))

	// Read a range in the middle.
	buf = make([]byte, 4)
	n, err = t.mo.ReadAt(buf, initialContentsLen+1)

	AssertEq(nil, err)
	ExpectEq(4, n)
	ExpectEq("\x00tac", string(buf[:n]))
}

func (t *MutableObjectTest) WriteWithinObjectThenRead() {
	var n int
	var err error
	var buf []byte

	// NewReader
	s := initialContents
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader(s)), nil))

	// Overwrite some data in the middle.
	n, err = t.mo.WriteAt([]byte("11"), 2)
	AssertEq(nil, err)
	AssertEq(len("11"), n)

	// Read the whole thing.
	buf = make([]byte, 1024)
	n, err = t.mo.ReadAt(buf, 0)

	AssertEq(io.EOF, err)
	ExpectEq(initialContentsLen, n)
	ExpectEq(initialContents[0:2]+"11"+initialContents[4:], string(buf[:n]))
}

func (t *MutableObjectTest) Truncate_CallsNewReader() {
	// NewReader
	ExpectCall(t.bucket, "NewReader")(
		Any(),
		AllOf(nameIs(t.src.Name), generationIs(t.src.Generation))).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Truncate
	t.mo.Truncate(17)
}

func (t *MutableObjectTest) GrowByTruncating() {
	var n int
	var err error
	var buf []byte

	// NewReader
	s := initialContents
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader(s)), nil))

	// Truncate
	err = t.mo.Truncate(int64(initialContentsLen + 4))
	AssertEq(nil, err)

	// Read the whole thing.
	buf = make([]byte, 1024)
	n, err = t.mo.ReadAt(buf, 0)

	AssertEq(io.EOF, err)
	ExpectEq(initialContentsLen+4, n)
	ExpectEq(s+"\x00\x00\x00\x00", string(buf[:n]))
}

func (t *MutableObjectTest) ShrinkByTruncating() {
	var n int
	var err error
	var buf []byte

	// NewReader
	s := initialContents
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader(s)), nil))

	// Truncate
	err = t.mo.Truncate(int64(initialContentsLen - 4))
	AssertEq(nil, err)

	// Read the whole thing.
	buf = make([]byte, 1024)
	n, err = t.mo.ReadAt(buf, 0)

	AssertEq(io.EOF, err)
	ExpectEq(initialContentsLen-4, n)
	ExpectEq(s[:initialContentsLen-4], string(buf[:n]))
}

func (t *MutableObjectTest) Sync_NoInteractions() {
	// There should be nothing to do.
	err := t.mo.Sync()

	AssertEq(nil, err)
	ExpectEq(t.src.Generation, t.mo.SourceGeneration())
}

func (t *MutableObjectTest) Sync_AfterReading() {
	contents := "taco" + initialContents[len("taco"):]
	buf := make([]byte, 1024)
	var n int
	var err error

	// Successfully read a fiew bytes.
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader(contents)), nil))

	n, err = t.mo.ReadAt(buf[:4], 0)

	AssertEq(nil, err)
	AssertEq(4, n)
	ExpectEq("taco", string(buf[:n]))

	// Sync should still need to do nothing.
	err = t.mo.Sync()

	AssertEq(nil, err)
	ExpectEq(t.src.Generation, t.mo.SourceGeneration())
}

func (t *MutableObjectTest) Sync_AfterWriting() {
	var n int
	var err error

	// Successfully write a few bytes.
	s := initialContents
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader(s)), nil))

	n, err = t.mo.WriteAt([]byte("taco"), 0)

	AssertEq(nil, err)
	AssertEq(4, n)

	// Sync should regard us as dirty.
	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(oglemock.Return(nil, errors.New("")))

	t.mo.Sync()
}

func (t *MutableObjectTest) Sync_AfterTruncating() {
	var err error

	// Successfully truncate.
	s := initialContents
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader(s)), nil))

	err = t.mo.Truncate(17)
	AssertEq(nil, err)

	// Sync should regard us as dirty.
	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(oglemock.Return(nil, errors.New("")))

	t.mo.Sync()
}

func (t *MutableObjectTest) Sync_CallsCreateObject() {
	var err error

	// Dirty the object by truncating.
	s := initialContents
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader(s)), nil))

	err = t.mo.Truncate(1)
	AssertEq(nil, err)

	// CreateObject should be called with the correct precondition.
	ExpectCall(t.bucket, "CreateObject")(
		Any(),
		AllOf(
			nameIs(t.src.Name),
			contentsAre(initialContents[:1]),
			generationIs(t.src.Generation))).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Sync
	t.mo.Sync()
}

func (t *MutableObjectTest) Sync_CreateObjectFails() {
	// Dirty the proxy.
	s := initialContents
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader(s)), nil))

	t.mo.Truncate(0)

	// CreateObject -- return an error.
	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// Sync
	err := t.mo.Sync()

	AssertNe(nil, err)
	ExpectThat(err, Not(HasSameTypeAs(&gcs.PreconditionError{})))
	ExpectThat(err, Error(HasSubstr("CreateObject")))
	ExpectThat(err, Error(HasSubstr("taco")))

	// Nothing should have changed.
	ExpectEq(t.src.Generation, t.mo.SourceGeneration())

	// A further call to Sync should cause the bucket to be called again.
	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(oglemock.Return(nil, errors.New("")))

	t.mo.Sync()
}

func (t *MutableObjectTest) Sync_CreateObjectSaysPreconditionFailed() {
	// Dirty the proxy.
	s := initialContents
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader(s)), nil))

	t.mo.Truncate(0)

	// CreateObject -- return a precondition error.
	e := &gcs.PreconditionError{Err: errors.New("taco")}
	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(oglemock.Return(nil, e))

	// Sync
	err := t.mo.Sync()

	AssertThat(err, HasSameTypeAs(&gcs.PreconditionError{}))
	ExpectThat(err, Error(HasSubstr("CreateObject")))
	ExpectThat(err, Error(HasSubstr("taco")))

	// Nothing should have changed.
	ExpectEq(t.src.Generation, t.mo.SourceGeneration())

	// A further call to Sync should cause the bucket to be called again.
	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(oglemock.Return(nil, errors.New("")))

	t.mo.Sync()
}

func (t *MutableObjectTest) Sync_Successful() {
	var n int
	var err error

	// Dirty the proxy.
	s := initialContents
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader(s)), nil))

	n, err = t.mo.WriteAt([]byte("taco"), 0)
	AssertEq(nil, err)
	AssertEq(len("taco"), n)

	// Have the call to CreateObject succeed.
	o := &gcs.Object{
		Name:       t.src.Name,
		Generation: 17,
	}

	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(oglemock.Return(o, nil))

	// Sync -- should succeed
	err = t.mo.Sync()

	AssertEq(nil, err)
	ExpectEq(17, t.mo.SourceGeneration())

	// Further calls to Sync should do nothing.
	err = t.mo.Sync()

	AssertEq(nil, err)
	ExpectEq(17, t.mo.SourceGeneration())

	// Further calls to read should see the contents we wrote.
	buf := make([]byte, 4)
	n, _ = t.mo.ReadAt(buf, 0)

	ExpectEq(4, n)
	ExpectEq("taco", string(buf[:n]))
}

func (t *MutableObjectTest) WriteThenSyncThenWriteThenSync() {
	var n int
	var err error

	// NewReader
	s := initialContents
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader(s)), nil))

	// Dirty the proxy.
	err = t.mo.Truncate(2)
	AssertEq(nil, err)

	// Sync -- should cause the contents so far to be written out.
	o := &gcs.Object{
		Name:       t.src.Name,
		Generation: 1,
		Size:       2,
	}

	ExpectCall(t.bucket, "CreateObject")(Any(), contentsAre(initialContents[:2])).
		WillOnce(oglemock.Return(o, nil))

	err = t.mo.Sync()
	AssertEq(nil, err)

	// Write some more data at the end. The pre-Sync contents from before should
	// be re-used, so NewReader should not be called.
	n, err = t.mo.WriteAt([]byte("burrito"), 1)
	AssertEq(nil, err)
	AssertEq(len("burrito"), n)

	// Sync -- should cause the full contents to be written out.
	expected := initialContents[:1] + "burrito"
	o.Generation = 2
	ExpectCall(t.bucket, "CreateObject")(Any(), contentsAre(expected)).
		WillOnce(oglemock.Return(o, nil))

	err = t.mo.Sync()
	AssertEq(nil, err)
}

func (t *MutableObjectTest) Stat_CallsBucket() {
	// StatObject
	ExpectCall(t.bucket, "StatObject")(Any(), nameIs(t.src.Name)).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Stat
	t.mo.Stat(true)
}

func (t *MutableObjectTest) Stat_BucketFails() {
	// StatObject
	ExpectCall(t.bucket, "StatObject")(Any(), Any()).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// Stat
	_, err := t.mo.Stat(true)

	ExpectThat(err, Error(HasSubstr("StatObject")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *MutableObjectTest) Stat_BucketSaysNotFound_NotDirty() {
	// StatObject
	ExpectCall(t.bucket, "StatObject")(Any(), Any()).
		WillOnce(oglemock.Return(nil, &gcs.NotFoundError{}))

	// Stat
	sr, err := t.mo.Stat(true)

	AssertEq(nil, err)
	ExpectEq(t.src.Size, sr.Size)
	ExpectThat(sr.Mtime, timeutil.TimeEq(t.src.Updated))
	ExpectTrue(sr.Clobbered)
}

func (t *MutableObjectTest) Stat_BucketSaysNotFound_Dirty() {
	var err error

	// Dirty the object by truncating.
	s := initialContents
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader(s)), nil))

	t.clock.AdvanceTime(time.Second)
	truncateTime := t.clock.Now()

	err = t.mo.Truncate(17)
	AssertEq(nil, err)

	t.clock.AdvanceTime(time.Second)

	// StatObject
	ExpectCall(t.bucket, "StatObject")(Any(), Any()).
		WillOnce(oglemock.Return(nil, &gcs.NotFoundError{}))

	// Stat
	sr, err := t.mo.Stat(true)

	AssertEq(nil, err)
	ExpectEq(17, sr.Size)
	ExpectThat(sr.Mtime, timeutil.TimeEq(truncateTime))
	ExpectTrue(sr.Clobbered)
}

func (t *MutableObjectTest) Stat_InitialState() {
	var err error

	// StatObject
	o := &gcs.Object{
		Name:       t.src.Name,
		Generation: t.src.Generation,
		Size:       t.src.Size,
	}

	ExpectCall(t.bucket, "StatObject")(Any(), Any()).
		WillOnce(oglemock.Return(o, nil))

	// Stat
	sr, err := t.mo.Stat(true)

	AssertEq(nil, err)
	ExpectEq(t.src.Size, sr.Size)
	ExpectThat(sr.Mtime, timeutil.TimeEq(t.src.Updated))
	ExpectFalse(sr.Clobbered)
}

func (t *MutableObjectTest) Stat_AfterShortening() {
	var err error

	// Truncate
	s := initialContents
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader(s)), nil))

	t.clock.AdvanceTime(time.Second)
	truncateTime := t.clock.Now()

	err = t.mo.Truncate(int64(t.src.Size - 1))
	AssertEq(nil, err)

	t.clock.AdvanceTime(time.Second)

	// StatObject
	o := &gcs.Object{
		Name:       t.src.Name,
		Generation: t.src.Generation,
		Size:       t.src.Size,
	}

	ExpectCall(t.bucket, "StatObject")(Any(), Any()).
		WillOnce(oglemock.Return(o, nil))

	// Stat
	sr, err := t.mo.Stat(true)

	AssertEq(nil, err)
	ExpectEq(t.src.Size-1, sr.Size)
	ExpectThat(sr.Mtime, timeutil.TimeEq(truncateTime))
	ExpectFalse(sr.Clobbered)
}

func (t *MutableObjectTest) Stat_AfterGrowing() {
	var err error

	// Truncate
	s := initialContents
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader(s)), nil))

	t.clock.AdvanceTime(time.Second)
	truncateTime := t.clock.Now()

	err = t.mo.Truncate(int64(t.src.Size + 17))
	AssertEq(nil, err)

	t.clock.AdvanceTime(time.Second)

	// StatObject
	o := &gcs.Object{
		Name:       t.src.Name,
		Generation: t.src.Generation,
		Size:       t.src.Size,
	}

	ExpectCall(t.bucket, "StatObject")(Any(), Any()).
		WillOnce(oglemock.Return(o, nil))

	// Stat
	sr, err := t.mo.Stat(true)

	AssertEq(nil, err)
	ExpectEq(t.src.Size+17, sr.Size)
	ExpectThat(sr.Mtime, timeutil.TimeEq(truncateTime))
	ExpectFalse(sr.Clobbered)
}

func (t *MutableObjectTest) Stat_AfterReading() {
	var err error

	// Read
	s := initialContents
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader(s)), nil))

	_, err = t.mo.ReadAt([]byte{}, 0)
	AssertEq(nil, err)

	// StatObject
	o := &gcs.Object{
		Name:       t.src.Name,
		Generation: t.src.Generation,
		Size:       t.src.Size,
	}

	ExpectCall(t.bucket, "StatObject")(Any(), Any()).
		WillOnce(oglemock.Return(o, nil))

	// Stat
	sr, err := t.mo.Stat(true)

	AssertEq(nil, err)
	ExpectEq(t.src.Size, sr.Size)
	ExpectThat(sr.Mtime, timeutil.TimeEq(t.src.Updated))
	ExpectFalse(sr.Clobbered)
}

func (t *MutableObjectTest) Stat_AfterWriting() {
	var err error

	// Extend by writing.
	s := initialContents
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader(s)), nil))

	t.clock.AdvanceTime(time.Second)
	writeTime := t.clock.Now()

	_, err = t.mo.WriteAt([]byte("taco"), int64(t.src.Size))
	AssertEq(nil, err)

	t.clock.AdvanceTime(time.Second)

	// StatObject
	o := &gcs.Object{
		Name:       t.src.Name,
		Generation: t.src.Generation,
		Size:       t.src.Size,
	}

	ExpectCall(t.bucket, "StatObject")(Any(), Any()).
		WillOnce(oglemock.Return(o, nil))

	// Stat
	sr, err := t.mo.Stat(true)

	AssertEq(nil, err)
	ExpectEq(int(t.src.Size)+len("taco"), sr.Size)
	ExpectThat(sr.Mtime, timeutil.TimeEq(writeTime))
	ExpectFalse(sr.Clobbered)
}

func (t *MutableObjectTest) Stat_ClobberedByNewGeneration_NotDirty() {
	// StatObject
	o := &gcs.Object{
		Name:       t.src.Name,
		Generation: t.src.Generation + 17,
		Size:       t.src.Size,
	}

	ExpectCall(t.bucket, "StatObject")(Any(), Any()).
		WillOnce(oglemock.Return(o, nil))

	// Stat
	sr, err := t.mo.Stat(true)

	AssertEq(nil, err)
	ExpectEq(t.src.Size, sr.Size)
	ExpectThat(sr.Mtime, timeutil.TimeEq(t.src.Updated))
	ExpectTrue(sr.Clobbered)
}

func (t *MutableObjectTest) Stat_ClobberedByNewGeneration_Dirty() {
	var err error

	// Truncate
	s := initialContents
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader(s)), nil))

	t.clock.AdvanceTime(time.Second)
	truncateTime := t.clock.Now()

	err = t.mo.Truncate(int64(t.src.Size + 17))
	AssertEq(nil, err)

	t.clock.AdvanceTime(time.Second)

	// StatObject
	o := &gcs.Object{
		Name:       t.src.Name,
		Generation: t.src.Generation + 19,
		Size:       t.src.Size,
	}

	ExpectCall(t.bucket, "StatObject")(Any(), Any()).
		WillOnce(oglemock.Return(o, nil))

	// Stat
	sr, err := t.mo.Stat(true)

	AssertEq(nil, err)
	ExpectEq(t.src.Size+17, sr.Size)
	ExpectThat(sr.Mtime, timeutil.TimeEq(truncateTime))
	ExpectTrue(sr.Clobbered)
}

func (t *MutableObjectTest) Stat_DontNeedClobberedInfo() {
	var err error

	// Stat
	sr, err := t.mo.Stat(false)

	AssertEq(nil, err)
	ExpectEq(t.src.Size, sr.Size)
	ExpectThat(sr.Mtime, timeutil.TimeEq(t.src.Updated))
	ExpectFalse(sr.Clobbered)
}
