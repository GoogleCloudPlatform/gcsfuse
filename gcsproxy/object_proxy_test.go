// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package gcsproxy_test

import (
	"io"
	"testing"

	"github.com/jacobsa/gcloud/gcs/mock_gcs"
	"github.com/jacobsa/gcsfuse/gcsproxy"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"golang.org/x/net/context"
	"google.golang.org/cloud/storage"
)

func TestOgletest(t *testing.T) { RunTests(t) }

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

func (op *checkingObjectProxy) Sync(ctx context.Context) (*storage.Object, error) {
	op.wrapped.CheckInvariants()
	defer op.wrapped.CheckInvariants()
	return op.wrapped.Sync(ctx)
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

func (t *NoSourceObjectTest) WriteToEndOfFileThenRead() {
	AssertTrue(false, "TODO")
}

func (t *NoSourceObjectTest) WritePastEndOfFileThenRead() {
	AssertTrue(false, "TODO")
}

func (t *NoSourceObjectTest) WriteWithinFileThenRead() {
	AssertTrue(false, "TODO")
}

func (t *NoSourceObjectTest) GrowByTruncating() {
	AssertTrue(false, "TODO")
}

func (t *NoSourceObjectTest) Sync_NoChanges() {
	AssertTrue(false, "TODO")
}

func (t *NoSourceObjectTest) Sync_AfterWriting() {
	AssertTrue(false, "TODO")
}

func (t *NoSourceObjectTest) Sync_AfterTruncating() {
	AssertTrue(false, "TODO")
}

////////////////////////////////////////////////////////////////////////
// Source object present
////////////////////////////////////////////////////////////////////////

// A test whose initial conditions are an object proxy branching from a source
// object in the bucket.
type SourceObjectPresentTest struct {
	ObjectProxyTest
	sourceObject storage.Object
}

var _ SetUpInterface = &SourceObjectPresentTest{}

func init() { RegisterTestSuite(&SourceObjectPresentTest{}) }

func (t *SourceObjectPresentTest) SetUp(ti *TestInfo) {
	t.ObjectProxyTest.SetUp(ti)

	// Set up the source object.
	t.sourceObject = storage.Object{
		Name:       t.objectName,
		Generation: 123,
		Size:       456,
	}

	if err := t.op.NoteLatest(&t.sourceObject); err != nil {
		panic(err)
	}
}

func (t *SourceObjectPresentTest) DoesFoo() {
	AssertTrue(false, "TODO")
}
