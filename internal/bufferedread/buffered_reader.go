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
	"errors"
	"fmt"

	"github.com/googlecloudplatform/gcsfuse/v3/common"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/block"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/workerpool"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"golang.org/x/sync/semaphore"
)

// ErrPrefetchBlockNotAvailable is returned when a block cannot be
// acquired from the pool for prefetching. This can be used by callers to
// implement a fallback mechanism, e.g. falling back to a direct GCS read.
var ErrPrefetchBlockNotAvailable = errors.New("block for prefetching not available")

type BufferedReadConfig struct {
	MaxPrefetchBlockCnt     int64 // Maximum number of blocks that can be prefetched.
	PrefetchBlockSizeBytes  int64 // Size of each block to be prefetched.
	InitialPrefetchBlockCnt int64 // Number of blocks to prefetch initially.
	PrefetchMultiplier      int64 // Multiplier for number of blocks to prefetch.
	RandomReadsThreshold    int64 // Number of random reads after which the reader falls back to GCS reader.
}

// blockQueueEntry holds a data block with a function
// to cancel its in-flight download.
type blockQueueEntry struct {
	block  block.PrefetchBlock
	cancel context.CancelFunc
}

type BufferedReader struct {
	gcsx.Reader
	object *gcs.MinObject
	bucket gcs.Bucket
	config *BufferedReadConfig

	// nextBlockIndexToPrefetch is the index of the next block to be
	// prefetched.
	nextBlockIndexToPrefetch int64

	// randomSeekCount is the number of random seeks performed. This is used to
	// detect if the read pattern is random and fall back to a simpler GCS reader.
	randomSeekCount int64

	// numPrefetchBlocks is the number of blocks to prefetch in the next
	// prefetching operation.
	numPrefetchBlocks int64

	blockQueue common.Queue[*blockQueueEntry]

	readHandle []byte // For zonal bucket.

	blockPool    *block.GenBlockPool[block.PrefetchBlock]
	workerPool   workerpool.WorkerPool
	metricHandle metrics.MetricHandle

	ctx        context.Context
	cancelFunc context.CancelFunc
}

// NewBufferedReader returns a new bufferedReader instance.
func NewBufferedReader(object *gcs.MinObject, bucket gcs.Bucket, config *BufferedReadConfig, globalMaxBlocksSem *semaphore.Weighted, workerPool workerpool.WorkerPool, metricHandle metrics.MetricHandle) (*BufferedReader, error) {
	blockpool, err := block.NewPrefetchBlockPool(config.PrefetchBlockSizeBytes, config.MaxPrefetchBlockCnt, globalMaxBlocksSem)
	if err != nil {
		return nil, fmt.Errorf("failed to create worker pool: %w", err)
	}

	reader := &BufferedReader{
		object:                   object,
		bucket:                   bucket,
		config:                   config,
		nextBlockIndexToPrefetch: 0,
		randomSeekCount:          0,
		numPrefetchBlocks:        config.InitialPrefetchBlockCnt,
		blockQueue:               common.NewLinkedListQueue[*blockQueueEntry](),
		blockPool:                blockpool,
		workerPool:               workerPool,
		metricHandle:             metricHandle,
	}

	reader.ctx, reader.cancelFunc = context.WithCancel(context.Background())
	return reader, nil
}

func (p *BufferedReader) prefetch() error {
	// Do not schedule more blocks if the prefetch queue has reached its maximum configured size.
	availableSlots := p.config.MaxPrefetchBlockCnt - int64(p.blockQueue.Len())
	if availableSlots <= 0 {
		return nil
	}
	blockCountToPrefetch := min(p.numPrefetchBlocks, availableSlots)
	if blockCountToPrefetch <= 0 {
		return nil
	}

	logger.Debugf("Prefetching %d blocks", blockCountToPrefetch)

	for i := int64(0); i < blockCountToPrefetch; i++ {
		if p.nextBlockIndexToPrefetch >= p.totalBlockCount() {
			break
		}
		if err := p.scheduleNextBlock(false); err != nil {
			return fmt.Errorf("failed to schedule block index %d: %v", p.nextBlockIndexToPrefetch, err)
		}
	}

	// Set the size for the next multiplicative prefetch.
	p.numPrefetchBlocks *= p.config.PrefetchMultiplier

	// Ensure the number of blocks for the next prefetch does not exceed the configured maximum.
	if p.numPrefetchBlocks > p.config.MaxPrefetchBlockCnt {
		p.numPrefetchBlocks = p.config.MaxPrefetchBlockCnt
	}
	return nil
}

func (p *BufferedReader) totalBlockCount() int64 {
	return (int64(p.object.Size) + p.config.PrefetchBlockSizeBytes - 1) / p.config.PrefetchBlockSizeBytes
}

func (p *BufferedReader) freshStart(currentOffset int64) error {
	blockIndex := currentOffset / p.config.PrefetchBlockSizeBytes
	p.nextBlockIndexToPrefetch = blockIndex

	// Determine the number of blocks for the initial prefetch.
	p.numPrefetchBlocks = min(p.config.InitialPrefetchBlockCnt, p.config.MaxPrefetchBlockCnt)

	// Schedule the first block as urgent.
	if err := p.scheduleNextBlock(true); err != nil {
		return fmt.Errorf("initial scheduling failed: %w", err)
	}

	// Prefetch the initial blocks.
	if err := p.prefetch(); err != nil {
		return fmt.Errorf("freshStart: prefetch failed: %w", err)
	}
	return nil
}

// scheduleNextBlock schedules the next block for prefetch.
func (p *BufferedReader) scheduleNextBlock(urgent bool) error {
	// TODO(b/426060431): Replace Get() with TryGet(). Assuming, the current blockPool.Get() gets blocked if block is not available.
	b, err := p.blockPool.Get()
	if err != nil || b == nil {
		if err != nil {
			logger.Warnf("failed to get block from pool: %v", err)
		}
		return ErrPrefetchBlockNotAvailable
	}

	if err := p.scheduleBlockWithIndex(b, p.nextBlockIndexToPrefetch, urgent); err != nil {
		p.blockPool.Release(b)
		return err
	}
	p.nextBlockIndexToPrefetch++
	return nil
}

// scheduleBlockWithIndex schedules a block with a specific index.
func (p *BufferedReader) scheduleBlockWithIndex(b block.PrefetchBlock, blockIndex int64, urgent bool) error {
	startOffset := blockIndex * p.config.PrefetchBlockSizeBytes
	if err := b.SetAbsStartOff(startOffset); err != nil {
		return fmt.Errorf("failed to set start offset on block: %w", err)
	}

	ctx, cancel := context.WithCancel(p.ctx)
	task := NewDownloadTask(ctx, p.object, p.bucket, b, p.readHandle)

	logger.Tracef("Scheduling block (%s, offset %d).", p.object.Name, startOffset)
	p.blockQueue.Push(&blockQueueEntry{
		block:  b,
		cancel: cancel,
	})
	p.workerPool.Schedule(urgent, task)
	return nil
}

func (p *BufferedReader) Destroy() {
	if p.cancelFunc != nil {
		p.cancelFunc()
		p.cancelFunc = nil
	}

	for !p.blockQueue.IsEmpty() {
		bqe := p.blockQueue.Pop()
		bqe.cancel()

		// We expect a context.Canceled error here, but we wait to ensure the
		// block's worker goroutine has finished before releasing the block.
		if _, err := bqe.block.AwaitReady(p.ctx); err != nil && err != context.Canceled {
			logger.Warnf("bufferedread: error waiting for block on destroy: %v", err)
		}
		p.blockPool.Release(bqe.block)
	}

	err := p.blockPool.ClearFreeBlockChannel(true)
	if err != nil {
		logger.Warnf("bufferedread: error clearing free block channel: %v", err)
	}
	p.blockPool = nil
}

// CheckInvariants checks for internal consistency of the reader.
func (p *BufferedReader) CheckInvariants() {

	// The number of items in the blockQueue should not exceed the configured limit.
	if int64(p.blockQueue.Len()) > p.config.MaxPrefetchBlockCnt {
		panic(fmt.Sprintf("BufferedReader: blockQueue length %d exceeds limit %d", p.blockQueue.Len(), p.config.MaxPrefetchBlockCnt))
	}

	// The random seek count should never exceed the configured threshold.
	if p.randomSeekCount > p.config.RandomReadsThreshold {
		panic(fmt.Sprintf("BufferedReader: randomSeekCount %d exceeds threshold %d", p.randomSeekCount, p.config.RandomReadsThreshold))
	}
}
