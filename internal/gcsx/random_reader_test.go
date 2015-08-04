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
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"
	"testing/iotest"

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
// Counting closer
////////////////////////////////////////////////////////////////////////

type countingCloser struct {
	io.Reader
	closeCount int
}

func (cc *countingCloser) Close() (err error) {
	cc.closeCount++
	return
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func rangeStartIs(expected uint64) (m Matcher) {
	pred := func(c interface{}) (err error) {
		req := c.(*gcs.ReadObjectRequest)
		if req.Range == nil {
			err = errors.New("which has a nil range")
			return
		}

		if req.Range.Start != expected {
			err = fmt.Errorf("which has Start == %d", req.Range.Start)
			return
		}

		return
	}

	m = NewMatcher(pred, fmt.Sprintf("has range start %d", expected))
	return
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
	t.rr.ctx = ti.Ctx

	// Manufacture an object record.
	t.object = &gcs.Object{
		Name:       "foo",
		Size:       17,
		Generation: 1234,
	}

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
	// Bucket
	r := iotest.OneByteReader(iotest.TimeoutReader(strings.NewReader("xxx")))
	rc := ioutil.NopCloser(r)

	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(Return(rc, nil))

	// Call
	buf := make([]byte, 3)
	_, err := t.rr.ReadAt(buf, 0)

	ExpectThat(err, Error(HasSubstr("readFull")))
	ExpectThat(err, Error(HasSubstr(iotest.ErrTimeout.Error())))
}

func (t *RandomReaderTest) ReaderOvershootsRange() {
	// Simulate a reader that is supposed to return two more bytes, but actually
	// returns three when asked to.
	t.rr.wrapped.reader = ioutil.NopCloser(strings.NewReader("xxx"))
	t.rr.wrapped.cancel = func() {}
	t.rr.wrapped.start = 0
	t.rr.wrapped.limit = 2

	// Try to read three bytes.
	buf := make([]byte, 3)
	_, err := t.rr.ReadAt(buf, 0)

	ExpectThat(err, Error(HasSubstr("1 too many bytes")))
}

func (t *RandomReaderTest) ReaderNotExhausted() {
	// Set up a reader that has three bytes left to give.
	rc := &countingCloser{
		Reader: strings.NewReader("abc"),
	}

	t.rr.wrapped.reader = rc
	t.rr.wrapped.cancel = func() {}
	t.rr.wrapped.start = 1
	t.rr.wrapped.limit = 4

	// Read two bytes.
	buf := make([]byte, 2)
	n, err := t.rr.ReadAt(buf, 1)

	ExpectEq(2, n)
	ExpectEq(nil, err)
	ExpectEq("ab", string(buf[:n]))

	ExpectEq(0, rc.closeCount)
	ExpectEq(rc, t.rr.wrapped.reader)
	ExpectEq(3, t.rr.wrapped.start)
	ExpectEq(4, t.rr.wrapped.limit)
}

func (t *RandomReaderTest) ReaderExhausted_ReadFinished() {
	// Set up a reader that has three bytes left to give.
	rc := &countingCloser{
		Reader: strings.NewReader("abc"),
	}

	t.rr.wrapped.reader = rc
	t.rr.wrapped.cancel = func() {}
	t.rr.wrapped.start = 1
	t.rr.wrapped.limit = 4

	// Read three bytes.
	buf := make([]byte, 3)
	n, err := t.rr.ReadAt(buf, 1)

	ExpectEq(3, n)
	ExpectEq(nil, err)
	ExpectEq("abc", string(buf[:n]))

	ExpectEq(1, rc.closeCount)
	ExpectEq(nil, t.rr.wrapped.reader)
	ExpectEq(nil, t.rr.wrapped.cancel)
	ExpectEq(4, t.rr.wrapped.limit)
}

func (t *RandomReaderTest) ReaderExhausted_ReadNotFinished() {
	// Set up a reader that has three bytes left to give.
	rc := &countingCloser{
		Reader: strings.NewReader("abc"),
	}

	t.rr.wrapped.reader = rc
	t.rr.wrapped.cancel = func() {}
	t.rr.wrapped.start = 1
	t.rr.wrapped.limit = 4

	// The bucket should be called at the previous limit to obtain a new reader.
	ExpectCall(t.bucket, "NewReader")(Any(), rangeStartIs(4)).
		WillOnce(Return(nil, errors.New("")))

	// Attempt to read four bytes.
	buf := make([]byte, 4)
	n, _ := t.rr.ReadAt(buf, 1)

	AssertGe(n, 3)
	ExpectEq("abc", string(buf[:3]))

	ExpectEq(1, rc.closeCount)
	ExpectEq(nil, t.rr.wrapped.reader)
	ExpectEq(nil, t.rr.wrapped.cancel)
	ExpectEq(4, t.rr.wrapped.limit)
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
