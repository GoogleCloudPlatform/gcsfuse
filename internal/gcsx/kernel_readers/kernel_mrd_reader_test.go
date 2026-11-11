// Copyright 2026 Google LLC
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

package kernel_readers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/mem"
	"google.golang.org/grpc/status"
)

type KernelMRDReaderTest struct {
	suite.Suite
	object      *gcs.MinObject
	bucket      *storage.TestifyMockBucket
	cache       *lru.Cache
	inodeID     fuseops.InodeID
	config      *cfg.Config
	mrdInstance *gcsx.MrdInstance
	reader      *KernelMRDReader
}

func TestKernelMRDReaderTestSuite(t *testing.T) {
	suite.Run(t, new(KernelMRDReaderTest))
}

func (t *KernelMRDReaderTest) SetupTest() {
	t.object = &gcs.MinObject{
		Name:       "foo",
		Size:       1024 * gcsx.MiB,
		Generation: 1234,
	}
	t.bucket = new(storage.TestifyMockBucket)
	t.cache = lru.NewCache(2)
	t.inodeID = 100
	t.config = &cfg.Config{Mrd: cfg.MrdConfig{PoolSize: 1}}

	t.mrdInstance = gcsx.NewMrdInstance(t.object, t.bucket, t.cache, t.inodeID, t.config)
	t.reader = NewKernelMRDReader(t.mrdInstance, metrics.NewNoopMetrics())
}

func (t *KernelMRDReaderTest) TestNewKernelMRDReader() {
	assert.NotNil(t.T(), t.reader)
	assert.Equal(t.T(), t.mrdInstance, t.reader.mrdInstance)
}

func (t *KernelMRDReaderTest) TestReadAt_EmptyBuffer() {
	req := &gcsx.ReadRequest{
		Buffer: []byte{},
		Offset: 0,
	}

	resp, err := t.reader.ReadAt(context.Background(), req)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 0, resp.Size)
}

func (t *KernelMRDReaderTest) TestReadAt_Success() {
	data := []byte("hello world")
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, data)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	req := &gcsx.ReadRequest{
		Buffer: make([]byte, 5),
		Offset: 0,
	}
	// Verify initial refCount
	assert.Equal(t.T(), int64(0), t.mrdInstance.RefCount())

	resp, err := t.reader.ReadAt(context.Background(), req)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 5, resp.Size)
	assert.Equal(t.T(), "hello", string(req.Buffer))
	// Verify refCount incremented
	assert.Equal(t.T(), int64(1), t.mrdInstance.RefCount())
}

func (t *KernelMRDReaderTest) TestReadAt_MultipleCalls() {
	data := []byte("hello world")
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, data)
	// Expect NewMultiRangeDownloader only once because subsequent reads reuse the instance/pool
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	req := &gcsx.ReadRequest{
		Buffer: make([]byte, 5),
		Offset: 0,
	}

	// First call
	_, err := t.reader.ReadAt(context.Background(), req)
	assert.NoError(t.T(), err)

	// Second call
	_, err = t.reader.ReadAt(context.Background(), req)
	assert.NoError(t.T(), err)

	// Verify refCount is still 1
	assert.Equal(t.T(), int64(1), t.mrdInstance.RefCount())
}

func (t *KernelMRDReaderTest) TestReadAt_NilMrdInstance() {
	t.reader.mrdInstance = nil
	req := &gcsx.ReadRequest{
		Buffer: make([]byte, 5),
		Offset: 0,
	}

	resp, err := t.reader.ReadAt(context.Background(), req)

	assert.Error(t.T(), err)
	assert.Contains(t.T(), err.Error(), "mrdInstance is nil")
	assert.Equal(t.T(), 0, resp.Size)
}

func (t *KernelMRDReaderTest) TestReadAt_ShortRead_NoRetry() {
	data := []byte("hello world")
	// MRD returns short read.
	fakeMRD := fake.NewFakeMultiRangeDownloaderWithShortRead(t.object, data)
	// Expectation:
	// 1. Read calls ensureMRDPool -> NewMRDPool -> NewMultiRangeDownloader. Returns fakeMRD.
	// 2. Read returns short read.
	// 3. isShortRead returns false because err is nil.
	// 4. ReadAt returns the short read without retrying.
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	buf := make([]byte, len(data))
	req := &gcsx.ReadRequest{Buffer: buf, Offset: 0}

	resp, err := t.reader.ReadAt(context.Background(), req)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 5, resp.Size) // Short read size
	assert.Equal(t.T(), "hello", string(buf[:5]))
	// Verify refCount incremented (only once)
	assert.Equal(t.T(), int64(1), t.mrdInstance.RefCount())
	t.bucket.AssertExpectations(t.T())
}

func (t *KernelMRDReaderTest) TestReadAt_OutOfRange_TriggersRetry() {
	data := []byte("hello world")
	// First MRD returns OutOfRange error.
	outOfRangeErr := status.Error(codes.OutOfRange, "Out of range")
	fakeMRD1 := fake.NewFakeMultiRangeDownloaderWithSleepAndDefaultError(t.object, []byte{}, 0, outOfRangeErr)
	// Second MRD returns full read.
	fakeMRD2 := fake.NewFakeMultiRangeDownloader(t.object, data)
	// Expectation:
	// 1. Initial Read calls ensureMRDPool -> NewMRDPool -> NewMultiRangeDownloader. Returns fakeMRD1.
	// 2. Read returns OutOfRange. isShortRead detects this as recoverable.
	// 3. ReadAt calls RecreateMRD -> NewMRDPool -> NewMultiRangeDownloader. Returns fakeMRD2.
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD1, nil).Once()
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD2, nil).Once()
	// Create the ReadRequest.
	buf := make([]byte, len(data))
	req := &gcsx.ReadRequest{
		Buffer: buf,
		Offset: 0,
	}

	resp, err := t.reader.ReadAt(context.Background(), req)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), len(data), resp.Size)
	assert.Equal(t.T(), string(data), string(buf))
	// Verify refCount incremented
	assert.Equal(t.T(), int64(1), t.mrdInstance.RefCount())
	t.bucket.AssertExpectations(t.T())
}

func (t *KernelMRDReaderTest) TestReadAt_OutOfRange_RetryFails() {
	// First MRD returns OutOfRange error.
	outOfRangeErr := status.Error(codes.OutOfRange, "Out of range")
	fakeMRD1 := fake.NewFakeMultiRangeDownloaderWithSleepAndDefaultError(t.object, []byte{}, 0, outOfRangeErr)
	// Second MRD returns Internal error.
	retryErr := status.Error(codes.Internal, "Internal error")
	fakeMRD2 := fake.NewFakeMultiRangeDownloaderWithSleepAndDefaultError(t.object, []byte{}, 0, retryErr)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD1, nil).Once()
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD2, nil).Once()
	// Create the ReadRequest.
	buf := make([]byte, 10)
	req := &gcsx.ReadRequest{
		Buffer: buf,
		Offset: 0,
	}

	resp, err := t.reader.ReadAt(context.Background(), req)

	// Should return the error from the retry.
	assert.ErrorIs(t.T(), err, retryErr)
	assert.Equal(t.T(), 0, resp.Size)
	t.bucket.AssertExpectations(t.T())
}

func TestIsShortRead(t *testing.T) {
	testCases := []struct {
		name       string
		bytesRead  int
		bufferSize int
		err        error
		expected   bool
	}{
		{
			name:       "Full read, no error",
			bytesRead:  10,
			bufferSize: 10,
			err:        nil,
			expected:   false,
		},
		{
			name:       "Full read, error",
			bytesRead:  10,
			bufferSize: 10,
			err:        errors.New("error"),
			expected:   false,
		},
		{
			name:       "Short read, no error",
			bytesRead:  5,
			bufferSize: 10,
			err:        nil,
			expected:   false,
		},
		{
			name:       "Short read, OutOfRange",
			bytesRead:  0,
			bufferSize: 10,
			err:        status.Error(codes.OutOfRange, "out of range"),
			expected:   true,
		},
		{
			name:       "Short read, Wrapped OutOfRange",
			bytesRead:  0,
			bufferSize: 10,
			err:        fmt.Errorf("wrapped: %w", status.Error(codes.OutOfRange, "out of range")),
			expected:   true,
		},
		{
			name:       "Short read, Internal error",
			bytesRead:  5,
			bufferSize: 10,
			err:        status.Error(codes.Internal, "internal error"),
			expected:   false,
		},
		{
			name:       "Short read, Generic error",
			bytesRead:  5,
			bufferSize: 10,
			err:        errors.New("generic error"),
			expected:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, isShortRead(tc.bytesRead, tc.bufferSize, tc.err))
		})
	}
}

func (t *KernelMRDReaderTest) TestDestroy() {
	// Setup state where refCount is incremented
	t.reader.mrdInstanceInUse.Store(true)
	t.mrdInstance.IncrementRefCount()

	t.reader.Destroy()

	assert.Nil(t.T(), t.reader.mrdInstance)
	assert.False(t.T(), t.reader.mrdInstanceInUse.Load())
	// Verify refCount decremented
	assert.Equal(t.T(), int64(0), t.mrdInstance.RefCount())
	// Verify that calling Destroy again doesn't panic
	t.reader.Destroy()
}

func (t *KernelMRDReaderTest) TestReadAt_RecreateMRDFails_RetriesWithOldMRD() {
	// First MRD returns OutOfRange error.
	outOfRangeErr := status.Error(codes.OutOfRange, "Out of range")
	fakeMRD1 := fake.NewFakeMultiRangeDownloaderWithSleepAndDefaultError(t.object, []byte{}, 0, outOfRangeErr)
	// Expectation:
	// 1. Initial Read calls ensureMRDPool -> NewMRDPool -> NewMultiRangeDownloader. Returns fakeMRD1.
	// 2. Read returns OutOfRange.
	// 3. ReadAt calls RecreateMRD -> NewMRDPool -> NewMultiRangeDownloader. Returns ERROR.
	// 4. ReadAt logs warning and retries with existing pool (fakeMRD1).
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD1, nil).Once()
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(nil, errors.New("recreate failed")).Once()
	buf := make([]byte, 10)
	req := &gcsx.ReadRequest{
		Buffer: buf,
		Offset: 0,
	}

	resp, err := t.reader.ReadAt(context.Background(), req)

	// Assert
	// We verify that we didn't get the "recreate failed" error.
	assert.ErrorIs(t.T(), err, outOfRangeErr)
	assert.NotEqual(t.T(), "recreate failed", err.Error())
	// We expect 0 bytes to be read as OutOfRange returns 0 bytes.
	assert.Equal(t.T(), 0, resp.Size)
	t.bucket.AssertExpectations(t.T())
}

func (t *KernelMRDReaderTest) TestDestroy_NoReadAt() {
	// Check initial refCount
	assert.Equal(t.T(), int64(0), t.mrdInstance.RefCount())
	// Capture logs
	var buf bytes.Buffer
	logger.SetOutput(&buf)
	defer logger.SetOutput(os.Stdout)

	// Act
	t.reader.Destroy()

	// Assert
	assert.Equal(t.T(), int64(0), t.mrdInstance.RefCount())
	assert.NotContains(t.T(), buf.String(), "MrdInstance::DecrementRefCount: Refcount cannot be negative")
}

func (t *KernelMRDReaderTest) TestReadAt_ContextCanceled() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := &gcsx.ReadRequest{
		Buffer: make([]byte, 10),
		Offset: 0,
	}
	// Use sleep to ensure context cancellation is detected before read completes
	fakeMRD := fake.NewFakeMultiRangeDownloaderWithSleep(t.object, []byte("data"), 100*time.Millisecond)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()

	resp, err := t.reader.ReadAt(ctx, req)

	assert.ErrorIs(t.T(), err, context.Canceled)
	assert.Equal(t.T(), 0, resp.Size)
}

type pinnedFakeMRD struct {
	gcs.MultiRangeDownloader
	addFn func(output io.Writer, offset, length int64, callback func(int64, int64, error))
}

func (m *pinnedFakeMRD) Add(output io.Writer, offset, length int64, callback func(int64, int64, error)) {
	m.addFn(output, offset, length, callback)
}

func (m *pinnedFakeMRD) Close() error      { return nil }
func (m *pinnedFakeMRD) Wait()             {}
func (m *pinnedFakeMRD) Error() error      { return nil }
func (m *pinnedFakeMRD) GetHandle() []byte { return nil }

func (t *KernelMRDReaderTest) TestReadAt_PartialShortReadWithPinnedBuffer_OutOfRangeRetryPreservesPrefix() {
	outOfRangeErr := status.Error(codes.OutOfRange, "out of range")
	pool := mem.DefaultBufferPool()
	rawBuf := pool.Get(2048)
	copy((*rawBuf)[:6], "hello ")
	pinnedSlice := (*rawBuf)[:6]

	firstMRD := &pinnedFakeMRD{
		addFn: func(output io.Writer, offset, length int64, callback func(int64, int64, error)) {
			n, _ := output.Write(pinnedSlice)
			pool.Put(rawBuf)
			callback(offset, int64(n), outOfRangeErr)
		},
	}
	secondMRD := fake.NewFakeMultiRangeDownloader(t.object, []byte("hello world"))
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(firstMRD, nil).Once()
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(secondMRD, nil).Once()

	buf := make([]byte, 11)
	resp, err := t.reader.ReadAt(context.Background(), &gcsx.ReadRequest{
		Buffer: buf,
		Offset: 0,
	})

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 11, resp.Size)
	assert.Equal(t.T(), "hello world", string(buf))
	t.bucket.AssertExpectations(t.T())
}

func (t *KernelMRDReaderTest) TestAcquirePrefetchChunk_OutOfOrderConcurrentReadahead_DoesNotPrematurelyEvict() {
	t.object.Size = 16 * gcsx.MiB
	data := bytes.Repeat([]byte("A"), int(t.object.Size))
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, data)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()

	// Seed prefetch queue at offset 1 MiB.
	t.reader.mu.Lock()
	t.reader.lastReadEnd = 1 * gcsx.MiB
	t.reader.nextPrefetchOffset = 1 * gcsx.MiB
	t.reader.seqReads = 1
	t.reader.schedulePrefetchLocked()
	// First chunk is [1 MiB, 3 MiB) (2 MiB initialPrefetchChunkSize), second is [3 MiB, 7 MiB).
	assert.Len(t.T(), t.reader.chunks, 2)
	assert.Equal(t.T(), int64(1*gcsx.MiB), t.reader.chunks[0].Offset)
	assert.Equal(t.T(), int(2*gcsx.MiB), t.reader.chunks[0].Length)
	t.reader.mu.Unlock()

	// Out-of-order lock acquisition: [2 MiB, 3 MiB) acquires lock before [1 MiB, 2 MiB).
	c2, s2, e2 := t.reader.acquirePrefetchChunk(2*gcsx.MiB, 1*gcsx.MiB)
	assert.NotNil(t.T(), c2)
	assert.Equal(t.T(), int(1*gcsx.MiB), s2)
	assert.Equal(t.T(), int(2*gcsx.MiB), e2)

	// Chunk [1 MiB, 3 MiB) must still be in kmr.chunks because only 1 MiB of 2 MiB was consumed.
	t.reader.mu.Lock()
	assert.Equal(t.T(), int64(1*gcsx.MiB), t.reader.chunks[0].Offset)
	assert.Equal(t.T(), int64(3*gcsx.MiB), t.reader.lastReadEnd)
	t.reader.mu.Unlock()

	// Now [1 MiB, 2 MiB) acquires lock: should hit the same chunk, evict it as fully consumed,
	// preserve the lookahead queue, and keep lastReadEnd monotonic at 3 MiB.
	c1, s1, e1 := t.reader.acquirePrefetchChunk(1*gcsx.MiB, 1*gcsx.MiB)
	assert.Same(t.T(), c2, c1)
	assert.Equal(t.T(), 0, s1)
	assert.Equal(t.T(), int(1*gcsx.MiB), e1)

	t.reader.mu.Lock()
	assert.NotEmpty(t.T(), t.reader.chunks)
	assert.Equal(t.T(), int64(3*gcsx.MiB), t.reader.chunks[0].Offset)
	assert.Equal(t.T(), int64(3*gcsx.MiB), t.reader.lastReadEnd)
	t.reader.mu.Unlock()

	c1.Release()
	c2.Release()
	t.reader.Destroy()
}

func (t *KernelMRDReaderTest) TestReadAt_InvalidatesPrefetchOnGenerationOrSizeChange() {
	t.object.Size = 8 * gcsx.MiB
	oldData := bytes.Repeat([]byte("O"), int(t.object.Size))
	fakeMRD1 := fake.NewFakeMultiRangeDownloader(t.object, oldData)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD1, nil).Once()

	// Seed prefetch queue for generation 1234 starting at offset 1 MiB.
	t.reader.mu.Lock()
	t.reader.lastReadEnd = 1 * gcsx.MiB
	t.reader.nextPrefetchOffset = 1 * gcsx.MiB
	t.reader.seqReads = 1
	t.reader.schedulePrefetchLocked()
	assert.NotEmpty(t.T(), t.reader.chunks)
	t.reader.mu.Unlock()

	// Update object generation to 1235 with new data.
	newObj := &gcs.MinObject{
		Name:       t.object.Name,
		Size:       8 * gcsx.MiB,
		Generation: 1235,
	}
	newData := bytes.Repeat([]byte("N"), int(newObj.Size))
	fakeMRD2 := fake.NewFakeMultiRangeDownloader(newObj, newData)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD2, nil).Once()
	assert.NoError(t.T(), t.mrdInstance.SetMinObject(newObj))

	// Read [1 MiB, 2 MiB): must invalidate stale gen-1234 prefetch chunks and serve new gen-1235 data.
	buf := make([]byte, 1*gcsx.MiB)
	resp, err := t.reader.ReadAt(context.Background(), &gcsx.ReadRequest{
		Buffer: buf,
		Offset: 1 * gcsx.MiB,
	})
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 1*gcsx.MiB, resp.Size)
	assert.Equal(t.T(), byte('N'), buf[0])
	if resp.Callback != nil {
		resp.Callback()
	}

	// Re-seed prefetch queue for gen 1235, then simulate truncation to 512 KiB.
	t.reader.mu.Lock()
	t.reader.lastReadEnd = 1 * gcsx.MiB
	t.reader.nextPrefetchOffset = 1 * gcsx.MiB
	t.reader.seqReads = 1
	t.reader.schedulePrefetchLocked()
	assert.NotEmpty(t.T(), t.reader.chunks)
	t.reader.mu.Unlock()

	truncatedObj := &gcs.MinObject{
		Name:       t.object.Name,
		Size:       512 * 1024,
		Generation: 1235,
	}
	assert.NoError(t.T(), t.mrdInstance.SetMinObject(truncatedObj))

	chunk, _, _ := t.reader.acquirePrefetchChunk(1*gcsx.MiB, 1*gcsx.MiB)
	assert.Nil(t.T(), chunk)
	t.reader.mu.Lock()
	assert.Empty(t.T(), t.reader.chunks)
	t.reader.mu.Unlock()
}

func (t *KernelMRDReaderTest) TestReadAt_SmallReadsConsumeAndClearPrefetch() {
	t.object.Size = 8 * gcsx.MiB
	data := bytes.Repeat([]byte("P"), int(t.object.Size))
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, data)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()

	// Seed prefetch queue at offset 256 KiB.
	t.reader.mu.Lock()
	t.reader.lastReadEnd = 256 * 1024
	t.reader.nextPrefetchOffset = 256 * 1024
	t.reader.seqReads = 1
	t.reader.schedulePrefetchLocked()
	assert.Len(t.T(), t.reader.chunks, 2)
	t.reader.mu.Unlock()

	// 1. Small sequential read (< minPrefetchReadSize = 256 KiB) at offset 256 KiB hits prefetched chunk.
	smallSeqBuf := make([]byte, 64*1024)
	resp, err := t.reader.ReadAt(context.Background(), &gcsx.ReadRequest{
		Buffer: smallSeqBuf,
		Offset: 256 * 1024,
	})
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 64*1024, resp.Size)
	assert.NotNil(t.T(), resp.Callback, "Small sequential read should be served zero-copy from prefetched chunk")
	resp.Callback()

	// 2. Small random read (< 256 KiB) outside prefetched range clears stale lookahead chunks.
	smallRandBuf := make([]byte, 4096)
	resp2, err := t.reader.ReadAt(context.Background(), &gcsx.ReadRequest{
		Buffer: smallRandBuf,
		Offset: 7 * gcsx.MiB,
	})
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 4096, resp2.Size)
	if resp2.Callback != nil {
		resp2.Callback()
	}

	t.reader.mu.Lock()
	assert.Empty(t.T(), t.reader.chunks, "Small random read must clear stale prefetched chunks")
	assert.Equal(t.T(), 0, t.reader.seqReads)
	t.reader.mu.Unlock()
}
