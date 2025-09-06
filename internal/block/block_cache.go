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
	"container/list"
	"context"
	"fmt"
	"sync"

	"golang.org/x/sync/semaphore"
)

// CacheKey represents a unique identifier for cached blocks
type CacheKey string

// CachedBlock represents a block with metadata for caching
type CachedBlock struct {
	Block
	Key         CacheKey
	Size        int64
	RefCount    int32
	LastAccess  int64 // timestamp for LRU tracking
	element     *list.Element
}

// BlockCache represents a cache of blocks with LRU eviction policy
type BlockCache struct {
	mu           sync.RWMutex
	blockPool    *GenBlockPool[Block]
	blocks       map[CacheKey]*CachedBlock
	lruList      *list.List
	maxBlocks    int64
	currentCount int64
	accessTime   int64 // monotonic counter for LRU ordering
	
	// Async download manager (optional)
	downloadManager *AsyncDownloadManager
}

// BlockCacheConfig holds configuration for the block cache
type BlockCacheConfig struct {
	BlockSize         int64
	MaxBlocks         int64
	MaxBlocksPerFile  int64
	ReservedBlocks    int64
	BlockType         BlockType
	GlobalMaxBlocksSem *semaphore.Weighted
}

// NewBlockCache creates a new block cache with the specified configuration
func NewBlockCache(config *BlockCacheConfig) (*BlockCache, error) {
	if config.MaxBlocks <= 0 {
		return nil, fmt.Errorf("maxBlocks must be greater than 0")
	}
	
	if config.BlockSize <= 0 {
		return nil, fmt.Errorf("blockSize must be greater than 0")
	}

	var blockPool *GenBlockPool[Block]
	var err error
	
	// Create the appropriate block pool based on the block type
	switch config.BlockType {
	case DiskBlock:
		blockPool, err = NewDiskBlockPool(config.BlockSize, config.MaxBlocksPerFile, config.ReservedBlocks, config.GlobalMaxBlocksSem)
	case MemoryBlock:
		blockPool, err = NewMemoryBlockPool(config.BlockSize, config.MaxBlocksPerFile, config.ReservedBlocks, config.GlobalMaxBlocksSem)
	default:
		blockPool, err = NewBlockPool(config.BlockSize, config.MaxBlocksPerFile, config.ReservedBlocks, config.GlobalMaxBlocksSem)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to create block pool: %w", err)
	}

	return &BlockCache{
		blockPool:    blockPool,
		blocks:       make(map[CacheKey]*CachedBlock),
		lruList:      list.New(),
		maxBlocks:    config.MaxBlocks,
		currentCount: 0,
		accessTime:   0,
	}, nil
}

// Get retrieves a block from the cache or creates a new one if not found
func (bc *BlockCache) Get(key CacheKey) (*CachedBlock, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Check if block exists in cache
	if cachedBlock, exists := bc.blocks[key]; exists {
		// Update access time and move to front of LRU list
		bc.accessTime++
		cachedBlock.LastAccess = bc.accessTime
		cachedBlock.RefCount++
		bc.lruList.MoveToFront(cachedBlock.element)
		return cachedBlock, nil
	}

	// Block not in cache, create a new one
	block, err := bc.blockPool.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get block from pool: %w", err)
	}

	// Check if we need to evict blocks to make room
	if bc.currentCount >= bc.maxBlocks {
		if err := bc.evictLRU(); err != nil {
			// Return the block to the pool since we couldn't make room
			bc.blockPool.Release(block)
			return nil, fmt.Errorf("failed to evict LRU block: %w", err)
		}
	}

	// Create cached block
	bc.accessTime++
	cachedBlock := &CachedBlock{
		Block:      block,
		Key:        key,
		Size:       block.Cap(),
		RefCount:   1,
		LastAccess: bc.accessTime,
	}

	// Add to cache
	cachedBlock.element = bc.lruList.PushFront(cachedBlock)
	bc.blocks[key] = cachedBlock
	bc.currentCount++

	return cachedBlock, nil
}

// Release decrements the reference count and potentially makes the block available for eviction
func (bc *BlockCache) Release(cachedBlock *CachedBlock) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if cachedBlock.RefCount > 0 {
		cachedBlock.RefCount--
	}
}

// Remove explicitly removes a block from the cache
func (bc *BlockCache) Remove(key CacheKey) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	cachedBlock, exists := bc.blocks[key]
	if !exists {
		return nil // Block not in cache, nothing to do
	}

	if cachedBlock.RefCount > 0 {
		return fmt.Errorf("cannot remove block with active references (refCount: %d)", cachedBlock.RefCount)
	}

	return bc.removeBlock(cachedBlock)
}

// evictLRU evicts the least recently used block that has no active references
func (bc *BlockCache) evictLRU() error {
	for element := bc.lruList.Back(); element != nil; element = element.Prev() {
		cachedBlock := element.Value.(*CachedBlock)
		
		// Only evict blocks with no active references
		if cachedBlock.RefCount == 0 {
			return bc.removeBlock(cachedBlock)
		}
	}
	
	return fmt.Errorf("no blocks available for eviction (all blocks have active references)")
}

// removeBlock removes a block from the cache and returns it to the pool
func (bc *BlockCache) removeBlock(cachedBlock *CachedBlock) error {
	// Remove from LRU list
	bc.lruList.Remove(cachedBlock.element)
	
	// Remove from cache map
	delete(bc.blocks, cachedBlock.Key)
	
	// Return block to pool
	bc.blockPool.Release(cachedBlock.Block)
	
	bc.currentCount--
	return nil
}

// Stats returns cache statistics
func (bc *BlockCache) Stats() BlockCacheStats {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	return BlockCacheStats{
		TotalBlocks:     bc.currentCount,
		MaxBlocks:       bc.maxBlocks,
		BlocksInUse:     bc.countBlocksInUse(),
		BlocksAvailable: bc.currentCount - bc.countBlocksInUse(),
	}
}

// countBlocksInUse counts blocks with active references
func (bc *BlockCache) countBlocksInUse() int64 {
	var inUse int64
	for _, cachedBlock := range bc.blocks {
		if cachedBlock.RefCount > 0 {
			inUse++
		}
	}
	return inUse
}

// Clear removes all blocks from the cache
func (bc *BlockCache) Clear() error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Check if any blocks are still in use
	for key, cachedBlock := range bc.blocks {
		if cachedBlock.RefCount > 0 {
			return fmt.Errorf("cannot clear cache: block %s has active references (refCount: %d)", key, cachedBlock.RefCount)
		}
	}

	// Remove all blocks
	for key, cachedBlock := range bc.blocks {
		bc.blockPool.Release(cachedBlock.Block)
		delete(bc.blocks, key)
	}

	bc.lruList.Init()
	bc.currentCount = 0
	
	// Cancel any active downloads if download manager is configured
	if bc.downloadManager != nil {
		bc.downloadManager.Shutdown()
	}
	
	return nil
}

// SetAsyncDownloadManager sets the async download manager for the cache
func (bc *BlockCache) SetAsyncDownloadManager(manager *AsyncDownloadManager) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.downloadManager = manager
}

// ScheduleAsyncDownload schedules an asynchronous download for a block
// This method requires that an AsyncDownloadManager has been set
func (bc *BlockCache) ScheduleAsyncDownload(ctx context.Context, request *BlockDownloadRequest) (*AsyncBlockDownloadTask, error) {
	bc.mu.RLock()
	manager := bc.downloadManager
	bc.mu.RUnlock()
	
	if manager == nil {
		return nil, fmt.Errorf("async download manager not configured")
	}
	
	return manager.ScheduleDownload(ctx, request)
}

// GetAsyncDownloadStatus returns the status of an async download
func (bc *BlockCache) GetAsyncDownloadStatus(key CacheKey) (*DownloadStatus, error) {
	bc.mu.RLock()
	manager := bc.downloadManager
	bc.mu.RUnlock()
	
	if manager == nil {
		return nil, fmt.Errorf("async download manager not configured")
	}
	
	return manager.GetDownloadStatus(key)
}

// CancelAsyncDownload cancels an active async download
func (bc *BlockCache) CancelAsyncDownload(key CacheKey) error {
	bc.mu.RLock()
	manager := bc.downloadManager
	bc.mu.RUnlock()
	
	if manager == nil {
		return fmt.Errorf("async download manager not configured")
	}
	
	return manager.CancelDownload(key)
}

// ListActiveDownloads returns a list of active download keys
func (bc *BlockCache) ListActiveDownloads() []CacheKey {
	bc.mu.RLock()
	manager := bc.downloadManager
	bc.mu.RUnlock()
	
	if manager == nil {
		return nil
	}
	
	return manager.ListActiveDownloads()
}

// CleanupCompletedDownloads removes completed downloads from tracking
func (bc *BlockCache) CleanupCompletedDownloads() int {
	bc.mu.RLock()
	manager := bc.downloadManager
	bc.mu.RUnlock()
	
	if manager == nil {
		return 0
	}
	
	return manager.CleanupCompletedDownloads()
}

// GetOrScheduleDownload gets a block from cache or schedules its download if not present
// This is a convenience method that combines cache lookup with async download
func (bc *BlockCache) GetOrScheduleDownload(ctx context.Context, request *BlockDownloadRequest) (*CachedBlock, *AsyncBlockDownloadTask, error) {
	// First try to get the block from cache
	block, err := bc.Get(request.Key)
	if err == nil {
		// Block already in cache
		return block, nil, nil
	}
	
	// Block not in cache, schedule download if manager is available
	bc.mu.RLock()
	manager := bc.downloadManager
	bc.mu.RUnlock()
	
	if manager == nil {
		return nil, nil, fmt.Errorf("block not in cache and async download manager not configured")
	}
	
	// Schedule the download
	task, err := manager.ScheduleDownload(ctx, request)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to schedule download: %w", err)
	}
	
	return nil, task, nil
}

// Destroy cleans up the cache and its underlying block pool
func (bc *BlockCache) Destroy() error {
	if err := bc.Clear(); err != nil {
		return err
	}
	
	// Clear the block pool's free blocks but don't release reserved blocks
	return bc.blockPool.ClearFreeBlockChannel(false)
}

// BlockCacheStats contains cache statistics
type BlockCacheStats struct {
	TotalBlocks     int64
	MaxBlocks       int64
	BlocksInUse     int64
	BlocksAvailable int64
}

// String returns a human-readable representation of the cache stats
func (stats BlockCacheStats) String() string {
	return fmt.Sprintf("BlockCache{total: %d, max: %d, inUse: %d, available: %d}", 
		stats.TotalBlocks, stats.MaxBlocks, stats.BlocksInUse, stats.BlocksAvailable)
}
