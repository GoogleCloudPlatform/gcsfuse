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
			).Return(&fake.FakeReader{ReadCloser: io.NopCloser(bytes.NewReader(make([]byte, testPrefetchBlockSizeBytes)))}, nil).Once()

			err = reader.scheduleNextBlock(tc.urgent)
			require.NoError(t.T(), err)

			bqe := reader.blockQueue.Peek()
			blockstatus, err := bqe.block.AwaitReady(t.ctx)
			assert.Equal(t.T(), int64(1), reader.nextBlockIndexToPrefetch)
			require.NoError(t.T(), err)
			assert.Equal(t.T(), block.BlockStatus{State: block.BlockStateDownloaded}, blockstatus)
			assert.Equal(t.T(), initialBlockCount+1, reader.blockQueue.Len())
			assert.Equal(t.T(), int64(0), bqe.block.AbsStartOff())

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
	).Return(&fake.FakeReader{ReadCloser: io.NopCloser(bytes.NewReader(make([]byte, testPrefetchBlockSizeBytes)))}, nil).Once()

	err = reader.scheduleNextBlock(false)
	require.NoError(t.T(), err)

	bqe1 := reader.blockQueue.Pop()
	status1, err := bqe1.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), block.BlockStateDownloaded, status1.State)
	assert.Equal(t.T(), int64(0), bqe1.block.AbsStartOff())

	t.bucket.On("NewReaderWithReadHandle",
		mock.Anything,
		mock.AnythingOfType("*gcs.ReadObjectRequest"),
	).Return(&fake.FakeReader{ReadCloser: io.NopCloser(bytes.NewReader(make([]byte, testPrefetchBlockSizeBytes)))}, nil).Once()

	err = reader.scheduleNextBlock(false)
	require.NoError(t.T(), err)

	bqe2 := reader.blockQueue.Pop()
	status2, err := bqe2.block.AwaitReady(t.ctx)
	require.NoError(t.T(), err)
	assert.Equal(t.T(), block.BlockStateDownloaded, status2.State)
	assert.Equal(t.T(), int64(testPrefetchBlockSizeBytes), bqe2.block.AbsStartOff())

	assert.Equal(t.T(), int64(2), reader.nextBlockIndexToPrefetch)
	assert.Equal(t.T(), initialBlockCount, reader.blockQueue.Len())

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
			).Return(&fake.FakeReader{ReadCloser: io.NopCloser(bytes.NewReader(make([]byte, testPrefetchBlockSizeBytes)))}, nil).Once()

			b, err := reader.blockPool.Get()
			require.NoError(t.T(), err)

			err = reader.scheduleBlockWithIndex(b, tc.blockIndex, tc.urgent)
			require.NoError(t.T(), err)

			bqe := reader.blockQueue.Peek()
			blockstatus, err := bqe.block.AwaitReady(t.ctx)
			require.NoError(t.T(), err)
			assert.Equal(t.T(), block.BlockStatus{State: block.BlockStateDownloaded}, blockstatus)
			assert.Equal(t.T(), initialBlockCount+1, reader.blockQueue.Len())
			assert.Equal(t.T(), startOffset, bqe.block.AbsStartOff())

			t.bucket.AssertExpectations(t.T())
		})
	}
}
