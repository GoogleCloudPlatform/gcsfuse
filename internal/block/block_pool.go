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

package block

import (
	"fmt"

	"golang.org/x/sync/semaphore"
)

// BlockPool handles the creation of blocks as per the user configuration.
type BlockPool struct {
	// Channel holding free blocks.
	freeBlocksCh chan Block

	// Size of each block this pool holds.
	blockSize int64

	// Max number of blocks this blockPool can create.
	maxBlocks int64

	// Total number of blocks created so far.
	totalBlocks int64

	// Semaphore used to limit the total number of blocks created across
	// different files.
	globalMaxBlocksSem *semaphore.Weighted
}

// NewBlockPool creates the blockPool based on the user configuration.
func NewBlockPool(blockSize int64, maxBlocks int64, globalMaxBlocksSem *semaphore.Weighted) (bp *BlockPool, err error) {
	if blockSize <= 0 || maxBlocks <= 0 {
		err = fmt.Errorf("invalid configuration provided for blockPool, blocksize: %d, maxBlocks: %d", blockSize, maxBlocks)
		return
	}

	bp = &BlockPool{
		freeBlocksCh:       make(chan Block, maxBlocks),
		blockSize:          blockSize,
		maxBlocks:          maxBlocks,
		totalBlocks:        0,
		globalMaxBlocksSem: globalMaxBlocksSem,
	}
	return
}

// Get returns a block. It returns an existing block if it's ready for reuse or
// creates a new one if required.
func (bp *BlockPool) Get() (Block, error) {
	for {
		select {
		case b := <-bp.freeBlocksCh:
			// Reset the block for reuse.
			b.Reuse()
			return b, nil

		default:
			// No lock is required here since blockPool is per file and all write
			// calls to a single file are serialized because of inode.lock().
			if bp.canAllocateBlock() {
				b, err := createBlock(bp.blockSize)
				if err != nil {
					return nil, err
				}

				bp.totalBlocks++
				return b, nil
			}
		}
	}
}

// canAllocateBlock checks if a new block can be allocated.
func (bp *BlockPool) canAllocateBlock() bool {
	// If max blocks limit is reached, then no more blocks can be allocated.
	if bp.totalBlocks >= bp.maxBlocks {
		return false
	}

	// Always allow allocation if this is the first block for the file.
	if bp.totalBlocks == 0 {
		return true
	}

	// Otherwise, check if we can acquire a semaphore.
	semAcquired := bp.globalMaxBlocksSem.TryAcquire(1)
	return semAcquired
}

// FreeBlocksChannel returns the freeBlocksCh being used by the block pool.
func (bp *BlockPool) FreeBlocksChannel() chan Block {
	return bp.freeBlocksCh
}

// BlockSize returns the block size used by the blockPool.
func (bp *BlockPool) BlockSize() int64 {
	return bp.blockSize
}

func (bp *BlockPool) ClearFreeBlockChannel() error {
	for {
		select {
		case b := <-bp.freeBlocksCh:
			err := b.Deallocate()
			if err != nil {
				// if we get here, there is likely memory corruption.
				return fmt.Errorf("munmap error: %v", err)
			}
			bp.totalBlocks--
			if bp.totalBlocks != 0 {
				bp.globalMaxBlocksSem.Release(1)
			}
		default:
			// Return if there are no more blocks on the channel.
			return nil
		}
	}
}
