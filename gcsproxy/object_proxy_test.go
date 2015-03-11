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

func (t *NoSourceObjectTest) Stat_CallsBucket() {
	AssertTrue(false, "TODO")
}

func (t *NoSourceObjectTest) Stat_BucketFails() {
	AssertTrue(false, "TODO")
}

func (t *NoSourceObjectTest) Stat_BucketSaysFound() {
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

func (t *NoSourceObjectTest) Read_InitialState() {
	AssertTrue(false, "TODO")
}

func (t *NoSourceObjectTest) WriteToEndOfObjectThenRead() {
	AssertTrue(false, "TODO")
}

func (t *NoSourceObjectTest) WritePastEndOfObjectThenRead() {
	AssertTrue(false, "TODO")
}

func (t *NoSourceObjectTest) WriteWithinObjectThenRead() {
	AssertTrue(false, "TODO")
}

func (t *NoSourceObjectTest) GrowByTruncating() {
	AssertTrue(false, "TODO")
}

func (t *NoSourceObjectTest) Sync_NoInteractions() {
	AssertTrue(false, "TODO")
}

func (t *NoSourceObjectTest) Sync_ReadCallsOnly() {
	AssertTrue(false, "TODO")
}

func (t *NoSourceObjectTest) Sync_AfterWriting() {
	AssertTrue(false, "TODO")
}

func (t *NoSourceObjectTest) Sync_AfterTruncating() {
	AssertTrue(false, "TODO")
}

func (t *NoSourceObjectTest) Sync_CreateObjectFails() {
	AssertTrue(false, "TODO")
}

func (t *NoSourceObjectTest) Sync_CreateObjectSaysPreconditionFailed() {
	AssertTrue(false, "TODO")
}

func (t *NoSourceObjectTest) Sync_Successful() {
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

func (t *SourceObjectPresentTest) Size_InitialState() {
	size, err := t.op.Size()
	AssertEq(nil, err)
	ExpectEq(t.sourceObject.Size, size)
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
	var err error

	// NoteLatest
	o := &storage.Object{}
	*o = *t.sourceObject

	err = t.op.NoteLatest(o)
	AssertEq(nil, err)

	// The input should have been ignored.
	syncResult, err := t.op.Sync()

	AssertEq(nil, err)
	ExpectEq(t.sourceObject, syncResult)
}

func (t *SourceObjectPresentTest) NoteLatest_NewerThanPrev() {
	var err error

	// NoteLatest
	o := &storage.Object{}
	*o = *t.sourceObject
	o.Generation++

	err = t.op.NoteLatest(o)
	AssertEq(nil, err)

	// The input should have been adopted.
	syncResult, err := t.op.Sync()

	AssertEq(nil, err)
	ExpectEq(o, syncResult)
}

func (t *SourceObjectPresentTest) Clean_NoInteractions() {
	var err error

	// Clean
	err = t.op.Clean()
	AssertEq(nil, err)

	// Sync should still have to do nothing.
	syncResult, err := t.op.Sync()
	AssertEq(nil, err)
	ExpectEq(t.sourceObject, syncResult)
}

func (t *SourceObjectPresentTest) Clean_AfterReading() {
	buf := make([]byte, 1024)
	var n int
	var err error

	// Read, successfully.
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader("taco")), nil))

	n, err = t.op.ReadAt(buf, 0)
	AssertEq(io.EOF, err)
	AssertEq("taco", string(buf[:n]))

	// Clean
	err = t.op.Clean()
	AssertEq(nil, err)

	// Sync should need to do nothing.
	syncResult, err := t.op.Sync()
	AssertEq(nil, err)
	ExpectEq(t.sourceObject, syncResult)

	// The next read should need to fetch the object again.
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(nil, errors.New("")))

	t.op.ReadAt(buf, 0)
}

func (t *SourceObjectPresentTest) Clean_AfterWriting() {
	var err error

	// Write, successfully.
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader("")), nil))

	_, err = t.op.WriteAt([]byte("a"), 0)
	AssertEq(nil, err)

	// Clean
	err = t.op.Clean()
	AssertEq(nil, err)

	// Sync should need to do nothing.
	syncResult, err := t.op.Sync()
	AssertEq(nil, err)
	ExpectEq(t.sourceObject, syncResult)

	// The next write should need to fetch the object again.
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(nil, errors.New("")))

	t.op.WriteAt([]byte("a"), 0)
}

func (t *SourceObjectPresentTest) Clean_AfterTruncating() {
	var err error

	// Truncate, successfully.
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(ioutil.NopCloser(strings.NewReader("")), nil))

	err = t.op.Truncate(1)
	AssertEq(nil, err)

	// Clean
	err = t.op.Clean()
	AssertEq(nil, err)

	// Sync should need to do nothing.
	syncResult, err := t.op.Sync()
	AssertEq(nil, err)
	ExpectEq(t.sourceObject, syncResult)

	// The next truncation should need to fetch the object again.
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(oglemock.Return(nil, errors.New("")))

	t.op.Truncate(1)
}
