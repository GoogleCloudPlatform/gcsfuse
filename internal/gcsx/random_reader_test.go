// Copyright 2015 Google LLC
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
	"os"
	"path"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/file"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/file/downloader"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/gcsfuse_errors"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	testutil "github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/jacobsa/fuse/fuseops"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"golang.org/x/net/context"
)

// NOTE: Please add new tests in random_reader_stretchr_test.go file. This file
// is deprecated and these tests will be moved to the random_reader_stretchr_test.go
func TestRandomReader(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Invariant-checking random reader
////////////////////////////////////////////////////////////////////////

type checkingRandomReader struct {
	ctx     context.Context
	wrapped *randomReader
}

func (rr *checkingRandomReader) ReadAt(p []byte, offset int64) (ObjectData, error) {
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
	object       *gcs.MinObject
	bucket       storage.MockBucket
	rr           checkingRandomReader
	cacheDir     string
	jobManager   *downloader.JobManager
	cacheHandler *file.CacheHandler
	bucketType   gcs.BucketType
}

func init() {
	RegisterTestSuite(&RandomReaderTest{bucketType: gcs.BucketType{}})
	RegisterTestSuite(&RandomReaderTest{bucketType: gcs.BucketType{Zonal: true, Hierarchical: true}})
}

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
	lruCache := lru.NewCache(cacheMaxSize)
	t.jobManager = downloader.NewJobManager(lruCache, util.DefaultFilePerm, util.DefaultDirPerm, t.cacheDir, sequentialReadSizeInMb, &cfg.FileCacheConfig{
		EnableCrc: false,
	}, metrics.NewNoopMetrics())
	t.cacheHandler = file.NewCacheHandler(lruCache, t.jobManager, t.cacheDir, util.DefaultFilePerm, util.DefaultDirPerm, "")

	// Set up the reader.
	rr := NewRandomReader(t.object, t.bucket, sequentialReadSizeInMb, nil, false, metrics.NewNoopMetrics(), nil, nil)
	t.rr.wrapped = rr.(*randomReader)
}

func (t *RandomReaderTest) TearDown() {
	t.rr.Destroy()
}

func (t *RandomReaderTest) mockNewReaderWithHandleCallForTestBucket(start uint64, limit uint64, rd gcs.StorageReader) {
	ExpectCall(t.bucket, "NewReaderWithReadHandle")(
		Any(), AllOf(rangeStartIs(start), rangeLimitIs(limit))).
		WillRepeatedly(Return(rd, nil))
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *RandomReaderTest) EmptyRead() {
	// Nothing should happen.
	buf := make([]byte, 0)

	objectData, err := t.rr.ReadAt(buf, 0)

	ExpectEq(0, objectData.Size)
	ExpectEq(nil, err)
}

func (t *RandomReaderTest) ReadAtEndOfObject() {
	buf := make([]byte, 1)

	objectData, err := t.rr.ReadAt(buf, int64(t.object.Size))

	ExpectEq(0, objectData.Size)
	ExpectEq(io.EOF, err)
}

func (t *RandomReaderTest) ReadPastEndOfObject() {
	buf := make([]byte, 1)

	objectData, err := t.rr.ReadAt(buf, int64(t.object.Size)+1)

	ExpectFalse(objectData.CacheHit)
	ExpectEq(0, objectData.Size)
	ExpectEq(io.EOF, err)
}

func (t *RandomReaderTest) NoExistingReader() {
	// The bucket should be called to set up a new reader.
	ExpectCall(t.bucket, "NewReaderWithReadHandle")(Any(), Any()).
		WillOnce(Return(nil, errors.New("")))
	ExpectCall(t.bucket, "BucketType")().Times(2).WillOnce(Return(t.bucketType))
	buf := make([]byte, 1)

	_, err := t.rr.ReadAt(buf, 0)

	AssertNe(nil, err)
}

func (t *RandomReaderTest) ExistingReader_ReadAtOffsetAfterTheReaderPosition() {
	var currentStartOffset int64 = 2
	var readerLimit int64 = 15
	var readAtOffset int64 = 10
	var readSize int64 = 1
	var expectedStartOffsetAfterRead = readAtOffset + readSize
	// Simulate an existing reader.
	nopCloser := io.NopCloser(strings.NewReader(strings.Repeat("x", int(readerLimit))))
	rc := &fake.FakeReader{ReadCloser: nopCloser}
	t.rr.wrapped.reader = rc
	t.rr.wrapped.cancel = func() {}
	t.rr.wrapped.start = currentStartOffset
	t.rr.wrapped.limit = readerLimit

	ExpectCall(t.bucket, "BucketType")().Times(2).WillOnce(Return(t.bucketType))

	buf := make([]byte, readSize)
	_, err := t.rr.ReadAt(buf, readAtOffset)

	AssertEq(nil, err)
	ExpectThat(rc, DeepEquals(t.rr.wrapped.reader))
	ExpectEq(expectedStartOffsetAfterRead, t.rr.wrapped.start)
	ExpectEq(readerLimit, t.rr.wrapped.limit)
}

func (t *RandomReaderTest) NewReaderReturnsError() {
	ExpectCall(t.bucket, "NewReaderWithReadHandle")(Any(), Any()).
		WillOnce(Return(nil, errors.New("taco")))
	ExpectCall(t.bucket, "BucketType")().Times(2).WillOnce(Return(t.bucketType))
	buf := make([]byte, 1)

	_, err := t.rr.ReadAt(buf, 0)

	ExpectThat(err, Error(HasSubstr("NewReaderWithReadHandle")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *RandomReaderTest) ReaderFails() {
	// Bucket
	r := iotest.OneByteReader(iotest.TimeoutReader(strings.NewReader("xxx")))
	rc := &fake.FakeReader{ReadCloser: io.NopCloser(r)}

	ExpectCall(t.bucket, "NewReaderWithReadHandle")(Any(), Any()).
		WillOnce(Return(rc, nil))
	ExpectCall(t.bucket, "BucketType")().Times(2).WillOnce(Return(t.bucketType))

	// Call
	buf := make([]byte, 3)
	_, err := t.rr.ReadAt(buf, 0)

	ExpectThat(err, Error(HasSubstr("readFull")))
	ExpectThat(err, Error(HasSubstr(iotest.ErrTimeout.Error())))
}

func (t *RandomReaderTest) ReaderNotExhausted() {
	// Set up a reader that has three bytes left to give.
	cc := &countingCloser{
		Reader: strings.NewReader("abc"),
	}
	rc := &fake.FakeReader{ReadCloser: cc}

	ExpectCall(t.bucket, "BucketType")().Times(2).WillOnce(Return(t.bucketType))

	t.rr.wrapped.reader = rc
	t.rr.wrapped.cancel = func() {}
	t.rr.wrapped.start = 1
	t.rr.wrapped.limit = 4

	// Read two bytes.
	buf := make([]byte, 2)
	objectData, err := t.rr.ReadAt(buf, 1)

	ExpectFalse(objectData.CacheHit)
	ExpectEq(2, objectData.Size)
	ExpectEq(nil, err)
	ExpectEq("ab", string(buf[:objectData.Size]))

	ExpectEq(0, cc.closeCount)
	ExpectEq(rc, t.rr.wrapped.reader)
	ExpectEq(3, t.rr.wrapped.start)
	ExpectEq(4, t.rr.wrapped.limit)
}

func (t *RandomReaderTest) ReaderExhausted_ReadFinished() {
	// Set up a reader that has three bytes left to give.
	rc := &countingCloser{
		Reader: strings.NewReader("abc"),
	}

	ExpectCall(t.bucket, "BucketType")().Times(2).WillOnce(Return(t.bucketType))

	t.rr.wrapped.reader = &fake.FakeReader{ReadCloser: rc}
	t.rr.wrapped.cancel = func() {}
	t.rr.wrapped.start = 1
	t.rr.wrapped.limit = 4

	// Read three bytes.
	buf := make([]byte, 3)
	objectData, err := t.rr.ReadAt(buf, 1)

	ExpectFalse(objectData.CacheHit)
	ExpectEq(3, objectData.Size)
	ExpectEq(nil, err)
	ExpectEq("abc", string(buf[:objectData.Size]))

	ExpectEq(1, rc.closeCount)
	ExpectEq(nil, t.rr.wrapped.reader)
	ExpectEq(nil, t.rr.wrapped.cancel)
	ExpectEq(4, t.rr.wrapped.limit)
}

func (t *RandomReaderTest) PropagatesCancellation() {
	// Set up a reader that will block until we tell it to return.
	finishRead := make(chan struct{})
	rc := io.NopCloser(&blockingReader{finishRead})

	ExpectCall(t.bucket, "BucketType")().Times(2).WillOnce(Return(t.bucketType))

	t.rr.wrapped.reader = &fake.FakeReader{ReadCloser: rc}
	t.rr.wrapped.start = 1
	t.rr.wrapped.limit = 4
	t.rr.wrapped.config = &cfg.Config{FileSystem: cfg.FileSystemConfig{IgnoreInterrupts: false}}

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
	t.rr.wrapped.reader = &fake.FakeReader{ReadCloser: getReadCloser([]byte("xxx"))}
	t.rr.wrapped.start = 1
	t.rr.wrapped.limit = 4

	ExpectCall(t.bucket, "BucketType")().Times(2).WillOnce(Return(t.bucketType))

	// Snoop on when cancel is called.
	cancelCalled := make(chan struct{})
	t.rr.wrapped.cancel = func() { close(cancelCalled) }

	// Successfully read two bytes using a context whose cancellation we control.
	ctx, cancel := context.WithCancel(context.Background())
	buf := make([]byte, 2)
	objectData, err := t.rr.wrapped.ReadAt(ctx, buf, 1)

	AssertEq(nil, err)
	ExpectFalse(objectData.CacheHit)
	AssertEq(2, objectData.Size)

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
	const objectSize = 2 * MiB
	t.object.Size = objectSize

	const readSize = 10
	AssertLt(readSize, objectSize)

	// Simulate an existing reader at a mismatched offset.
	t.rr.wrapped.reader = &fake.FakeReader{ReadCloser: getReadCloser([]byte("xxx"))}
	t.rr.wrapped.cancel = func() {}
	t.rr.wrapped.start = 2
	t.rr.wrapped.limit = 5

	// The bucket should be asked to read the entire object, even though we only
	// ask for readSize bytes below, to minimize the cost for GCS requests.
	r := strings.NewReader(strings.Repeat("x", objectSize))
	rc := &fake.FakeReader{ReadCloser: io.NopCloser(r)}

	ExpectCall(t.bucket, "NewReaderWithReadHandle")(
		Any(),
		AllOf(rangeStartIs(1), rangeLimitIs(objectSize))).
		WillOnce(Return(rc, nil))
	ExpectCall(t.bucket, "BucketType")().Times(2).WillOnce(Return(t.bucketType))

	// Call through.
	buf := make([]byte, readSize)
	objectData, err := t.rr.ReadAt(buf, 1)

	// Check the state now.
	ExpectFalse(objectData.CacheHit)
	ExpectEq(nil, err)
	ExpectEq(1+readSize, t.rr.wrapped.start)
	ExpectEq(objectSize, t.rr.wrapped.limit)
}

func (t *RandomReaderTest) UpgradeReadsToAverageSize() {
	t.object.Size = 1 << 40
	const totalReadBytes = 6 * MiB
	const numReads = 2
	const avgReadBytes = totalReadBytes / numReads

	const expectedBytesToRead = avgReadBytes
	const start = 1
	const readSize = 2 * minReadSize

	// Simulate an existing reader at a mismatched offset.
	t.rr.wrapped.seeks.Store(numReads)
	t.rr.wrapped.totalReadBytes.Store(totalReadBytes)
	t.rr.wrapped.reader = &fake.FakeReader{ReadCloser: getReadCloser([]byte("xxx"))}
	t.rr.wrapped.cancel = func() {}
	t.rr.wrapped.start = 2
	t.rr.wrapped.limit = 5
	t.rr.wrapped.expectedOffset.Store(2)

	// The bucket should be asked to read expectedBytesToRead bytes.
	r := strings.NewReader(strings.Repeat("x", expectedBytesToRead))
	rc := &fake.FakeReader{ReadCloser: io.NopCloser(r)}

	ExpectCall(t.bucket, "NewReaderWithReadHandle")(
		Any(),
		AllOf(
			rangeStartIs(start),
			rangeLimitIs(start+expectedBytesToRead),
		)).WillOnce(Return(rc, nil))
	ExpectCall(t.bucket, "BucketType")().Times(2).WillOnce(Return(t.bucketType))

	// Call through.
	buf := make([]byte, readSize)
	objectData, err := t.rr.ReadAt(buf, start)

	// Check the state now.
	ExpectFalse(objectData.CacheHit)
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

	t.rr.wrapped.reader = &fake.FakeReader{ReadCloser: io.NopCloser(r)}
	t.rr.wrapped.cancel = func() {}
	t.rr.wrapped.start = 1
	t.rr.wrapped.limit = 1 + existingSize

	// The bucket should be asked to read up to the end of the object.
	r = strings.NewReader(strings.Repeat("y", readSize))
	rc := &fake.FakeReader{ReadCloser: io.NopCloser(r)}

	ExpectCall(t.bucket, "NewReaderWithReadHandle")(
		Any(),
		AllOf(rangeStartIs(1), rangeLimitIs(1+sequentialReadSizeInBytes))).
		WillOnce(Return(rc, nil))
	ExpectCall(t.bucket, "BucketType")().Times(2).WillOnce(Return(t.bucketType))

	// Call through.
	buf := make([]byte, readSize)
	objectData, err := t.rr.ReadAt(buf, 1)

	// Check the state now.
	ExpectFalse(objectData.CacheHit)
	AssertEq(nil, err)
	ExpectEq(1+readSize, t.rr.wrapped.start)
	// Limit is same as the byteRange of last GCS call made.
	ExpectEq(1+sequentialReadSizeInBytes, t.rr.wrapped.limit)
}

func (t *RandomReaderTest) UpgradesSequentialReads_NoExistingReader() {
	t.object.Size = 1 << 40
	const readSize = 1 * MiB
	// Set up the custom randomReader.
	rr := NewRandomReader(t.object, t.bucket, readSize/MiB, nil, false, metrics.NewNoopMetrics(), nil, nil)
	t.rr.wrapped = rr.(*randomReader)

	// Simulate a previous exhausted reader that ended at the offset from which
	// we read below.
	t.rr.wrapped.start = 1
	t.rr.wrapped.limit = 1

	// The bucket should be asked to read up to the end of the object.
	data := strings.Repeat("x", readSize)
	r := strings.NewReader(data)
	rc := &fake.FakeReader{ReadCloser: io.NopCloser(r)}

	ExpectCall(t.bucket, "NewReaderWithReadHandle")(
		Any(),
		AllOf(rangeStartIs(1), rangeLimitIs(1+readSize))).
		WillOnce(Return(rc, nil))
	ExpectCall(t.bucket, "BucketType")().Times(2).WillOnce(Return(t.bucketType))

	// Call through.
	buf := make([]byte, readSize)
	objectData, err := t.rr.ReadAt(buf, 1)

	// Check the state now.
	ExpectFalse(objectData.CacheHit)
	ExpectEq(nil, err)
	ExpectEq(1+readSize, t.rr.wrapped.start)
	ExpectEq(1+readSize, t.rr.wrapped.limit)
}

/******************* File cache specific tests ***********************/

func (t *RandomReaderTest) Test_ReadAt_SequentialFullObject() {
	t.rr.wrapped.fileCacheHandler = t.cacheHandler
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd)
	ExpectCall(t.bucket, "Name")().WillRepeatedly(Return("test"))
	ExpectCall(t.bucket, "BucketType")().WillRepeatedly(Return(t.bucketType))
	buf := make([]byte, objectSize)
	objectData, err := t.rr.ReadAt(buf, 0)
	ExpectFalse(objectData.CacheHit)
	ExpectEq(nil, err)
	ExpectTrue(reflect.DeepEqual(testContent, buf))

	objectData, err = t.rr.ReadAt(buf, 0)

	ExpectTrue(objectData.CacheHit)
	ExpectEq(nil, err)
	ExpectTrue(reflect.DeepEqual(testContent, buf))
}

func (t *RandomReaderTest) Test_ReadAt_SequentialRangeRead() {
	t.rr.wrapped.fileCacheHandler = t.cacheHandler
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd)
	ExpectCall(t.bucket, "Name")().WillRepeatedly(Return("test"))
	ExpectCall(t.bucket, "BucketType")().WillOnce(Return(t.bucketType))
	start := 0
	end := 10 // not included
	AssertLt(end, objectSize)
	buf := make([]byte, end-start)

	objectData, err := t.rr.ReadAt(buf, int64(start))

	ExpectFalse(objectData.CacheHit)
	ExpectEq(nil, err)
	ExpectTrue(reflect.DeepEqual(testContent[start:end], buf))
}

func (t *RandomReaderTest) Test_ReadAt_SequentialSubsequentReadOffsetLessThanReadChunkSize() {
	t.rr.wrapped.fileCacheHandler = t.cacheHandler
	t.object.Size = 20 * util.MiB
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd)
	ExpectCall(t.bucket, "Name")().WillRepeatedly(Return("test"))
	ExpectCall(t.bucket, "BucketType")().WillRepeatedly(Return(t.bucketType))
	start1 := 0
	end1 := util.MiB // not included
	AssertLt(end1, objectSize)
	// First call from offset 0 - sequential read
	buf := make([]byte, end1-start1)
	objectData, err := t.rr.ReadAt(buf, int64(start1))
	ExpectFalse(objectData.CacheHit)
	ExpectEq(nil, err)
	ExpectTrue(reflect.DeepEqual(testContent[start1:end1], buf))
	start2 := 3*util.MiB + 4
	end2 := start2 + util.MiB
	buf2 := make([]byte, end2-start2)

	// Assuming start2 offset download in progress
	objectData, err = t.rr.ReadAt(buf2, int64(start2))

	ExpectTrue(objectData.CacheHit)
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
	rc := &fake.FakeReader{ReadCloser: getReadCloser(testContent[start:])}
	t.mockNewReaderWithHandleCallForTestBucket(uint64(start), objectSize, rc)
	ExpectCall(t.bucket, "Name")().WillRepeatedly(Return("test"))
	ExpectCall(t.bucket, "BucketType")().WillRepeatedly(Return(t.bucketType))
	buf := make([]byte, end-start)
	objectData, err := t.rr.ReadAt(buf, int64(start))
	ExpectFalse(objectData.CacheHit)
	ExpectEq(nil, err)
	ExpectTrue(reflect.DeepEqual(testContent[start:end], buf))
	job := t.jobManager.CreateJobIfNotExists(t.object, t.bucket)
	jobStatus := job.GetStatus()
	ExpectTrue(jobStatus.Name == downloader.NotStarted)

	// Second read call should be a cache miss
	objectData, err = t.rr.ReadAt(buf, int64(start))

	ExpectEq(nil, err)
	ExpectFalse(objectData.CacheHit)
}

func (t *RandomReaderTest) Test_ReadAt_RandomReadNotStartWithZeroOffsetWhenCacheForRangeReadIsTrue() {
	t.rr.wrapped.fileCacheHandler = t.cacheHandler
	objectSize := t.object.Size
	t.rr.wrapped.cacheFileForRangeRead = true
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	start := 5
	end := 10 // not included
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent[start:])}
	// Mock for random-reader's NewReader call
	t.mockNewReaderWithHandleCallForTestBucket(uint64(start), objectSize, rd)
	rd1 := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	// Mock for download job's NewReader call
	t.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd1)
	ExpectCall(t.bucket, "Name")().WillRepeatedly(Return("test"))
	ExpectCall(t.bucket, "BucketType")().WillRepeatedly(Return(t.bucketType))
	buf := make([]byte, end-start)

	objectData, err := t.rr.ReadAt(buf, int64(start))

	ExpectFalse(objectData.CacheHit)
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
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	// Mock for download job's NewReader call
	t.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd)
	ExpectCall(t.bucket, "Name")().WillRepeatedly(Return("test"))
	ExpectCall(t.bucket, "BucketType")().WillRepeatedly(Return(t.bucketType))
	start1 := 0
	end1 := util.MiB // not included
	AssertLt(end1, objectSize)
	// First call from offset 0 - sequential read
	buf := make([]byte, end1-start1)
	objectData, err := t.rr.ReadAt(buf, int64(start1))
	ExpectFalse(objectData.CacheHit)
	ExpectEq(nil, err)
	ExpectTrue(reflect.DeepEqual(testContent[start1:end1], buf))
	start2 := 16*util.MiB + 4
	end2 := start2 + util.MiB
	rd2 := &fake.FakeReader{ReadCloser: getReadCloser(testContent[start2:])}
	// Mock for random-reader's NewReader call
	t.mockNewReaderWithHandleCallForTestBucket(uint64(start2), objectSize, rd2)
	buf2 := make([]byte, end2-start2)
	// Assuming start2 offset download in progress
	// Assuming start2 offset download in progress
	objectData, err = t.rr.ReadAt(buf2, int64(start2))

	ExpectFalse(objectData.CacheHit)
	ExpectEq(nil, err)
	ExpectTrue(reflect.DeepEqual(testContent[start2:end2], buf2))
}

func (t *RandomReaderTest) Test_ReadAt_SequentialToRandomSubsequentReadOffsetLessThanPrevious() {
	t.rr.wrapped.fileCacheHandler = t.cacheHandler
	t.object.Size = 20 * util.MiB
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd)
	ExpectCall(t.bucket, "Name")().WillRepeatedly(Return("test"))
	ExpectCall(t.bucket, "BucketType")().WillRepeatedly(Return(t.bucketType))
	start1 := 0
	end1 := util.MiB // not included
	AssertLt(end1, objectSize)
	// First call from offset 0 - sequential read
	buf := make([]byte, end1-start1)
	objectData, err := t.rr.ReadAt(buf, int64(start1))
	ExpectFalse(objectData.CacheHit)
	ExpectEq(nil, err)
	ExpectTrue(reflect.DeepEqual(testContent[start1:end1], buf))
	start2 := 16*util.MiB + 4
	end2 := start2 + util.MiB
	rc2 := &fake.FakeReader{ReadCloser: getReadCloser(testContent[start2:])}
	t.mockNewReaderWithHandleCallForTestBucket(uint64(start2), objectSize, rc2)
	buf2 := make([]byte, end2-start2)
	// Assuming start2 offset download in progress
	objectData, err = t.rr.ReadAt(buf2, int64(start2))
	ExpectFalse(objectData.CacheHit)
	ExpectEq(nil, err)
	ExpectTrue(reflect.DeepEqual(testContent[start2:end2], buf2))
	start3 := util.MiB
	end3 := start3 + util.MiB
	buf3 := make([]byte, end3-start3)

	objectData, err = t.rr.ReadAt(buf3, int64(start3))

	ExpectEq(nil, err)
	ExpectTrue(objectData.CacheHit)
	ExpectTrue(reflect.DeepEqual(testContent[start3:end3], buf3))
}

func (t *RandomReaderTest) Test_ReadAt_CacheMissDueToInvalidJob() {
	t.rr.wrapped.fileCacheHandler = t.cacheHandler
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rc1 := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rc1)
	ExpectCall(t.bucket, "Name")().WillRepeatedly(Return("test"))
	ExpectCall(t.bucket, "BucketType")().WillRepeatedly(Return(t.bucketType))
	buf := make([]byte, objectSize)
	objectData, err := t.rr.ReadAt(buf, 0)
	AssertEq(nil, err)
	ExpectFalse(objectData.CacheHit)
	AssertTrue(reflect.DeepEqual(testContent, buf))
	job := t.jobManager.GetJob(t.object.Name, t.bucket.Name())
	if job != nil {
		jobStatus := job.GetStatus().Name
		AssertTrue(jobStatus == downloader.Downloading || jobStatus == downloader.Completed, fmt.Sprintf("the actual status is %v", jobStatus))
	}

	err = t.rr.wrapped.fileCacheHandler.InvalidateCache(t.object.Name, t.bucket.Name())
	AssertEq(nil, err)
	// Second reader (rc2) is required, since first reader (rc) is completely read.
	// Reading again will return EOF.
	rc2 := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rc2)

	objectData, err = t.rr.ReadAt(buf, 0)

	ExpectEq(nil, err)
	ExpectFalse(objectData.CacheHit)
	ExpectTrue(reflect.DeepEqual(testContent, buf))
	ExpectEq(nil, t.rr.wrapped.fileCacheHandle)
}

func (t *RandomReaderTest) Test_ReadAt_CachePopulatedAndThenCacheMissDueToInvalidJob() {
	t.rr.wrapped.fileCacheHandler = t.cacheHandler
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rd1 := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd1)
	ExpectCall(t.bucket, "Name")().WillRepeatedly(Return("test"))
	ExpectCall(t.bucket, "BucketType")().WillRepeatedly(Return(t.bucketType))
	buf := make([]byte, objectSize)
	objectData, err := t.rr.ReadAt(buf, 0)
	AssertEq(nil, err)
	AssertFalse(objectData.CacheHit)
	AssertTrue(reflect.DeepEqual(testContent, buf))
	job := t.jobManager.GetJob(t.object.Name, t.bucket.Name())
	if job != nil {
		jobStatus := job.GetStatus().Name
		AssertTrue(jobStatus == downloader.Downloading || jobStatus == downloader.Completed, fmt.Sprintf("the actual status is %v", jobStatus))
	}
	err = t.rr.wrapped.fileCacheHandler.InvalidateCache(t.object.Name, t.bucket.Name())
	AssertEq(nil, err)
	// Second reader (rc2) is required, since first reader (rc) is completely read.
	// Reading again will return EOF.
	rc2 := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rc2)
	objectData, err = t.rr.ReadAt(buf, 0)
	ExpectEq(nil, err)
	ExpectFalse(objectData.CacheHit)
	ExpectTrue(reflect.DeepEqual(testContent, buf))
	ExpectEq(nil, t.rr.wrapped.fileCacheHandle)
	rd3 := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd3)
	// This call will populate the cache again.
	objectData, err = t.rr.ReadAt(buf, 0)

	ExpectEq(nil, err)
	ExpectFalse(objectData.CacheHit)
	ExpectTrue(reflect.DeepEqual(testContent, buf))
	ExpectNe(nil, t.rr.wrapped.fileCacheHandle)
}

func (t *RandomReaderTest) Test_ReadAt_CachePopulatedAndThenCacheMissDueToInvalidFileHandle() {
	t.rr.wrapped.fileCacheHandler = t.cacheHandler
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd)
	ExpectCall(t.bucket, "Name")().WillRepeatedly(Return("test"))
	ExpectCall(t.bucket, "BucketType")().WillRepeatedly(Return(t.bucketType))
	buf := make([]byte, objectSize)
	objectData, err := t.rr.ReadAt(buf, 0)
	AssertEq(nil, err)
	AssertFalse(objectData.CacheHit)
	AssertTrue(reflect.DeepEqual(testContent, buf))
	AssertNe(nil, t.rr.wrapped.fileCacheHandle)
	err = t.rr.wrapped.fileCacheHandle.Close()
	AssertEq(nil, err)
	// Second reader (rc2) is required, since first reader (rc) is completely read.
	// Reading again will return EOF.
	rc2 := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rc2)
	objectData, err = t.rr.ReadAt(buf, 0)
	ExpectEq(nil, err)
	ExpectFalse(objectData.CacheHit)
	ExpectTrue(reflect.DeepEqual(testContent, buf))
	ExpectEq(nil, t.rr.wrapped.fileCacheHandle)
	rc3 := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rc3)
	// Reading from file cache with new file cache handle.
	objectData, err = t.rr.ReadAt(buf, 0)

	ExpectEq(nil, err)
	ExpectTrue(objectData.CacheHit)
	ExpectTrue(reflect.DeepEqual(testContent, buf))
	ExpectNe(nil, t.rr.wrapped.fileCacheHandle)
}

func (t *RandomReaderTest) Test_ReadAt_IfCacheFileGetsDeleted() {
	t.rr.wrapped.fileCacheHandler = t.cacheHandler
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd)
	ExpectCall(t.bucket, "Name")().WillRepeatedly(Return("test"))
	ExpectCall(t.bucket, "BucketType")().WillOnce(Return(t.bucketType))
	buf := make([]byte, objectSize)
	objectData, err := t.rr.ReadAt(buf, 0)
	AssertEq(nil, err)
	AssertFalse(objectData.CacheHit)
	AssertTrue(reflect.DeepEqual(testContent, buf))
	AssertNe(nil, t.rr.wrapped.fileCacheHandle)
	err = t.rr.wrapped.fileCacheHandle.Close()
	AssertEq(nil, err)
	t.rr.wrapped.fileCacheHandle = nil
	// Delete the local cache file.
	filePath := util.GetDownloadPath(t.cacheDir, util.GetObjectPath(t.bucket.Name(), t.object.Name))
	err = os.Remove(filePath)
	AssertEq(nil, err)
	// Second reader (rd2) is required, since first reader (rd) is completely read.
	// Reading again will return EOF.
	rd2 := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd2)

	_, err = t.rr.ReadAt(buf, 0)

	AssertNe(nil, err)
	AssertTrue(errors.Is(err, util.ErrFileNotPresentInCache))
}

func (t *RandomReaderTest) Test_ReadAt_IfCacheFileGetsDeletedWithCacheHandleOpen() {
	t.rr.wrapped.fileCacheHandler = t.cacheHandler
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd)
	ExpectCall(t.bucket, "Name")().WillRepeatedly(Return("test"))
	ExpectCall(t.bucket, "BucketType")().WillRepeatedly(Return(t.bucketType))
	buf := make([]byte, objectSize)
	objectData, err := t.rr.ReadAt(buf, 0)
	AssertEq(nil, err)
	AssertFalse(objectData.CacheHit)
	AssertTrue(reflect.DeepEqual(testContent, buf))
	AssertNe(nil, t.rr.wrapped.fileCacheHandle)
	// Delete the local cache file.
	filePath := util.GetDownloadPath(t.cacheDir, util.GetObjectPath(t.bucket.Name(), t.object.Name))
	err = os.Remove(filePath)
	AssertEq(nil, err)

	// Read via cache only, as we have old fileHandle open and linux
	// doesn't delete the file until the fileHandle count for the file is zero.
	objectData, err = t.rr.ReadAt(buf, 0)

	AssertEq(nil, err)
	ExpectTrue(objectData.CacheHit)
	ExpectTrue(reflect.DeepEqual(testContent, buf))
	ExpectNe(nil, t.rr.wrapped.fileCacheHandle)
}

func (t *RandomReaderTest) Test_ReadAt_FailedJobRestartAndCacheHit() {
	t.rr.wrapped.fileCacheHandler = t.cacheHandler
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rc := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	// First call goes to file cache succeeded by next call to random reader.
	// First NewReaderWithReadHandle-call throws error, hence async job fails.
	// Later NewReader-call returns a valid readCloser object hence fallback to
	// GCS read will succeed.
	ExpectCall(t.bucket, "NewReaderWithReadHandle")(Any(), Any()).
		WillOnce(Return(nil, errors.New(""))).WillRepeatedly(Return(rc, nil))
	ExpectCall(t.bucket, "Name")().WillRepeatedly(Return("test"))
	ExpectCall(t.bucket, "BucketType")().WillRepeatedly(Return(t.bucketType))
	buf := make([]byte, objectSize)
	objectData, err := t.rr.ReadAt(buf, 0)
	AssertEq(nil, err)
	ExpectFalse(objectData.CacheHit)
	AssertTrue(reflect.DeepEqual(testContent, buf))
	job := t.jobManager.GetJob(t.object.Name, t.bucket.Name())
	AssertTrue(job == nil || job.GetStatus().Name == downloader.Failed)
	// Second reader (rc2) is required, since first reader (rc) is completely read.
	// Reading again will return EOF.
	rd2 := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd2)
	// This call will populate the cache again.
	objectData, err = t.rr.ReadAt(buf, 0)
	ExpectEq(nil, err)
	ExpectFalse(objectData.CacheHit)
	ExpectTrue(reflect.DeepEqual(testContent, buf))
	ExpectNe(nil, t.rr.wrapped.fileCacheHandle)

	objectData, err = t.rr.ReadAt(buf, 0)

	ExpectEq(nil, err)
	ExpectTrue(objectData.CacheHit)
	ExpectTrue(reflect.DeepEqual(testContent, buf))
	ExpectNe(nil, t.rr.wrapped.fileCacheHandle)
}

// Only writing two unit tests for tryReadingFromFileCache as
// unit tests of ReadAt method covers all the workflows.
func (t *RandomReaderTest) Test_tryReadingFromFileCache_CacheHit() {
	t.rr.wrapped.fileCacheHandler = t.cacheHandler
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd)
	ExpectCall(t.bucket, "Name")().WillRepeatedly(Return("test"))
	ExpectCall(t.bucket, "BucketType")().WillRepeatedly(Return(t.bucketType))
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
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd)
	ExpectCall(t.bucket, "Name")().WillRepeatedly(Return("test"))
	ExpectCall(t.bucket, "BucketType")().WillOnce(Return(t.bucketType))
	start1 := 0
	end1 := util.MiB // equal to objectSize
	// First call from offset 0 - objectSize
	buf := make([]byte, end1-start1)
	objectData, err := t.rr.ReadAt(buf, int64(start1))
	ExpectFalse(objectData.CacheHit)
	ExpectEq(nil, err)
	ExpectTrue(reflect.DeepEqual(testContent[start1:end1], buf))
	start2 := util.MiB // offset equal to objectSize
	end2 := start2 + util.MiB
	buf2 := make([]byte, end2-start2)

	// read for offset equal to objectSize
	objectData, err = t.rr.ReadAt(buf2, int64(start2))

	// nothing should be read
	ExpectFalse(objectData.CacheHit)
	ExpectEq(io.EOF, err)
	ExpectEq(0, objectData.Size)
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
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd)
	ExpectCall(t.bucket, "Name")().WillRepeatedly(Return("test"))
	ExpectCall(t.bucket, "BucketType")().WillOnce(Return(t.bucketType))
	buf := make([]byte, objectSize)
	_, cacheHit, err := t.rr.wrapped.tryReadingFromFileCache(t.rr.ctx, buf, 0)
	AssertFalse(cacheHit)
	AssertEq(nil, err)
	AssertTrue(reflect.DeepEqual(testContent, buf))
	AssertNe(nil, t.rr.wrapped.fileCacheHandle)

	t.rr.wrapped.Destroy()

	ExpectEq(nil, t.rr.wrapped.fileCacheHandle)
}

func (t *RandomReaderTest) TestNewReader_FileClobbered() {
	var notFoundError *gcs.NotFoundError

	ExpectCall(t.bucket, "NewReaderWithReadHandle")(Any(), Any()).
		WillOnce(Return(nil, notFoundError))

	err := t.rr.wrapped.startRead(0, 1)

	AssertNe(nil, err)
	var clobberedErr *gcsfuse_errors.FileClobberedError
	AssertTrue(errors.As(err, &clobberedErr))
}

// TODO (raj-prince) - to add unit tests for failed scenario while reading via cache.
// This requires mocking CacheHandle object, whose read method will return some unexpected
// error.
