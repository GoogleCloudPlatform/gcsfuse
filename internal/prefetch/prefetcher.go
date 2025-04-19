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
	"container/list"
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
	PrefetchCount     int64
	PrefetchChunkSize int64
}

func getDefaultPrefetchConfig() *PrefetchConfig {
	return &PrefetchConfig{
		PrefetchCount:     6,
		PrefetchChunkSize: 8 * int64(_1MB),
	}
}

type PrefetchReader struct {
	object              *gcs.MinObject
	bucket              gcs.Bucket
	PrefetchConfig      *PrefetchConfig
	lastReadOffset      int64
	nextBlockToPrefetch int64

	randomSeekCount int64

	blockIndexMap map[int64]*Block
	blockQueue    *Queue

	readHandle []byte // For zonal bucket.

	blockPool  *BlockPool
	threadPool *ThreadPool

	metricHandle common.MetricHandle

	// Create a cancellable context and cancellable function 
	cancelFunc context.CancelFunc
	ctx context.Context
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
		blockQueue:          NewQueue(),
		blockIndexMap:       make(map[int64]*Block),
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

	logger.Infof("%.10v <- ReadAt(%d, %d, %d).", requestId, offset, len(inputBuffer), blockIndex)

	defer func() {
		if err != nil && err != io.EOF {
			logger.Errorf("%.10v -> ReadAt(%d, %d, %d) with error: %v", requestId, offset, len(inputBuffer), blockIndex, err)
		} else {
			logger.Infof("%.10v -> ReadAt(%d, %d, %d): ok(%v)", requestId, offset, len(inputBuffer), blockIndex, time.Since(stime))
		}
	}()

	if offset >= int64(p.object.Size) {
		err = io.EOF
		return
	}

	dataRead := int(0)
	for 
	for dataRead < len(inputBuffer) {
		var block *Block
		block, err = p.findBlock(ctx, offset)
		if err != nil {
			return n, err
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
	}
	n = int64(dataRead)
	return
}

/**
 * findBlock finds the block containing the given offset.
 * It first checks if the block is already in the cache.
 * If it is, it returns the block.
 * If it is not, it downloads the block from GCS and then returns.
 */
func (p *PrefetchReader) findBlock(ctx context.Context, offset int64) (*Block, error) {
	blockIndex := offset / p.PrefetchConfig.PrefetchChunkSize
	logger.Tracef("findBlock: <- (%s, %d)", p.object.Name, offset)

	if p.blockQueue.IsEmpty() {
		p.blockQueue.Push(blockIndex)
	} else {
			
	}
	entry, ok := p.blockIndexMap[blockIndex]
	if !ok {
		if p.lastReadOffset == -1 { // First read.
			err := p.fetch(ctx, blockIndex, false)
			if err != nil && err != io.EOF {
				return nil, err
			}
		} else { // random read.
			p.randomSeekCount++
			err := p.fetch(ctx, blockIndex, false)
			if err != nil && err != io.EOF {
				return nil, err
			}
		}

		entry, ok = p.blockIndexMap[blockIndex]
		if !ok {
			logger.Errorf("findBlock: not able to find block after scheduling")
			return nil, io.EOF
		}
	}

	block := entry

	// Wait for this block to complete the download.
	t, ok := <-block.state
	if ok {
		block.Unblock()

		switch t {
		case BlockStatusDownloaded:
			block.flags.Clear(BlockFlagDownloading)
			p.addToCookedBlocks(block)
			if p.randomSeekCount < 2 {
				if uint64(p.nextBlockToPrefetch*p.PrefetchConfig.PrefetchChunkSize) < p.object.Size {
					err := p.fetch(ctx, p.nextBlockToPrefetch, true)
					if err != nil && err != io.EOF {
						return nil, err
					}

				}
			}
		case BlockStatusDownloadFailed:
			block.flags.Set(BlockFlagFailed)
			return nil, fmt.Errorf("findBlock: failed to download block (%s, %v, %d).", p.object.Name, block.id, offset)

		case BlockStatusDownloadCancelled:
			block.flags.Set(BlockFlagFailed)
			return nil, fmt.Errorf("findBlock: download cancelled for block (%s, %v, %d).", p.object.Name, block.id, offset)
		default:
			logger.Errorf("findBlock: unknown status %d for block (%s, %v, %d)", t, p.object.Name, block.id, offset)
		}
	}
	return block, nil
}

func (p *PrefetchReader) fetch(ctx context.Context, blockIndex int64, prefetch bool) (err error) {
	logger.Tracef("fetch: <- block (%s, %d, %v)", p.object.Name, blockIndex, prefetch)

	currentBlockCnt := p.cookingBlocks.Len() + p.cookedBlocks.Len()
	newlyCreatedBlockCnt := int(0)

	for ; currentBlockCnt < int(p.PrefetchConfig.PrefetchCount) && newlyCreatedBlockCnt < min_prefetch; currentBlockCnt++ {
		block := p.blockPool.MustGet()
		if block != nil {
			block.node = p.cookedBlocks.PushFront(block)
			newlyCreatedBlockCnt++
		}
	}

	// If no buffers were allocated then we have all the buffers allocated to this prefetcher.
	// So, re-use the buffer in the sliding window faship.
	if newlyCreatedBlockCnt == 0 {
		logger.Infof("fetch (%s, %d) moved into sliding window mode.", p.object.Name, blockIndex)
		newlyCreatedBlockCnt = 1
	}

	for i := 0; i < newlyCreatedBlockCnt; i++ {
		_, ok := p.blockIndexMap[blockIndex]
		if !ok {
			err = p.refreshBlock(ctx, blockIndex, prefetch || i > 0)
			if err != nil {
				return
			}
			blockIndex++
		} else {
			logger.Warnf("Not expected")
		}
	}

	return
}

// refreshBlock refreshes a block by scheduling the block to thread-pool to download.
func (p *PrefetchReader) refreshBlock(ctx context.Context, blockIndex int64, prefetch bool) error {
	offset := blockIndex * p.PrefetchConfig.PrefetchChunkSize
	if uint64(offset) >= p.object.Size {
		return io.EOF
	}

	nodeList := p.cookedBlocks
	if nodeList.Len() == 0 && !prefetch {
		block := p.blockPool.MustGet()
		if block == nil {
			return fmt.Errorf("unable to allocate block")
		}

		block.node = p.cookedBlocks.PushFront(block)
	}

	node := nodeList.Front()
	if node != nil {
		block := node.Value.(*Block)

		if block.id != -1 {
			delete(p.blockIndexMap, block.id)
		}

		block.ReUse()
		block.id = int64(blockIndex)
		block.offset = uint64(offset)
		p.blockIndexMap[blockIndex] = block
		p.nextBlockToPrefetch = blockIndex + 1

		p.scheduleBlock(ctx, block, prefetch)
	}
	return nil
}

/**
 * scheduleBlock schedules a block for download
 */
func (p *PrefetchReader) scheduleBlock(ctx context.Context, block *Block, prefetch bool) {
	task := &PrefetchTask{
		ctx:      p.ctx,
		object:   p.object,
		bucket:   p.bucket,
		block:    block,
		prefetch: prefetch,
		blockId:  block.id,
	}

	logger.Infof("Scheduling block (%s, %d, %v).", p.object.Name, block.id, prefetch)

	// Add to cooking block.
	p.addToCookingBlocks(block)
	block.flags.Set(BlockFlagDownloading)

	p.threadPool.Schedule(!prefetch, task)
}

func (p *PrefetchReader) addToCookedBlocks(block *Block) {
	if block.node != nil {
		p.cookedBlocks.Remove(block.node)
		p.cookingBlocks.Remove(block.node)
	}

	block.node = p.cookedBlocks.PushBack(block)
}

func (p *PrefetchReader) addToCookingBlocks(block *Block) {
	if block.node != nil {
		_ = p.cookedBlocks.Remove(block.node)
		_ = p.cookingBlocks.Remove(block.node)
	}

	block.node = p.cookingBlocks.PushBack(block)
}

func (p *PrefetchReader) Destroy() {
	if p.cancelFunc != nil {
		p.cancelFunc()
		p.cancelFunc = nil
	}

	// Release the buffers which are still under download after they have been written
	blockList := p.cookingBlocks
	node := blockList.Front()
	for ; node != nil; node = blockList.Front() {
		block := blockList.Remove(node).(*Block)

		// Wait for download to complete and then free up this block
		<-block.state
		block.node = nil
		block.ReUse()
		p.blockPool.Release(block)
	}
	p.cookingBlocks = nil

	// Release the blocks that are ready to be reused
	blockList = p.cookedBlocks
	node = blockList.Front()
	for ; node != nil; node = blockList.Front() {
		block := blockList.Remove(node).(*Block)
		// block.Unblock()
		block.node = nil
		block.ReUse()
		p.blockPool.Release(block)
	}
	p.cookedBlocks = nil
}
