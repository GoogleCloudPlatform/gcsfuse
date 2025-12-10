// Copyright 2025 Google Inc. All Rights Reserved.

package folio

import (
	"sync"

	"github.com/google/btree"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/folio"
)

const PageSize = 4096

type CacheEntry struct {
	Inode  uint64
	Offset int64
	Size   int64
	Folio  *folio.Folio
	prev   *CacheEntry
	next   *CacheEntry
}

func (e *CacheEntry) Less(than btree.Item) bool {
	other := than.(*CacheEntry)
	return e.Offset < other.Offset
}

type LRUCache struct {
	mu          sync.RWMutex
	inodeTrees  map[uint64]*btree.BTree
	head        *CacheEntry
	tail        *CacheEntry
	maxSize     int64
	currentSize int64
	maxEntries  int
	entryCount  int
	btreeDegree int
}

type LRUCacheConfig struct {
	MaxSize     int64
	MaxEntries  int
	BTreeDegree int
}

type CacheStats struct {
	EntryCount  int
	CurrentSize int64
	MaxSize     int64
	MaxEntries  int
	InodeCount  int
}

func NewLRUCache(config LRUCacheConfig) *LRUCache {
	if config.BTreeDegree <= 0 {
		config.BTreeDegree = 32
	}
	return &LRUCache{
		inodeTrees:  make(map[uint64]*btree.BTree),
		maxSize:     config.MaxSize,
		maxEntries:  config.MaxEntries,
		btreeDegree: config.BTreeDegree,
	}
}

func alignOffset(offset int64) int64 {
	return (offset / PageSize) * PageSize
}

func alignSize(size int64) int64 {
	return ((size + PageSize - 1) / PageSize) * PageSize
}

func (c *LRUCache) Get(inode uint64, offset, size int64) []*folio.Folio {
	if size <= 0 {
		return nil
	}

	// Assume offset and size are already page-aligned
	end := offset + size
	start := offset

	c.mu.Lock()
	defer c.mu.Unlock()

	tree := c.inodeTrees[inode]
	if tree == nil {
		tree = btree.New(c.btreeDegree)
		c.inodeTrees[inode] = tree
	}

	var result []*folio.Folio
	var newEntries []*CacheEntry

	// Iterate through existing entries starting from the requested offset
	tree.AscendGreaterOrEqual(&CacheEntry{Offset: start}, func(item btree.Item) bool {
		entry := item.(*CacheEntry)
		entryStart := entry.Offset
		entryEnd := entry.Offset + entry.Size

		if start < entryStart {
			// Create new folios to fill the gap until the next entry or end of range
			gapEnd := entryStart
			if gapEnd > end {
				gapEnd = end
			}

			// Break gap into page-sized chunks
			for currentOffset := start; currentOffset < gapEnd; currentOffset += PageSize {
				chunkEnd := currentOffset + PageSize
				if chunkEnd > gapEnd {
					chunkEnd = gapEnd
				}
				chunkSize := chunkEnd - currentOffset

				newFolio := folio.NewFolio(currentOffset, chunkEnd, nil)
				result = append(result, newFolio)

				newEntry := &CacheEntry{
					Inode:  inode,
					Offset: currentOffset,
					Size:   chunkSize,
					Folio:  newFolio,
				}
				newEntries = append(newEntries, newEntry)
			}
			start = gapEnd
		}

		if end > entryStart {
			// Reuse existing folio
			c.moveToFront(entry)
			result = append(result, entry.Folio)

			folioEnd := entryEnd
			if folioEnd > end {
				folioEnd = end
			}
			start = folioEnd
		}

		return start < end
	})

	// Create final folios if there's still a gap at the end
	if start < end {
		// Break remaining gap into page-sized chunks
		for currentOffset := start; currentOffset < end; currentOffset += PageSize {
			chunkEnd := currentOffset + PageSize
			if chunkEnd > end {
				chunkEnd = end
			}
			chunkSize := chunkEnd - currentOffset

			newFolio := folio.NewFolio(currentOffset, chunkEnd, nil)
			result = append(result, newFolio)

			newEntry := &CacheEntry{
				Inode:  inode,
				Offset: currentOffset,
				Size:   chunkSize,
				Folio:  newFolio,
			}
			newEntries = append(newEntries, newEntry)
		}
	}

	// Insert all new entries into the tree (must be done outside of Ascend)
	for _, entry := range newEntries {
		tree.ReplaceOrInsert(entry)
		c.addToFront(entry)
		c.currentSize += entry.Size
		c.entryCount++
	}

	c.evictIfNeeded()

	return result
}

func (c *LRUCache) Remove(inode uint64, offset, size int64) {
	alignedOffset := alignOffset(offset)
	alignedEnd := alignOffset(offset + size + PageSize - 1)

	c.mu.Lock()
	defer c.mu.Unlock()

	tree := c.inodeTrees[inode]
	if tree == nil {
		return
	}

	var toRemove []*CacheEntry
	tree.AscendGreaterOrEqual(&CacheEntry{Offset: alignedOffset}, func(item btree.Item) bool {
		entry := item.(*CacheEntry)
		if entry.Offset >= alignedEnd {
			return false
		}
		entryEnd := entry.Offset + entry.Size
		if entryEnd > alignedOffset {
			toRemove = append(toRemove, entry)
		}
		return true
	})

	for _, entry := range toRemove {
		tree.Delete(entry)
		c.removeFromList(entry)
		c.currentSize -= entry.Size
		c.entryCount--
	}

	if tree.Len() == 0 {
		delete(c.inodeTrees, inode)
	}
}

func (c *LRUCache) RemoveInode(inode uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	tree := c.inodeTrees[inode]
	if tree == nil {
		return
	}

	tree.Ascend(func(item btree.Item) bool {
		entry := item.(*CacheEntry)
		c.removeFromList(entry)
		c.currentSize -= entry.Size
		c.entryCount--
		return true
	})

	delete(c.inodeTrees, inode)
}

// InvalidateInode removes all cached entries for the specified inode.
// This is typically called when file contents change and cached data becomes stale.
func (c *LRUCache) InvalidateInode(inode uint64) {
	c.RemoveInode(inode)
}

func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.inodeTrees = make(map[uint64]*btree.BTree)
	c.head = nil
	c.tail = nil
	c.currentSize = 0
	c.entryCount = 0
}

func (c *LRUCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return CacheStats{
		EntryCount:  c.entryCount,
		CurrentSize: c.currentSize,
		MaxSize:     c.maxSize,
		MaxEntries:  c.maxEntries,
		InodeCount:  len(c.inodeTrees),
	}
}

func (c *LRUCache) moveToFront(entry *CacheEntry) {
	if entry == c.head {
		return
	}
	c.removeFromList(entry)
	c.addToFront(entry)
}

func (c *LRUCache) addToFront(entry *CacheEntry) {
	entry.next = c.head
	entry.prev = nil
	if c.head != nil {
		c.head.prev = entry
	}
	c.head = entry
	if c.tail == nil {
		c.tail = entry
	}
}

func (c *LRUCache) removeFromList(entry *CacheEntry) {
	if entry.prev != nil {
		entry.prev.next = entry.next
	} else {
		c.head = entry.next
	}
	if entry.next != nil {
		entry.next.prev = entry.prev
	} else {
		c.tail = entry.prev
	}
	entry.prev = nil
	entry.next = nil
}

func (c *LRUCache) evictIfNeeded() {
	for c.maxEntries > 0 && c.entryCount > c.maxEntries {
		c.evictTail()
	}
	for c.maxSize > 0 && c.currentSize > c.maxSize {
		c.evictTail()
	}
}

func (c *LRUCache) evictTail() {
	if c.tail == nil {
		return
	}
	entry := c.tail
	if tree := c.inodeTrees[entry.Inode]; tree != nil {
		tree.Delete(entry)
		if tree.Len() == 0 {
			delete(c.inodeTrees, entry.Inode)
		}
	}
	c.removeFromList(entry)
	c.currentSize -= entry.Size
	c.entryCount--
}
