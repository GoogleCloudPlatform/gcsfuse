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
	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"golang.org/x/net/context"
)

type prefetchConfig struct {
	prefetchCount      int64
	prefetchChunkSize  int64
	prefetchMultiplier int64

	// Fast forward without reading.
	fastForwardBytes int64
}

func getDefaultPrefetchConfig() *prefetchConfig {
	return &prefetchConfig{
		prefetchCount:      6,
		prefetchChunkSize:  8 * int64(_1MB),
		prefetchMultiplier: 2,
	}
}

type prefetchReader struct {
	object              *gcs.MinObject
	bucket              gcs.Bucket
	prefetchConfig      *prefetchConfig
	lastReadOffset      int64
	nextBlockToPrefetch int64

	randomSeekCount int64

	cookedBlocks  *list.List
	cookingBlocks *list.List

	blockIndexMap map[int64]*Block

	readHandle []byte // For zonal bucket.

	blockPool  *BlockPool
	threadPool *ThreadPool

	metricHandle common.MetricHandle
}

func NewPrefetchReader(object *gcs.MinObject, bucket gcs.Bucket, prefetchConfig *prefetchConfig, blockPool *BlockPool, threadPool *ThreadPool) *prefetchReader {
	return &prefetchReader{
		object:              object,
		bucket:              bucket,
		prefetchConfig:      prefetchConfig,
		lastReadOffset:      -1,
		nextBlockToPrefetch: 0,
		randomSeekCount:     0,
		cookedBlocks:        list.New(),
		cookingBlocks:       list.New(),
		blockIndexMap:       make(map[int64]*Block),
		blockPool:           blockPool,
		threadPool:          threadPool,
		metricHandle:        nil,
	}
}

/**
 * ReadAt reads the data from the object at the given offset.
 * It first checks if the block containing the offset is already in memory.
 * If it is, it reads the data from the block.
 * If it is not, it downloads the block from GCS and then reads the data from the block.
 */
func (p *prefetchReader) ReadAt(ctx context.Context, inputBuffer []byte, offset int64) (objectData gcsx.ObjectData, err error) {
	// Generate an unique id in the request
	requestId := uuid.New()
	stime := time.Now()

	logger.Infof("%.10v <- ReadAt(%d, %d).", requestId, offset, len(inputBuffer))

	defer func() {
		if err != nil && err != io.EOF {
			logger.Errorf("%.10v -> ReadAt(%d, %d) with error: %v", requestId, offset, len(inputBuffer), err)
		} else {
			logger.Infof("%.10v -> ReadAt(%d, %d): ok(%v)", requestId, offset, len(inputBuffer), time.Since(stime))
		}
	}()

	objectData = gcsx.ObjectData{
		DataBuf:  inputBuffer,
		CacheHit: false,
		Size:     0,
	}

	if offset >= int64(p.object.Size) {
		err = io.EOF
		return
	}

	dataRead := int(0)
	for dataRead < len(inputBuffer) {
		block, err := p.findBlock(ctx, offset)
		if err != nil {
			return objectData, err
		}

		readOffset := uint64(offset) - block.offset
		blockSize := GetBlockSize(block, uint64(p.prefetchConfig.prefetchChunkSize), uint64(p.object.Size))

		bytesRead := copy(inputBuffer[dataRead:], block.data[readOffset:blockSize])

		dataRead += bytesRead
		offset += int64(bytesRead)

		if offset >= int64(p.object.Size) {
			objectData.Size = dataRead
			return objectData, io.EOF
		}
	}

	objectData.Size = dataRead
	return
}

/**
 * findBlock finds the block containing the given offset.
 * It first checks if the block is already in the cache.
 * If it is, it returns the block.
 * If it is not, it downloads the block from GCS and then returns.
 */
func (p *prefetchReader) findBlock(ctx context.Context, offset int64) (*Block, error) {
	blockIndex := offset / p.prefetchConfig.prefetchChunkSize
	logger.Tracef("findBlock: <- (%s, %d)", p.object.Name, offset)

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

			if p.randomSeekCount < 2 {
				if uint64(p.nextBlockToPrefetch*p.prefetchConfig.prefetchChunkSize) < p.object.Size {
					err := p.fetch(ctx, p.nextBlockToPrefetch, true)
					if err != nil && err != io.EOF {
						return nil, err
					}

				}
			}
		case BlockStatusDownloadFailed:
			logger.Errorf("findBlock: failed to download block (%s, %v, %d).", p.object.Name, block.id, offset)
			block.flags.Set(BlockFlagFailed)
		default:
			logger.Errorf("findBlock: unknown status %d for block (%s, %v, %d)", t, p.object.Name, block.id, offset)
		}
	}
	return block, nil
}

func (p *prefetchReader) fetch(ctx context.Context, blockIndex int64, prefetch bool) (err error) {
	logger.Tracef("fetch: <- block (%s, %d, %v)", p.object.Name, blockIndex, prefetch)

	currentBlockCnt := p.cookingBlocks.Len() + p.cookedBlocks.Len()
	newlyCreatedBlockCnt := int(0)

	for ; currentBlockCnt < int(p.prefetchConfig.prefetchCount) && newlyCreatedBlockCnt < 4; currentBlockCnt++ {
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
		}
	}

	return
}

// refreshBlock refreshes a block by scheduling the block to thread-pool to download.
func (p *prefetchReader) refreshBlock(ctx context.Context, blockIndex int64, prefetch bool) error {
	offset := blockIndex * p.prefetchConfig.prefetchChunkSize
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
func (p *prefetchReader) scheduleBlock(ctx context.Context, block *Block, prefetch bool) {
	task := &prefetchTask{
		ctx:      ctx,
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

func (p *prefetchReader) addToCookedBlocks(block *Block) {
	if block.node != nil {
		p.cookedBlocks.Remove(block.node)
		p.cookingBlocks.Remove(block.node)
	}

	block.node = p.cookedBlocks.PushBack(block)
}

func (p *prefetchReader) addToCookingBlocks(block *Block) {
	if block.node != nil {
		_ = p.cookedBlocks.Remove(block.node)
		_ = p.cookingBlocks.Remove(block.node)
	}

	block.node = p.cookingBlocks.PushBack(block)
}

func (p *prefetchReader) Destroy() {
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
