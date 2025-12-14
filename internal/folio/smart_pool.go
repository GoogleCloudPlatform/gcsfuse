// Copyright 2025 Google LLC
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

package folio

import (
	"fmt"
	"sync"
)

// PoolConfig specifies the block size and optional reserved memory for a pool
type PoolConfig struct {
	BlockSize      int   // Size of each block in bytes
	ReservedMemory int64 // Optional: memory to reserve in bytes (0 = no reservation)
}

// SmartPool intelligently manages multiple pools of different block sizes
// and returns a list of blocks to satisfy any requested memory size.
type SmartPool struct {
	pools          []*Pool       // Sorted by block size (largest first)
	blockSizes     []int         // Block sizes corresponding to pools (largest first)
	reservedMemory map[int]int64 // Reserved memory in bytes per block size
	totalReserved  int64         // Total reserved memory across all pools
	mu             sync.RWMutex
}

// Allocation represents a memory allocation composed of multiple blocks
type Allocation struct {
	Blocks     []*Block
	TotalSize  int
	blockSizes []int      // Track size of each block for proper release
	smartPool  *SmartPool // Reference back to pool for release
}

// NewSmartPool creates a new smart pool with the specified block sizes.
// Block sizes will be sorted from largest to smallest automatically.
// If no block sizes are provided, defaults to [1MB, 64KB].
func NewSmartPool(blockSizes ...int) (*SmartPool, error) {
	configs := make([]PoolConfig, len(blockSizes))
	for i, size := range blockSizes {
		configs[i] = PoolConfig{BlockSize: size, ReservedMemory: 0}
	}
	return NewSmartPoolWithReserved(configs...)
}

// NewSmartPoolWithReserved creates a new smart pool with reserved memory per pool size.
// Each PoolConfig specifies the block size and optional reserved memory for that pool.
// Block sizes will be sorted from largest to smallest automatically.
// If no configs are provided, defaults to [1MB, 64KB] with no reservation.
func NewSmartPoolWithReserved(configs ...PoolConfig) (*SmartPool, error) {
	// Default to 1MB and 64KB if not specified
	if len(configs) == 0 {
		configs = []PoolConfig{
			{BlockSize: Block1MB, ReservedMemory: 0},
			{BlockSize: Block64KB, ReservedMemory: 0},
		}
	}

	// Validate configs and build maps
	reservedMemory := make(map[int]int64)
	var totalReserved int64
	blockSizes := make([]int, len(configs))

	for i, cfg := range configs {
		if cfg.BlockSize <= 0 {
			return nil, fmt.Errorf("invalid block size: %d", cfg.BlockSize)
		}
		if cfg.ReservedMemory < 0 {
			return nil, fmt.Errorf("invalid reserved memory for size %d: %d bytes", cfg.BlockSize, cfg.ReservedMemory)
		}
		blockSizes[i] = cfg.BlockSize
		reservedMemory[cfg.BlockSize] = cfg.ReservedMemory
		totalReserved += cfg.ReservedMemory
	}

	// Sort block sizes from largest to smallest
	sortedSizes := make([]int, len(blockSizes))
	copy(sortedSizes, blockSizes)
	for i := 0; i < len(sortedSizes); i++ {
		for j := i + 1; j < len(sortedSizes); j++ {
			if sortedSizes[i] < sortedSizes[j] {
				sortedSizes[i], sortedSizes[j] = sortedSizes[j], sortedSizes[i]
			}
		}
	}

	// Create pools for each block size
	pools := make([]*Pool, len(sortedSizes))
	for i, size := range sortedSizes {
		reservedBytes := reservedMemory[size]
		pool, err := NewPool(BlockSize(size), reservedBytes)
		if err != nil {
			// Cleanup already created pools
			for j := 0; j < i; j++ {
				pools[j].Close()
			}
			return nil, fmt.Errorf("failed to create pool for block size %d: %w", size, err)
		}
		pools[i] = pool
	}

	return &SmartPool{
		pools:          pools,
		blockSizes:     sortedSizes,
		reservedMemory: reservedMemory,
		totalReserved:  totalReserved,
	}, nil
}

// Allocate returns a list of blocks that satisfy the requested size.
// Strategy:
// - Use largest blocks first for bulk allocation
// - Use progressively smaller blocks for remainder
// - Optimize to minimize number of blocks
func (sp *SmartPool) Allocate(size int) (*Allocation, error) {
	if size <= 0 {
		return nil, fmt.Errorf("invalid size: %d", size)
	}

	sp.mu.RLock()
	smallestBlockSize := sp.blockSizes[len(sp.blockSizes)-1]

	// Calculate total number of blocks needed to pre-allocate slices
	remaining := size
	totalBlocks := 0

	// Special case: if size is smaller than 4x the smallest block, use only smallest blocks
	if len(sp.blockSizes) > 1 && size < 4*smallestBlockSize {
		totalBlocks = (size + smallestBlockSize - 1) / smallestBlockSize
	} else {
		// Estimate total blocks needed
		tempRemaining := size
		for _, blockSize := range sp.blockSizes {
			numBlocks := tempRemaining / blockSize
			totalBlocks += numBlocks
			tempRemaining = tempRemaining % blockSize
		}
		if tempRemaining > 0 {
			totalBlocks += (tempRemaining + smallestBlockSize - 1) / smallestBlockSize
		}
	}

	// Pre-allocate slices to avoid repeated allocations
	blocks := make([]*Block, 0, totalBlocks)
	blockSizes := make([]int, 0, totalBlocks)
	sp.mu.RUnlock()

	// Special case: if size is smaller than 4x the smallest block, use only smallest blocks
	if len(sp.blockSizes) > 1 && size < 4*smallestBlockSize {
		sp.mu.RLock()
		smallestPool := sp.pools[len(sp.pools)-1]
		sp.mu.RUnlock()

		numBlocks := (size + smallestBlockSize - 1) / smallestBlockSize

		// Use batch acquire for efficiency
		batchBlocks, err := smallestPool.AcquireBatch(numBlocks)
		if err != nil {
			return nil, fmt.Errorf("failed to acquire %d blocks of size %d: %w", numBlocks, smallestBlockSize, err)
		}
		blocks = append(blocks, batchBlocks...)
		for range batchBlocks {
			blockSizes = append(blockSizes, smallestBlockSize)
		}

		return &Allocation{
			Blocks:     blocks,
			TotalSize:  size,
			blockSizes: blockSizes,
			smartPool:  sp,
		}, nil
	}

	// General strategy: use largest blocks first, then progressively smaller blocks
	sp.mu.RLock()
	poolsCopy := sp.pools
	blockSizesCopy := sp.blockSizes
	sp.mu.RUnlock()

	for i, blockSize := range blockSizesCopy {
		if remaining <= 0 {
			break
		}

		pool := poolsCopy[i]
		numBlocks := remaining / blockSize

		if numBlocks > 0 {
			// Use batch acquire for efficiency
			batchBlocks, err := pool.AcquireBatch(numBlocks)
			if err != nil {
				sp.releaseBlocks(blocks, blockSizes)
				return nil, fmt.Errorf("failed to acquire %d blocks of size %d: %w", numBlocks, blockSize, err)
			}
			blocks = append(blocks, batchBlocks...)
			for range batchBlocks {
				blockSizes = append(blockSizes, blockSize)
			}
			remaining -= numBlocks * blockSize
		}
	}

	// Handle any remaining bytes with the smallest block size
	if remaining > 0 {
		sp.mu.RLock()
		smallestPool := sp.pools[len(sp.pools)-1]
		sp.mu.RUnlock()

		numBlocks := (remaining + smallestBlockSize - 1) / smallestBlockSize

		// Use batch acquire for efficiency
		batchBlocks, err := smallestPool.AcquireBatch(numBlocks)
		if err != nil {
			sp.releaseBlocks(blocks, blockSizes)
			return nil, fmt.Errorf("failed to acquire %d blocks of size %d: %w", numBlocks, smallestBlockSize, err)
		}
		blocks = append(blocks, batchBlocks...)
		for range batchBlocks {
			blockSizes = append(blockSizes, smallestBlockSize)
		}
	}

	return &Allocation{
		Blocks:     blocks,
		TotalSize:  size,
		blockSizes: blockSizes,
		smartPool:  sp,
	}, nil
}

// AllocateExact returns blocks that exactly match the requested size.
// This is useful when you need precise memory layout.
func (sp *SmartPool) AllocateExact(size int) (*Allocation, error) {
	if size <= 0 {
		return nil, fmt.Errorf("invalid size: %d", size)
	}

	sp.mu.RLock()
	defer sp.mu.RUnlock()

	// Check if size is a multiple of any available block size
	for i, blockSize := range sp.blockSizes {
		if size%blockSize == 0 {
			numBlocks := size / blockSize
			return sp.allocateFromPool(sp.pools[i], numBlocks, blockSize)
		}
	}

	// Not an exact multiple, fall back to regular allocation
	return sp.Allocate(size)
}

// allocateFromPool is a helper to allocate multiple blocks from a single pool
func (sp *SmartPool) allocateFromPool(pool *Pool, numBlocks, blockSize int) (*Allocation, error) {
	blocks := make([]*Block, 0, numBlocks)
	blockSizes := make([]int, 0, numBlocks)

	for i := 0; i < numBlocks; i++ {
		block, err := pool.Acquire()
		if err != nil {
			// Cleanup on failure
			sp.releaseBlocks(blocks, blockSizes)
			return nil, fmt.Errorf("failed to acquire block %d: %w", i, err)
		}
		blocks = append(blocks, block)
		blockSizes = append(blockSizes, blockSize)
	}

	return &Allocation{
		Blocks:     blocks,
		TotalSize:  numBlocks * blockSize,
		blockSizes: blockSizes,
		smartPool:  sp,
	}, nil
}

// Release returns all blocks in the allocation back to their respective pools
func (sp *SmartPool) Release(alloc *Allocation) {
	if alloc == nil {
		return
	}

	sp.releaseBlocks(alloc.Blocks, alloc.blockSizes)

	// Clear the allocation
	alloc.Blocks = nil
	alloc.blockSizes = nil
	alloc.TotalSize = 0
}

// releaseBlocks is a helper to release blocks back to appropriate pools
func (sp *SmartPool) releaseBlocks(blocks []*Block, blockSizes []int) {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	for i, block := range blocks {
		if i >= len(blockSizes) {
			continue
		}

		// Find the pool matching this block size
		for j, poolBlockSize := range sp.blockSizes {
			if blockSizes[i] == poolBlockSize {
				sp.pools[j].Release(block)
				break
			}
		}
	}
}

// Stats returns combined statistics from all pools
func (sp *SmartPool) Stats() SmartPoolStats {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	poolStats := make([]PoolStats, len(sp.pools))
	for i, pool := range sp.pools {
		poolStats[i] = pool.Stats()
	}

	return SmartPoolStats{
		PoolStats: poolStats,
	}
}

// SmartPoolStats contains statistics from all pools
type SmartPoolStats struct {
	PoolStats []PoolStats
}

// Close releases all resources from all pools
func (sp *SmartPool) Close() error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	var firstErr error
	for _, pool := range sp.pools {
		if pool != nil {
			if err := pool.Close(); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}

	return firstErr
}

// GetContiguousView returns a contiguous view of the allocated memory.
// Note: This creates a new slice and copies data, so use sparingly.
// For most operations, work directly with individual blocks for better performance.
func (alloc *Allocation) GetContiguousView() []byte {
	if alloc == nil || len(alloc.Blocks) == 0 {
		return nil
	}

	result := make([]byte, alloc.TotalSize)
	offset := 0

	for i, block := range alloc.Blocks {
		blockSize := alloc.blockSizes[i]
		copySize := minInt(blockSize, alloc.TotalSize-offset)
		copy(result[offset:offset+copySize], block.Data[:copySize])
		offset += copySize
	}

	return result
}

// WriteToBlocks writes data across the blocks in the allocation
func (alloc *Allocation) WriteToBlocks(data []byte) error {
	if alloc == nil || len(alloc.Blocks) == 0 {
		return fmt.Errorf("invalid allocation")
	}

	if len(data) > alloc.TotalSize {
		return fmt.Errorf("data size %d exceeds allocation size %d", len(data), alloc.TotalSize)
	}

	offset := 0
	for i, block := range alloc.Blocks {
		if offset >= len(data) {
			break
		}

		blockSize := alloc.blockSizes[i]
		copySize := minInt(blockSize, len(data)-offset)
		copy(block.Data[:copySize], data[offset:offset+copySize])
		offset += copySize
	}

	return nil
}

// ReservedMemory returns a map of block sizes to their reserved memory in bytes
func (sp *SmartPool) ReservedMemory() map[int]int64 {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make(map[int]int64, len(sp.reservedMemory))
	for k, v := range sp.reservedMemory {
		result[k] = v
	}
	return result
}

// TotalReservedMemory returns the total reserved memory across all pools in bytes
func (sp *SmartPool) TotalReservedMemory() int64 {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	return sp.totalReserved
}

// TotalAllocatedMemory returns the total currently allocated memory across all pools
func (sp *SmartPool) TotalAllocatedMemory() int64 {
	sp.mu.RLock()
	pools := sp.pools
	sp.mu.RUnlock()

	var total int64
	for _, pool := range pools {
		total += pool.AllocatedMemory()
	}
	return total
}

// GetPoolForSize returns the pool for the specified block size, or nil if not found
func (sp *SmartPool) GetPoolForSize(blockSize int) *Pool {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	for i, size := range sp.blockSizes {
		if size == blockSize {
			return sp.pools[i]
		}
	}
	return nil
}

// ReadFromBlocks reads data from the blocks in the allocation
func (alloc *Allocation) ReadFromBlocks(dst []byte) (int, error) {
	if alloc == nil || len(alloc.Blocks) == 0 {
		return 0, fmt.Errorf("invalid allocation")
	}

	totalRead := 0
	offset := 0

	for i, block := range alloc.Blocks {
		if offset >= len(dst) {
			break
		}

		blockSize := alloc.blockSizes[i]
		available := minInt(blockSize, alloc.TotalSize-totalRead)
		copySize := minInt(available, len(dst)-offset)

		copy(dst[offset:offset+copySize], block.Data[:copySize])
		offset += copySize
		totalRead += copySize
	}

	return totalRead, nil
}

// NumBlocks returns the number of blocks in this allocation
func (alloc *Allocation) NumBlocks() int {
	if alloc == nil {
		return 0
	}
	return len(alloc.Blocks)
}
