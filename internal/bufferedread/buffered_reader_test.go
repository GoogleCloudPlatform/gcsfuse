// Copyright 2025 Google LLC
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

package bufferedread

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/block"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/workerpool"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/stretchr/testify/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/semaphore"
)

const (
	testMaxPrefetchBlockCnt     int64 = 10
	testMinBlocksPerHandle      int64 = 2
	testRandomSeekThreshold     int64 = 3
	testGlobalMaxBlocks         int64 = 20
	testPrefetchBlockSizeBytes  int64 = 1024
	testInitialPrefetchBlockCnt int64 = 2
	oneTB                       int64 = 1 << 40
)

type BufferedReaderTest struct {
	suite.Suite
	ctx                context.Context
	object             *gcs.MinObject
	bucket             *storage.TestifyMockBucket
	globalMaxBlocksSem *semaphore.Weighted
	config             *BufferedReadConfig
	workerPool         workerpool.WorkerPool
	metricHandle       metrics.MetricHandle
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// createFakeReaderWithOffset returns a FakeReader with deterministic, non-zero content
// starting from a specific absolute offset.
func createFakeReaderWithOffset(t *testing.T, size int, startOffset int64) *fake.FakeReader {
	t.Helper()
	content := make([]byte, size)
	for i := range content {
		content[i] = byte('A' + ((int(startOffset) + i) % 26)) // A-Z repeating pattern
	}
	return &fake.FakeReader{
		ReadCloser: io.NopCloser(bytes.NewReader(content)),
	}
}

// assertBlockContent validates that block data matches expected pattern (A-Z loop).
func assertBlockContent(t *testing.T, blk block.PrefetchBlock, expectedOffset int64, length int) {
	t.Helper()
	buf := make([]byte, length)
	n, err := blk.ReadAt(buf, 0)
	require.NoError(t, err)
	require.Equal(t, length, n)
	assertBufferContent(t, buf, expectedOffset)
}

// assertBufferContent validates that a buffer's data matches the expected A-Z repeating pattern
// for a given absolute starting offset.
func assertBufferContent(t *testing.T, buf []byte, absStartOffset int64) {
	t.Helper()
	for i := 0; i < len(buf); i++ {
		expected := byte('A' + ((int(absStartOffset) + i) % 26))
		assert.Equalf(t, expected, buf[i], "Mismatch at buffer index %d (absolute offset %d)", i, absStartOffset+int64(i))
	}
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func TestBufferedReaderTestSuite(t *testing.T) {
	suite.Run(t, new(BufferedReaderTest))
}

func (t *BufferedReaderTest) SetupTest() {
	t.object = &gcs.MinObject{
		Name:       "test_object",
		Size:       8192,
		Generation: 1234567890,
	}
	t.bucket = new(storage.TestifyMockBucket)
	t.globalMaxBlocksSem = semaphore.NewWeighted(testGlobalMaxBlocks)
	t.config = &BufferedReadConfig{
		MaxPrefetchBlockCnt:     testMaxPrefetchBlockCnt,
		PrefetchBlockSizeBytes:  testPrefetchBlockSizeBytes,
		InitialPrefetchBlockCnt: testInitialPrefetchBlockCnt,
		MinBlocksPerHandle:      testMinBlocksPerHandle,
		RandomSeekThreshold:     testRandomSeekThreshold,
	}
	var err error
	t.workerPool, err = workerpool.NewStaticWorkerPool(5, 10, 15)
	require.NoError(t.T(), err, "Failed to create worker pool")
	t.workerPool.Start()
	t.metricHandle = metrics.NewNoopMetrics()
	t.ctx = context.Background()
}

func (t *BufferedReaderTest) TearDownTest() {
	t.workerPool.Stop()
}

func (t *BufferedReaderTest) TestNewBufferedReader() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err, "NewBufferedReader should not return error")

	assert.Equal(t.T(), t.object, reader.object, "object should match")
	assert.Equal(t.T(), t.bucket, reader.bucket, "bucket should match")
	assert.Equal(t.T(), t.config, reader.config, "config should match")
	assert.Equal(t.T(), int64(0), reader.nextBlockIndexToPrefetch, "nextBlockIndexToPrefetch should be 0")
	assert.Equal(t.T(), int64(0), reader.randomSeekCount, "randomSeekCount should be 0")
	assert.Equal(t.T(), testInitialPrefetchBlockCnt, reader.numPrefetchBlocks, "numPrefetchBlocks should match")
	assert.NotNil(t.T(), reader.blockQueue, "blockQueue should not be nil")
	assert.NotNil(t.T(), reader.blockPool, "blockPool should have been created")
	assert.Equal(t.T(), t.workerPool, reader.workerPool)
	assert.Equal(t.T(), t.metricHandle, reader.metricHandle)
	assert.NotNil(t.T(), reader.ctx)
	assert.NotNil(t.T(), reader.cancelFunc)
}

func (t *BufferedReaderTest) TestNewBufferedReaderReservesRequiredBlocks() {
	testCases := []struct {
		name               string
		objectSize         uint64
		minBlocksPerHandle int64
		expectedReserved   int64
	}{
		{
			name:               "SmallFile",
			objectSize:         uint64(testPrefetchBlockSizeBytes) / 2, // Requires 1 block
			minBlocksPerHandle: 5,
			expectedReserved:   1,
		},
		{
			name:               "LargeFile",
			objectSize:         uint64(testPrefetchBlockSizeBytes) * 10, // Requires 10 blocks
			minBlocksPerHandle: 5,
			expectedReserved:   5,
		},
		{
			name:               "ZeroSizeFile",
			objectSize:         0, // Requires 0 blocks
			minBlocksPerHandle: 5,
			expectedReserved:   0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.object.Size = tc.objectSize
			t.config.MinBlocksPerHandle = tc.minBlocksPerHandle
			t.globalMaxBlocksSem = semaphore.NewWeighted(testGlobalMaxBlocks)

			reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)

			require.NoError(t.T(), err)
			require.NotNil(t.T(), reader)
			// Verify that the correct number of blocks were reserved by checking the semaphore's state.
			assert.True(t.T(), t.globalMaxBlocksSem.TryAcquire(testGlobalMaxBlocks-tc.expectedReserved), "Should acquire remaining permits")
			assert.False(t.T(), t.globalMaxBlocksSem.TryAcquire(1), "Should not acquire more permits")
		})
	}
}

func (t *BufferedReaderTest) TestNewBufferedReaderFailsWhenPoolAllocationFails() {
	t.globalMaxBlocksSem = semaphore.NewWeighted(1)

	_, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)

	require.Error(t.T(), err)
	assert.ErrorIs(t.T(), err, block.CantAllocateAnyBlockError)
}

func (t *BufferedReaderTest) TestNewBufferedReaderWithMinimumBlockNotAvailableInPool() {
	// Simulate no blocks available globally.
	t.globalMaxBlocksSem = semaphore.NewWeighted(1)

	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)

	assert.Error(t.T(), err)
	assert.ErrorIs(t.T(), err, block.CantAllocateAnyBlockError)
	assert.Nil(t.T(), reader, "BufferedReader should be nil on error")
}

func (t *BufferedReaderTest) TestNewBufferedReaderWithZeroBlockSize() {
	t.config.PrefetchBlockSizeBytes = 0

	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)

	assert.ErrorContains(t.T(), err, "PrefetchBlockSizeBytes must be positive")
	assert.Nil(t.T(), reader, "BufferedReader should be nil on error")
}

func (t *BufferedReaderTest) TestDestroySuccess() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err, "NewBufferedReader should not return error")
	b, err := reader.blockPool.Get()
	require.NoError(t.T(), err, "Failed to get block from pool")
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-ctx.Done()
		b.NotifyReady(block.BlockStatus{State: block.BlockStateDownloadFailed, Err: context.Canceled})
	}()
	reader.blockQueue.Push(&blockQueueEntry{
		block:  b,
		cancel: cancel,
	})

	reader.Destroy()

	assert.Nil(t.T(), reader.cancelFunc)
	assert.True(t.T(), reader.blockQueue.IsEmpty())
	assert.Nil(t.T(), reader.blockPool)
}

func (t *BufferedReaderTest) TestDestroyAwaitReadyError() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err, "NewBufferedReader should not return error")
	b, err := reader.blockPool.Get()
	require.NoError(t.T(), err, "Failed to get block from pool")
	reader.blockQueue.Push(&blockQueueEntry{
		block:  b,
		cancel: func() {},
	})

	b.NotifyReady(block.BlockStatus{State: block.BlockStateDownloadFailed, Err: errors.New("test error")})
	reader.Destroy()

	assert.Nil(t.T(), reader.cancelFunc)
	assert.True(t.T(), reader.blockQueue.IsEmpty(), "blockQueue should be empty after Destroy")
	assert.Nil(t.T(), reader.blockPool)
}

func (t *BufferedReaderTest) TestCheckInvariantsBlockQueueExceedsLimit() {
	t.config.MaxPrefetchBlockCnt = 2
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err, "NewBufferedReader should not return error")
	b, err := reader.blockPool.Get()
	require.NoError(t.T(), err, "Failed to get block from pool")

	// Push 3 blocks to exceed the limit of 2.
	reader.blockQueue.Push(&blockQueueEntry{block: b, cancel: func() {}})
	reader.blockQueue.Push(&blockQueueEntry{block: b, cancel: func() {}})
	reader.blockQueue.Push(&blockQueueEntry{block: b, cancel: func() {}})

	assert.Panics(t.T(), func() { reader.CheckInvariants() })
}

func (t *BufferedReaderTest) TestCheckInvariantsRandomSeekCountExceedsThreshold() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err, "NewBufferedReader should not return error")

	reader.randomSeekCount = reader.randomReadsThreshold + 1

	assert.Panics(t.T(), func() { reader.CheckInvariants() })
}

func (t *BufferedReaderTest) TestCheckInvariantsPrefetchBlockSizeNotPositive() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err, "NewBufferedReader should not return error")
	testCases := []struct {
		name      string
		blockSize int64
	}{
		{
			name:      "zero block size",
			blockSize: 0,
		},
		{
			name:      "negative block size",
			blockSize: -1,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func() {
			reader.config.PrefetchBlockSizeBytes = tc.blockSize

			assert.Panics(t.T(), func() { reader.CheckInvariants() }, "Should panic for non-positive block size")
		})
	}
}

func (t *BufferedReaderTest) TestCheckInvariantsPrefetchBlockSizeTooSmall() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err, "NewBufferedReader should not return error")

	reader.config.PrefetchBlockSizeBytes = util.MiB - 1

	assert.Panics(t.T(), func() { reader.CheckInvariants() }, "Should panic for block size less than 1 MiB")
}

func (t *BufferedReaderTest) TestCheckInvariantsNoPanic() {
	t.config.PrefetchBlockSizeBytes = util.MiB
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err, "NewBufferedReader should not return error")

	assert.NotPanics(t.T(), func() { reader.CheckInvariants() })
}

func (t *BufferedReaderTest) TestScheduleNextBlock() {
	testCases := []struct {
		name   string
		urgent bool
	}{
		{name: "non-urgent", urgent: false},
		{name: "urgent", urgent: true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func() {
			reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
			require.NoError(t.T(), err)
			initialBlockCount := reader.blockQueue.Len()
			startOffset := int64(0)
			t.bucket.On("NewReaderWithReadHandle",
				mock.Anything,
				mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == uint64(startOffset) }),
			).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), startOffset), nil).Once()

			err = reader.scheduleNextBlock(tc.urgent)

			require.NoError(t.T(), err)
			bqe := reader.blockQueue.Peek()
			assert.Equal(t.T(), int64(1), reader.nextBlockIndexToPrefetch)
			status, err := bqe.block.AwaitReady(t.ctx)
			require.NoError(t.T(), err)
			assert.Equal(t.T(), block.BlockStateDownloaded, status.State)
			assert.Equal(t.T(), initialBlockCount+1, reader.blockQueue.Len())
			assert.Equal(t.T(), int64(0), bqe.block.AbsStartOff())
			assertBlockContent(t.T(), bqe.block, bqe.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
			t.bucket.AssertExpectations(t.T())
		})
	}
}

func (t *BufferedReaderTest) TestScheduleNextBlockSuccessive() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	initialBlockCount := reader.blockQueue.Len()
	startOffset1 := int64(0)
	t.bucket.On("NewReaderWithReadHandle",
		mock.Anything,
		mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == uint64(startOffset1) }),
	).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), startOffset1), nil).Once()
	err = reader.scheduleNextBlock(false)
	require.NoError(t.T(), err)
	bqe1 := reader.blockQueue.Pop()
	assert.Equal(t.T(), int64(1), reader.nextBlockIndexToPrefetch)
	status1, err := bqe1.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), block.BlockStateDownloaded, status1.State)
	assert.Equal(t.T(), int64(0), bqe1.block.AbsStartOff())
	assertBlockContent(t.T(), bqe1.block, bqe1.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	startOffset2 := int64(testPrefetchBlockSizeBytes)
	t.bucket.On("NewReaderWithReadHandle",
		mock.Anything,
		mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == uint64(startOffset2) }),
	).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), startOffset2), nil).Once()

	err = reader.scheduleNextBlock(false)

	require.NoError(t.T(), err)
	bqe2 := reader.blockQueue.Pop()
	assert.Equal(t.T(), int64(2), reader.nextBlockIndexToPrefetch)
	status2, err := bqe2.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), block.BlockStateDownloaded, status2.State)
	assert.Equal(t.T(), int64(testPrefetchBlockSizeBytes), bqe2.block.AbsStartOff())
	assert.Equal(t.T(), int64(2), reader.nextBlockIndexToPrefetch)
	assert.Equal(t.T(), initialBlockCount, reader.blockQueue.Len())
	assertBlockContent(t.T(), bqe2.block, bqe2.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestScheduleBlockWithIndex() {
	testCases := []struct {
		name       string
		urgent     bool
		blockIndex int64
	}{
		{name: "non-urgent", urgent: false, blockIndex: 5},
		{name: "urgent", urgent: true, blockIndex: 3},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func() {
			reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
			require.NoError(t.T(), err)
			initialBlockCount := reader.blockQueue.Len()
			startOffset := tc.blockIndex * reader.config.PrefetchBlockSizeBytes
			t.bucket.On("NewReaderWithReadHandle",
				mock.Anything,
				mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == uint64(startOffset) }),
			).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), startOffset), nil).Once()
			b, err := reader.blockPool.Get()
			require.NoError(t.T(), err)

			err = reader.scheduleBlockWithIndex(b, tc.blockIndex, tc.urgent)

			require.NoError(t.T(), err)
			bqe := reader.blockQueue.Peek()
			status, err := bqe.block.AwaitReady(t.ctx)
			require.NoError(t.T(), err)
			assert.Equal(t.T(), block.BlockStateDownloaded, status.State)
			assert.Equal(t.T(), initialBlockCount+1, reader.blockQueue.Len())
			assert.Equal(t.T(), startOffset, bqe.block.AbsStartOff())
			assertBlockContent(t.T(), bqe.block, bqe.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
			t.bucket.AssertExpectations(t.T())
		})
	}
}

func (t *BufferedReaderTest) TestFreshStart() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	currentOffset := int64(2048) // Start prefetching from offset 2048 (block 2).
	// freshStart schedules 1 urgent block and 2 initial prefetch blocks, totaling 3 blocks.
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 2048 })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 2048), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 3072 })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 3072), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 4096 })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 4096), nil).Once()

	err = reader.freshStart(currentOffset)

	require.NoError(t.T(), err)
	// nextBlockIndexToPrefetch should be current block index (2) + scheduled blocks (3).
	assert.Equal(t.T(), int64(5), reader.nextBlockIndexToPrefetch)
	// numPrefetchBlocks for the next prefetch should be initialPrefetchBlockCnt (2) * prefetchMultiplier (2).
	assert.Equal(t.T(), int64(4), reader.numPrefetchBlocks)
	assert.Equal(t.T(), 3, reader.blockQueue.Len())
	// Pop and verify the downloaded blocks.
	bqe1 := reader.blockQueue.Pop()
	assert.Equal(t.T(), int64(2048), bqe1.block.AbsStartOff())
	status1, err1 := bqe1.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err1)
	assert.Equal(t.T(), block.BlockStateDownloaded, status1.State)
	assertBlockContent(t.T(), bqe1.block, bqe1.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	bqe2 := reader.blockQueue.Pop()
	assert.Equal(t.T(), int64(3072), bqe2.block.AbsStartOff())
	status2, err2 := bqe2.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err2)
	assert.Equal(t.T(), block.BlockStateDownloaded, status2.State)
	assertBlockContent(t.T(), bqe2.block, bqe2.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	bqe3 := reader.blockQueue.Pop()
	assert.Equal(t.T(), int64(4096), bqe3.block.AbsStartOff())
	status3, err3 := bqe3.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err3)
	assert.Equal(t.T(), block.BlockStateDownloaded, status3.State)
	assertBlockContent(t.T(), bqe3.block, bqe3.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestFreshStartWithNonBlockAlignedOffset() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	currentOffset := int64(2500) // Start prefetching from offset 2500 (inside block 2).
	// freshStart should start prefetching from block 2. It schedules 1 urgent block
	// and 2 initial prefetch blocks, totaling 3 blocks.
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 2048 })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 2048), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 3072 })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 3072), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 4096 })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 4096), nil).Once()

	err = reader.freshStart(currentOffset)

	require.NoError(t.T(), err)
	// nextBlockIndexToPrefetch should be current block index (2) + scheduled blocks (3).
	assert.Equal(t.T(), int64(5), reader.nextBlockIndexToPrefetch)
	// numPrefetchBlocks for the next prefetch should be initialPrefetchBlockCnt (2) * prefetchMultiplier (2).
	assert.Equal(t.T(), int64(4), reader.numPrefetchBlocks)
	assert.Equal(t.T(), 3, reader.blockQueue.Len())
	// Pop and verify the downloaded blocks.
	bqe1 := reader.blockQueue.Pop()
	assert.Equal(t.T(), int64(2048), bqe1.block.AbsStartOff())
	status1, err1 := bqe1.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err1)
	assert.Equal(t.T(), block.BlockStateDownloaded, status1.State)
	assertBlockContent(t.T(), bqe1.block, bqe1.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	bqe2 := reader.blockQueue.Pop()
	assert.Equal(t.T(), int64(3072), bqe2.block.AbsStartOff())
	status2, err2 := bqe2.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err2)
	assert.Equal(t.T(), block.BlockStateDownloaded, status2.State)
	assertBlockContent(t.T(), bqe2.block, bqe2.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	bqe3 := reader.blockQueue.Pop()
	assert.Equal(t.T(), int64(4096), bqe3.block.AbsStartOff())
	status3, err3 := bqe3.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err3)
	assert.Equal(t.T(), block.BlockStateDownloaded, status3.State)
	assertBlockContent(t.T(), bqe3.block, bqe3.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestFreshStartWhenInitialCountGreaterThanMax() {
	t.config.MaxPrefetchBlockCnt = 3
	t.config.InitialPrefetchBlockCnt = 4
	t.object.Size = 4096
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	// freshStart schedules 1 urgent block and 2 prefetch blocks (InitialPrefetchBlockCnt capped by MaxPrefetchBlockCnt).
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 0 })).Return(createFakeReaderWithOffset(t.T(), 1024, 0), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 1024 })).Return(createFakeReaderWithOffset(t.T(), 1024, 1024), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 2048 })).Return(createFakeReaderWithOffset(t.T(), 1024, 2048), nil).Once()

	err = reader.freshStart(0)

	require.NoError(t.T(), err)
	// nextBlockIndexToPrefetch should be start block index (0) + scheduled blocks (3).
	assert.Equal(t.T(), int64(3), reader.nextBlockIndexToPrefetch)
	// numPrefetchBlocks for next prefetch should be capped at MaxPrefetchBlockCnt (3).
	assert.Equal(t.T(), int64(3), reader.numPrefetchBlocks)
	assert.Equal(t.T(), 3, reader.blockQueue.Len())
	// Pop and verify blocks are downloaded.
	bqe1 := reader.blockQueue.Pop()
	status1, err1 := bqe1.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err1)
	assert.Equal(t.T(), block.BlockStateDownloaded, status1.State)
	assertBlockContent(t.T(), bqe1.block, bqe1.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	bqe2 := reader.blockQueue.Pop()
	status2, err2 := bqe2.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err2)
	assert.Equal(t.T(), block.BlockStateDownloaded, status2.State)
	assertBlockContent(t.T(), bqe2.block, bqe2.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	bqe3 := reader.blockQueue.Pop()
	status3, err3 := bqe3.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err3)
	assert.Equal(t.T(), block.BlockStateDownloaded, status3.State)
	assertBlockContent(t.T(), bqe3.block, bqe3.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestFreshStartStopsAtObjectEnd() {
	t.object.Size = 4000 // Object size is 3 blocks + a partial block.
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	currentOffset := int64(2048) // Start from block 2.
	// freshStart schedules 1 urgent block (block 2) and 1 prefetch block (block 3 - partial).
	// The object ends after block 3, so only these 2 blocks are scheduled.
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 2*uint64(testPrefetchBlockSizeBytes) })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 2*testPrefetchBlockSizeBytes), nil).Once()
	partialBlockSize := int(int64(t.object.Size) - (3 * testPrefetchBlockSizeBytes))
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 3*uint64(testPrefetchBlockSizeBytes) })).Return(createFakeReaderWithOffset(t.T(), partialBlockSize, 3*testPrefetchBlockSizeBytes), nil).Once()

	err = reader.freshStart(currentOffset)

	require.NoError(t.T(), err)
	// nextBlockIndexToPrefetch should be start block index (2) + scheduled blocks (2).
	assert.Equal(t.T(), int64(4), reader.nextBlockIndexToPrefetch)
	// numPrefetchBlocks for the next prefetch should be initialPrefetchBlockCnt (2) * prefetchMultiplier (2).
	assert.Equal(t.T(), int64(4), reader.numPrefetchBlocks)
	assert.Equal(t.T(), 2, reader.blockQueue.Len())
	// Verify block 2.
	bqe1 := reader.blockQueue.Pop()
	assert.Equal(t.T(), int64(2048), bqe1.block.AbsStartOff())
	status1, err1 := bqe1.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err1)
	assert.Equal(t.T(), block.BlockStateDownloaded, status1.State)
	assertBlockContent(t.T(), bqe1.block, bqe1.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	// Verify block 3.
	bqe2 := reader.blockQueue.Pop()
	assert.Equal(t.T(), int64(3072), bqe2.block.AbsStartOff())
	status2, err2 := bqe2.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err2)
	assert.Equal(t.T(), block.BlockStateDownloaded, status2.State)
	// Assert content for the partial block.
	assertBlockContent(t.T(), bqe2.block, bqe2.block.AbsStartOff(), partialBlockSize)
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestPrefetch() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 0 })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 0), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 1024 })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 1024), nil).Once()

	err = reader.prefetch()

	require.NoError(t.T(), err)
	// nextBlockIndexToPrefetch should be start block index (0) + initialPrefetchBlockCnt (2).
	assert.Equal(t.T(), int64(2), reader.nextBlockIndexToPrefetch)
	// numPrefetchBlocks for the next prefetch should be initialPrefetchBlockCnt (2) * prefetchMultiplier (2).
	assert.Equal(t.T(), int64(4), reader.numPrefetchBlocks)
	assert.Equal(t.T(), 2, reader.blockQueue.Len())
	// Wait for all downloads to complete.
	bqe1 := reader.blockQueue.Pop()
	status1, err1 := bqe1.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err1)
	assert.Equal(t.T(), block.BlockStateDownloaded, status1.State)
	assertBlockContent(t.T(), bqe1.block, bqe1.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	bqe2 := reader.blockQueue.Pop()
	status2, err2 := bqe2.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err2)
	assert.Equal(t.T(), block.BlockStateDownloaded, status2.State)
	assertBlockContent(t.T(), bqe2.block, bqe2.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestPrefetchWithMultiplicativeIncrease() {
	t.config.InitialPrefetchBlockCnt = 1
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	// First prefetch schedules 1 block.
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 0 })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 0), nil).Once()
	err = reader.prefetch()
	require.NoError(t.T(), err)
	// Wait for the first prefetch to complete and drain the queue.
	bqe1 := reader.blockQueue.Pop()
	status1, err1 := bqe1.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err1)
	assert.Equal(t.T(), block.BlockStateDownloaded, status1.State)
	assertBlockContent(t.T(), bqe1.block, bqe1.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	// Second prefetch should schedule 2 blocks due to multiplicative increase.
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 1024 })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 1024), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 2048 })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 2048), nil).Once()

	err = reader.prefetch()

	require.NoError(t.T(), err)
	// nextBlockIndexToPrefetch should be blocks from first prefetch (1) + blocks from second prefetch (2).
	assert.Equal(t.T(), int64(3), reader.nextBlockIndexToPrefetch)
	// numPrefetchBlocks for the next prefetch should be numPrefetchBlocks from previous prefetch (2) * prefetchMultiplier (2).
	assert.Equal(t.T(), int64(4), reader.numPrefetchBlocks)
	assert.Equal(t.T(), 2, reader.blockQueue.Len())
	// Wait for the second prefetch to complete.
	bqe2 := reader.blockQueue.Pop()
	status2, err2 := bqe2.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err2)
	assert.Equal(t.T(), block.BlockStateDownloaded, status2.State)
	assertBlockContent(t.T(), bqe2.block, bqe2.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	bqe3 := reader.blockQueue.Pop()
	status3, err3 := bqe3.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err3)
	assert.Equal(t.T(), block.BlockStateDownloaded, status3.State)
	assertBlockContent(t.T(), bqe3.block, bqe3.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestPrefetchWhenQueueIsFull() {
	t.config.MaxPrefetchBlockCnt = 2
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	b, err := reader.blockPool.Get()
	require.NoError(t.T(), err)
	// Fill the block queue to its maximum capacity.
	reader.blockQueue.Push(&blockQueueEntry{block: b})
	reader.blockQueue.Push(&blockQueueEntry{block: b})

	err = reader.prefetch()

	require.NoError(t.T(), err)
	// No new blocks should be prefetched, so the index remains 0.
	assert.Equal(t.T(), int64(0), reader.nextBlockIndexToPrefetch)
	// The queue length should remain at MaxPrefetchBlockCnt.
	assert.Equal(t.T(), 2, reader.blockQueue.Len())
	// numPrefetchBlocks should remain at its default/current value (2 in this case, due to InitialPrefetchBlockCnt).
	assert.Equal(t.T(), int64(2), reader.numPrefetchBlocks)
}

func (t *BufferedReaderTest) TestPrefetchWhenQueueIsPartiallyFull() {
	t.config.MaxPrefetchBlockCnt = 4
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	b, err := reader.blockPool.Get()
	require.NoError(t.T(), err)
	reader.blockQueue.Push(&blockQueueEntry{block: b})
	reader.blockQueue.Push(&blockQueueEntry{block: b})
	// blockCountToPrefetch = min(numPrefetchBlocks (2), availableSlots (2)) = 2.
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 0 })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 0), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 1024 })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 1024), nil).Once()

	err = reader.prefetch()

	require.NoError(t.T(), err)
	// nextBlockIndexToPrefetch should be the number of scheduled blocks (2).
	assert.Equal(t.T(), int64(2), reader.nextBlockIndexToPrefetch)
	// blockQueue.Len() should be already in queue (2) + newly scheduled blocks (2).
	assert.Equal(t.T(), 4, reader.blockQueue.Len())
	// numPrefetchBlocks for the next prefetch should be previous numPrefetchBlocks (2) * prefetchMultiplier (2).
	assert.Equal(t.T(), int64(4), reader.numPrefetchBlocks)
	// Wait for the newly scheduled downloads to complete. The old blocks are dummies.
	bqe1 := reader.blockQueue.Pop()
	reader.blockPool.Release(bqe1.block)
	bqe2 := reader.blockQueue.Pop()
	reader.blockPool.Release(bqe2.block)
	bqe3 := reader.blockQueue.Pop()
	status3, err3 := bqe3.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err3)
	assert.Equal(t.T(), block.BlockStateDownloaded, status3.State)
	assertBlockContent(t.T(), bqe3.block, bqe3.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	bqe4 := reader.blockQueue.Pop()
	status4, err4 := bqe4.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err4)
	assert.Equal(t.T(), block.BlockStateDownloaded, status4.State)
	assertBlockContent(t.T(), bqe4.block, bqe4.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestPrefetchLimitedByAvailableSlots() {
	t.config.MaxPrefetchBlockCnt = 4
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	reader.numPrefetchBlocks = 4
	b, err := reader.blockPool.Get()
	require.NoError(t.T(), err)
	reader.blockQueue.Push(&blockQueueEntry{block: b})
	reader.blockQueue.Push(&blockQueueEntry{block: b})
	reader.blockQueue.Push(&blockQueueEntry{block: b})
	// blockCountToPrefetch = min(numPrefetchBlocks (4), availableSlots (1)) = 1.
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 0 })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 0), nil).Once()

	err = reader.prefetch()

	require.NoError(t.T(), err)
	// nextBlockIndexToPrefetch should be the number of scheduled blocks (1).
	assert.Equal(t.T(), int64(1), reader.nextBlockIndexToPrefetch)
	// blockQueue.Len() should be already in queue (3) + newly scheduled blocks (1).
	assert.Equal(t.T(), 4, reader.blockQueue.Len())
	// numPrefetchBlocks for the next prefetch should be current numPrefetchBlocks (4) * prefetchMultiplier (2) = 8,
	// but capped at MaxPrefetchBlockCnt (4).
	assert.Equal(t.T(), int64(4), reader.numPrefetchBlocks)
	// Release dummy blocks and wait for the newly scheduled download to complete.
	bqe1 := reader.blockQueue.Pop()
	reader.blockPool.Release(bqe1.block)
	bqe2 := reader.blockQueue.Pop()
	reader.blockPool.Release(bqe2.block)
	bqe3 := reader.blockQueue.Pop()
	reader.blockPool.Release(bqe3.block)
	bqe4 := reader.blockQueue.Pop()
	status4, err4 := bqe4.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err4)
	assert.Equal(t.T(), block.BlockStateDownloaded, status4.State)
	assertBlockContent(t.T(), bqe4.block, bqe4.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestPrefetchStopsWhenPoolIsExhausted() {
	// Configure a small pool that will be exhausted, to test the case where
	// prefetching is not possible.
	t.config.MaxPrefetchBlockCnt = 4
	t.config.InitialPrefetchBlockCnt = 2
	// The global semaphore only has enough permits for the reserved blocks.
	t.globalMaxBlocksSem = semaphore.NewWeighted(2)
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	// At this point, NewBufferedReader has acquired 2 permits for its reserved blocks.
	// The global semaphore is now empty.
	// The first prefetch() call will succeed by allocating the 2 reserved blocks.
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 0 })).Return(createFakeReaderWithOffset(t.T(), 1024, 0), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 1024 })).Return(createFakeReaderWithOffset(t.T(), 1024, 1024), nil).Once()
	err = reader.prefetch()
	require.NoError(t.T(), err)
	require.Equal(t.T(), 2, reader.blockQueue.Len())
	assert.Equal(t.T(), int64(4), reader.numPrefetchBlocks, "numPrefetchBlocks should be multiplied after successful prefetch")
	// The pool has now created 2 blocks (totalBlocks=2), which is its max (maxBlocks=2).
	// To simulate a state where the pool is exhausted, we drain the queue without
	// releasing the blocks back to the pool's free channel. We must wait for the
	// downloads to complete before proceeding.
	bqe1 := reader.blockQueue.Pop()
	_, _ = bqe1.block.AwaitReady(t.ctx)
	bqe2 := reader.blockQueue.Pop()
	_, _ = bqe2.block.AwaitReady(t.ctx)
	// Now the blockQueue and freeBlocksCh are empty, but totalBlocks is at its limit.

	// The next prefetch call should attempt to schedule blocks but fail to get
	// any from the exhausted pool. It should not return an error.
	err = reader.prefetch()

	require.NoError(t.T(), err, "prefetch should handle block unavailability gracefully")
	assert.Equal(t.T(), 0, reader.blockQueue.Len(), "No new blocks should have been scheduled")
	assert.Equal(t.T(), int64(2), reader.nextBlockIndexToPrefetch, "The index should not have advanced")
	assert.Equal(t.T(), int64(4), reader.numPrefetchBlocks, "numPrefetchBlocks should not increase when prefetch is incomplete")
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestReadAtOffsetBeyondEOF() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	buf := make([]byte, 10)
	t.bucket.On("Name").Return("test-bucket").Maybe() // Bucket name used for logging.

	resp, err := reader.ReadAt(t.ctx, buf, int64(t.object.Size+1))

	assert.ErrorIs(t.T(), err, io.EOF)
	assert.Zero(t.T(), resp.Size)
}

func (t *BufferedReaderTest) TestReadAtEmptyBuffer() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	buf := make([]byte, 0)
	t.bucket.On("Name").Return("test-bucket").Maybe() // Bucket name used for logging.

	resp, err := reader.ReadAt(t.ctx, buf, 0)

	assert.NoError(t.T(), err)
	assert.Zero(t.T(), resp.Size)
}

func (t *BufferedReaderTest) TestReadAtBackwardSeekIsRandomRead() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	// Perform a read that populates the prefetch queue.
	// This is a random read since offset != 0 and queue is empty.
	startOffset := int64(3072) // block 3
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == uint64(startOffset) })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), startOffset), nil).Once()
	t.bucket.On("Name").Return("test-bucket").Maybe() // Bucket name used for logging.
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool {
		return r.Range.Start == uint64(startOffset+testPrefetchBlockSizeBytes)
	})).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), startOffset+testPrefetchBlockSizeBytes), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool {
		return r.Range.Start == uint64(startOffset+2*testPrefetchBlockSizeBytes)
	})).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), startOffset+2*testPrefetchBlockSizeBytes), nil).Once()
	_, err = reader.ReadAt(t.ctx, make([]byte, 10), startOffset)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), int64(1), reader.randomSeekCount, "First read should be counted as random.")
	require.Equal(t.T(), 3, reader.blockQueue.Len(), "Queue should be populated after first read.")
	// Perform a backward seek, which is another random read.
	// This should clear the existing queue and start a new prefetch.
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 0 })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 0), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == uint64(testPrefetchBlockSizeBytes) })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), testPrefetchBlockSizeBytes), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 2*uint64(testPrefetchBlockSizeBytes) })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 2*testPrefetchBlockSizeBytes), nil).Once()
	buf := make([]byte, 1024)

	resp, err := reader.ReadAt(t.ctx, buf, 0)

	require.NoError(t.T(), err)
	assert.Equal(t.T(), int(1024), resp.Size)
	assert.Equal(t.T(), int64(2), reader.randomSeekCount, "Second read should be counted as random.")
	assert.Equal(t.T(), 2, reader.blockQueue.Len(), "Queue should contain newly prefetched blocks.")
	assertBufferContent(t.T(), buf, 0)
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestReadAtForwardSeekDiscardsPreviousBlocks() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	var cancelCount int
	addBlockToQueue := func(offset int64) {
		b, poolErr := reader.blockPool.Get()
		require.NoError(t.T(), poolErr)
		require.NoError(t.T(), b.SetAbsStartOff(offset))
		_, writeErr := b.Write(make([]byte, testPrefetchBlockSizeBytes))
		require.NoError(t.T(), writeErr)
		b.NotifyReady(block.BlockStatus{State: block.BlockStateDownloaded})
		reader.blockQueue.Push(&blockQueueEntry{
			block:  b,
			cancel: func() { cancelCount++ },
		})
	}
	addBlockToQueue(0)    // block 0
	addBlockToQueue(1024) // block 1
	addBlockToQueue(2048) // block 2
	// Manually update the reader's state to reflect the manually added blocks.
	reader.nextBlockIndexToPrefetch = 3
	require.Equal(t.T(), 3, reader.blockQueue.Len())
	// Reading block 2 should trigger a prefetch for blocks 3 and 4.
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 3*uint64(testPrefetchBlockSizeBytes) })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 3*testPrefetchBlockSizeBytes), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 4*uint64(testPrefetchBlockSizeBytes) })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 4*testPrefetchBlockSizeBytes), nil).Once()
	readOffset := int64(2048)
	t.bucket.On("Name").Return("test-bucket").Maybe() // Bucket name used for logging.

	// Read the entire block at offset 2048 to trigger the prefetch logic.
	_, err = reader.ReadAt(t.ctx, make([]byte, 1024), readOffset)

	require.NoError(t.T(), err)
	assert.Equal(t.T(), 2, cancelCount, "Expected 2 blocks to be discarded")
	// The queue should now contain the two newly prefetched blocks.
	require.Equal(t.T(), 2, reader.blockQueue.Len(), "Queue should contain the 2 newly prefetched blocks")
	// Wait for the async prefetch tasks to complete to verify the mock calls.
	bqe3 := reader.blockQueue.Pop()
	bqe4 := reader.blockQueue.Pop()
	_, err = bqe3.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err, "AwaitReady for block 3 failed")
	_, err = bqe4.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err, "AwaitReady for block 4 failed")
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestReadAtInitialDownloadFails() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	downloadError := errors.New("gcs error")
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.AnythingOfType("*gcs.ReadObjectRequest")).Return(nil, downloadError)
	t.bucket.On("Name").Return("test-bucket").Maybe() // Bucket name used for logging.
	buf := make([]byte, 10)

	_, err = reader.ReadAt(t.ctx, buf, 0)

	assert.ErrorContains(t.T(), err, "download failed")
	assert.ErrorIs(t.T(), err, downloadError)
	// After the failed read, the other prefetched blocks should also have failed.
	// We wait for them to finish to avoid a race condition and to verify their state.
	require.Equal(t.T(), 2, reader.blockQueue.Len())
	bqe1 := reader.blockQueue.Pop()
	status1, err1 := bqe1.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err1)
	assert.Equal(t.T(), block.BlockStateDownloadFailed, status1.State)
	assert.ErrorIs(t.T(), status1.Err, downloadError)
	bqe2 := reader.blockQueue.Pop()
	status2, err2 := bqe2.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err2)
	assert.Equal(t.T(), block.BlockStateDownloadFailed, status2.State)
	assert.ErrorIs(t.T(), status2.Err, downloadError)
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestReadAtAwaitReadyCancelled() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	b, err := reader.blockPool.Get()
	require.NoError(t.T(), err)
	err = b.SetAbsStartOff(0)
	require.NoError(t.T(), err)
	reader.blockQueue.Push(&blockQueueEntry{block: b, cancel: func() {}})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	t.bucket.On("Name").Return("test-bucket").Maybe() // Bucket name used for logging.

	// Read with a cancelled context.
	_, err = reader.ReadAt(ctx, make([]byte, 10), 0)

	assert.ErrorIs(t.T(), err, context.Canceled)
}

func (t *BufferedReaderTest) TestReadAtBlockStateDownloadFailed() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	b, err := reader.blockPool.Get()
	require.NoError(t.T(), err)
	err = b.SetAbsStartOff(0)
	require.NoError(t.T(), err)
	downloadError := errors.New("simulated download error")
	b.NotifyReady(block.BlockStatus{State: block.BlockStateDownloadFailed, Err: downloadError})
	reader.blockQueue.Push(&blockQueueEntry{block: b, cancel: func() {}})
	t.bucket.On("Name").Return("test-bucket").Maybe() // Bucket name used for logging.

	// Read from a reader where the next block has failed to download.
	_, err = reader.ReadAt(t.ctx, make([]byte, 10), 0)

	assert.ErrorIs(t.T(), err, downloadError)
	status, err := b.AwaitReady(t.ctx)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), block.BlockStateDownloadFailed, status.State)
	assert.ErrorIs(t.T(), status.Err, downloadError)
	assert.True(t.T(), reader.blockQueue.IsEmpty())
}

func (t *BufferedReaderTest) TestReadAtBlockDownloadCancelled() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	b, err := reader.blockPool.Get()
	require.NoError(t.T(), err)
	err = b.SetAbsStartOff(0)
	require.NoError(t.T(), err)
	b.NotifyReady(block.BlockStatus{State: block.BlockStateDownloadFailed, Err: context.Canceled})
	reader.blockQueue.Push(&blockQueueEntry{block: b, cancel: func() {}})
	t.bucket.On("Name").Return("test-bucket").Maybe() // Bucket name used for logging.

	// Read from a reader where the next block download was cancelled.
	_, err = reader.ReadAt(t.ctx, make([]byte, 10), 0)

	assert.ErrorIs(t.T(), err, context.Canceled)
	status, err := b.AwaitReady(t.ctx)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), block.BlockStateDownloadFailed, status.State)
	assert.ErrorIs(t.T(), status.Err, context.Canceled)
	assert.True(t.T(), reader.blockQueue.IsEmpty())
}

func (t *BufferedReaderTest) TestReadAtBlockStateUnexpected() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	b, err := reader.blockPool.Get()
	require.NoError(t.T(), err)
	err = b.SetAbsStartOff(0)
	require.NoError(t.T(), err)
	b.NotifyReady(block.BlockStatus{State: block.BlockStateInProgress})
	reader.blockQueue.Push(&blockQueueEntry{block: b, cancel: func() {}})
	t.bucket.On("Name").Return("test-bucket").Maybe() // Bucket name used for logging.

	// Read from a reader where the next block is in an unexpected state.
	_, err = reader.ReadAt(t.ctx, make([]byte, 10), 0)

	assert.ErrorContains(t.T(), err, "unexpected block state")
	status, err := b.AwaitReady(t.ctx)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), block.BlockStateInProgress, status.State)
	assert.Nil(t.T(), status.Err)
	assert.True(t.T(), reader.blockQueue.IsEmpty())
}

func (t *BufferedReaderTest) TestReadAtFromDownloadedBlock() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	b, err := reader.blockPool.Get()
	require.NoError(t.T(), err)
	err = b.SetAbsStartOff(0)
	require.NoError(t.T(), err)
	content := []byte("abcdefghijk")
	_, err = b.Write(content)
	require.NoError(t.T(), err)
	b.NotifyReady(block.BlockStatus{State: block.BlockStateDownloaded})
	reader.blockQueue.Push(&blockQueueEntry{block: b, cancel: func() {}})
	buf := make([]byte, 5)
	t.bucket.On("Name").Return("test-bucket").Maybe() // Bucket name used for logging.

	// Read from a block that is already downloaded and in the queue.
	resp, err := reader.ReadAt(t.ctx, buf, 0)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 5, resp.Size)
	assert.Equal(t.T(), "abcde", string(buf))
	assert.False(t.T(), reader.blockQueue.IsEmpty())
}

func (t *BufferedReaderTest) TestReadAtExactlyToEndOfFile() {
	t.object.Size = uint64(testPrefetchBlockSizeBytes + 50) // 1 full block and 50 bytes
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 0 })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 0), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == uint64(testPrefetchBlockSizeBytes) })).Return(createFakeReaderWithOffset(t.T(), 50, testPrefetchBlockSizeBytes), nil).Once()
	buf := make([]byte, t.object.Size)
	t.bucket.On("Name").Return("test-bucket").Maybe() // Bucket name used for logging.

	// Read the entire file.
	resp, err := reader.ReadAt(t.ctx, buf, 0)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), int(t.object.Size), resp.Size)
	assertBufferContent(t.T(), buf, 0)
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestReadAtSucceedsWhenPrefetchFails() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	// Mock GCS reads where the initial read and first prefetch succeed, but the second prefetch fails.
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 0 })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 0), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == uint64(testPrefetchBlockSizeBytes) })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), testPrefetchBlockSizeBytes), nil).Once()
	prefetchError := errors.New("prefetch failed")
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 2*uint64(testPrefetchBlockSizeBytes) })).Return(nil, prefetchError).Once()
	buf := make([]byte, testPrefetchBlockSizeBytes)
	t.bucket.On("Name").Return("test-bucket").Maybe() // Bucket name used for logging.

	// Read the first block. This should succeed, even though a background prefetch will fail.
	resp, err := reader.ReadAt(t.ctx, buf, 0)

	require.NoError(t.T(), err)
	assert.Equal(t.T(), int(testPrefetchBlockSizeBytes), resp.Size)
	assertBufferContent(t.T(), buf, 0)
	// After reading block 0, the queue should contain the successful and failed prefetched blocks.
	require.Equal(t.T(), 2, reader.blockQueue.Len())
	// Wait for background downloads to complete to prevent a race condition.
	bqe1 := reader.blockQueue.Pop()
	status1, err1 := bqe1.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err1)
	assert.Equal(t.T(), block.BlockStateDownloaded, status1.State)
	bqe2 := reader.blockQueue.Pop()
	status2, err2 := bqe2.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err2)
	assert.Equal(t.T(), block.BlockStateDownloadFailed, status2.State)
	assert.ErrorIs(t.T(), status2.Err, prefetchError)
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestReadAtSpanningMultipleBlocks() {
	// Read 2.5 blocks of data in a single ReadAt call.
	readSize := 2560
	readOffset := int64(0)
	t.object.Size = 3072
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	buf := make([]byte, readSize)
	// freshStart will be called, downloading block 0 (urgent) and
	// prefetching blocks 1 and 2 (InitialPrefetchBlockCnt=2).
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool {
		return r.Range.Start == uint64(0*testPrefetchBlockSizeBytes)
	})).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 0*testPrefetchBlockSizeBytes), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool {
		return r.Range.Start == uint64(1*testPrefetchBlockSizeBytes)
	})).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 1*testPrefetchBlockSizeBytes), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool {
		return r.Range.Start == uint64(2*testPrefetchBlockSizeBytes)
	})).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 2*testPrefetchBlockSizeBytes), nil).Once()
	t.bucket.On("Name").Return("test-bucket").Maybe() // Bucket name used for logging.

	resp, err := reader.ReadAt(t.ctx, buf, readOffset)

	require.NoError(t.T(), err)
	assert.Equal(t.T(), 2560, resp.Size)
	assertBufferContent(t.T(), buf, readOffset)
	assert.Equal(t.T(), 1, reader.blockQueue.Len(), "Block 2 should be left in the queue.")
	assert.Equal(t.T(), int64(2048), reader.blockQueue.Peek().block.AbsStartOff())
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestReadAtSequentialReadAcrossBlocks() {
	t.config.InitialPrefetchBlockCnt = 1
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	// Mock reads for all blocks that will be downloaded.
	// First ReadAt(0) triggers freshStart, which downloads block 0 (urgent) and prefetches block 1.
	// Second ReadAt(1024) consumes block 1 and triggers prefetch for blocks 2, 3.
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool {
		return r.Range.Start == uint64(0*testPrefetchBlockSizeBytes)
	})).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 0*testPrefetchBlockSizeBytes), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool {
		return r.Range.Start == uint64(1*testPrefetchBlockSizeBytes)
	})).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 1*testPrefetchBlockSizeBytes), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool {
		return r.Range.Start == uint64(2*testPrefetchBlockSizeBytes)
	})).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 2*testPrefetchBlockSizeBytes), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool {
		return r.Range.Start == uint64(3*testPrefetchBlockSizeBytes)
	})).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 3*testPrefetchBlockSizeBytes), nil).Once()
	buf1 := make([]byte, testPrefetchBlockSizeBytes)
	buf2 := make([]byte, testPrefetchBlockSizeBytes)
	t.bucket.On("Name").Return("test-bucket").Maybe() // Bucket name used for logging.

	// Perform two sequential reads.
	_, err = reader.ReadAt(t.ctx, buf1, 0)
	require.NoError(t.T(), err)
	_, err = reader.ReadAt(t.ctx, buf2, testPrefetchBlockSizeBytes)
	require.NoError(t.T(), err)

	assert.Equal(t.T(), int64(0), reader.randomSeekCount)
	assertBufferContent(t.T(), buf1, 0)
	assertBufferContent(t.T(), buf2, testPrefetchBlockSizeBytes)
	// Wait for all background prefetches to complete before asserting mock expectations.
	require.Equal(t.T(), 2, reader.blockQueue.Len())
	bqe1 := reader.blockQueue.Pop()
	_, err = bqe1.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err)
	bqe2 := reader.blockQueue.Pop()
	_, err = bqe2.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err)
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestReadAtFallsBackAfterRandomReads() {
	t.config.InitialPrefetchBlockCnt = 1
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	reader.randomReadsThreshold = 2
	require.NoError(t.T(), err)
	buf := make([]byte, 10)
	// Mock GCS calls for the first random read, which will download block 2 and prefetch block 3.
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 2*uint64(testPrefetchBlockSizeBytes) })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 2*testPrefetchBlockSizeBytes), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 3*uint64(testPrefetchBlockSizeBytes) })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 3*testPrefetchBlockSizeBytes), nil).Once()
	t.bucket.On("Name").Return("test-bucket").Maybe() // Bucket name used for logging.
	// First random read should succeed.
	_, err = reader.ReadAt(t.ctx, buf, 2*testPrefetchBlockSizeBytes)
	require.NoError(t.T(), err, "Random read #1 should succeed")
	// Mock GCS calls for the second random read, which will download block 5 and prefetch block 6.
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 5*uint64(testPrefetchBlockSizeBytes) })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 5*testPrefetchBlockSizeBytes), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 6*uint64(testPrefetchBlockSizeBytes) })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 6*testPrefetchBlockSizeBytes), nil).Once()
	// Second random read should succeed.
	_, err = reader.ReadAt(t.ctx, buf, 5*testPrefetchBlockSizeBytes)
	require.NoError(t.T(), err, "Random read #2 should succeed")

	// The third random read should exceed the threshold and trigger the fallback.
	_, err = reader.ReadAt(t.ctx, buf, 0)

	assert.ErrorIs(t.T(), err, gcsx.FallbackToAnotherReader, "Error should be FallbackToAnotherReader")
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestReadAtFallbackOnFreshStartFailure() {
	t.config.MaxPrefetchBlockCnt = 2
	t.config.InitialPrefetchBlockCnt = 2
	t.globalMaxBlocksSem = semaphore.NewWeighted(2)
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	// Manually exhaust the pool's blocks to simulate a scenario where all blocks are in use.
	_, err = reader.blockPool.Get()
	require.NoError(t.T(), err)
	_, err = reader.blockPool.Get()
	require.NoError(t.T(), err)
	t.bucket.On("Name").Return("test-bucket").Maybe()

	_, err = reader.ReadAt(t.ctx, make([]byte, 10), 0)

	assert.ErrorIs(t.T(), err, gcsx.FallbackToAnotherReader, "ReadAt should fall back when freshStart fails to get a block")
}

func (t *BufferedReaderTest) TestReadAtFallbackOnMmapFailure() {
	// Configure a huge block size that will likely cause mmap to fail.
	// This simulates a non-recoverable error during block creation within the
	// buffered reader, which should cause a fallback.
	t.config.PrefetchBlockSizeBytes = oneTB
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	t.bucket.On("Name").Return("test-bucket").Maybe()

	_, err = reader.ReadAt(t.ctx, make([]byte, 10), 0)

	assert.ErrorIs(t.T(), err, gcsx.FallbackToAnotherReader, "ReadAt should fall back when mmap fails")
}

func (t *BufferedReaderTest) TestReadAtExceedsObjectSize() {
	objectSize := uint64(1536) // 1.5 blocks
	readOffset := int64(1024)
	readSize := int(1024) // Tries to read 1024 bytes, but only 512 are available.
	t.object.Size = objectSize
	t.object.Generation = 12345
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	buf := make([]byte, readSize)
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool {
		return r.Range.Start == uint64(testPrefetchBlockSizeBytes) && r.Range.Limit == objectSize
	})).Return(createFakeReaderWithOffset(t.T(), 512, testPrefetchBlockSizeBytes), nil).Once()
	t.bucket.On("Name").Return("test-bucket").Maybe() // Bucket name used for logging.

	resp, err := reader.ReadAt(t.ctx, buf, readOffset)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 512, resp.Size)
	assertBufferContent(t.T(), buf[:resp.Size], readOffset)
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestReadAtSucceedsWhenBackgroundPrefetchFailsDueToGlobalSem() {
	// Configure a scenario where the initial read succeeds, but the subsequent
	// background prefetch fails due to an exhausted global semaphore.
	t.config.MaxPrefetchBlockCnt = 3
	t.config.InitialPrefetchBlockCnt = 1
	t.globalMaxBlocksSem = semaphore.NewWeighted(2)
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 0 })).Return(createFakeReaderWithOffset(t.T(), 1024, 0), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 1024 })).Return(createFakeReaderWithOffset(t.T(), 1024, 1024), nil).Once()
	t.bucket.On("Name").Return("test-bucket").Maybe()
	buf := make([]byte, 1024)

	// The read should succeed. When this read consumes block 0, it will trigger
	// a background prefetch for block 2, which will fail because the global
	// semaphore is exhausted. This failure should not affect the foreground read.
	resp, err := reader.ReadAt(t.ctx, buf, 0)

	require.NoError(t.T(), err)
	assert.Equal(t.T(), 1024, resp.Size)
	assertBufferContent(t.T(), buf, 0)
	require.Equal(t.T(), 1, reader.blockQueue.Len())
	assert.Equal(t.T(), int64(1024), reader.blockQueue.Peek().block.AbsStartOff())
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestReadAtSucceedsWhenBackgroundPrefetchFailsOnGCSError() {
	t.config.MaxPrefetchBlockCnt = 2
	t.config.InitialPrefetchBlockCnt = 2
	t.globalMaxBlocksSem = semaphore.NewWeighted(2)
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	// Mock the first block download to succeed, but the second (prefetched) block
	// to fail with a GCS error.
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 0 })).Return(createFakeReaderWithOffset(t.T(), 1024, 0), nil).Once()
	gcsError := errors.New("simulated GCS error")
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 1024 })).Return(nil, gcsError).Once()
	t.bucket.On("Name").Return("test-bucket").Maybe()
	buf := make([]byte, 10)

	// The initial read should succeed because it reads from the first block, which
	// was downloaded successfully. The background prefetch failure for the second
	// block should not affect this call.
	resp, err := reader.ReadAt(t.ctx, buf, 0)

	require.NoError(t.T(), err)
	assert.Equal(t.T(), 10, resp.Size)
	assertBufferContent(t.T(), buf, 0)
	// A subsequent attempt to read the second block (which failed to prefetch)
	// should return the original GCS error.
	_, err = reader.ReadAt(t.ctx, buf, 1024)
	assert.ErrorIs(t.T(), err, gcsError)
	assert.ErrorContains(t.T(), err, "download failed")
}

func (t *BufferedReaderTest) TestReadAtSubsequentReadAfterFallbackAlsoFallsBack() {
	t.config.InitialPrefetchBlockCnt = 1
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	reader.randomReadsThreshold = 1 // Set a low threshold for the test.
	require.NoError(t.T(), err)
	buf := make([]byte, 10)
	t.bucket.On("Name").Return("test-bucket").Maybe()
	// First random read (offset != 0): should succeed and count as 1st random seek.
	// This will trigger a freshStart, downloading block 2 and prefetching block 3.
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 2*uint64(testPrefetchBlockSizeBytes) })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 2*testPrefetchBlockSizeBytes), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 3*uint64(testPrefetchBlockSizeBytes) })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 3*testPrefetchBlockSizeBytes), nil).Once()
	_, err = reader.ReadAt(t.ctx, buf, 2*testPrefetchBlockSizeBytes)
	require.NoError(t.T(), err, "Random read #1 should succeed")
	assert.Equal(t.T(), int64(1), reader.randomSeekCount)
	// Second random read: should exceed threshold and trigger fallback.
	_, err = reader.ReadAt(t.ctx, buf, 5*testPrefetchBlockSizeBytes)
	assert.ErrorIs(t.T(), err, gcsx.FallbackToAnotherReader, "Random read #2 should trigger fallback")
	assert.Equal(t.T(), int64(2), reader.randomSeekCount)

	// Third read (at any offset): should also fall back immediately because the reader is already in a fallback state.
	_, err = reader.ReadAt(t.ctx, buf, 0)

	assert.ErrorIs(t.T(), err, gcsx.FallbackToAnotherReader, "Subsequent read should also fallback")
	assert.Equal(t.T(), int64(2), reader.randomSeekCount, "Random seek count should not change")
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestReadAtConcurrentReads() {
	const (
		fileSize      = 10 * util.MiB
		numGoroutines = 3
		blockSize     = 1 * util.MiB
		readSize      = 1 * util.MiB
	)
	t.object.Size = fileSize
	t.config.PrefetchBlockSizeBytes = blockSize
	t.config.MaxPrefetchBlockCnt = 10
	t.config.InitialPrefetchBlockCnt = 2 // This will prefetch 2 blocks after the initial one.
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	// Set up mocks for all possible block reads. Because the goroutines run
	// concurrently, we prepare mocks for all blocks that could be read or
	// prefetched (2 blocks) and use .Maybe() to allow them to be called in
	// any order.
	for i := 0; i <= 8; i++ {
		start := uint64(i * blockSize)
		// Create content for this block using the A-Z pattern from the test helpers.
		blockContent := make([]byte, blockSize)
		for j := range blockContent {
			blockContent[j] = byte('A' + ((int(start) + j) % 26))
		}
		t.bucket.On("NewReaderWithReadHandle",
			mock.Anything,
			mock.MatchedBy(func(req *gcs.ReadObjectRequest) bool {
				return req.Range.Start == start
			}),
		).Return(&fake.FakeReader{ReadCloser: io.NopCloser(bytes.NewReader(blockContent))}, nil).Maybe()
	}
	t.bucket.On("Name").Return("test-bucket").Maybe()
	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	results := make([][]byte, numGoroutines)

	// Each go routine will read different range to avoid duplicate calls for same range.
	// That's why we are multiplying by 3 to have offset 3 blocks apart.
	var readIndex = 3
	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()
			offset := int64(index * readIndex * readSize)
			readBuf := make([]byte, readSize)

			resp, err := reader.ReadAt(t.ctx, readBuf, offset)

			require.NoError(t.T(), err)
			require.Equal(t.T(), readSize, resp.Size)
			// Copy the result to a new slice to avoid data races on readBuf.
			results[index] = make([]byte, readSize)
			copy(results[index], readBuf)
		}(i)
	}

	wg.Wait()
	// Verify the results from all goroutines individually.
	for i, res := range results {
		offset := int64(i * readIndex * readSize)
		assertBufferContent(t.T(), res, offset)
	}
	t.bucket.AssertExpectations(t.T())
}
