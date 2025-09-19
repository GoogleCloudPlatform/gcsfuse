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

// Prefetcher encapsulates the state and logic for prefetching blocks from GCS.
type Prefetcher struct {
	// Dependencies
	object       *gcs.MinObject
	bucket       gcs.Bucket
	pool         *block.GenBlockPool[block.PrefetchBlock]
	workerPool   workerpool.WorkerPool
	queue        common.Queue[*blockQueueEntry]
	retired      RetiredBlockCache
	metricHandle metrics.MetricHandle
	config       *BufferedReadConfig
	readerCtx    context.Context
	readHandle   []byte

	// State
	nextBlockIndexToPrefetch int64
	numPrefetchBlocks        int64
	prefetchMultiplier       int64
}

// PrefetcherOptions holds the dependencies for a Prefetcher.
type PrefetcherOptions struct {
	Object       *gcs.MinObject
	Bucket       gcs.Bucket
	Config       *BufferedReadConfig
	Pool         *block.GenBlockPool[block.PrefetchBlock]
	WorkerPool   workerpool.WorkerPool
	Queue        common.Queue[*blockQueueEntry]
	Retired      RetiredBlockCache
	MetricHandle metrics.MetricHandle
	ReaderCtx    context.Context
	ReadHandle   []byte
}

// NewPrefetcher creates a new Prefetcher instance.
func NewPrefetcher(opts *PrefetcherOptions) *Prefetcher {
	return &Prefetcher{
		object:                   opts.Object,
		bucket:                   opts.Bucket,
		config:                   opts.Config,
		pool:                     opts.Pool,
		workerPool:               opts.WorkerPool,
		queue:                    opts.Queue,
		retired:                  opts.Retired,
		metricHandle:             opts.MetricHandle,
		readerCtx:                opts.ReaderCtx,
		readHandle:               opts.ReadHandle,
		nextBlockIndexToPrefetch: 0,
		numPrefetchBlocks:        opts.Config.InitialPrefetchBlockCnt,
		prefetchMultiplier:       defaultPrefetchMultiplier,
	}
}

// Prefetch schedules the next set of blocks for prefetching.
func (p *Prefetcher) Prefetch() error {
	availableSlots := p.config.MaxPrefetchBlockCnt - (int64(p.queue.Len()) + int64(p.retired.Len()))
	if availableSlots <= 0 {
		return nil
	}
	totalBlockCount := (int64(p.object.Size) + p.config.PrefetchBlockSizeBytes - 1) / p.config.PrefetchBlockSizeBytes
	remainingBlocksInFile := totalBlockCount - p.nextBlockIndexToPrefetch
	blockCountToPrefetch := min(min(p.numPrefetchBlocks, availableSlots), remainingBlocksInFile)
	if blockCountToPrefetch <= 0 {
		return nil
	}

	allBlocksScheduledSuccessfully := true
	for i := int64(0); i < blockCountToPrefetch; i++ {
		if err := p.scheduleNextBlock(false); err != nil {
			if errors.Is(err, ErrPrefetchBlockNotAvailable) {
				allBlocksScheduledSuccessfully = false
				break
			}
			return fmt.Errorf("Prefetch: scheduling block index %d: %w", p.nextBlockIndexToPrefetch, err)
		}
	}

	if allBlocksScheduledSuccessfully {
		p.numPrefetchBlocks *= p.prefetchMultiplier
		if p.numPrefetchBlocks > p.config.MaxPrefetchBlockCnt {
			p.numPrefetchBlocks = p.config.MaxPrefetchBlockCnt
		}
	}
	return nil
}

// FreshStart resets the prefetching state and schedules the initial set of blocks.
func (p *Prefetcher) FreshStart(currentOffset int64) error {
	blockIndex := currentOffset / p.config.PrefetchBlockSizeBytes
	p.nextBlockIndexToPrefetch = blockIndex
	p.numPrefetchBlocks = min(p.config.InitialPrefetchBlockCnt, p.config.MaxPrefetchBlockCnt)

	if err := p.scheduleNextBlock(true); err != nil {
		return fmt.Errorf("FreshStart: scheduling first block: %w", err)
	}

	if err := p.Prefetch(); err != nil {
		logger.Warnf("FreshStart: initial prefetch failed: %v", err)
	}
	return nil
}

// scheduleNextBlock schedules the next block in the sequence for prefetch.
func (p *Prefetcher) scheduleNextBlock(urgent bool) error {
	b, err := p.pool.TryGet()
	if err != nil {
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

// scheduleBlockWithIndex schedules a download task for a specific block index.
func (p *Prefetcher) scheduleBlockWithIndex(b block.PrefetchBlock, blockIndex int64, urgent bool) error {
	startOffset := blockIndex * p.config.PrefetchBlockSizeBytes
	if err := b.SetAbsStartOff(startOffset); err != nil {
		return fmt.Errorf("scheduleBlockWithIndex: setting start offset: %w", err)
	}

	ctx, cancel := context.WithCancel(p.readerCtx)
	task := NewDownloadTask(&DownloadTaskOptions{
		Ctx:          ctx,
		Object:       p.object,
		Bucket:       p.bucket,
		Block:        b,
		ReadHandle:   p.readHandle,
		MetricHandle: p.metricHandle,
	})

	logger.Tracef("Scheduling block: (%s, %d, %t).", p.object.Name, blockIndex, urgent)
	p.queue.Push(&blockQueueEntry{
		block:             b,
		cancel:            cancel,
		prefetchTriggered: false,
	})
	p.workerPool.Schedule(urgent, task)
	return nil
}
