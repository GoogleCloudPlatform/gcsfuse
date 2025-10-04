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
	"sync"
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
	MinBlocksPerHandle      int64 // Minimum number of blocks available in block-pool to start buffered-read.
	RandomSeekThreshold     int64 // Seek count threshold to switch another reader
}

const (
	defaultPrefetchMultiplier = 2
	ReadOp                    = "readOp"
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

	metricHandle metrics.MetricHandle

	readHandle []byte // For zonal bucket.

	ctx        context.Context
	cancelFunc context.CancelFunc

	prefetchMultiplier int64 // Multiplier for number of blocks to prefetch.

	randomReadsThreshold int64 // Number of random reads after which the reader falls back to another reader.

	// `mu` synchronizes access to the buffered reader's shared state.
	// All shared variables, such as the block pool and queue, require this lock before any operation.
	mu sync.Mutex

	// GUARDED by (mu)
	workerPool workerpool.WorkerPool

	// blockQueue is the core of the prefetching pipeline, holding blocks that are
	// either downloaded or in the process of being downloaded.
	// GUARDED by (mu)
	blockQueue common.Queue[*blockQueueEntry]

	// blockPool is a pool of blocks that can be reused for prefetching.
	// It is used to avoid allocating new blocks for each prefetch operation.
	// The pool is initialized with a maximum number of blocks that can be
	// prefetched at a time, and it allows for efficient reuse of blocks.
	// The pool is also responsible for managing the global limit on the number
	// of blocks that can be allocated across all BufferedReader instances.
	// GUARDED by (mu)
	blockPool *block.GenBlockPool[block.PrefetchBlock]
}

// NewBufferedReader returns a new bufferedReader instance.
func NewBufferedReader(object *gcs.MinObject, bucket gcs.Bucket, config *BufferedReadConfig, globalMaxBlocksSem *semaphore.Weighted, workerPool workerpool.WorkerPool, metricHandle metrics.MetricHandle) (*BufferedReader, error) {
	if config.PrefetchBlockSizeBytes <= 0 {
		return nil, fmt.Errorf("NewBufferedReader: PrefetchBlockSizeBytes must be positive, but is %d", config.PrefetchBlockSizeBytes)
	}
	// To optimize resource usage, reserve only the number of blocks required for
	// the file, capped by the configured minimum.
	blocksInFile := (int64(object.Size) + config.PrefetchBlockSizeBytes - 1) / config.PrefetchBlockSizeBytes
	numBlocksToReserve := min(blocksInFile, config.MinBlocksPerHandle)
	blockpool, err := block.NewPrefetchBlockPool(config.PrefetchBlockSizeBytes, config.MaxPrefetchBlockCnt, numBlocksToReserve, globalMaxBlocksSem)
	if err != nil {
		if errors.Is(err, block.CantAllocateAnyBlockError) {
			metricHandle.BufferedReadFallbackTriggerCount(1, "insufficient_memory")
		}
		return nil, fmt.Errorf("NewBufferedReader: creating block-pool: %w", err)
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
		randomReadsThreshold:     config.RandomSeekThreshold,
	}

	reader.ctx, reader.cancelFunc = context.WithCancel(context.Background())
	return reader, nil
}

// handleRandomRead detects and handles random read patterns. A read is considered
// random if the requested offset is outside the currently prefetched window.
// If the number of detected random reads exceeds a configured threshold, it
// returns a gcsx.FallbackToAnotherReader error to signal that another reader
// should be used. It takes handleID for logging purposes.
// LOCKS_REQUIRED(p.mu)
func (p *BufferedReader) handleRandomRead(offset int64, handleID int64) error {
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
			logger.Warnf("handleRandomRead: AwaitReady during discard (offset=%d): %v", offset, waitErr)
		}
		p.blockPool.Release(entry.block)
	}

	if p.randomSeekCount > p.randomReadsThreshold {
		logger.Warnf("Fallback to another reader for object %q, handle %d. Random seek count %d exceeded threshold %d.", p.object.Name, handleID, p.randomSeekCount, p.randomReadsThreshold)
		p.metricHandle.BufferedReadFallbackTriggerCount(1, "random_read_detected")
		return gcsx.FallbackToAnotherReader
	}

	return nil
}

// isRandomSeek checks if the read for the given offset is random or not.
// LOCKS_REQUIRED(p.mu)
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
// LOCKS_REQUIRED(p.mu)
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
				logger.Warnf("prepareQueueForOffset: AwaitReady during discard (offset=%d): %v", offset, waitErr)
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
//
// LOCKS_EXCLUDED(p.mu)
func (p *BufferedReader) ReadAt(ctx context.Context, inputBuf []byte, off int64) (gcsx.ReaderResponse, error) {
	resp := gcsx.ReaderResponse{DataBuf: inputBuf}
	reqID := uuid.New()
	start := time.Now()
	initOff := off
	blockIdx := initOff / p.config.PrefetchBlockSizeBytes
	var bytesRead int
	var err error
	handleID := int64(-1) // As 0 is a valid handle ID, we use -1 to indicate no handle.
	if readOp, ok := ctx.Value(ReadOp).(*fuseops.ReadFileOp); ok {
		handleID = int64(readOp.Handle)
	}

	logger.Tracef("%.13v <- ReadAt(%s:/%s, %d, %d, %d, %d)", reqID, p.bucket.Name(), p.object.Name, handleID, off, len(inputBuf), blockIdx)

	if off >= int64(p.object.Size) {
		err = io.EOF
		return resp, err
	}

	if len(inputBuf) == 0 {
		return resp, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	defer func() {
		dur := time.Since(start)
		p.metricHandle.BufferedReadReadLatency(ctx, dur)
		if err == nil || errors.Is(err, io.EOF) {
			logger.Tracef("%.13v -> ReadAt(): Ok(%v)", reqID, dur)
		}
	}()

	if err = p.handleRandomRead(off, handleID); err != nil {
		err = fmt.Errorf("BufferedReader.ReadAt: handleRandomRead: %w", err)
		return resp, err
	}

	prefetchTriggered := false

	for bytesRead < len(inputBuf) {
		p.prepareQueueForOffset(off)

		if p.blockQueue.IsEmpty() {
			if err = p.freshStart(off); err != nil {
				logger.Warnf("Fallback to another reader for object %q, handle %d, due to freshStart failure: %v", p.object.Name, handleID, err)
				p.metricHandle.BufferedReadFallbackTriggerCount(1, "insufficient_memory")
				return resp, gcsx.FallbackToAnotherReader
			}
			prefetchTriggered = true
		}

		entry := p.blockQueue.Peek()
		blk := entry.block

		status, waitErr := blk.AwaitReady(ctx)
		if waitErr != nil {
			err = fmt.Errorf("BufferedReader.ReadAt: AwaitReady: %w", waitErr)
			break
		}

		if status.State != block.BlockStateDownloaded {
			p.blockQueue.Pop()
			p.blockPool.Release(blk)
			entry.cancel()

			switch status.State {
			case block.BlockStateDownloadFailed:
				err = fmt.Errorf("BufferedReader.ReadAt: download failed: %w", status.Err)
			default:
				err = fmt.Errorf("BufferedReader.ReadAt: unexpected block state: %d", status.State)
			}
			break
		}

		relOff := off - blk.AbsStartOff()
		n, readErr := blk.ReadAt(inputBuf[bytesRead:], relOff)
		bytesRead += n
		off += int64(n)

		if readErr != nil && !errors.Is(readErr, io.EOF) {
			err = fmt.Errorf("BufferedReader.ReadAt: block.ReadAt: %w", readErr)
			break
		}

		if off >= int64(p.object.Size) {
			break
		}

		if off >= blk.AbsStartOff()+blk.Size() {
			p.blockQueue.Pop()
			p.blockPool.Release(blk)

			if !prefetchTriggered {
				prefetchTriggered = true
				if pfErr := p.prefetch(); pfErr != nil {
					logger.Warnf("BufferedReader.ReadAt: while prefetching: %v", pfErr)
				}
			}
		}
	}

	resp.Size = bytesRead
	return resp, err
}

// prefetch schedules the next set of blocks for prefetching starting from
// the nextBlockIndexToPrefetch.
// LOCKS_REQUIRED(p.mu)
func (p *BufferedReader) prefetch() error {
	// Determine the number of blocks to prefetch in this cycle, respecting the
	// MaxPrefetchBlockCnt and the number of blocks remaining in the file.
	availableSlots := p.config.MaxPrefetchBlockCnt - int64(p.blockQueue.Len())
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
	// intended blocks. This is a more conservative approach that prevents the
	// window from growing aggressively if block pool is consistently under pressure.
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
// blocks starting from the given offset.
// LOCKS_REQUIRED(p.mu)
func (p *BufferedReader) freshStart(currentOffset int64) error {
	blockIndex := currentOffset / p.config.PrefetchBlockSizeBytes
	p.nextBlockIndexToPrefetch = blockIndex

	// Determine the number of blocks for the initial prefetch.
	p.numPrefetchBlocks = min(p.config.InitialPrefetchBlockCnt, p.config.MaxPrefetchBlockCnt)

	// Schedule the first block as urgent.
	if err := p.scheduleNextBlock(true); err != nil {
		return fmt.Errorf("freshStart: scheduling first block: %w", err)
	}

	// Prefetch the initial blocks.
	if err := p.prefetch(); err != nil {
		// A failure during the initial prefetch is not fatal, as the first block
		// has already been scheduled. Log the error and continue.
		logger.Warnf("freshStart: initial prefetch: %v", err)
	}
	return nil
}

// scheduleNextBlock schedules the next block for prefetch.
// LOCKS_REQUIRED(p.mu)
func (p *BufferedReader) scheduleNextBlock(urgent bool) error {
	b, err := p.blockPool.TryGet()
	if err != nil {
		// Any error from TryGet (e.g., pool exhausted, mmap failure) means we
		// can't get a block. For the buffered reader, this is a recoverable
		// condition that should either trigger a fallback to another reader (for
		// urgent reads) or be ignored (for background prefetches).
		logger.Tracef("scheduleNextBlock: could not get block from pool (urgent=%t): %v", urgent, err)
		return ErrPrefetchBlockNotAvailable
	}

	if err := p.scheduleBlockWithIndex(b, p.nextBlockIndexToPrefetch, urgent); err != nil {
		p.blockPool.Release(b)
		return fmt.Errorf("scheduleNextBlock: %w", err)
	}
	p.nextBlockIndexToPrefetch++
	return nil
}

// scheduleBlockWithIndex schedules a block with a specific index.
// LOCKS_REQUIRED(p.mu)
func (p *BufferedReader) scheduleBlockWithIndex(b block.PrefetchBlock, blockIndex int64, urgent bool) error {
	startOffset := blockIndex * p.config.PrefetchBlockSizeBytes
	if err := b.SetAbsStartOff(startOffset); err != nil {
		return fmt.Errorf("scheduleBlockWithIndex: setting start offset: %w", err)
	}

	ctx, cancel := context.WithCancel(p.ctx)
	task := NewDownloadTask(ctx, p.object, p.bucket, b, p.readHandle, p.metricHandle, p.updateReadHandle)

	logger.Tracef("Scheduling block: (%s, %d, %t).", p.object.Name, blockIndex, urgent)
	p.blockQueue.Push(&blockQueueEntry{
		block:  b,
		cancel: cancel,
	})
	p.workerPool.Schedule(urgent, task)
	return nil
}

// LOCKS_EXCLUDED(p.mu)
func (p *BufferedReader) Destroy() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for !p.blockQueue.IsEmpty() {
		bqe := p.blockQueue.Pop()
		bqe.cancel()

		// We wait for the block's worker goroutine to finish. We expect its
		// status to contain a context.Canceled error because we just called cancel.
		status, err := bqe.block.AwaitReady(context.Background())
		if err != nil {
			logger.Warnf("Destroy: AwaitReady for block failed: %v", err)
		} else if status.Err != nil && !errors.Is(status.Err, context.Canceled) {
			logger.Warnf("Destroy: waiting for block on destroy: %v", status.Err)
		}
		p.blockPool.Release(bqe.block)
	}

	if p.cancelFunc != nil {
		p.cancelFunc()
		p.cancelFunc = nil
	}

	err := p.blockPool.ClearFreeBlockChannel(true)
	if err != nil {
		logger.Warnf("Destroy: clearing free block channel: %v", err)
	}
	p.blockPool = nil
}

// updateReadHandle updates the read handle used for subsequent reads.
// This is called by DownloadTask after successful reads to enable more efficient subsequent reads.
// This method is non-blocking to avoid deadlocks because of cyclic dependencies between main reader
// thread and download task threads.
func (p *BufferedReader) updateReadHandle(newReadHandle []byte) {
	// Use TryLock to avoid any blocking - if we can't get the lock immediately,
	// we just skip the update since it's an optimization, not critical functionality.
	if p.mu.TryLock() {
		logger.Tracef("updateReadHandle: updating read handle for object: %v", p.object.Name)
		p.readHandle = newReadHandle
		p.mu.Unlock()
	}
}

// CheckInvariants checks for internal consistency of the reader.
// LOCKS_EXCLUDED(p.mu)
func (p *BufferedReader) CheckInvariants() {
	p.mu.Lock()
	defer p.mu.Unlock()

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
