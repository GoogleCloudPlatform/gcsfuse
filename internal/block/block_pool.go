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
	"errors"
	"fmt"

	"golang.org/x/sync/semaphore"
)

var CantAllocateAnyBlockError error = errors.New("cant allocate any streaming write block as global max blocks limit is reached")

type GenBlock interface {
	// Reuse resets the block for reuse.
	Reuse()

	// Deallocate releases the resources held by the block.
	Deallocate() error
}

// GenBlockPool handles the creation of blocks as per the user configuration.
type GenBlockPool[T GenBlock] struct {
	// Channel holding free blocks.
	freeBlocksCh chan T

	// Size of each block this pool holds.
	blockSize int64

	// Max number of blocks this blockPool can create.
	maxBlocks int64

	// Total number of blocks created so far.
	totalBlocks int64

	// Semaphore used to limit the total number of blocks created across
	// different files.
	globalMaxBlocksSem *semaphore.Weighted

	// createBlockFunc is a function that creates a new block of type T
	createBlockFunc func(blockSize int64) (T, error)
}

// NewGenBlockPool creates the blockPool based on the user configuration.
func NewGenBlockPool[T GenBlock](blockSize int64, maxBlocks int64, globalMaxBlocksSem *semaphore.Weighted, createBlockFunc func(blockSize int64) (T, error)) (bp *GenBlockPool[T], err error) {
	if blockSize <= 0 || maxBlocks <= 0 {
		err = fmt.Errorf("invalid configuration provided for blockPool, blocksize: %d, maxBlocks: %d", blockSize, maxBlocks)
		return
	}

	bp = &GenBlockPool[T]{
		freeBlocksCh:       make(chan T, maxBlocks),
		blockSize:          blockSize,
		maxBlocks:          maxBlocks,
		totalBlocks:        0,
		globalMaxBlocksSem: globalMaxBlocksSem,
		createBlockFunc:    createBlockFunc,
	}
	semAcquired := bp.globalMaxBlocksSem.TryAcquire(1)
	if !semAcquired {
		return nil, CantAllocateAnyBlockError
	}

	return bp, nil
}

// Get returns a block. It returns an existing block if it's ready for reuse or
// creates a new one if required.
// Not thread-safe, calling from multiple goroutines may lead memory leaks because
// of race conditions.
func (bp *GenBlockPool[T]) Get() (T, error) {
	for {
		select {
		case b := <-bp.freeBlocksCh:
			// Reset the block for reuse.
			b.Reuse()
			return b, nil

		default:
			if bp.canAllocateBlock() {
				b, err := bp.createBlockFunc(bp.blockSize)
				if err != nil {
					var zero T
					return zero, err
				}

				bp.totalBlocks++
				return b, nil
			}
		}
	}
}

// TryGet returns a block if available, or an error if no blocks can be allocated.
// It returns an existing block if it's ready for reuse or creates a new one if required.
// Not thread-safe, calling from multiple goroutines may lead to memory leaks because
// of race conditions.
func (bp *GenBlockPool[T]) TryGet() (T, error) {
	select {
	case b := <-bp.freeBlocksCh:
		// Reset the block for reuse.
		b.Reuse()
		return b, nil

	default:
		if bp.canAllocateBlock() {
			b, err := bp.createBlockFunc(bp.blockSize)
			if err != nil {
				var zero T
				return zero, err
			}

			bp.totalBlocks++
			return b, nil
		}
		var zero T
		return zero, CantAllocateAnyBlockError
	}
}

// canAllocateBlock checks if a new block can be allocated.
func (bp *GenBlockPool[T]) canAllocateBlock() bool {
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

// Release puts the block back into the free blocks channel for reuse.
func (bp *GenBlockPool[T]) Release(b T) {
	select {
	case bp.freeBlocksCh <- b:
	default:
		panic("Block pool's free blocks channel is full, this should never happen")
	}
}

// BlockSize returns the block size used by the blockPool.
func (bp *GenBlockPool[T]) BlockSize() int64 {
	return bp.blockSize
}

func (bp *GenBlockPool[T]) ClearFreeBlockChannel(releaseLastBlock bool) error {
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
			// We are here, it means there are no more blocks in the free blocks channel.
			// Release semaphore for last block iff releaseLastBlock is true.
			if releaseLastBlock {
				bp.globalMaxBlocksSem.Release(1)
			}
			return nil
		}
	}
}

// TotalFreeBlocks returns the total number of free blocks available in the pool.
// This is useful for testing and debugging purposes.
func (bp *GenBlockPool[T]) TotalFreeBlocks() int {
	return len(bp.freeBlocksCh)
}

// NewBlockPool creates GenBlockPool for block.Block interface.
func NewBlockPool(blockSize int64, maxBlocks int64, globalMaxBlocksSem *semaphore.Weighted) (bp *GenBlockPool[Block], err error) {
	return NewGenBlockPool(blockSize, maxBlocks, globalMaxBlocksSem, createBlock)
}

// NewPrefetchBlockPool creates GenBlockPool for block.PrefetchBlock interface.
func NewPrefetchBlockPool(blockSize int64, maxBlocks int64, globalMaxBlocksSem *semaphore.Weighted) (bp *GenBlockPool[PrefetchBlock], err error) {
	return NewGenBlockPool(blockSize, maxBlocks, globalMaxBlocksSem, createPrefetchBlock)
}
