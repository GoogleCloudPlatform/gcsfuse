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

// var CantAllocateAnyBlockError error = errors.New("cant allocate any streaming write block as global max blocks limit is reached")

// PrefetchBlockPool handles the creation of blocks as per the user configuration.
type PrefetchBlockPool struct {
	// Channel holding free blocks.
	freeBlocksCh chan PrefetchBlock

	// Size of each block this pool holds.
	blockSize int64

	// Max number of blocks this PrefetchBlockPool can create.
	maxBlocks int64

	// Total number of blocks created so far.
	totalBlocks int64

	// Semaphore used to limit the total number of blocks created across
	// different files.
	globalMaxBlocksSem *semaphore.Weighted
}

// NewPrefetchBlockPool creates the PrefetchBlockPool based on the user configuration.
func NewPrefetchBlockPool(blockSize int64, maxBlocks int64, globalMaxBlocksSem *semaphore.Weighted) (bp *PrefetchBlockPool, err error) {
	if blockSize <= 0 || maxBlocks <= 0 {
		err = fmt.Errorf("invalid configuration provided for PrefetchBlockPool, blocksize: %d, maxBlocks: %d", blockSize, maxBlocks)
		return
	}

	bp = &PrefetchBlockPool{
		freeBlocksCh:       make(chan PrefetchBlock, maxBlocks),
		blockSize:          blockSize,
		maxBlocks:          maxBlocks,
		totalBlocks:        0,
		globalMaxBlocksSem: globalMaxBlocksSem,
	}
	semAcquired := bp.globalMaxBlocksSem.TryAcquire(1)
	if !semAcquired {
		return nil, CantAllocateAnyBlockError
	}

	return bp, nil
}

// Get returns a block. It returns an existing block if it's ready for reuse or
// creates a new one if required.
func (bp *PrefetchBlockPool) Get() (PrefetchBlock, error) {
	for {
		select {
		case b := <-bp.freeBlocksCh:
			// Reset the block for reuse.
			b.Reuse()
			return b, nil

		default:
			// No lock is required here since PrefetchBlockPool is per file and all write
			// calls to a single file are serialized because of inode.lock().
			if bp.canAllocateBlock() {
				b, err := CreatePrefetchBlock(bp.blockSize)
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
func (bp *PrefetchBlockPool) canAllocateBlock() bool {
	// If max blocks limit is reached, then no more blocks can be allocated.
	if bp.totalBlocks >= bp.maxBlocks {
		return false
	}

	// Always allow allocation if this is the first block for the file since it has been reserved at
	// the time of block pool creation.
	if bp.totalBlocks == 0 {
		return true
	}

	// Otherwise, check if we can acquire a semaphore.
	semAcquired := bp.globalMaxBlocksSem.TryAcquire(1)
	return semAcquired
}

// FreeBlocksChannel returns the freeBlocksCh being used by the block pool.
func (bp *PrefetchBlockPool) FreeBlocksChannel() chan PrefetchBlock {
	return bp.freeBlocksCh
}

// BlockSize returns the block size used by the PrefetchBlockPool.
func (bp *PrefetchBlockPool) BlockSize() int64 {
	return bp.blockSize
}

func (bp *PrefetchBlockPool) Release(b PrefetchBlock) {
	// If the block is not deallocated, then put it back on the channel.
	select {
	case bp.freeBlocksCh <- b:
	default:
		// If the channel is full, then deallocate the block.
		err := b.Deallocate()
		if err != nil {
			// if we get here, there is likely memory corruption.
		}
		bp.totalBlocks--
		bp.globalMaxBlocksSem.Release(1)
	}
}

func (bp *PrefetchBlockPool) ClearFreeBlockChannel(releaseLastBlock bool) error {
	for {
		select {
		case b := <-bp.freeBlocksCh:
			err := b.Deallocate()
			if err != nil {
				// if we get here, there is likely memory corruption.
				return fmt.Errorf("munmap error: %v", err)
			}
			bp.totalBlocks--
			// Release semaphore for last block iff releaseLastBlock is true.
			if bp.totalBlocks != 0 || releaseLastBlock {
				bp.globalMaxBlocksSem.Release(1)
			}
		default:
			// Return if there are no more blocks on the channel.
			return nil
		}
	}
}
