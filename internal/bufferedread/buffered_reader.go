package bufferedread

import (
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/googlecloudplatform/gcsfuse/v3/common"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/block"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/workerpool"
	"golang.org/x/net/context"
)

const (
	// RandomReadsThreshold is the number of random reads after which the reader
	// will fall back to a GCS-throttled reader.
	RandomReadsThreshold = 3
)

type BufferedReadConfig struct {
	PrefetchQueueCapacity   int64 // Maximum number of blocks that can be prefetched.
	PrefetchBlockSizeBytes  int64 // Size of each block to be prefetched.
	InitialPrefetchBlockCnt int64 // Number of blocks to prefetch initially.
	PrefetchMultiplier      int64 // Multiplier for number of blocks to prefetch.
}

type BufferedReader struct {
	object *gcs.MinObject
	bucket gcs.Bucket
	config *BufferedReadConfig

	nextBlockIndexToPrefetch int64
	randomSeekCount          int64
	nextPrefetchBlockCount   int64

	taskQueue TaskQueue

	readHandle atomic.Pointer[[]byte] // For zonal bucket optimization.

	blockPool    *block.BlockPool
	workerPool   workerpool.WorkerPool
	metricHandle common.MetricHandle

	ctx        context.Context
	cancelFunc context.CancelFunc
}

func (p *BufferedReader) Close() {
	p.Destroy()
}

func NewBufferedReader(object *gcs.MinObject, bucket gcs.Bucket, config *BufferedReadConfig, blockPool *block.BlockPool, workerPool workerpool.WorkerPool, metricHandle common.MetricHandle) *BufferedReader {
	reader := &BufferedReader{
		object:       object,
		bucket:       bucket,
		config:       config,
		taskQueue:    NewTaskQueue(),
		blockPool:    blockPool,
		workerPool:   workerPool,
		metricHandle: metricHandle,
	}

	reader.ctx, reader.cancelFunc = context.WithCancel(context.Background())
	return reader
}

func (p *BufferedReader) ReadAt(ctx context.Context, inputBuffer []byte, offset int64) (gcsx.ReaderResponse, error) {
	readerResponse := gcsx.ReaderResponse{
		DataBuf: inputBuffer,
	}
	requestID := uuid.New()
	startTime := time.Now()
	blockIndex := offset / p.config.PrefetchBlockSizeBytes

	var n int
	var err error

	logger.Tracef("[%s] ReadAt(offset=%d, len=%d, blockIndex=%d)", requestID, offset, len(inputBuffer), blockIndex)

	if offset >= int64(p.object.Size) {
		return readerResponse, io.EOF
	}
	if len(inputBuffer) == 0 {
		return readerResponse, nil
	}

	defer func() {
		if err != nil && err != io.EOF {
			logger.Errorf("%.10v -> ReadAt(offset=%d, len=%d, blockIndex=%d) with error: %v", requestID, offset, len(inputBuffer), blockIndex, err)
		} else {
			logger.Tracef("%.10v -> ReadAt(offset=%d, len=%d, blockIndex=%d): read %d bytes, ok(%v)", requestID, offset, len(inputBuffer), blockIndex, n, time.Since(startTime))
		}
	}()

	// Detect random reads and fallback if the threshold is exceeded.
	isReset := p.shouldResetPrefetcher(offset)
	isRandom := isReset || (p.taskQueue.IsEmpty() && offset != 0)
	if isRandom {
		p.randomSeekCount++
	}

	if p.randomSeekCount > RandomReadsThreshold {
		logger.Tracef("Fallback to GCS reader for object %q due to too many random reads (%d > %d).",
			p.object.Name, p.randomSeekCount, RandomReadsThreshold)
		return readerResponse, gcsx.FallbackToAnotherReader
	}

	if isReset {
		if err = p.resetPrefetcher(); err != nil {
			return readerResponse, fmt.Errorf("failed to reset prefetcher: %w", err)
		}
	}

	n, err = p.readBlocksAt(ctx, inputBuffer, offset)
	readerResponse.Size = n
	return readerResponse, err
}

func (p *BufferedReader) shouldResetPrefetcher(offset int64) bool {
	if p.taskQueue.IsEmpty() {
		return false
	}
	startOffset := p.taskQueue.PeekStart().block.AbsStartOff()
	endOffset := p.taskQueue.PeekEnd().block.AbsStartOff() + p.config.PrefetchBlockSizeBytes
	return offset < startOffset || offset >= endOffset
}

func (p *BufferedReader) readBlocksAt(ctx context.Context, inputBuffer []byte, offset int64) (int, error) {
	var bytesRead int
	prefetchTriggered := false

	for bytesRead < len(inputBuffer) {
		if err := p.cleanupStaleBlocks(ctx, offset); err != nil {
			return bytesRead, err
		}

		if p.taskQueue.IsEmpty() {
			if err := p.freshStart(offset); err != nil {
				return bytesRead, err
			}
			prefetchTriggered = true
		}

		task := p.taskQueue.PeekStart()
		b := task.block

		status, waitErr := b.AwaitReady(ctx)
		if waitErr != nil {
			return bytesRead, fmt.Errorf("error waiting for block: %w", waitErr)
		}

		if status != block.BlockStatusDownloaded {
			p.taskQueue.Pop()
			p.blockPool.Release(b)
			continue
		}

		readOffset := offset - b.AbsStartOff()
		n, readErr := b.ReadAt(inputBuffer[bytesRead:], readOffset)
		bytesRead += n
		offset += int64(n)

		if readErr != nil && readErr != io.EOF {
			return bytesRead, readErr
		}

		if offset >= int64(p.object.Size) {
			return bytesRead, io.EOF
		}

		// A block is consumed if the new offset is at or beyond its end.
		// We check this to decide whether to pop the block and trigger a prefetch.
		blockConsumed := offset >= b.AbsStartOff()+b.Cap()

		if blockConsumed {
			consumedTask := p.taskQueue.Pop()
			p.blockPool.Release(consumedTask.block)

			if !prefetchTriggered {
				prefetchTriggered = true
				if err := p.prefetch(); err != nil {
					logger.Warnf("readBlocksAt: prefetch failed: %v", err)
				}
			}
		}
	}
	return bytesRead, nil
}

func (p *BufferedReader) cleanupStaleBlocks(ctx context.Context, offset int64) error {
	for !p.taskQueue.IsEmpty() {
		task := p.taskQueue.PeekStart()
		if task.block.AbsStartOff()+task.block.Cap() <= offset {
			task = p.taskQueue.Pop()
			task.Cancel()

			if _, err := task.block.AwaitReady(ctx); err != nil && err != context.Canceled {
				return fmt.Errorf("error cleaning up stale block: %w", err)
			}
			p.blockPool.Release(task.block)
		} else {
			break
		}
	}
	return nil
}

func (p *BufferedReader) prefetch() error {
	// Do not schedule more blocks if the queue is already at capacity.
	availableCapacity := p.config.PrefetchQueueCapacity - int64(p.taskQueue.Len())
	if availableCapacity <= 0 {
		return nil
	}

	// Determine the number of blocks for this prefetch operation, respecting
	// both the multiplicative growth and the available capacity.
	blockCountToPrefetch := min(p.nextPrefetchBlockCount, availableCapacity)
	if blockCountToPrefetch <= 0 {
		return nil
	}

	logger.Debugf("Prefetching %d blocks", blockCountToPrefetch)

	for i := int64(0); i < blockCountToPrefetch; i++ {
		if p.nextBlockIndexToPrefetch >= p.maxBlockCount() {
			break
		}
		if err := p.scheduleNextBlock(false); err != nil {
			return err
		}
	}
	// Set the size for the next multiplicative prefetch.
	p.nextPrefetchBlockCount *= p.config.PrefetchMultiplier
	if p.nextPrefetchBlockCount > p.config.PrefetchQueueCapacity {
		p.nextPrefetchBlockCount = p.config.PrefetchQueueCapacity
	}
	return nil
}

func (p *BufferedReader) freshStart(currentOffset int64) error {
	blockIndex := currentOffset / p.config.PrefetchBlockSizeBytes
	p.nextBlockIndexToPrefetch = blockIndex

	// Determine the number of blocks for the initial prefetch.
	numToPrefetch := p.config.InitialPrefetchBlockCnt
	if numToPrefetch <= 0 {
		numToPrefetch = 1 // Default to at least 1.
	}
	// But don't prefetch more than the total capacity.
	numToPrefetch = min(numToPrefetch, p.config.PrefetchQueueCapacity)

	// Schedule the initial blocks.
	for i := int64(0); i < numToPrefetch; i++ {
		if p.nextBlockIndexToPrefetch >= p.maxBlockCount() {
			break
		}
		// The first block is considered urgent to unblock the current read.
		isUrgent := (i == 0)
		if err := p.scheduleNextBlock(isUrgent); err != nil {
			return fmt.Errorf("freshStart: initial scheduling failed: %w", err)
		}
	}

	// Set the size for the next multiplicative prefetch.
	p.nextPrefetchBlockCount = numToPrefetch * p.config.PrefetchMultiplier
	if p.nextPrefetchBlockCount > p.config.PrefetchQueueCapacity {
		p.nextPrefetchBlockCount = p.config.PrefetchQueueCapacity
	}
	return nil
}

func (p *BufferedReader) scheduleNextBlock(urgent bool) error {
	b, err := p.blockPool.Get()
	if b == nil || err != nil {
		return fmt.Errorf("unable to allocate block: %v", err)
	}

	p.scheduleBlockWithIndex(b, p.nextBlockIndexToPrefetch, urgent)
	p.nextBlockIndexToPrefetch++

	return nil
}

func (p *BufferedReader) scheduleBlockWithIndex(b block.Block, blockIndex int64, urgent bool) {

	startOffset := blockIndex * p.config.PrefetchBlockSizeBytes
	if err := b.SetAbsStartOff(startOffset); err != nil {
		logger.Errorf("Failed to set start offset on block: %v", err)
		p.blockPool.Release(b)
		return
	}

	rhPtr := p.readHandle.Load()
	var rh []byte
	if rhPtr != nil {
		rh = *rhPtr
	}
	task := NewDownloadTask(p.ctx, p.object, p.bucket, b, rh)

	logger.Debugf("Scheduling block (%s, offset %d).", p.object.Name, startOffset)

	p.taskQueue.Push(task)
	p.workerPool.Schedule(urgent, task)
}

func (p *BufferedReader) Destroy() {
	if p.cancelFunc != nil {
		p.cancelFunc()
		p.cancelFunc = nil
	}

	for !p.taskQueue.IsEmpty() {
		task := p.taskQueue.Pop()
		task.Cancel()
		// We expect a context.Canceled error here, but we wait to ensure the
		// block's worker goroutine has finished before releasing the block.
		if _, err := task.block.AwaitReady(p.ctx); err != nil && err != context.Canceled {
			logger.Warnf("bufferedread: error waiting for block on destroy: %v", err)
		}
		p.blockPool.Release(task.block)
	}

	p.blockPool = nil
}

func (p *BufferedReader) resetPrefetcher() error {
	var firstErr error
	for !p.taskQueue.IsEmpty() {
		task := p.taskQueue.Pop()
		task.Cancel()
		// Use a background context because p.ctx is the overall reader context,
		// and we are just clearing the prefetch queue, not closing the reader.
		if _, err := task.block.AwaitReady(context.Background()); err != nil && err != context.Canceled && firstErr == nil {
			firstErr = fmt.Errorf("bufferedread: error waiting for block on reset: %w", err)
			logger.Warnf("%v", firstErr)
		}
		p.blockPool.Release(task.block)
	}
	return firstErr
}

func (p *BufferedReader) maxBlockCount() int64 {
	if p.config.PrefetchBlockSizeBytes <= 0 {
		// A non-positive chunk size is an invalid configuration.
		// Log a warning and return 0 to prevent division by zero.
		logger.Warnf("Invalid PrefetchChunkSizeBytes (%d); must be positive.", p.config.PrefetchBlockSizeBytes)
		return 0
	}
	return (int64(p.object.Size) + p.config.PrefetchBlockSizeBytes - 1) / p.config.PrefetchBlockSizeBytes
}

// CheckInvariants checks for internal consistency of the reader.
func (p *BufferedReader) CheckInvariants() {
	if p.blockPool == nil {
		panic("BufferedReader: blockPool is nil")
	}
	if p.workerPool == nil {
		panic("BufferedReader: workerPool is nil")
	}
	if p.taskQueue == nil {
		panic("BufferedReader: taskQueue is nil")
	}
	// The number of items in the queue should not exceed the configured capacity.
	if int64(p.taskQueue.Len()) > p.config.PrefetchQueueCapacity {
		panic(fmt.Sprintf("BufferedReader: taskQueue length %d exceeds capacity %d", p.taskQueue.Len(), p.config.PrefetchQueueCapacity))
	}
}
