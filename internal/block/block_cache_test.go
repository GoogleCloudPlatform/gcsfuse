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
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/semaphore"
)

type BlockCacheTest struct {
	suite.Suite
	cache *BlockCache
}

func TestBlockCacheTestSuite(t *testing.T) {
	suite.Run(t, new(BlockCacheTest))
}

func (testSuite *BlockCacheTest) SetupTest() {
	config := &BlockCacheConfig{
		BlockSize:          1024,
		MaxBlocks:          5,
		MaxBlocksPerFile:   10,
		ReservedBlocks:     1,
		BlockType:          MemoryBlock,
		GlobalMaxBlocksSem: semaphore.NewWeighted(100),
	}
	
	var err error
	testSuite.cache, err = NewBlockCache(config)
	require.Nil(testSuite.T(), err)
}

func (testSuite *BlockCacheTest) TearDownTest() {
	if testSuite.cache != nil {
		testSuite.cache.Destroy()
	}
}

func (testSuite *BlockCacheTest) TestNewBlockCache() {
	config := &BlockCacheConfig{
		BlockSize:          2048,
		MaxBlocks:          10,
		MaxBlocksPerFile:   15,
		ReservedBlocks:     2,
		BlockType:          DiskBlock,
		GlobalMaxBlocksSem: semaphore.NewWeighted(100),
	}
	
	cache, err := NewBlockCache(config)
	require.Nil(testSuite.T(), err)
	require.NotNil(testSuite.T(), cache)
	defer cache.Destroy()
	
	stats := cache.Stats()
	assert.Equal(testSuite.T(), int64(0), stats.TotalBlocks)
	assert.Equal(testSuite.T(), int64(10), stats.MaxBlocks)
}

func (testSuite *BlockCacheTest) TestNewBlockCacheInvalidConfig() {
	// Test invalid MaxBlocks
	config := &BlockCacheConfig{
		BlockSize:          1024,
		MaxBlocks:          0,
		MaxBlocksPerFile:   10,
		ReservedBlocks:     1,
		BlockType:          MemoryBlock,
		GlobalMaxBlocksSem: semaphore.NewWeighted(100),
	}
	
	cache, err := NewBlockCache(config)
	assert.NotNil(testSuite.T(), err)
	assert.Nil(testSuite.T(), cache)
	assert.Contains(testSuite.T(), err.Error(), "maxBlocks must be greater than 0")
	
	// Test invalid BlockSize
	config.MaxBlocks = 5
	config.BlockSize = 0
	cache, err = NewBlockCache(config)
	assert.NotNil(testSuite.T(), err)
	assert.Nil(testSuite.T(), cache)
	assert.Contains(testSuite.T(), err.Error(), "blockSize must be greater than 0")
}

func (testSuite *BlockCacheTest) TestGetNewBlock() {
	key := CacheKey("test-block-1")
	
	cachedBlock, err := testSuite.cache.Get(key)
	require.Nil(testSuite.T(), err)
	require.NotNil(testSuite.T(), cachedBlock)
	
	assert.Equal(testSuite.T(), key, cachedBlock.Key)
	assert.Equal(testSuite.T(), int64(1024), cachedBlock.Size)
	assert.Equal(testSuite.T(), int32(1), cachedBlock.RefCount)
	assert.NotNil(testSuite.T(), cachedBlock.Block)
	
	stats := testSuite.cache.Stats()
	assert.Equal(testSuite.T(), int64(1), stats.TotalBlocks)
	assert.Equal(testSuite.T(), int64(1), stats.BlocksInUse)
	assert.Equal(testSuite.T(), int64(0), stats.BlocksAvailable)
}

func (testSuite *BlockCacheTest) TestGetExistingBlock() {
	key := CacheKey("test-block-1")
	
	// Get block first time
	cachedBlock1, err := testSuite.cache.Get(key)
	require.Nil(testSuite.T(), err)
	
	// Get same block again
	cachedBlock2, err := testSuite.cache.Get(key)
	require.Nil(testSuite.T(), err)
	
	// Should be the same block with incremented ref count
	assert.Equal(testSuite.T(), cachedBlock1.Block, cachedBlock2.Block)
	assert.Equal(testSuite.T(), int32(2), cachedBlock2.RefCount)
	
	stats := testSuite.cache.Stats()
	assert.Equal(testSuite.T(), int64(1), stats.TotalBlocks)
	assert.Equal(testSuite.T(), int64(1), stats.BlocksInUse)
}

func (testSuite *BlockCacheTest) TestReleaseBlock() {
	key := CacheKey("test-block-1")
	
	cachedBlock, err := testSuite.cache.Get(key)
	require.Nil(testSuite.T(), err)
	
	// Release the block
	testSuite.cache.Release(cachedBlock)
	assert.Equal(testSuite.T(), int32(0), cachedBlock.RefCount)
	
	stats := testSuite.cache.Stats()
	assert.Equal(testSuite.T(), int64(1), stats.TotalBlocks)
	assert.Equal(testSuite.T(), int64(0), stats.BlocksInUse)
	assert.Equal(testSuite.T(), int64(1), stats.BlocksAvailable)
}

func (testSuite *BlockCacheTest) TestRemoveBlock() {
	key := CacheKey("test-block-1")
	
	cachedBlock, err := testSuite.cache.Get(key)
	require.Nil(testSuite.T(), err)
	
	// Cannot remove block with active references
	err = testSuite.cache.Remove(key)
	assert.NotNil(testSuite.T(), err)
	assert.Contains(testSuite.T(), err.Error(), "cannot remove block with active references")
	
	// Release the block
	testSuite.cache.Release(cachedBlock)
	
	// Now we can remove it
	err = testSuite.cache.Remove(key)
	assert.Nil(testSuite.T(), err)
	
	stats := testSuite.cache.Stats()
	assert.Equal(testSuite.T(), int64(0), stats.TotalBlocks)
}

func (testSuite *BlockCacheTest) TestLRUEviction() {
	// Fill cache to capacity (5 blocks)
	keys := make([]CacheKey, 5)
	blocks := make([]*CachedBlock, 5)
	
	for i := 0; i < 5; i++ {
		keys[i] = CacheKey(fmt.Sprintf("block-%d", i))
		var err error
		blocks[i], err = testSuite.cache.Get(keys[i])
		require.Nil(testSuite.T(), err)
		
		// Release blocks to make them available for eviction
		testSuite.cache.Release(blocks[i])
	}
	
	stats := testSuite.cache.Stats()
	assert.Equal(testSuite.T(), int64(5), stats.TotalBlocks)
	
	// Add one more block - should evict the LRU (first) block
	newKey := CacheKey("block-new")
	newBlock, err := testSuite.cache.Get(newKey)
	require.Nil(testSuite.T(), err)
	defer testSuite.cache.Release(newBlock)
	
	// Cache should still have 5 blocks (one evicted, one added)
	stats = testSuite.cache.Stats()
	assert.Equal(testSuite.T(), int64(5), stats.TotalBlocks)
	
	// The first block (LRU) should have been evicted
	// Try to get it again - should create a new cached block (even if same underlying block)
	firstBlock, err := testSuite.cache.Get(keys[0])
	require.Nil(testSuite.T(), err)
	defer testSuite.cache.Release(firstBlock)
	
	// Should be a different cached block instance (cache entry was removed and recreated)
	// Note: underlying block may be reused from pool, but cached block should be different
	assert.True(testSuite.T(), firstBlock != blocks[0], "cached block should be different after eviction")
}

func (testSuite *BlockCacheTest) TestLRUEvictionOrder() {
	// Create and release 3 blocks
	blocks := make([]*CachedBlock, 3)
	for i := 0; i < 3; i++ {
		key := CacheKey(fmt.Sprintf("block-%d", i))
		var err error
		blocks[i], err = testSuite.cache.Get(key)
		require.Nil(testSuite.T(), err)
		testSuite.cache.Release(blocks[i])
	}
	
	// Access block 0 again to make it more recently used
	updatedBlock0, err := testSuite.cache.Get(CacheKey("block-0"))
	require.Nil(testSuite.T(), err)
	testSuite.cache.Release(updatedBlock0)
	
	// Fill remaining capacity
	for i := 3; i < 5; i++ {
		key := CacheKey(fmt.Sprintf("block-%d", i))
		block, err := testSuite.cache.Get(key)
		require.Nil(testSuite.T(), err)
		testSuite.cache.Release(block)
	}
	
	// Add one more to trigger eviction
	newBlock, err := testSuite.cache.Get(CacheKey("block-new"))
	require.Nil(testSuite.T(), err)
	defer testSuite.cache.Release(newBlock)
	
	// Block 1 should have been evicted (it was LRU after block 0 was accessed)
	// Block 0 should still be in cache
	block0, err := testSuite.cache.Get(CacheKey("block-0"))
	require.Nil(testSuite.T(), err)
	defer testSuite.cache.Release(block0)
	assert.True(testSuite.T(), updatedBlock0 == block0, "block 0 should be same cached instance") // Same cached block instance
	
	// Try to get block 1 - should be a new cached instance (was evicted)
	block1, err := testSuite.cache.Get(CacheKey("block-1"))
	require.Nil(testSuite.T(), err)
	defer testSuite.cache.Release(block1)
	assert.True(testSuite.T(), blocks[1] != block1, "block 1 should be different cached instance after eviction")
}

func (testSuite *BlockCacheTest) TestEvictionWithActiveReferences() {
	// Fill cache and keep references to all blocks
	blocks := make([]*CachedBlock, 5)
	for i := 0; i < 5; i++ {
		key := CacheKey(fmt.Sprintf("block-%d", i))
		var err error
		blocks[i], err = testSuite.cache.Get(key)
		require.Nil(testSuite.T(), err)
		// Don't release - keep references active
	}
	
	// Try to add one more block - should fail because no blocks can be evicted
	newKey := CacheKey("block-new")
	newBlock, err := testSuite.cache.Get(newKey)
	assert.NotNil(testSuite.T(), err)
	assert.Nil(testSuite.T(), newBlock)
	assert.Contains(testSuite.T(), err.Error(), "no blocks available for eviction")
	
	// Release one block
	testSuite.cache.Release(blocks[0])
	
	// Now we should be able to add a new block
	newBlock, err = testSuite.cache.Get(newKey)
	assert.Nil(testSuite.T(), err)
	assert.NotNil(testSuite.T(), newBlock)
	defer testSuite.cache.Release(newBlock)
	
	// Clean up
	for i := 1; i < 5; i++ {
		testSuite.cache.Release(blocks[i])
	}
}

func (testSuite *BlockCacheTest) TestClear() {
	// Add some blocks
	for i := 0; i < 3; i++ {
		key := CacheKey(fmt.Sprintf("block-%d", i))
		block, err := testSuite.cache.Get(key)
		require.Nil(testSuite.T(), err)
		testSuite.cache.Release(block)
	}
	
	stats := testSuite.cache.Stats()
	assert.Equal(testSuite.T(), int64(3), stats.TotalBlocks)
	
	// Clear cache
	err := testSuite.cache.Clear()
	assert.Nil(testSuite.T(), err)
	
	stats = testSuite.cache.Stats()
	assert.Equal(testSuite.T(), int64(0), stats.TotalBlocks)
}

func (testSuite *BlockCacheTest) TestClearWithActiveReferences() {
	// Add a block with active reference
	key := CacheKey("block-1")
	block, err := testSuite.cache.Get(key)
	require.Nil(testSuite.T(), err)
	// Don't release - keep reference active
	
	// Clear should fail
	err = testSuite.cache.Clear()
	assert.NotNil(testSuite.T(), err)
	assert.Contains(testSuite.T(), err.Error(), "cannot clear cache")
	
	// Release the block
	testSuite.cache.Release(block)
	
	// Now clear should succeed
	err = testSuite.cache.Clear()
	assert.Nil(testSuite.T(), err)
}

func (testSuite *BlockCacheTest) TestStats() {
	// Initially empty
	stats := testSuite.cache.Stats()
	assert.Equal(testSuite.T(), int64(0), stats.TotalBlocks)
	assert.Equal(testSuite.T(), int64(5), stats.MaxBlocks)
	assert.Equal(testSuite.T(), int64(0), stats.BlocksInUse)
	assert.Equal(testSuite.T(), int64(0), stats.BlocksAvailable)
	
	// Add blocks with different reference states
	block1, err := testSuite.cache.Get(CacheKey("block-1"))
	require.Nil(testSuite.T(), err)
	// Keep reference active
	
	block2, err := testSuite.cache.Get(CacheKey("block-2"))
	require.Nil(testSuite.T(), err)
	testSuite.cache.Release(block2) // Release reference
	
	stats = testSuite.cache.Stats()
	assert.Equal(testSuite.T(), int64(2), stats.TotalBlocks)
	assert.Equal(testSuite.T(), int64(1), stats.BlocksInUse)
	assert.Equal(testSuite.T(), int64(1), stats.BlocksAvailable)
	
	// Clean up
	testSuite.cache.Release(block1)
}

func (testSuite *BlockCacheTest) TestStatsString() {
	stats := testSuite.cache.Stats()
	str := stats.String()
	assert.Contains(testSuite.T(), str, "BlockCache{")
	assert.Contains(testSuite.T(), str, "total: 0")
	assert.Contains(testSuite.T(), str, "max: 5")
	assert.Contains(testSuite.T(), str, "inUse: 0")
	assert.Contains(testSuite.T(), str, "available: 0")
}

func (testSuite *BlockCacheTest) TestConcurrentAccess() {
	// This test verifies that the cache is thread-safe
	// We'll create multiple goroutines that access the cache concurrently
	
	const numGoroutines = 10
	const numOperations = 100
	
	done := make(chan bool, numGoroutines)
	
	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			defer func() { done <- true }()
			
			for i := 0; i < numOperations; i++ {
				key := CacheKey(fmt.Sprintf("block-%d-%d", goroutineID, i%5))
				
				block, err := testSuite.cache.Get(key)
				if err != nil {
					// May fail due to eviction policy, which is expected
					continue
				}
				
				// Write some data to the block
				data := []byte(fmt.Sprintf("data-%d-%d", goroutineID, i))
				block.Write(data)
				
				// Sometimes release immediately, sometimes keep reference
				if i%2 == 0 {
					testSuite.cache.Release(block)
				} else {
					// Release later
					go func(b *CachedBlock) {
						testSuite.cache.Release(b)
					}(block)
				}
			}
		}(g)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
	
	// Cache should still be functional
	stats := testSuite.cache.Stats()
	assert.True(testSuite.T(), stats.TotalBlocks >= 0)
	assert.True(testSuite.T(), stats.TotalBlocks <= stats.MaxBlocks)
}

func (testSuite *BlockCacheTest) TestDiskBlockCache() {
	// Test with disk blocks
	config := &BlockCacheConfig{
		BlockSize:          2048,
		MaxBlocks:          3,
		MaxBlocksPerFile:   10,
		ReservedBlocks:     1,
		BlockType:          DiskBlock,
		GlobalMaxBlocksSem: semaphore.NewWeighted(100),
	}
	
	diskCache, err := NewBlockCache(config)
	require.Nil(testSuite.T(), err)
	defer diskCache.Destroy()
	
	// Test basic functionality with disk blocks
	key := CacheKey("disk-block-1")
	block, err := diskCache.Get(key)
	require.Nil(testSuite.T(), err)
	
	// Write data to disk block
	testData := []byte("Hello, disk-based block cache!")
	n, err := block.Write(testData)
	assert.Nil(testSuite.T(), err)
	assert.Equal(testSuite.T(), len(testData), n)
	
	// Seek back to start before reading
	_, err = block.Seek(0, 0)
	assert.Nil(testSuite.T(), err)
	
	// Read data back
	readData := make([]byte, len(testData))
	n, err = block.Read(readData)
	// Accept EOF as valid when we've read all the data
	assert.True(testSuite.T(), err == nil || err == io.EOF, "expected nil or EOF, got: %v", err)
	assert.Equal(testSuite.T(), len(testData), n, "should read all data")
	assert.Equal(testSuite.T(), testData, readData)
	
	diskCache.Release(block)
}
