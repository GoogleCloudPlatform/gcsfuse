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

	"github.com/google/btree"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/folio"
	"github.com/jacobsa/fuse/fuseops"
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
	smartPool   *folio.SmartPool
}

type LRUCacheConfig struct {
	MaxSize     int64
	MaxEntries  int
	BTreeDegree int
	SmartPool   *folio.SmartPool
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
		smartPool:   config.SmartPool,
	}
}

func (c *LRUCache) Get(inode uint64, offset, size int64) ([]*folio.Folio, error) {
	if size <= 0 {
		return nil, nil
	}

	if c.smartPool == nil {
		return nil, fmt.Errorf("SmartPool is required but not configured")
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
	var allocErr error

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

			// Allocate folios for the gap using SmartPool
			gapFolios, err := folio.AllocateFolios(start, gapEnd, fuseops.InodeID(inode), c.smartPool)
			if err != nil {
				allocErr = fmt.Errorf("failed to allocate folios for gap [%d, %d): %w", start, gapEnd, err)
				return false
			}

			// Add folios to result and create cache entries
			for _, f := range gapFolios {
				result = append(result, f)
				newEntry := &CacheEntry{
					Inode:  inode,
					Offset: f.Start,
					Size:   f.End - f.Start,
					Folio:  f,
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

	if allocErr != nil {
		return nil, allocErr
	}

	// Create final folios if there's still a gap at the end
	if start < end {
		finalFolios, err := folio.AllocateFolios(start, end, fuseops.InodeID(inode), c.smartPool)
		if err != nil {
			return nil, fmt.Errorf("failed to allocate folios for final gap [%d, %d): %w", start, end, err)
		}

		// Add folios to result and create cache entries
		for _, f := range finalFolios {
			result = append(result, f)
			newEntry := &CacheEntry{
				Inode:  inode,
				Offset: f.Start,
				Size:   f.End - f.Start,
				Folio:  f,
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

	return result, nil
}

func (c *LRUCache) Remove(inode uint64, offset, size int64) {
	alignedOffset := offset
	alignedEnd := offset + size

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
