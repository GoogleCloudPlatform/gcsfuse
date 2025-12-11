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
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/folio"
)

func createTestCache() *LRUCache {
	pool, _ := folio.NewSmartPool(int(folio.Size1MB), int(folio.Size64KB))
	return NewLRUCache(LRUCacheConfig{
		MaxSize:    10 * 1024 * 1024, // 10MB
		MaxEntries: 100,
		SmartPool:  pool,
	})
}

func TestLRUCache_Basic(t *testing.T) {
	cache := createTestCache()

	// Get entry - will create folio automatically
	result, err := cache.Get(1, 0, 4096)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("Expected 1 result, got %d", len(result))
	}

	// Check stats
	stats := cache.Stats()
	if stats.EntryCount != 1 {
		t.Errorf("Expected 1 entry, got %d", stats.EntryCount)
	}
	if stats.CurrentSize != 4096 {
		t.Errorf("Expected size 4096, got %d", stats.CurrentSize)
	}
}

func TestLRUCache_PageAlignment(t *testing.T) {
	cache := createTestCache()

	// Get creates aligned entries automatically
	result, err := cache.Get(1, 0, 4096)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("Expected 1 entry at aligned offset, got %d", len(result))
	}

	// Query larger range - should reuse first entry and create new one
	result, err = cache.Get(1, 0, 8192)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 entries with aligned query, got %d", len(result))
	}

	// Both entries should now be cached
	stats := cache.Stats()
	if stats.EntryCount != 2 {
		t.Errorf("Expected 2 cached entries, got %d", stats.EntryCount)
	}
}

func TestLRUCache_RangeQuery(t *testing.T) {
	cache := createTestCache()

	// Create folios via Get - allocates based on SmartPool block sizes
	result1, err := cache.Get(1, 0, 10*PageSize)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	initialFolios := len(result1)
	stats1 := cache.Stats()
	initialEntryCount := stats1.EntryCount

	// Query the same range - should reuse existing folios without creating new ones
	result, err := cache.Get(1, 0, 10*PageSize)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	// Should return same folios
	if len(result) != initialFolios {
		t.Errorf("Expected %d folios (same as initial), got %d", initialFolios, len(result))
	}
	// Should not create new entries
	stats := cache.Stats()
	if stats.EntryCount != initialEntryCount {
		t.Errorf("Expected %d entries (no new allocations), got %d", initialEntryCount, stats.EntryCount)
	}
}

func TestLRUCache_LRUEviction(t *testing.T) {
	pool, _ := folio.NewSmartPool(int(folio.Size1MB), int(folio.Size64KB))
	cache := NewLRUCache(LRUCacheConfig{
		MaxEntries: 5,
		SmartPool:  pool,
	})

	// Create 10 entries via individual Gets
	for i := 0; i < 10; i++ {
		offset := int64(i * PageSize)
		_, err := cache.Get(1, offset, PageSize)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
	}

	// Should have evicted the first 5 entries
	stats := cache.Stats()
	if stats.EntryCount != 5 {
		t.Errorf("Expected 5 entries after eviction, got %d", stats.EntryCount)
	}

	// First entries should be evicted, but Get will create a new folio
	result, err := cache.Get(1, 0, PageSize)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("Expected Get to create new folio for evicted entry, got %d", len(result))
	}

	// Last entries should still be there
	result, err = cache.Get(1, 9*PageSize, PageSize)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(result) != 1 {
		t.Error("Last entry should still be in cache")
	}
}

func TestLRUCache_RemoveInode(t *testing.T) {
	cache := createTestCache()

	// Create entries for multiple inodes via Get
	for inode := uint64(1); inode <= 3; inode++ {
		_, err := cache.Get(inode, 0, 5*PageSize)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
	}

	// Remove all entries for inode 2
	cache.RemoveInode(2)

	stats := cache.Stats()
	entryCountAfterRemove := stats.EntryCount
	if stats.InodeCount != 2 {
		t.Errorf("Expected 2 inodes, got %d", stats.InodeCount)
	}

	// Inode 2 entries should be gone, but Get will create new folios for the range
	result, err := cache.Get(2, 0, 10*PageSize)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(result) < 1 {
		t.Errorf("Expected at least 1 folio for removed inode, got %d", len(result))
	}
	// Verify new entries were created
	stats = cache.Stats()
	if stats.EntryCount <= entryCountAfterRemove {
		t.Errorf("Expected entry count to increase from %d after new allocation, got %d", entryCountAfterRemove, stats.EntryCount)
	}
}

func TestLRUCache_InvalidateInode(t *testing.T) {
	cache := createTestCache()

	// Create entries for multiple inodes via Get
	for inode := uint64(1); inode <= 3; inode++ {
		_, err := cache.Get(inode, 0, 5*PageSize)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
	}

	// Invalidate inode 2
	cache.InvalidateInode(2)

	stats := cache.Stats()
	entryCountAfterInvalidate := stats.EntryCount
	if stats.InodeCount != 2 {
		t.Errorf("Expected 2 inodes, got %d", stats.InodeCount)
	}

	// Inode 2 entries should be invalidated, but Get will create new folios
	result, err := cache.Get(2, 0, 10*PageSize)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(result) < 1 {
		t.Errorf("Expected at least 1 folio for invalidated inode, got %d", len(result))
	}

	// Other inodes should still exist with their original entries
	result1, err := cache.Get(1, 0, 10*PageSize)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(result1) < 1 {
		t.Errorf("Inode 1 should have at least 1 folio, got %d", len(result1))
	}

	result3, err := cache.Get(3, 0, 10*PageSize)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(result3) < 1 {
		t.Errorf("Inode 3 should have at least 1 folio, got %d", len(result3))
	}

	// Verify new entries were created for inode 2
	stats = cache.Stats()
	if stats.EntryCount <= entryCountAfterInvalidate {
		t.Errorf("Expected entry count to increase from %d after new allocation, got %d", entryCountAfterInvalidate, stats.EntryCount)
	}
}

func TestLRUCache_Clear(t *testing.T) {
	cache := createTestCache()

	// Create entries via Get
	_, err := cache.Get(1, 0, 10*PageSize)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Clear cache
	cache.Clear()

	stats := cache.Stats()
	if stats.EntryCount != 0 {
		t.Errorf("Expected 0 entries after clear, got %d", stats.EntryCount)
	}
}
