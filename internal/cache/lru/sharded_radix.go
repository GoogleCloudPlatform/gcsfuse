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

// compactRadixNode represents a unified cache-aligned node struct in the RCU LCRS radix tree with SIEVE eviction.
type compactRadixNode struct {
	prefixPtr *byte                            //  8 bytes
	child     atomic.Pointer[compactRadixNode] //  8 bytes
	sibling   atomic.Pointer[compactRadixNode] //  8 bytes
	parent    *compactRadixNode                //  8 bytes
	sievePrev *compactRadixNode                //  8 bytes
	sieveNext *compactRadixNode                //  8 bytes
	value     unsafe.Pointer                   //  8 bytes (atomic pointer to *ValueType)
	packed    atomic.Uint64                    //  8 bytes
}

// Compile-time assertion to guarantee node struct size (64 bytes).
var _ = [1]struct{}{}[64-unsafe.Sizeof(compactRadixNode{})]

func newCompactRadixNode(prefix string) *compactRadixNode {
	n := &compactRadixNode{}
	if len(prefix) > 0 {
		n.prefixPtr = unsafe.StringData(prefix)
		n.packed.Store(uint64(len(prefix)) << 47)
	}
	return n
}

func (n *compactRadixNode) getPrefix() string {
	packed := n.packed.Load()
	prefixLen := int((packed >> 47) & 0xFFFF)
	if prefixLen == 0 {
		return ""
	}
	return unsafe.String(n.prefixPtr, prefixLen)
}

func (n *compactRadixNode) getExtraSize() uint64 {
	return n.packed.Load() & 0x7FFFFFFFFFFF
}

func (n *compactRadixNode) setExtraSize(size uint64) {
	for {
		old := n.packed.Load()
		if size > 0x7FFFFFFFFFFF {
			panic("extraSize overflow")
		}
		newPacked := (old &^ 0x7FFFFFFFFFFF) | size
		if n.packed.CompareAndSwap(old, newPacked) {
			break
		}
	}
}

func (n *compactRadixNode) addExtraSize(delta uint64) {
	for {
		old := n.packed.Load()
		oldSize := old & 0x7FFFFFFFFFFF
		newSize := oldSize + delta
		if newSize > 0x7FFFFFFFFFFF {
			panic("extraSize overflow")
		}
		newPacked := (old &^ 0x7FFFFFFFFFFF) | newSize
		if n.packed.CompareAndSwap(old, newPacked) {
			break
		}
	}
}

func (n *compactRadixNode) getAccessed() bool {
	return (n.packed.Load() & (1 << 63)) != 0
}

func (n *compactRadixNode) setAccessed(v bool) {
	for {
		old := n.packed.Load()
		var newPacked uint64
		if v {
			newPacked = old | (1 << 63)
		} else {
			newPacked = old &^ (1 << 63)
		}
		if old == newPacked || n.packed.CompareAndSwap(old, newPacked) {
			break
		}
	}
}

func (n *compactRadixNode) isPayload() bool {
	return n != nil && n.getValue() != nil
}

func (n *compactRadixNode) getValue() ValueType {
	if n == nil {
		return nil
	}
	ptr := atomic.LoadPointer(&n.value)
	if ptr == nil {
		return nil
	}
	return *(*ValueType)(ptr)
}

func (n *compactRadixNode) setValue(val ValueType) {
	if n == nil {
		return
	}
	if val == nil {
		atomic.StorePointer(&n.value, nil)
		return
	}
	box := new(ValueType)
	*box = val
	atomic.StorePointer(&n.value, unsafe.Pointer(box))
}

// getChild finds a child node whose prefix starts with the given byte (RCU read without lock).
func (n *compactRadixNode) getChild(b byte) *compactRadixNode {
	for curr := n.child.Load(); curr != nil; curr = curr.sibling.Load() {
		pref := curr.getPrefix()
		if len(pref) > 0 && pref[0] == b {
			return curr
		}
		// Sibling chains are lexicographically sorted by prefix[0], allowing early exit.
		if len(pref) > 0 && pref[0] > b {
			return nil
		}
	}
	return nil
}

type detachedSubtree struct {
	root  *compactRadixNode
	stack []*compactRadixNode
}

func newDetachedSubtree(root *compactRadixNode) detachedSubtree {
	if root == nil {
		return detachedSubtree{}
	}
	return detachedSubtree{
		root:  root,
		stack: []*compactRadixNode{root},
	}
}

// cacheShard represents a single shard padded to exactly 128 bytes to eliminate false sharing.
type cacheShard struct {
	mu          sync.Mutex
	maxSize     uint64
	currentSize uint64
	len         int

	root atomic.Pointer[compactRadixNode]

	// SIEVE clock eviction pointers
	sieveHead *compactRadixNode
	sieveTail *compactRadixNode
	sieveHand *compactRadixNode

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

// ShardedRadixCache implements lru.Cache with 32-way sharding, RCU lookups, and SIEVE eviction.
type ShardedRadixCache struct {
	shards        [32]cacheShard
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

	baseShardSize := maxSize / 32
	remainder := maxSize % 32

	for i := 0; i < 32; i++ {
		shardSize := baseShardSize
		if uint64(i) < remainder {
			shardSize++
		}
		c.shards[i].maxSize = shardSize
		c.shards[i].root.Store(&compactRadixNode{})
	}

	c.wg.Add(1)
	go c.backgroundWorker()

	return c
}

func (c *ShardedRadixCache) GetCurrentSize() uint64 {
	return uint64(c.currentSize.Load())
}

// GetShardSize returns the size of cacheShard struct in bytes for verification.
func GetShardSize() uintptr {
	return unsafe.Sizeof(cacheShard{})
}

func ParentDirectoryPrefix(key string) string {
	idx := strings.LastIndex(key, "/")
	if idx >= 0 {
		return key[:idx+1]
	}
	return key
}

// StripBucketPrefix slices "bucketName/" from key if key starts with "bucketName/".
// It performs zero allocations and strictly preserves trailing slashes without path cleaning.
func StripBucketPrefix(key string, bucketName string) string {
	if bucketName == "" {
		return key
	}
	prefixLen := len(bucketName) + 1
	if len(key) >= prefixLen && key[len(bucketName)] == '/' && strings.HasPrefix(key, bucketName) {
		return key[prefixLen:]
	}
	return key
}

func (c *ShardedRadixCache) getShard(key string) *cacheShard {
	s, _ := c.getShardWithIdx(key)
	return s
}

func (c *ShardedRadixCache) getShardWithIdx(key string) (*cacheShard, int) {
	shardKey := ParentDirectoryPrefix(key)
	hash := uint32(2166136261)
	for i := 0; i < len(shardKey); i++ {
		hash ^= uint32(shardKey[i])
		hash *= 16777619
	}
	idx := int(hash & 0x1F)
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
	for i := 0; i < 32; i++ {
		c.shards[i].processDetachedSubtreesBatch(maxNodes, c)
	}
}

func (c *ShardedRadixCache) drainAll() {
	for i := 0; i < 32; i++ {
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
	for i := 0; i < 32; i++ {
		c.shards[i].pruneAllEmptyLeaves()
	}
}

// --- Tree mutation helpers (must be called with shard.mu held) ---

func (s *cacheShard) addChildAtomic(parent, newChild *compactRadixNode) {
	newChild.parent = parent
	pcurr := &parent.child
	newPref := newChild.getPrefix()
	for pcurr.Load() != nil && pcurr.Load().getPrefix()[0] < newPref[0] {
		pcurr = &pcurr.Load().sibling
	}
	newChild.sibling.Store(pcurr.Load())
	pcurr.Store(newChild)
}

func (s *cacheShard) removeChildAtomic(parent, childToRemove *compactRadixNode) {
	for pcurr := &parent.child; pcurr.Load() != nil; pcurr = &pcurr.Load().sibling {
		if pcurr.Load() == childToRemove {
			pcurr.Store(childToRemove.sibling.Load())
			return
		}
	}
}

func (s *cacheShard) replaceChildAtomic(parent, oldChild, newChild *compactRadixNode) {
	for pcurr := &parent.child; pcurr.Load() != nil; pcurr = &pcurr.Load().sibling {
		if pcurr.Load() == oldChild {
			newChild.sibling.Store(oldChild.sibling.Load())
			pcurr.Store(newChild)
			return
		}
	}
}

// --- SIEVE List helpers (must be called with shard.mu held) ---

func (s *cacheShard) sievePushHead(node *compactRadixNode) {
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

func (s *cacheShard) sieveRemove(node *compactRadixNode) {
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

// --- Eviction and Pruning ---

func (s *cacheShard) evictOneLocked(c *ShardedRadixCache) ValueType {
	for s.sieveTail != nil {
		node := s.sieveHand
		if node == nil {
			node = s.sieveTail
		}
		s.sieveHand = node.sievePrev

		if node.getAccessed() {
			node.setAccessed(false)
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
		idx := int(c.evictShardIdx.Add(1) % 32)
		if idx == protectedShardIdx {
			continue
		}
		if evicted := c.shards[idx].evictOne(c); evicted != nil {
			if evictedValues != nil {
				*evictedValues = append(*evictedValues, evicted)
			}
		} else {
			var totalLen int
			for i := 0; i < 32; i++ {
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

func (s *cacheShard) eraseNodeInternal(node *compactRadixNode, c *ShardedRadixCache) ValueType {
	if node == nil {
		return nil
	}
	val := node.getValue()
	if val == nil {
		return nil
	}
	sz := val.Size() + node.getExtraSize()
	s.currentSize -= sz
	c.currentSize.Add(-int64(sz))
	node.setExtraSize(0)
	s.sieveRemove(node)
	node.setValue(nil)
	s.compressPathUpwards(node)
	return val
}

func (s *cacheShard) compressPathUpwards(curr *compactRadixNode) {
	for curr != nil && curr != s.root.Load() {
		if curr.isPayload() && curr.getValue() != nil {
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
			mergedPrefix := Intern(curr.getPrefix() + child.getPrefix())
			mergedNode := newCompactRadixNode(mergedPrefix)
			mergedNode.parent = parent
			mergedNode.value = child.value
			mergedNode.setExtraSize(child.getExtraSize())
			mergedNode.child.Store(child.child.Load())
			if child.getValue() != nil {
				mergedNode.setAccessed(child.getAccessed())
				if s.sieveHead == child {
					s.sieveHead = mergedNode
				}
				if s.sieveTail == child {
					s.sieveTail = mergedNode
				}
				if s.sieveHand == child {
					s.sieveHand = mergedNode
				}
				if child.sievePrev != nil {
					child.sievePrev.sieveNext = mergedNode
				}
				if child.sieveNext != nil {
					child.sieveNext.sievePrev = mergedNode
				}
				mergedNode.sievePrev = child.sievePrev
				mergedNode.sieveNext = child.sieveNext
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
	for len(s.detachedQueue) > 0 && maxNodes > 0 {
		item := &s.detachedQueue[0]
		
		for len(item.stack) > 0 && maxNodes > 0 {
			curr := item.stack[len(item.stack)-1]
			item.stack = item.stack[:len(item.stack)-1]

			if curr.isPayload() {
				if val := curr.getValue(); val != nil {
					sz := val.Size() + curr.getExtraSize()
					s.currentSize -= sz
					c.currentSize.Add(-int64(sz))
					s.sieveRemove(curr)
					curr.setValue(nil)
					curr.setExtraSize(0)
				}
			}
			maxNodes--

			if ch := curr.child.Load(); ch != nil {
				item.stack = append(item.stack, ch)
			}
			if curr != item.root {
				if sib := curr.sibling.Load(); sib != nil {
					item.stack = append(item.stack, sib)
				}
			}
		}

		if len(item.stack) == 0 {
			s.detachedQueue = s.detachedQueue[1:]
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

			if !curr.isPayload() || curr.getValue() == nil {
				if curr.child.Load() == nil {
					s.removeChildAtomic(parent, curr)
				} else if curr.child.Load().sibling.Load() == nil {
					onlyChild := curr.child.Load()
					mergedPrefix := Intern(curr.getPrefix() + onlyChild.getPrefix())
					mergedNode := newCompactRadixNode(mergedPrefix)
					mergedNode.parent = parent
					mergedNode.value = onlyChild.value
					mergedNode.setExtraSize(onlyChild.getExtraSize())
					mergedNode.child.Store(onlyChild.child.Load())
					if onlyChild.getValue() != nil {
						mergedNode.setAccessed(onlyChild.getAccessed())
						if s.sieveHead == onlyChild {
							s.sieveHead = mergedNode
						}
						if s.sieveTail == onlyChild {
							s.sieveTail = mergedNode
						}
						if s.sieveHand == onlyChild {
							s.sieveHand = mergedNode
						}
						if onlyChild.sievePrev != nil {
							onlyChild.sievePrev.sieveNext = mergedNode
						}
						if onlyChild.sieveNext != nil {
							onlyChild.sieveNext.sievePrev = mergedNode
						}
						mergedNode.sievePrev = onlyChild.sievePrev
						mergedNode.sieveNext = onlyChild.sieveNext
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
					node.setAccessed(true)
				}
				return val
			}
			return nil
		}

		child := node.getChild(search[0])
		if child == nil {
			return nil
		}

		childPref := child.getPrefix()
		if strings.HasPrefix(search, childPref) {
			search = search[len(childPref):]
			node = child
			continue
		}
		return nil
	}
	return nil
}

func (s *cacheShard) getNode(key string) (*compactRadixNode, bool) {
	node := s.root.Load()
	search := key

	for {
		if len(search) == 0 {
			if node.isPayload() && node.getValue() != nil {
				return node, true
			}
			return nil, false
		}

		child := node.getChild(search[0])
		if child == nil {
			return nil, false
		}

		childPref := child.getPrefix()
		if strings.HasPrefix(search, childPref) {
			search = search[len(childPref):]
			node = child
			continue
		}
		return nil, false
	}
}

func (s *cacheShard) insertNode(key string, value ValueType) (*compactRadixNode, ValueType) {
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
			newLeaf := newCompactRadixNode(Intern(search))
			newLeaf.setValue(value)
			s.addChildAtomic(node, newLeaf)
			return newLeaf, nil
		}

		childPref := child.getPrefix()
		lcp := longestCommonPrefix(search, childPref)
		if lcp == len(childPref) {
			search = search[lcp:]
			node = child
			continue
		}

		ncHub := newCompactRadixNode(Intern(childPref[lcp:]))
		ncHub.value = child.value
		ncHub.setExtraSize(child.getExtraSize())
		ncHub.child.Store(child.child.Load())
		if child.getValue() != nil {
			ncHub.setAccessed(child.getAccessed())
			if s.sieveHead == child {
				s.sieveHead = ncHub
			}
			if s.sieveTail == child {
				s.sieveTail = ncHub
			}
			if s.sieveHand == child {
				s.sieveHand = ncHub
			}
			if child.sievePrev != nil {
				child.sievePrev.sieveNext = ncHub
			}
			if child.sieveNext != nil {
				child.sieveNext.sievePrev = ncHub
			}
			ncHub.sievePrev = child.sievePrev
			ncHub.sieveNext = child.sieveNext
		}
		for ch := ncHub.child.Load(); ch != nil; ch = ch.sibling.Load() {
			ch.parent = ncHub
		}

		splitNode := newCompactRadixNode(Intern(childPref[:lcp]))
		splitNode.parent = node

		var targetNode *compactRadixNode
		if lcp == len(search) {
			splitNode.setValue(value)
			targetNode = splitNode
		}

		s.addChildAtomic(splitNode, ncHub)
		s.replaceChildAtomic(node, child, splitNode)

		if lcp == len(search) {
			return targetNode, nil
		}

		newLeaf := newCompactRadixNode(Intern(search[lcp:]))
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
		node.setAccessed(true)
		oldExtra := node.getExtraSize()
		node.setExtraSize(0)
		diff := int64(valueSize) - int64(oldVal.Size()+oldExtra)
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
	if !ok || node.getValue() == nil {
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

	node.addExtraSize(sizeDelta)
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
		for i := 0; i < 32; i++ {
			s := &c.shards[i]
			s.mu.Lock()
			oldSize := s.currentSize
			if oldSize > 0 {
				c.currentSize.Add(-int64(oldSize))
			}
			s.root.Store(&compactRadixNode{})
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

	for i := 0; i < 32; i++ {
		s := &c.shards[i]
		s.mu.Lock()
		node := s.root.Load()
		search := prefix
		for len(search) > 0 && node != nil {
			var fullMatches []*compactRadixNode
			var partialMatch *compactRadixNode

			for curr := node.child.Load(); curr != nil; curr = curr.sibling.Load() {
				currPref := curr.getPrefix()
				lcp := longestCommonPrefix(search, currPref)
				if lcp == len(search) {
					fullMatches = append(fullMatches, curr)
				} else if lcp == len(currPref) && partialMatch == nil {
					partialMatch = curr
				}
			}

			if len(fullMatches) > 0 {
				for _, child := range fullMatches {
					s.removeChildAtomic(node, child)
					child.parent = nil
					s.detachedQueue = append(s.detachedQueue, newDetachedSubtree(child))
					s.processDetachedSubtreesBatchLocked(math.MaxInt, c)
				}
				break
			}

			if partialMatch != nil {
				search = search[len(partialMatch.getPrefix()):]
				node = partialMatch
				continue
			}

			break
		}
		s.mu.Unlock()
	}
	c.notifySweep()
}
