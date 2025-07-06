// Copyright 2024 Google LLC
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
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/googlecloudplatform/gcsfuse/v3/common"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/block"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/workerpool"
	"golang.org/x/net/context"
)

type BufferedReadConfig struct {
	PrefetchCount           int64
	PrefetchChunkSize       int64
	InitialPrefetchBlockCnt int64
	PrefetchMultiplier      int64
}

func getDefaultBufferedReadConfig() *BufferedReadConfig {
	return &BufferedReadConfig{
		PrefetchCount:           6,
		PrefetchChunkSize:       10 * int64(1024*1024),
		InitialPrefetchBlockCnt: 1,
		PrefetchMultiplier:      2,
	}
}

type BufferedReader struct {
	object              *gcs.MinObject
	bucket              gcs.Bucket
	config              *BufferedReadConfig
	lastReadOffset      int64
	nextBlockToPrefetch int64

	randomSeekCount   int64
	lastPrefetchCount int64

	// blockIndexMap map[int64]*Block
	blockQueue common.Queue[BlockQueueEntry]

	readHandle []byte // For zonal bucket.

	// blockPool  *BlockPool
	// workerPool *WorkerPool
	blockPool  *block.PrefetchBlockPool
	workerPool workerpool.WorkerPool

	metricHandle common.MetricHandle

	// Create a cancellable context and cancellable function
	cancelFunc context.CancelFunc
	ctx        context.Context
}

func (p *BufferedReader) Close() error {
	p.Destroy()
	return nil

}

func NewBufferedReader(object *gcs.MinObject, bucket gcs.Bucket, config *BufferedReadConfig, blockPool *block.PrefetchBlockPool, workerPool workerpool.WorkerPool) *BufferedReader {
	bufferedReader := &BufferedReader{
		object:              object,
		bucket:              bucket,
		config:              config,
		lastReadOffset:      -1,
		nextBlockToPrefetch: 0,
		randomSeekCount:     0,
		blockQueue:          common.NewLinkedListQueue[BlockQueueEntry](),
		blockPool:           blockPool,
		workerPool:          workerPool,
		metricHandle:        nil,
	}

	bufferedReader.ctx, bufferedReader.cancelFunc = context.WithCancel(context.Background())
	return bufferedReader
}

/**
 * ReadAt reads the data from the object at the given offset.
 * It first checks if the block containing the offset is already in memory.
 * If it is, it reads the data from the block.
 * If it is not, it downloads the block from GCS and then reads the data from the block.
 */
func (p *BufferedReader) ReadAt(ctx context.Context, inputBuffer []byte, offset int64) (n int64, err error) {
	// Generate an unique id in the request
	requestId := uuid.New()
	stime := time.Now()
	blockIndex := offset / p.config.PrefetchChunkSize

	logger.Tracef("%.10v <- ReadAt(%d, %d, %d).", requestId, offset, len(inputBuffer), blockIndex)

	defer func() {
		if err != nil && err != io.EOF {
			logger.Errorf("%.10v -> ReadAt(%d, %d, %d) with error: %v", requestId, offset, len(inputBuffer), blockIndex, err)
		} else {
			logger.Tracef("%.10v -> ReadAt(%d, %d, %d): ok(%v)", requestId, offset, len(inputBuffer), blockIndex, time.Since(stime))
		}
	}()

	if offset >= int64(p.object.Size) {
		err = io.EOF
		return
	}
	prefetchHappened := false

	dataRead := int(0)
	for dataRead < len(inputBuffer) {
		for !p.blockQueue.IsEmpty() {
			curNode := p.blockQueue.Peek()
			cur := curNode.block
			startOffset := cur.AbsStartOff()
			endOffset := startOffset + cur.Size()
			if uint64(startOffset) > uint64(offset) || uint64(endOffset) <= uint64(offset) {
				curNode = p.blockQueue.Pop()
				curNode.cancelFunc() // Cancel the download if it is in progress
				_, err := curNode.block.AwaitReady(ctx)
				if err != nil {
					return 0, fmt.Errorf("ReadAt: error waiting for block download: %v", err)
				}
				p.blockPool.Release(cur)
				continue
			} else {
				break
			}
		}

		if p.blockQueue.IsEmpty() {
			p.freshStart(offset)
			prefetchHappened = true
		}

		blk := p.blockQueue.Peek().block
		status, waiterr := blk.AwaitReady(ctx)
		if waiterr != nil {
			return 0, fmt.Errorf("ReadAt: error waiting for block download: %v", waiterr)
		}

		switch status {
		case block.BlockStatusDownloaded:
		case block.BlockStatusDownloadFailed:
			return 0, fmt.Errorf("Block download failed")
		case block.BlockStatusDownloadCancelled:
			return 0, fmt.Errorf("Block download unexpected cancelled")
		default:
		}

		readOffset := uint64(offset) - uint64(blk.AbsStartOff())
		nn, readErr := blk.ReadAt(inputBuffer[dataRead:], int64(readOffset))
		if readErr != nil && readErr != block.PartialBlockReadErr {
			return 0, fmt.Errorf("ReadAt: error reading from block: %v", readErr)
		}
		offset += int64(nn)
		dataRead += int(nn)

		if offset >= int64(p.object.Size) {
			n = int64(dataRead)
			err = io.EOF
			return
		}

		if nn == 0 {
			n = int64(dataRead)
			return n, io.EOF
		}

		if dataRead < len(inputBuffer) || (dataRead == len(inputBuffer) && offset == int64(blk.AbsStartOff()+blk.Size())) {
			consumedBlock := p.blockQueue.Pop()
			p.blockPool.Release(consumedBlock.block)

			if !prefetchHappened {
				prefetchHappened = true
				err = p.prefetch()
				if err != nil {
					return
				}
			}

		}
	}
	n = int64(dataRead)
	return
}

func (p *BufferedReader) prefetch() error {
	nextPrefetchCount := p.config.PrefetchMultiplier * p.lastPrefetchCount
	nextPrefetchCount = min(nextPrefetchCount, p.config.PrefetchCount-int64(p.blockQueue.Len()))

	logger.Debugf("Next Prefetch Count: %d", nextPrefetchCount)

	p.lastPrefetchCount = nextPrefetchCount

	for i := 0; i < int(nextPrefetchCount) && p.nextBlockToPrefetch < p.maxBlockCount(); i++ {
		err := p.scheduleNextBlock(true)
		if err != nil {
			return err
		}
	}

	return nil
}

// freshStart to start from a given offset.
func (p *BufferedReader) freshStart(currentSeek int64) error {
	p.resetPrefetcher()
	blockIndex := currentSeek / p.config.PrefetchChunkSize
	p.nextBlockToPrefetch = blockIndex

	err := p.scheduleNextBlock(false)
	if err != nil {
		return fmt.Errorf("freshStart: initial scheduling failed with %d", err)
	}

	p.lastPrefetchCount = 1

	err = p.prefetch()
	if err != nil {
		return fmt.Errorf("freshStart: prefetch failed with %d", err)
	}

	return nil
}

func (p *BufferedReader) scheduleNextBlock(prefetch bool) error {
	block, err := p.blockPool.Get()
	if block == nil || err != nil {
		return fmt.Errorf("unable to allocate block")
	}

	p.scheduleBlockWithIndex(block, p.nextBlockToPrefetch, prefetch)
	p.nextBlockToPrefetch++

	return nil
}

// scheduleBlock create a prefetch task for the given block and schedule it to thread-pool.
func (p *BufferedReader) scheduleBlockWithIndex(block block.PrefetchBlock, blockIndex int64, prefetch bool) {
	blockCtx, cancel := context.WithCancel(p.ctx)

	// p.prepareBlock(blockIndex, block)
	block.SetAbsStartOff(blockIndex * p.config.PrefetchChunkSize)
	task := &PrefetchTask{
		ctx:      blockCtx,
		object:   p.object,
		bucket:   p.bucket,
		block:    block,
		prefetch: prefetch,
	}

	logger.Tracef("Scheduling block (%s, %d, %v).", p.object.Name, blockIndex, prefetch)
	queueBlockEntry := BlockQueueEntry{
		block:      block,
		task:       task,
		cancelFunc: cancel,
	}

	// Queue has always a right cancel function.
	// block.cancelFunc = cancel

	p.blockQueue.Push(queueBlockEntry)
	p.workerPool.Schedule(!prefetch, task)
}

// prepareBlock initializes block-state according to blockIndex.
// func (p *BufferedReader) prepareBlock(blockIndex int64, block *Block) {
// 	block.id = blockIndex
// 	block.offset = uint64(blockIndex * p.config.PrefetchChunkSize)
// 	block.writeSeek = 0
// 	block.endOffset = min(block.offset+p.blockPool.blockSize, uint64(p.object.Size))
// }

// Destroy cancels/discards the blocks in the blockQueue.
func (p *BufferedReader) Destroy() {
	if p.cancelFunc != nil {
		p.cancelFunc()
		p.cancelFunc = nil
	}

	// Cancel all the processed or in process queue in the queue.
	for !p.blockQueue.IsEmpty() {
		blockQueueEntry := p.blockQueue.Pop()
		blockQueueEntry.cancelFunc()

		_, _ = blockQueueEntry.block.AwaitReady(p.ctx)

		p.blockPool.Release(blockQueueEntry.block)
	}
}

func (p *BufferedReader) resetPrefetcher() {
	// Cancel all the processed or in process queue in the queue.
	for !p.blockQueue.IsEmpty() {
		blockQueueEntry := p.blockQueue.Pop()
		blockQueueEntry.cancelFunc()

		_, _ = blockQueueEntry.block.AwaitReady(p.ctx)

		p.blockPool.Release(blockQueueEntry.block)
	}
}

func (p *BufferedReader) maxBlockCount() int64 {
	return (int64(p.object.Size) + p.config.PrefetchChunkSize - 1) / p.config.PrefetchChunkSize
}
