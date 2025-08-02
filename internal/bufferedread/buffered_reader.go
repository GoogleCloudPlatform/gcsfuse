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
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/googlecloudplatform/gcsfuse/v3/common"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/block"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/workerpool"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/jacobsa/fuse/fuseops"
	"golang.org/x/sync/semaphore"
)

// ErrPrefetchBlockNotAvailable is returned when a block cannot be
// acquired from the pool for prefetching. This can be used by callers to
// implement a fallback mechanism, e.g. falling back to another reader.
var ErrPrefetchBlockNotAvailable = errors.New("block for prefetching not available")

type BufferedReadConfig struct {
	MaxPrefetchBlockCnt     int64 // Maximum number of blocks that can be prefetched.
	PrefetchBlockSizeBytes  int64 // Size of each block to be prefetched.
	InitialPrefetchBlockCnt int64 // Number of blocks to prefetch initially.
}

const (
	defaultRandomReadsThreshold = 3
	defaultPrefetchMultiplier   = 2
	ReadOp                      = "readOp"
)

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
	// detect if the read pattern is random and fall back to another reader.
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

	prefetchMultiplier int64 // Multiplier for number of blocks to prefetch.

	randomReadsThreshold int64 // Number of random reads after which the reader falls back to another reader.
}

// NewBufferedReader returns a new bufferedReader instance.
func NewBufferedReader(object *gcs.MinObject, bucket gcs.Bucket, config *BufferedReadConfig, globalMaxBlocksSem *semaphore.Weighted, workerPool workerpool.WorkerPool, metricHandle metrics.MetricHandle) (*BufferedReader, error) {
	blockpool, err := block.NewPrefetchBlockPool(config.PrefetchBlockSizeBytes, config.MaxPrefetchBlockCnt, globalMaxBlocksSem)
	if err != nil {
		return nil, fmt.Errorf("NewBufferedReader: failed to create block-pool: %w", err)
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
		prefetchMultiplier:       defaultPrefetchMultiplier,
		randomReadsThreshold:     defaultRandomReadsThreshold,
	}

	reader.ctx, reader.cancelFunc = context.WithCancel(context.Background())
	return reader, nil
}

// handleRandomRead detects and handles random read patterns. A read is considered
// random if the requested offset is outside the currently prefetched window.
// If the number of detected random reads exceeds a configured threshold, it
// returns a gcsx.FallbackToAnotherReader error to signal that another reader
// should be used.
func (p *BufferedReader) handleRandomRead(offset int64) error {
	// Exit early if we have already decided to fall back to another reader.
	// This avoids re-evaluating the read pattern on every call when the random
	// read threshold has been met.
	if p.randomSeekCount > p.randomReadsThreshold {
		return gcsx.FallbackToAnotherReader
	}

	if !p.isRandomSeek(offset) {
		return nil
	}

	p.randomSeekCount++

	// When a random seek is detected, the prefetched blocks in the queue become
	// irrelevant. We must clear the queue, cancel any ongoing downloads, and
	// release the blocks back to the pool.
	for !p.blockQueue.IsEmpty() {
		entry := p.blockQueue.Pop()
		entry.cancel()
		if _, waitErr := entry.block.AwaitReady(context.Background()); waitErr != nil {
			logger.Warnf("handleRandomRead: AwaitReady error during discard (offset=%d): %v", offset, waitErr)
		}
		p.blockPool.Release(entry.block)
	}

	if p.randomSeekCount > p.randomReadsThreshold {
		logger.Warnf("handleRandomRead: random seek count %d exceeded threshold %d, falling back to another reader", p.randomSeekCount, p.randomReadsThreshold)
		return gcsx.FallbackToAnotherReader
	}

	return nil
}

// isRandomSeek checks if the read for the given offset is random or not.
func (p *BufferedReader) isRandomSeek(offset int64) bool {
	if p.blockQueue.IsEmpty() {
		return offset != 0
	}

	start := p.blockQueue.Peek().block.AbsStartOff()
	end := start + int64(p.blockQueue.Len())*p.config.PrefetchBlockSizeBytes
	if offset < start || offset >= end {
		return true
	}

	return false
}

// prepareQueueForOffset cleans the head of the block queue by discarding any
// blocks that are no longer relevant for the given read offset. This occurs on
// seeks (both forward and backward) that land outside the current block.
// For each discarded block, its download is cancelled, and it is returned to
// the block pool.
func (p *BufferedReader) prepareQueueForOffset(offset int64) {
	for !p.blockQueue.IsEmpty() {
		entry := p.blockQueue.Peek()
		block := entry.block
		blockStart := block.AbsStartOff()
		blockEnd := blockStart + block.Cap()

		if offset < blockStart || offset >= blockEnd {
			// Offset is either before or beyond this block – discard.
			p.blockQueue.Pop()
			entry.cancel()

			if _, waitErr := block.AwaitReady(context.Background()); waitErr != nil {
				logger.Warnf("prepareQueueForOffset: AwaitReady error during discard (offset=%d): %v", offset, waitErr)
			}

			p.blockPool.Release(block)
		} else {
			break
		}
	}
}

// ReadAt reads data from the GCS object into the provided buffer starting at
// the given offset. It implements the gcsx.Reader interface.
//
// The read is satisfied by reading from in-memory blocks that are prefetched
// in the background. The core logic is as follows:
//  1. Detect if the read pattern is random. If so, and if the random read
//     threshold is exceeded, it returns a FallbackToAnotherReader error.
//  2. Prepare the internal block queue by discarding any stale blocks from the
//     head of the queue that are before the requested offset.
//  3. If the queue becomes empty (e.g., on a fresh read or a large seek), it
//     initiates a "fresh start" to prefetch blocks starting from the current
//     offset.
//  4. It then enters a loop to fill the destination buffer:
//     a. It waits for the block at the head of the queue to be downloaded.
//     b. If the download failed or was cancelled, it returns an appropriate error.
//     c. If successful, it copies data from the downloaded block into the buffer.
//     d. If a block is fully consumed, it is removed from the queue, and a new
//     prefetch operation is triggered to keep the pipeline full.
//  5. The loop continues until the buffer is full, the end of the file is
//     reached, or an error occurs.
func (p *BufferedReader) ReadAt(ctx context.Context, inputBuf []byte, off int64) (gcsx.ReaderResponse, error) {
	resp := gcsx.ReaderResponse{DataBuf: inputBuf}
	reqID := uuid.New()
	start := time.Now()
	initOff := off
	blockIdx := initOff / p.config.PrefetchBlockSizeBytes
	var handleID uint64
	if readOp, ok := ctx.Value(ReadOp).(*fuseops.ReadFileOp); ok {
		handleID = uint64(readOp.Handle)
	}

	var bytesRead int
	var err error
	logger.Tracef("%.13v <- ReadAt(%s:/%s, %d, %d, %d, %d)", reqID, p.bucket.Name(), p.object.Name, handleID, off, len(inputBuf), blockIdx)

	if off >= int64(p.object.Size) {
		err = io.EOF
		return resp, err
	}

	if len(inputBuf) == 0 {
		return resp, nil
	}

	defer func() {
		dur := time.Since(start)
		if err == nil || errors.Is(err, io.EOF) {
			logger.Tracef("%.13v -> ReadAt(): Ok(%v)", reqID, dur)
		}
	}()

	if err = p.handleRandomRead(off); err != nil {
		err = fmt.Errorf("ReadAt: handleRandomRead failed: %w", err)
		return resp, err
	}

	prefetchTriggered := false

	for bytesRead < len(inputBuf) {
		p.prepareQueueForOffset(off)

		if p.blockQueue.IsEmpty() {
			if err = p.freshStart(off); err != nil {
				if errors.Is(err, ErrPrefetchBlockNotAvailable) {
					return resp, gcsx.FallbackToAnotherReader
				}
				err = fmt.Errorf("ReadAt: freshStart failed: %w", err)
				break
			}
			prefetchTriggered = true
		}

		entry := p.blockQueue.Peek()
		blk := entry.block

		status, waitErr := blk.AwaitReady(ctx)
		if waitErr != nil {
			err = fmt.Errorf("ReadAt: AwaitReady failed: %w", waitErr)
			break
		}

		if status.State != block.BlockStateDownloaded {
			p.blockQueue.Pop()
			p.blockPool.Release(blk)
			entry.cancel()

			switch status.State {
			case block.BlockStateDownloadFailed:
				err = fmt.Errorf("ReadAt: download failed: %w", status.Err)
			default:
				err = fmt.Errorf("ReadAt: unexpected block state: %d", status.State)
			}
			break
		}

		relOff := off - blk.AbsStartOff()
		n, readErr := blk.ReadAt(inputBuf[bytesRead:], relOff)
		bytesRead += n
		off += int64(n)

		if readErr != nil && !errors.Is(readErr, io.EOF) {
			err = fmt.Errorf("ReadAt: block.ReadAt failed: %w", readErr)
			break
		}

		if off >= int64(p.object.Size) {
			err = io.EOF
			break
		}

		if off >= blk.AbsStartOff()+blk.Size() {
			p.blockQueue.Pop()
			p.blockPool.Release(blk)

			if !prefetchTriggered {
				prefetchTriggered = true
				if pfErr := p.prefetch(); pfErr != nil {
					logger.Warnf("ReadAt: prefetch failed: %v", pfErr)
				}
			}
		}
	}

	resp.Size = bytesRead
	return resp, err
}

// prefetch schedules the next set of blocks for prefetching starting from
// the nextBlockIndexToPrefetch.
func (p *BufferedReader) prefetch() error {
	// Do not schedule more than MaxPrefetchBlockCnt.
	availableSlots := p.config.MaxPrefetchBlockCnt - int64(p.blockQueue.Len())
	if availableSlots <= 0 {
		return nil
	}
	blockCountToPrefetch := min(p.numPrefetchBlocks, availableSlots)
	if blockCountToPrefetch <= 0 {
		return nil
	}

	totalBlockCount := (int64(p.object.Size) + p.config.PrefetchBlockSizeBytes - 1) / p.config.PrefetchBlockSizeBytes
	for i := int64(0); i < blockCountToPrefetch; i++ {
		if p.nextBlockIndexToPrefetch >= totalBlockCount {
			break
		}
		if err := p.scheduleNextBlock(false); err != nil {
			return fmt.Errorf("prefetch: failed to schedule block index %d: %v", p.nextBlockIndexToPrefetch, err)
		}
	}

	// Set the size for the next multiplicative prefetch.
	p.numPrefetchBlocks *= p.prefetchMultiplier

	// Do not prefetch more than MaxPrefetchBlockCnt blocks.
	if p.numPrefetchBlocks > p.config.MaxPrefetchBlockCnt {
		p.numPrefetchBlocks = p.config.MaxPrefetchBlockCnt
	}
	return nil
}

// freshStart resets the prefetching state and schedules the initial set of
// blocks starting from the given offset.
func (p *BufferedReader) freshStart(currentOffset int64) error {
	blockIndex := currentOffset / p.config.PrefetchBlockSizeBytes
	p.nextBlockIndexToPrefetch = blockIndex

	// Determine the number of blocks for the initial prefetch.
	p.numPrefetchBlocks = min(p.config.InitialPrefetchBlockCnt, p.config.MaxPrefetchBlockCnt)

	// Schedule the first block as urgent.
	if err := p.scheduleNextBlock(true); err != nil {
		return fmt.Errorf("freshStart: initial scheduling failed: %w", err)
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
		return fmt.Errorf("scheduleBlockWithIndex: failed to set start offset: %w", err)
	}

	ctx, cancel := context.WithCancel(p.ctx)
	task := NewDownloadTask(ctx, p.object, p.bucket, b, p.readHandle)

	logger.Tracef("Scheduling block: (%s, %d, %t).", p.object.Name, blockIndex, urgent)
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

	// The prefetch block size must be positive.
	if p.config.PrefetchBlockSizeBytes <= 0 {
		panic(fmt.Sprintf("BufferedReader: PrefetchBlockSizeBytes must be positive, but is %d", p.config.PrefetchBlockSizeBytes))
	}

	// The prefetch block size must be at least 1 MiB.
	if p.config.PrefetchBlockSizeBytes < util.MiB {
		panic(fmt.Sprintf("BufferedReader: PrefetchBlockSizeBytes must be at least 1 MiB, but is %d", p.config.PrefetchBlockSizeBytes))
	}

	// The number of items in the blockQueue should not exceed MaxPrefetchBlockCnt.
	if int64(p.blockQueue.Len()) > p.config.MaxPrefetchBlockCnt {
		panic(fmt.Sprintf("BufferedReader: blockQueue length %d exceeds limit %d", p.blockQueue.Len(), p.config.MaxPrefetchBlockCnt))
	}

	// The random seek count should never exceed randomReadsThreshold.
	if p.randomSeekCount > p.randomReadsThreshold {
		panic(fmt.Sprintf("BufferedReader: randomSeekCount %d exceeds threshold %d", p.randomSeekCount, p.randomReadsThreshold))
	}
}
