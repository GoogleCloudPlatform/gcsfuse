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
 	"io"
	"fmt"
	
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
		prefetchChunkSize:  10 * int64(_1MB),
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
		threadPool:           threadPool,
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
	logger.Infof("ReadAt: input parameters: buffer length: %d, offset: %d", len(inputBuffer), offset)

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
		logger.Infof("ReadAt: reading data from block at offset %d", offset)

		block, err := p.findBlock(ctx, offset)
		if err != nil {
			logger.Errorf("ReadAt: error in finding block: %v", err)
			return objectData, err
		}

		readOffset := uint64(offset) - block.offset
		blockSize := GetBlockSize(block, uint64(p.prefetchConfig.prefetchChunkSize), uint64(p.object.Size))

		bytesRead := copy(inputBuffer[dataRead:], block.data[readOffset:blockSize])
		
		dataRead += bytesRead
		offset += int64(bytesRead)

		logger.Infof("ReadAt: read %d bytes from block %d, offset: %d, block offset: %d", bytesRead, block.id, offset, block.offset)

		
		if offset >= int64(p.object.Size) {
			logger.Info("Early return: %v, error: %v", dataRead, io.EOF)
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
	logger.Infof("findBlock: looking for block %v for object %s at offset %d", blockIndex, p.object.Name, offset)

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
			logger.Tracef("findBlock: downloaded block %v for object %s at offset %d", block.id, p.object.Name, offset)
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
			logger.Errorf("findBlock: failed to download block %v for object %s at offset %d", block.id, p.object.Name, offset)
			block.flags.Set(BlockFlagFailed)
		default:
			logger.Errorf("findBlock: unknown status %d for block %v for object %s at offset %d", t, block.id, p.object.Name, offset)
		}
	}
	return block, nil
}

func (p *prefetchReader) fetch(ctx context.Context, blockIndex int64, prefetch bool) (err error) {
	logger.Infof("fetch: start fetching block %v for object %s, prefetch: %v", blockIndex, p.object.Name, prefetch)


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
		logger.Warnf("Hit sliding window code")
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
	logger.Tracef("refreshBlock: request to download block: %d, object: %s, prefetch: %v", blockIndex, p.object.Name, prefetch)

	offset := blockIndex * p.prefetchConfig.prefetchChunkSize
	if uint64(offset) >= p.object.Size {
		return io.EOF
	}

	nodeList := p.cookedBlocks
	if nodeList.Len() == 0 && !prefetch {
		block := p.blockPool.MustGet()
		if block == nil {
			logger.Tracef("refreshBlock: Unable to allocate block: %d, object: %s, prefetch: %v", blockIndex, p.object.Name, prefetch)
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
* download downloads a range of bytes from the object.
* This method is used by the thread-pool scheduler.
 */
func (p *prefetchReader) download(task *prefetchTask) {
	logger.Infof("download: start downloading block %v for object %s", task.blockId, task.object.Name)

	start := uint64(task.block.offset)
	end := task.block.offset + GetBlockSize(task.block, uint64(len(task.block.data)), task.object.Size)

	newReader, err := task.bucket.NewReaderWithReadHandle(
		task.ctx,
		&gcs.ReadObjectRequest{
			Name:       task.object.Name,
			Generation: task.object.Generation,
			Range: &gcs.ByteRange{
				Start: start,
				Limit: end,
			},
			ReadCompressed: task.object.HasContentEncodingGzip(),
			ReadHandle:     nil,
		})
	if err != nil {
		logger.Errorf("downloadRange: error in creating NewReader with start %d and limit %d: %v", start, end, err)
		task.block.Failed()
		task.block.Ready(BlockStatusDownloadFailed)
		return
	}

	_, err = io.CopyN(task.block, newReader, int64(end-start))
	if err != nil {
		logger.Errorf("downloadRange: error at the time of copying content to cache file %v", err)
		task.block.Failed()
		task.block.Ready(BlockStatusDownloadFailed)
		return
	}

	task.block.Ready(BlockStatusDownloaded)

	logger.Infof("download: completed downloading block %v for object %s", task.blockId, task.object.Name)
}

func GetBlockSize(block *Block, blockSize uint64, objectSize uint64) uint64 {
	return min(blockSize, objectSize-block.offset)
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
