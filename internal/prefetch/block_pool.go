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
	"sync"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
)

const _1MB uint64 = (1024 * 1024)

// BlockPool is a pool of Blocks
type BlockPool struct {
	// Channel holding free blocks
	blocksCh chan *Block

	// block having null data
	zeroBlock *Block

	// channel to reset the data in a block
	resetBlockCh chan *Block

	// Wait group to wait for resetBlock() thread to finish
	wg sync.WaitGroup

	// Size of each block this pool holds
	blockSize uint64

	// Number of block that this pool can handle at max
	maxBlocks uint32
}

// NewBlockPool allocates a new pool of blocks
func NewBlockPool(blockSize uint64, memSize uint64) *BlockPool {
	// Ignore if config is invalid
	if blockSize == 0 || memSize < blockSize {
		logger.Errorf("blockpool::NewBlockPool : blockSize : %v, memsize: %v", blockSize, memSize)
		return nil
	}

	// Calculate how many blocks can be allocated
	blockCount := uint32(memSize / blockSize)

	pool := &BlockPool{
		blocksCh:     make(chan *Block, blockCount-1),
		resetBlockCh: make(chan *Block, blockCount-1),
		maxBlocks:    uint32(blockCount),
		blockSize:    blockSize,
	}

	// Preallocate all blocks so that during runtime we do not spend CPU cycles on this
	for i := (uint32)(0); i < blockCount; i++ {
		b, err := AllocateBlock(blockSize)
		if err != nil {
			logger.Errorf("BlockPool::NewBlockPool : Failed to allocate block [%v]", err.Error())
			return nil
		}

		if i == blockCount-1 {
			pool.zeroBlock = b
		} else {
			pool.blocksCh <- b
		}
	}

	// run a thread to reset the data in a block
	pool.wg.Add(1)
	go pool.resetBlock()

	return pool
}

// Terminate ends the block pool life
func (pool *BlockPool) Terminate() {
	// TODO: call terminate after all the threads have completed
	close(pool.resetBlockCh)
	pool.wg.Wait()

	close(pool.blocksCh)

	_ = pool.zeroBlock.Delete()

	// Release back the memory allocated to each block
	for {
		b := <-pool.blocksCh
		if b == nil {
			break
		}
		_ = b.Delete()
	}
}

// Usage provides % usage of this block pool
func (pool *BlockPool) Usage() uint32 {
	return ((pool.maxBlocks - (uint32)(len(pool.blocksCh)+len(pool.resetBlockCh))) * 100) / pool.maxBlocks
}

// MustGet a Block from the pool, wait until something is free
func (pool *BlockPool) MustGet() *Block {
	var b *Block = nil

	select {
	case b = <-pool.blocksCh:
		break

	default:
		// There are no free blocks so we must allocate one and return here
		// As the consumer of the pool needs a block immediately
		logger.Info("BlockPool::MustGet : No free blocks, allocating a new one")
		var err error
		b, err = AllocateBlock(pool.blockSize)
		if err != nil {
			return nil
		}
	}

	// Mark the buffer ready for reuse now
	b.ReUse()
	return b
}

// TryGet a Block from the pool, return back if nothing is available
func (pool *BlockPool) TryGet() *Block {
	var b *Block = nil

	select {
	case b = <-pool.blocksCh:
		break

	default:
		return nil
	}

	// Mark the buffer ready for reuse now
	b.ReUse()
	return b
}

// Release back the Block to the pool
func (pool *BlockPool) Release(b *Block) {
	select {
	case pool.resetBlockCh <- b:
		break
	default:
		_ = b.Delete()
	}
}

// reset the data in a block before its next use
func (pool *BlockPool) resetBlock() {
	defer pool.wg.Done()

	for b := range pool.resetBlockCh {
		// reset the data with null entries
		copy(b.data, pool.zeroBlock.data)

		select {
		case pool.blocksCh <- b:
			continue
		default:
			_ = b.Delete()
		}
	}
}