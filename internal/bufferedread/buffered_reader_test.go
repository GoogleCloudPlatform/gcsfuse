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
// It's used for mocking GCS object reads.
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

	reader.randomSeekCount = t.config.RandomReadsThreshold + 1

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
	status1, err1 := bqe1.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err1)
	assert.Equal(t.T(), block.BlockStateDownloaded, status1.State)
	// Verify block 3.
	bqe2 := reader.blockQueue.Pop()
	assert.Equal(t.T(), int64(3072), bqe2.block.AbsStartOff())
	status2, err2 := bqe2.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err2)
	assert.Equal(t.T(), block.BlockStateDownloaded, status2.State)
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
	status1, err1 := bqe1.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err1)
	assert.Equal(t.T(), block.BlockStateDownloaded, status1.State)
	// Second prefetch should schedule 2 blocks due to multiplicative increase.
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
	status3, err3 := bqe3.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err3)
	assert.Equal(t.T(), block.BlockStateDownloaded, status3.State)
	bqe4 := reader.blockQueue.Pop()
	status4, err4 := bqe4.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err4)
	assert.Equal(t.T(), block.BlockStateDownloaded, status4.State)
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
	status4, err4 := bqe4.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err4)
	assert.Equal(t.T(), block.BlockStateDownloaded, status4.State)
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestReadAtOffsetBeyondEOF() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	buf := make([]byte, 10)

	resp, err := reader.ReadAt(t.ctx, buf, int64(t.object.Size+1))

	assert.ErrorIs(t.T(), err, io.EOF)
	assert.Zero(t.T(), resp.Size)
}

func (t *BufferedReaderTest) TestReadAtEmptyBuffer() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	buf := make([]byte, 0)

	resp, err := reader.ReadAt(t.ctx, buf, 0)

	assert.NoError(t.T(), err)
	assert.Zero(t.T(), resp.Size)
}

func (t *BufferedReaderTest) TestReadAtBackwardSeekIsRandomRead() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	startOffset := 3 * testPrefetchBlockSizeBytes
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == uint64(startOffset) })).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool {
		return r.Range.Start == uint64(startOffset+testPrefetchBlockSizeBytes)
	})).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool {
		return r.Range.Start == uint64(startOffset+2*testPrefetchBlockSizeBytes)
	})).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	_, err = reader.ReadAt(t.ctx, make([]byte, 10), startOffset)
	require.NoError(t.T(), err)
	require.Equal(t.T(), 3, reader.blockQueue.Len())
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 0 })).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == uint64(testPrefetchBlockSizeBytes) })).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 2*uint64(testPrefetchBlockSizeBytes) })).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 3*uint64(testPrefetchBlockSizeBytes) })).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	readSize := testPrefetchBlockSizeBytes
	buf := make([]byte, readSize)

	resp, err := reader.ReadAt(t.ctx, buf, 0)

	require.NoError(t.T(), err)
	assert.Equal(t.T(), int(readSize), resp.Size)
	assert.Equal(t.T(), int64(1), reader.randomSeekCount)
	assert.Equal(t.T(), 3, reader.blockQueue.Len())
	expectedContent := make([]byte, readSize)
	for i := 0; i < int(readSize); i++ {
		expectedContent[i] = byte('A' + (i % 26))
	}
	assert.Equal(t.T(), expectedContent, buf)
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestReadAtForwardSeekDiscardsPreviousBlocks() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	var cancelCount int
	addBlockToQueue := func(offset int64) {
		b, err := reader.blockPool.Get()
		require.NoError(t.T(), err)
		err = b.SetAbsStartOff(offset)
		require.NoError(t.T(), err)
		b.NotifyReady(block.BlockStatus{State: block.BlockStateDownloaded})
		reader.blockQueue.Push(&blockQueueEntry{
			block: b,
			cancel: func() {
				cancelCount++
			},
		})
	}
	addBlockToQueue(0)
	addBlockToQueue(testPrefetchBlockSizeBytes)
	addBlockToQueue(2 * testPrefetchBlockSizeBytes)
	require.Equal(t.T(), 3, reader.blockQueue.Len())
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 3*uint64(testPrefetchBlockSizeBytes) })).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 4*uint64(testPrefetchBlockSizeBytes) })).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	readOffset := 2 * testPrefetchBlockSizeBytes
	readSize := 10

	_, err = reader.ReadAt(t.ctx, make([]byte, readSize), readOffset)

	require.NoError(t.T(), err)
	assert.Equal(t.T(), 2, cancelCount)
	assert.Equal(t.T(), 2, reader.blockQueue.Len())
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestReadAtInitialDownloadFails() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	downloadError := errors.New("gcs error")
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.AnythingOfType("*gcs.ReadObjectRequest")).Return(nil, downloadError).Once()
	buf := make([]byte, 10)

	_, err = reader.ReadAt(t.ctx, buf, 0)

	assert.ErrorContains(t.T(), err, "freshStart failed")
	assert.ErrorIs(t.T(), err, downloadError)
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestReadAtAwaitReadyCancelled() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	b, _ := reader.blockPool.Get()
	b.SetAbsStartOff(0)
	reader.blockQueue.Push(&blockQueueEntry{block: b, cancel: func() {}})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = reader.ReadAt(ctx, make([]byte, 10), 0)

	assert.ErrorIs(t.T(), err, context.Canceled)
}

func (t *BufferedReaderTest) TestReadAtBlockStateDownloadFailed() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	b, _ := reader.blockPool.Get()
	err = b.SetAbsStartOff(0)
	require.NoError(t.T(), err)
	downloadError := errors.New("simulated download error")
	b.NotifyReady(block.BlockStatus{State: block.BlockStateDownloadFailed, Err: downloadError})
	reader.blockQueue.Push(&blockQueueEntry{block: b, cancel: func() {}})

	_, err = reader.ReadAt(t.ctx, make([]byte, 10), 0)

	assert.ErrorIs(t.T(), err, downloadError)
	assert.True(t.T(), reader.blockQueue.IsEmpty())
}

func (t *BufferedReaderTest) TestReadAtBlockStateCancelled() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	b, _ := reader.blockPool.Get()
	err = b.SetAbsStartOff(0)
	require.NoError(t.T(), err)
	b.NotifyReady(block.BlockStatus{State: block.BlockStateDownloadCancelled})
	reader.blockQueue.Push(&blockQueueEntry{block: b, cancel: func() {}})

	_, err = reader.ReadAt(t.ctx, make([]byte, 10), 0)

	assert.ErrorIs(t.T(), err, context.Canceled)
	assert.True(t.T(), reader.blockQueue.IsEmpty())
}

func (t *BufferedReaderTest) TestReadAtBlockStateUnexpected() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	b, _ := reader.blockPool.Get()
	b.SetAbsStartOff(0)
	// Use a state that is not handled in the switch case.
	b.NotifyReady(block.BlockStatus{State: block.BlockStateInProgress})
	reader.blockQueue.Push(&blockQueueEntry{block: b, cancel: func() {}})

	_, err = reader.ReadAt(t.ctx, make([]byte, 10), 0)

	assert.ErrorContains(t.T(), err, "unexpected block state")
	assert.True(t.T(), reader.blockQueue.IsEmpty())
}

func (t *BufferedReaderTest) TestReadAtFromDownloadedBlock() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	b, _ := reader.blockPool.Get()
	err = b.SetAbsStartOff(0)
	require.NoError(t.T(), err)
	content := []byte("hello world")
	_, err = b.Write(content)
	require.NoError(t.T(), err)
	b.NotifyReady(block.BlockStatus{State: block.BlockStateDownloaded})
	reader.blockQueue.Push(&blockQueueEntry{block: b, cancel: func() {}})
	buf := make([]byte, 5)

	resp, err := reader.ReadAt(t.ctx, buf, 0)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 5, resp.Size)
	assert.Equal(t.T(), "hello", string(buf))
	assert.False(t.T(), reader.blockQueue.IsEmpty())
}

func (t *BufferedReaderTest) TestReadAtExactlyToEndOfFile() {
	t.object.Size = uint64(testPrefetchBlockSizeBytes + 50) // 1 full block and 50 bytes
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 0 })).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == uint64(testPrefetchBlockSizeBytes) })).Return(createFakeReader(t.T(), 50), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 2*uint64(testPrefetchBlockSizeBytes) })).Return(createFakeReader(t.T(), 0), nil).Once()
	buf := make([]byte, t.object.Size)

	resp, err := reader.ReadAt(t.ctx, buf, 0)

	assert.ErrorIs(t.T(), err, io.EOF)
	assert.Equal(t.T(), int(t.object.Size), resp.Size)
	expectedContent := make([]byte, t.object.Size)
	for i := 0; i < int(t.object.Size); i++ {
		expectedContent[i] = byte('A' + (i % 26))
	}
	assert.Equal(t.T(), expectedContent, buf)
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestReadAtSucceedsWhenPrefetchFails() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 0 })).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == uint64(testPrefetchBlockSizeBytes) })).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 2*uint64(testPrefetchBlockSizeBytes) })).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	prefetchError := errors.New("prefetch failed")
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 3*uint64(testPrefetchBlockSizeBytes) })).Return(nil, prefetchError).Once()
	buf := make([]byte, testPrefetchBlockSizeBytes)

	resp, err := reader.ReadAt(t.ctx, buf, 0)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), int(testPrefetchBlockSizeBytes), resp.Size)
	assert.Equal(t.T(), 2, reader.blockQueue.Len())
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestReadAtSequentialReadAcrossBlocks() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 0 })).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == uint64(testPrefetchBlockSizeBytes) })).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 2*uint64(testPrefetchBlockSizeBytes) })).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 3*uint64(testPrefetchBlockSizeBytes) })).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 4*uint64(testPrefetchBlockSizeBytes) })).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	buf1 := make([]byte, testPrefetchBlockSizeBytes)

	_, err = reader.ReadAt(t.ctx, buf1, 0)

	require.NoError(t.T(), err)
	assert.Equal(t.T(), int64(0), reader.randomSeekCount)
	buf2 := make([]byte, testPrefetchBlockSizeBytes)

	_, err = reader.ReadAt(t.ctx, buf2, testPrefetchBlockSizeBytes)

	require.NoError(t.T(), err)
	assert.Equal(t.T(), int64(0), reader.randomSeekCount)
	t.bucket.AssertExpectations(t.T())
}
