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
	"reflect"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/mock_gcs"
	"github.com/jacobsa/gcsfuse/gcsproxy"
	. "github.com/jacobsa/oglematchers"
	"github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"golang.org/x/net/context"
	"google.golang.org/cloud/storage"
)

func TestObjectProxy(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func nameIs(name string) Matcher {
	return NewMatcher(
		func(candidate interface{}) error {
			var actual string
			switch typed := candidate.(type) {
			case *gcs.CreateObjectRequest:
				actual = typed.Attrs.Name

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
// Invariant-checking object proxy
////////////////////////////////////////////////////////////////////////

// A wrapper around ObjectProxy that calls CheckInvariants whenever invariants
// should hold. For catching logic errors early in the test.
type checkingObjectProxy struct {
	wrapped *gcsproxy.ObjectProxy
}

func (op *checkingObjectProxy) Name() string {
	op.wrapped.CheckInvariants()
	defer op.wrapped.CheckInvariants()
	return op.wrapped.Name()
}

func (op *checkingObjectProxy) Stat() (int64, bool, error) {
	op.wrapped.CheckInvariants()
	defer op.wrapped.CheckInvariants()
	return op.wrapped.Stat(context.Background())
}

func (op *checkingObjectProxy) ReadAt(b []byte, o int64) (int, error) {
	op.wrapped.CheckInvariants()
	defer op.wrapped.CheckInvariants()
	return op.wrapped.ReadAt(context.Background(), b, o)
}

func (op *checkingObjectProxy) WriteAt(b []byte, o int64) (int, error) {
	op.wrapped.CheckInvariants()
	defer op.wrapped.CheckInvariants()
	return op.wrapped.WriteAt(context.Background(), b, o)
}

func (op *checkingObjectProxy) Truncate(n int64) error {
	op.wrapped.CheckInvariants()
	defer op.wrapped.CheckInvariants()
	return op.wrapped.Truncate(context.Background(), n)
}

func (op *checkingObjectProxy) Sync() (int64, error) {
	op.wrapped.CheckInvariants()
	defer op.wrapped.CheckInvariants()
	return op.wrapped.Sync(context.Background())
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type ObjectProxyTest struct {
	src    storage.Object
	bucket mock_gcs.MockBucket
	op     checkingObjectProxy
}

var _ SetUpInterface = &ObjectProxyTest{}

func init() { RegisterTestSuite(&ObjectProxyTest{}) }

func (t *ObjectProxyTest) SetUp(ti *TestInfo) {
	t.src = storage.Object{
		Name:       "some/object",
		Generation: 123,
		Size:       456,
	}

	t.bucket = mock_gcs.NewMockBucket(ti.MockController, "bucket")

	var err error
	t.op.wrapped, err = gcsproxy.NewObjectProxy(
		context.Background(),
		t.bucket,
		&t.src)

	if err != nil {
		panic(err)
	}
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ObjectProxyTest) Read_CallsNewReader() {
	// NewReader
	ExpectCall(t.bucket, "NewReader")(
		Any(),
		AllOf(nameIs(t.src.Name), generationIs(t.src.Generation))).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// ReadAt
	t.op.ReadAt([]byte{}, 0)
}

func (t *ObjectProxyTest) Read_NewReaderFails() {
	// NewReader
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// ReadAt
	_, err := t.op.ReadAt([]byte{}, 0)

	ExpectThat(err, Error(HasSubstr("NewReader")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *ObjectProxyTest) Read_ReadError() {
	// NewReader -- return a reader that returns an error after the first byte.
	rc := ioutil.NopCloser(
		iotest.TimeoutReader(
			iotest.OneByteReader(
				strings.NewReader("aaa"))))

	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(rc, nil))

	// ReadAt
	_, err := t.op.ReadAt([]byte{}, 0)

	ExpectThat(err, Error(HasSubstr("Copy:")))
	ExpectThat(err, Error(HasSubstr("timeout")))
}

func (t *ObjectProxyTest) Read_CloseError() {
	// NewReader -- return a ReadCloser that will fail to close.
	rc := &errorReadCloser{
		wrapped: strings.NewReader(""),
		err:     errors.New("taco"),
	}

	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(rc, nil))

	// ReadAt
	_, err := t.op.ReadAt([]byte{}, 0)

	ExpectThat(err, Error(HasSubstr("Close:")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *ObjectProxyTest) Read_NewReaderSucceeds() {
	const contents = "tacoburrito"
	buf := make([]byte, 1024)
	var n int
	var err error

	// NewReader
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader(contents)), nil))

	// Read once.
	n, err = t.op.ReadAt(buf[:4], 0)

	AssertEq(nil, err)
	AssertEq(4, n)
	ExpectEq("taco", string(buf[:n]))

	// The second read should work without calling NewReader again.
	n, err = t.op.ReadAt(buf[:4], 2)

	AssertEq(nil, err)
	AssertEq(4, n)
	ExpectEq("cobu", string(buf[:n]))
}

func (t *ObjectProxyTest) Write_CallsNewReader() {
	// NewReader
	ExpectCall(t.bucket, "NewReader")(
		Any(),
		AllOf(nameIs(t.src.Name), generationIs(t.src.Generation))).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// WriteAt
	t.op.WriteAt([]byte{}, 0)
}

func (t *ObjectProxyTest) WriteToEndOfObjectThenRead() {
	var buf []byte
	var n int
	var err error

	// NewReader
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader("")), nil))

	// Extend the object by writing twice.
	n, err = t.op.WriteAt([]byte("taco"), 0)
	AssertEq(nil, err)
	AssertEq(len("taco"), n)

	n, err = t.op.WriteAt([]byte("burrito"), int64(len("taco")))
	AssertEq(nil, err)
	AssertEq(len("burrito"), n)

	// Read the whole thing.
	buf = make([]byte, 1024)
	n, err = t.op.ReadAt(buf, 0)

	AssertEq(io.EOF, err)
	ExpectEq(len("tacoburrito"), n)
	ExpectEq("tacoburrito", string(buf[:n]))

	// Read a range in the middle.
	buf = make([]byte, 4)
	n, err = t.op.ReadAt(buf, 3)

	AssertEq(nil, err)
	ExpectEq(4, n)
	ExpectEq("obur", string(buf[:n]))
}

func (t *ObjectProxyTest) WritePastEndOfObjectThenRead() {
	var n int
	var err error
	var buf []byte

	// NewReader
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader("")), nil))

	// Extend the object by writing past its end.
	n, err = t.op.WriteAt([]byte("taco"), 2)
	AssertEq(nil, err)
	AssertEq(len("taco"), n)

	// Read the whole thing.
	buf = make([]byte, 1024)
	n, err = t.op.ReadAt(buf, 0)

	AssertEq(io.EOF, err)
	ExpectEq(2+len("taco"), n)
	ExpectEq("\x00\x00taco", string(buf[:n]))

	// Read a range in the middle.
	buf = make([]byte, 4)
	n, err = t.op.ReadAt(buf, 1)

	AssertEq(nil, err)
	ExpectEq(4, n)
	ExpectEq("\x00tac", string(buf[:n]))
}

func (t *ObjectProxyTest) WriteWithinObjectThenRead() {
	var n int
	var err error
	var buf []byte

	// NewReader
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader("")), nil))

	// Write several bytes to extend the object.
	n, err = t.op.WriteAt([]byte("00000"), 0)
	AssertEq(nil, err)
	AssertEq(len("00000"), n)

	// Overwrite some in the middle.
	n, err = t.op.WriteAt([]byte("11"), 1)
	AssertEq(nil, err)
	AssertEq(len("11"), n)

	// Read the whole thing.
	buf = make([]byte, 1024)
	n, err = t.op.ReadAt(buf, 0)

	AssertEq(io.EOF, err)
	ExpectEq(len("01100"), n)
	ExpectEq("01100", string(buf[:n]))
}

func (t *ObjectProxyTest) Truncate_CallsNewReader() {
	// NewReader
	ExpectCall(t.bucket, "NewReader")(
		Any(),
		AllOf(nameIs(t.src.Name), generationIs(t.src.Generation))).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Truncate
	t.op.Truncate(17)
}

func (t *ObjectProxyTest) GrowByTruncating() {
	var n int
	var err error
	var buf []byte

	// NewReader
	s := strings.Repeat("a", int(t.src.Size))
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader(s)), nil))

	// Truncate
	err = t.op.Truncate(t.src.Size + 4)
	AssertEq(nil, err)

	// Read the whole thing.
	buf = make([]byte, 1024)
	n, err = t.op.ReadAt(buf, 0)

	AssertEq(nil, err)
	ExpectEq(t.src.Size+17, n)
	ExpectEq(s+"\x00\x00\x00\x00", string(buf[:n]))
}

func (t *ObjectProxyTest) ShrinkByTruncating() {
	AssertTrue(false, "TODO")
}

func (t *ObjectProxyTest) Sync_NoInteractions() {
	// There should be nothing to do.
	gen, err := t.op.Sync()

	AssertEq(nil, err)
	ExpectEq(t.src.Generation, gen)
}

func (t *ObjectProxyTest) Sync_AfterReading() {
	const contents = "tacoburrito"
	buf := make([]byte, 1024)
	var n int
	var err error

	// Successfully read a fiew bytes.
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader(contents)), nil))

	n, err = t.op.ReadAt(buf[:4], 0)

	AssertEq(nil, err)
	AssertEq(4, n)
	ExpectEq("taco", string(buf[:n]))

	// Sync should still need to do nothing.
	gen, err := t.op.Sync()

	AssertEq(nil, err)
	ExpectEq(t.src.Generation, gen)
}

func (t *ObjectProxyTest) Sync_AfterWriting() {
	var n int
	var err error

	// Successfully write a fiew bytes.
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader("")), nil))

	n, err = t.op.WriteAt([]byte("taco"), 0)

	AssertEq(nil, err)
	AssertEq(4, n)

	// Sync should regard us as dirty.
	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(oglemock.Return(nil, errors.New("")))

	t.op.Sync()
}

func (t *ObjectProxyTest) Sync_AfterTruncating() {
	var err error

	// Successfully truncate.
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader("")), nil))

	err = t.op.Truncate(17)
	AssertEq(nil, err)

	// Sync should regard us as dirty.
	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(oglemock.Return(nil, errors.New("")))

	t.op.Sync()
}

func (t *ObjectProxyTest) Sync_CallsCreateObject() {
	var err error

	// Dirty the object by truncating.
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader("")), nil))

	err = t.op.Truncate(1)
	AssertEq(nil, err)

	// CreateObject should be called with the correct precondition.
	ExpectCall(t.bucket, "CreateObject")(
		Any(),
		AllOf(
			nameIs(t.src.Name),
			contentsAre("\x00"),
			generationIs(t.src.Generation))).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Sync
	t.op.Sync()
}

func (t *ObjectProxyTest) Sync_CreateObjectFails() {
	// CreateObject -- return an error.
	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// Sync
	_, err := t.op.Sync()

	AssertNe(nil, err)
	ExpectThat(err, Not(HasSameTypeAs(&gcs.PreconditionError{})))
	ExpectThat(err, Error(HasSubstr("CreateObject")))
	ExpectThat(err, Error(HasSubstr("taco")))

	// A further call to Sync should cause the bucket to be called again.
	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(oglemock.Return(nil, errors.New("")))

	t.op.Sync()
}

func (t *ObjectProxyTest) Sync_CreateObjectSaysPreconditionFailed() {
	// CreateObject -- return a precondition error.
	e := &gcs.PreconditionError{Err: errors.New("taco")}
	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(oglemock.Return(nil, e))

	// Sync
	_, err := t.op.Sync()

	AssertThat(err, HasSameTypeAs(&gcs.PreconditionError{}))
	ExpectThat(err, Error(HasSubstr("CreateObject")))
	ExpectThat(err, Error(HasSubstr("taco")))

	// A further call to Sync should cause the bucket to be called again.
	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(oglemock.Return(nil, errors.New("")))

	t.op.Sync()
}

func (t *ObjectProxyTest) Sync_BucketReturnsZeroGeneration() {
	// CreateObject
	o := &storage.Object{
		Name:       t.src.Name,
		Generation: 0,
	}

	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(oglemock.Return(o, nil))

	// Sync
	_, err := t.op.Sync()

	AssertNe(nil, err)
	ExpectThat(err, Not(HasSameTypeAs(&gcs.PreconditionError{})))
	ExpectThat(err, Error(HasSubstr("CreateObject")))
	ExpectThat(err, Error(HasSubstr("invalid generation")))
	ExpectThat(err, Error(HasSubstr("0")))
}

func (t *ObjectProxyTest) Sync_Successful() {
	var n int
	var err error

	// Dirty the proxy.
	n, err = t.op.WriteAt([]byte("taco"), 0)
	AssertEq(nil, err)
	AssertEq(len("taco"), n)

	// Have the call to CreateObject succeed.
	o := &storage.Object{
		Name:       t.src.Name,
		Generation: 17,
	}

	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(oglemock.Return(o, nil))

	// Sync -- should succeed
	gen, err := t.op.Sync()

	AssertEq(nil, err)
	ExpectEq(17, gen)

	// Further calls to Sync should do nothing.
	gen, err = t.op.Sync()

	AssertEq(nil, err)
	ExpectEq(17, gen)

	// The data we wrote before should still be present.
	buf := make([]byte, 1024)
	n, err = t.op.ReadAt(buf, 0)

	AssertEq(io.EOF, err)
	ExpectEq("taco", string(buf[:n]))
}

func (t *ObjectProxyTest) WriteThenSyncThenWriteThenSync() {
	var n int
	var err error

	// NewReader
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader("")), nil))

	// Dirty the proxy.
	n, err = t.op.WriteAt([]byte("taco"), 0)
	AssertEq(nil, err)
	AssertEq(len("taco"), n)

	// Sync -- should cause the contents so far to be written out.
	o := &storage.Object{
		Name:       t.src.Name,
		Generation: 1,
	}

	ExpectCall(t.bucket, "CreateObject")(Any(), contentsAre("taco")).
		WillOnce(oglemock.Return(o, nil))

	_, err = t.op.Sync()
	AssertEq(nil, err)

	// Write some more data at the end.
	n, err = t.op.WriteAt([]byte("burrito"), 4)
	AssertEq(nil, err)
	AssertEq(len("burrito"), n)

	// Sync -- should cause the full contents to be written out.
	o.Generation = 2
	ExpectCall(t.bucket, "CreateObject")(Any(), contentsAre("tacoburrito")).
		WillOnce(oglemock.Return(o, nil))

	_, err = t.op.Sync()
	AssertEq(nil, err)
}

func (t *ObjectProxyTest) Stat_CallsBucket() {
	// StatObject
	ExpectCall(t.bucket, "StatObject")(Any(), nameIs(t.src.Name)).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Stat
	t.op.Stat()
}

func (t *ObjectProxyTest) Stat_BucketFails() {
	// StatObject
	ExpectCall(t.bucket, "StatObject")(Any(), Any()).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// Stat
	_, _, err := t.op.Stat()

	ExpectThat(err, Error(HasSubstr("StatObject")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *ObjectProxyTest) Stat_BucketSaysNotFound_NotDirty() {
	// StatObject
	ExpectCall(t.bucket, "StatObject")(Any(), Any()).
		WillOnce(oglemock.Return(nil, &gcs.NotFoundError{}))

	// Stat
	size, clobbered, err := t.op.Stat()

	AssertEq(nil, err)
	ExpectEq(t.src.Size, size)
	ExpectTrue(clobbered)
}

func (t *ObjectProxyTest) Stat_BucketSaysNotFound_Dirty() {
	var err error

	// Dirty the object by truncating.
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader("")), nil))

	err = t.op.Truncate(17)
	AssertEq(nil, err)

	// StatObject
	ExpectCall(t.bucket, "StatObject")(Any(), Any()).
		WillOnce(oglemock.Return(nil, &gcs.NotFoundError{}))

	// Stat
	size, clobbered, err := t.op.Stat()

	AssertEq(nil, err)
	ExpectEq(17, size)
	ExpectTrue(clobbered)
}

func (t *ObjectProxyTest) Stat_InitialState() {
	var err error

	// StatObject
	o := &storage.Object{
		Name:       t.src.Name,
		Generation: t.src.Generation,
		Size:       t.src.Size,
	}

	ExpectCall(t.bucket, "StatObject")(Any(), Any()).
		WillOnce(oglemock.Return(o, nil))

	// Stat
	size, clobbered, err := t.op.Stat()

	AssertEq(nil, err)
	ExpectEq(t.src.Size, size)
	ExpectFalse(clobbered)
}

func (t *ObjectProxyTest) Stat_AfterShortening() {
	var err error

	// Truncate
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader("")), nil))

	err = t.op.Truncate(t.src.Size - 1)
	AssertEq(nil, err)

	// StatObject
	o := &storage.Object{
		Name:       t.src.Name,
		Generation: t.src.Generation,
		Size:       t.src.Size,
	}

	ExpectCall(t.bucket, "StatObject")(Any(), Any()).
		WillOnce(oglemock.Return(o, nil))

	// Stat
	size, clobbered, err := t.op.Stat()

	AssertEq(nil, err)
	ExpectEq(t.src.Size-1, size)
	ExpectFalse(clobbered)
}

func (t *ObjectProxyTest) Stat_AfterGrowing() {
	var err error

	// Truncate
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader("")), nil))

	err = t.op.Truncate(t.src.Size + 17)
	AssertEq(nil, err)

	// StatObject
	o := &storage.Object{
		Name:       t.src.Name,
		Generation: t.src.Generation,
		Size:       t.src.Size,
	}

	ExpectCall(t.bucket, "StatObject")(Any(), Any()).
		WillOnce(oglemock.Return(o, nil))

	// Stat
	size, clobbered, err := t.op.Stat()

	AssertEq(nil, err)
	ExpectEq(t.src.Size+17, size)
	ExpectFalse(clobbered)
}

func (t *ObjectProxyTest) Stat_AfterReading() {
	var err error

	// Read
	s := strings.Repeat("a", int(t.src.Size))
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader(s)), nil))

	_, err = t.op.ReadAt([]byte{}, 0)
	AssertEq(nil, err)

	// StatObject
	o := &storage.Object{
		Name:       t.src.Name,
		Generation: t.src.Generation,
		Size:       t.src.Size,
	}

	ExpectCall(t.bucket, "StatObject")(Any(), Any()).
		WillOnce(oglemock.Return(o, nil))

	// Stat
	size, clobbered, err := t.op.Stat()

	AssertEq(nil, err)
	ExpectEq(t.src.Size, size)
	ExpectFalse(clobbered)
}

func (t *ObjectProxyTest) Stat_AfterWriting() {
	var err error

	// Extend by writing.
	s := strings.Repeat("a", int(t.src.Size))
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader(s)), nil))

	_, err = t.op.WriteAt([]byte("taco"), t.src.Size)
	AssertEq(nil, err)

	// StatObject
	o := &storage.Object{
		Name:       t.src.Name,
		Generation: t.src.Generation,
		Size:       t.src.Size,
	}

	ExpectCall(t.bucket, "StatObject")(Any(), Any()).
		WillOnce(oglemock.Return(o, nil))

	// Stat
	size, clobbered, err := t.op.Stat()

	AssertEq(nil, err)
	ExpectEq(t.src.Size+int64(len("taco")), size)
	ExpectFalse(clobbered)
}

func (t *ObjectProxyTest) Stat_ClobberedByNewGeneration_NotDirty() {
	// StatObject
	o := &storage.Object{
		Name:       t.src.Name,
		Generation: t.src.Generation + 17,
		Size:       t.src.Size,
	}

	ExpectCall(t.bucket, "StatObject")(Any(), Any()).
		WillOnce(oglemock.Return(o, nil))

	// Stat
	size, clobbered, err := t.op.Stat()

	AssertEq(nil, err)
	ExpectEq(t.src.Size, size)
	ExpectTrue(clobbered)
}

func (t *ObjectProxyTest) Stat_ClobberedByNewGeneration_Dirty() {
	var err error

	// Truncate
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader("")), nil))

	err = t.op.Truncate(t.src.Size + 17)
	AssertEq(nil, err)

	// StatObject
	o := &storage.Object{
		Name:       t.src.Name,
		Generation: t.src.Generation + 19,
		Size:       t.src.Size,
	}

	ExpectCall(t.bucket, "StatObject")(Any(), Any()).
		WillOnce(oglemock.Return(o, nil))

	// Stat
	size, clobbered, err := t.op.Stat()

	AssertEq(nil, err)
	ExpectEq(t.src.Size+17, size)
	ExpectTrue(clobbered)
}
