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

package prefetch

import (
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"golang.org/x/net/context"
)

const (
	min_prefetch = 10
)

type PrefetchConfig struct {
	PrefetchCount           int64
	PrefetchChunkSize       int64
	InitialPrefetchBlockCnt int64
	PrefetchMultiplier      int64
}

func getDefaultPrefetchConfig() *PrefetchConfig {
	return &PrefetchConfig{
		PrefetchCount:           6,
		PrefetchChunkSize:       10 * int64(_1MB),
		InitialPrefetchBlockCnt: 1,
		PrefetchMultiplier:      2,
	}
}

type PrefetchReader struct {
	object              *gcs.MinObject
	bucket              gcs.Bucket
	PrefetchConfig      *PrefetchConfig
	lastReadOffset      int64
	nextBlockToPrefetch int64

	randomSeekCount   int64
	lastPrefetchCount int64

	// blockIndexMap map[int64]*Block
	blockQueue *BlockQueue

	readHandle []byte // For zonal bucket.

	blockPool  *BlockPool
	threadPool *ThreadPool

	metricHandle common.MetricHandle

	// Create a cancellable context and cancellable function
	cancelFunc context.CancelFunc
	ctx        context.Context
}

func (p *PrefetchReader) Close() error {
	p.Destroy()
	return nil

}

func NewPrefetchReader(object *gcs.MinObject, bucket gcs.Bucket, PrefetchConfig *PrefetchConfig, blockPool *BlockPool, threadPool *ThreadPool) *PrefetchReader {
	prefetchReader := &PrefetchReader{
		object:              object,
		bucket:              bucket,
		PrefetchConfig:      PrefetchConfig,
		lastReadOffset:      -1,
		nextBlockToPrefetch: 0,
		randomSeekCount:     0,
		blockQueue:          NewBlockQueue(),
		blockPool:           blockPool,
		threadPool:          threadPool,
		metricHandle:        nil,
	}

	prefetchReader.ctx, prefetchReader.cancelFunc = context.WithCancel(context.Background())
	return prefetchReader
}

/**
 * ReadAt reads the data from the object at the given offset.
 * It first checks if the block containing the offset is already in memory.
 * If it is, it reads the data from the block.
 * If it is not, it downloads the block from GCS and then reads the data from the block.
 */
func (p *PrefetchReader) ReadAt(ctx context.Context, inputBuffer []byte, offset int64) (n int64, err error) {
	// Generate an unique id in the request
	requestId := uuid.New()
	stime := time.Now()
	blockIndex := offset / p.PrefetchConfig.PrefetchChunkSize

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
			cur := p.blockQueue.Peek()
			if cur.offset > uint64(offset) || cur.endOffset <= uint64(offset) {
				cur = p.blockQueue.Pop()
				cur.Cancel()
				<-cur.status
				cur.ReUse()
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

		block := p.blockQueue.Peek()
		t, ok := <-block.status
		if ok {
			block.Unblock() // close the status signal

			switch t {
			case BlockStatusDownloaded:
				break
			case BlockStatusDownloadFailed:
				return 0, fmt.Errorf("Block download failed")
			case BlockStatusDownloadCancelled:
				return 0, fmt.Errorf("Block download unexpected cancelled")
			default:
			}
		}

		readOffset := uint64(offset) - block.offset
		blockSize := GetBlockSize(block, uint64(p.PrefetchConfig.PrefetchChunkSize), uint64(p.object.Size))
		bytesRead := copy(inputBuffer[dataRead:], block.data[readOffset:blockSize])
		dataRead += bytesRead
		offset += int64(bytesRead)

		if offset >= int64(p.object.Size) {
			n = int64(dataRead)
			err = io.EOF
			return
		}

		if bytesRead == 0 {
			n = int64(dataRead)
			return n, io.EOF
		}

		if dataRead < len(inputBuffer) || (dataRead == len(inputBuffer) && offset == int64(block.endOffset)) {
			if !prefetchHappened {
				tmp := p.blockQueue.Pop()
				tmp.Cancel()
				<-tmp.status
				tmp.ReUse()
				p.blockPool.Release(tmp)

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

func (p *PrefetchReader) prefetch() error {
	nextPrefetchCount := p.PrefetchConfig.PrefetchMultiplier * p.lastPrefetchCount
	nextPrefetchCount = min(nextPrefetchCount, p.PrefetchConfig.PrefetchCount-int64(p.blockQueue.Len()))

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
func (p *PrefetchReader) freshStart(currentSeek int64) error {
	p.resetPrefetcher()
	blockIndex := currentSeek / p.PrefetchConfig.PrefetchChunkSize
	p.nextBlockToPrefetch = blockIndex

	err := p.scheduleNextBlock(false)
	if err != nil {
		return err
	}

	for i := 0; i < int(p.PrefetchConfig.InitialPrefetchBlockCnt) && p.nextBlockToPrefetch < p.maxBlockCount(); i++ {
		err := p.scheduleNextBlock(true)
		if err != nil {
			return err
		}
	}

	p.lastPrefetchCount = 1
	return nil
}

func (p *PrefetchReader) scheduleNextBlock(prefetch bool) error {
	block := p.blockPool.MustGet()
	if block == nil {
		return fmt.Errorf("unable to allocate block")
	}

	p.scheduleBlockWithIndex(block, p.nextBlockToPrefetch, prefetch)
	p.nextBlockToPrefetch++

	return nil
}

// scheduleBlock create a prefetch task for the given block and schedule it to thread-pool.
func (p *PrefetchReader) scheduleBlockWithIndex(block *Block, blockIndex int64, prefetch bool) {
	blockCtx, cancel := context.WithCancel(p.ctx)

	p.prepareBlock(blockIndex, block)
	task := &PrefetchTask{
		ctx:      blockCtx,
		object:   p.object,
		bucket:   p.bucket,
		block:    block,
		prefetch: prefetch,
	}

	logger.Tracef("Scheduling block (%s, %d, %v).", p.object.Name, block.id, prefetch)

	// Queue has always a right cancel function.
	block.cancelFunc = cancel

	p.blockQueue.Push(block)
	p.threadPool.Schedule(!prefetch, task)
}

// prepareBlock initializes block-state according to blockIndex.
func (p *PrefetchReader) prepareBlock(blockIndex int64, block *Block) {
	block.id = blockIndex
	block.offset = uint64(blockIndex * p.PrefetchConfig.PrefetchChunkSize)
	block.writeSeek = 0
	block.endOffset = min(block.offset+p.blockPool.blockSize, uint64(p.object.Size))
}

// Destroy cancels/discards the blocks in the blockQueue.
func (p *PrefetchReader) Destroy() {
	if p.cancelFunc != nil {
		p.cancelFunc()
		p.cancelFunc = nil
	}

	// Cancel all the processed or in process queue in the queue.
	for !p.blockQueue.IsEmpty() {
		block := p.blockQueue.Pop()
		block.Cancel()

		// Wait for download to complete and then free up this block
		<-block.status
		block.ReUse()
		p.blockPool.Release(block)
	}

	p.blockPool = nil
}

func (p *PrefetchReader) resetPrefetcher() {
	// Cancel all the processed or in process queue in the queue.
	for !p.blockQueue.IsEmpty() {
		block := p.blockQueue.Pop()
		block.Cancel()

		// Wait for download to complete and then free up this block
		<-block.status
		block.ReUse()
		p.blockPool.Release(block)
	}
}

func (p *PrefetchReader) maxBlockCount() int64 {
	return (int64(p.object.Size) + p.PrefetchConfig.PrefetchChunkSize - 1) / p.PrefetchConfig.PrefetchChunkSize
}
