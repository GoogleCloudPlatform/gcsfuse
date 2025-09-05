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
	"log"

	"golang.org/x/sync/semaphore"
)

// ExampleUsage demonstrates how to use the disk-based block cache.
// This is not a test but an example for documentation purposes.
func ExampleUsage() {
	// Create a global semaphore for limiting total blocks across all pools
	globalSem := semaphore.NewWeighted(20) // Allow up to 20 blocks globally

	// Create block cache configuration for disk-based blocks
	config := &BlockCacheConfig{
		BlockSize:          1024 * 1024, // 1MB blocks
		MaxBlocks:          5,           // Cache up to 5 blocks
		MaxBlocksPerFile:   10,          // Max 10 blocks per file
		ReservedBlocks:     0,           // No reserved blocks
		BlockType:          DiskBlock,   // Use disk-based blocks
		GlobalMaxBlocksSem: globalSem,
	}

	// Create the block cache
	cache, err := NewBlockCache(config)
	if err != nil {
		log.Fatalf("Failed to create block cache: %v", err)
	}

	// Use the cache
	key := CacheKey("example-block-1")
	
	// Get a block from cache (will create new if not exists)
	cachedBlock, err := cache.Get(key)
	if err != nil {
		log.Fatalf("Failed to get block: %v", err)
	}

	// Use the block
	data := []byte("Hello, disk-based block cache!")
	_, err = cachedBlock.Write(data)
	if err != nil {
		log.Fatalf("Failed to write to block: %v", err)
	}

	// Important: Always release blocks when done
	cache.Release(cachedBlock)

	// Get the block again (should be from cache)
	cachedBlock2, err := cache.Get(key)
	if err != nil {
		log.Fatalf("Failed to get block again: %v", err)
	}

	// Read the data back
	_, err = cachedBlock2.Seek(0, 0) // Seek to beginning
	if err != nil {
		log.Fatalf("Failed to seek: %v", err)
	}

	readData := make([]byte, len(data))
	_, err = cachedBlock2.Read(readData)
	if err != nil && err.Error() != "EOF" {
		log.Fatalf("Failed to read: %v", err)
	}

	fmt.Printf("Read data: %s\n", string(readData))

	// Release the block
	cache.Release(cachedBlock2)

	// Print cache statistics
	stats := cache.Stats()
	fmt.Printf("Cache stats: %s\n", stats.String())

	// Clean up
	cache.Clear()
}
