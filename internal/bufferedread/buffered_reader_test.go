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
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/block"
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
	testGlobalMaxBlocks         int64 = 20
	testPrefetchBlockSizeBytes  int64 = 1024
	testInitialPrefetchBlockCnt int64 = 2
	testPrefetchMultiplier      int64 = 2
	testRandomReadsThreshold    int64 = 3
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

// createFakeReader returns a FakeReader with deterministic, non-zero content.
func createFakeReader(t *testing.T, size int) *fake.FakeReader {
	t.Helper()
	content := make([]byte, size)
	for i := range content {
		content[i] = byte('A' + (i % 26)) // A-Z repeating pattern
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
	for i := 0; i < n; i++ {
		expected := byte('A' + (i % 26))
		assert.Equalf(t, expected, buf[i], "Mismatch at offset %d", expectedOffset+int64(i))
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
		PrefetchMultiplier:      testPrefetchMultiplier,
		RandomReadsThreshold:    testRandomReadsThreshold,
	}
	var err error
	t.workerPool, err = workerpool.NewStaticWorkerPool(5, 10)
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

func (t *BufferedReaderTest) TestDestroySuccess() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err, "NewBufferedReader should not return error")
	b, err := reader.blockPool.Get()
	require.NoError(t.T(), err, "Failed to get block from pool")
	reader.blockQueue.Push(&blockQueueEntry{
		block:  b,
		cancel: func() {},
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

	b.NotifyReady(block.BlockStatus{State: block.BlockStateDownloadCancelled, Err: errors.New("test error")})
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

	reader.randomSeekCount = t.config.RandomReadsThreshold + 1

	assert.Panics(t.T(), func() { reader.CheckInvariants() })
}

func (t *BufferedReaderTest) TestCheckInvariantsPrefetchBlockSizeNotPositive() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err, "NewBufferedReader should not return error")

	// Test with zero
	reader.config.PrefetchBlockSizeBytes = 0
	assert.Panics(t.T(), func() { reader.CheckInvariants() }, "Should panic for zero block size")

	// Test with negative
	reader.config.PrefetchBlockSizeBytes = -1
	assert.Panics(t.T(), func() { reader.CheckInvariants() }, "Should panic for negative block size")
}

func (t *BufferedReaderTest) TestCheckInvariantsNoPanic() {
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
			t.bucket.On("NewReaderWithReadHandle",
				mock.Anything,
				mock.AnythingOfType("*gcs.ReadObjectRequest"),
			).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()

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
	t.bucket.On("NewReaderWithReadHandle",
		mock.Anything,
		mock.AnythingOfType("*gcs.ReadObjectRequest"),
	).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	err = reader.scheduleNextBlock(false)
	require.NoError(t.T(), err)
	bqe1 := reader.blockQueue.Pop()
	assert.Equal(t.T(), int64(1), reader.nextBlockIndexToPrefetch)
	status1, err := bqe1.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), block.BlockStateDownloaded, status1.State)
	assert.Equal(t.T(), int64(0), bqe1.block.AbsStartOff())
	assertBlockContent(t.T(), bqe1.block, bqe1.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	t.bucket.On("NewReaderWithReadHandle",
		mock.Anything,
		mock.AnythingOfType("*gcs.ReadObjectRequest"),
	).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()

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
				mock.AnythingOfType("*gcs.ReadObjectRequest"),
			).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
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
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.AnythingOfType("*gcs.ReadObjectRequest")).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.AnythingOfType("*gcs.ReadObjectRequest")).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.AnythingOfType("*gcs.ReadObjectRequest")).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()

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
	bqe2 := reader.blockQueue.Pop()
	assert.Equal(t.T(), int64(3072), bqe2.block.AbsStartOff())
	status2, err2 := bqe2.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err2)
	assert.Equal(t.T(), block.BlockStateDownloaded, status2.State)
	bqe3 := reader.blockQueue.Pop()
	assert.Equal(t.T(), int64(4096), bqe3.block.AbsStartOff())
	status3, err3 := bqe3.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err3)
	assert.Equal(t.T(), block.BlockStateDownloaded, status3.State)
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestFreshStartWithNonBlockAlignedOffset() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	currentOffset := int64(2500) // Start prefetching from offset 2500 (inside block 2).
	// freshStart should start prefetching from block 2. It schedules 1 urgent block
	// and 2 initial prefetch blocks, totaling 3 blocks.
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.AnythingOfType("*gcs.ReadObjectRequest")).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.AnythingOfType("*gcs.ReadObjectRequest")).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.AnythingOfType("*gcs.ReadObjectRequest")).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()

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
	bqe2 := reader.blockQueue.Pop()
	assert.Equal(t.T(), int64(3072), bqe2.block.AbsStartOff())
	status2, err2 := bqe2.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err2)
	assert.Equal(t.T(), block.BlockStateDownloaded, status2.State)
	bqe3 := reader.blockQueue.Pop()
	assert.Equal(t.T(), int64(4096), bqe3.block.AbsStartOff())
	status3, err3 := bqe3.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err3)
	assert.Equal(t.T(), block.BlockStateDownloaded, status3.State)
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestFreshStartWhenInitialCountGreaterThanMax() {
	t.config.MaxPrefetchBlockCnt = 3
	t.config.InitialPrefetchBlockCnt = 4
	t.object.Size = 4096
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	// freshStart schedules 1 urgent block and 2 prefetch blocks (InitialPrefetchBlockCnt capped by MaxPrefetchBlockCnt).
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.AnythingOfType("*gcs.ReadObjectRequest")).Return(createFakeReader(t.T(), 1024), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.AnythingOfType("*gcs.ReadObjectRequest")).Return(createFakeReader(t.T(), 1024), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.AnythingOfType("*gcs.ReadObjectRequest")).Return(createFakeReader(t.T(), 1024), nil).Once()

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
	bqe2 := reader.blockQueue.Pop()
	status2, err2 := bqe2.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err2)
	assert.Equal(t.T(), block.BlockStateDownloaded, status2.State)
	bqe3 := reader.blockQueue.Pop()
	status3, err3 := bqe3.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err3)
	assert.Equal(t.T(), block.BlockStateDownloaded, status3.State)
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestFreshStartStopsAtObjectEnd() {
	t.object.Size = 4000 // Object size is 3 blocks + a partial block.
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	currentOffset := int64(2048) // Start from block 2.
	// freshStart schedules 1 urgent block (block 2) and 1 prefetch block (block 3).
	// Prefetching stops because the object ends after block 3, totaling 2 blocks scheduled.
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.AnythingOfType("*gcs.ReadObjectRequest")).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.AnythingOfType("*gcs.ReadObjectRequest")).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()

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
	_, err = bqe1.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err)
	// Verify block 3.
	bqe2 := reader.blockQueue.Pop()
	assert.Equal(t.T(), int64(3072), bqe2.block.AbsStartOff())
	_, err = bqe2.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err)
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestPrefetch() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.AnythingOfType("*gcs.ReadObjectRequest")).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.AnythingOfType("*gcs.ReadObjectRequest")).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()

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
	bqe2 := reader.blockQueue.Pop()
	status2, err2 := bqe2.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err2)
	assert.Equal(t.T(), block.BlockStateDownloaded, status2.State)
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestPrefetchWithMultiplicativeIncrease() {
	t.config.InitialPrefetchBlockCnt = 1
	t.config.PrefetchMultiplier = 2
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	// First prefetch schedules 1 block.
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.AnythingOfType("*gcs.ReadObjectRequest")).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	err = reader.prefetch()
	require.NoError(t.T(), err)
	// Wait for the first prefetch to complete and drain the queue.
	bqe1 := reader.blockQueue.Pop()
	_, err1 := bqe1.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err1)
	// Second prefetch schedules 2 blocks due to multiplicative increase.
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.AnythingOfType("*gcs.ReadObjectRequest")).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.AnythingOfType("*gcs.ReadObjectRequest")).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()

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
	bqe3 := reader.blockQueue.Pop()
	status3, err3 := bqe3.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err3)
	assert.Equal(t.T(), block.BlockStateDownloaded, status3.State)
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
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.AnythingOfType("*gcs.ReadObjectRequest")).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.AnythingOfType("*gcs.ReadObjectRequest")).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()

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
	_, err3 := bqe3.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err3)
	bqe4 := reader.blockQueue.Pop()
	_, err4 := bqe4.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err4)
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
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.AnythingOfType("*gcs.ReadObjectRequest")).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()

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
	_, err4 := bqe4.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err4)
	t.bucket.AssertExpectations(t.T())
}
