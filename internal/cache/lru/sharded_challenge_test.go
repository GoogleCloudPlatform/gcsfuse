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
	"fmt"
	"hash/fnv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type dummyValue struct {
	size uint64
}

func (v dummyValue) Size() uint64 {
	return v.size
}

// -----------------------------------------------------------------------------
// Prototype of Proposed Candidate B1 + D (Hybrid Sharded Radix Trie with Async Promo)
// -----------------------------------------------------------------------------

type prototypeShard struct {
	mu        sync.RWMutex
	trie      *radixCache
	promoChan chan *radixNode
	closed    chan struct{}
}

type PrototypeShardedRadixCache struct {
	shards    []*prototypeShard
	numShards int
	mask      uint32
}

func NewPrototypeShardedRadixCache(maxSize uint64, numShards int, promoBufSize int) *PrototypeShardedRadixCache {
	if numShards <= 0 || (numShards&(numShards-1)) != 0 {
		panic("numShards must be a power of 2")
	}

	shardMaxSize := maxSize / uint64(numShards)
	if shardMaxSize == 0 {
		shardMaxSize = 1
	}

	c := &PrototypeShardedRadixCache{
		shards:    make([]*prototypeShard, numShards),
		numShards: numShards,
		mask:      uint32(numShards - 1),
	}

	for i := 0; i < numShards; i++ {
		rawCache := NewRadixCache(shardMaxSize).(*radixCache)
		s := &prototypeShard{
			trie:      rawCache,
			promoChan: make(chan *radixNode, promoBufSize),
			closed:    make(chan struct{}),
		}
		c.shards[i] = s
		go s.runPromoWorker()
	}

	return c
}

func (s *prototypeShard) runPromoWorker() {
	for {
		select {
		case <-s.closed:
			return
		case node, ok := <-s.promoChan:
			if !ok {
				return
			}
			s.mu.Lock()
			// Proposed behavior: process promotion under write lock
			if node != nil && node.value != nil {
				s.trie.moveToFront(node)
			}
			s.mu.Unlock()
		}
	}
}

func (c *PrototypeShardedRadixCache) Close() {
	for _, s := range c.shards {
		close(s.closed)
	}
}

func (c *PrototypeShardedRadixCache) getShard(key string) *prototypeShard {
	h := fnv.New32a()
	h.Write([]byte(key))
	shardIdx := h.Sum32() & c.mask
	return c.shards[shardIdx]
}

func (c *PrototypeShardedRadixCache) Insert(key string, value ValueType) ([]ValueType, error) {
	s := c.getShard(key)
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.trie.Insert(key, value)
}

func (c *PrototypeShardedRadixCache) LookUp(key string) ValueType {
	s := c.getShard(key)
	s.mu.RLock()
	node, ok := s.trie.getNode(key)
	if !ok {
		s.mu.RUnlock()
		return nil
	}
	val := node.value
	// Candidate D read path: non-blocking push to promotion channel
	select {
	case s.promoChan <- node:
	default:
		// Promotion dropped when channel full!
	}
	s.mu.RUnlock()
	return val
}

func (c *PrototypeShardedRadixCache) EraseEntriesWithGivenPrefix(prefix string) {
	// Candidate B1: Broadcast across all shards
	var wg sync.WaitGroup
	wg.Add(c.numShards)
	for i := 0; i < c.numShards; i++ {
		s := c.shards[i]
		go func(shard *prototypeShard) {
			defer wg.Done()
			shard.mu.Lock()
			shard.trie.EraseEntriesWithGivenPrefix(prefix)
			shard.mu.Unlock()
		}(s)
	}
	wg.Wait()
}

// -----------------------------------------------------------------------------
// Empirical Challenge 1: Broadcast Overhead for Prefix Erasure
// -----------------------------------------------------------------------------
func TestChallenge1_PrefixErasureBroadcastOverhead(t *testing.T) {
	const totalKeys = 10000
	const numShards = 32
	const prefix = "dir1/subdir/"

	// 1. Unified Radix Trie
	unifiedCache := NewRadixCache(1000000)
	for i := 0; i < totalKeys; i++ {
		key := fmt.Sprintf("dir1/subdir/file_%d.txt", i)
		_, _ = unifiedCache.Insert(key, dummyValue{size: 10})
	}
	// Add some unmatching keys
	for i := 0; i < totalKeys; i++ {
		key := fmt.Sprintf("dir2/otherdir/file_%d.txt", i)
		_, _ = unifiedCache.Insert(key, dummyValue{size: 10})
	}

	startUnified := time.Now()
	for i := 0; i < 100; i++ {
		unifiedCache.EraseEntriesWithGivenPrefix(prefix)
	}
	durationUnified := time.Since(startUnified)

	// 2. Prototype Sharded Radix Trie (32 shards)
	shardedCache := NewPrototypeShardedRadixCache(1000000, numShards, 100)
	defer shardedCache.Close()
	for i := 0; i < totalKeys; i++ {
		key := fmt.Sprintf("dir1/subdir/file_%d.txt", i)
		_, _ = shardedCache.Insert(key, dummyValue{size: 10})
	}
	for i := 0; i < totalKeys; i++ {
		key := fmt.Sprintf("dir2/otherdir/file_%d.txt", i)
		_, _ = shardedCache.Insert(key, dummyValue{size: 10})
	}

	startSharded := time.Now()
	for i := 0; i < 100; i++ {
		shardedCache.EraseEntriesWithGivenPrefix(prefix)
	}
	durationSharded := time.Since(startSharded)

	t.Logf("[PREFIX BROADCAST OVERHEAD RESULT]")
	t.Logf("Unified Trie 100x Prefix Erase: %v", durationUnified)
	t.Logf("Sharded Trie (32 Shards) 100x Prefix Erase: %v", durationSharded)
	t.Logf("Slowdown ratio: %.2fx", float64(durationSharded.Nanoseconds())/float64(durationUnified.Nanoseconds()))
}

// -----------------------------------------------------------------------------
// Empirical Challenge 2: Promotion Channel Race Condition & LRU Invariant Violation
// -----------------------------------------------------------------------------
func TestChallenge2_PromoChanRaceWithErase(t *testing.T) {
	// Tests race condition where promoChan holds a node reference while Erase deletes it
	s := &prototypeShard{
		trie:      NewRadixCache(10000).(*radixCache),
		promoChan: make(chan *radixNode, 100),
		closed:    make(chan struct{}),
	}

	// Insert key "a/b/c"
	node, _ := s.trie.insertNode("a/b/c", dummyValue{size: 10})
	s.trie.pushFront(node)
	s.trie.currentSize += 10

	// Step 1: LookUp buffers node pointer into promoChan
	s.promoChan <- node

	// Step 2: Erase removes node from tree & LRU list
	s.trie.Erase("a/b/c")

	// Step 3: Insert new value under key "a/b/c" -> creates a NEW radixNode node2
	s.trie.Insert("a/b/c", dummyValue{size: 15})

	// Step 4: Naive promo worker drains promoChan and calls moveToFront(node) on stale node
	// Note: Even if worker checks node.value != nil, what if node was pooled or value assigned?
	// Here we test what happens if stale node is processed by moveToFront:
	s.mu.Lock()
	staleNode := <-s.promoChan
	// Processing stale node reference without checking if it's still attached to tree:
	s.trie.moveToFront(staleNode)
	s.mu.Unlock()

	// Verify Invariants!
	s.mu.Lock()
	defer s.mu.Unlock()

	var panicMsg interface{}
	func() {
		defer func() { panicMsg = recover() }()
		s.trie.checkInvariants()
	}()

	if panicMsg != nil {
		t.Logf("[CONFIRMED LRU CORRUPTION BUG] checkInvariants PANICKED: %v", panicMsg)
	} else {
		t.Logf("Unexpected: checkInvariants passed")
	}
}

// -----------------------------------------------------------------------------
// Empirical Challenge 4: Memory Pinning of Unlinked Subtrees
// -----------------------------------------------------------------------------
func TestChallenge4_MemoryPinningOfUnlinkedSubtrees(t *testing.T) {
	c := NewRadixCache(1000000).(*radixCache)

	// Build a deep tree under prefix "dir1/"
	for i := 0; i < 100; i++ {
		_, _ = c.Insert(fmt.Sprintf("dir1/subdir/nested/file_%d.txt", i), dummyValue{size: 10})
	}

	// Find a deep leaf node
	node, ok := c.getNode("dir1/subdir/nested/file_50.txt")
	if !ok {
		t.Fatalf("Node not found")
	}

	// Now erase prefix "dir1/"
	c.EraseEntriesWithGivenPrefix("dir1/")

	// Verify detached node state:
	// sweepAndUnlink cleared node.value and unlinked from LRU list,
	// BUT parent/child/sibling pointers inside the detached subtree remain intact!
	if node.parent != nil {
		t.Logf("[CONFIRMED MEMORY LEAK RISK] Detached leaf node still holds pointer to parent: prefix='%s', parent.prefix='%s'", node.prefix, node.parent.prefix)
	}

	// Walk up parent chain from leaf node
	curr := node
	ancestorCount := 0
	for curr.parent != nil {
		ancestorCount++
		curr = curr.parent
	}
	t.Logf("[MEMORY LEAK RISK] Retaining reference to 1 detached leaf node pins %d ancestor nodes in heap memory!", ancestorCount)
}


// -----------------------------------------------------------------------------
// Empirical Challenge 3: Exact vs Approximate LRU Eviction Pathology (Hot Data Thrashing)
// -----------------------------------------------------------------------------
func TestChallenge3_EvictionPathologyUnderHighReadLoad(t *testing.T) {
	// Cache maxSize = 100 bytes (fits 10 entries of size 10)
	// Sharded cache with 1 shard to isolate channel drop logic
	cache := NewPrototypeShardedRadixCache(100, 1, 4) // Tiny promotion channel (size 4)
	defer cache.Close()

	// Insert 8 hot items
	for i := 1; i <= 8; i++ {
		key := fmt.Sprintf("hot_key_%d", i)
		_, _ = cache.Insert(key, dummyValue{size: 10})
	}

	// High concurrency read on hot items (fills promotion channel & drops subsequent promotions)
	var wg sync.WaitGroup
	stopReads := make(chan struct{})
	var droppedPromotions uint64

	for r := 0; r < 10; r++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-stopReads:
					return
				default:
					key := fmt.Sprintf("hot_key_%d", (id%8)+1)
					s := cache.getShard(key)
					s.mu.RLock()
					node, ok := s.trie.getNode(key)
					if ok {
						select {
						case s.promoChan <- node:
						default:
							atomic.AddUint64(&droppedPromotions, 1)
						}
					}
					s.mu.RUnlock()
				}
			}
		}(r)
	}

	time.Sleep(20 * time.Millisecond)

	// Interleave sequential cold inserts that exceed cache size
	for i := 1; i <= 5; i++ {
		coldKey := fmt.Sprintf("cold_key_%d", i)
		_, _ = cache.Insert(coldKey, dummyValue{size: 10})
	}

	close(stopReads)
	wg.Wait()

	t.Logf("Total dropped promotions: %d", atomic.LoadUint64(&droppedPromotions))

	// Check if hot items were evicted!
	hotEvictedCount := 0
	for i := 1; i <= 8; i++ {
		key := fmt.Sprintf("hot_key_%d", i)
		if cache.LookUp(key) == nil {
			hotEvictedCount++
			t.Logf("HOT ITEM EVICTED: %s", key)
		}
	}

	t.Logf("[EVICTION PATHOLOGY RESULT] Hot items evicted: %d out of 8", hotEvictedCount)
	if hotEvictedCount > 0 {
		t.Logf("[CONFIRMED EVICTION PATHOLOGY] Hot data was evicted due to dropped promotion requests under read load!")
	}
}
