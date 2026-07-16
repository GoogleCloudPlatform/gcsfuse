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
	"context"
	"errors"
	"fmt"
	"sync"
)

var CantAllocateAnyBlockError error = errors.New("cant allocate any block as global max blocks limit is reached")

// BlockSemaphore is a channel-backed counting semaphore.
type BlockSemaphore struct {
	tokens chan struct{}
	mu     sync.Mutex
}

const MaxBlockSemaphoreCapacity int64 = 1000000

// NewBlockSemaphore creates a new BlockSemaphore with the given capacity.
func NewBlockSemaphore(capacity int64) *BlockSemaphore {
	if capacity < 0 {
		capacity = 0
	}
	if capacity > MaxBlockSemaphoreCapacity {
		capacity = MaxBlockSemaphoreCapacity
	}
	ch := make(chan struct{}, capacity)
	for i := int64(0); i < capacity; i++ {
		ch <- struct{}{}
	}
	return &BlockSemaphore{
		tokens: ch,
	}
}

// TokenChan returns the read-only channel of token permits.
func (s *BlockSemaphore) TokenChan() <-chan struct{} {
	return s.tokens
}

// TryAcquire attempts to acquire n permits without blocking.
func (s *BlockSemaphore) TryAcquire(n int64) bool {
	if n <= 0 {
		return true
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	var acquired int64
	for acquired < n {
		select {
		case <-s.tokens:
			acquired++
		default:
			s.releaseLocked(acquired)
			return false
		}
	}
	return true
}

// Acquire acquires n permits, blocking until available or ctx is done.
func (s *BlockSemaphore) Acquire(ctx context.Context, n int64) error {
	if n <= 0 {
		return nil
	}
	if n == 1 {
		select {
		case <-s.tokens:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var acquired int64
	for acquired < n {
		select {
		case <-s.tokens:
			acquired++
		case <-ctx.Done():
			s.releaseLocked(acquired)
			return ctx.Err()
		}
	}
	return nil
}

// Release releases n permits back to the semaphore.
func (s *BlockSemaphore) Release(n int64) {
	if n <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.releaseLocked(n)
}

func (s *BlockSemaphore) releaseLocked(n int64) {
	for i := int64(0); i < n; i++ {
		s.tokens <- struct{}{}
	}
}

type GenBlock interface {
	// Reuse resets the block for reuse.
	Reuse()

	// Deallocate releases the resources held by the block.
	Deallocate() error
}

// GenBlockPool is a generic block pool for managing blocks that implement the GenBlock interface.
// It offers methods to get blocks, return blocks to the free pool, and clear the free pool.
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

	mu sync.Mutex

	// Total number of blocks created so far.
	totalBlocks int64

	// Number of blocks reserved at the time of block pool creation.
	reservedBlocks int64

	// Semaphore used to limit the total number of blocks created across different files.
	globalMaxBlocksSem *BlockSemaphore

	// createBlockFunc is a function that creates a new block of type T
	createBlockFunc func(blockSize int64) (T, error)
}

// NewGenBlockPool creates the blockPool based on the user configuration.
func NewGenBlockPool[T GenBlock](
	blockSize int64,
	maxBlocks int64,
	reservedBlocks int64,
	globalMaxBlocksSem *BlockSemaphore,
	createBlockFunc func(blockSize int64) (T, error),
) (bp *GenBlockPool[T], err error) {
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
		return nil, CantAllocateAnyBlockError
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
func (bp *GenBlockPool[T]) Get() (T, error) {
	// Try to get a block immediately (non-blocking).
	b, err := bp.TryGet()
	if err == nil {
		return b, nil
	}
	if !errors.Is(err, CantAllocateAnyBlockError) {
		return b, err
	}

	total := bp.getTotalBlocks()

	// 1. At local pool limit: wait exclusively for a local block to be released.
	if total >= bp.maxBlocks {
		return bp.waitAndGetFromLocalPool()
	}

	// 2. Below local limit: wait for EITHER local release OR global permit.
	return bp.waitAndGetConcurrent()
}

func (bp *GenBlockPool[T]) waitAndGetFromLocalPool() (T, error) {
	b := <-bp.freeBlocksCh
	b.Reuse()
	return b, nil
}

func (bp *GenBlockPool[T]) waitAndGetConcurrent() (T, error) {
	select {
	case b := <-bp.freeBlocksCh:
		b.Reuse()
		return b, nil

	case <-bp.globalMaxBlocksSem.TokenChan():
		select {
		case b := <-bp.freeBlocksCh:
			bp.globalMaxBlocksSem.Release(1)
			b.Reuse()
			return b, nil
		default:
			return bp.allocateNewBlock(true)
		}
	}
}

// TryGet returns a block if available, or an error if no blocks can be allocated.
// It returns an existing block if it's ready for reuse or creates a new one if required.
func (bp *GenBlockPool[T]) TryGet() (T, error) {
	select {
	case b := <-bp.freeBlocksCh:
		b.Reuse()
		return b, nil
	default:
	}

	bp.mu.Lock()
	defer bp.mu.Unlock()

	if bp.totalBlocks >= bp.maxBlocks {
		var zero T
		return zero, CantAllocateAnyBlockError
	}

	wasPermitAcquired := false
	if bp.totalBlocks >= bp.reservedBlocks {
		if !bp.globalMaxBlocksSem.TryAcquire(1) {
			var zero T
			return zero, CantAllocateAnyBlockError
		}
		wasPermitAcquired = true
	}

	b, err := bp.createBlockFunc(bp.blockSize)
	if err != nil {
		if wasPermitAcquired {
			bp.globalMaxBlocksSem.Release(1)
		}
		var zero T
		return zero, err
	}
	bp.totalBlocks++
	return b, nil
}

// canAllocateBlock checks if a new block can be allocated.
func (bp *GenBlockPool[T]) canAllocateBlock() bool {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	if bp.totalBlocks >= bp.maxBlocks {
		return false
	}

	if bp.totalBlocks < bp.reservedBlocks {
		return true
	}

	return bp.globalMaxBlocksSem.TryAcquire(1)
}

// allocateNewBlock handles the physical allocation of a new block, tracking totalBlocks and returning any system error.
// If wasPermitAcquired is true, it releases the global semaphore permit on allocation failure or if maxBlocks limit was reached.
func (bp *GenBlockPool[T]) allocateNewBlock(wasPermitAcquired bool) (T, error) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	if bp.totalBlocks >= bp.maxBlocks {
		if wasPermitAcquired {
			bp.globalMaxBlocksSem.Release(1)
		}
		var zero T
		return zero, CantAllocateAnyBlockError
	}

	b, err := bp.createBlockFunc(bp.blockSize)
	if err != nil {
		if wasPermitAcquired {
			bp.globalMaxBlocksSem.Release(1)
		}
		var zero T
		return zero, err
	}
	bp.totalBlocks++
	return b, nil
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
			bp.mu.Lock()
			if bp.totalBlocks > bp.reservedBlocks {
				bp.globalMaxBlocksSem.Release(1)
			}
			bp.totalBlocks--
			bp.mu.Unlock()
		default:
			// We are here, it means there are no more blocks in the free blocks channel.
			// Release semaphore for the released blocks iff releaseReservedBlocks is true.
			if releaseReservedBlocks {
				bp.mu.Lock()
				if bp.reservedBlocks > 0 {
					bp.globalMaxBlocksSem.Release(bp.reservedBlocks)
					bp.reservedBlocks = 0
				}
				bp.mu.Unlock()
			}
			return nil
		}
	}
}

func (bp *GenBlockPool[T]) getTotalBlocks() int64 {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	return bp.totalBlocks
}

// TotalFreeBlocks returns the total number of free blocks available in the pool.
// This is useful for testing and debugging purposes.
func (bp *GenBlockPool[T]) TotalFreeBlocks() int {
	return len(bp.freeBlocksCh)
}

// NewBlockPool creates GenBlockPool for block.Block interface.
func NewBlockPool(blockSize int64, maxBlocks int64, reservedBlocks int64, globalMaxBlocksSem *BlockSemaphore) (bp *GenBlockPool[Block], err error) {
	return NewGenBlockPool(blockSize, maxBlocks, reservedBlocks, globalMaxBlocksSem, createBlock)
}

// NewPrefetchBlockPool creates GenBlockPool for block.PrefetchBlock interface.
func NewPrefetchBlockPool(blockSize int64, maxBlocks int64, reservedBlocks int64, globalMaxBlocksSem *BlockSemaphore) (bp *GenBlockPool[PrefetchBlock], err error) {
	return NewGenBlockPool(blockSize, maxBlocks, reservedBlocks, globalMaxBlocksSem, createPrefetchBlock)
}

