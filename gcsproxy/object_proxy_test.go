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

func nameIs(name string) Matcher {
	return NewMatcher(
		func(candidate interface{}) error {
			req := candidate.(*gcs.CreateObjectRequest)
			if req.Attrs.Name != name {
				return errors.New("")
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
		req := c.(*gcs.CreateObjectRequest)
		if req.GenerationPrecondition == nil {
			return errors.New("which has a nil GenerationPrecondition field.")
		}

		if *req.GenerationPrecondition != g {
			return fmt.Errorf(
				"Which has *GenerationPrecondition == %v",
				*req.GenerationPrecondition)
		}

		return nil
	}

	return NewMatcher(
		pred,
		fmt.Sprintf("*GenerationPrecondition == %v", g))
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

func (op *checkingObjectProxy) Stat() (uint64, bool, error) {
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

func (op *checkingObjectProxy) Truncate(n uint64) error {
	op.wrapped.CheckInvariants()
	defer op.wrapped.CheckInvariants()
	return op.wrapped.Truncate(context.Background(), n)
}

func (op *checkingObjectProxy) Sync() (uint64, error) {
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

func (t *ObjectProxyTest) setUp(ti *TestInfo, srcGeneration uint64) {
	t.objectName = "some/object"
	t.bucket = mock_gcs.NewMockBucket(ti.MockController, "bucket")

	var err error
	t.op.wrapped, err = gcsproxy.NewObjectProxy(
		context.Background(),
		t.bucket,
		t.objectName,
		srcGeneration)

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

func (t *NoSourceObjectTest) SetUp(ti *TestInfo) {
	t.ObjectProxyTest.setUp(ti, 0)
}

func (t *NoSourceObjectTest) Name() {
	ExpectEq(t.objectName, t.op.Name())
}

func (t *NoSourceObjectTest) Read_InitialState() {
	buf := make([]byte, 1024)
	n, err := t.op.ReadAt(buf, 0)

	ExpectEq(io.EOF, err)
	ExpectEq(0, n)
}

func (t *NoSourceObjectTest) WriteToEndOfObjectThenRead() {
	var buf []byte
	var n int
	var err error

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

	// Read the whole thing.
	buf = make([]byte, 1024)
	n, err = t.op.ReadAt(buf, 0)

	AssertEq(io.EOF, err)
	ExpectEq(4, n)
	ExpectEq("\x00\x00\x00\x00", string(buf[:n]))
}

func (t *NoSourceObjectTest) Sync_CallsCreateObject_NoInteractions() {
	// CreateObject -- should receive an empty string and a generation zero
	// precondition.
	ExpectCall(t.bucket, "CreateObject")(
		Any(),
		AllOf(nameIs(t.objectName), contentsAre(""), generationIs(0))).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Sync
	t.op.Sync()
}

func (t *NoSourceObjectTest) Sync_CallsCreateObject_AfterWriting() {
	// Write some data.
	n, err := t.op.WriteAt([]byte("taco"), 0)
	AssertEq(nil, err)
	AssertEq(4, n)

	// CreateObject -- should receive "taco" and a generation zero precondition.
	ExpectCall(t.bucket, "CreateObject")(
		Any(),
		AllOf(nameIs(t.objectName), contentsAre("taco"), generationIs(0))).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Sync
	t.op.Sync()
}

func (t *NoSourceObjectTest) Sync_CreateObjectFails() {
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

func (t *NoSourceObjectTest) Sync_CreateObjectSaysPreconditionFailed() {
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

func (t *NoSourceObjectTest) Sync_BucketReturnsNegativeGeneration() {
	// CreateObject
	o := &storage.Object{
		Name:       t.objectName,
		Generation: -1,
	}

	ExpectCall(t.bucket, "CreateObject")(Any(), Any()).
		WillOnce(oglemock.Return(o, nil))

	// Sync
	_, err := t.op.Sync()

	AssertNe(nil, err)
	ExpectThat(err, Not(HasSameTypeAs(&gcs.PreconditionError{})))
	ExpectThat(err, Error(HasSubstr("CreateObject")))
	ExpectThat(err, Error(HasSubstr("invalid generation")))
	ExpectThat(err, Error(HasSubstr("-1")))
}

func (t *NoSourceObjectTest) Sync_BucketReturnsZeroGeneration() {
	// CreateObject
	o := &storage.Object{
		Name:       t.objectName,
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

func (t *NoSourceObjectTest) Sync_Successful() {
	var n int
	var err error

	// Dirty the proxy.
	n, err = t.op.WriteAt([]byte("taco"), 0)
	AssertEq(nil, err)
	AssertEq(len("taco"), n)

	// Have the call to CreateObject succeed.
	o := &storage.Object{
		Name:       t.objectName,
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

func (t *NoSourceObjectTest) WriteThenSyncThenWriteThenSync() {
	var n int
	var err error

	// Dirty the proxy.
	n, err = t.op.WriteAt([]byte("taco"), 0)
	AssertEq(nil, err)
	AssertEq(len("taco"), n)

	// Sync -- should cause the contents so far to be written out.
	o := &storage.Object{
		Name:       t.objectName,
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

func (t *NoSourceObjectTest) Stat_CallsBucket() {
	AssertTrue(false, "TODO")
}

func (t *NoSourceObjectTest) Stat_BucketFails() {
	AssertTrue(false, "TODO")
}

func (t *NoSourceObjectTest) Stat_InitialState() {
	AssertTrue(false, "TODO")
}

func (t *NoSourceObjectTest) Stat_AfterShortening() {
	AssertTrue(false, "TODO")
}

func (t *NoSourceObjectTest) Stat_AfterGrowing() {
	AssertTrue(false, "TODO")
}

func (t *NoSourceObjectTest) Stat_AfterReading() {
	AssertTrue(false, "TODO")
}

func (t *NoSourceObjectTest) Stat_AfterWriting() {
	AssertTrue(false, "TODO")
}

func (t *NoSourceObjectTest) Stat_Clobbered() {
	AssertTrue(false, "TODO")
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
	t.ObjectProxyTest.setUp(ti, 123)
}

func (t *SourceObjectPresentTest) Read_CallsNewReader() {
	AssertTrue(false, "TODO")
}

func (t *SourceObjectPresentTest) Read_NewReaderFails() {
	AssertTrue(false, "TODO")
}

func (t *SourceObjectPresentTest) Read_NewReaderSucceeds() {
	AssertTrue(false, "TODO")
}

func (t *SourceObjectPresentTest) Write_CallsNewReader() {
	AssertTrue(false, "TODO")
}

func (t *SourceObjectPresentTest) Write_NewReaderFails() {
	AssertTrue(false, "TODO")
}

func (t *SourceObjectPresentTest) Write_NewReaderSucceeds() {
	AssertTrue(false, "TODO")
}

func (t *SourceObjectPresentTest) Truncate_CallsNewReader() {
	AssertTrue(false, "TODO")
}

func (t *SourceObjectPresentTest) Truncate_NewReaderFails() {
	AssertTrue(false, "TODO")
}

func (t *SourceObjectPresentTest) Truncate_NewReaderSucceeds() {
	AssertTrue(false, "TODO")
}

func (t *SourceObjectPresentTest) Sync_NoInteractions() {
	AssertTrue(false, "TODO")
}

func (t *SourceObjectPresentTest) Sync_AfterReading() {
	AssertTrue(false, "TODO")
}

func (t *SourceObjectPresentTest) Sync_AfterWriting() {
	AssertTrue(false, "TODO")
}

func (t *SourceObjectPresentTest) Sync_AfterTruncating() {
	AssertTrue(false, "TODO")
}

func (t *SourceObjectPresentTest) Sync_CallsCreateObject() {
	AssertTrue(false, "TODO")
}

func (t *SourceObjectPresentTest) Stat_CallsBucket() {
	AssertTrue(false, "TODO")
}

func (t *SourceObjectPresentTest) Stat_BucketFails() {
	AssertTrue(false, "TODO")
}

func (t *SourceObjectPresentTest) Stat_BucketSaysNotFound() {
	AssertTrue(false, "TODO")
}

func (t *SourceObjectPresentTest) Stat_InitialState() {
	AssertTrue(false, "TODO")
}

func (t *SourceObjectPresentTest) Stat_AfterShortening() {
	AssertTrue(false, "TODO")
}

func (t *SourceObjectPresentTest) Stat_AfterGrowing() {
	AssertTrue(false, "TODO")
}

func (t *SourceObjectPresentTest) Stat_AfterReading() {
	AssertTrue(false, "TODO")
}

func (t *SourceObjectPresentTest) Stat_AfterWriting() {
	AssertTrue(false, "TODO")
}

func (t *SourceObjectPresentTest) Stat_Clobbered() {
	AssertTrue(false, "TODO")
}
