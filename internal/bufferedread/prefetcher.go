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
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/workerpool"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
)

// prefetcher encapsulates the state and logic for prefetching blocks from GCS. It
// is responsible for managing the prefetch window, scheduling download tasks,
// and handling the multiplicative increase in the number of blocks to prefetch.
type prefetcher struct {
	object       *gcs.MinObject
	bucket       gcs.Bucket
	pool         *block.GenBlockPool[block.PrefetchBlock]
	workerPool   workerpool.WorkerPool
	queue        common.Queue[*blockQueueEntry]
	metricHandle metrics.MetricHandle
	config       *BufferedReadConfig
	readerCtx    context.Context
	readHandle   []byte // For zonal bucket.

	// nextBlockIndexToPrefetch is the index of the next block to be prefetched.
	nextBlockIndexToPrefetch int64

	// numPrefetchBlocks is the number of blocks to prefetch in the next
	// prefetching operation.
	numPrefetchBlocks int64

	// prefetchMultiplier is the factor by which numPrefetchBlocks is multiplied
	// after a successful prefetch cycle.
	prefetchMultiplier int64
}

// prefetcherOptions holds the dependencies for creating a new Prefetcher.
type prefetcherOptions struct {
	object       *gcs.MinObject
	bucket       gcs.Bucket
	config       *BufferedReadConfig
	pool         *block.GenBlockPool[block.PrefetchBlock]
	workerPool   workerpool.WorkerPool
	queue        common.Queue[*blockQueueEntry]
	metricHandle metrics.MetricHandle
	readerCtx    context.Context
	readHandle   []byte
}

// newPrefetcher creates a new Prefetcher instance.
func newPrefetcher(opts *prefetcherOptions) *prefetcher {
	return &prefetcher{
		object:                   opts.object,
		bucket:                   opts.bucket,
		config:                   opts.config,
		pool:                     opts.pool,
		workerPool:               opts.workerPool,
		queue:                    opts.queue,
		metricHandle:             opts.metricHandle,
		readerCtx:                opts.readerCtx,
		readHandle:               opts.readHandle,
		nextBlockIndexToPrefetch: 0,
		numPrefetchBlocks:        opts.config.InitialPrefetchBlockCnt,
		prefetchMultiplier:       defaultPrefetchMultiplier,
	}
}

// prefetch schedules the next set of blocks for prefetching starting from
// the nextBlockIndexToPrefetch.
// LOCKS_REQUIRED(p.mu)
func (p *prefetcher) prefetch() error {
	// Determine the number of blocks to prefetch in this cycle, respecting the
	// MaxPrefetchBlockCnt and the number of blocks remaining in the file.
	availableSlots := p.config.MaxPrefetchBlockCnt - int64(p.queue.Len())
	if availableSlots <= 0 {
		return nil
	}
	totalBlockCount := (int64(p.object.Size) + p.config.PrefetchBlockSizeBytes - 1) / p.config.PrefetchBlockSizeBytes
	remainingBlocksInFile := totalBlockCount - p.nextBlockIndexToPrefetch
	blockCountToPrefetch := min(min(p.numPrefetchBlocks, availableSlots), remainingBlocksInFile)
	if blockCountToPrefetch <= 0 {
		// No blocks left to prefetch in the file or no blocks requested for this
		// cycle.
		return nil
	}

	allBlocksScheduledSuccessfully := true
	for range blockCountToPrefetch {
		if err := p.scheduleNextBlock(false); err != nil {
			if errors.Is(err, ErrPrefetchBlockNotAvailable) {
				// This is not a critical error for a background prefetch. We just stop
				// trying to prefetch more in this cycle. The specific reason has
				// already been logged by scheduleNextBlock.
				allBlocksScheduledSuccessfully = false
				break // Stop prefetching more blocks.
			}
			return fmt.Errorf("prefetch: scheduling block index %d: %w", p.nextBlockIndexToPrefetch, err)
		}
	}

	// Only increase the prefetch window size if we successfully scheduled all the
	// intended blocks. This is a conservative approach that prevents the window
	// from growing aggressively if the block pool is consistently under pressure.
	if allBlocksScheduledSuccessfully {
		// Set the size for the next multiplicative prefetch.
		p.numPrefetchBlocks *= p.prefetchMultiplier

		// Cap the prefetch window size for the next cycle at the configured
		// maximum to prevent unbounded growth.
		if p.numPrefetchBlocks > p.config.MaxPrefetchBlockCnt {
			p.numPrefetchBlocks = p.config.MaxPrefetchBlockCnt
		}
	}
	return nil
}

// freshStart resets the prefetching state and schedules the initial set of
// blocks starting from the given offset. This is called on the first read or
// after a random seek.
// LOCKS_REQUIRED(p.mu)
func (p *prefetcher) freshStart(currentOffset int64) error {
	blockIndex := currentOffset / p.config.PrefetchBlockSizeBytes
	p.nextBlockIndexToPrefetch = blockIndex

	// Reset the prefetch window to the initial configured size.
	p.numPrefetchBlocks = min(p.config.InitialPrefetchBlockCnt, p.config.MaxPrefetchBlockCnt)

	// Schedule the first block as urgent to unblock the foreground read.
	if err := p.scheduleNextBlock(true); err != nil {
		return fmt.Errorf("freshStart: scheduling first block: %w", err)
	}

	// Prefetch the subsequent initial set of blocks in the background.
	if err := p.prefetch(); err != nil {
		// A failure during the initial prefetch is not fatal, as the first block
		// has already been scheduled. Log the error and continue.
		logger.Warnf("freshStart: initial prefetch failed: %v", err)
	}
	return nil
}

// scheduleNextBlock attempts to get a block from the pool and schedules the
// next block in the sequence for prefetching.
// LOCKS_REQUIRED(p.mu)
func (p *prefetcher) scheduleNextBlock(urgent bool) error {
	b, err := p.pool.TryGet()
	if err != nil {
		// Any error from TryGet (e.g., pool exhausted, mmap failure) means we
		// can't get a block. For the buffered reader, this is a recoverable
		// condition that should either trigger a fallback to another reader (for
		// urgent reads) or be ignored (for background prefetches).
		logger.Tracef("scheduleNextBlock: could not get block from pool (urgent=%t): %v", urgent, err)
		return ErrPrefetchBlockNotAvailable
	}

	if err := p.scheduleBlockWithIndex(b, p.nextBlockIndexToPrefetch, urgent); err != nil {
		p.pool.Release(b)
		return fmt.Errorf("scheduleNextBlock: %w", err)
	}
	p.nextBlockIndexToPrefetch++
	return nil
}

// scheduleBlockWithIndex creates and schedules a download task for a specific
// block index.
// LOCKS_REQUIRED(p.mu)
func (p *prefetcher) scheduleBlockWithIndex(b block.PrefetchBlock, blockIndex int64, urgent bool) error {
	startOffset := blockIndex * p.config.PrefetchBlockSizeBytes
	if err := b.SetAbsStartOff(startOffset); err != nil {
		return fmt.Errorf("scheduleBlockWithIndex: setting start offset: %w", err)
	}

	ctx, cancel := context.WithCancel(p.readerCtx)
	task := newDownloadTask(&downloadTaskOptions{
		ctx:          ctx,
		object:       p.object,
		bucket:       p.bucket,
		block:        b,
		readHandle:   p.readHandle,
		metricHandle: p.metricHandle,
	})

	logger.Tracef("Scheduling block: (%s, %d, %t).", p.object.Name, blockIndex, urgent)
	p.queue.Push(&blockQueueEntry{
		block:  b,
		cancel: cancel,
	})
	p.workerPool.Schedule(urgent, task)
	return nil
}
