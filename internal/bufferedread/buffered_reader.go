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
	RetiredBlocksPerHandle  int64 // Number of retired blocks to keep for a file handle.
}

const (
	defaultPrefetchMultiplier = 2
	ReadOp                    = "readOp"
)

type BufferedReader struct {
	gcsx.Reader
	object *gcs.MinObject
	bucket gcs.Bucket
	config *BufferedReadConfig

	metricHandle metrics.MetricHandle

	readHandle []byte // For zonal bucket.

	ctx        context.Context
	cancelFunc context.CancelFunc

	// `mu` synchronizes access to the buffered reader's shared state.
	// All shared variables, such as the block pool and queue, require this lock before any operation.
	mu sync.Mutex

	// GUARDED by (mu)
	workerPool workerpool.WorkerPool

	// patternDetector is responsible for detecting random read patterns.
	// GUARDED by (mu)
	patternDetector *readPatternDetector

	// prefetcher manages the state and logic for prefetching blocks.
	// GUARDED by (mu)
	prefetcher *prefetcher

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

	// retiredBlocks is an LRU cache that stores blocks that have been consumed
	// from the prefetch queue but are kept for a short period to handle
	// out-of-order or concurrent reads without treating them as random seeks.
	// GUARDED by (mu)
	retiredBlocks RetiredBlockCache
}

// BufferedReaderOptions holds the dependencies for a BufferedReader.
type BufferedReaderOptions struct {
	Object             *gcs.MinObject
	Bucket             gcs.Bucket
	Config             *BufferedReadConfig
	GlobalMaxBlocksSem *semaphore.Weighted
	WorkerPool         workerpool.WorkerPool
	MetricHandle       metrics.MetricHandle
}

// NewBufferedReader returns a new bufferedReader instance.
func NewBufferedReader(opts *BufferedReaderOptions) (*BufferedReader, error) {
	if opts.Config.PrefetchBlockSizeBytes <= 0 {
		return nil, fmt.Errorf("NewBufferedReader: PrefetchBlockSizeBytes must be positive, but is %d", opts.Config.PrefetchBlockSizeBytes)
	}
	// To optimize resource usage, reserve only the number of blocks required for
	// the file, capped by the configured minimum.
	blocksInFile := (int64(opts.Object.Size) + opts.Config.PrefetchBlockSizeBytes - 1) / opts.Config.PrefetchBlockSizeBytes
	numBlocksToReserve := min(blocksInFile, opts.Config.MinBlocksPerHandle)
	blockpool, err := block.NewPrefetchBlockPool(opts.Config.PrefetchBlockSizeBytes, opts.Config.MaxPrefetchBlockCnt, numBlocksToReserve, opts.GlobalMaxBlocksSem)
	if err != nil {
		if errors.Is(err, block.CantAllocateAnyBlockError) {
			opts.MetricHandle.BufferedReadFallbackTriggerCount(1, "insufficient_memory")
		}
		return nil, fmt.Errorf("NewBufferedReader: creating block-pool: %w", err)
	}

	reader := &BufferedReader{
		object:          opts.Object,
		bucket:          opts.Bucket,
		config:          opts.Config,
		blockQueue:      common.NewLinkedListQueue[*blockQueueEntry](),
		blockPool:       blockpool,
		workerPool:      opts.WorkerPool,
		metricHandle:    opts.MetricHandle,
		patternDetector: newReadPatternDetector(opts.Config.RandomSeekThreshold, opts.Config.PrefetchBlockSizeBytes),
	}
	reader.ctx, reader.cancelFunc = context.WithCancel(context.Background())

	// The retiredBlocks cache holds blocks that have been consumed but are kept
	// to handle potential out-of-order reads. Its size is set to be the same as
	// the maximum number of prefetch blocks to provide a reasonable buffer for
	// such reads. If `read.retired-blocks-per-handle` is 0, this feature is disabled.
	if opts.Config.RetiredBlocksPerHandle > 0 {
		reader.retiredBlocks = NewLruRetiredBlockCache(uint64(opts.Config.RetiredBlocksPerHandle * opts.Config.PrefetchBlockSizeBytes))
	} else {
		reader.retiredBlocks = &NoOpRetiredBlockCache{}
	}

	prefetcherOpts := &prefetcherOptions{
		Object:       opts.Object,
		Bucket:       opts.Bucket,
		Config:       opts.Config,
		Pool:         blockpool,
		WorkerPool:   opts.WorkerPool,
		Queue:        reader.blockQueue,
		Retired:      reader.retiredBlocks,
		MetricHandle: opts.MetricHandle,
		ReaderCtx:    reader.ctx,
		ReadHandle:   reader.readHandle,
	}
	reader.prefetcher = newPrefetcher(prefetcherOpts)

	return reader, nil
}

// retireBlock moves a block from the active prefetch queue to the retired
// blocks LRU cache. If the cache is full, it evicts the least recently used
// block and releases it back to the pool.
// LOCKS_REQUIRED(p.mu)
func (p *BufferedReader) retireBlock(entry *blockQueueEntry) {
	// If the retired blocks feature is disabled (using the no-op cache),
	// we treat the block as if it were immediately evicted.
	if _, ok := p.retiredBlocks.(*NoOpRetiredBlockCache); ok {
		if entry.block.RefCount() == 0 {
			p.releaseBlock(entry)
		} else {
			entry.block.SetWasEvicted(true)
		}
		return
	}

	blockIndex := entry.block.AbsStartOff() / p.config.PrefetchBlockSizeBytes
	evicted, err := p.retiredBlocks.Insert(blockIndex, entry)
	if err != nil {
		// An error occurred (e.g., entry is too large for the cache). The block
		// was not inserted, so we treat it as if it were immediately evicted.
		logger.Warnf("BufferedReader.retireBlock: failed to insert block %d into retired cache: %v", blockIndex, err)
		if entry.block.RefCount() == 0 {
			p.releaseBlock(entry)
		} else {
			entry.block.SetWasEvicted(true)
		}
		// There should be no evictions if insertion failed, but we proceed just in case.
	}

	for _, evictedEntry := range evicted {
		// This block is being evicted from the retired cache. If it's not in use,
		// we can release it. Otherwise, we set a callback to release it later.
		if evictedEntry.block.RefCount() == 0 {
			p.releaseBlock(evictedEntry)
		} else {
			evictedEntry.block.SetWasEvicted(true)
		}
	}
}

// tryZeroCopyRead attempts to perform a zero-copy read from the given block.
// It returns the reader response and a boolean indicating success. A successful
// zero-copy read means the entire requested data was returned as a slice of the
// block's buffer, which is returned as part of a ReaderResponse struct.
func (p *BufferedReader) tryZeroCopyRead(entry *blockQueueEntry, off int64, inputBuf []byte) (gcsx.ReaderResponse, bool) {
	resp := gcsx.ReaderResponse{}
	blk := entry.block
	relOff := off - blk.AbsStartOff()

	// Check if the entire read can be satisfied by this single block.
	if relOff >= 0 && relOff+int64(len(inputBuf)) <= int64(blk.Size()) {
		slice, sliceErr := blk.ReadAtSlice(len(inputBuf), relOff)
		if sliceErr == nil {
			// For zero-copy reads, we must ensure the block is not released until the
			// kernel is done with the buffer. We increment its reference count here,
			// and the Done function will decrement it later.
			blk.IncrementRef()
			resp.DataBuf = slice
			resp.Size = len(slice)
			resp.Done = func() {
				p.handleZeroCopyDone(entry)
			}
			// log.Printf("Zero Copy succeeded <-(%d, %d, ref_count: %d)", off, len(slice), blk.RefCount())
			return resp, true
		}
		logger.Warnf("BufferedReader.ReadAt: ReadAtSlice failed, falling back to copy: %v", sliceErr)
	}
	return resp, false
}

// handleZeroCopyDone is called when the kernel is finished with a zero-copy buffer.
// It decrements the block's reference count and releases it back to the pool if
// the count drops to zero and it was previously marked for eviction.
func (p *BufferedReader) handleZeroCopyDone(entry *blockQueueEntry) {
	p.mu.Lock()
	defer p.mu.Unlock()

	blk := entry.block
	blk.DecrementRef()
	if blk.RefCount() == 0 && blk.WasEvicted() {
		p.releaseBlock(entry)
	}
}

// prepareQueueForOffset discards blocks from the head of the prefetch queue
// that are no longer needed because the current read offset has moved past them.
// Discarded blocks are moved to the retiredBlocks cache, allowing them to be
// reused by other concurrent or slightly out-of-order reads without being
// re-downloaded.
// LOCKS_REQUIRED(p.mu)
func (p *BufferedReader) prepareQueueForOffset(offset int64) {
	for !p.blockQueue.IsEmpty() {
		entry := p.blockQueue.Peek()
		block := entry.block
		blockStart := block.AbsStartOff()
		blockEnd := blockStart + block.Cap()

		if offset < blockStart || offset >= blockEnd {
			// Offset is either before or beyond this block – discard.
			// The block is moved to the retired cache.
			p.blockQueue.Pop()
			p.retireBlock(entry)
		} else {
			break
		}
	}
}

// ReadAt reads data from the GCS object into the provided buffer starting at
// the given offset. It implements the gcsx.Reader interface.
// The read process is as follows:
//  1. It first handles any random read patterns, which may result in falling
//     back to another reader.
//  2. It prepares the prefetch queue by discarding any blocks from the
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

	isRandom, shouldFallback := p.patternDetector.check(&patternDetectorCheck{
		Offset:        off,
		Queue:         p.blockQueue,
		RetiredBlocks: p.retiredBlocks,
	})
	if shouldFallback {
		logger.Warnf("Fallback to another reader for object %q, handle %d, at offset %d. Random seek count exceeded threshold %d.", p.object.Name, handleID, off, p.patternDetector.threshold())
		p.metricHandle.BufferedReadFallbackTriggerCount(1, "random_read_detected")
		return resp, gcsx.FallbackToAnotherReader
	}

	if isRandom {
		// On a random read, clear the prefetch queue by retiring all blocks.
		for !p.blockQueue.IsEmpty() {
			entry := p.blockQueue.Pop()
			p.retireBlock(entry)
		}
	}

	for bytesRead < len(inputBuf) {
		// Check if the required block is in the retired cache.
		blockIndex := off / p.config.PrefetchBlockSizeBytes
		if entry := p.retiredBlocks.LookUp(blockIndex); entry != nil {
			blk := entry.block

			status, waitErr := blk.AwaitReady(ctx)
			if waitErr != nil {
				err = fmt.Errorf("BufferedReader.ReadAt: AwaitReady from retired: %w", waitErr)
				break
			}
			if status.State != block.BlockStateDownloaded {
				p.retiredBlocks.Erase(blockIndex)
				p.blockPool.Release(blk)
				err = fmt.Errorf("BufferedReader.ReadAt: retired block not downloaded, state: %d", status.State)
				break
			}

			relOff := off - blk.AbsStartOff()
			n, readErr := blk.ReadAt(inputBuf[bytesRead:], relOff)
			bytesRead += n
			off += int64(n)

			if readErr != nil && !errors.Is(readErr, io.EOF) {
				err = fmt.Errorf("BufferedReader.ReadAt: block.ReadAt from retired: %w", readErr)
				break
			}
			if off >= int64(p.object.Size) {
				break
			}
			continue
		}
		p.prepareQueueForOffset(off)

		if p.blockQueue.IsEmpty() {
			if err = p.prefetcher.freshStart(off); err != nil {
				logger.Warnf("Fallback to another reader for object %q, handle %d, at offset %d, due to freshStart failure: %v", p.object.Name, handleID, off, err)
				p.metricHandle.BufferedReadFallbackTriggerCount(1, "insufficient_memory")
				return resp, gcsx.FallbackToAnotherReader
			}
		}

		entry := p.blockQueue.Peek()

		// Proactively trigger the next prefetch as soon as we start processing a
		// block, ensuring the pipeline stays full.
		if !entry.prefetchTriggered {
			p.prefetcher.prefetch()
			entry.prefetchTriggered = true
		}

		status, waitErr := entry.block.AwaitReady(ctx)

		blk := entry.block

		if waitErr != nil {
			err = fmt.Errorf("BufferedReader.ReadAt: AwaitReady: %w", waitErr)
			break
		}

		if status.State != block.BlockStateDownloaded {
			p.blockQueue.Pop() // The block is invalid, remove it.
			p.releaseBlock(entry)

			switch status.State {
			case block.BlockStateDownloadFailed:
				err = fmt.Errorf("BufferedReader.ReadAt: download failed: %w", status.Err)
			default:
				err = fmt.Errorf("BufferedReader.ReadAt: unexpected block state: %d", status.State)
			}
			break
		}

		// On the first iteration, check if the read can be satisfied from a single
		// block without copying.
		// A zero-copy read is only possible if the entire request can be fulfilled
		// by the current block's buffer.
		if bytesRead == 0 && len(inputBuf) <= int(blk.Size()) {
			zeroCopyResp, zeroCopySuccess := p.tryZeroCopyRead(entry, off, inputBuf)
			if zeroCopySuccess {
				return zeroCopyResp, nil
			}
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
			entry := p.blockQueue.Pop()
			p.retireBlock(entry)
		}
	}

	resp.Size = bytesRead
	return resp, err
}

// LOCKS_EXCLUDED(p.mu)
func (p *BufferedReader) Destroy() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for !p.blockQueue.IsEmpty() {
		entry := p.blockQueue.Pop()
		if entry.block.RefCount() == 0 {
			p.releaseBlock(entry)
		}
	}

	// Clear the retired blocks cache and release all blocks.
	if p.retiredBlocks != nil {
		evictedEntries := p.retiredBlocks.Clear()
		for _, evictedEntry := range evictedEntries {
			// If the block is not in use, we can release it.
			if evictedEntry.block.RefCount() == 0 {
				p.releaseBlock(evictedEntry)
			}
		}
		p.retiredBlocks = nil
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

// releaseBlock cancels the download if in progress, waits for it to complete,
// and releases the block back to the pool.
// LOCKS_REQUIRED(p.mu)
func (p *BufferedReader) releaseBlock(entry *blockQueueEntry) {
	entry.cancel()
	// We wait for the block's worker goroutine to finish. We expect its
	// status to contain a context.Canceled error because we just called cancel.
	status, err := entry.block.AwaitReady(context.Background())
	if err != nil {
		logger.Warnf("releaseBlock: AwaitReady for block failed: %v", err)
	} else if status.Err != nil && !errors.Is(status.Err, context.Canceled) {
		logger.Warnf("releaseBlock: waiting for block on destroy: %v", status.Err)
	}
	p.blockPool.Release(entry.block)
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
	if p.patternDetector.isAboveThreshold() {
		panic(fmt.Sprintf("BufferedReader: random seek count has exceeded threshold %d", p.patternDetector.threshold()))
	}

	if p.retiredBlocks == nil {
		panic("BufferedReader: retiredBlocks is nil")
	}
}
