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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/block"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
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
)

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

	// handleID is the file handle id, used for logging.
	handleID fuseops.HandleID

	// inodeID is the inode id, used for cache key generation.
	inodeID fuseops.InodeID

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

	// blockCache is the core of the prefetching pipeline, holding blocks that are
	// either downloaded or in the process of being downloaded, managed by LRU policy.
	// GUARDED by (mu)
	blockCache *lru.Cache

	// blockPool is a pool of blocks that can be reused for prefetching.
	// It is used to avoid allocating new blocks for each prefetch operation.
	// The pool is initialized with a maximum number of blocks that can be
	// prefetched at a time, and it allows for efficient reuse of blocks.
	// The pool is also responsible for managing the global limit on the number
	// of blocks that can be allocated across all BufferedReader instances.
	// GUARDED by (mu)
	blockPool *block.GenBlockPool[block.PrefetchBlock]

	// GUARDED by (mu)
	globalMaxBlocksSem *semaphore.Weighted

	// A WaitGroup to synchronize the destruction of the reader with any ongoing
	// FUSE read callback goroutines. This ensures that all callbacks for
	// in-flight data slices have completed before the reader is fully torn down.
	inflightCallbackWg sync.WaitGroup

	// readTypeClassifier tracks the read access pattern (e.g., sequential, random)
	// to optimize read strategies. It is shared across different reader layers.
	readTypeClassifier *gcsx.ReadTypeClassifier
}

// BufferedReaderOptions holds the dependencies for a BufferedReader.
type BufferedReaderOptions struct {
	Object             *gcs.MinObject
	Bucket             gcs.Bucket
	Config             *BufferedReadConfig
	GlobalMaxBlocksSem *semaphore.Weighted
	WorkerPool         workerpool.WorkerPool
	MetricHandle       metrics.MetricHandle
	ReadTypeClassifier *gcsx.ReadTypeClassifier
	BlockCache         *lru.Cache
	HandleID           fuseops.HandleID
	BlockPool          *block.GenBlockPool[block.PrefetchBlock]
	InodeID            fuseops.InodeID
}

// NewBufferedReader returns a new bufferedReader instance.
func NewBufferedReader(opts *BufferedReaderOptions) (*BufferedReader, error) {
	if opts.Config.PrefetchBlockSizeBytes <= 0 {
		return nil, fmt.Errorf("NewBufferedReader: PrefetchBlockSizeBytes must be positive, but is %d", opts.Config.PrefetchBlockSizeBytes)
	}

	reader := &BufferedReader{
		object:                   opts.Object,
		bucket:                   opts.Bucket,
		config:                   opts.Config,
		nextBlockIndexToPrefetch: 0,
		randomSeekCount:          0,
		numPrefetchBlocks:        opts.Config.InitialPrefetchBlockCnt,
		blockCache:               opts.BlockCache,
		blockPool:                opts.BlockPool,
		workerPool:               opts.WorkerPool,
		globalMaxBlocksSem:       opts.GlobalMaxBlocksSem,
		metricHandle:             opts.MetricHandle,
		handleID:                 opts.HandleID,
		inodeID:                  opts.InodeID,
		prefetchMultiplier:       defaultPrefetchMultiplier,
		randomReadsThreshold:     opts.Config.RandomSeekThreshold,
		readTypeClassifier:       opts.ReadTypeClassifier,
	}

	reader.ctx, reader.cancelFunc = context.WithCancel(context.Background())
	return reader, nil
}

// handleRandomRead evaluates the read pattern based on the given offset.
// If a random read is detected, it increments a counter. If the counter
// exceeds a threshold, it may trigger a fallback to a different reader,
// unless a sequential read pattern is re-established, in which case it
// resets the reader's state to resume efficient prefetching.
// LOCKS_REQUIRED(p.mu)
func (p *BufferedReader) handleRandomRead(offset int64) error {
	// Exit early if we have already decided to fall back to another reader.
	// This avoids re-evaluating the read pattern on every call when the random
	// read threshold has been met.
	if p.randomSeekCount > p.randomReadsThreshold {
		if p.readTypeClassifier.IsReadSequential() {
			logger.Tracef("Restarting buffered reader due to sequential read pattern detected for object %q, handle %d", p.object.Name, p.handleID)
			p.resetBufferedReaderState()
			return nil
		}
		return gcsx.FallbackToAnotherReader
	}

	if !p.isRandomSeek(offset) {
		return nil
	}

	p.randomSeekCount++

	if p.randomSeekCount > p.randomReadsThreshold {
		// If the read pattern becomes sequential again, reset the state to resume buffered reading.
		if p.readTypeClassifier.IsReadSequential() {
			logger.Tracef("Restarting buffered reader due to sequential read pattern detected for object %q, handle %d", p.object.Name, p.handleID)
			p.resetBufferedReaderState()
			return nil
		}
		logger.Warnf("Fallback to another reader for object %q, handle %d. Random seek count %d exceeded threshold %d and read pattern is not sequential.", p.object.Name, p.handleID, p.randomSeekCount, p.randomReadsThreshold)
		p.metricHandle.BufferedReadFallbackTriggerCount(1, "random_read_detected")
		return gcsx.FallbackToAnotherReader
	}

	return nil
}

// isRandomSeek checks if the read for the given offset is random or not.
// LOCKS_REQUIRED(p.mu)
func (p *BufferedReader) isRandomSeek(offset int64) bool {
	blockIndex := offset / p.config.PrefetchBlockSizeBytes
	key := p.generateCacheKey(blockIndex)
	// A read is considered random if the offset is not zero and the
	// corresponding block is not present in the cache.
	isBlockInCache := p.blockCache.LookUpWithoutChangingOrder(key) != nil
	return offset != 0 && !isBlockInCache
}

// ReadAt reads data from the GCS object into the provided buffer starting at
// the given offset. It implements the gcsx.Reader interface.
//
// The read is satisfied by reading from in-memory blocks that are prefetched
// in the background. The core logic is as follows:
//  1. Detect if the read pattern is random. If so, and if the random read
//     threshold is exceeded, it returns a FallbackToAnotherReader error.
//  2. Prepare the internal block queue by discarding any stale blocks from the
//     head of the queue that are before the requested offset. This is now handled by LRU.
//  3. If the required block is not in the cache, it initiates a "fresh start" to prefetch blocks.
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
func (p *BufferedReader) ReadAt(ctx context.Context, req *gcsx.ReadRequest) (gcsx.ReadResponse, error) {
	resp := gcsx.ReadResponse{}
	reqID := uuid.New()
	start := time.Now()
	readOffset := req.Offset
	blockIdx := readOffset / p.config.PrefetchBlockSizeBytes
	var bytesRead int
	var err error

	logger.Tracef("%.13v <- ReadAt(%s:/%s, %d, %d, %d, %d)", reqID, p.bucket.Name(), p.object.Name, p.handleID, readOffset, len(req.Buffer), blockIdx)

	if readOffset >= int64(p.object.Size) {
		err = io.EOF
		return resp, err
	}

	if len(req.Buffer) == 0 {
		return resp, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	defer func() {
		dur := time.Since(start)
		p.metricHandle.BufferedReadReadLatency(ctx, dur)
		p.metricHandle.GcsReadBytesCount(int64(bytesRead), metrics.ReaderBufferedAttr)
		if err == nil || errors.Is(err, io.EOF) {
			logger.Tracef("%.13v -> ReadAt(): Ok(%v)", reqID, dur)
		}
	}()

	if err = p.handleRandomRead(readOffset); err != nil {
		return resp, fmt.Errorf("BufferedReader.ReadAt: handleRandomRead: %w", err)
	}

	prefetchTriggered := false

	urgentFetchJustHappened := false
	// urgentFetch indicates if the current ReadAt call triggered an urgent download.
	var dataSlices [][]byte
	var entriesToCallback []*blockCacheEntry
	for bytesRead < len(req.Buffer) {
		currentBlockIndex := readOffset / p.config.PrefetchBlockSizeBytes
		key := p.generateCacheKey(currentBlockIndex)
		val := p.blockCache.LookUp(key)

		if val == nil {
			if err = p.freshStart(readOffset); err != nil {
				logger.Warnf("Fallback to another reader for object %q, handle %d, block %d, due to freshStart failure: %v", p.object.Name, p.handleID, currentBlockIndex, err)
				p.metricHandle.BufferedReadFallbackTriggerCount(1, "insufficient_memory")
				return resp, gcsx.FallbackToAnotherReader
			}
			// An urgent download was just triggered for the required block.
			urgentFetchJustHappened = true
			prefetchTriggered = true
			val = p.blockCache.LookUp(key)
			if val == nil {
				err = fmt.Errorf("BufferedReader.ReadAt: block %d not found after freshStart", currentBlockIndex)
				break
			}
		}

		entry := val.(*blockCacheEntry)
		blk := entry.block

		status, waitErr := blk.AwaitReady(ctx)
		if waitErr != nil {
			err = fmt.Errorf("BufferedReader.ReadAt: AwaitReady: %w", waitErr)
			break
		}

		if status.State != block.BlockStateDownloaded {
			p.blockCache.Erase(key)
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

		relOff := readOffset - blk.AbsStartOff()
		bytesToRead := len(req.Buffer) - bytesRead
		dataSlice, readErr := blk.ReadAtSlice(relOff, bytesToRead)
		sliceLen := len(dataSlice)
		bytesRead += sliceLen
		readOffset += int64(sliceLen)

		if readErr != nil && !errors.Is(readErr, io.EOF) {
			err = fmt.Errorf("BufferedReader.ReadAt: block.ReadAt: %w", readErr)
			break
		}

		if sliceLen > 0 {
			dataSlices = append(dataSlices, dataSlice)
			p.inflightCallbackWg.Add(1)
			// The ref count for an urgently fetched block is already incremented
			// during scheduling. We only increment here if it was a cache hit.
			if !urgentFetchJustHappened {
				blk.IncRef()
			}
			entriesToCallback = append(entriesToCallback, entry)
		}

		if readOffset >= int64(p.object.Size) {
			break
		}

		// Reset the flag after the first block is processed.
		urgentFetchJustHappened = false

		if readOffset >= blk.AbsStartOff()+blk.Size() {
			// Block is fully read, but we don't remove it from cache here. LRU will handle it.

			if !prefetchTriggered {
				prefetchTriggered = true
				if pfErr := p.prefetch(); pfErr != nil {
					logger.Warnf("BufferedReader.ReadAt: while prefetching: %v", pfErr)
				}
			}
		}
	}

	resp.Data = dataSlices
	resp.Callback = func() { p.callback(entriesToCallback) }
	resp.Size = bytesRead
	return resp, err
}

// callback is called when the FUSE library is finished with buffer slices that
// were returned directly from blocks. It decrements the reference count for each
// associated block and releases it back to the pool if the count drops to zero
// and it was previously marked for eviction.
func (p *BufferedReader) callback(entries []*blockCacheEntry) {
	defer func() {
		for range entries {
			p.inflightCallbackWg.Done()
		}
	}()
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, entry := range entries {
		if entry.block.DecRef() && entry.wasEvicted {
			p.blockPool.Release(entry.block)
		}
	}
}

// prefetch schedules the next set of blocks for prefetching starting from
// the nextBlockIndexToPrefetch.
// LOCKS_REQUIRED(p.mu)
func (p *BufferedReader) prefetch() error {
	// Determine the number of blocks to prefetch in this cycle, respecting the
	// MaxPrefetchBlockCnt and the number of blocks remaining in the file.
	availableSlots := p.config.MaxPrefetchBlockCnt - int64(p.blockCache.Len())
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
	// If the cache is full, evict some blocks to make space for new ones.
	if int64(p.blockCache.Len()) >= 30 {
		p.handleEvictedEntries(p.blockCache.Evict(10))
	}
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
	task := &downloadTask{
		ctx:          ctx,
		object:       p.object,
		bucket:       p.bucket,
		block:        b,
		readHandle:   p.readHandle,
		metricHandle: p.metricHandle,
	}

	logger.Tracef("Scheduling block: (%s, %d, %t).", p.object.Name, blockIndex, urgent)
	entry := &blockCacheEntry{
		block:  b,
		cancel: cancel,
	}

	if urgent {
		b.IncRef()
	}

	key := p.generateCacheKey(blockIndex)
	evicted, err := p.blockCache.Insert(key, entry)
	if err != nil {
		return fmt.Errorf("scheduleBlockWithIndex: inserting into cache: %w", err)
	}
	p.handleEvictedEntries(evicted)

	p.workerPool.Schedule(urgent, task)
	return nil
}

// LOCKS_EXCLUDED(p.mu)
func (p *BufferedReader) Destroy() {
	p.mu.Lock()
	evicted := p.blockCache.EraseAll()
	p.handleEvictedEntries(evicted)
	p.mu.Unlock()

	// Wait for any remaining operations where data slices were returned directly
	// to complete, with a timeout to prevent indefinite blocking. Their Done
	// callbacks will handle the final release of those blocks.
	done := make(chan struct{})
	go func() {
		defer close(done)
		p.inflightCallbackWg.Wait()
	}()

	select {
	case <-done:
		// Wait completed successfully.
	case <-time.After(10 * time.Second):
		logger.Warnf("BufferedReader.Destroy: timed out waiting for outstanding data slice references to be released.")
	}

	// After the wait, no new read calls can arrive. This is because the kernel
	// prevents read operations on a closed file handle, so no further locking is
	// necessary for the final cleanup.
	if p.cancelFunc != nil {
		p.cancelFunc()
		p.cancelFunc = nil
	}
}

func (p *BufferedReader) handleEvictedEntries(evicted []lru.ValueType) {
	for _, val := range evicted {
		if entry, ok := val.(*blockCacheEntry); ok {
			entry.cancelAndWait()
			p.releaseOrMarkEvicted(entry)
		}
	}
}

// releaseOrMarkEvicted handles the release of a block that has been removed
// from the prefetch queue. If the block has no outstanding references (i.e.,
// it has not been returned to a FUSE read), it is immediately returned to the
// block pool. Otherwise, the block is marked as evicted, and its final release
// is deferred until the last reference's callback is executed.
// LOCKS_REQUIRED(p.mu)
func (p *BufferedReader) releaseOrMarkEvicted(entry *blockCacheEntry) {
	// If the block still has outstanding references, do not release it to the
	// pool. Instead, mark it as evicted so the callback can release it later,
	// when the reference count drops to zero.
	if entry.block.RefCount() > 0 {
		entry.wasEvicted = true
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
	if int64(p.blockCache.Len()) > p.config.MaxPrefetchBlockCnt {
		panic(fmt.Sprintf("BufferedReader: blockCache length %d exceeds limit %d", p.blockCache.Len(), p.config.MaxPrefetchBlockCnt))
	}

	// The random seek count should never exceed randomReadsThreshold.
	if p.randomSeekCount > p.randomReadsThreshold {
		panic(fmt.Sprintf("BufferedReader: randomSeekCount %d exceeds threshold %d", p.randomSeekCount, p.randomReadsThreshold))
	}
}

// resetBufferedReaderState resets the internal state to restart buffered reading.
// LOCKS_REQUIRED(p.mu)
func (p *BufferedReader) resetBufferedReaderState() {
	// Reset the reader state
	p.randomSeekCount = 0
	p.nextBlockIndexToPrefetch = 0
	p.numPrefetchBlocks = p.config.InitialPrefetchBlockCnt
}

// generateCacheKey creates a unique key for a block based on the inode ID and block index.
func (p *BufferedReader) generateCacheKey(blockIndex int64) string {
	var sb strings.Builder
	sb.WriteString(strconv.FormatUint(uint64(p.inodeID), 10))
	sb.WriteString(strconv.FormatInt(blockIndex, 10))
	return sb.String()
}

// getLruKeys returns a comma-separated string of keys currently in the LRU cache.
// This is intended for logging and debugging purposes only.
// LOCKS_REQUIRED(p.mu)
func (p *BufferedReader) getLruKeys() string {
	if p.blockCache == nil {
		return ""
	}
	return p.blockCache.Keys()
}
