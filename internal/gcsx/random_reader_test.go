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
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/cache/file"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/file/downloader"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/gcs"
	testutil "github.com/googlecloudplatform/gcsfuse/internal/util"
	"github.com/jacobsa/fuse/fuseops"
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

func (rr *checkingRandomReader) ReadAt(p []byte, offset int64) (int, bool, error) {
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

const sequentialReadSizeInMb = 22
const sequentialReadSizeInBytes = sequentialReadSizeInMb * MB
const CacheMaxSize = 2 * sequentialReadSizeInMb * util.MiB

type RandomReaderTest struct {
	object       *gcs.MinObject
	bucket       storage.MockBucket
	rr           checkingRandomReader
	cacheDir     string
	jobManager   *downloader.JobManager
	cacheHandler *file.CacheHandler
}

func init() { RegisterTestSuite(&RandomReaderTest{}) }

var _ SetUpInterface = &RandomReaderTest{}
var _ TearDownInterface = &RandomReaderTest{}

func (t *RandomReaderTest) SetUp(ti *TestInfo) {
	readOp := fuseops.ReadFileOp{Handle: 1}
	t.rr.ctx = context.WithValue(ti.Ctx, ReadOp, &readOp)

	// Manufacture an object record.
	t.object = &gcs.MinObject{
		Name:       "foo",
		Size:       17,
		Generation: 1234,
	}

	// Create the bucket.
	t.bucket = storage.NewMockBucket(ti.MockController, "bucket")

	t.cacheDir = path.Join(os.Getenv("HOME"), "cache/dir")
	lruCache := lru.NewCache(CacheMaxSize)
	t.jobManager = downloader.NewJobManager(lruCache, util.DefaultFilePerm, util.DefaultDirPerm, t.cacheDir, sequentialReadSizeInMb)
	t.cacheHandler = file.NewCacheHandler(lruCache, t.jobManager, t.cacheDir, util.DefaultFilePerm, util.DefaultDirPerm)

	// Set up the reader.
	rr := NewRandomReader(t.object, t.bucket, sequentialReadSizeInMb, nil, false)
	t.rr.wrapped = rr.(*randomReader)
}

func (t *RandomReaderTest) TearDown() {
	t.rr.Destroy()
}

func getReadCloser(content []byte) io.ReadCloser {
	r := bytes.NewReader(content)
	rc := io.NopCloser(r)
	return rc
}

func (t *RandomReaderTest) mockNewReaderCallForTestBucket(start uint64, limit uint64, rc io.ReadCloser) {
	ExpectCall(t.bucket, "NewReader")(
		Any(),
		AllOf(rangeStartIs(start), rangeLimitIs(limit))).
		WillRepeatedly(Return(rc, nil))
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *RandomReaderTest) EmptyRead() {
	// Nothing should happen.
	buf := make([]byte, 0)

	n, _, err := t.rr.ReadAt(buf, 0)

	ExpectEq(0, n)
	ExpectEq(nil, err)
}

func (t *RandomReaderTest) ReadAtEndOfObject() {
	buf := make([]byte, 1)

	n, _, err := t.rr.ReadAt(buf, int64(t.object.Size))

	ExpectEq(0, n)
	ExpectEq(io.EOF, err)
}

func (t *RandomReaderTest) ReadPastEndOfObject() {
	buf := make([]byte, 1)

	n, cacheHit, err := t.rr.ReadAt(buf, int64(t.object.Size)+1)

	ExpectFalse(cacheHit)
	ExpectEq(0, n)
	ExpectEq(io.EOF, err)
}

func (t *RandomReaderTest) NoExistingReader() {
	// The bucket should be called to set up a new reader.
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(Return(nil, errors.New("")))
	buf := make([]byte, 1)

	_, _, err := t.rr.ReadAt(buf, 0)

	AssertNe(nil, err)
}

func (t *RandomReaderTest) ExistingReader_WrongOffset() {
	// Simulate an existing reader.
	t.rr.wrapped.reader = io.NopCloser(strings.NewReader("xxx"))
	t.rr.wrapped.cancel = func() {}
	t.rr.wrapped.start = 2
	t.rr.wrapped.limit = 5
	// The bucket should be called to set up a new reader.
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(Return(nil, errors.New("")))
	buf := make([]byte, 1)

	_, _, err := t.rr.ReadAt(buf, 0)

	AssertNe(nil, err)
}

func (t *RandomReaderTest) ExistingReader_ReadAtOffsetAfterTheReaderPosition() {
	var currentStartOffset int64 = 2
	var readerLimit int64 = 15
	var readAtOffset int64 = 10
	var readSize int64 = 1
	var expectedStartOffsetAfterRead = readAtOffset + readSize
	// Simulate an existing reader.
	rc := io.NopCloser(strings.NewReader(strings.Repeat("x", int(readerLimit))))
	t.rr.wrapped.reader = rc
	t.rr.wrapped.cancel = func() {}
	t.rr.wrapped.start = currentStartOffset
	t.rr.wrapped.limit = readerLimit

	buf := make([]byte, readSize)
	_, _, err := t.rr.ReadAt(buf, readAtOffset)

	AssertEq(nil, err)
	ExpectThat(rc, DeepEquals(t.rr.wrapped.reader))
	ExpectEq(expectedStartOffsetAfterRead, t.rr.wrapped.start)
	ExpectEq(readerLimit, t.rr.wrapped.limit)
}

func (t *RandomReaderTest) NewReaderReturnsError() {
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(Return(nil, errors.New("taco")))
	buf := make([]byte, 1)

	_, _, err := t.rr.ReadAt(buf, 0)

	ExpectThat(err, Error(HasSubstr("NewReader")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *RandomReaderTest) ReaderFails() {
	// Bucket
	r := iotest.OneByteReader(iotest.TimeoutReader(strings.NewReader("xxx")))
	rc := io.NopCloser(r)

	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(Return(rc, nil))

	// Call
	buf := make([]byte, 3)
	_, _, err := t.rr.ReadAt(buf, 0)

	ExpectThat(err, Error(HasSubstr("readFull")))
	ExpectThat(err, Error(HasSubstr(iotest.ErrTimeout.Error())))
}

func (t *RandomReaderTest) ReaderOvershootsRange() {
	// Simulate a reader that is supposed to return two more bytes, but actually
	// returns three when asked to.
	t.rr.wrapped.reader = io.NopCloser(strings.NewReader("xxx"))
	t.rr.wrapped.cancel = func() {}
	t.rr.wrapped.start = 0
	t.rr.wrapped.limit = 2

	// Try to read three bytes.
	buf := make([]byte, 3)
	_, _, err := t.rr.ReadAt(buf, 0)

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
	n, cacheHit, err := t.rr.ReadAt(buf, 1)

	ExpectFalse(cacheHit)
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
	n, cacheHit, err := t.rr.ReadAt(buf, 1)

	ExpectFalse(cacheHit)
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
	n, cacheHit, _ := t.rr.ReadAt(buf, 1)

	ExpectFalse(cacheHit)
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
	rc := io.NopCloser(&blockingReader{finishRead})

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
	t.rr.wrapped.reader = io.NopCloser(strings.NewReader("xxx"))
	t.rr.wrapped.start = 1
	t.rr.wrapped.limit = 4

	// Snoop on when cancel is called.
	cancelCalled := make(chan struct{})
	t.rr.wrapped.cancel = func() { close(cancelCalled) }

	// Successfully read two bytes using a context whose cancellation we control.
	ctx, cancel := context.WithCancel(context.Background())
	buf := make([]byte, 2)
	n, cacheHit, err := t.rr.wrapped.ReadAt(ctx, buf, 1)

	ExpectFalse(cacheHit)
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
	t.rr.wrapped.reader = io.NopCloser(strings.NewReader("xxx"))
	t.rr.wrapped.cancel = func() {}
	t.rr.wrapped.start = 2
	t.rr.wrapped.limit = 5

	// The bucket should be asked to read the entire object, even though we only
	// ask for readSize bytes below, to minimize the cost for GCS requests.
	r := strings.NewReader(strings.Repeat("x", objectSize))
	rc := io.NopCloser(r)

	ExpectCall(t.bucket, "NewReader")(
		Any(),
		AllOf(rangeStartIs(1), rangeLimitIs(objectSize))).
		WillOnce(Return(rc, nil))

	// Call through.
	buf := make([]byte, readSize)
	_, cacheHit, err := t.rr.ReadAt(buf, 1)

	// Check the state now.
	ExpectFalse(cacheHit)
	ExpectEq(nil, err)
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
	t.rr.wrapped.reader = io.NopCloser(strings.NewReader("xxx"))
	t.rr.wrapped.cancel = func() {}
	t.rr.wrapped.start = 2
	t.rr.wrapped.limit = 5

	// The bucket should be asked to read expectedBytesToRead bytes.
	r := strings.NewReader(strings.Repeat("x", expectedBytesToRead))
	rc := io.NopCloser(r)

	ExpectCall(t.bucket, "NewReader")(
		Any(),
		AllOf(
			rangeStartIs(start),
			rangeLimitIs(start+expectedBytesToRead),
		)).WillOnce(Return(rc, nil))

	// Call through.
	buf := make([]byte, readSize)
	_, cacheHit, err := t.rr.ReadAt(buf, start)

	// Check the state now.
	ExpectFalse(cacheHit)
	AssertEq(nil, err)
	ExpectEq(start+expectedBytesToRead, t.rr.wrapped.limit)
}

func (t *RandomReaderTest) UpgradesSequentialReads_ExistingReader() {
	t.object.Size = 1 << 40
	const readSize = 10

	// Simulate an existing reader at the correct offset, which will be exhausted
	// by the read below.
	const existingSize = 3
	r := strings.NewReader(strings.Repeat("x", existingSize))

	t.rr.wrapped.reader = io.NopCloser(r)
	t.rr.wrapped.cancel = func() {}
	t.rr.wrapped.start = 1
	t.rr.wrapped.limit = 1 + existingSize

	// The bucket should be asked to read up to the end of the object.
	r = strings.NewReader(strings.Repeat("x", readSize-existingSize))
	rc := io.NopCloser(r)

	ExpectCall(t.bucket, "NewReader")(
		Any(),
		AllOf(rangeStartIs(1+existingSize), rangeLimitIs(1+existingSize+sequentialReadSizeInBytes))).
		WillOnce(Return(rc, nil))

	// Call through.
	buf := make([]byte, readSize)
	_, cacheHit, err := t.rr.ReadAt(buf, 1)

	// Check the state now.
	ExpectFalse(cacheHit)
	AssertEq(nil, err)
	ExpectEq(1+readSize, t.rr.wrapped.start)
	// Limit is same as the byteRange of last GCS call made.
	ExpectEq(1+existingSize+sequentialReadSizeInBytes, t.rr.wrapped.limit)
}

func (t *RandomReaderTest) UpgradesSequentialReads_NoExistingReader() {
	t.object.Size = 1 << 40
	const readSize = 1 * MB
	// Set up the custom randomReader.
	rr := NewRandomReader(t.object, t.bucket, readSize/MB, nil, false)
	t.rr.wrapped = rr.(*randomReader)

	// Simulate a previous exhausted reader that ended at the offset from which
	// we read below.
	t.rr.wrapped.start = 1
	t.rr.wrapped.limit = 1

	// The bucket should be asked to read up to the end of the object.
	r := strings.NewReader(strings.Repeat("x", readSize))
	rc := io.NopCloser(r)

	ExpectCall(t.bucket, "NewReader")(
		Any(),
		AllOf(rangeStartIs(1), rangeLimitIs(1+readSize))).
		WillOnce(Return(rc, nil))

	// Call through.
	buf := make([]byte, readSize)
	_, cacheHit, err := t.rr.ReadAt(buf, 1)

	// Check the state now.
	ExpectFalse(cacheHit)
	ExpectEq(nil, err)
	ExpectEq(1+readSize, t.rr.wrapped.start)
	ExpectEq(1+readSize, t.rr.wrapped.limit)
}

func (t *RandomReaderTest) SequentialReads_NoExistingReader_requestedSizeGreaterThanChunkSize() {
	t.object.Size = 1 << 40
	const chunkSize = 1 * MB
	const readSize = 3 * MB
	// Set up the custom randomReader.
	rr := NewRandomReader(t.object, t.bucket, chunkSize/MB, nil, false)
	t.rr.wrapped = rr.(*randomReader)
	// Create readers for each chunk.
	chunk1Reader := strings.NewReader(strings.Repeat("x", chunkSize))
	chunk1RC := io.NopCloser(chunk1Reader)
	chunk2Reader := strings.NewReader(strings.Repeat("x", chunkSize))
	chunk2RC := io.NopCloser(chunk2Reader)
	chunk3Reader := strings.NewReader(strings.Repeat("x", chunkSize))
	chunk3RC := io.NopCloser(chunk3Reader)
	// Mock the NewReader calls to return chunkReaders created above.
	// We will make 3 GCS calls to satisfy the requested read size. But since we
	// already have a reader with 'existingSize' data, we will first read that data
	// and then make GCS calls. So call sequence is
	//  [0, chunkSize) -> newReader
	//  [hunkSize, chunkSize*2) -> newReader
	//  [chunkSize*2, chunkSize*3) -> newReader
	ExpectCall(t.bucket, "NewReader")(
		Any(),
		AllOf(rangeStartIs(0), rangeLimitIs(chunkSize))).
		WillOnce(Return(chunk1RC, nil))
	ExpectCall(t.bucket, "NewReader")(
		Any(),
		AllOf(rangeStartIs(chunkSize), rangeLimitIs(chunkSize*2))).
		WillOnce(Return(chunk2RC, nil))
	ExpectCall(t.bucket, "NewReader")(
		Any(),
		AllOf(rangeStartIs(chunkSize*2), rangeLimitIs(chunkSize*3))).
		WillOnce(Return(chunk3RC, nil))

	// Call through.
	buf := make([]byte, readSize)
	_, cacheHit, err := t.rr.ReadAt(buf, 0)

	// Check the state now.
	ExpectFalse(cacheHit)
	ExpectEq(nil, err)
	// Start is the total data read.
	ExpectEq(readSize, t.rr.wrapped.start)
	// Limit is same as the byteRange of last GCS call made.
	ExpectEq(readSize, t.rr.wrapped.limit)
}

func (t *RandomReaderTest) SequentialReads_existingReader_requestedSizeGreaterThanChunkSize() {
	t.object.Size = 1 << 40
	const chunkSize = 1 * MB
	const readSize = 3 * MB
	// Set up the custom randomReader.
	rr := NewRandomReader(t.object, t.bucket, chunkSize/MB, nil, false)
	t.rr.wrapped = rr.(*randomReader)
	// Simulate an existing reader at the correct offset, which will be exhausted
	// by the read below.
	const existingSize = 3
	r := strings.NewReader(strings.Repeat("x", existingSize))
	t.rr.wrapped.reader = io.NopCloser(r)
	t.rr.wrapped.cancel = func() {}
	t.rr.wrapped.start = 0
	t.rr.wrapped.limit = existingSize
	// Create readers for each chunk.
	chunk1Reader := strings.NewReader(strings.Repeat("x", chunkSize))
	chunk1RC := io.NopCloser(chunk1Reader)
	chunk2Reader := strings.NewReader(strings.Repeat("x", chunkSize))
	chunk2RC := io.NopCloser(chunk2Reader)
	chunk3Reader := strings.NewReader(strings.Repeat("x", chunkSize))
	chunk3RC := io.NopCloser(chunk3Reader)
	// Mock the NewReader calls to return chunkReaders created above.
	// We will make 3 GCS calls to satisfy the requested read size. But since we
	// already have a reader with 'existingSize' data, we will first read that data
	// and then make GCS calls. So call sequence is
	//  [0, existingSize) -> existing reader
	//  [existingSize, existingSize+chunkSize) -> newReader
	//  [existingSize+chunkSize, existingSize+chunkSize*2) -> newReader
	//  [existingSize+chunkSize*2, existingSize+chunkSize*3) -> newReader
	ExpectCall(t.bucket, "NewReader")(
		Any(),
		AllOf(rangeStartIs(existingSize), rangeLimitIs(existingSize+chunkSize))).
		WillOnce(Return(chunk1RC, nil))
	ExpectCall(t.bucket, "NewReader")(
		Any(),
		AllOf(rangeStartIs(existingSize+chunkSize), rangeLimitIs(existingSize+chunkSize*2))).
		WillOnce(Return(chunk2RC, nil))
	ExpectCall(t.bucket, "NewReader")(
		Any(),
		AllOf(rangeStartIs(existingSize+chunkSize*2), rangeLimitIs(existingSize+chunkSize*3))).
		WillOnce(Return(chunk3RC, nil))

	// Call through.
	buf := make([]byte, readSize)
	_, cacheHit, err := t.rr.ReadAt(buf, 0)

	// Check the state now.
	ExpectFalse(cacheHit)
	ExpectEq(nil, err)
	// Start is the total data read.
	ExpectEq(readSize, t.rr.wrapped.start)
	// Limit is same as the byteRange of last GCS call made.
	ExpectEq(existingSize+readSize, t.rr.wrapped.limit)
}

/******************* File cache specific tests ***********************/

func (t *RandomReaderTest) Test_ReadAt_SequentialFullObject() {
	t.rr.wrapped.fileCacheHandler = t.cacheHandler
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rc := getReadCloser(testContent)
	t.mockNewReaderCallForTestBucket(0, objectSize, rc)
	ExpectCall(t.bucket, "Name")().WillRepeatedly(Return("test"))
	buf := make([]byte, objectSize)
	_, cacheHit, err := t.rr.ReadAt(buf, 0)
	ExpectFalse(cacheHit)
	ExpectEq(nil, err)
	ExpectTrue(reflect.DeepEqual(testContent, buf))

	_, cacheHit, err = t.rr.ReadAt(buf, 0)

	ExpectTrue(cacheHit)
	ExpectEq(nil, err)
	ExpectTrue(reflect.DeepEqual(testContent, buf))
}

func (t *RandomReaderTest) Test_ReadAt_SequentialRangeRead() {
	t.rr.wrapped.fileCacheHandler = t.cacheHandler
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rc := getReadCloser(testContent)
	t.mockNewReaderCallForTestBucket(0, objectSize, rc)
	ExpectCall(t.bucket, "Name")().WillRepeatedly(Return("test"))
	start := 0
	end := 10 // not included
	AssertLt(end, objectSize)
	buf := make([]byte, end-start)

	_, cacheHit, err := t.rr.ReadAt(buf, int64(start))

	ExpectFalse(cacheHit)
	ExpectEq(nil, err)
	ExpectTrue(reflect.DeepEqual(testContent[start:end], buf))
}

func (t *RandomReaderTest) Test_ReadAt_SequentialSubsequentReadOffsetLessThanReadChunkSize() {
	t.rr.wrapped.fileCacheHandler = t.cacheHandler
	t.object.Size = 20 * util.MiB
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rc := getReadCloser(testContent)
	t.mockNewReaderCallForTestBucket(0, objectSize, rc)
	ExpectCall(t.bucket, "Name")().WillRepeatedly(Return("test"))
	start1 := 0
	end1 := util.MiB // not included
	AssertLt(end1, objectSize)
	// First call from offset 0 - sequential read
	buf := make([]byte, end1-start1)
	_, cacheHit, err := t.rr.ReadAt(buf, int64(start1))
	ExpectFalse(cacheHit)
	ExpectEq(nil, err)
	ExpectTrue(reflect.DeepEqual(testContent[start1:end1], buf))
	start2 := 3*util.MiB + 4
	end2 := start2 + util.MiB
	buf2 := make([]byte, end2-start2)

	// Assuming start2 offset download in progress
	_, cacheHit, err = t.rr.ReadAt(buf2, int64(start2))

	ExpectTrue(cacheHit)
	ExpectEq(nil, err)
	ExpectTrue(reflect.DeepEqual(testContent[start2:end2], buf2))
}

func (t *RandomReaderTest) Test_ReadAt_RandomReadNotStartWithZeroOffsetWhenCacheForRangeReadIsFalse() {
	t.rr.wrapped.fileCacheHandler = t.cacheHandler
	objectSize := t.object.Size
	t.rr.wrapped.cacheFileForRangeRead = false
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	start := 5
	end := 10 // not included
	rc := getReadCloser(testContent[start:])
	t.mockNewReaderCallForTestBucket(uint64(start), objectSize, rc)
	ExpectCall(t.bucket, "Name")().WillRepeatedly(Return("test"))
	buf := make([]byte, end-start)
	_, cacheHit, err := t.rr.ReadAt(buf, int64(start))
	ExpectFalse(cacheHit)
	ExpectEq(nil, err)
	ExpectTrue(reflect.DeepEqual(testContent[start:end], buf))
	job := t.jobManager.CreateJobIfNotExists(t.object, t.bucket)
	jobStatus := job.GetStatus()
	ExpectTrue(jobStatus.Name == downloader.NotStarted)

	// Second read call should be a cache miss
	_, cacheHit, err = t.rr.ReadAt(buf, int64(start))

	ExpectEq(nil, err)
	ExpectFalse(cacheHit)
}

func (t *RandomReaderTest) Test_ReadAt_RandomReadNotStartWithZeroOffsetWhenCacheForRangeReadIsTrue() {
	t.rr.wrapped.fileCacheHandler = t.cacheHandler
	objectSize := t.object.Size
	t.rr.wrapped.cacheFileForRangeRead = true
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	start := 5
	end := 10 // not included
	rc := getReadCloser(testContent[start:])
	t.mockNewReaderCallForTestBucket(uint64(start), objectSize, rc) // Mock for random-reader's NewReader call
	rc2 := getReadCloser(testContent)
	t.mockNewReaderCallForTestBucket(0, objectSize, rc2) // Mock for download job's NewReader call
	ExpectCall(t.bucket, "Name")().WillRepeatedly(Return("test"))
	buf := make([]byte, end-start)

	_, cacheHit, err := t.rr.ReadAt(buf, int64(start))

	ExpectFalse(cacheHit)
	ExpectEq(nil, err)
	ExpectTrue(reflect.DeepEqual(testContent[start:end], buf))
	job := t.jobManager.GetJob(t.object.Name, t.bucket.Name())
	ExpectTrue(job == nil || job.GetStatus().Name == downloader.Downloading)
}

func (t *RandomReaderTest) Test_ReadAt_SequentialToRandomSubsequentReadOffsetMoreThanReadChunkSize() {
	t.rr.wrapped.fileCacheHandler = t.cacheHandler
	t.object.Size = 20 * util.MiB
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rc := getReadCloser(testContent)
	t.mockNewReaderCallForTestBucket(0, objectSize, rc)
	ExpectCall(t.bucket, "Name")().WillRepeatedly(Return("test"))
	start1 := 0
	end1 := util.MiB // not included
	AssertLt(end1, objectSize)
	// First call from offset 0 - sequential read
	buf := make([]byte, end1-start1)
	_, cacheHit, err := t.rr.ReadAt(buf, int64(start1))
	ExpectFalse(cacheHit)
	ExpectEq(nil, err)
	ExpectTrue(reflect.DeepEqual(testContent[start1:end1], buf))
	start2 := 16*util.MiB + 4
	end2 := start2 + util.MiB
	rc2 := getReadCloser(testContent[start2:])
	t.mockNewReaderCallForTestBucket(uint64(start2), objectSize, rc2)
	buf2 := make([]byte, end2-start2)

	// Assuming start2 offset download in progress
	_, cacheHit, err = t.rr.ReadAt(buf2, int64(start2))

	ExpectFalse(cacheHit)
	ExpectEq(nil, err)
	ExpectTrue(reflect.DeepEqual(testContent[start2:end2], buf2))
}

func (t *RandomReaderTest) Test_ReadAt_SequentialToRandomSubsequentReadOffsetLessThanPrevious() {
	t.rr.wrapped.fileCacheHandler = t.cacheHandler
	t.object.Size = 20 * util.MiB
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rc := getReadCloser(testContent)
	t.mockNewReaderCallForTestBucket(0, objectSize, rc)
	ExpectCall(t.bucket, "Name")().WillRepeatedly(Return("test"))
	start1 := 0
	end1 := util.MiB // not included
	AssertLt(end1, objectSize)
	// First call from offset 0 - sequential read
	buf := make([]byte, end1-start1)
	_, cacheHit, err := t.rr.ReadAt(buf, int64(start1))
	ExpectFalse(cacheHit)
	ExpectEq(nil, err)
	ExpectTrue(reflect.DeepEqual(testContent[start1:end1], buf))
	start2 := 16*util.MiB + 4
	end2 := start2 + util.MiB
	rc2 := getReadCloser(testContent[start2:])
	t.mockNewReaderCallForTestBucket(uint64(start2), objectSize, rc2)
	buf2 := make([]byte, end2-start2)
	// Assuming start2 offset download in progress
	_, cacheHit, err = t.rr.ReadAt(buf2, int64(start2))
	ExpectFalse(cacheHit)
	ExpectEq(nil, err)
	ExpectTrue(reflect.DeepEqual(testContent[start2:end2], buf2))
	start3 := util.MiB
	end3 := start3 + util.MiB
	buf3 := make([]byte, end3-start3)

	_, cacheHit, err = t.rr.ReadAt(buf3, int64(start3))

	ExpectEq(nil, err)
	ExpectTrue(cacheHit)
	ExpectTrue(reflect.DeepEqual(testContent[start3:end3], buf3))
}

func (t *RandomReaderTest) Test_ReadAt_CacheMissDueToInvalidJob() {
	t.rr.wrapped.fileCacheHandler = t.cacheHandler
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rc1 := getReadCloser(testContent)
	t.mockNewReaderCallForTestBucket(0, objectSize, rc1)
	ExpectCall(t.bucket, "Name")().WillRepeatedly(Return("test"))
	buf := make([]byte, objectSize)
	_, cacheHit, err := t.rr.ReadAt(buf, 0)
	AssertEq(nil, err)
	ExpectFalse(cacheHit)
	AssertTrue(reflect.DeepEqual(testContent, buf))
	job := t.jobManager.GetJob(t.object.Name, t.bucket.Name())
	AssertTrue(job == nil || job.GetStatus().Name == downloader.Completed)
	err = t.rr.wrapped.fileCacheHandler.InvalidateCache(t.object.Name, t.bucket.Name())
	AssertEq(nil, err)
	// Second reader (rc2) is required, since first reader (rc) is completely read.
	// Reading again will return EOF.
	rc2 := getReadCloser(testContent)
	t.mockNewReaderCallForTestBucket(0, objectSize, rc2)

	_, cacheHit, err = t.rr.ReadAt(buf, 0)

	ExpectEq(nil, err)
	ExpectFalse(cacheHit)
	ExpectTrue(reflect.DeepEqual(testContent, buf))
	ExpectEq(nil, t.rr.wrapped.fileCacheHandle)
}

func (t *RandomReaderTest) Test_ReadAt_CachePopulatedAndThenCacheMissDueToInvalidJob() {
	t.rr.wrapped.fileCacheHandler = t.cacheHandler
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rc1 := getReadCloser(testContent)
	t.mockNewReaderCallForTestBucket(0, objectSize, rc1)
	ExpectCall(t.bucket, "Name")().WillRepeatedly(Return("test"))
	buf := make([]byte, objectSize)
	_, cacheHit, err := t.rr.ReadAt(buf, 0)
	AssertEq(nil, err)
	AssertFalse(cacheHit)
	AssertTrue(reflect.DeepEqual(testContent, buf))
	job := t.jobManager.GetJob(t.object.Name, t.bucket.Name())
	AssertTrue(job == nil || job.GetStatus().Name == downloader.Completed)
	err = t.rr.wrapped.fileCacheHandler.InvalidateCache(t.object.Name, t.bucket.Name())
	AssertEq(nil, err)
	// Second reader (rc2) is required, since first reader (rc) is completely read.
	// Reading again will return EOF.
	rc2 := getReadCloser(testContent)
	t.mockNewReaderCallForTestBucket(0, objectSize, rc2)
	_, cacheHit, err = t.rr.ReadAt(buf, 0)
	ExpectEq(nil, err)
	ExpectFalse(cacheHit)
	ExpectTrue(reflect.DeepEqual(testContent, buf))
	ExpectEq(nil, t.rr.wrapped.fileCacheHandle)
	rc3 := getReadCloser(testContent)
	t.mockNewReaderCallForTestBucket(0, objectSize, rc3)

	_, cacheHit, err = t.rr.ReadAt(buf, 0)

	ExpectEq(nil, err)
	ExpectFalse(cacheHit)
	ExpectTrue(reflect.DeepEqual(testContent, buf))
	ExpectNe(nil, t.rr.wrapped.fileCacheHandle)
}

func (t *RandomReaderTest) Test_ReadAt_CachePopulatedAndThenCacheMissDueToInvalidFileHandle() {
	t.rr.wrapped.fileCacheHandler = t.cacheHandler
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rc1 := getReadCloser(testContent)
	t.mockNewReaderCallForTestBucket(0, objectSize, rc1)
	ExpectCall(t.bucket, "Name")().WillRepeatedly(Return("test"))
	buf := make([]byte, objectSize)
	_, cacheHit, err := t.rr.ReadAt(buf, 0)
	AssertEq(nil, err)
	AssertFalse(cacheHit)
	AssertTrue(reflect.DeepEqual(testContent, buf))
	AssertNe(nil, t.rr.wrapped.fileCacheHandle)
	err = t.rr.wrapped.fileCacheHandle.Close()
	AssertEq(nil, err)
	// Second reader (rc2) is required, since first reader (rc) is completely read.
	// Reading again will return EOF.
	rc2 := getReadCloser(testContent)
	t.mockNewReaderCallForTestBucket(0, objectSize, rc2)
	_, cacheHit, err = t.rr.ReadAt(buf, 0)
	ExpectEq(nil, err)
	ExpectFalse(cacheHit)
	ExpectTrue(reflect.DeepEqual(testContent, buf))
	ExpectEq(nil, t.rr.wrapped.fileCacheHandle)
	rc3 := getReadCloser(testContent)
	t.mockNewReaderCallForTestBucket(0, objectSize, rc3)

	_, cacheHit, err = t.rr.ReadAt(buf, 0)

	ExpectEq(nil, err)
	ExpectTrue(cacheHit)
	ExpectTrue(reflect.DeepEqual(testContent, buf))
	ExpectNe(nil, t.rr.wrapped.fileCacheHandle)
}

func (t *RandomReaderTest) Test_ReadAt_IfCacheFileGetsDeleted() {
	t.rr.wrapped.fileCacheHandler = t.cacheHandler
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rc1 := getReadCloser(testContent)
	t.mockNewReaderCallForTestBucket(0, objectSize, rc1)
	ExpectCall(t.bucket, "Name")().WillRepeatedly(Return("test"))
	buf := make([]byte, objectSize)
	_, cacheHit, err := t.rr.ReadAt(buf, 0)
	AssertEq(nil, err)
	AssertFalse(cacheHit)
	AssertTrue(reflect.DeepEqual(testContent, buf))
	AssertNe(nil, t.rr.wrapped.fileCacheHandle)
	err = t.rr.wrapped.fileCacheHandle.Close()
	AssertEq(nil, err)
	t.rr.wrapped.fileCacheHandle = nil
	// Delete the local cache file.
	filePath := util.GetDownloadPath(t.cacheDir, util.GetObjectPath(t.bucket.Name(), t.object.Name))
	err = os.Remove(filePath)
	AssertEq(nil, err)
	// Second reader (rc2) is required, since first reader (rc) is completely read.
	// Reading again will return EOF.
	rc2 := getReadCloser(testContent)
	t.mockNewReaderCallForTestBucket(0, objectSize, rc2)

	_, _, err = t.rr.ReadAt(buf, 0)

	AssertNe(nil, err)
	ExpectTrue(strings.Contains(err.Error(), util.FileNotPresentInCacheErrMsg))
}

func (t *RandomReaderTest) Test_ReadAt_IfCacheFileGetsDeletedWithCacheHandleOpen() {
	t.rr.wrapped.fileCacheHandler = t.cacheHandler
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rc1 := getReadCloser(testContent)
	t.mockNewReaderCallForTestBucket(0, objectSize, rc1)
	ExpectCall(t.bucket, "Name")().WillRepeatedly(Return("test"))
	buf := make([]byte, objectSize)
	_, cacheHit, err := t.rr.ReadAt(buf, 0)
	AssertEq(nil, err)
	AssertFalse(cacheHit)
	AssertTrue(reflect.DeepEqual(testContent, buf))
	AssertNe(nil, t.rr.wrapped.fileCacheHandle)
	// Delete the local cache file.
	filePath := util.GetDownloadPath(t.cacheDir, util.GetObjectPath(t.bucket.Name(), t.object.Name))
	err = os.Remove(filePath)
	AssertEq(nil, err)

	// Read via cache only, as we have old fileHandle open and linux
	// doesn't delete the file until the fileHandle count for the file is zero.
	_, cacheHit, err = t.rr.ReadAt(buf, 0)

	AssertEq(nil, err)
	ExpectTrue(cacheHit)
	ExpectTrue(reflect.DeepEqual(testContent, buf))
	ExpectNe(nil, t.rr.wrapped.fileCacheHandle)
}

func (t *RandomReaderTest) Test_ReadAt_FailedJobRestartAndCacheHit() {
	t.rr.wrapped.fileCacheHandler = t.cacheHandler
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rc1 := getReadCloser(testContent)
	// First NewReader-call throws error, hence async job fails.
	// Later NewReader-call returns a valid readCloser object hence fallback to
	// GCS read will succeed.
	ExpectCall(t.bucket, "NewReader")(Any(), Any()).
		WillOnce(Return(nil, errors.New(""))).WillRepeatedly(Return(rc1, nil))
	ExpectCall(t.bucket, "Name")().WillRepeatedly(Return("test"))
	buf := make([]byte, objectSize)
	_, cacheHit, err := t.rr.ReadAt(buf, 0)
	AssertEq(nil, err)
	ExpectFalse(cacheHit)
	AssertTrue(reflect.DeepEqual(testContent, buf))
	job := t.jobManager.GetJob(t.object.Name, t.bucket.Name())
	AssertTrue(job == nil || job.GetStatus().Name == downloader.Failed)
	// Second reader (rc2) is required, since first reader (rc) is completely read.
	// Reading again will return EOF.
	rc2 := getReadCloser(testContent)
	t.mockNewReaderCallForTestBucket(0, objectSize, rc2)
	// This call will populate the cache again.
	_, cacheHit, err = t.rr.ReadAt(buf, 0)
	ExpectEq(nil, err)
	ExpectFalse(cacheHit)
	ExpectTrue(reflect.DeepEqual(testContent, buf))
	ExpectNe(nil, t.rr.wrapped.fileCacheHandle)

	_, cacheHit, err = t.rr.ReadAt(buf, 0)

	ExpectEq(nil, err)
	ExpectTrue(cacheHit)
	ExpectTrue(reflect.DeepEqual(testContent, buf))
	ExpectNe(nil, t.rr.wrapped.fileCacheHandle)
}

// Only writing two unit tests for tryReadingFromFileCache as
// unit tests of ReadAt method covers all the workflows.
func (t *RandomReaderTest) Test_tryReadingFromFileCache_CacheHit() {
	t.rr.wrapped.fileCacheHandler = t.cacheHandler
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rc := getReadCloser(testContent)
	t.mockNewReaderCallForTestBucket(0, objectSize, rc)
	ExpectCall(t.bucket, "Name")().WillRepeatedly(Return("test"))
	buf := make([]byte, objectSize)
	// First read will be a cache miss.
	_, cacheHit, err := t.rr.wrapped.tryReadingFromFileCache(t.rr.ctx, buf, 0)
	ExpectFalse(cacheHit)
	ExpectEq(nil, err)
	ExpectTrue(reflect.DeepEqual(testContent, buf))

	// Second read will be a cache hit.
	_, cacheHit, err = t.rr.wrapped.tryReadingFromFileCache(t.rr.ctx, buf, 0)
	ExpectTrue(cacheHit)
	ExpectEq(nil, err)
	ExpectTrue(reflect.DeepEqual(testContent, buf))
}

func (t *RandomReaderTest) Test_tryReadingFromFileCache_CacheMiss() {
	t.rr.wrapped.fileCacheHandler = t.cacheHandler
	t.rr.wrapped.cacheFileForRangeRead = false
	start := 5
	end := 10
	ExpectCall(t.bucket, "Name")().WillRepeatedly(Return("test"))
	buf := make([]byte, end-start)

	_, cacheHit, err := t.rr.wrapped.tryReadingFromFileCache(t.rr.ctx, buf, int64(start))

	ExpectFalse(cacheHit)
	ExpectEq(nil, err)
}

func (t *RandomReaderTest) Test_ReadAt_OffsetEqualToObjectSize() {
	t.rr.wrapped.fileCacheHandler = t.cacheHandler
	t.object.Size = util.MiB
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rc := getReadCloser(testContent)
	t.mockNewReaderCallForTestBucket(0, objectSize, rc)
	ExpectCall(t.bucket, "Name")().WillRepeatedly(Return("test"))
	start1 := 0
	end1 := util.MiB // equal to objectSize
	// First call from offset 0 - objectSize
	buf := make([]byte, end1-start1)
	_, cacheHit, err := t.rr.ReadAt(buf, int64(start1))
	ExpectFalse(cacheHit)
	ExpectEq(nil, err)
	ExpectTrue(reflect.DeepEqual(testContent[start1:end1], buf))
	start2 := util.MiB // offset equal to objectSize
	end2 := start2 + util.MiB
	buf2 := make([]byte, end2-start2)

	// read for offset equal to objectSize
	n, cacheHit, err := t.rr.ReadAt(buf2, int64(start2))

	// nothing should be read
	ExpectFalse(cacheHit)
	ExpectEq(io.EOF, err)
	ExpectEq(0, n)
}

func (t *RandomReaderTest) Test_Destroy_NilCacheHandle() {
	t.rr.wrapped.fileCacheHandler = t.cacheHandler

	t.rr.Destroy()

	ExpectEq(nil, t.rr.wrapped.fileCacheHandle)
}

func (t *RandomReaderTest) Test_Destroy_NonNilCacheHandle() {
	t.rr.wrapped.fileCacheHandler = t.cacheHandler
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rc := getReadCloser(testContent)
	t.mockNewReaderCallForTestBucket(0, objectSize, rc)
	ExpectCall(t.bucket, "Name")().WillRepeatedly(Return("test"))
	buf := make([]byte, objectSize)
	_, cacheHit, err := t.rr.wrapped.tryReadingFromFileCache(t.rr.ctx, buf, 0)
	AssertFalse(cacheHit)
	AssertEq(nil, err)
	AssertTrue(reflect.DeepEqual(testContent, buf))
	AssertNe(nil, t.rr.wrapped.fileCacheHandle)

	t.rr.wrapped.Destroy()

	ExpectEq(nil, t.rr.wrapped.fileCacheHandle)
}

// TODO (raj-prince) - to add unit tests for failed scenario while reading via cache.
// This requires mocking CacheHandle object, whose read method will return some unexpected
// error.
