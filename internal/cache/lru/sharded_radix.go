// Copyright 2026 Google LLC
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

package lru

import (
	"math"
	"math/rand"
	"sort"
	"sync/atomic"
)

const defaultNumShards = 32

// ShardedRadixCache partitions key space across multiple radixCache shards
// to eliminate global RWMutex write contention and allow scalable parallel reads.
type ShardedRadixCache struct {
	shards            []*radixCache
	numShards         int
	mask              uint32
	totalMaxSize      uint64
	globalCurrentSize atomic.Int64
}

// NewShardedRadixCache returns an lru.Cache implementation backed by 32 sharded radix trie caches.
func NewShardedRadixCache(totalMaxSize uint64) Cache {
	return NewShardedRadixCacheWithCustomShards(totalMaxSize, defaultNumShards)
}

// NewShardedRadixCacheWithCustomShards creates a sharded radix cache with custom shard count.
func NewShardedRadixCacheWithCustomShards(totalMaxSize uint64, numShards int) *ShardedRadixCache {
	if totalMaxSize == 0 {
		panic("Invalid totalMaxSize")
	}
	if numShards <= 0 || (numShards&(numShards-1)) != 0 {
		panic("numShards must be a positive power of 2")
	}

	c := &ShardedRadixCache{
		shards:       make([]*radixCache, numShards),
		numShards:    numShards,
		mask:         uint32(numShards - 1),
		totalMaxSize: totalMaxSize,
	}

	for i := 0; i < numShards; i++ {
		c.shards[i] = newShardRadixCache(totalMaxSize, &c.globalCurrentSize)
	}

	return c
}

// pickRandomShards picks up to count distinct random shard indices and returns them sorted in ascending order.
func pickRandomShards(total, count int) []int {
	if count >= total {
		res := make([]int, total)
		for i := 0; i < total; i++ {
			res[i] = i
		}
		return res
	}
	res := make([]int, 0, count)
	for len(res) < count {
		r := rand.Intn(total)
		found := false
		for _, v := range res {
			if v == r {
				found = true
				break
			}
		}
		if !found {
			res = append(res, r)
		}
	}
	sort.Ints(res)
	return res
}

// fnv1a returns the 32-bit FNV-1a hash of string key.
func fnv1a(key string) uint32 {
	var hash uint32 = 2166136261
	for i := 0; i < len(key); i++ {
		hash ^= uint32(key[i])
		hash *= 16777619
	}
	return hash
}

func (c *ShardedRadixCache) getShard(key string) *radixCache {
	idx := fnv1a(key) & c.mask
	return c.shards[idx]
}

func (c *ShardedRadixCache) Insert(key string, value ValueType) ([]ValueType, error) {
	if value == nil {
		return nil, ErrInvalidEntry
	}

	valueSize := value.Size()
	if valueSize > c.totalMaxSize {
		return nil, ErrInvalidEntrySize
	}

	shard := c.getShard(key)
	shard.mu.Lock()

	shard.flushPromotions()

	// Dynamic calculation of pending size delta based on current presence of key in shard.
	// If existingNode self-evicts during local or global eviction, calcDelta() automatically
	// updates from (valueSize - oldSize) to valueSize.
	calcDelta := func() int64 {
		if node, ok := shard.getNode(key); ok && node.value != nil {
			return int64(valueSize) - int64(node.size)
		}
		return int64(valueSize)
	}

	var evictedValues []ValueType

	// 1. Local eviction in target shard using dynamic calcDelta()
	for c.globalCurrentSize.Load()+calcDelta() > int64(c.totalMaxSize) && shard.tail != nil {
		ev := shard.evictOne()
		if ev != nil {
			evictedValues = append(evictedValues, ev)
		}
	}

	// 2. Multi-shard fallback eviction across all shards using sampled Approximate LRU
	if c.globalCurrentSize.Load()+calcDelta() > int64(c.totalMaxSize) {
		shard.mu.Unlock()

		for c.globalCurrentSize.Load()+calcDelta() > int64(c.totalMaxSize) {
			indices := pickRandomShards(c.numShards, 3)
			for _, idx := range indices {
				c.shards[idx].mu.Lock()
			}

			var oldestIdx = -1
			var oldestTime int64 = math.MaxInt64

			for _, idx := range indices {
				c.shards[idx].flushPromotions()
				if c.shards[idx].tail != nil {
					t := c.shards[idx].tail.accessTime.Load()
					if t < oldestTime {
						oldestTime = t
						oldestIdx = idx
					}
				}
			}

			if oldestIdx != -1 {
				ev := c.shards[oldestIdx].evictOne()
				if ev != nil {
					evictedValues = append(evictedValues, ev)
				}
			}

			for i := len(indices) - 1; i >= 0; i-- {
				c.shards[indices[i]].mu.Unlock()
			}
		}

		shard.mu.Lock()
		evs, err := shard.insertNodeAndAdjust(key, value, valueSize)
		shard.mu.Unlock()

		if err == nil {
			evictedValues = append(evictedValues, evs...)
		}
		return evictedValues, err
	}

	evs, err := shard.insertNodeAndAdjust(key, value, valueSize)
	shard.mu.Unlock()

	evictedValues = append(evictedValues, evs...)
	return evictedValues, err
}

func (c *ShardedRadixCache) Erase(key string) ValueType {
	shard := c.getShard(key)
	return shard.Erase(key)
}

func (c *ShardedRadixCache) LookUp(key string) ValueType {
	shard := c.getShard(key)
	return shard.LookUp(key)
}

func (c *ShardedRadixCache) LookUpWithoutChangingOrder(key string) ValueType {
	shard := c.getShard(key)
	return shard.LookUpWithoutChangingOrder(key)
}

func (c *ShardedRadixCache) UpdateWithoutChangingOrder(key string, value ValueType) error {
	shard := c.getShard(key)
	return shard.UpdateWithoutChangingOrder(key, value)
}

func (c *ShardedRadixCache) UpdateSize(key string, sizeDelta uint64) error {
	shard := c.getShard(key)
	shard.mu.Lock()

	shard.flushPromotions()

	node, ok := shard.getNode(key)
	if !ok {
		shard.mu.Unlock()
		return ErrEntryNotExist
	}

	node.size += sizeDelta
	shard.currentSize += sizeDelta
	c.globalCurrentSize.Add(int64(sizeDelta))

	// 1. Local eviction in target shard
	for c.globalCurrentSize.Load() > int64(c.totalMaxSize) && shard.tail != nil {
		shard.evictOne()
	}

	// 2. Multi-shard eviction fallback across all shards using sampled Approximate LRU
	if c.globalCurrentSize.Load() > int64(c.totalMaxSize) {
		shard.mu.Unlock()

		for c.globalCurrentSize.Load() > int64(c.totalMaxSize) {
			indices := pickRandomShards(c.numShards, 3)
			for _, idx := range indices {
				c.shards[idx].mu.Lock()
			}

			var oldestIdx = -1
			var oldestTime int64 = math.MaxInt64

			for _, idx := range indices {
				c.shards[idx].flushPromotions()
				if c.shards[idx].tail != nil {
					t := c.shards[idx].tail.accessTime.Load()
					if t < oldestTime {
						oldestTime = t
						oldestIdx = idx
					}
				}
			}

			if oldestIdx != -1 {
				c.shards[oldestIdx].evictOne()
			}

			for i := len(indices) - 1; i >= 0; i-- {
				c.shards[indices[i]].mu.Unlock()
			}
		}
		return nil
	}

	shard.mu.Unlock()
	return nil
}

func (c *ShardedRadixCache) EraseEntriesWithGivenPrefix(prefix string) {
	// Lock all shards in strictly ascending index order (0..31) to guarantee zero deadlocks
	for i := 0; i < c.numShards; i++ {
		c.shards[i].mu.Lock()
	}
	defer func() {
		// Unlock all shards in descending index order (31..0)
		for i := c.numShards - 1; i >= 0; i-- {
			c.shards[i].mu.Unlock()
		}
	}()

	for i := 0; i < c.numShards; i++ {
		c.shards[i].flushPromotions()
		c.shards[i].eraseEntriesWithGivenPrefixInternal(prefix)
	}
}

func (c *ShardedRadixCache) Close() {
	for i := 0; i < c.numShards; i++ {
		c.shards[i].Close()
	}
}
