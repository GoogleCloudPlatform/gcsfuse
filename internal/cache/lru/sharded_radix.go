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
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// valueWrapper holds interface values for safe use with atomic.Pointer.
type valueWrapper struct {
	val ValueType
}

// shardedRadixNode represents an RCU-compatible LCRS radix tree node with SIEVE eviction.
type shardedRadixNode struct {
	prefix string // Read-only after publishing to tree; cloned string

	// RCU atomic pointers for tree routing and value
	child   atomic.Pointer[shardedRadixNode]
	sibling atomic.Pointer[shardedRadixNode]
	value   atomic.Pointer[valueWrapper]

	// Protected by shard mutex during modifications
	parent *shardedRadixNode

	// SIEVE eviction fields
	accessed  atomic.Bool
	sievePrev *shardedRadixNode
	sieveNext *shardedRadixNode
}

func (n *shardedRadixNode) getValue() ValueType {
	w := n.value.Load()
	if w == nil {
		return nil
	}
	return w.val
}

func (n *shardedRadixNode) setValue(val ValueType) {
	if val == nil {
		n.value.Store(nil)
	} else {
		n.value.Store(&valueWrapper{val: val})
	}
}

// getChild finds a child node whose prefix starts with the given byte (RCU read without lock).
func (n *shardedRadixNode) getChild(b byte) *shardedRadixNode {
	for curr := n.child.Load(); curr != nil; curr = curr.sibling.Load() {
		if len(curr.prefix) > 0 && curr.prefix[0] == b {
			return curr
		}
		// Sibling chains are lexicographically sorted by prefix[0], allowing early exit.
		if len(curr.prefix) > 0 && curr.prefix[0] > b {
			return nil
		}
	}
	return nil
}

type detachedSubtree struct {
	root *shardedRadixNode
	curr *shardedRadixNode
}

// cacheShard represents a single shard padded to exactly 128 bytes to eliminate false sharing.
type cacheShard struct {
	mu          sync.Mutex
	maxSize     uint64
	currentSize uint64
	len         int

	root atomic.Pointer[shardedRadixNode]

	// SIEVE clock eviction pointers
	sieveHead *shardedRadixNode
	sieveTail *shardedRadixNode
	sieveHand *shardedRadixNode

	// Background sweep queue for detached subtrees
	detachedQueue []detachedSubtree

	// Padding to ensure total struct size is exactly 128 bytes.
	// 8 (mu) + 8 (maxSize) + 8 (currentSize) + 8 (len) + 8 (root) +
	// 8 (sieveHead) + 8 (sieveTail) + 8 (sieveHand) + 24 (detachedQueue slice header) = 88 bytes.
	// 128 - 88 = 40 bytes of padding.
	_pad [40]byte
}

// Compile-time assertion to guarantee exact 128-byte shard padding.
var _ = [1]struct{}{}[128-unsafe.Sizeof(cacheShard{})]

// ShardedRadixCache implements lru.Cache with 256-way sharding, RCU lookups, and SIEVE eviction.
type ShardedRadixCache struct {
	shards        [256]cacheShard
	maxSize       uint64
	currentSize   atomic.Int64
	evictShardIdx atomic.Uint32

	closeCh       chan struct{}
	sweepNotifyCh chan struct{}
	wg            sync.WaitGroup
}

// Compile-time assertion that ShardedRadixCache implements Cache.
var _ Cache = (*ShardedRadixCache)(nil)

// NewShardedRadixCache returns an initialized sharded radix cache.
func NewShardedRadixCache(maxSize uint64) Cache {
	if maxSize == 0 {
		panic("Invalid maxSize")
	}

	c := &ShardedRadixCache{
		maxSize:       maxSize,
		closeCh:       make(chan struct{}),
		sweepNotifyCh: make(chan struct{}, 1),
	}

	baseShardSize := maxSize / 256
	remainder := maxSize % 256

	for i := 0; i < 256; i++ {
		shardSize := baseShardSize
		if uint64(i) < remainder {
			shardSize++
		}
		c.shards[i].maxSize = shardSize
		c.shards[i].root.Store(&shardedRadixNode{})
	}

	c.wg.Add(1)
	go c.backgroundWorker()

	return c
}

// GetShardSize returns the size of cacheShard struct in bytes for verification.
func GetShardSize() uintptr {
	return unsafe.Sizeof(cacheShard{})
}

func (c *ShardedRadixCache) getShard(key string) *cacheShard {
	s, _ := c.getShardWithIdx(key)
	return s
}

func (c *ShardedRadixCache) getShardWithIdx(key string) (*cacheShard, int) {
	hash := uint32(2166136261)
	for i := 0; i < len(key); i++ {
		hash ^= uint32(key[i])
		hash *= 16777619
	}
	idx := int(hash & 0xFF)
	return &c.shards[idx], idx
}

func (c *ShardedRadixCache) notifySweep() {
	select {
	case c.sweepNotifyCh <- struct{}{}:
	default:
	}
}

func (c *ShardedRadixCache) backgroundWorker() {
	defer c.wg.Done()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-c.closeCh:
			c.drainAll()
			return
		case <-c.sweepNotifyCh:
			c.sweepAllBatches(64)
		case <-ticker.C:
			c.sweepAllBatches(64)
			c.PruneAllEmptyLeaves()
		}
	}
}

func (c *ShardedRadixCache) sweepAllBatches(maxNodes int) {
	for i := 0; i < 256; i++ {
		c.shards[i].processDetachedSubtreesBatch(maxNodes, c)
	}
}

func (c *ShardedRadixCache) drainAll() {
	for i := 0; i < 256; i++ {
		c.shards[i].processDetachedSubtreesBatch(math.MaxInt, c)
		c.shards[i].pruneAllEmptyLeaves()
	}
}

// Close gracefully stops background sweeping and pruning routines.
func (c *ShardedRadixCache) Close() error {
	close(c.closeCh)
	c.wg.Wait()
	return nil
}

// PruneAllEmptyLeaves prunes empty routing leaf nodes across all shards.
func (c *ShardedRadixCache) PruneAllEmptyLeaves() {
	for i := 0; i < 256; i++ {
		c.shards[i].pruneAllEmptyLeaves()
	}
}

// --- Tree mutation helpers (must be called with shard.mu held) ---

func (s *cacheShard) addChildAtomic(parent, newChild *shardedRadixNode) {
	newChild.parent = parent
	pcurr := &parent.child
	for pcurr.Load() != nil && pcurr.Load().prefix[0] < newChild.prefix[0] {
		pcurr = &pcurr.Load().sibling
	}
	newChild.sibling.Store(pcurr.Load())
	pcurr.Store(newChild)
}

func (s *cacheShard) removeChildAtomic(parent, childToRemove *shardedRadixNode) {
	for pcurr := &parent.child; pcurr.Load() != nil; pcurr = &pcurr.Load().sibling {
		if pcurr.Load() == childToRemove {
			pcurr.Store(childToRemove.sibling.Load())
			return
		}
	}
}

func (s *cacheShard) replaceChildAtomic(parent, oldChild, newChild *shardedRadixNode) {
	for pcurr := &parent.child; pcurr.Load() != nil; pcurr = &pcurr.Load().sibling {
		if pcurr.Load() == oldChild {
			newChild.sibling.Store(oldChild.sibling.Load())
			pcurr.Store(newChild)
			return
		}
	}
}

// --- SIEVE List helpers (must be called with shard.mu held) ---

func (s *cacheShard) sievePushHead(node *shardedRadixNode) {
	node.sievePrev = nil
	node.sieveNext = s.sieveHead
	if s.sieveHead != nil {
		s.sieveHead.sievePrev = node
	}
	s.sieveHead = node
	if s.sieveTail == nil {
		s.sieveTail = node
	}
	s.len++
}

func (s *cacheShard) sieveRemove(node *shardedRadixNode) {
	if s.sieveHand == node {
		s.sieveHand = node.sievePrev
	}
	if node.sievePrev != nil {
		node.sievePrev.sieveNext = node.sieveNext
	} else {
		s.sieveHead = node.sieveNext
	}
	if node.sieveNext != nil {
		node.sieveNext.sievePrev = node.sievePrev
	} else {
		s.sieveTail = node.sievePrev
	}
	node.sievePrev = nil
	node.sieveNext = nil
	s.len--
}

func (s *cacheShard) sieveReplace(oldNode, newNode *shardedRadixNode) {
	newNode.sievePrev = oldNode.sievePrev
	newNode.sieveNext = oldNode.sieveNext
	if oldNode.sievePrev != nil {
		oldNode.sievePrev.sieveNext = newNode
	} else {
		s.sieveHead = newNode
	}
	if oldNode.sieveNext != nil {
		oldNode.sieveNext.sievePrev = newNode
	} else {
		s.sieveTail = newNode
	}
	if s.sieveHand == oldNode {
		s.sieveHand = newNode
	}
	oldNode.sievePrev = nil
	oldNode.sieveNext = nil
}

// --- Eviction and Pruning ---

func (s *cacheShard) evictOneLocked(c *ShardedRadixCache) ValueType {
	for s.sieveTail != nil {
		node := s.sieveHand
		if node == nil {
			node = s.sieveTail
		}
		s.sieveHand = node.sievePrev

		if node.accessed.Load() {
			node.accessed.Store(false)
		} else {
			return s.eraseNodeInternal(node, c)
		}
	}
	return nil
}

func (s *cacheShard) evictOne(c *ShardedRadixCache) ValueType {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.evictOneLocked(c)
}

func (c *ShardedRadixCache) evictGlobal(evictedValues *[]ValueType, protectedShardIdx int) {
	// First pass: try evicting from all shards except protectedShardIdx
	for uint64(c.currentSize.Load()) > c.maxSize {
		idx := int(c.evictShardIdx.Add(1) % 256)
		if idx == protectedShardIdx {
			continue
		}
		if evicted := c.shards[idx].evictOne(c); evicted != nil {
			if evictedValues != nil {
				*evictedValues = append(*evictedValues, evicted)
			}
		} else {
			var totalLen int
			for i := 0; i < 256; i++ {
				if i == protectedShardIdx {
					continue
				}
				c.shards[i].mu.Lock()
				totalLen += c.shards[i].len
				c.shards[i].mu.Unlock()
			}
			if totalLen == 0 {
				break
			}
		}
	}

	// Second pass: if we still exceed maxSize, allow evicting from protectedShardIdx
	for uint64(c.currentSize.Load()) > c.maxSize {
		if evicted := c.shards[protectedShardIdx].evictOne(c); evicted != nil {
			if evictedValues != nil {
				*evictedValues = append(*evictedValues, evicted)
			}
		} else {
			break
		}
	}
}

func (s *cacheShard) eraseNodeInternal(node *shardedRadixNode, c *ShardedRadixCache) ValueType {
	val := node.getValue()
	if val == nil {
		return nil
	}
	sz := val.Size()
	s.currentSize -= sz
	c.currentSize.Add(-int64(sz))
	s.sieveRemove(node)
	node.setValue(nil)
	s.compressPathUpwards(node)
	return val
}

func (s *cacheShard) compressPathUpwards(curr *shardedRadixNode) {
	for curr != nil && curr != s.root.Load() {
		if curr.getValue() != nil {
			break
		}

		child := curr.child.Load()
		if child == nil {
			parent := curr.parent
			if parent != nil {
				s.removeChildAtomic(parent, curr)
				curr = parent
				continue
			}
			break
		}

		if child.sibling.Load() == nil {
			parent := curr.parent
			if parent == nil {
				break
			}
			mergedNode := &shardedRadixNode{
				prefix: strings.Clone(curr.prefix + child.prefix),
				parent: parent,
			}
			mergedNode.child.Store(child.child.Load())
			mergedNode.accessed.Store(child.accessed.Load())
			if val := child.getValue(); val != nil {
				mergedNode.setValue(val)
				s.sieveReplace(child, mergedNode)
			}
			for ch := mergedNode.child.Load(); ch != nil; ch = ch.sibling.Load() {
				ch.parent = mergedNode
			}
			s.replaceChildAtomic(parent, curr, mergedNode)
			curr = parent
			continue
		}
		break
	}
}

func (s *cacheShard) processDetachedSubtreesBatchLocked(maxNodes int, c *ShardedRadixCache) {
	if len(s.detachedQueue) == 0 {
		return
	}

	nodesProcessed := 0
	for len(s.detachedQueue) > 0 && nodesProcessed < maxNodes {
		item := &s.detachedQueue[0]
		subRoot := item.root
		curr := item.curr
		for curr != nil && nodesProcessed < maxNodes {
			if val := curr.getValue(); val != nil {
				sz := val.Size()
				s.currentSize -= sz
				c.currentSize.Add(-int64(sz))
				s.sieveRemove(curr)
				curr.setValue(nil)
			}
			nodesProcessed++

			if child := curr.child.Load(); child != nil {
				curr = child
				continue
			}
			for curr != subRoot && curr.sibling.Load() == nil {
				curr = curr.parent
			}
			if curr == subRoot {
				curr = nil
				break
			}
			curr = curr.sibling.Load()
		}
		if curr == nil {
			s.detachedQueue = s.detachedQueue[1:]
		} else {
			item.curr = curr
			break
		}
	}
}

func (s *cacheShard) processDetachedSubtreesBatch(maxNodes int, c *ShardedRadixCache) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.processDetachedSubtreesBatchLocked(maxNodes, c)
}

func (s *cacheShard) pruneAllEmptyLeaves() {
	s.mu.Lock()
	defer s.mu.Unlock()

	root := s.root.Load()
	if root == nil {
		return
	}
	curr := root
	for curr != nil {
		if child := curr.child.Load(); child != nil {
			curr = child
			continue
		}

		for curr != nil {
			if curr == root {
				return
			}
			next := curr.sibling.Load()
			parent := curr.parent

			if curr.getValue() == nil {
				if curr.child.Load() == nil {
					s.removeChildAtomic(parent, curr)
				} else if curr.child.Load().sibling.Load() == nil {
					onlyChild := curr.child.Load()
					mergedNode := &shardedRadixNode{
						prefix: strings.Clone(curr.prefix + onlyChild.prefix),
						parent: parent,
					}
					mergedNode.child.Store(onlyChild.child.Load())
					mergedNode.accessed.Store(onlyChild.accessed.Load())
					if val := onlyChild.getValue(); val != nil {
						mergedNode.setValue(val)
						s.sieveReplace(onlyChild, mergedNode)
					}
					for ch := mergedNode.child.Load(); ch != nil; ch = ch.sibling.Load() {
						ch.parent = mergedNode
					}
					s.replaceChildAtomic(parent, curr, mergedNode)
				}
			}

			if next != nil {
				curr = next
				break
			}
			curr = parent
		}
	}
}

// --- Lookup & Mutate Core ---

func (s *cacheShard) lookUp(key string, markAccessed bool) ValueType {
	node := s.root.Load()
	search := key

	for node != nil {
		if len(search) == 0 {
			val := node.getValue()
			if val != nil {
				if markAccessed {
					node.accessed.Store(true)
				}
				return val
			}
			return nil
		}

		child := node.getChild(search[0])
		if child == nil {
			return nil
		}

		if strings.HasPrefix(search, child.prefix) {
			search = search[len(child.prefix):]
			node = child
			continue
		}
		return nil
	}
	return nil
}

func (s *cacheShard) getNode(key string) (*shardedRadixNode, bool) {
	node := s.root.Load()
	search := key

	for {
		if len(search) == 0 {
			if node.getValue() != nil {
				return node, true
			}
			return nil, false
		}

		child := node.getChild(search[0])
		if child == nil {
			return nil, false
		}

		if strings.HasPrefix(search, child.prefix) {
			search = search[len(child.prefix):]
			node = child
			continue
		}
		return nil, false
	}
}

func (s *cacheShard) insertNode(key string, value ValueType) (*shardedRadixNode, ValueType) {
	node := s.root.Load()
	search := key

	for {
		if len(search) == 0 {
			oldVal := node.getValue()
			node.setValue(value)
			return node, oldVal
		}

		child := node.getChild(search[0])
		if child == nil {
			newLeaf := &shardedRadixNode{
				prefix: strings.Clone(search),
			}
			newLeaf.setValue(value)
			s.addChildAtomic(node, newLeaf)
			return newLeaf, nil
		}

		lcp := longestCommonPrefix(search, child.prefix)
		if lcp == len(child.prefix) {
			search = search[lcp:]
			node = child
			continue
		}

		newChild := &shardedRadixNode{
			prefix: strings.Clone(child.prefix[lcp:]),
		}
		newChild.child.Store(child.child.Load())
		newChild.accessed.Store(child.accessed.Load())
		if oldVal := child.getValue(); oldVal != nil {
			newChild.setValue(oldVal)
			s.sieveReplace(child, newChild)
		}
		for ch := newChild.child.Load(); ch != nil; ch = ch.sibling.Load() {
			ch.parent = newChild
		}

		splitNode := &shardedRadixNode{
			prefix: strings.Clone(child.prefix[:lcp]),
			parent: node,
		}
		s.addChildAtomic(splitNode, newChild)

		s.replaceChildAtomic(node, child, splitNode)

		if lcp == len(search) {
			oldVal := splitNode.getValue()
			splitNode.setValue(value)
			return splitNode, oldVal
		}

		newLeaf := &shardedRadixNode{
			prefix: strings.Clone(search[lcp:]),
		}
		newLeaf.setValue(value)
		s.addChildAtomic(splitNode, newLeaf)
		return newLeaf, nil
	}
}

// --- Interface Methods ---

func (c *ShardedRadixCache) Insert(key string, value ValueType) ([]ValueType, error) {
	if value == nil {
		return nil, ErrInvalidEntry
	}
	valueSize := value.Size()
	if valueSize > c.maxSize {
		return nil, ErrInvalidEntrySize
	}

	s, sIdx := c.getShardWithIdx(key)
	s.mu.Lock()

	s.processDetachedSubtreesBatchLocked(64, c)

	node, oldVal := s.insertNode(key, value)
	if oldVal != nil {
		node.accessed.Store(true)
		diff := int64(valueSize) - int64(oldVal.Size())
		s.currentSize = uint64(int64(s.currentSize) + diff)
		c.currentSize.Add(diff)
	} else {
		s.sievePushHead(node)
		s.currentSize += valueSize
		c.currentSize.Add(int64(valueSize))
	}

	var evictedValues []ValueType
	for s.maxSize >= 1024 && s.currentSize > s.maxSize && s.sieveTail != nil && s.sieveTail != node {
		if evicted := s.evictOneLocked(c); evicted != nil {
			evictedValues = append(evictedValues, evicted)
		}
	}
	s.mu.Unlock()

	c.evictGlobal(&evictedValues, sIdx)
	return evictedValues, nil
}

func (c *ShardedRadixCache) Erase(key string) ValueType {
	s := c.getShard(key)
	s.mu.Lock()
	defer s.mu.Unlock()

	s.processDetachedSubtreesBatchLocked(64, c)

	node, ok := s.getNode(key)
	if !ok {
		return nil
	}
	return s.eraseNodeInternal(node, c)
}

func (c *ShardedRadixCache) LookUp(key string) ValueType {
	s := c.getShard(key)
	return s.lookUp(key, true)
}

func (c *ShardedRadixCache) LookUpWithoutChangingOrder(key string) ValueType {
	s := c.getShard(key)
	return s.lookUp(key, false)
}

func (c *ShardedRadixCache) UpdateWithoutChangingOrder(key string, value ValueType) error {
	if value == nil {
		return ErrInvalidEntry
	}
	s := c.getShard(key)
	s.mu.Lock()
	defer s.mu.Unlock()

	node, ok := s.getNode(key)
	if !ok {
		return ErrEntryNotExist
	}
	val := node.getValue()
	if val == nil {
		return ErrEntryNotExist
	}
	if val.Size() != value.Size() {
		return ErrInvalidUpdateEntrySize
	}
	node.setValue(value)
	return nil
}

func (c *ShardedRadixCache) UpdateSize(key string, sizeDelta uint64) error {
	s, sIdx := c.getShardWithIdx(key)
	s.mu.Lock()

	node, ok := s.getNode(key)
	if !ok || node.getValue() == nil {
		s.mu.Unlock()
		return ErrEntryNotExist
	}

	s.currentSize += sizeDelta
	c.currentSize.Add(int64(sizeDelta))

	for s.maxSize >= 1024 && s.currentSize > s.maxSize && s.sieveTail != nil && s.sieveTail != node {
		s.evictOneLocked(c)
	}
	s.mu.Unlock()

	c.evictGlobal(nil, sIdx)
	return nil
}

func (c *ShardedRadixCache) EraseEntriesWithGivenPrefix(prefix string) {
	if prefix == "" {
		for i := 0; i < 256; i++ {
			s := &c.shards[i]
			s.mu.Lock()
			oldSize := s.currentSize
			if oldSize > 0 {
				c.currentSize.Add(-int64(oldSize))
			}
			s.root.Store(&shardedRadixNode{})
			s.sieveHead = nil
			s.sieveTail = nil
			s.sieveHand = nil
			s.len = 0
			s.currentSize = 0
			s.detachedQueue = nil
			s.mu.Unlock()
		}
		c.notifySweep()
		return
	}

	for i := 0; i < 256; i++ {
		s := &c.shards[i]
		s.mu.Lock()
		node := s.root.Load()
		search := prefix
		for len(search) > 0 {
			child := node.getChild(search[0])
			if child == nil {
				break
			}
			lcp := longestCommonPrefix(search, child.prefix)
			if lcp == len(search) {
				s.removeChildAtomic(node, child)
				child.parent = nil
				s.detachedQueue = append(s.detachedQueue, detachedSubtree{root: child, curr: child})
				s.processDetachedSubtreesBatchLocked(64, c)
				break
			}
			if lcp == len(child.prefix) {
				search = search[len(child.prefix):]
				node = child
				continue
			}
			break
		}
		s.mu.Unlock()
	}
	c.notifySweep()
}
