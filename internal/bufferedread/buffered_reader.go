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
	RandomSeekThreshold     int64 // Seek count threshold to switch another reader.
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

	// A waitgroup to track all in-flight zero-copy read operations.
	zeroCopyWg sync.WaitGroup

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

	prefetcherOpts := &prefetcherOptions{
		Object:       opts.Object,
		Bucket:       opts.Bucket,
		Config:       opts.Config,
		Pool:         blockpool,
		WorkerPool:   opts.WorkerPool,
		Queue:        reader.blockQueue,
		MetricHandle: opts.MetricHandle,
		ReaderCtx:    reader.ctx,
		ReadHandle:   reader.readHandle,
	}
	reader.prefetcher = newPrefetcher(prefetcherOpts)

	return reader, nil
}

// handleZeroCopyDone is called when the kernel is finished with a zero-copy buffer.
// It decrements the block's reference count and releases it back to the pool if
// the count drops to zero and it was previously marked for eviction.
func (p *BufferedReader) handleZeroCopyDone(entry *blockQueueEntry) {
	defer p.zeroCopyWg.Done()
	p.mu.Lock()
	defer p.mu.Unlock()

	blk := entry.block
	if blk.DecrementRef() == 0 && blk.WasEvicted() {
		// The block is no longer referenced by the kernel and has already been
		// marked for eviction from the active queue. It is now safe to release
		// it back to the pool.
		p.blockPool.Release(entry.block)
	}
}

// prepareQueueForOffset discards blocks from the head of the prefetch queue
// that are no longer needed because the current read offset has moved past them.
// Discarded blocks are released, allowing them to be
// reused by other reads without being
// re-downloaded.
// LOCKS_REQUIRED(p.mu)
func (p *BufferedReader) prepareQueueForOffset(offset int64) {
	for !p.blockQueue.IsEmpty() {
		entry := p.blockQueue.Peek()
		block := entry.block
		blockStart := block.AbsStartOff()
		blockEnd := blockStart + block.Cap()

		if offset >= blockStart && offset < blockEnd {
			// The read offset is within the bounds of the current block.
			break
		}
		// Offset is before or beyond this block – discard.
		p.blockQueue.Pop()
		p.releaseBlock(entry)
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
	resp := gcsx.ReaderResponse{}
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
		Offset: off,
		Queue:  p.blockQueue,
	})
	if shouldFallback {
		logger.Warnf("Fallback to another reader for object %q, handle %d, at offset %d. Random seek count exceeded threshold %d.", p.object.Name, handleID, off, p.patternDetector.threshold())
		p.metricHandle.BufferedReadFallbackTriggerCount(1, "random_read_detected")
		return resp, gcsx.FallbackToAnotherReader
	}

	if isRandom {
		// On a random read, clear the prefetch queue by retiring all blocks.
		p.prepareQueueForOffset(-1) // Discard all blocks.
	}

	var data [][]byte
	for bytesRead < len(inputBuf) {
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

		relOff := off - blk.AbsStartOff()
		bytesToRead := len(inputBuf) - bytesRead
		slice, n, readErr := blk.ReadAtSlice(bytesToRead, relOff)
		bytesRead += n
		off += int64(n)

		if readErr != nil && !errors.Is(readErr, io.EOF) {
			err = fmt.Errorf("BufferedReader.ReadAt: block.ReadAt: %w", readErr)
			break
		}

		if n > 0 {
			data = append(data, slice)
			p.zeroCopyWg.Add(1)
			blk.IncrementRef()
			resp.Done = func() { p.handleZeroCopyDone(entry) }
		}

		// The read is complete if the buffer is full or we have reached the end of the object.
		if bytesRead >= len(inputBuf) || off >= int64(p.object.Size) {
			break
		}

		if off >= blk.AbsStartOff()+blk.Size() {
			entry := p.blockQueue.Pop()
			// The block is fully consumed, release it.
			p.releaseBlock(entry)
		}
	}

	resp.Size = bytesRead
	resp.Data = data
	return resp, err
}

// LOCKS_EXCLUDED(p.mu)
func (p *BufferedReader) Destroy() {
	p.mu.Lock()
	for !p.blockQueue.IsEmpty() {
		entry := p.blockQueue.Pop()
		if entry.block.RefCount() == 0 {
			p.releaseBlock(entry)
		} else {
			entry.block.SetWasEvicted(true)
		}
	}
	p.mu.Unlock()

	// Wait for any remaining in-flight zero-copy operations to complete, with a
	// timeout to prevent indefinite blocking. Their Done callbacks will handle
	// the final release of those blocks.
	done := make(chan struct{})
	go func() {
		defer close(done)
		p.zeroCopyWg.Wait()
	}()

	select {
	case <-done:
		// Wait completed successfully.
	case <-time.After(10 * time.Second):
		logger.Warnf("BufferedReader.Destroy: timed out waiting for zero-copy operations to complete.")
	}

	if p.cancelFunc != nil {
		p.cancelFunc()
		p.cancelFunc = nil
	}

	if err := p.blockPool.ClearFreeBlockChannel(true); err != nil {
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
	if err != nil || (status.Err != nil && !errors.Is(status.Err, context.Canceled)) {
		logger.Warnf("releaseBlock: waiting for block on destroy: %v", status.Err)
	}

	// If the block is still referenced (e.g., by a zero-copy read), do not
	// release it to the pool. Mark it as evicted so the Done callback can clean it up.
	if entry.block.RefCount() > 0 {
		entry.block.SetWasEvicted(true)
	} else {
		p.blockPool.Release(entry.block)
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
	if p.patternDetector.isAboveThreshold() {
		panic(fmt.Sprintf("BufferedReader: random seek count has exceeded threshold %d", p.patternDetector.threshold()))
	}
}
