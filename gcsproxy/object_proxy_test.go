// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package gcsproxy_test

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/mock_gcs"
	"github.com/jacobsa/gcsfuse/gcsproxy"
	. "github.com/jacobsa/oglematchers"
	"github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"golang.org/x/net/context"
	"google.golang.org/cloud/storage"
)

func TestOgletest(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// An oglemock.Matcher that accepts a predicate function and a description,
// making it easy to make anonymous matcher types.
type predicateMatcher struct {
	Desc      string
	Predicate func(interface{}) error
}

var _ Matcher = &predicateMatcher{}

func (m *predicateMatcher) Matches(candidate interface{}) error {
	return m.Predicate(candidate)
}

func (m *predicateMatcher) Description() string {
	return m.Desc
}

func nameIs(name string) Matcher {
	return &predicateMatcher{
		Desc: fmt.Sprintf("Name is: %s", name),
		Predicate: func(candidate interface{}) error {
			req := candidate.(*gcs.CreateObjectRequest)
			if req.Attrs.Name != name {
				return errors.New("")
			}

			return nil
		},
	}
}

func contentsAre(s string) Matcher {
	return &predicateMatcher{
		Desc: fmt.Sprintf("Object contents are: %s", s),
		Predicate: func(candidate interface{}) error {
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
	}
}

////////////////////////////////////////////////////////////////////////
// Invariant-checking object proxy
////////////////////////////////////////////////////////////////////////

// A wrapper around ObjectProxy that calls CheckInvariants whenever invariants
// should hold. For catching logic errors early in the test.
type checkingObjectProxy struct {
	wrapped *gcsproxy.ObjectProxy
}

func (op *checkingObjectProxy) NoteLatest(o *storage.Object) error {
	op.wrapped.CheckInvariants()
	defer op.wrapped.CheckInvariants()
	return op.wrapped.NoteLatest(o)
}

func (op *checkingObjectProxy) Size() (uint64, error) {
	op.wrapped.CheckInvariants()
	defer op.wrapped.CheckInvariants()
	return op.wrapped.Size()
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

func (op *checkingObjectProxy) Truncate(n uint64) error {
	op.wrapped.CheckInvariants()
	defer op.wrapped.CheckInvariants()
	return op.wrapped.Truncate(context.Background(), n)
}

func (op *checkingObjectProxy) Sync() (*storage.Object, error) {
	op.wrapped.CheckInvariants()
	defer op.wrapped.CheckInvariants()
	return op.wrapped.Sync(context.Background())
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type ObjectProxyTest struct {
	objectName string
	bucket     mock_gcs.MockBucket
	op         checkingObjectProxy
}

var _ SetUpInterface = &ObjectProxyTest{}

func (t *ObjectProxyTest) SetUp(ti *TestInfo) {
	t.objectName = "some/object"
	t.bucket = mock_gcs.NewMockBucket(ti.MockController, "bucket")

	var err error
	t.op.wrapped, err = gcsproxy.NewObjectProxy(
		t.bucket,
		t.objectName)

	if err != nil {
		panic(err)
	}
}

////////////////////////////////////////////////////////////////////////
// No source object
////////////////////////////////////////////////////////////////////////

// A test whose initial conditions are a fresh object proxy without a source
// object set.
type NoSourceObjectTest struct {
	ObjectProxyTest
}

var _ SetUpInterface = &NoSourceObjectTest{}

func init() { RegisterTestSuite(&NoSourceObjectTest{}) }

func (t *NoSourceObjectTest) NoteLatest_NegativeSize() {
	o := &storage.Object{
		Name:       t.objectName,
		Generation: 1234,
		Size:       -1,
	}

	err := t.op.NoteLatest(o)

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("size")))
	ExpectThat(err, Error(HasSubstr("-1")))
}

func (t *NoSourceObjectTest) NoteLatest_WrongName() {
	o := &storage.Object{
		Name:       t.objectName + "foo",
		Generation: 1234,
		Size:       0,
	}

	err := t.op.NoteLatest(o)

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("name")))
	ExpectThat(err, Error(HasSubstr("foo")))
}

func (t *NoSourceObjectTest) Size_InitialState() {
	size, err := t.op.Size()

	AssertEq(nil, err)
	ExpectEq(0, size)
}

func (t *NoSourceObjectTest) Size_AfterTruncatingToZero() {
	var err error

	// Truncate
	err = t.op.Truncate(0)
	AssertEq(nil, err)

	// Size
	size, err := t.op.Size()

	AssertEq(nil, err)
	ExpectEq(0, size)
}

func (t *NoSourceObjectTest) Size_AfterTruncatingToNonZero() {
	var err error

	// Truncate
	err = t.op.Truncate(123)
	AssertEq(nil, err)

	// Size
	size, err := t.op.Size()

	AssertEq(nil, err)
	ExpectEq(123, size)
}

func (t *NoSourceObjectTest) Size_AfterReading() {
	var err error

	// Read
	buf := make([]byte, 0)
	n, err := t.op.ReadAt(buf, 0)

	AssertEq(nil, err)
	AssertEq(0, n)

	// Size
	size, err := t.op.Size()

	AssertEq(nil, err)
	ExpectEq(0, size)
}

func (t *NoSourceObjectTest) Read_InitialState() {
	type testCase struct {
		offset      int64
		size        int
		expectedErr error
		expectedN   int
	}

	testCases := []testCase{
		// Empty ranges
		testCase{0, 0, nil, 0},
		testCase{17, 0, nil, 0},

		// Non-empty ranges
		testCase{0, 10, io.EOF, 0},
		testCase{17, 10, io.EOF, 0},
	}

	for _, tc := range testCases {
		buf := make([]byte, tc.size)
		n, err := t.op.ReadAt(buf, tc.offset)

		AssertEq(tc.expectedErr, err, "Test case: %v", tc)
		AssertEq(tc.expectedN, n, "Test case: %v", tc)
	}
}

func (t *NoSourceObjectTest) WriteToEndOfObjectThenRead() {
	var n int
	var err error
	var buf []byte

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

func (t *NoSourceObjectTest) WritePastEndOfObjectThenRead() {
	var n int
	var err error
	var buf []byte

	// Extend the object by writing past its end.
	n, err = t.op.WriteAt([]byte("taco"), 2)
	AssertEq(nil, err)
	AssertEq(len("taco"), n)

	// Check its size.
	size, err := t.op.Size()
	AssertEq(nil, err)
	ExpectEq(2+len("taco"), size)

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

func (t *NoSourceObjectTest) WriteWithinObjectThenRead() {
	var n int
	var err error
	var buf []byte

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

func (t *NoSourceObjectTest) GrowByTruncating() {
	var n int
	var err error
	var buf []byte

	// Truncate
	err = t.op.Truncate(4)
	AssertEq(nil, err)

	// Size
	size, err := t.op.Size()
	AssertEq(nil, err)
	ExpectEq(4, size)

	// Read the whole thing.
	buf = make([]byte, 1024)
	n, err = t.op.ReadAt(buf, 0)

	AssertEq(io.EOF, err)
	ExpectEq(4, n)
	ExpectEq("\x00\x00\x00\x00", string(buf[:n]))
}

func (t *NoSourceObjectTest) Sync_NoInteractions() {
	// CreateObject -- should receive an empty string.
	ExpectCall(t.bucket, "CreateObject")(
		Any(),
		AllOf(nameIs(t.objectName), contentsAre(""))).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Sync
	t.op.Sync()
}

func (t *NoSourceObjectTest) Sync_ReadCallsOnly() {
	// Make a Read call
	buf := make([]byte, 0)
	n, err := t.op.ReadAt(buf, 0)

	AssertEq(nil, err)
	AssertEq(0, n)

	// CreateObject -- should receive an empty string.
	ExpectCall(t.bucket, "CreateObject")(
		Any(),
		AllOf(nameIs(t.objectName), contentsAre(""))).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Sync
	t.op.Sync()
}

func (t *NoSourceObjectTest) Sync_AfterWriting() {
	var n int
	var err error

	// Write some data.
	n, err = t.op.WriteAt([]byte("taco"), 0)
	AssertEq(nil, err)
	AssertEq(len("taco"), n)

	n, err = t.op.WriteAt([]byte("burrito"), int64(len("taco")))
	AssertEq(nil, err)
	AssertEq(len("burrito"), n)

	// CreateObject -- should receive the contents we wrote above.
	ExpectCall(t.bucket, "CreateObject")(
		Any(),
		AllOf(nameIs(t.objectName), contentsAre("tacoburrito"))).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Sync
	t.op.Sync()
}

func (t *NoSourceObjectTest) Sync_AfterTruncating() {
	// Truncate outwards.
	err := t.op.Truncate(2)
	AssertEq(nil, err)

	// CreateObject -- should receive null bytes.
	ExpectCall(t.bucket, "CreateObject")(
		Any(),
		AllOf(nameIs(t.objectName), contentsAre("\x00\x00"))).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Sync
	t.op.Sync()
}

func (t *NoSourceObjectTest) Sync_CreateObjectFails() {
	var n int
	var err error

	// Write some data.
	n, err = t.op.WriteAt([]byte("taco"), 0)
	AssertEq(nil, err)
	AssertEq(len("taco"), n)

	// First call to create object: fail.
	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// Sync -- should fail.
	_, err = t.op.Sync()

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("CreateObject")))
	ExpectThat(err, Error(HasSubstr("taco")))

	// The data we wrote before should still be present.
	size, err := t.op.Size()
	AssertEq(nil, err)
	ExpectEq(len("taco"), size)

	buf := make([]byte, 1024)
	n, err = t.op.ReadAt(buf, 0)

	AssertEq(io.EOF, err)
	ExpectEq("taco", string(buf[:n]))

	// The file should still be regarded as dirty -- a further call to Sync
	// should call CreateObject again.
	ExpectCall(t.bucket, "CreateObject")(
		Any(),
		AllOf(nameIs(t.objectName), contentsAre("taco"))).
		WillOnce(oglemock.Return(nil, errors.New("")))

	t.op.Sync()
}

func (t *NoSourceObjectTest) Sync_Successful() {
	var n int
	var err error

	// Dirty the proxy.
	n, err = t.op.WriteAt([]byte("taco"), 0)
	AssertEq(nil, err)
	AssertEq(len("taco"), n)

	// Have the call to CreateObject succeed.
	expected := &storage.Object{
		Name: t.objectName,
	}

	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(oglemock.Return(expected, nil))

	// Sync -- should succeed
	o, err := t.op.Sync()

	AssertEq(nil, err)
	ExpectEq(expected, o)

	// Further calls to Sync should do nothing.
	o2, err := t.op.Sync()

	AssertEq(nil, err)
	ExpectEq(o, o2)

	// The data we wrote before should still be present.
	size, err := t.op.Size()
	AssertEq(nil, err)
	ExpectEq(len("taco"), size)

	buf := make([]byte, 1024)
	n, err = t.op.ReadAt(buf, 0)

	AssertEq(io.EOF, err)
	ExpectEq("taco", string(buf[:n]))
}

func (t *NoSourceObjectTest) NoteLatest_NoInteractions() {
	// NoteLatest
	newObj := &storage.Object{
		Name:       t.objectName,
		Generation: 17,
		Size:       19,
	}

	err := t.op.NoteLatest(newObj)
	AssertEq(nil, err)

	// The size should be reflected.
	size, err := t.op.Size()
	AssertEq(nil, err)
	ExpectEq(19, size)

	// Sync should return the new object without doing anything interesting.
	syncResult, err := t.op.Sync()
	AssertEq(nil, err)
	ExpectEq(newObj, syncResult)

	// A read should cause the object to be faulted in.
	ExpectCall(t.bucket, "NewReader")(Any(), t.objectName).
		WillOnce(oglemock.Return(nil, errors.New("")))

	t.op.ReadAt(make([]byte, 1), 0)
}

func (t *NoSourceObjectTest) NoteLatest_AfterWriting() {
	var err error

	// Write
	_, err = t.op.WriteAt([]byte("taco"), 0)
	AssertEq(nil, err)

	// NoteLatest
	newObj := &storage.Object{
		Name:       t.objectName,
		Generation: 17,
		Size:       19,
	}

	err = t.op.NoteLatest(newObj)
	AssertEq(nil, err)

	// The new size should be reflected.
	size, err := t.op.Size()
	AssertEq(nil, err)
	ExpectEq(19, size)

	// Sync should return the new object without doing anything interesting.
	syncResult, err := t.op.Sync()
	AssertEq(nil, err)
	ExpectEq(newObj, syncResult)

	// A read should cause the object to be faulted in.
	ExpectCall(t.bucket, "NewReader")(Any(), t.objectName).
		WillOnce(oglemock.Return(nil, errors.New("")))

	t.op.ReadAt(make([]byte, 1), 0)
}

////////////////////////////////////////////////////////////////////////
// Source object present
////////////////////////////////////////////////////////////////////////

// A test whose initial conditions are an object proxy branching from a source
// object in the bucket.
type SourceObjectPresentTest struct {
	ObjectProxyTest
	sourceObject *storage.Object
}

var _ SetUpInterface = &SourceObjectPresentTest{}

func init() { RegisterTestSuite(&SourceObjectPresentTest{}) }

func (t *SourceObjectPresentTest) SetUp(ti *TestInfo) {
	t.ObjectProxyTest.SetUp(ti)

	// Set up the source object.
	t.sourceObject = &storage.Object{
		Name:       t.objectName,
		Generation: 123,
		Size:       456,
	}

	if err := t.op.NoteLatest(t.sourceObject); err != nil {
		panic(err)
	}
}

func (t *SourceObjectPresentTest) Read_CallsNewReader() {
	// Bucket.NewReader
	ExpectCall(t.bucket, "NewReader")(Any(), t.sourceObject.Name).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// ReadAt
	t.op.ReadAt(make([]byte, 1), 0)
}

func (t *SourceObjectPresentTest) Read_NewReaderFails() {
	// Bucket.NewReader
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// ReadAt
	_, err := t.op.ReadAt(make([]byte, 1), 0)

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("NewReader")))
	ExpectThat(err, Error(HasSubstr("taco")))

	// A subsequent call should cause it to happen all over again.
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(nil, errors.New("")))

	t.op.ReadAt(make([]byte, 1), 0)
}

func (t *SourceObjectPresentTest) Read_NewReaderSucceeds() {
	buf := make([]byte, 1024)
	var n int
	var err error

	// Bucket.NewReader
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader("taco")), nil))

	// Reads
	n, err = t.op.ReadAt(buf[:1], 2)
	AssertEq(nil, err)
	ExpectEq("c", string(buf[:n]))

	n, err = t.op.ReadAt(buf[:10], 0)
	AssertEq(io.EOF, err)
	ExpectEq("taco", string(buf[:n]))

	// Sync should do nothing interesting.
	syncResult, err := t.op.Sync()

	AssertEq(nil, err)
	ExpectEq(t.sourceObject, syncResult)
}

func (t *SourceObjectPresentTest) Write_CallsNewReader() {
	// Bucket.NewReader
	ExpectCall(t.bucket, "NewReader")(Any(), t.sourceObject.Name).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// WriteAt
	t.op.WriteAt([]byte(""), 0)
}

func (t *SourceObjectPresentTest) Write_NewReaderFails() {
	// Bucket.NewReader
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// ReadAt
	_, err := t.op.WriteAt([]byte(""), 0)

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("NewReader")))
	ExpectThat(err, Error(HasSubstr("taco")))

	// A subsequent call should cause it to happen all over again.
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(nil, errors.New("")))

	t.op.WriteAt([]byte(""), 0)
}

func (t *SourceObjectPresentTest) Write_NewReaderSucceeds() {
	buf := make([]byte, 1024)
	var n int
	var err error

	// Bucket.NewReader
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader("taco")), nil))

	// Write
	_, err = t.op.WriteAt([]byte("burrito"), 3)
	AssertEq(nil, err)

	// Read
	n, err = t.op.ReadAt(buf, 0)
	AssertEq(io.EOF, err)
	ExpectEq("tacburrito", string(buf[:n]))

	// The object should be regarded as dirty by Sync.
	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(oglemock.Return(nil, errors.New("")))

	t.op.Sync()
}

func (t *SourceObjectPresentTest) Truncate_CallsNewReader() {
	// Bucket.NewReader
	ExpectCall(t.bucket, "NewReader")(Any(), t.sourceObject.Name).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// WriteAt
	t.op.Truncate(1)
}

func (t *SourceObjectPresentTest) Truncate_NewReaderFails() {
	// Bucket.NewReader
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// ReadAt
	err := t.op.Truncate(1)

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("NewReader")))
	ExpectThat(err, Error(HasSubstr("taco")))

	// A subsequent call should cause it to happen all over again.
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(nil, errors.New("")))

	t.op.Truncate(1)
}

func (t *SourceObjectPresentTest) Truncate_NewReaderSucceeds() {
	buf := make([]byte, 1024)
	var n int
	var err error

	// Bucket.NewReader
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader("taco")), nil))

	// Truncate
	err = t.op.Truncate(1)
	AssertEq(nil, err)

	// Read
	n, err = t.op.ReadAt(buf, 0)
	AssertEq(io.EOF, err)
	ExpectEq("t", string(buf[:n]))

	// The object should be regarded as dirty by Sync.
	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(oglemock.Return(nil, errors.New("")))

	t.op.Sync()
}

func (t *SourceObjectPresentTest) Sync_NoInteractions() {
	// Sync should do nothing interesting.
	syncResult, err := t.op.Sync()

	AssertEq(nil, err)
	ExpectEq(t.sourceObject, syncResult)
}

func (t *SourceObjectPresentTest) NoteLatest_EarlierThanPrev() {
	var err error

	// NoteLatest
	o := &storage.Object{}
	*o = *t.sourceObject
	o.Generation--

	err = t.op.NoteLatest(o)
	AssertEq(nil, err)

	// The input should have been ignored.
	syncResult, err := t.op.Sync()

	AssertEq(nil, err)
	ExpectEq(t.sourceObject, syncResult)
}

func (t *SourceObjectPresentTest) NoteLatest_SameAsPrev() {
	AssertTrue(false, "TODO")
}

func (t *SourceObjectPresentTest) NoteLatest_NewerThanPrev() {
	AssertTrue(false, "TODO")
}
