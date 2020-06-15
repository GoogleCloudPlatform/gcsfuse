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
	"time"

	"github.com/jacobsa/gcloud/gcs"
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
// Blocking reader
////////////////////////////////////////////////////////////////////////

// A reader that blocks until a channel is closed, then returns an error.
type blockingReader struct {
	c chan struct{}
}

func (br *blockingReader) Read(p []byte) (n int, err error) {
	<-br.c
	err = errors.New("blockingReader")
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

func rangeLimitIs(expected uint64) (m Matcher) {
	pred := func(c interface{}) (err error) {
		req := c.(*gcs.ReadObjectRequest)
		if req.Range == nil {
			err = errors.New("which has a nil range")
			return
		}

		if req.Range.Limit != expected {
			err = fmt.Errorf("which has Limit == %d", req.Range.Limit)
			return
		}

		return
	}

	m = NewMatcher(pred, fmt.Sprintf("has range limit %d", expected))
	return
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type RandomReaderTest struct {
	object *gcs.Object
	bucket gcs.MockBucket
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
	t.bucket = gcs.NewMockBucket(ti.MockController, "bucket")

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

func (t *RandomReaderTest) ReadAtEndOfObject() {
	buf := make([]byte, 1)

	n, err := t.rr.ReadAt(buf, int64(t.object.Size))
	ExpectEq(0, n)
	ExpectEq(io.EOF, err)
}

func (t *RandomReaderTest) ReadPastEndOfObject() {
	buf := make([]byte, 1)

	n, err := t.rr.ReadAt(buf, int64(t.object.Size)+1)
	ExpectEq(0, n)
	ExpectEq(io.EOF, err)
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
	// Set up a reader that will block until we tell it to return.
	finishRead := make(chan struct{})
	rc := ioutil.NopCloser(&blockingReader{finishRead})

	t.rr.wrapped.reader = rc
	t.rr.wrapped.start = 1
	t.rr.wrapped.limit = 4

	// Snoop on when cancel is called.
	cancelCalled := make(chan struct{})
	t.rr.wrapped.cancel = func() { close(cancelCalled) }

	// Start a read in the background using a context that we control. It should
	// not yet return.
	readReturned := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		buf := make([]byte, 2)
		t.rr.wrapped.ReadAt(ctx, buf, 1)
		close(readReturned)
	}()

	select {
	case <-time.After(10 * time.Millisecond):
	case <-readReturned:
		AddFailure("Read returned early.")
		AbortTest()
	}

	// When we cancel our context, the random reader should cancel the read
	// context.
	cancel()
	<-cancelCalled

	// Clean up.
	close(finishRead)
	<-readReturned
}

func (t *RandomReaderTest) DoesntPropagateCancellationAfterReturning() {
	// Set up a reader that will return three bytes.
	t.rr.wrapped.reader = ioutil.NopCloser(strings.NewReader("xxx"))
	t.rr.wrapped.start = 1
	t.rr.wrapped.limit = 4

	// Snoop on when cancel is called.
	cancelCalled := make(chan struct{})
	t.rr.wrapped.cancel = func() { close(cancelCalled) }

	// Successfully read two bytes using a context whose cancellation we control.
	ctx, cancel := context.WithCancel(context.Background())
	buf := make([]byte, 2)
	n, err := t.rr.wrapped.ReadAt(ctx, buf, 1)

	AssertEq(nil, err)
	AssertEq(2, n)

	// If we cancel the calling context now, it should not cause the underlying
	// read context to be cancelled.
	cancel()
	select {
	case <-time.After(10 * time.Millisecond):
	case <-cancelCalled:
		AddFailure("Read context unexpectedly cancelled.")
		AbortTest()
	}
}

func (t *RandomReaderTest) UpgradesReadsToObjectSize() {
	const objectSize = 2 * MB
	t.object.Size = objectSize

	const readSize = 10
	AssertLt(readSize, objectSize)

	// Simulate an existing reader at a mismatched offset.
	t.rr.wrapped.reader = ioutil.NopCloser(strings.NewReader("xxx"))
	t.rr.wrapped.cancel = func() {}
	t.rr.wrapped.start = 2
	t.rr.wrapped.limit = 5

	// The bucket should be asked to read the entire object, even though we only
	// ask for readSize bytes below, to minimize the cost for GCS requests.
	r := strings.NewReader(strings.Repeat("x", objectSize))
	rc := ioutil.NopCloser(r)

	ExpectCall(t.bucket, "NewReader")(
		Any(),
		AllOf(rangeStartIs(1), rangeLimitIs(objectSize))).
		WillOnce(Return(rc, nil))

	// Call through.
	buf := make([]byte, readSize)
	t.rr.ReadAt(buf, 1)

	// Check the state now.
	ExpectEq(1+readSize, t.rr.wrapped.start)
	ExpectEq(objectSize, t.rr.wrapped.limit)
}

func (t *RandomReaderTest) UpgradeReadsToAverageSize() {
	t.object.Size = 1 << 40
	const totalReadBytes = 6 * MB
	const numReads = 2
	const avgReadBytes = totalReadBytes / numReads

	const expectedBytesToRead = avgReadBytes
	const start = 1
	const readSize = 2 * minReadSize

	// Simulate an existing reader at a mismatched offset.
	t.rr.wrapped.seeks = numReads
	t.rr.wrapped.totalReadBytes = totalReadBytes
	t.rr.wrapped.reader = ioutil.NopCloser(strings.NewReader("xxx"))
	t.rr.wrapped.cancel = func() {}
	t.rr.wrapped.start = 2
	t.rr.wrapped.limit = 5

	// The bucket should be asked to read expectedBytesToRead bytes.
	r := strings.NewReader(strings.Repeat("x", expectedBytesToRead))
	rc := ioutil.NopCloser(r)

	ExpectCall(t.bucket, "NewReader")(
		Any(),
		AllOf(
			rangeStartIs(start),
			rangeLimitIs(start+expectedBytesToRead),
		)).WillOnce(Return(rc, nil))

	// Call through.
	buf := make([]byte, readSize)
	t.rr.ReadAt(buf, start)

	// Check the state now.
	ExpectEq(start+expectedBytesToRead, t.rr.wrapped.limit)
}

func (t *RandomReaderTest) UpgradesSequentialReads_ExistingReader() {
	t.object.Size = 1 << 40
	const readSize = 10

	// Simulate an existing reader at the correct offset, which will be exhausted
	// by the read below.
	const existingSize = 3
	r := strings.NewReader(strings.Repeat("x", existingSize))

	t.rr.wrapped.reader = ioutil.NopCloser(r)
	t.rr.wrapped.cancel = func() {}
	t.rr.wrapped.start = 1
	t.rr.wrapped.limit = 1 + existingSize

	// The bucket should be asked to read up to the end of the object.
	r = strings.NewReader(strings.Repeat("x", readSize-existingSize))
	rc := ioutil.NopCloser(r)

	ExpectCall(t.bucket, "NewReader")(
		Any(),
		AllOf(rangeStartIs(1+existingSize), rangeLimitIs(t.object.Size))).
		WillOnce(Return(rc, nil))

	// Call through.
	buf := make([]byte, readSize)
	t.rr.ReadAt(buf, 1)

	// Check the state now.
	ExpectEq(1+readSize, t.rr.wrapped.start)
	ExpectEq(t.object.Size, t.rr.wrapped.limit)
}

func (t *RandomReaderTest) UpgradesSequentialReads_NoExistingReader() {
	t.object.Size = 1 << 40
	const readSize = 10

	// Simulate a previous exhausted reader that ended at the offset from which
	// we read below.
	t.rr.wrapped.start = 1
	t.rr.wrapped.limit = 1

	// The bucket should be asked to read up to the end of the object.
	r := strings.NewReader(strings.Repeat("x", readSize))
	rc := ioutil.NopCloser(r)

	ExpectCall(t.bucket, "NewReader")(
		Any(),
		AllOf(rangeStartIs(1), rangeLimitIs(t.object.Size))).
		WillOnce(Return(rc, nil))

	// Call through.
	buf := make([]byte, readSize)
	t.rr.ReadAt(buf, 1)

	// Check the state now.
	ExpectEq(1+readSize, t.rr.wrapped.start)
	ExpectEq(t.object.Size, t.rr.wrapped.limit)
}
