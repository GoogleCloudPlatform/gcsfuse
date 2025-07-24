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
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err, "NewBufferedReader should not return error")
	b, err := reader.blockPool.Get()
	require.NoError(t.T(), err, "Failed to get block from pool")

	for range int(t.config.MaxPrefetchBlockCnt + 1) {
		reader.blockQueue.Push(&blockQueueEntry{
			block:  b,
			cancel: func() {},
		})
	}

	assert.Panics(t.T(), func() { reader.CheckInvariants() })
}

func (t *BufferedReaderTest) TestCheckInvariantsRandomSeekCountExceedsThreshold() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err, "NewBufferedReader should not return error")

	reader.randomSeekCount = t.config.RandomReadsThreshold + 1

	assert.Panics(t.T(), func() { reader.CheckInvariants() })
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

func (t *BufferedReaderTest) TestTotalBlockCountLogic() {
	testCases := []struct {
		name                   string
		objectSize             uint64
		prefetchBlockSizeBytes int64
		expectedBlockCount     int64
	}{
		{
			name:                   "object size is a multiple of block size",
			objectSize:             uint64(testPrefetchBlockSizeBytes * 5),
			prefetchBlockSizeBytes: testPrefetchBlockSizeBytes,
			expectedBlockCount:     5,
		},
		{
			name:                   "object size is not a multiple of block size",
			objectSize:             uint64(testPrefetchBlockSizeBytes*5 + 1),
			prefetchBlockSizeBytes: testPrefetchBlockSizeBytes,
			expectedBlockCount:     6,
		},
		{
			name:                   "object size is zero",
			objectSize:             0,
			prefetchBlockSizeBytes: testPrefetchBlockSizeBytes,
			expectedBlockCount:     0,
		},
		{
			name:                   "object size is less than block size",
			objectSize:             uint64(testPrefetchBlockSizeBytes - 1),
			prefetchBlockSizeBytes: testPrefetchBlockSizeBytes,
			expectedBlockCount:     1,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.object.Size = tc.objectSize
			t.config.PrefetchBlockSizeBytes = tc.prefetchBlockSizeBytes
			reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
			require.NoError(t.T(), err)

			blockCount := reader.totalBlockCount()

			assert.Equal(t.T(), tc.expectedBlockCount, blockCount)
		})
	}
}

func (t *BufferedReaderTest) TestFreshStart() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	currentOffset := int64(2 * testPrefetchBlockSizeBytes)
	expectedStartBlockIndex := currentOffset / testPrefetchBlockSizeBytes
	for i := 0; i < int(testInitialPrefetchBlockCnt+1); i++ {
		t.bucket.On("NewReaderWithReadHandle",
			mock.Anything,
			mock.AnythingOfType("*gcs.ReadObjectRequest"),
		).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	}

	err = reader.freshStart(currentOffset)

	require.NoError(t.T(), err)
	assert.Equal(t.T(), expectedStartBlockIndex+testInitialPrefetchBlockCnt+1, reader.nextBlockIndexToPrefetch)
	assert.Equal(t.T(), testInitialPrefetchBlockCnt*testPrefetchMultiplier, reader.numPrefetchBlocks)
	assert.Equal(t.T(), int(testInitialPrefetchBlockCnt+1), reader.blockQueue.Len())
	for i := int64(0); i < testInitialPrefetchBlockCnt+1; i++ {
		bqe := reader.blockQueue.Pop()
		expectedOffset := (expectedStartBlockIndex + i) * testPrefetchBlockSizeBytes
		assert.Equal(t.T(), expectedOffset, bqe.block.AbsStartOff())
		status, err := bqe.block.AwaitReady(t.ctx)
		require.NoError(t.T(), err)
		assert.Equal(t.T(), block.BlockStateDownloaded, status.State)
	}
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestFreshStartWhenInitialCountGreaterThanMax() {
	t.config.InitialPrefetchBlockCnt = testMaxPrefetchBlockCnt + 5
	t.object.Size = uint64((testMaxPrefetchBlockCnt + 5) * testPrefetchBlockSizeBytes)
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	// With InitialPrefetchBlockCnt > MaxPrefetchBlockCnt, freshStart will schedule
	// 1 urgent block, then prefetch will schedule MaxPrefetchBlockCnt - 1 blocks.
	expectedTotalPrefetchCount := testMaxPrefetchBlockCnt
	for i := 0; i < int(expectedTotalPrefetchCount); i++ {
		t.bucket.On("NewReaderWithReadHandle",
			mock.Anything,
			mock.AnythingOfType("*gcs.ReadObjectRequest"),
		).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	}

	err = reader.freshStart(0)

	require.NoError(t.T(), err)
	assert.Equal(t.T(), expectedTotalPrefetchCount, reader.nextBlockIndexToPrefetch)
	assert.Equal(t.T(), testMaxPrefetchBlockCnt, reader.numPrefetchBlocks)
	assert.Equal(t.T(), int(expectedTotalPrefetchCount), reader.blockQueue.Len())
	for i := 0; i < int(expectedTotalPrefetchCount); i++ {
		bqe := reader.blockQueue.Pop()
		status, err := bqe.block.AwaitReady(t.ctx)
		require.NoError(t.T(), err)
		assert.Equal(t.T(), block.BlockStateDownloaded, status.State)
	}
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestFreshStartStopsAtObjectEnd() {
	t.object.Size = uint64(3 * testPrefetchBlockSizeBytes)
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	currentOffset := int64(2 * testPrefetchBlockSizeBytes)
	expectedPrefetchCount := 1
	t.bucket.On("NewReaderWithReadHandle",
		mock.Anything,
		mock.AnythingOfType("*gcs.ReadObjectRequest"),
	).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()

	err = reader.freshStart(currentOffset)

	require.NoError(t.T(), err)
	assert.Equal(t.T(), int64(3), reader.nextBlockIndexToPrefetch)
	assert.Equal(t.T(), testInitialPrefetchBlockCnt*testPrefetchMultiplier, reader.numPrefetchBlocks)
	assert.Equal(t.T(), expectedPrefetchCount, reader.blockQueue.Len())
	bqe := reader.blockQueue.Pop()
	assert.Equal(t.T(), currentOffset, bqe.block.AbsStartOff())
	status, err := bqe.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), block.BlockStateDownloaded, status.State)
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestPrefetch() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	for i := 0; i < int(testInitialPrefetchBlockCnt); i++ {
		t.bucket.On("NewReaderWithReadHandle",
			mock.Anything,
			mock.AnythingOfType("*gcs.ReadObjectRequest"),
		).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	}

	err = reader.prefetch()

	require.NoError(t.T(), err)
	assert.Equal(t.T(), testInitialPrefetchBlockCnt, reader.nextBlockIndexToPrefetch)
	assert.Equal(t.T(), testInitialPrefetchBlockCnt*testPrefetchMultiplier, reader.numPrefetchBlocks)
	assert.Equal(t.T(), int(testInitialPrefetchBlockCnt), reader.blockQueue.Len())
	// Wait for all downloads to complete.
	for i := int64(0); i < testInitialPrefetchBlockCnt; i++ {
		bqe := reader.blockQueue.Pop()
		status, err := bqe.block.AwaitReady(t.ctx)
		require.NoError(t.T(), err)
		assert.Equal(t.T(), block.BlockStateDownloaded, status.State)
	}
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestPrefetchWithMultiplicativeIncrease() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	for i := 0; i < int(testInitialPrefetchBlockCnt); i++ {
		t.bucket.On("NewReaderWithReadHandle",
			mock.Anything,
			mock.AnythingOfType("*gcs.ReadObjectRequest"),
		).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	}
	err = reader.prefetch()
	require.NoError(t.T(), err)
	// Wait for first prefetch to complete and drain the queue.
	for i := int64(0); i < testInitialPrefetchBlockCnt; i++ {
		bqe := reader.blockQueue.Pop()
		_, err := bqe.block.AwaitReady(t.ctx)
		require.NoError(t.T(), err)
	}
	expectedNextPrefetchCount := testInitialPrefetchBlockCnt * testPrefetchMultiplier
	for i := 0; i < int(expectedNextPrefetchCount); i++ {
		t.bucket.On("NewReaderWithReadHandle",
			mock.Anything,
			mock.AnythingOfType("*gcs.ReadObjectRequest"),
		).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	}

	err = reader.prefetch()

	require.NoError(t.T(), err)
	assert.Equal(t.T(), testInitialPrefetchBlockCnt+expectedNextPrefetchCount, reader.nextBlockIndexToPrefetch)
	assert.Equal(t.T(), expectedNextPrefetchCount*testPrefetchMultiplier, reader.numPrefetchBlocks)
	assert.Equal(t.T(), int(expectedNextPrefetchCount), reader.blockQueue.Len())
	// Wait for second prefetch to complete.
	for i := int64(0); i < expectedNextPrefetchCount; i++ {
		bqe := reader.blockQueue.Pop()
		status, err := bqe.block.AwaitReady(t.ctx)
		require.NoError(t.T(), err)
		assert.Equal(t.T(), block.BlockStateDownloaded, status.State)
	}
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestPrefetchWhenQueueIsFull() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	for i := int64(0); i < testMaxPrefetchBlockCnt; i++ {
		b, err := reader.blockPool.Get()
		require.NoError(t.T(), err)
		reader.blockQueue.Push(&blockQueueEntry{block: b})
	}

	err = reader.prefetch()

	require.NoError(t.T(), err)
	assert.Equal(t.T(), int64(0), reader.nextBlockIndexToPrefetch)
	assert.Equal(t.T(), int(testMaxPrefetchBlockCnt), reader.blockQueue.Len())
	assert.Equal(t.T(), testInitialPrefetchBlockCnt, reader.numPrefetchBlocks)
	t.bucket.AssertNotCalled(t.T(), "NewReaderWithReadHandle", mock.Anything, mock.Anything)
}

func (t *BufferedReaderTest) TestPrefetchWhenQueueIsPartiallyFull() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	numAlreadyInQueue := 5
	for i := 0; i < numAlreadyInQueue; i++ {
		b, err := reader.blockPool.Get()
		require.NoError(t.T(), err)
		reader.blockQueue.Push(&blockQueueEntry{block: b})
	}
	expectedPrefetchCount := int(min(testInitialPrefetchBlockCnt, testMaxPrefetchBlockCnt-int64(numAlreadyInQueue)))
	for i := 0; i < expectedPrefetchCount; i++ {
		t.bucket.On("NewReaderWithReadHandle",
			mock.Anything,
			mock.AnythingOfType("*gcs.ReadObjectRequest"),
		).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	}

	err = reader.prefetch()

	require.NoError(t.T(), err)
	assert.Equal(t.T(), int64(expectedPrefetchCount), reader.nextBlockIndexToPrefetch)
	assert.Equal(t.T(), numAlreadyInQueue+expectedPrefetchCount, reader.blockQueue.Len())
	assert.Equal(t.T(), testInitialPrefetchBlockCnt*testPrefetchMultiplier, reader.numPrefetchBlocks)
	// Wait for the newly scheduled downloads to complete. The old blocks are
	// dummies and not downloaded, so we pop them first.
	for i := 0; i < numAlreadyInQueue; i++ {
		bqe := reader.blockQueue.Pop()
		reader.blockPool.Release(bqe.block)
	}
	for i := 0; i < expectedPrefetchCount; i++ {
		bqe := reader.blockQueue.Pop()
		_, err := bqe.block.AwaitReady(t.ctx)
		require.NoError(t.T(), err)
	}
	t.bucket.AssertExpectations(t.T())
}

func (t *BufferedReaderTest) TestPrefetchLimitedByAvailableSlots() {
	reader, err := NewBufferedReader(t.object, t.bucket, t.config, t.globalMaxBlocksSem, t.workerPool, t.metricHandle)
	require.NoError(t.T(), err)
	reader.numPrefetchBlocks = 4
	numAlreadyInQueue := 7
	for i := 0; i < numAlreadyInQueue; i++ {
		b, err := reader.blockPool.Get()
		require.NoError(t.T(), err)
		reader.blockQueue.Push(&blockQueueEntry{block: b})
	}
	expectedPrefetchCount := 3
	for i := 0; i < expectedPrefetchCount; i++ {
		t.bucket.On("NewReaderWithReadHandle",
			mock.Anything,
			mock.AnythingOfType("*gcs.ReadObjectRequest"),
		).Return(createFakeReader(t.T(), int(testPrefetchBlockSizeBytes)), nil).Once()
	}

	err = reader.prefetch()

	require.NoError(t.T(), err)
	assert.Equal(t.T(), int64(expectedPrefetchCount), reader.nextBlockIndexToPrefetch)
	assert.Equal(t.T(), numAlreadyInQueue+expectedPrefetchCount, reader.blockQueue.Len())
	assert.Equal(t.T(), (testInitialPrefetchBlockCnt*testPrefetchMultiplier)*testPrefetchMultiplier, reader.numPrefetchBlocks)
	for i := 0; i < numAlreadyInQueue; i++ {
		bqe := reader.blockQueue.Pop()
		reader.blockPool.Release(bqe.block)
	}
	for i := 0; i < expectedPrefetchCount; i++ {
		bqe := reader.blockQueue.Pop()
		_, err := bqe.block.AwaitReady(t.ctx)
		require.NoError(t.T(), err)
	}
	t.bucket.AssertExpectations(t.T())
}
