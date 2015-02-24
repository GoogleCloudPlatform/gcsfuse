// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)

package gcsproxy_test

import (
	"testing"

	"github.com/jacobsa/gcloud/gcs/mock_gcs"
	"github.com/jacobsa/gcsfuse/gcsproxy"
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
	return op.wrapped.ReadAt(b, o)
}

func (op *checkingObjectProxy) WriteAt(b []byte, o int64) (int, error) {
	op.wrapped.CheckInvariants()
	defer op.wrapped.CheckInvariants()
	return op.wrapped.WriteAt(b, o)
}

func (op *checkingObjectProxy) Truncate(n uint64) error {
	op.wrapped.CheckInvariants()
	defer op.wrapped.CheckInvariants()
	return op.wrapped.Truncate(n)
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
	AssertTrue(false, "TODO")
}

func (t *NoSourceObjectTest) NoteLatest_WrongName() {
	AssertTrue(false, "TODO")
}

func (t *NoSourceObjectTest) Size_InitialState() {
	AssertTrue(false, "TODO")
}

func (t *NoSourceObjectTest) Size_AfterTruncating() {
	AssertTrue(false, "TODO")
}

func (t *NoSourceObjectTest) Size_AfterReading() {
	AssertTrue(false, "TODO")
}

func (t *NoSourceObjectTest) Read_InitialState() {
	AssertTrue(false, "TODO")
}

func (t *NoSourceObjectTest) WriteThenRead() {
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
