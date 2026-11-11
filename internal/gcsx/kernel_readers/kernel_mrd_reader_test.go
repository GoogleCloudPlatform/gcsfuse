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
	"sync"
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

func (t *KernelMRDReaderTest) TestReadAt_ConcurrentInitialSequentialReads_InOrderAndOutOfOrder() {
	for _, outOfOrder := range []bool{false, true} {
		name := "InOrder"
		if outOfOrder {
			name = "OutOfOrder"
		}
		t.Run(name, func() {
			t.SetupTest()
			t.object.Size = 16 * gcsx.MiB
			pool := mem.DefaultBufferPool()

			entered0 := make(chan struct{})
			entered1 := make(chan struct{})
			release0 := make(chan struct{})
			release1 := make(chan struct{})

			mrd := &pinnedFakeMRD{
				addFn: func(output io.Writer, offset, length int64, callback func(int64, int64, error)) {
					go func() {
						switch offset {
						case 0:
							close(entered0)
							<-release0
						case 1 * gcsx.MiB:
							close(entered1)
							<-release1
						}
						rawBuf := pool.Get(int(length))
						slice := (*rawBuf)[:length]
						fillByte := byte('A' + (offset / gcsx.MiB))
						for i := range slice {
							slice[i] = fillByte
						}
						n, _ := output.Write(slice)
						pool.Put(rawBuf)
						callback(offset, int64(n), nil)
					}()
				},
			}
			t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(mrd, nil).Once()

			var (
				resp0, resp1 gcsx.ReadResponse
				err0, err1   error
				done0        = make(chan struct{})
				done1        = make(chan struct{})
			)

			go func() {
				defer close(done0)
				buf0 := make([]byte, 1*gcsx.MiB)
				resp0, err0 = t.reader.ReadAt(context.Background(), &gcsx.ReadRequest{
					Buffer: buf0,
					Offset: 0,
				})
			}()

			go func() {
				defer close(done1)
				buf1 := make([]byte, 1*gcsx.MiB)
				resp1, err1 = t.reader.ReadAt(context.Background(), &gcsx.ReadRequest{
					Buffer: buf1,
					Offset: 1 * gcsx.MiB,
				})
			}()

			// Wait until both concurrent initial reads are in-flight inside ReadVectored.
			<-entered0
			<-entered1

			if !outOfOrder {
				close(release0)
				<-done0
				close(release1)
				<-done1
			} else {
				// Out-of-order completion: [1 MiB, 2 MiB) completes before [0, 1 MiB).
				close(release1)
				<-done1
				t.reader.mu.Lock()
				assert.Equal(t.T(), int64(2*gcsx.MiB), t.reader.lastReadEnd)
				t.reader.mu.Unlock()

				close(release0)
				<-done0
			}

			assert.NoError(t.T(), err0)
			assert.NoError(t.T(), err1)
			assert.Equal(t.T(), 1*gcsx.MiB, resp0.Size)
			assert.Equal(t.T(), 1*gcsx.MiB, resp1.Size)
			if resp0.Callback != nil {
				resp0.Callback()
			}
			if resp1.Callback != nil {
				resp1.Callback()
			}

			t.reader.mu.Lock()
			assert.Empty(t.T(), t.reader.inFlight)
			assert.Equal(t.T(), int64(2*gcsx.MiB), t.reader.lastReadEnd, "lastReadEnd must never rewind to 1 MiB")
			assert.Equal(t.T(), 2, t.reader.seqReads)
			assert.NotEmpty(t.T(), t.reader.chunks)
			assert.Equal(t.T(), int64(2*gcsx.MiB), t.reader.chunks[0].Offset, "Prefetch must start at 2 MiB past both concurrent in-flight ranges")
			t.reader.mu.Unlock()

			// Read [2 MiB, 6 MiB) in 1 MiB steps and verify no zombie chunks remain at the front.
			for step := int64(2); step < 6; step++ {
				buf := make([]byte, 1*gcsx.MiB)
				resp, err := t.reader.ReadAt(context.Background(), &gcsx.ReadRequest{
					Buffer: buf,
					Offset: step * gcsx.MiB,
				})
				assert.NoError(t.T(), err)
				assert.Equal(t.T(), 1*gcsx.MiB, resp.Size)
				assert.NotNil(t.T(), resp.Callback)
				resp.Callback()
			}

			t.reader.mu.Lock()
			assert.NotEmpty(t.T(), t.reader.chunks)
			assert.Equal(t.T(), int64(6*gcsx.MiB), t.reader.chunks[0].Offset, "Consumed chunks must be evicted without stuck zombie chunks")
			t.reader.mu.Unlock()

			t.reader.Destroy()
		})
	}
}

func (t *KernelMRDReaderTest) TestReadAt_StraddlingPrefetchChunkBoundary_ServedZeroCopyWithoutClearingQueue() {
	t.object.Size = 16 * gcsx.MiB
	pool := mem.DefaultBufferPool()

	var (
		mu             sync.Mutex
		requestedSpans [][2]int64
	)

	mrd := &pinnedFakeMRD{
		addFn: func(output io.Writer, offset, length int64, callback func(int64, int64, error)) {
			mu.Lock()
			requestedSpans = append(requestedSpans, [2]int64{offset, length})
			mu.Unlock()

			rawBuf := pool.Get(int(length))
			slice := (*rawBuf)[:length]
			for i := range slice {
				slice[i] = byte((offset + int64(i)) / (256 * 1024))
			}
			n, _ := output.Write(slice)
			pool.Put(rawBuf)
			callback(offset, int64(n), nil)
		},
	}
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(mrd, nil).Once()

	// 1. Initial 256 KiB read at offset 0 triggers prefetch starting at 256 KiB:
	//    chunk[0] = [256 KiB, 2304 KiB) (2 MiB), chunk[1] = [2304 KiB, 6400 KiB) (4 MiB).
	resp1, err := t.reader.ReadAt(context.Background(), &gcsx.ReadRequest{
		Buffer: make([]byte, 256*1024),
		Offset: 0,
	})
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 256*1024, resp1.Size)
	assert.NotNil(t.T(), resp1.Callback)
	resp1.Callback()

	t.reader.mu.Lock()
	assert.Len(t.T(), t.reader.chunks, 2)
	assert.Equal(t.T(), int64(256*1024), t.reader.chunks[0].Offset)
	assert.Equal(t.T(), int(2*gcsx.MiB), t.reader.chunks[0].Length)
	assert.Equal(t.T(), int64(2304*1024), t.reader.chunks[1].Offset)
	assert.Equal(t.T(), int(4*gcsx.MiB), t.reader.chunks[1].Length)
	t.reader.mu.Unlock()

	// 2. Read [256 KiB, 768 KiB) (512 KiB) from chunk[0].
	resp2, err := t.reader.ReadAt(context.Background(), &gcsx.ReadRequest{
		Buffer: make([]byte, 512*1024),
		Offset: 256 * 1024,
	})
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 512*1024, resp2.Size)
	assert.NotNil(t.T(), resp2.Callback)
	resp2.Callback()

	// 3. Read [768 KiB, 1792 KiB) (1 MiB) from chunk[0].
	resp3, err := t.reader.ReadAt(context.Background(), &gcsx.ReadRequest{
		Buffer: make([]byte, 1*gcsx.MiB),
		Offset: 768 * 1024,
	})
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 1*gcsx.MiB, resp3.Size)
	assert.NotNil(t.T(), resp3.Callback)
	resp3.Callback()

	// 4. Read [1792 KiB, 2816 KiB) (1 MiB) straddling the 2304 KiB boundary between chunk[0] and chunk[1].
	resp4, err := t.reader.ReadAt(context.Background(), &gcsx.ReadRequest{
		Buffer: make([]byte, 1*gcsx.MiB),
		Offset: 1792 * 1024,
	})
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 1*gcsx.MiB, resp4.Size)
	assert.NotNil(t.T(), resp4.Callback, "Straddling sequential read must be served zero-copy from contiguous prefetch chunks")

	// Verify concatenated payload across the 2304 KiB chunk boundary.
	var got bytes.Buffer
	for _, s := range resp4.Data {
		got.Write(s)
	}
	assert.Equal(t.T(), 1*gcsx.MiB, got.Len())
	gotBytes := got.Bytes()
	assert.Equal(t.T(), byte((1792*1024)/(256*1024)), gotBytes[0])
	assert.Equal(t.T(), byte((2304*1024-1)/(256*1024)), gotBytes[512*1024-1])
	assert.Equal(t.T(), byte((2304*1024)/(256*1024)), gotBytes[512*1024])
	assert.Equal(t.T(), byte((2816*1024-1)/(256*1024)), gotBytes[1*gcsx.MiB-1])
	resp4.Callback()

	// Verify chunk[0] ([256 KiB, 2304 KiB)) was evicted while chunk[1] ([2304 KiB, 6400 KiB)) remained
	// with Consumed == 512 KiB and a new lookahead chunk [6400 KiB, 10496 KiB) was scheduled.
	t.reader.mu.Lock()
	assert.Len(t.T(), t.reader.chunks, 2, "Prefetch queue must not be cleared on straddling read")
	assert.Equal(t.T(), int64(2304*1024), t.reader.chunks[0].Offset)
	assert.Equal(t.T(), 512*1024, t.reader.chunks[0].Consumed)
	assert.Equal(t.T(), int64(6400*1024), t.reader.chunks[1].Offset)
	t.reader.mu.Unlock()

	// Verify no synchronous fallback ReadVectored download occurred for [1792 KiB, 2816 KiB).
	mu.Lock()
	for _, sp := range requestedSpans {
		assert.NotEqual(t.T(), [2]int64{1792 * 1024, 1 * gcsx.MiB}, sp, "Straddling read must not fall back to ReadVectored")
	}
	mu.Unlock()

	t.reader.Destroy()
}

func (t *KernelMRDReaderTest) TestReadAt_ContextCancellationDuringChunkWait_RollsBackConsumedAndReinsertsEvictedChunk() {
	t.object.Size = 16 * gcsx.MiB
	pool := mem.DefaultBufferPool()

	finishChunk0 := make(chan struct{})
	mrd := &pinnedFakeMRD{
		addFn: func(output io.Writer, offset, length int64, callback func(int64, int64, error)) {
			go func() {
				if offset == 1*gcsx.MiB {
					<-finishChunk0
				}
				rawBuf := pool.Get(int(length))
				slice := (*rawBuf)[:length]
				for i := range slice {
					slice[i] = 'Z'
				}
				n, _ := output.Write(slice)
				pool.Put(rawBuf)
				callback(offset, int64(n), nil)
			}()
		},
	}
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(mrd, nil).Once()

	// Seed prefetch queue at offset 1 MiB: chunk[0] = [1 MiB, 3 MiB), chunk[1] = [3 MiB, 7 MiB).
	t.reader.mu.Lock()
	t.reader.lastReadEnd = 1 * gcsx.MiB
	t.reader.nextPrefetchOffset = 1 * gcsx.MiB
	t.reader.seqReads = 1
	t.reader.schedulePrefetchLocked()
	assert.Len(t.T(), t.reader.chunks, 2)
	origChunk0 := t.reader.chunks[0]
	t.reader.mu.Unlock()

	// 1. Cancel a full-chunk read [1 MiB, 3 MiB) (which temporarily evicts chunk[0] in acquirePrefetchSpans).
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()
	resp, err := t.reader.ReadAt(cancelCtx, &gcsx.ReadRequest{
		Buffer: make([]byte, 2*gcsx.MiB),
		Offset: 1 * gcsx.MiB,
	})
	assert.ErrorIs(t.T(), err, context.Canceled)
	assert.Equal(t.T(), 0, resp.Size)

	// Verify chunk[0] had Consumed rolled back to 0 and was re-inserted at the head of kmr.chunks.
	t.reader.mu.Lock()
	assert.NotEmpty(t.T(), t.reader.chunks)
	assert.Same(t.T(), origChunk0, t.reader.chunks[0], "Evicted chunk must be re-inserted at head of kmr.chunks on cancellation")
	assert.Equal(t.T(), 0, t.reader.chunks[0].Consumed, "Consumed must be rolled back to 0 on cancellation")
	t.reader.mu.Unlock()

	// 2. Cancel a partial-chunk read [1 MiB, 2 MiB) and verify Consumed is rolled back to 0.
	respPartial, errPartial := t.reader.ReadAt(cancelCtx, &gcsx.ReadRequest{
		Buffer: make([]byte, 1*gcsx.MiB),
		Offset: 1 * gcsx.MiB,
	})
	assert.ErrorIs(t.T(), errPartial, context.Canceled)
	assert.Equal(t.T(), 0, respPartial.Size)

	t.reader.mu.Lock()
	assert.Same(t.T(), origChunk0, t.reader.chunks[0])
	assert.Equal(t.T(), 0, t.reader.chunks[0].Consumed)
	t.reader.mu.Unlock()

	// 3. Unblock chunk[0] download and retry [1 MiB, 3 MiB): must succeed zero-copy from origChunk0.
	close(finishChunk0)
	retryResp, retryErr := t.reader.ReadAt(context.Background(), &gcsx.ReadRequest{
		Buffer: make([]byte, 2*gcsx.MiB),
		Offset: 1 * gcsx.MiB,
	})
	assert.NoError(t.T(), retryErr)
	assert.Equal(t.T(), 2*gcsx.MiB, retryResp.Size)
	assert.NotNil(t.T(), retryResp.Callback, "Retried read after context cancellation must hit the restored prefetch chunk")
	retryResp.Callback()

	t.reader.mu.Lock()
	assert.NotEmpty(t.T(), t.reader.chunks)
	assert.Equal(t.T(), int64(3*gcsx.MiB), t.reader.chunks[0].Offset)
	t.reader.mu.Unlock()

	t.reader.Destroy()
}

func (t *KernelMRDReaderTest) TestReadAt_FooterReadFollowedByRandomReadsAndBackwardSequentialSeek() {
	t.object.Size = 96 * gcsx.MiB
	pool := mem.DefaultBufferPool()

	var (
		mu             sync.Mutex
		requestedSpans [][2]int64
	)

	mrd := &pinnedFakeMRD{
		addFn: func(output io.Writer, offset, length int64, callback func(int64, int64, error)) {
			mu.Lock()
			requestedSpans = append(requestedSpans, [2]int64{offset, length})
			mu.Unlock()

			rawBuf := pool.Get(int(length))
			slice := (*rawBuf)[:length]
			for i := range slice {
				slice[i] = byte((offset + int64(i)) / gcsx.MiB)
			}
			n, _ := output.Write(slice)
			pool.Put(rawBuf)
			callback(offset, int64(n), nil)
		},
	}
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(mrd, nil).Once()

	// 1. High-offset footer read at [90 MiB, 91 MiB).
	respFooter, err := t.reader.ReadAt(context.Background(), &gcsx.ReadRequest{
		Buffer: make([]byte, 1*gcsx.MiB),
		Offset: 90 * gcsx.MiB,
	})
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 1*gcsx.MiB, respFooter.Size)
	if respFooter.Callback != nil {
		respFooter.Callback()
	}

	// 2. Non-contiguous random reads below 90 MiB must NOT trigger false prefetches.
	for _, off := range []int64{50 * gcsx.MiB, 20 * gcsx.MiB, 70 * gcsx.MiB} {
		respRand, errRand := t.reader.ReadAt(context.Background(), &gcsx.ReadRequest{
			Buffer: make([]byte, 1*gcsx.MiB),
			Offset: off,
		})
		assert.NoError(t.T(), errRand)
		assert.Equal(t.T(), 1*gcsx.MiB, respRand.Size)
		if respRand.Callback != nil {
			respRand.Callback()
		}
		t.reader.mu.Lock()
		assert.Empty(t.T(), t.reader.chunks, "Disjoint random reads below previous footer read must not trigger prefetch")
		assert.Equal(t.T(), 1, t.reader.seqReads)
		t.reader.mu.Unlock()
	}

	// 3. Backward seek to 10 MiB followed by sequential reads [10 MiB, 11 MiB), [11 MiB, 12 MiB), [12 MiB, 13 MiB).
	resp10, err := t.reader.ReadAt(context.Background(), &gcsx.ReadRequest{
		Buffer: make([]byte, 1*gcsx.MiB),
		Offset: 10 * gcsx.MiB,
	})
	assert.NoError(t.T(), err)
	if resp10.Callback != nil {
		resp10.Callback()
	}

	resp11, err := t.reader.ReadAt(context.Background(), &gcsx.ReadRequest{
		Buffer: make([]byte, 1*gcsx.MiB),
		Offset: 11 * gcsx.MiB,
	})
	assert.NoError(t.T(), err)
	if resp11.Callback != nil {
		resp11.Callback()
	}

	// After 2 sequential reads [10 MiB, 11 MiB) and [11 MiB, 12 MiB), prefetch must be scheduled at 12 MiB (NOT 91 MiB).
	t.reader.mu.Lock()
	assert.NotEmpty(t.T(), t.reader.chunks)
	assert.Equal(t.T(), int64(12*gcsx.MiB), t.reader.chunks[0].Offset, "Prefetch after backward seek must start at 12 MiB")
	t.reader.mu.Unlock()

	mu.Lock()
	countBefore12 := len(requestedSpans)
	mu.Unlock()

	// Reading [12 MiB, 13 MiB) must hit the prefetched chunk zero-copy without issuing a synchronous ReadVectored for [12 MiB, 13 MiB).
	resp12, err := t.reader.ReadAt(context.Background(), &gcsx.ReadRequest{
		Buffer: make([]byte, 1*gcsx.MiB),
		Offset: 12 * gcsx.MiB,
	})
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 1*gcsx.MiB, resp12.Size)
	assert.NotNil(t.T(), resp12.Callback)
	resp12.Callback()

	mu.Lock()
	for _, sp := range requestedSpans[countBefore12:] {
		assert.NotEqual(t.T(), [2]int64{12 * gcsx.MiB, 1 * gcsx.MiB}, sp, "Read at 12 MiB must be served from prefetched chunk")
	}
	mu.Unlock()

	t.reader.Destroy()
}

func (t *KernelMRDReaderTest) TestReadAt_ContextCancellationAtMaxPrefetchChunks_RestoresEvictedHeadChunk() {
	t.object.Size = 600 * gcsx.MiB
	t.config = &cfg.Config{Mrd: cfg.MrdConfig{PoolSize: 4}}
	t.mrdInstance = gcsx.NewMrdInstance(t.object, t.bucket, t.cache, t.inodeID, t.config)
	t.reader = NewKernelMRDReader(t.mrdInstance, metrics.NewNoopMetrics())

	pool := mem.DefaultBufferPool()
	finishChunk0 := make(chan struct{})

	mrd := &pinnedFakeMRD{
		addFn: func(output io.Writer, offset, length int64, callback func(int64, int64, error)) {
			go func() {
				if offset == 4*gcsx.MiB {
					<-finishChunk0
				}
				rawBuf := pool.Get(int(length))
				slice := (*rawBuf)[:length]
				for i := range slice {
					slice[i] = 'M'
				}
				n, _ := output.Write(slice)
				pool.Put(rawBuf)
				callback(offset, int64(n), nil)
			}()
		},
	}
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(mrd, nil).Times(4)

	// Perform initial read at [0, 4 MiB) to initialize the MRD pool, wait for all 4 MRDs in the pool to be ready,
	// and ensure 6 prefetch chunks starting at 4 MiB are scheduled.
	initResp, err := t.reader.ReadAt(context.Background(), &gcsx.ReadRequest{
		Buffer: make([]byte, 4*gcsx.MiB),
		Offset: 0,
	})
	assert.NoError(t.T(), err)
	if initResp.Callback != nil {
		initResp.Callback()
	}
	assert.Eventually(t.T(), func() bool {
		return t.mrdInstance.ReadySize() == 4
	}, time.Second, time.Millisecond)

	t.reader.mu.Lock()
	t.reader.schedulePrefetchLocked()
	assert.Len(t.T(), t.reader.chunks, maxPrefetchChunks)
	origChunk0 := t.reader.chunks[0]
	assert.Equal(t.T(), int64(4*gcsx.MiB), origChunk0.Offset)
	t.reader.mu.Unlock()

	// Cancel a full-chunk read [4 MiB, 8 MiB): acquirePrefetchSpans evicts origChunk0 and refills kmr.chunks to maxPrefetchChunks (6),
	// then Wait() returns context.Canceled and rollbackPrefetchSpans must re-insert origChunk0 at index 0 despite len(kmr.chunks) == maxPrefetchChunks.
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()
	resp, err := t.reader.ReadAt(cancelCtx, &gcsx.ReadRequest{
		Buffer: make([]byte, 4*gcsx.MiB),
		Offset: 4 * gcsx.MiB,
	})
	assert.ErrorIs(t.T(), err, context.Canceled)
	assert.Equal(t.T(), 0, resp.Size)

	t.reader.mu.Lock()
	assert.Same(t.T(), origChunk0, t.reader.chunks[0], "Evicted head chunk must be re-inserted at index 0 even when queue was refilled to maxPrefetchChunks")
	assert.Equal(t.T(), 0, origChunk0.Consumed)
	t.reader.mu.Unlock()

	// Unblock origChunk0 and retry [4 MiB, 8 MiB): must hit origChunk0 zero-copy.
	close(finishChunk0)
	retryResp, retryErr := t.reader.ReadAt(context.Background(), &gcsx.ReadRequest{
		Buffer: make([]byte, 4*gcsx.MiB),
		Offset: 4 * gcsx.MiB,
	})
	assert.NoError(t.T(), retryErr)
	assert.Equal(t.T(), 4*gcsx.MiB, retryResp.Size)
	assert.NotNil(t.T(), retryResp.Callback)
	retryResp.Callback()

	t.reader.Destroy()
}

func (t *KernelMRDReaderTest) TestAcquirePrefetchSpans_DuplicateAndOverlappingReads_DoNotPrematurelyEvictChunk() {
	t.object.Size = 16 * gcsx.MiB
	data := bytes.Repeat([]byte("D"), int(t.object.Size))
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, data)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()

	// Seed prefetch queue at offset 1 MiB: chunk[0] is [1 MiB, 3 MiB) (2 MiB).
	t.reader.mu.Lock()
	t.reader.lastReadEnd = 1 * gcsx.MiB
	t.reader.nextPrefetchOffset = 1 * gcsx.MiB
	t.reader.seqReads = 1
	t.reader.schedulePrefetchLocked()
	assert.Len(t.T(), t.reader.chunks, 2)
	chunk0 := t.reader.chunks[0]
	t.reader.mu.Unlock()

	// 1. Read [1 MiB, 2 MiB) first time -> Consumed == 1 MiB.
	c1, _, _ := t.reader.acquirePrefetchChunk(1*gcsx.MiB, 1*gcsx.MiB)
	assert.Same(t.T(), chunk0, c1)
	assert.Equal(t.T(), int(1*gcsx.MiB), chunk0.Consumed)
	c1.Release()

	// 2. Duplicate read [1 MiB, 2 MiB) second time -> Consumed must remain 1 MiB and chunk0 must NOT be evicted.
	c2, _, _ := t.reader.acquirePrefetchChunk(1*gcsx.MiB, 1*gcsx.MiB)
	assert.Same(t.T(), chunk0, c2)
	assert.Equal(t.T(), int(1*gcsx.MiB), chunk0.Consumed, "Duplicate read of same sub-range must not double-count Consumed")
	c2.Release()

	t.reader.mu.Lock()
	assert.Same(t.T(), chunk0, t.reader.chunks[0], "Chunk must still be in queue while [2 MiB, 3 MiB) is unconsumed")
	t.reader.mu.Unlock()

	// 3. Overlapping read [1.5 MiB, 2.5 MiB) -> Consumed union is [1 MiB, 2.5 MiB) = 1.5 MiB, chunk0 still in queue.
	c3, _, _ := t.reader.acquirePrefetchChunk(1*gcsx.MiB+512*1024, 1*gcsx.MiB)
	assert.Same(t.T(), chunk0, c3)
	assert.Equal(t.T(), int(1*gcsx.MiB+512*1024), chunk0.Consumed)
	c3.Release()

	t.reader.mu.Lock()
	assert.Same(t.T(), chunk0, t.reader.chunks[0])
	t.reader.mu.Unlock()

	// 4. Read remaining [2.5 MiB, 3 MiB) -> Consumed reaches 2 MiB and chunk0 is now evicted.
	c4, _, _ := t.reader.acquirePrefetchChunk(2*gcsx.MiB+512*1024, 512*1024)
	assert.Same(t.T(), chunk0, c4)
	assert.Equal(t.T(), int(2*gcsx.MiB), chunk0.Consumed)
	c4.Release()

	t.reader.mu.Lock()
	assert.Equal(t.T(), int64(3*gcsx.MiB), t.reader.chunks[0].Offset, "Chunk0 must be evicted once full [1 MiB, 3 MiB) union is consumed")
	t.reader.mu.Unlock()

	t.reader.Destroy()
}

func (t *KernelMRDReaderTest) TestAcquirePrefetchSpans_OutOfOrderAcrossChunkBoundary_PreservesPartiallyConsumedPreviousChunk() {
	t.object.Size = 16 * gcsx.MiB
	data := bytes.Repeat([]byte("P"), int(t.object.Size))
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, data)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()

	// Seed prefetch queue at offset 1 MiB:
	// chunk[0] = [1 MiB, 3 MiB) (2 MiB), chunk[1] = [3 MiB, 7 MiB) (4 MiB).
	t.reader.mu.Lock()
	t.reader.lastReadEnd = 1 * gcsx.MiB
	t.reader.nextPrefetchOffset = 1 * gcsx.MiB
	t.reader.seqReads = 1
	t.reader.schedulePrefetchLocked()
	assert.Len(t.T(), t.reader.chunks, 2)
	chunk0 := t.reader.chunks[0]
	chunk1 := t.reader.chunks[1]
	t.reader.mu.Unlock()

	// Concurrent FUSE workers arrive out-of-order across the 3 MiB chunk boundary:
	// Worker B ([2 MiB, 3 MiB) in chunk0) and Worker C ([3 MiB, 4 MiB) in chunk1) arrive BEFORE Worker A ([1 MiB, 2 MiB) in chunk0).
	cB, _, _ := t.reader.acquirePrefetchChunk(2*gcsx.MiB, 1*gcsx.MiB)
	assert.Same(t.T(), chunk0, cB)
	cB.Release()

	cC, _, _ := t.reader.acquirePrefetchChunk(3*gcsx.MiB, 1*gcsx.MiB)
	assert.Same(t.T(), chunk1, cC)
	cC.Release()

	// Chunk 0 ([1 MiB, 3 MiB)) is only half-consumed (1 MiB of 2 MiB) and must NOT be evicted by Worker C ([3 MiB, 4 MiB)).
	t.reader.mu.Lock()
	assert.Same(t.T(), chunk0, t.reader.chunks[0], "Partially-consumed chunk0 must not be evicted when adjacent chunk1 is accessed out-of-order")
	t.reader.mu.Unlock()

	// Now Worker A ([1 MiB, 2 MiB)) arrives: must hit chunk0, finish consuming chunk0 (2 MiB / 2 MiB), and evict chunk0 without clearing the queue!
	cA, _, _ := t.reader.acquirePrefetchChunk(1*gcsx.MiB, 1*gcsx.MiB)
	assert.Same(t.T(), chunk0, cA, "Late-arriving Worker A must hit chunk0 instead of seeing startIdx==-1 and wiping the prefetch queue")
	cA.Release()

	t.reader.mu.Lock()
	assert.NotEmpty(t.T(), t.reader.chunks)
	assert.Same(t.T(), chunk1, t.reader.chunks[0], "Chunk0 should now be evicted and chunk1 should be at the head of the queue")
	t.reader.mu.Unlock()

	t.reader.Destroy()
}

func (t *KernelMRDReaderTest) TestReadAt_PrefetchHitWhileEarlierContiguousReadVectoredInFlight_PreservesPrefetchQueue() {
	t.object.Size = 16 * gcsx.MiB
	pool := mem.DefaultBufferPool()

	entered0 := make(chan struct{})
	entered1 := make(chan struct{})
	release0 := make(chan struct{})
	release1 := make(chan struct{})

	mrd := &pinnedFakeMRD{
		addFn: func(output io.Writer, offset, length int64, callback func(int64, int64, error)) {
			go func() {
				switch {
				case offset == 0 && length == 1*gcsx.MiB:
					close(entered0)
					<-release0
				case offset == 1*gcsx.MiB && length == 1*gcsx.MiB:
					close(entered1)
					<-release1
				}
				rawBuf := pool.Get(int(length))
				slice := (*rawBuf)[:length]
				fillByte := byte('A' + (offset / gcsx.MiB))
				for i := range slice {
					slice[i] = fillByte
				}
				n, _ := output.Write(slice)
				pool.Put(rawBuf)
				callback(offset, int64(n), nil)
			}()
		},
	}
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(mrd, nil).Once()

	var (
		resp0, resp1 gcsx.ReadResponse
		err0, err1   error
		done0        = make(chan struct{})
		done1        = make(chan struct{})
	)

	go func() {
		defer close(done0)
		resp0, err0 = t.reader.ReadAt(context.Background(), &gcsx.ReadRequest{
			Buffer: make([]byte, 1*gcsx.MiB),
			Offset: 0,
		})
	}()

	go func() {
		defer close(done1)
		resp1, err1 = t.reader.ReadAt(context.Background(), &gcsx.ReadRequest{
			Buffer: make([]byte, 1*gcsx.MiB),
			Offset: 1 * gcsx.MiB,
		})
	}()

	// Wait until both [0, 1 MiB) and [1 MiB, 2 MiB) are in-flight in ReadVectored.
	<-entered0
	<-entered1

	// 1. Let [0, 1 MiB) finish ReadVectored first; it schedules prefetch starting at 2 MiB (past in-flight [1 MiB, 2 MiB)).
	close(release0)
	<-done0
	assert.NoError(t.T(), err0)
	assert.Equal(t.T(), 1*gcsx.MiB, resp0.Size)
	if resp0.Callback != nil {
		resp0.Callback()
	}

	// 2. Before [1 MiB, 2 MiB) finishes ReadVectored, read [2 MiB, 3 MiB) which hits the prefetched chunk at 2 MiB.
	resp2, err2 := t.reader.ReadAt(context.Background(), &gcsx.ReadRequest{
		Buffer: make([]byte, 1*gcsx.MiB),
		Offset: 2 * gcsx.MiB,
	})
	assert.NoError(t.T(), err2)
	assert.Equal(t.T(), 1*gcsx.MiB, resp2.Size)
	assert.NotNil(t.T(), resp2.Callback, "Read [2 MiB, 3 MiB) must hit prefetched chunk zero-copy")
	resp2.Callback()

	t.reader.mu.Lock()
	assert.Equal(t.T(), int64(3*gcsx.MiB), t.reader.lastReadEnd)
	t.reader.mu.Unlock()

	// 3. Now let [1 MiB, 2 MiB) complete ReadVectored. It must recognize that [2 MiB, 3 MiB) already bridged across it
	//    and must NOT clear kmr.chunks or rewind kmr.lastReadEnd.
	close(release1)
	<-done1
	assert.NoError(t.T(), err1)
	assert.Equal(t.T(), 1*gcsx.MiB, resp1.Size)
	if resp1.Callback != nil {
		resp1.Callback()
	}

	t.reader.mu.Lock()
	assert.Empty(t.T(), t.reader.inFlight)
	assert.Equal(t.T(), int64(3*gcsx.MiB), t.reader.lastReadEnd, "Completing earlier in-flight read must not rewind lastReadEnd")
	assert.NotEmpty(t.T(), t.reader.chunks, "Completing earlier in-flight read must not wipe active prefetch queue")
	t.reader.mu.Unlock()

	// 4. Subsequent sequential read [3 MiB, 4 MiB) must still hit the remaining prefetched chunk zero-copy.
	resp3, err3 := t.reader.ReadAt(context.Background(), &gcsx.ReadRequest{
		Buffer: make([]byte, 1*gcsx.MiB),
		Offset: 3 * gcsx.MiB,
	})
	assert.NoError(t.T(), err3)
	assert.Equal(t.T(), 1*gcsx.MiB, resp3.Size)
	assert.NotNil(t.T(), resp3.Callback)
	resp3.Callback()

	t.reader.Destroy()
}

func (t *KernelMRDReaderTest) TestReadAt_ContextCancellationDuringWait_AfterDestroyOrSeek_DoesNotReinsertStaleChunks() {
	t.Run("DestroyBeforeRollback", func() {
		t.SetupTest()
		t.object.Size = 16 * gcsx.MiB
		data := bytes.Repeat([]byte("X"), int(t.object.Size))
		fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, data)
		t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()

		t.reader.mu.Lock()
		t.reader.lastReadEnd = 1 * gcsx.MiB
		t.reader.nextPrefetchOffset = 1 * gcsx.MiB
		t.reader.seqReads = 1
		t.reader.schedulePrefetchLocked()
		assert.Len(t.T(), t.reader.chunks, 2)
		t.reader.mu.Unlock()

		spans := t.reader.acquirePrefetchSpans(1*gcsx.MiB, 2*gcsx.MiB)
		assert.Len(t.T(), spans, 1)

		t.reader.Destroy()
		t.reader.rollbackPrefetchSpans(spans)

		t.reader.mu.Lock()
		assert.Empty(t.T(), t.reader.chunks, "rollbackPrefetchSpans must not re-insert chunks after Destroy()")
		t.reader.mu.Unlock()
	})

	t.Run("ConcurrentClearPrefetchDuringWait", func() {
		t.SetupTest()
		t.object.Size = 16 * gcsx.MiB
		pool := mem.DefaultBufferPool()

		chunk0Started := make(chan struct{})
		finishChunk0 := make(chan struct{})
		mrd := &pinnedFakeMRD{
			addFn: func(output io.Writer, offset, length int64, callback func(int64, int64, error)) {
				go func() {
					if offset == 1*gcsx.MiB {
						close(chunk0Started)
						<-finishChunk0
					}
					rawBuf := pool.Get(int(length))
					slice := (*rawBuf)[:length]
					n, _ := output.Write(slice)
					pool.Put(rawBuf)
					callback(offset, int64(n), nil)
				}()
			},
		}
		t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(mrd, nil).Once()

		t.reader.mu.Lock()
		t.reader.lastReadEnd = 1 * gcsx.MiB
		t.reader.nextPrefetchOffset = 1 * gcsx.MiB
		t.reader.seqReads = 1
		t.reader.schedulePrefetchLocked()
		assert.Len(t.T(), t.reader.chunks, 2)
		t.reader.mu.Unlock()
		<-chunk0Started

		ctx, cancel := context.WithCancel(context.Background())
		readDone := make(chan error, 1)
		go func() {
			_, err := t.reader.ReadAt(ctx, &gcsx.ReadRequest{
				Buffer: make([]byte, 2*gcsx.MiB),
				Offset: 1 * gcsx.MiB,
			})
			readDone <- err
		}()

		// Wait until ReadAt has acquired & evicted chunk[0] [1 MiB, 3 MiB) and is blocked in chunk.Wait().
		assert.Eventually(t.T(), func() bool {
			t.reader.mu.Lock()
			defer t.reader.mu.Unlock()
			return len(t.reader.chunks) > 0 && t.reader.chunks[0].Offset == 3*gcsx.MiB
		}, time.Second, time.Millisecond)

		// Concurrently clear prefetch while Wait() is blocked, then cancel ctx.
		t.reader.clearPrefetch()
		cancel()
		close(finishChunk0)

		err := <-readDone
		assert.ErrorIs(t.T(), err, context.Canceled)

		t.reader.mu.Lock()
		assert.Empty(t.T(), t.reader.chunks, "rollbackPrefetchSpans must not re-insert chunks after clearPrefetch()")
		t.reader.mu.Unlock()

		t.reader.Destroy()
	})

	t.Run("ConcurrentSeekDuringWait", func() {
		t.SetupTest()
		t.object.Size = 32 * gcsx.MiB
		pool := mem.DefaultBufferPool()

		chunk0Started := make(chan struct{})
		finishChunk0 := make(chan struct{})
		mrd := &pinnedFakeMRD{
			addFn: func(output io.Writer, offset, length int64, callback func(int64, int64, error)) {
				go func() {
					if offset == 1*gcsx.MiB {
						close(chunk0Started)
						<-finishChunk0
					}
					rawBuf := pool.Get(int(length))
					slice := (*rawBuf)[:length]
					n, _ := output.Write(slice)
					pool.Put(rawBuf)
					callback(offset, int64(n), nil)
				}()
			},
		}
		t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(mrd, nil).Once()

		t.reader.mu.Lock()
		t.reader.lastReadEnd = 1 * gcsx.MiB
		t.reader.nextPrefetchOffset = 1 * gcsx.MiB
		t.reader.seqReads = 1
		t.reader.schedulePrefetchLocked()
		t.reader.mu.Unlock()
		<-chunk0Started

		ctx, cancel := context.WithCancel(context.Background())
		readDone := make(chan error, 1)
		go func() {
			_, err := t.reader.ReadAt(ctx, &gcsx.ReadRequest{
				Buffer: make([]byte, 2*gcsx.MiB),
				Offset: 1 * gcsx.MiB,
			})
			readDone <- err
		}()

		assert.Eventually(t.T(), func() bool {
			t.reader.mu.Lock()
			defer t.reader.mu.Unlock()
			return len(t.reader.chunks) > 0 && t.reader.chunks[0].Offset == 3*gcsx.MiB
		}, time.Second, time.Millisecond)

		// Simulate a concurrent stream seek to 20 MiB that resets prefetch to [20 MiB, ...).
		t.reader.mu.Lock()
		t.reader.clearPrefetchLocked()
		t.reader.lastReadEnd = 20 * gcsx.MiB
		t.reader.nextPrefetchOffset = 20 * gcsx.MiB
		t.reader.seqReads = 2
		t.reader.schedulePrefetchLocked()
		assert.NotEmpty(t.T(), t.reader.chunks)
		assert.Equal(t.T(), int64(20*gcsx.MiB), t.reader.chunks[0].Offset)
		t.reader.mu.Unlock()

		cancel()
		close(finishChunk0)

		err := <-readDone
		assert.ErrorIs(t.T(), err, context.Canceled)

		t.reader.mu.Lock()
		assert.NotEmpty(t.T(), t.reader.chunks)
		assert.Equal(t.T(), int64(20*gcsx.MiB), t.reader.chunks[0].Offset, "Disjoint stale chunk [1 MiB, 3 MiB) must not be re-inserted into [20 MiB, ...) queue")
		t.reader.mu.Unlock()

		t.reader.Destroy()
	})
}

func (t *KernelMRDReaderTest) TestReadAt_SequentialReadPastEOF_HitsPrefetchedTailChunkZeroCopy() {
	t.object.Size = 2*gcsx.MiB + 512*1024 // 2.5 MiB
	pool := mem.DefaultBufferPool()

	var (
		mu             sync.Mutex
		requestedSpans [][2]int64
	)

	mrd := &pinnedFakeMRD{
		addFn: func(output io.Writer, offset, length int64, callback func(int64, int64, error)) {
			mu.Lock()
			requestedSpans = append(requestedSpans, [2]int64{offset, length})
			mu.Unlock()

			rawBuf := pool.Get(int(length))
			slice := (*rawBuf)[:length]
			for i := range slice {
				slice[i] = byte('A' + ((offset + int64(i)) / (512 * 1024)))
			}
			n, _ := output.Write(slice)
			pool.Put(rawBuf)
			callback(offset, int64(n), nil)
		},
	}
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(mrd, nil).Once()

	// 1. Read [0, 1 MiB) -> triggers prefetch of tail chunk [1 MiB, 2.5 MiB) (1.5 MiB).
	resp0, err := t.reader.ReadAt(context.Background(), &gcsx.ReadRequest{
		Buffer: make([]byte, 1*gcsx.MiB),
		Offset: 0,
	})
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 1*gcsx.MiB, resp0.Size)
	if resp0.Callback != nil {
		resp0.Callback()
	}

	// 2. Read [1 MiB, 2 MiB) -> served zero-copy from prefetched [1 MiB, 2.5 MiB) chunk.
	resp1, err := t.reader.ReadAt(context.Background(), &gcsx.ReadRequest{
		Buffer: make([]byte, 1*gcsx.MiB),
		Offset: 1 * gcsx.MiB,
	})
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 1*gcsx.MiB, resp1.Size)
	assert.NotNil(t.T(), resp1.Callback)
	resp1.Callback()

	// 3. Final 1 MiB read at offset 2 MiB extends past objSize (2.5 MiB).
	//    Must hit the remaining [2 MiB, 2.5 MiB) slice of the prefetched tail chunk zero-copy without falling back to ReadVectored.
	resp2, err := t.reader.ReadAt(context.Background(), &gcsx.ReadRequest{
		Buffer: make([]byte, 1*gcsx.MiB),
		Offset: 2 * gcsx.MiB,
	})
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 512*1024, resp2.Size)
	assert.NotNil(t.T(), resp2.Callback, "EOF-straddling sequential read must hit prefetched tail chunk zero-copy")
	var got bytes.Buffer
	for _, s := range resp2.Data {
		got.Write(s)
	}
	assert.Equal(t.T(), 512*1024, got.Len())
	assert.Equal(t.T(), byte('A'+4), got.Bytes()[0])
	resp2.Callback()

	mu.Lock()
	for _, sp := range requestedSpans {
		assert.NotEqual(t.T(), int64(2*gcsx.MiB), sp[0], "EOF tail read must not trigger a fallback ReadVectored call at 2 MiB")
	}
	mu.Unlock()

	t.reader.Destroy()
}

func (t *KernelMRDReaderTest) TestReadAt_ConcurrentFirstReads_IncrementRefCountOnceBeforeReadsProceed() {
	data := bytes.Repeat([]byte("R"), 64*1024)
	mrd := &pinnedFakeMRD{
		addFn: func(output io.Writer, offset, length int64, callback func(int64, int64, error)) {
			assert.Equal(t.T(), int64(1), t.mrdInstance.RefCount(), "IncrementRefCount must complete before any concurrent reader enters MRD")
			assert.True(t.T(), t.reader.mrdInstanceInUse.Load())
			n, _ := output.Write(data[offset : offset+length])
			callback(offset, int64(n), nil)
		},
	}
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(mrd, nil).Once()

	const numReaders = 16
	startBarrier := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(numReaders)
	for i := range numReaders {
		go func(idx int) {
			defer wg.Done()
			<-startBarrier
			buf := make([]byte, 1024)
			resp, err := t.reader.ReadAt(context.Background(), &gcsx.ReadRequest{
				Buffer: buf,
				Offset: int64(idx * 1024),
			})
			assert.NoError(t.T(), err)
			assert.Equal(t.T(), 1024, resp.Size)
			if resp.Callback != nil {
				resp.Callback()
			}
		}(i)
	}
	close(startBarrier)
	wg.Wait()

	assert.Equal(t.T(), int64(1), t.mrdInstance.RefCount())
	t.reader.Destroy()
	assert.Equal(t.T(), int64(0), t.mrdInstance.RefCount())
	assert.False(t.T(), t.reader.mrdInstanceInUse.Load())
}
