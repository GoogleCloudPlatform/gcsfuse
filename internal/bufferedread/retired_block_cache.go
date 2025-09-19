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

package bufferedread

import (
	"fmt"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
)

// RetiredBlockCache defines the interface for a cache of retired blocks.
// This allows for different cache implementations to be used with the BufferedReader.
type RetiredBlockCache interface {
	// Insert adds a block to the cache. It may evict older entries if the cache
	// is full. It returns the evicted entries.
	Insert(blockIndex int64, entry *blockQueueEntry) (evicted []*blockQueueEntry, err error)

	// LookUp retrieves a block from the cache, marking it as recently used.
	LookUp(blockIndex int64) *blockQueueEntry

	// Erase removes a block from the cache.
	Erase(blockIndex int64)

	// Len returns the number of blocks currently in the cache.
	Len() int

	// Clear removes all entries from the cache and returns them.
	Clear() []*blockQueueEntry
}

// lruRetiredBlockCache is an implementation of RetiredBlockCache using an LRU cache.
type lruRetiredBlockCache struct {
	lruCache *lru.Cache
}

// NewLruRetiredBlockCache creates a new lruRetiredBlockCache with the given capacity.
func NewLruRetiredBlockCache(capacity uint64) RetiredBlockCache {
	return &lruRetiredBlockCache{
		lruCache: lru.NewCache(capacity),
	}
}

func (l *lruRetiredBlockCache) toKey(blockIndex int64) string {
	return fmt.Sprintf("%d", blockIndex)
}

func (l *lruRetiredBlockCache) Insert(blockIndex int64, entry *blockQueueEntry) ([]*blockQueueEntry, error) {
	key := l.toKey(blockIndex)
	evicted, err := l.lruCache.Insert(key, entry)
	if err != nil {
		return nil, err
	}

	evictedEntries := make([]*blockQueueEntry, len(evicted))
	for i, v := range evicted {
		evictedEntries[i] = v.(*blockQueueEntry)
	}
	return evictedEntries, nil
}

func (l *lruRetiredBlockCache) LookUp(blockIndex int64) *blockQueueEntry {
	key := l.toKey(blockIndex)
	val := l.lruCache.LookUp(key)
	if val == nil {
		return nil
	}
	return val.(*blockQueueEntry)
}

func (l *lruRetiredBlockCache) Erase(blockIndex int64) {
	key := l.toKey(blockIndex)
	l.lruCache.Erase(key)
}

func (l *lruRetiredBlockCache) Len() int {
	return l.lruCache.Len()
}

func (l *lruRetiredBlockCache) Clear() []*blockQueueEntry {
	evicted := l.lruCache.Clear()
	evictedEntries := make([]*blockQueueEntry, len(evicted))
	for i, v := range evicted {
		evictedEntries[i] = v.(*blockQueueEntry)
	}
	return evictedEntries
}
