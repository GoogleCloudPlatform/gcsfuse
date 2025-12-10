// Copyright 2025 Google Inc. All Rights Reserved.

package folio

import (
	"testing"
)

func TestLRUCache_Basic(t *testing.T) {
	cache := NewLRUCache(LRUCacheConfig{
		MaxSize:    10 * 1024 * 1024, // 10MB
		MaxEntries: 100,
	})

	// Get entry - will create folio automatically
	result := cache.Get(1, 0, 4096)
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
	cache := NewLRUCache(LRUCacheConfig{})

	// Get creates aligned entries automatically
	result := cache.Get(1, 0, 4096)
	if len(result) != 1 {
		t.Errorf("Expected 1 entry at aligned offset, got %d", len(result))
	}

	// Query larger range - should reuse first entry and create new one
	result = cache.Get(1, 0, 8192)
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
	cache := NewLRUCache(LRUCacheConfig{})

	// Create folios via Get - creates 10 page-sized entries
	cache.Get(1, 0, 10*PageSize)

	// Query a range covering part of cached entries
	result := cache.Get(1, 2*PageSize, 5*PageSize)
	// Should get entries at offsets: 2*PageSize, 3*PageSize, 4*PageSize, 5*PageSize, 6*PageSize
	if len(result) != 5 {
		t.Errorf("Expected 5 entries in range, got %d", len(result))
	}
}

func TestLRUCache_LRUEviction(t *testing.T) {
	cache := NewLRUCache(LRUCacheConfig{
		MaxEntries: 5,
	})

	// Create 10 entries via individual Gets
	for i := 0; i < 10; i++ {
		offset := int64(i * PageSize)
		cache.Get(1, offset, PageSize)
	}

	// Should have evicted the first 5 entries
	stats := cache.Stats()
	if stats.EntryCount != 5 {
		t.Errorf("Expected 5 entries after eviction, got %d", stats.EntryCount)
	}

	// First entries should be evicted, but Get will create a new folio
	result := cache.Get(1, 0, PageSize)
	if len(result) != 1 {
		t.Errorf("Expected Get to create new folio for evicted entry, got %d", len(result))
	}

	// Last entries should still be there
	result = cache.Get(1, 9*PageSize, PageSize)
	if len(result) != 1 {
		t.Error("Last entry should still be in cache")
	}
}

func TestLRUCache_RemoveInode(t *testing.T) {
	cache := NewLRUCache(LRUCacheConfig{})

	// Create entries for multiple inodes via Get
	for inode := uint64(1); inode <= 3; inode++ {
		cache.Get(inode, 0, 5*PageSize)
	}

	// Remove all entries for inode 2
	cache.RemoveInode(2)

	stats := cache.Stats()
	if stats.EntryCount != 10 {
		t.Errorf("Expected 10 entries after removing inode, got %d", stats.EntryCount)
	}
	if stats.InodeCount != 2 {
		t.Errorf("Expected 2 inodes, got %d", stats.InodeCount)
	}

	// Inode 2 entries should be gone, but Get will create new folios for the range
	result := cache.Get(2, 0, 10*PageSize)
	if len(result) != 10 {
		t.Errorf("Expected Get to create new folios for removed entries, got %d", len(result))
	}
	// Verify they are newly created by checking cache stats increased
	stats = cache.Stats()
	if stats.EntryCount != 20 {
		t.Errorf("Expected 20 entries after Get creates new ones, got %d", stats.EntryCount)
	}
}

func TestLRUCache_InvalidateInode(t *testing.T) {
	cache := NewLRUCache(LRUCacheConfig{})

	// Create entries for multiple inodes via Get
	for inode := uint64(1); inode <= 3; inode++ {
		cache.Get(inode, 0, 5*PageSize)
	}

	// Invalidate inode 2
	cache.InvalidateInode(2)

	stats := cache.Stats()
	if stats.EntryCount != 10 {
		t.Errorf("Expected 10 entries after invalidating one inode, got %d", stats.EntryCount)
	}
	if stats.InodeCount != 2 {
		t.Errorf("Expected 2 inodes, got %d", stats.InodeCount)
	}

	// Inode 2 entries should be invalidated, but Get will create new folios
	result := cache.Get(2, 0, 10*PageSize)
	if len(result) != 10 {
		t.Errorf("Expected Get to create new folios for invalidated inode, got %d", len(result))
	}

	// Other inodes should still exist with their original entries (5 cached + 5 gaps filled)
	if len(cache.Get(1, 0, 10*PageSize)) != 10 {
		t.Errorf("Inode 1 should have 5 cached + 5 gap-filled entries, got %d", len(cache.Get(1, 0, 10*PageSize)))
	}
	if len(cache.Get(3, 0, 10*PageSize)) != 10 {
		t.Errorf("Inode 3 should have 5 cached + 5 gap-filled entries, got %d", len(cache.Get(3, 0, 10*PageSize)))
	}
}

func TestLRUCache_Clear(t *testing.T) {
	cache := NewLRUCache(LRUCacheConfig{})

	// Create entries via Get
	cache.Get(1, 0, 10*PageSize)

	// Clear cache
	cache.Clear()

	stats := cache.Stats()
	if stats.EntryCount != 0 {
		t.Errorf("Expected 0 entries after clear, got %d", stats.EntryCount)
	}
}
