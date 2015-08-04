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
	"io/ioutil"
	"strings"
	"testing"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/mock_gcs"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"golang.org/x/net/context"
)

func TestRandomReader(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Invariant-checking random reader
////////////////////////////////////////////////////////////////////////

type checkingRandomReader struct {
	ctx     context.Context
	wrapped *randomReader
}

func (rr *checkingRandomReader) ReadAt(p []byte, offset int64) (int, error) {
	rr.wrapped.CheckInvariants()
	defer rr.wrapped.CheckInvariants()
	return rr.wrapped.ReadAt(rr.ctx, p, offset)
}

func (rr *checkingRandomReader) Destroy() {
	rr.wrapped.CheckInvariants()
	rr.wrapped.Destroy()
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type RandomReaderTest struct {
	object *gcs.Object
	bucket mock_gcs.MockBucket
	rr     checkingRandomReader
}

func init() { RegisterTestSuite(&RandomReaderTest{}) }

var _ SetUpInterface = &RandomReaderTest{}
var _ TearDownInterface = &RandomReaderTest{}

func (t *RandomReaderTest) SetUp(ti *TestInfo) {
	// Manufacture an object record.
	t.object = &gcs.Object{Name: "foo", Generation: 17}

	// Create the bucket.
	t.bucket = mock_gcs.NewMockBucket(ti.MockController, "bucket")

	// Set up the reader.
	rr, err := NewRandomReader(t.object, t.bucket)
	AssertEq(nil, err)
	t.rr.wrapped = rr.(*randomReader)
}

func (t *RandomReaderTest) TearDown() {
	t.rr.Destroy()
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *RandomReaderTest) EmptyRead() {
	// Nothing should happen.
	buf := make([]byte, 0)

	n, err := t.rr.ReadAt(buf, 0)
	ExpectEq(0, n)
	ExpectEq(nil, err)
}

func (t *RandomReaderTest) NoExistingReader() {
	// The bucket should be called to set up a new reader.
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(Return(nil, errors.New("")))

	buf := make([]byte, 1)
	t.rr.ReadAt(buf, 0)
}

func (t *RandomReaderTest) ExistingReader_WrongOffset() {
	// Simulate an existing reader.
	t.rr.wrapped.reader = ioutil.NopCloser(strings.NewReader("xxx"))
	t.rr.wrapped.cancel = func() {}
	t.rr.wrapped.start = 2
	t.rr.wrapped.limit = 5

	// The bucket should be called to set up a new reader.
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(Return(nil, errors.New("")))

	buf := make([]byte, 1)
	t.rr.ReadAt(buf, 0)
}

func (t *RandomReaderTest) NewReaderReturnsError() {
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(Return(nil, errors.New("taco")))

	buf := make([]byte, 1)
	_, err := t.rr.ReadAt(buf, 0)

	ExpectThat(err, Error(HasSubstr("NewReader")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *RandomReaderTest) ReaderFails() {
	AssertTrue(false, "TODO")
}

func (t *RandomReaderTest) ReaderOvershootsRange() {
	AssertTrue(false, "TODO")
}

func (t *RandomReaderTest) ReaderExhausted_ReadFinished() {
	AssertTrue(false, "TODO")
}

func (t *RandomReaderTest) ReaderExhausted_ReadNotFinished() {
	AssertTrue(false, "TODO")
}

func (t *RandomReaderTest) PropagatesCancellation() {
	AssertTrue(false, "TODO")
}

func (t *RandomReaderTest) DoesntPropagateCancellationAfterReturning() {
	AssertTrue(false, "TODO")
}

func (t *RandomReaderTest) UpgradesReadsToMinimumSize() {
	AssertTrue(false, "TODO")
}

func (t *RandomReaderTest) UpgradesSequentialReads() {
	AssertTrue(false, "TODO")
}
