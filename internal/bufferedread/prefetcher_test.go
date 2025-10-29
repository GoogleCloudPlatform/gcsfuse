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
	"context"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/common"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/block"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/workerpool"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/semaphore"
)

type PrefetcherTest struct {
	suite.Suite
	ctx          context.Context
	object       *gcs.MinObject
	bucket       *storage.TestifyMockBucket
	config       *BufferedReadConfig
	workerPool   workerpool.WorkerPool
	metricHandle metrics.MetricHandle
	blockPool    *block.GenBlockPool[block.PrefetchBlock]
	blockQueue   common.Queue[*blockQueueEntry]
	prefetcher   *prefetcher
}

func TestPrefetcherTestSuite(t *testing.T) {
	suite.Run(t, new(PrefetcherTest))
}

func (t *PrefetcherTest) SetupTest() {
	t.object = &gcs.MinObject{
		Name:       "test_object",
		Size:       8192,
		Generation: 1234567890,
	}
	t.bucket = new(storage.TestifyMockBucket)
	globalMaxBlocksSem := semaphore.NewWeighted(testGlobalMaxBlocks)
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
	t.blockPool, err = block.NewPrefetchBlockPool(t.config.PrefetchBlockSizeBytes, t.config.MaxPrefetchBlockCnt, t.config.MinBlocksPerHandle, globalMaxBlocksSem)
	require.NoError(t.T(), err)
	t.blockQueue = common.NewLinkedListQueue[*blockQueueEntry]()
	t.prefetcher = newPrefetcher(&prefetcherOptions{
		object:       t.object,
		bucket:       t.bucket,
		config:       t.config,
		pool:         t.blockPool,
		workerPool:   t.workerPool,
		queue:        t.blockQueue,
		metricHandle: t.metricHandle,
		readerCtx:    t.ctx,
	})
}

func (t *PrefetcherTest) TearDownTest() {
	t.workerPool.Stop()
}

func (t *PrefetcherTest) TestNewPrefetcher() {
	assert.NotNil(t.T(), t.prefetcher, "prefetcher should not be nil")
	assert.Equal(t.T(), t.object, t.prefetcher.object, "object should match")
	assert.Equal(t.T(), t.bucket, t.prefetcher.bucket, "bucket should match")
	assert.Equal(t.T(), t.config, t.prefetcher.config, "config should match")
	assert.Equal(t.T(), t.blockPool, t.prefetcher.pool, "pool should match")
	assert.Equal(t.T(), t.workerPool, t.prefetcher.workerPool, "workerPool should match")
	assert.Equal(t.T(), t.blockQueue, t.prefetcher.queue, "queue should match")
	assert.Equal(t.T(), t.metricHandle, t.prefetcher.metricHandle, "metricHandle should match")
	assert.Equal(t.T(), t.ctx, t.prefetcher.readerCtx, "readerCtx should match")
	assert.Nil(t.T(), t.prefetcher.readHandle, "readHandle should be nil initially")
	assert.Equal(t.T(), int64(0), t.prefetcher.nextBlockIndexToPrefetch, "nextBlockIndexToPrefetch should be 0")
	assert.Equal(t.T(), t.config.InitialPrefetchBlockCnt, t.prefetcher.numPrefetchBlocks, "numPrefetchBlocks should be initial count")
	assert.Equal(t.T(), int64(defaultPrefetchMultiplier), t.prefetcher.prefetchMultiplier, "prefetchMultiplier should be default")
}

func (t *PrefetcherTest) TestScheduleNextBlock() {
	testCases := []struct {
		name   string
		urgent bool
	}{
		{name: "non-urgent", urgent: false},
		{name: "urgent", urgent: true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.SetupTest()
			initialBlockCount := t.blockQueue.Len()
			startOffset := int64(0)
			t.bucket.On("NewReaderWithReadHandle",
				mock.Anything,
				mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == uint64(startOffset) }),
			).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), startOffset), nil).Once()

			err := t.prefetcher.scheduleNextBlock(tc.urgent)

			require.NoError(t.T(), err)
			bqe := t.blockQueue.Peek()
			assert.Equal(t.T(), int64(1), t.prefetcher.nextBlockIndexToPrefetch)
			status, err := bqe.block.AwaitReady(t.ctx)
			require.NoError(t.T(), err)
			assert.Equal(t.T(), block.BlockStateDownloaded, status.State)
			assert.Equal(t.T(), initialBlockCount+1, t.blockQueue.Len())
			assert.Equal(t.T(), int64(0), bqe.block.AbsStartOff())
			assertBlockContent(t.T(), bqe.block, bqe.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
			t.bucket.AssertExpectations(t.T())
		})
	}
}

func (t *PrefetcherTest) TestScheduleNextBlockSuccessive() {
	initialBlockCount := t.blockQueue.Len()
	startOffset1 := int64(0)
	t.bucket.On("NewReaderWithReadHandle",
		mock.Anything,
		mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == uint64(startOffset1) }),
	).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), startOffset1), nil).Once()
	err := t.prefetcher.scheduleNextBlock(false)
	require.NoError(t.T(), err)
	bqe1 := t.blockQueue.Pop()
	assert.Equal(t.T(), int64(1), t.prefetcher.nextBlockIndexToPrefetch)
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

	err = t.prefetcher.scheduleNextBlock(false)

	require.NoError(t.T(), err)
	bqe2 := t.blockQueue.Pop()
	assert.Equal(t.T(), int64(2), t.prefetcher.nextBlockIndexToPrefetch)
	status2, err := bqe2.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), block.BlockStateDownloaded, status2.State)
	assert.Equal(t.T(), int64(testPrefetchBlockSizeBytes), bqe2.block.AbsStartOff())
	assert.Equal(t.T(), int64(2), t.prefetcher.nextBlockIndexToPrefetch)
	assert.Equal(t.T(), initialBlockCount, t.blockQueue.Len())
	assertBlockContent(t.T(), bqe2.block, bqe2.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	t.bucket.AssertExpectations(t.T())
}

func (t *PrefetcherTest) TestScheduleBlockWithIndex() {
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
			t.SetupTest()
			initialBlockCount := t.blockQueue.Len()
			startOffset := tc.blockIndex * t.config.PrefetchBlockSizeBytes
			t.bucket.On("NewReaderWithReadHandle",
				mock.Anything,
				mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == uint64(startOffset) }),
			).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), startOffset), nil).Once()
			b, err := t.blockPool.Get()
			require.NoError(t.T(), err)

			err = t.prefetcher.scheduleBlockWithIndex(b, tc.blockIndex, tc.urgent)

			require.NoError(t.T(), err)
			bqe := t.blockQueue.Peek()
			status, err := bqe.block.AwaitReady(t.ctx)
			require.NoError(t.T(), err)
			assert.Equal(t.T(), block.BlockStateDownloaded, status.State)
			assert.Equal(t.T(), initialBlockCount+1, t.blockQueue.Len())
			assert.Equal(t.T(), startOffset, bqe.block.AbsStartOff())
			assertBlockContent(t.T(), bqe.block, bqe.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
			t.bucket.AssertExpectations(t.T())
		})
	}
}

func (t *PrefetcherTest) TestFreshStart() {
	currentOffset := int64(2048) // Start prefetching from offset 2048 (block 2).
	// freshStart schedules 1 urgent block and 2 initial prefetch blocks, totaling 3 blocks.
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 2048 })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 2048), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 3072 })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 3072), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 4096 })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 4096), nil).Once()

	err := t.prefetcher.freshStart(currentOffset)

	require.NoError(t.T(), err)
	// nextBlockIndexToPrefetch should be current block index (2) + scheduled blocks (3).
	assert.Equal(t.T(), int64(5), t.prefetcher.nextBlockIndexToPrefetch)
	// numPrefetchBlocks for the next prefetch should be initialPrefetchBlockCnt (2) * prefetchMultiplier (2).
	assert.Equal(t.T(), int64(4), t.prefetcher.numPrefetchBlocks)
	assert.Equal(t.T(), 3, t.blockQueue.Len())
	// Pop and verify the downloaded blocks.
	bqe1 := t.blockQueue.Pop()
	assert.Equal(t.T(), int64(2048), bqe1.block.AbsStartOff())
	status1, err1 := bqe1.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err1)
	assert.Equal(t.T(), block.BlockStateDownloaded, status1.State)
	assertBlockContent(t.T(), bqe1.block, bqe1.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	bqe2 := t.blockQueue.Pop()
	assert.Equal(t.T(), int64(3072), bqe2.block.AbsStartOff())
	status2, err2 := bqe2.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err2)
	assert.Equal(t.T(), block.BlockStateDownloaded, status2.State)
	assertBlockContent(t.T(), bqe2.block, bqe2.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	bqe3 := t.blockQueue.Pop()
	assert.Equal(t.T(), int64(4096), bqe3.block.AbsStartOff())
	status3, err3 := bqe3.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err3)
	assert.Equal(t.T(), block.BlockStateDownloaded, status3.State)
	assertBlockContent(t.T(), bqe3.block, bqe3.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	t.bucket.AssertExpectations(t.T())
}

func (t *PrefetcherTest) TestFreshStartWithNonBlockAlignedOffset() {
	currentOffset := int64(2500) // Start prefetching from offset 2500 (inside block 2).
	// freshStart should start prefetching from block 2. It schedules 1 urgent block
	// and 2 initial prefetch blocks, totaling 3 blocks.
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 2048 })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 2048), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 3072 })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 3072), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 4096 })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 4096), nil).Once()

	err := t.prefetcher.freshStart(currentOffset)

	require.NoError(t.T(), err)
	// nextBlockIndexToPrefetch should be current block index (2) + scheduled blocks (3).
	assert.Equal(t.T(), int64(5), t.prefetcher.nextBlockIndexToPrefetch)
	// numPrefetchBlocks for the next prefetch should be initialPrefetchBlockCnt (2) * prefetchMultiplier (2).
	assert.Equal(t.T(), int64(4), t.prefetcher.numPrefetchBlocks)
	assert.Equal(t.T(), 3, t.blockQueue.Len())
	// Pop and verify the downloaded blocks.
	bqe1 := t.blockQueue.Pop()
	assert.Equal(t.T(), int64(2048), bqe1.block.AbsStartOff())
	status1, err1 := bqe1.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err1)
	assert.Equal(t.T(), block.BlockStateDownloaded, status1.State)
	assertBlockContent(t.T(), bqe1.block, bqe1.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	bqe2 := t.blockQueue.Pop()
	assert.Equal(t.T(), int64(3072), bqe2.block.AbsStartOff())
	status2, err2 := bqe2.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err2)
	assert.Equal(t.T(), block.BlockStateDownloaded, status2.State)
	assertBlockContent(t.T(), bqe2.block, bqe2.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	bqe3 := t.blockQueue.Pop()
	assert.Equal(t.T(), int64(4096), bqe3.block.AbsStartOff())
	status3, err3 := bqe3.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err3)
	assert.Equal(t.T(), block.BlockStateDownloaded, status3.State)
	assertBlockContent(t.T(), bqe3.block, bqe3.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	t.bucket.AssertExpectations(t.T())
}

func (t *PrefetcherTest) TestFreshStartWhenInitialCountGreaterThanMax() {
	t.config.MaxPrefetchBlockCnt = 3
	t.config.InitialPrefetchBlockCnt = 4
	t.object.Size = 4096
	// freshStart schedules 1 urgent block and 2 prefetch blocks (InitialPrefetchBlockCnt capped by MaxPrefetchBlockCnt).
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 0 })).Return(createFakeReaderWithOffset(t.T(), 1024, 0), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 1024 })).Return(createFakeReaderWithOffset(t.T(), 1024, 1024), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 2048 })).Return(createFakeReaderWithOffset(t.T(), 1024, 2048), nil).Once()

	err := t.prefetcher.freshStart(0)

	require.NoError(t.T(), err)
	// nextBlockIndexToPrefetch should be start block index (0) + scheduled blocks (3).
	assert.Equal(t.T(), int64(3), t.prefetcher.nextBlockIndexToPrefetch)
	// numPrefetchBlocks for next prefetch should be capped at MaxPrefetchBlockCnt (3).
	assert.Equal(t.T(), int64(3), t.prefetcher.numPrefetchBlocks)
	assert.Equal(t.T(), 3, t.blockQueue.Len())
	// Pop and verify blocks are downloaded.
	bqe1 := t.blockQueue.Pop()
	status1, err1 := bqe1.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err1)
	assert.Equal(t.T(), block.BlockStateDownloaded, status1.State)
	assertBlockContent(t.T(), bqe1.block, bqe1.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	bqe2 := t.blockQueue.Pop()
	status2, err2 := bqe2.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err2)
	assert.Equal(t.T(), block.BlockStateDownloaded, status2.State)
	assertBlockContent(t.T(), bqe2.block, bqe2.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	bqe3 := t.blockQueue.Pop()
	status3, err3 := bqe3.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err3)
	assert.Equal(t.T(), block.BlockStateDownloaded, status3.State)
	assertBlockContent(t.T(), bqe3.block, bqe3.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	t.bucket.AssertExpectations(t.T())
}

func (t *PrefetcherTest) TestFreshStartStopsAtObjectEnd() {
	t.object.Size = 4000         // Object size is 3 blocks + a partial block.
	currentOffset := int64(2048) // Start from block 2.
	// freshStart schedules 1 urgent block (block 2) and 1 prefetch block (block 3 - partial).
	// The object ends after block 3, so only these 2 blocks are scheduled.
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 2*uint64(testPrefetchBlockSizeBytes) })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 2*testPrefetchBlockSizeBytes), nil).Once()
	partialBlockSize := int(int64(t.object.Size) - (3 * testPrefetchBlockSizeBytes))
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 3*uint64(testPrefetchBlockSizeBytes) })).Return(createFakeReaderWithOffset(t.T(), partialBlockSize, 3*testPrefetchBlockSizeBytes), nil).Once()

	err := t.prefetcher.freshStart(currentOffset)

	require.NoError(t.T(), err)
	// nextBlockIndexToPrefetch should be start block index (2) + scheduled blocks (2).
	assert.Equal(t.T(), int64(4), t.prefetcher.nextBlockIndexToPrefetch)
	// numPrefetchBlocks for the next prefetch should be initialPrefetchBlockCnt (2) * prefetchMultiplier (2).
	assert.Equal(t.T(), int64(4), t.prefetcher.numPrefetchBlocks)
	assert.Equal(t.T(), 2, t.blockQueue.Len())
	// Verify block 2.
	bqe1 := t.blockQueue.Pop()
	assert.Equal(t.T(), int64(2048), bqe1.block.AbsStartOff())
	status1, err1 := bqe1.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err1)
	assert.Equal(t.T(), block.BlockStateDownloaded, status1.State)
	assertBlockContent(t.T(), bqe1.block, bqe1.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	// Verify block 3.
	bqe2 := t.blockQueue.Pop()
	assert.Equal(t.T(), int64(3072), bqe2.block.AbsStartOff())
	status2, err2 := bqe2.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err2)
	assert.Equal(t.T(), block.BlockStateDownloaded, status2.State)
	// Assert content for the partial block.
	assertBlockContent(t.T(), bqe2.block, bqe2.block.AbsStartOff(), partialBlockSize)
	t.bucket.AssertExpectations(t.T())
}

func (t *PrefetcherTest) TestPrefetch() {
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 0 })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 0), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 1024 })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 1024), nil).Once()

	err := t.prefetcher.prefetch()

	require.NoError(t.T(), err)
	// nextBlockIndexToPrefetch should be start block index (0) + initialPrefetchBlockCnt (2).
	assert.Equal(t.T(), int64(2), t.prefetcher.nextBlockIndexToPrefetch)
	// numPrefetchBlocks for the next prefetch should be initialPrefetchBlockCnt (2) * prefetchMultiplier (2).
	assert.Equal(t.T(), int64(4), t.prefetcher.numPrefetchBlocks)
	assert.Equal(t.T(), 2, t.blockQueue.Len())
	// Wait for all downloads to complete.
	bqe1 := t.blockQueue.Pop()
	status1, err1 := bqe1.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err1)
	assert.Equal(t.T(), block.BlockStateDownloaded, status1.State)
	assertBlockContent(t.T(), bqe1.block, bqe1.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	bqe2 := t.blockQueue.Pop()
	status2, err2 := bqe2.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err2)
	assert.Equal(t.T(), block.BlockStateDownloaded, status2.State)
	assertBlockContent(t.T(), bqe2.block, bqe2.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	t.bucket.AssertExpectations(t.T())
}

func (t *PrefetcherTest) TestPrefetchWithMultiplicativeIncrease() {
	t.config.InitialPrefetchBlockCnt = 1
	t.prefetcher.numPrefetchBlocks = 1
	// First prefetch schedules 1 block.
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 0 })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 0), nil).Once()
	err := t.prefetcher.prefetch()
	require.NoError(t.T(), err)
	// Wait for the first prefetch to complete and drain the queue.
	bqe1 := t.blockQueue.Pop()
	status1, err1 := bqe1.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err1)
	assert.Equal(t.T(), block.BlockStateDownloaded, status1.State)
	assertBlockContent(t.T(), bqe1.block, bqe1.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	// Second prefetch should schedule 2 blocks due to multiplicative increase.
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 1024 })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 1024), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 2048 })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 2048), nil).Once()

	err = t.prefetcher.prefetch()

	require.NoError(t.T(), err)
	// nextBlockIndexToPrefetch should be blocks from first prefetch (1) + blocks from second prefetch (2).
	assert.Equal(t.T(), int64(3), t.prefetcher.nextBlockIndexToPrefetch)
	// numPrefetchBlocks for the next prefetch should be numPrefetchBlocks from previous prefetch (2) * prefetchMultiplier (2).
	assert.Equal(t.T(), int64(4), t.prefetcher.numPrefetchBlocks)
	assert.Equal(t.T(), 2, t.blockQueue.Len())
	// Wait for the second prefetch to complete.
	bqe2 := t.blockQueue.Pop()
	status2, err2 := bqe2.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err2)
	assert.Equal(t.T(), block.BlockStateDownloaded, status2.State)
	assertBlockContent(t.T(), bqe2.block, bqe2.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	bqe3 := t.blockQueue.Pop()
	status3, err3 := bqe3.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err3)
	assert.Equal(t.T(), block.BlockStateDownloaded, status3.State)
	assertBlockContent(t.T(), bqe3.block, bqe3.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	t.bucket.AssertExpectations(t.T())
}

func (t *PrefetcherTest) TestPrefetchWhenQueueIsFull() {
	t.config.MaxPrefetchBlockCnt = 2
	b, err := t.blockPool.Get()
	require.NoError(t.T(), err)
	// Fill the block queue to its maximum capacity.
	t.blockQueue.Push(&blockQueueEntry{block: b})
	t.blockQueue.Push(&blockQueueEntry{block: b})

	err = t.prefetcher.prefetch()

	require.NoError(t.T(), err)
	// No new blocks should be prefetched, so the index remains 0.
	assert.Equal(t.T(), int64(0), t.prefetcher.nextBlockIndexToPrefetch)
	// The queue length should remain at MaxPrefetchBlockCnt.
	assert.Equal(t.T(), 2, t.blockQueue.Len())
	// numPrefetchBlocks should remain at its default/current value (2 in this case, due to InitialPrefetchBlockCnt).
	assert.Equal(t.T(), int64(2), t.prefetcher.numPrefetchBlocks)
}

func (t *PrefetcherTest) TestPrefetchWhenQueueIsPartiallyFull() {
	t.config.MaxPrefetchBlockCnt = 4
	b, err := t.blockPool.Get()
	require.NoError(t.T(), err)
	t.blockQueue.Push(&blockQueueEntry{block: b})
	t.blockQueue.Push(&blockQueueEntry{block: b})
	// blockCountToPrefetch = min(numPrefetchBlocks (2), availableSlots (2)) = 2.
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 0 })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 0), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 1024 })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 1024), nil).Once()

	err = t.prefetcher.prefetch()

	require.NoError(t.T(), err)
	// nextBlockIndexToPrefetch should be the number of scheduled blocks (2).
	assert.Equal(t.T(), int64(2), t.prefetcher.nextBlockIndexToPrefetch)
	// blockQueue.Len() should be already in queue (2) + newly scheduled blocks (2).
	assert.Equal(t.T(), 4, t.blockQueue.Len())
	// numPrefetchBlocks for the next prefetch should be previous numPrefetchBlocks (2) * prefetchMultiplier (2).
	assert.Equal(t.T(), int64(4), t.prefetcher.numPrefetchBlocks)
	// Wait for the newly scheduled downloads to complete. The old blocks are dummies.
	bqe1 := t.blockQueue.Pop()
	t.blockPool.Release(bqe1.block)
	bqe2 := t.blockQueue.Pop()
	t.blockPool.Release(bqe2.block)
	bqe3 := t.blockQueue.Pop()
	status3, err3 := bqe3.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err3)
	assert.Equal(t.T(), block.BlockStateDownloaded, status3.State)
	assertBlockContent(t.T(), bqe3.block, bqe3.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	bqe4 := t.blockQueue.Pop()
	status4, err4 := bqe4.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err4)
	assert.Equal(t.T(), block.BlockStateDownloaded, status4.State)
	assertBlockContent(t.T(), bqe4.block, bqe4.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	t.bucket.AssertExpectations(t.T())
}

func (t *PrefetcherTest) TestPrefetchLimitedByAvailableSlots() {
	t.config.MaxPrefetchBlockCnt = 4
	t.prefetcher.numPrefetchBlocks = 4
	b, err := t.blockPool.Get()
	require.NoError(t.T(), err)
	t.blockQueue.Push(&blockQueueEntry{block: b})
	t.blockQueue.Push(&blockQueueEntry{block: b})
	t.blockQueue.Push(&blockQueueEntry{block: b})
	// blockCountToPrefetch = min(numPrefetchBlocks (4), availableSlots (1)) = 1.
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 0 })).Return(createFakeReaderWithOffset(t.T(), int(testPrefetchBlockSizeBytes), 0), nil).Once()

	err = t.prefetcher.prefetch()

	require.NoError(t.T(), err)
	// nextBlockIndexToPrefetch should be the number of scheduled blocks (1).
	assert.Equal(t.T(), int64(1), t.prefetcher.nextBlockIndexToPrefetch)
	// blockQueue.Len() should be already in queue (3) + newly scheduled blocks (1).
	assert.Equal(t.T(), 4, t.blockQueue.Len())
	// numPrefetchBlocks for the next prefetch should be current numPrefetchBlocks (4) * prefetchMultiplier (2) = 8,
	// but capped at MaxPrefetchBlockCnt (4).
	assert.Equal(t.T(), int64(4), t.prefetcher.numPrefetchBlocks)
	// Release dummy blocks and wait for the newly scheduled download to complete.
	bqe1 := t.blockQueue.Pop()
	t.blockPool.Release(bqe1.block)
	bqe2 := t.blockQueue.Pop()
	t.blockPool.Release(bqe2.block)
	bqe3 := t.blockQueue.Pop()
	t.blockPool.Release(bqe3.block)
	bqe4 := t.blockQueue.Pop()
	status4, err4 := bqe4.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err4)
	assert.Equal(t.T(), block.BlockStateDownloaded, status4.State)
	assertBlockContent(t.T(), bqe4.block, bqe4.block.AbsStartOff(), int(testPrefetchBlockSizeBytes))
	t.bucket.AssertExpectations(t.T())
}

func (t *PrefetcherTest) TestPrefetchStopsWhenPoolIsExhausted() {
	// Configure a small pool that will be exhausted, to test the case where
	// prefetching is not possible.
	t.config.MaxPrefetchBlockCnt = 4
	t.config.InitialPrefetchBlockCnt = 2
	// The global semaphore only has enough permits for the reserved blocks.
	globalMaxBlocksSem := semaphore.NewWeighted(2)
	var err error
	t.blockPool, err = block.NewPrefetchBlockPool(t.config.PrefetchBlockSizeBytes, t.config.MaxPrefetchBlockCnt, t.config.MinBlocksPerHandle, globalMaxBlocksSem)
	require.NoError(t.T(), err)
	t.prefetcher.pool = t.blockPool
	// At this point, NewPrefetchBlockPool has acquired 2 permits for its reserved blocks.
	// The global semaphore is now empty.
	// The first prefetch() call will succeed by allocating the 2 reserved blocks.
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 0 })).Return(createFakeReaderWithOffset(t.T(), 1024, 0), nil).Once()
	t.bucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(r *gcs.ReadObjectRequest) bool { return r.Range.Start == 1024 })).Return(createFakeReaderWithOffset(t.T(), 1024, 1024), nil).Once()
	err = t.prefetcher.prefetch()
	require.NoError(t.T(), err)
	require.Equal(t.T(), 2, t.blockQueue.Len())
	assert.Equal(t.T(), int64(4), t.prefetcher.numPrefetchBlocks, "numPrefetchBlocks should be multiplied after successful prefetch")
	// The pool has now created 2 blocks (totalBlocks=2), which is its max (maxBlocks=2).
	// To simulate a state where the pool is exhausted, we drain the queue without
	// releasing the blocks back to the pool's free channel. We must wait for the
	// downloads to complete before proceeding.
	bqe1 := t.blockQueue.Pop()
	_, _ = bqe1.block.AwaitReady(t.ctx)
	bqe2 := t.blockQueue.Pop()
	_, _ = bqe2.block.AwaitReady(t.ctx)
	// Now the blockQueue and freeBlocksCh are empty, but totalBlocks is at its limit.

	// The next prefetch call should attempt to schedule blocks but fail to get
	// any from the exhausted pool. It should not return an error.
	err = t.prefetcher.prefetch()

	require.NoError(t.T(), err, "prefetch should handle block unavailability gracefully")
	assert.Equal(t.T(), 0, t.blockQueue.Len(), "No new blocks should have been scheduled")
	assert.Equal(t.T(), int64(2), t.prefetcher.nextBlockIndexToPrefetch, "The index should not have advanced")
	assert.Equal(t.T(), int64(4), t.prefetcher.numPrefetchBlocks, "numPrefetchBlocks should not increase when prefetch is incomplete")
	t.bucket.AssertExpectations(t.T())
}
