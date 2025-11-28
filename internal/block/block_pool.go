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

var ErrCantAllocateAnyBlockError error = errors.New("cant allocate any block as global max blocks limit is reached")

type GenBlock interface {
	// Reuse resets the block for reuse.
	Reuse()

	// Deallocate releases the resources held by the block.
	Deallocate() error
}

// GenBlockPool is a generic block pool for managing blocks that implement the GenBlock interface.
// It offers methods to get blocks, return blocks to the free pool, and clear the free pool.
// This implementation is NOT thread-safe - concurrent access from multiple goroutines requires external synchronization.
//
// Block allocation is controlled by maxBlocks (per-pool limit) and a global semaphore (cross-pool limit).
// When the global limit is reached, Get() will block until blocks become available, while TryGet()
// returns an error immediately to avoid blocking.
//
// The pool supports reserving blocks at creation time - these reserved blocks hold semaphore permits
// and can only be released when clearing the pool with releaseReservedBlocks=true.
type GenBlockPool[T GenBlock] struct {
	// Channel holding free blocks.
	freeBlocksCh chan T

	// Size of each block this pool holds.
	blockSize int64

	// Max number of blocks this blockPool can create.
	maxBlocks int64

	// Total number of blocks created so far.
	totalBlocks int64

	// Number of blocks reserved at the time of block pool creation.
	reservedBlocks int64

	// Semaphore used to limit the total number of blocks created across
	// different files.
	globalMaxBlocksSem *semaphore.Weighted

	// createBlockFunc is a function that creates a new block of type T
	createBlockFunc func(blockSize int64) (T, error)
}

// NewGenBlockPool creates the blockPool based on the user configuration.
func NewGenBlockPool[T GenBlock](blockSize int64, maxBlocks int64, reservedBlocks int64, globalMaxBlocksSem *semaphore.Weighted, createBlockFunc func(blockSize int64) (T, error)) (bp *GenBlockPool[T], err error) {
	if blockSize <= 0 || maxBlocks <= 0 {
		err = fmt.Errorf("invalid configuration provided for blockPool, blocksize: %d, maxBlocks: %d", blockSize, maxBlocks)
		return
	}

	if reservedBlocks < 0 || reservedBlocks > maxBlocks {
		err = fmt.Errorf("invalid reserved blocks count: %d, it should be between 0 and maxBlocks: %d", reservedBlocks, maxBlocks)
		return
	}

	semAcquired := globalMaxBlocksSem.TryAcquire(reservedBlocks)
	if !semAcquired {
		return nil, ErrCantAllocateAnyBlockError
	}

	return &GenBlockPool[T]{
		freeBlocksCh:       make(chan T, maxBlocks),
		blockSize:          blockSize,
		maxBlocks:          maxBlocks,
		reservedBlocks:     reservedBlocks,
		totalBlocks:        0,
		globalMaxBlocksSem: globalMaxBlocksSem,
		createBlockFunc:    createBlockFunc,
	}, nil
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
		return zero, ErrCantAllocateAnyBlockError
	}
}

// canAllocateBlock checks if a new block can be allocated.
func (bp *GenBlockPool[T]) canAllocateBlock() bool {
	// If max blocks limit is reached, then no more blocks can be allocated.
	if bp.totalBlocks >= bp.maxBlocks {
		return false
	}

	// Always allow allocation upto reserved number of blocks.
	if bp.totalBlocks < bp.reservedBlocks {
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

func (bp *GenBlockPool[T]) ClearFreeBlockChannel(releaseReservedBlocks bool) error {
	for {
		select {
		case b := <-bp.freeBlocksCh:
			err := b.Deallocate()
			if err != nil {
				// if we get here, there is likely memory corruption.
				return fmt.Errorf("munmap error: %v", err)
			}
			// Release semaphore for all but the reserved blocks.
			if bp.totalBlocks > bp.reservedBlocks {
				bp.globalMaxBlocksSem.Release(1)
			}
			bp.totalBlocks--
		default:
			// We are here, it means there are no more blocks in the free blocks channel.
			// Release semaphore for the released blocks iff releaseReservedBlocks is true.
			if releaseReservedBlocks {
				bp.globalMaxBlocksSem.Release(bp.reservedBlocks)
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
func NewBlockPool(blockSize int64, maxBlocks int64, reservedBlocks int64, globalMaxBlocksSem *semaphore.Weighted) (bp *GenBlockPool[Block], err error) {
	return NewGenBlockPool(blockSize, maxBlocks, reservedBlocks, globalMaxBlocksSem, createBlock)
}

// NewPrefetchBlockPool creates GenBlockPool for block.PrefetchBlock interface.
func NewPrefetchBlockPool(blockSize int64, maxBlocks int64, reservedBlocks int64, globalMaxBlocksSem *semaphore.Weighted) (bp *GenBlockPool[PrefetchBlock], err error) {
	return NewGenBlockPool(blockSize, maxBlocks, reservedBlocks, globalMaxBlocksSem, createPrefetchBlock)
}
