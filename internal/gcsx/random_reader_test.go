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
	"context"
	"errors"
	"fmt"
	"io"
	"os"
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
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util/diskutil"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/googlecloudplatform/gcsfuse/v3/tracing"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

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
// Helper Struct and Constructor
////////////////////////////////////////////////////////////////////////

type randomReaderTestHelper struct {
	t            *testing.T
	assert       *assert.Assertions
	require      *require.Assertions
	object       *gcs.MinObject
	bucket       *storage.TestifyMockBucket
	rr           checkingRandomReader
	cacheDir     string
	jobManager   *downloader.JobManager
	cacheHandler *file.CacheHandler
	bucketType   gcs.BucketType
}

func newRandomReaderTestHelper(t *testing.T, bucketType gcs.BucketType) *randomReaderTestHelper {
	h := &randomReaderTestHelper{
		t:          t,
		assert:     assert.New(t),
		require:    require.New(t),
		bucketType: bucketType,
	}

	readOp := fuseops.ReadFileOp{Handle: 1}
	h.rr.ctx = context.WithValue(context.Background(), ReadOp, &readOp)

	// Manufacture an object record.
	h.object = &gcs.MinObject{
		Name:       "foo",
		Size:       17,
		Generation: 1234,
	}

	// Create the bucket.
	h.bucket = &storage.TestifyMockBucket{}

	h.cacheDir = t.TempDir()
	lruCache := lru.NewCache(cacheMaxSize)
	fileCacheConfig := &cfg.FileCacheConfig{
		EnableCrc: false,
	}
	cacheDirVolumeBlockSize := diskutil.GetVolumeBlockSize(h.cacheDir)
	h.jobManager = downloader.NewJobManager(lruCache, util.DefaultFilePerm, util.DefaultDirPerm, h.cacheDir, sequentialReadSizeInMb, fileCacheConfig, metrics.NewNoopMetrics(), tracing.NewNoopTracer(), cacheDirVolumeBlockSize)
	h.cacheHandler = file.NewCacheHandler(lruCache, h.jobManager, h.cacheDir, util.DefaultFilePerm, util.DefaultDirPerm, "", "", false, cacheDirVolumeBlockSize)

	// Set up the reader.
	rr := NewRandomReader(h.object, h.bucket, sequentialReadSizeInMb, nil, false, metrics.NewNoopMetrics(), tracing.NewNoopTracer(), nil, nil, 0)
	h.rr.wrapped = rr.(*randomReader)

	return h
}

func (h *randomReaderTestHelper) tearDown() {
	h.rr.Destroy()
	h.bucket.AssertExpectations(h.t)
}

func (h *randomReaderTestHelper) mockNewReaderWithHandleCallForTestBucket(start uint64, limit uint64, rd gcs.StorageReader) {
	h.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(req *gcs.ReadObjectRequest) bool {
		if req.Range == nil {
			return false
		}
		return req.Range.Start == start && req.Range.Limit == limit
	})).Return(rd, nil).Once()
}

func runTest(t *testing.T, testFunc func(h *randomReaderTestHelper)) {
	h := newRandomReaderTestHelper(t, gcs.BucketType{})
	defer h.tearDown()
	testFunc(h)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func TestRandomReader_EmptyRead(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		buf := make([]byte, 0)

		objectData, err := h.rr.ReadAt(buf, 0)

		h.assert.Equal(0, objectData.Size)
		h.assert.NoError(err)
	})
}

func TestRandomReader_ReadAtEndOfObject(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		buf := make([]byte, 1)

		objectData, err := h.rr.ReadAt(buf, int64(h.object.Size))

		h.assert.Equal(0, objectData.Size)
		h.assert.Equal(io.EOF, err)
	})
}

func TestRandomReader_ReadPastEndOfObject(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		buf := make([]byte, 1)

		objectData, err := h.rr.ReadAt(buf, int64(h.object.Size)+1)

		h.assert.False(objectData.CacheHit)
		h.assert.Equal(0, objectData.Size)
		h.assert.Equal(io.EOF, err)
	})
}

func TestRandomReader_NoExistingReader(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		h.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).
			Return(nil, errors.New("")).Once()
		h.bucket.On("BucketType").Return(h.bucketType).Times(2)
		buf := make([]byte, 1)

		_, err := h.rr.ReadAt(buf, 0)

		h.require.Error(err)
	})
}

func TestRandomReader_ExistingReader_ReadAtOffsetAfterTheReaderPosition(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		var currentStartOffset int64 = 2
		var readerLimit int64 = 15
		var readAtOffset int64 = 10
		var readSize int64 = 1
		var expectedStartOffsetAfterRead = readAtOffset + readSize

		nopCloser := io.NopCloser(strings.NewReader(strings.Repeat("x", int(readerLimit))))
		rc := &fake.FakeReader{ReadCloser: nopCloser}
		h.rr.wrapped.reader = rc
		h.rr.wrapped.cancel = func() {}
		h.rr.wrapped.start = currentStartOffset
		h.rr.wrapped.limit = readerLimit

		h.bucket.On("BucketType").Return(h.bucketType).Times(2)

		buf := make([]byte, readSize)
		_, err := h.rr.ReadAt(buf, readAtOffset)

		h.require.NoError(err)
		h.assert.Equal(rc, h.rr.wrapped.reader)
		h.assert.Equal(expectedStartOffsetAfterRead, h.rr.wrapped.start)
		h.assert.Equal(readerLimit, h.rr.wrapped.limit)
	})
}

func TestRandomReader_NewReaderReturnsError(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		h.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).
			Return(nil, errors.New("taco")).Once()
		h.bucket.On("BucketType").Return(h.bucketType).Times(2)
		buf := make([]byte, 1)

		_, err := h.rr.ReadAt(buf, 0)

		h.assert.Error(err)
		h.assert.Contains(err.Error(), "NewReaderWithReadHandle")
		h.assert.Contains(err.Error(), "taco")
	})
}

func TestRandomReader_ReaderFails(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		r := iotest.OneByteReader(iotest.TimeoutReader(strings.NewReader("xxx")))
		rc := &fake.FakeReader{ReadCloser: io.NopCloser(r)}

		h.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).
			Return(rc, nil).Once()
		h.bucket.On("BucketType").Return(h.bucketType).Times(2)

		buf := make([]byte, 3)
		_, err := h.rr.ReadAt(buf, 0)

		h.assert.Error(err)
		h.assert.Contains(err.Error(), "readFull")
		h.assert.Contains(err.Error(), iotest.ErrTimeout.Error())
	})
}

func TestRandomReader_ReaderNotExhausted(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		cc := &countingCloser{
			Reader: strings.NewReader("abc"),
		}
		rc := &fake.FakeReader{ReadCloser: cc}

		h.bucket.On("BucketType").Return(h.bucketType).Times(2)

		h.rr.wrapped.reader = rc
		h.rr.wrapped.cancel = func() {}
		h.rr.wrapped.start = 1
		h.rr.wrapped.limit = 4

		buf := make([]byte, 2)
		objectData, err := h.rr.ReadAt(buf, 1)

		h.assert.False(objectData.CacheHit)
		h.assert.Equal(2, objectData.Size)
		h.assert.NoError(err)
		h.assert.Equal("ab", string(buf[:objectData.Size]))

		h.assert.Equal(0, cc.closeCount)
		h.assert.Equal(rc, h.rr.wrapped.reader)
		h.assert.Equal(int64(3), h.rr.wrapped.start)
		h.assert.Equal(int64(4), h.rr.wrapped.limit)
	})
}

func TestRandomReader_ReaderExhausted_ReadFinished(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		rc := &countingCloser{
			Reader: strings.NewReader("abc"),
		}

		h.bucket.On("BucketType").Return(h.bucketType).Times(2)

		h.rr.wrapped.reader = &fake.FakeReader{ReadCloser: rc}
		h.rr.wrapped.cancel = func() {}
		h.rr.wrapped.start = 1
		h.rr.wrapped.limit = 4

		buf := make([]byte, 3)
		objectData, err := h.rr.ReadAt(buf, 1)

		h.assert.False(objectData.CacheHit)
		h.assert.Equal(3, objectData.Size)
		h.assert.NoError(err)
		h.assert.Equal("abc", string(buf[:objectData.Size]))

		h.assert.Equal(1, rc.closeCount)
		h.assert.Nil(h.rr.wrapped.reader)
		h.assert.Nil(h.rr.wrapped.cancel)
		h.assert.Equal(int64(4), h.rr.wrapped.limit)
	})
}

func TestRandomReader_PropagatesCancellation(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		finishRead := make(chan struct{})
		rc := io.NopCloser(&blockingReader{finishRead})

		h.bucket.On("BucketType").Return(h.bucketType).Times(2)

		h.rr.wrapped.reader = &fake.FakeReader{ReadCloser: rc}
		h.rr.wrapped.start = 1
		h.rr.wrapped.limit = 4
		h.rr.wrapped.config = &cfg.Config{FileSystem: cfg.FileSystemConfig{IgnoreInterrupts: false}}

		cancelCalled := make(chan struct{})
		h.rr.wrapped.cancel = func() { close(cancelCalled) }

		readReturned := make(chan struct{})
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			buf := make([]byte, 2)
			_, _ = h.rr.wrapped.ReadAt(ctx, buf, 1)
			close(readReturned)
		}()

		select {
		case <-time.After(10 * time.Millisecond):
		case <-readReturned:
			h.t.Fatal("Read returned early.")
		}

		cancel()
		<-cancelCalled

		close(finishRead)
		<-readReturned
	})
}

func TestRandomReader_DoesntPropagateCancellationAfterReturning(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		h.rr.wrapped.reader = &fake.FakeReader{ReadCloser: getReadCloser([]byte("xxx"))}
		h.rr.wrapped.start = 1
		h.rr.wrapped.limit = 4

		h.bucket.On("BucketType").Return(h.bucketType).Times(2)

		cancelCalled := make(chan struct{})
		h.rr.wrapped.cancel = func() { close(cancelCalled) }

		ctx, cancel := context.WithCancel(context.Background())
		buf := make([]byte, 2)
		objectData, err := h.rr.wrapped.ReadAt(ctx, buf, 1)

		h.require.NoError(err)
		h.assert.False(objectData.CacheHit)
		h.require.Equal(2, objectData.Size)

		cancel()
		select {
		case <-time.After(10 * time.Millisecond):
		case <-cancelCalled:
			h.t.Fatal("Read context unexpectedly cancelled.")
		}
	})
}

func TestRandomReader_UpgradesReadsToObjectSize(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		const objectSize = 2 * MiB
		h.object.Size = objectSize

		const readSize = 10
		h.require.Less(readSize, objectSize)

		h.rr.wrapped.reader = &fake.FakeReader{ReadCloser: getReadCloser([]byte("xxx"))}
		h.rr.wrapped.cancel = func() {}
		h.rr.wrapped.start = 2
		h.rr.wrapped.limit = 5

		r := strings.NewReader(strings.Repeat("x", objectSize))
		rc := &fake.FakeReader{ReadCloser: io.NopCloser(r)}

		h.mockNewReaderWithHandleCallForTestBucket(1, objectSize, rc)
		h.bucket.On("BucketType").Return(h.bucketType).Times(2)

		buf := make([]byte, readSize)
		objectData, err := h.rr.ReadAt(buf, 1)

		h.assert.False(objectData.CacheHit)
		h.assert.NoError(err)
		h.assert.Equal(int64(1+readSize), h.rr.wrapped.start)
		h.assert.Equal(int64(objectSize), h.rr.wrapped.limit)
	})
}

func TestRandomReader_UpgradeReadsToAverageSize(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		h.object.Size = 1 << 40
		const totalReadBytes = 6 * MiB
		const numReads = 2
		const avgReadBytes = totalReadBytes / numReads

		const expectedBytesToRead = avgReadBytes
		const start = 1
		const readSize = 2 * minReadSize

		h.rr.wrapped.seeks.Store(numReads)
		h.rr.wrapped.totalReadBytes.Store(totalReadBytes)
		h.rr.wrapped.reader = &fake.FakeReader{ReadCloser: getReadCloser([]byte("xxx"))}
		h.rr.wrapped.cancel = func() {}
		h.rr.wrapped.start = 2
		h.rr.wrapped.limit = 5
		h.rr.wrapped.expectedOffset.Store(2)

		r := strings.NewReader(strings.Repeat("x", expectedBytesToRead))
		rc := &fake.FakeReader{ReadCloser: io.NopCloser(r)}

		h.mockNewReaderWithHandleCallForTestBucket(start, start+expectedBytesToRead, rc)
		h.bucket.On("BucketType").Return(h.bucketType).Times(2)

		buf := make([]byte, readSize)
		objectData, err := h.rr.ReadAt(buf, start)

		h.assert.False(objectData.CacheHit)
		h.require.NoError(err)
		h.assert.Equal(int64(start+expectedBytesToRead), h.rr.wrapped.limit)
	})
}

func TestRandomReader_UpgradesSequentialReads_ExistingReader(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		h.object.Size = 1 << 40
		const readSize = 10

		const existingSize = 3
		r := strings.NewReader(strings.Repeat("x", existingSize))

		h.rr.wrapped.reader = &fake.FakeReader{ReadCloser: io.NopCloser(r)}
		h.rr.wrapped.cancel = func() {}
		h.rr.wrapped.start = 1
		h.rr.wrapped.limit = 1 + existingSize

		r = strings.NewReader(strings.Repeat("y", readSize))
		rc := &fake.FakeReader{ReadCloser: io.NopCloser(r)}

		h.mockNewReaderWithHandleCallForTestBucket(1, 1+sequentialReadSizeInBytes, rc)
		h.bucket.On("BucketType").Return(h.bucketType).Times(2)

		buf := make([]byte, readSize)
		objectData, err := h.rr.ReadAt(buf, 1)

		h.assert.False(objectData.CacheHit)
		h.require.NoError(err)
		h.assert.Equal(int64(1+readSize), h.rr.wrapped.start)
		h.assert.Equal(int64(1+sequentialReadSizeInBytes), h.rr.wrapped.limit)
	})
}

func TestRandomReader_UpgradesSequentialReads_NoExistingReader(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		h.object.Size = 1 << 40
		const readSize = 1 * MiB
		rr := NewRandomReader(h.object, h.bucket, readSize/MiB, nil, false, metrics.NewNoopMetrics(), tracing.NewNoopTracer(), nil, nil, 0)
		h.rr.wrapped = rr.(*randomReader)

		h.rr.wrapped.start = 1
		h.rr.wrapped.limit = 1

		data := strings.Repeat("x", readSize)
		r := strings.NewReader(data)
		rc := &fake.FakeReader{ReadCloser: io.NopCloser(r)}

		h.mockNewReaderWithHandleCallForTestBucket(1, 1+readSize, rc)
		h.bucket.On("BucketType").Return(h.bucketType).Times(2)

		buf := make([]byte, readSize)
		objectData, err := h.rr.ReadAt(buf, 1)

		h.assert.False(objectData.CacheHit)
		h.require.NoError(err)
		h.assert.Equal(data, string(buf))
		h.assert.Equal(int64(1+readSize), h.rr.wrapped.start)
		h.assert.Equal(int64(1+readSize), h.rr.wrapped.limit)
	})
}

/******************* File cache specific tests ***********************/

func TestRandomReader_Test_ReadAt_SequentialFullObject(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		h.rr.wrapped.fileCacheHandler = h.cacheHandler
		objectSize := h.object.Size
		testContent := testutil.GenerateRandomBytes(int(objectSize))
		rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
		h.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd)
		h.bucket.On("Name").Return("test")
		h.bucket.On("BucketType").Return(h.bucketType)
		buf := make([]byte, objectSize)
		objectData, err := h.rr.ReadAt(buf, 0)
		h.assert.False(objectData.CacheHit)
		h.assert.NoError(err)
		h.assert.Equal(testContent, buf)

		objectData, err = h.rr.ReadAt(buf, 0)

		h.assert.True(objectData.CacheHit)
		h.assert.NoError(err)
		h.assert.Equal(testContent, buf)
	})
}

func TestRandomReader_Test_ReadAt_SequentialRangeRead(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		h.rr.wrapped.fileCacheHandler = h.cacheHandler
		objectSize := h.object.Size
		testContent := testutil.GenerateRandomBytes(int(objectSize))
		rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
		h.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd)
		h.bucket.On("Name").Return("test")
		h.bucket.On("BucketType").Return(h.bucketType).Once()
		start := 0
		end := 10
		h.require.Less(end, int(objectSize))
		buf := make([]byte, end-start)

		objectData, err := h.rr.ReadAt(buf, int64(start))

		h.assert.False(objectData.CacheHit)
		h.assert.NoError(err)
		h.assert.Equal(testContent[start:end], buf)
	})
}

func TestRandomReader_Test_ReadAt_SequentialSubsequentReadOffsetLessThanReadChunkSize(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		h.rr.wrapped.fileCacheHandler = h.cacheHandler
		h.object.Size = 20 * util.MiB
		objectSize := h.object.Size
		testContent := testutil.GenerateRandomBytes(int(objectSize))
		rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
		h.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd)
		h.bucket.On("Name").Return("test")
		h.bucket.On("BucketType").Return(h.bucketType)
		start1 := 0
		end1 := util.MiB
		h.require.Less(end1, int(objectSize))
		buf := make([]byte, end1-start1)
		objectData, err := h.rr.ReadAt(buf, int64(start1))
		h.assert.False(objectData.CacheHit)
		h.assert.NoError(err)
		h.assert.Equal(testContent[start1:end1], buf)
		start2 := 3*util.MiB + 4
		end2 := start2 + util.MiB
		buf2 := make([]byte, end2-start2)

		objectData, err = h.rr.ReadAt(buf2, int64(start2))

		h.assert.True(objectData.CacheHit)
		h.assert.NoError(err)
		h.assert.Equal(testContent[start2:end2], buf2)
	})
}

func TestRandomReader_Test_ReadAt_RandomReadNotStartWithZeroOffsetWhenCacheForRangeReadIsFalse(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		h.rr.wrapped.fileCacheHandler = h.cacheHandler
		objectSize := h.object.Size
		h.rr.wrapped.cacheFileForRangeRead = false
		testContent := testutil.GenerateRandomBytes(int(objectSize))
		start := 5
		end := 10
		rc := &fake.FakeReader{ReadCloser: getReadCloser(testContent[start:])}
		h.mockNewReaderWithHandleCallForTestBucket(uint64(start), objectSize, rc)
		rc2 := &fake.FakeReader{ReadCloser: getReadCloser(testContent[start:])}
		h.mockNewReaderWithHandleCallForTestBucket(uint64(start), objectSize, rc2)
		h.bucket.On("Name").Return("test")
		h.bucket.On("BucketType").Return(h.bucketType)
		buf := make([]byte, end-start)
		objectData, err := h.rr.ReadAt(buf, int64(start))
		h.assert.False(objectData.CacheHit)
		h.assert.NoError(err)
		h.assert.Equal(testContent[start:end], buf)
		job, err := h.jobManager.CreateJobIfNotExists(h.object, h.bucket)
		h.require.NoError(err)
		jobStatus := job.GetStatus()
		h.assert.True(jobStatus.Name == downloader.NotStarted)

		objectData, err = h.rr.ReadAt(buf, int64(start))

		h.assert.NoError(err)
		h.assert.False(objectData.CacheHit)
	})
}

func TestRandomReader_Test_ReadAt_RandomReadNotStartWithZeroOffsetWhenCacheForRangeReadIsTrue(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		h.rr.wrapped.fileCacheHandler = h.cacheHandler
		objectSize := h.object.Size
		h.rr.wrapped.cacheFileForRangeRead = true
		testContent := testutil.GenerateRandomBytes(int(objectSize))
		start := 5
		end := 10
		rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent[start:])}
		h.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(req *gcs.ReadObjectRequest) bool {
			if req.Range == nil {
				return false
			}
			return req.Range.Start == uint64(start) && req.Range.Limit == objectSize
		})).Return(rd, nil).Once()
		rd1 := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
		h.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd1)
		h.bucket.On("Name").Return("test")
		h.bucket.On("BucketType").Return(h.bucketType)
		buf := make([]byte, end-start)

		objectData, err := h.rr.ReadAt(buf, int64(start))

		h.assert.False(objectData.CacheHit)
		h.assert.NoError(err)
		h.assert.Equal(testContent[start:end], buf)
		// Second read call should be a cache hit
		// Wait for the download job to complete first to avoid race.
		job := h.jobManager.GetJob(h.object.Name, h.bucket.Name())
		h.require.NotNil(job)
		h.require.Eventually(func() bool {
			status := job.GetStatus()
			return status.Name == downloader.Completed || status.Name == downloader.Failed
		}, 2*time.Second, 10*time.Millisecond)

		objectData, err = h.rr.ReadAt(buf, int64(start))

		h.assert.NoError(err)
		h.assert.True(objectData.CacheHit)
	})
}

func TestRandomReader_Test_ReadAt_SequentialToRandomSubsequentReadOffsetMoreThanReadChunkSize(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		h.rr.wrapped.fileCacheHandler = h.cacheHandler
		h.object.Size = 20 * util.MiB
		objectSize := h.object.Size
		testContent := testutil.GenerateRandomBytes(int(objectSize))
		rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
		h.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd)
		h.bucket.On("Name").Return("test")
		h.bucket.On("BucketType").Return(h.bucketType)
		start1 := 0
		end1 := util.MiB
		h.require.Less(end1, int(objectSize))
		buf := make([]byte, end1-start1)
		objectData, err := h.rr.ReadAt(buf, int64(start1))
		h.assert.False(objectData.CacheHit)
		h.assert.NoError(err)
		h.assert.Equal(testContent[start1:end1], buf)
		start2 := 16*util.MiB + 4
		end2 := start2 + util.MiB
		rd2 := &fake.FakeReader{ReadCloser: getReadCloser(testContent[start2:])}
		h.mockNewReaderWithHandleCallForTestBucket(uint64(start2), objectSize, rd2)
		buf2 := make([]byte, end2-start2)

		objectData, err = h.rr.ReadAt(buf2, int64(start2))

		h.assert.False(objectData.CacheHit)
		h.assert.NoError(err)
		h.assert.Equal(testContent[start2:end2], buf2)
	})
}

func TestRandomReader_Test_ReadAt_SequentialToRandomSubsequentReadOffsetLessThanPrevious(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		h.rr.wrapped.fileCacheHandler = h.cacheHandler
		h.object.Size = 20 * util.MiB
		objectSize := h.object.Size
		testContent := testutil.GenerateRandomBytes(int(objectSize))
		rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
		h.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd)
		h.bucket.On("Name").Return("test")
		h.bucket.On("BucketType").Return(h.bucketType)
		start1 := 0
		end1 := util.MiB
		h.require.Less(end1, int(objectSize))
		buf := make([]byte, end1-start1)
		objectData, err := h.rr.ReadAt(buf, int64(start1))
		h.assert.False(objectData.CacheHit)
		h.assert.NoError(err)
		h.assert.Equal(testContent[start1:end1], buf)
		start2 := 16*util.MiB + 4
		end2 := start2 + util.MiB
		rc2 := &fake.FakeReader{ReadCloser: getReadCloser(testContent[start2:])}
		h.mockNewReaderWithHandleCallForTestBucket(uint64(start2), objectSize, rc2)
		buf2 := make([]byte, end2-start2)
		objectData, err = h.rr.ReadAt(buf2, int64(start2))
		h.assert.False(objectData.CacheHit)
		h.assert.NoError(err)
		h.assert.Equal(testContent[start2:end2], buf2)
		start3 := util.MiB
		end3 := start3 + util.MiB
		buf3 := make([]byte, end3-start3)

		objectData, err = h.rr.ReadAt(buf3, int64(start3))

		h.assert.NoError(err)
		h.assert.True(objectData.CacheHit)
		h.assert.Equal(testContent[start3:end3], buf3)
	})
}

func TestRandomReader_Test_ReadAt_CacheMissDueToInvalidJob(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		h.rr.wrapped.fileCacheHandler = h.cacheHandler
		objectSize := h.object.Size
		testContent := testutil.GenerateRandomBytes(int(objectSize))
		rc1 := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
		h.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rc1)
		h.bucket.On("Name").Return("test")
		h.bucket.On("BucketType").Return(h.bucketType)
		buf := make([]byte, objectSize)
		objectData, err := h.rr.ReadAt(buf, 0)
		h.require.NoError(err)
		h.assert.False(objectData.CacheHit)
		h.require.Equal(testContent, buf)
		job := h.jobManager.GetJob(h.object.Name, h.bucket.Name())
		if job != nil {
			jobStatus := job.GetStatus().Name
			h.require.True(jobStatus == downloader.Downloading || jobStatus == downloader.Completed, fmt.Sprintf("the actual status is %v", jobStatus))
		}

		err = h.rr.wrapped.fileCacheHandler.InvalidateCache(h.object.Name, h.bucket.Name())
		h.require.NoError(err)
		rc2 := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
		h.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rc2)

		objectData, err = h.rr.ReadAt(buf, 0)

		h.assert.NoError(err)
		h.assert.False(objectData.CacheHit)
		h.assert.Equal(testContent, buf)
		h.assert.Nil(h.rr.wrapped.fileCacheHandle)
	})
}

func TestRandomReader_Test_ReadAt_CachePopulatedAndThenCacheMissDueToInvalidJob(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		h.rr.wrapped.fileCacheHandler = h.cacheHandler
		objectSize := h.object.Size
		testContent := testutil.GenerateRandomBytes(int(objectSize))
		rd1 := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
		h.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd1)
		h.bucket.On("Name").Return("test")
		h.bucket.On("BucketType").Return(h.bucketType)
		buf := make([]byte, objectSize)
		objectData, err := h.rr.ReadAt(buf, 0)
		h.require.NoError(err)
		h.assert.False(objectData.CacheHit)
		h.require.Equal(testContent, buf)
		job := h.jobManager.GetJob(h.object.Name, h.bucket.Name())
		if job != nil {
			jobStatus := job.GetStatus().Name
			h.require.True(jobStatus == downloader.Downloading || jobStatus == downloader.Completed, fmt.Sprintf("the actual status is %v", jobStatus))
		}
		err = h.rr.wrapped.fileCacheHandler.InvalidateCache(h.object.Name, h.bucket.Name())
		h.require.NoError(err)
		rc2 := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
		h.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rc2)
		objectData, err = h.rr.ReadAt(buf, 0)
		h.assert.NoError(err)
		h.assert.False(objectData.CacheHit)
		h.assert.Equal(testContent, buf)
		h.assert.Nil(h.rr.wrapped.fileCacheHandle)
		rd3 := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
		h.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd3)

		objectData, err = h.rr.ReadAt(buf, 0)

		h.assert.NoError(err)
		h.assert.False(objectData.CacheHit)
		h.assert.Equal(testContent, buf)
		h.assert.NotNil(h.rr.wrapped.fileCacheHandle)
	})
}

func TestRandomReader_Test_ReadAt_CachePopulatedAndThenCacheMissDueToInvalidFileHandle(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		h.rr.wrapped.fileCacheHandler = h.cacheHandler
		objectSize := h.object.Size
		testContent := testutil.GenerateRandomBytes(int(objectSize))
		rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
		h.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd)
		h.bucket.On("Name").Return("test")
		h.bucket.On("BucketType").Return(h.bucketType)
		buf := make([]byte, objectSize)
		objectData, err := h.rr.ReadAt(buf, 0)
		h.require.NoError(err)
		h.assert.False(objectData.CacheHit)
		h.require.Equal(testContent, buf)
		h.require.NotNil(h.rr.wrapped.fileCacheHandle)
		err = h.rr.wrapped.fileCacheHandle.Close()
		h.require.NoError(err)
		rc2 := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
		h.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rc2)
		objectData, err = h.rr.ReadAt(buf, 0)
		h.assert.NoError(err)
		h.assert.False(objectData.CacheHit)
		h.assert.Equal(testContent, buf)
		h.assert.Nil(h.rr.wrapped.fileCacheHandle)

		objectData, err = h.rr.ReadAt(buf, 0)

		h.assert.NoError(err)
		h.assert.True(objectData.CacheHit)
		h.assert.Equal(testContent, buf)
		h.assert.NotNil(h.rr.wrapped.fileCacheHandle)
	})
}

func TestRandomReader_Test_ReadAt_IfCacheFileGetsDeleted(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		h.rr.wrapped.fileCacheHandler = h.cacheHandler
		objectSize := h.object.Size
		testContent := testutil.GenerateRandomBytes(int(objectSize))
		rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
		h.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd)
		h.bucket.On("Name").Return("test")
		h.bucket.On("BucketType").Return(h.bucketType).Once()
		buf := make([]byte, objectSize)
		objectData, err := h.rr.ReadAt(buf, 0)
		h.require.NoError(err)
		h.assert.False(objectData.CacheHit)
		h.require.Equal(testContent, buf)
		h.require.NotNil(h.rr.wrapped.fileCacheHandle)
		err = h.rr.wrapped.fileCacheHandle.Close()
		h.require.NoError(err)
		h.rr.wrapped.fileCacheHandle = nil
		filePath, err := util.GetDownloadPath(h.cacheDir, util.GetObjectPath(h.bucket.Name(), h.object.Name))
		h.require.NoError(err)
		err = os.Remove(filePath)
		h.require.NoError(err)

		_, err = h.rr.ReadAt(buf, 0)

		h.require.Error(err)
		h.require.True(errors.Is(err, util.ErrFileNotPresentInCache))
	})
}

func TestRandomReader_Test_ReadAt_IfCacheFileGetsDeletedWithCacheHandleOpen(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		h.rr.wrapped.fileCacheHandler = h.cacheHandler
		objectSize := h.object.Size
		testContent := testutil.GenerateRandomBytes(int(objectSize))
		rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
		h.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd)
		h.bucket.On("Name").Return("test")
		h.bucket.On("BucketType").Return(h.bucketType)
		buf := make([]byte, objectSize)
		objectData, err := h.rr.ReadAt(buf, 0)
		h.require.NoError(err)
		h.assert.False(objectData.CacheHit)
		h.require.Equal(testContent, buf)
		h.require.NotNil(h.rr.wrapped.fileCacheHandle)
		filePath, err := util.GetDownloadPath(h.cacheDir, util.GetObjectPath(h.bucket.Name(), h.object.Name))
		h.require.NoError(err)
		err = os.Remove(filePath)
		h.require.NoError(err)

		objectData, err = h.rr.ReadAt(buf, 0)

		h.assert.NoError(err)
		h.assert.True(objectData.CacheHit)
		h.assert.Equal(testContent, buf)
		h.assert.NotNil(h.rr.wrapped.fileCacheHandle)
	})
}

func TestRandomReader_Test_ReadAt_FailedJobRestartAndCacheHit(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		h.rr.wrapped.fileCacheHandler = h.cacheHandler
		objectSize := h.object.Size
		testContent := testutil.GenerateRandomBytes(int(objectSize))
		rc := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
		h.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).
			Return(nil, errors.New("")).Once()
		h.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).
			Return(rc, nil).Once()
		h.bucket.On("Name").Return("test")
		h.bucket.On("BucketType").Return(h.bucketType)
		buf := make([]byte, objectSize)
		objectData, err := h.rr.ReadAt(buf, 0)
		h.require.NoError(err)
		h.assert.False(objectData.CacheHit)
		h.require.Equal(testContent, buf)
		job := h.jobManager.GetJob(h.object.Name, h.bucket.Name())
		h.require.True(job == nil || job.GetStatus().Name == downloader.Failed)
		rd2 := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
		h.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd2)
		objectData, err = h.rr.ReadAt(buf, 0)
		h.assert.NoError(err)
		h.assert.False(objectData.CacheHit)
		h.assert.Equal(testContent, buf)
		h.assert.NotNil(h.rr.wrapped.fileCacheHandle)

		objectData, err = h.rr.ReadAt(buf, 0)

		h.assert.NoError(err)
		h.assert.True(objectData.CacheHit)
		h.assert.Equal(testContent, buf)
		h.assert.NotNil(h.rr.wrapped.fileCacheHandle)
	})
}

func TestRandomReader_Test_tryReadingFromFileCache_CacheHit(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		h.rr.wrapped.fileCacheHandler = h.cacheHandler
		objectSize := h.object.Size
		testContent := testutil.GenerateRandomBytes(int(objectSize))
		rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
		h.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd)
		h.bucket.On("Name").Return("test")
		h.bucket.On("BucketType").Return(h.bucketType)
		buf := make([]byte, objectSize)
		_, cacheHit, err := h.rr.wrapped.tryReadingFromFileCache(h.rr.ctx, buf, 0)
		h.assert.False(cacheHit)
		h.assert.NoError(err)
		h.assert.Equal(testContent, buf)

		_, cacheHit, err = h.rr.wrapped.tryReadingFromFileCache(h.rr.ctx, buf, 0)
		h.assert.True(cacheHit)
		h.assert.NoError(err)
		h.assert.Equal(testContent, buf)
	})
}

func TestRandomReader_Test_tryReadingFromFileCache_CacheMiss(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		h.rr.wrapped.fileCacheHandler = h.cacheHandler
		h.rr.wrapped.cacheFileForRangeRead = false
		start := 5
		end := 10
		h.bucket.On("Name").Return("test")
		buf := make([]byte, end-start)

		_, cacheHit, err := h.rr.wrapped.tryReadingFromFileCache(h.rr.ctx, buf, int64(start))

		h.assert.False(cacheHit)
		h.assert.NoError(err)
	})
}

func TestRandomReader_Test_ReadAt_OffsetEqualToObjectSize(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		h.rr.wrapped.fileCacheHandler = h.cacheHandler
		h.object.Size = util.MiB
		objectSize := h.object.Size
		testContent := testutil.GenerateRandomBytes(int(objectSize))
		rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
		h.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd)
		h.bucket.On("Name").Return("test")
		h.bucket.On("BucketType").Return(h.bucketType).Once()
		start1 := 0
		end1 := util.MiB
		buf := make([]byte, end1-start1)
		objectData, err := h.rr.ReadAt(buf, int64(start1))
		h.assert.False(objectData.CacheHit)
		h.assert.NoError(err)
		h.assert.Equal(testContent[start1:end1], buf)
		start2 := util.MiB
		end2 := start2 + util.MiB
		buf2 := make([]byte, end2-start2)

		objectData, err = h.rr.ReadAt(buf2, int64(start2))

		h.assert.False(objectData.CacheHit)
		h.assert.Equal(io.EOF, err)
		h.assert.Equal(0, objectData.Size)
	})
}

func TestRandomReader_Test_Destroy_NilCacheHandle(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		h.rr.wrapped.fileCacheHandler = h.cacheHandler

		h.rr.Destroy()

		h.assert.Nil(h.rr.wrapped.fileCacheHandle)
	})
}

func TestRandomReader_Test_Destroy_NonNilCacheHandle(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		h.rr.wrapped.fileCacheHandler = h.cacheHandler
		objectSize := h.object.Size
		testContent := testutil.GenerateRandomBytes(int(objectSize))
		rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
		h.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd)
		h.bucket.On("Name").Return("test")
		h.bucket.On("BucketType").Return(h.bucketType).Once()
		buf := make([]byte, objectSize)
		_, cacheHit, err := h.rr.wrapped.tryReadingFromFileCache(h.rr.ctx, buf, 0)
		h.assert.False(cacheHit)
		h.assert.NoError(err)
		h.assert.Equal(testContent, buf)
		h.assert.NotNil(h.rr.wrapped.fileCacheHandle)

		h.rr.wrapped.Destroy()

		h.assert.Nil(h.rr.wrapped.fileCacheHandle)
	})
}

func TestRandomReader_TestNewReader_FileClobbered(t *testing.T) {
	runTest(t, func(h *randomReaderTestHelper) {
		var notFoundError *gcs.NotFoundError

		h.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).
			Return(nil, notFoundError).Once()

		err := h.rr.wrapped.startRead(context.Background(), 0, 1, 0)

		h.assert.Error(err)
		var clobberedErr *gcsfuse_errors.FileClobberedError
		h.assert.True(errors.As(err, &clobberedErr))
	})
}
