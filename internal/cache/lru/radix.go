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
	"strings"
	"sync/atomic"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/locker"
)

// radixNode represents a node in the custom radix tree.
// It uses a Left-Child Right-Sibling (LCRS) representation to avoid slice allocations.
type radixNode struct {
	prefix  string
	value   ValueType
	size    uint64
	parent  *radixNode
	child   *radixNode
	sibling *radixNode
	// LRU Linked List pointers
	prev       *radixNode
	next       *radixNode
	evicted    bool
	accessTime atomic.Int64
}

// radixCache encapsulates the core tree structure and implements the lru.Cache interface.
type radixCache struct {
	/////////////////////////
	// Constant data
	/////////////////////////

	// INVARIANT: maxSize > 0
	maxSize uint64

	/////////////////////////
	// Mutable state
	/////////////////////////

	// Sum of Size() of all the entries in the cache.
	currentSize uint64

	root *radixNode

	// Head and tail of the LRU Doubly Linked List
	head *radixNode
	tail *radixNode

	// len is the number of nodes currently linked in the LRU list.
	len int

	promoChan chan *radixNode
	closed    atomic.Bool

	// Pointer to dynamic global current size counter, used when sharded.
	// For standalone radixCache, points to localGlobalSize below.
	globalCurrentSize *atomic.Int64
	localGlobalSize   atomic.Int64

	// All public methods of this Cache uses this RW mutex based locker while
	// accessing/updating Cache's data.
	mu locker.RWLocker
}

// NewRadixCache returns the reference of cache object by initialising the cache with
// the supplied maxSize, which must be greater than zero.
func NewRadixCache(maxSize uint64) Cache {
	if maxSize == 0 {
		panic("Invalid maxSize")
	}

	c := &radixCache{
		maxSize:   maxSize,
		root:      &radixNode{},
		promoChan: make(chan *radixNode, 1024),
	}
	c.globalCurrentSize = &c.localGlobalSize
	c.mu = locker.NewRW("RadixCache", c.checkInvariants)
	return c
}

func newShardRadixCache(maxSize uint64, globalSize *atomic.Int64) *radixCache {
	c := &radixCache{
		maxSize:           maxSize,
		root:              &radixNode{},
		promoChan:         make(chan *radixNode, 1024),
		globalCurrentSize: globalSize,
	}
	c.mu = locker.NewRW("RadixCacheShard", c.checkInvariants)
	return c
}

func (c *radixCache) checkInvariants() {
	// INVARIANT: currentSize <= maxSize
	if c.currentSize > c.maxSize {
		panic(fmt.Sprintf("CurrentSize %v over maxSize %v", c.currentSize, c.maxSize))
	}

	// INVARIANT: Each element in the LRU list must have a valid value
	lruCount := 0
	for curr := c.head; curr != nil; curr = curr.next {
		lruCount++
		if curr.value == nil {
			panic(fmt.Sprintf("Unexpected empty value in LRU list for prefix: %v", curr.prefix))
		}
	}

	if lruCount != c.len {
		panic(fmt.Sprintf("LRU list actual count %v does not match c.len %v", lruCount, c.len))
	}

	// INVARIANT: Every value-bearing node in the tree must exist in the LRU list exactly once
	treeCount := 0

	// Iterative pre-order traversal using parent/sibling pointers (O(1) space) to prevent stack overflows
	curr := c.root
	for curr != nil {
		if curr.value != nil {
			treeCount++
			// A node is verifiably in the LRU list if it is the head, or if it has a predecessor
			inLRU := c.head == curr || curr.prev != nil
			if !inLRU {
				panic(fmt.Sprintf("Mismatch: Node with prefix '%v' has a value but is missing from LRU list", curr.prefix))
			}
		}

		// Advance to next node
		if curr.child != nil {
			curr = curr.child
			continue
		}

		// Backtrack up parent chain
		for curr != c.root && curr.sibling == nil {
			curr = curr.parent
		}
		if curr == c.root {
			break
		}
		curr = curr.sibling
	}

	if treeCount != c.len {
		panic(fmt.Sprintf("Tree actual value count %v does not match LRU length %v", treeCount, c.len))
	}
}

// longestCommonPrefix finds the length of the longest common prefix of a and b.
func longestCommonPrefix(a, b string) int {
	i := 0
	for i < len(a) && i < len(b) && a[i] == b[i] {
		i++
	}
	return i
}

// getChild finds a child node whose prefix starts with the given byte.
func (n *radixNode) getChild(b byte) *radixNode {
	for curr := n.child; curr != nil; curr = curr.sibling {
		if curr.prefix[0] == b {
			return curr
		}
		//as the siblings are arranged in lexicographical order we can exit early
		if curr.prefix[0] > b {
			return nil
		}
	}
	return nil
}

// addChild adds a child node, maintaining sorted order by the first byte of the prefix.
func (n *radixNode) addChild(newChild *radixNode) {
	newChild.parent = n
	pcurr := &n.child
	for *pcurr != nil && (*pcurr).prefix[0] < newChild.prefix[0] {
		pcurr = &(*pcurr).sibling
	}
	newChild.sibling = *pcurr
	*pcurr = newChild
}

// removeChild directly removes a child node by reference from the sibling list.
func (n *radixNode) removeChild(childToRemove *radixNode) {
	for pcurr := &n.child; *pcurr != nil; pcurr = &(*pcurr).sibling {
		if *pcurr == childToRemove {
			*pcurr = childToRemove.sibling
			return
		}
	}
}

// replaceChild finds oldChild in the sibling linked-list and replaces it with newChild,
// seamlessly preserving the rest of the sibling chain.
func (n *radixNode) replaceChild(oldChild, newChild *radixNode) {
	for pcurr := &n.child; *pcurr != nil; pcurr = &(*pcurr).sibling {
		if *pcurr == oldChild {
			newChild.sibling = oldChild.sibling
			*pcurr = newChild
			return
		}
	}
}

// insertNode inserts a new key into the radix tree and returns the leaf node.
func (c *radixCache) insertNode(key string, value ValueType) (*radixNode, ValueType) {
	if value == nil {
		return nil, nil
	}

	node := c.root
	search := key

	for {
		if len(search) == 0 {
			oldValue := node.value
			node.value = value
			return node, oldValue
		}

		child := node.getChild(search[0])
		if child == nil {
			// clone the substring to prevent memory leaks otherwise, the slice pins the entire original string in memory
			newLeaf := &radixNode{
				prefix: strings.Clone(search),
				value:  value,
			}
			node.addChild(newLeaf)
			return newLeaf, nil
		}

		lcp := longestCommonPrefix(search, child.prefix)

		if lcp == len(child.prefix) {
			search = search[lcp:]
			node = child
			continue
		}

		splitNode := &radixNode{
			prefix: strings.Clone(child.prefix[:lcp]),
			parent: node,
		}

		node.replaceChild(child, splitNode)

		child.prefix = strings.Clone(child.prefix[lcp:])
		child.sibling = nil
		splitNode.addChild(child)

		if lcp == len(search) {
			oldValue := splitNode.value
			splitNode.value = value
			return splitNode, oldValue
		}

		newLeaf := &radixNode{
			prefix: strings.Clone(search[lcp:]),
			value:  value,
		}
		splitNode.addChild(newLeaf)
		return newLeaf, nil
	}
}

// getNode finds a leaf node by key.
func (c *radixCache) getNode(key string) (*radixNode, bool) {
	node := c.root
	search := key

	for {
		if len(search) == 0 {
			if node.value != nil {
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

// deleteNode removes a leaf node directly using its parent pointers.
func (c *radixCache) deleteNode(node *radixNode) {
	if node == nil || node.value == nil {
		return
	}

	node.value = nil
	c.compressPathUpwards(node)
}

// compressPathUpwards walks up the tree, pruning empty leaves and compressing single-child routing nodes.
func (c *radixCache) compressPathUpwards(curr *radixNode) {
	for curr != nil && curr != c.root {
		if curr.value != nil {
			break
		}

		if curr.child == nil {
			parent := curr.parent
			parent.removeChild(curr)
			curr = parent
			continue
		}

		if curr.child.sibling == nil {
			onlyChild := curr.child
			onlyChild.prefix = curr.prefix + onlyChild.prefix
			onlyChild.parent = curr.parent

			curr.parent.replaceChild(curr, onlyChild)

			curr = curr.parent
			continue
		}
		break
	}
}

func (c *radixCache) flushPromotions() {
	if c.promoChan == nil {
		return
	}
	for {
		select {
		case node := <-c.promoChan:
			if node != nil {
				c.moveToFront(node)
			}
		default:
			return
		}
	}
}

func (c *radixCache) Close() {
	c.closed.Store(true)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.promoChan != nil {
		for len(c.promoChan) > 0 {
			<-c.promoChan
		}
		c.promoChan = nil
	}
}

// --- LRU Logic ---
func (c *radixCache) moveToFront(node *radixNode) {
	if node == nil || node.evicted || node.value == nil {
		return
	}
	if c.head == node {
		return
	}
	if node.prev != nil {
		node.prev.next = node.next
	}
	if node.next != nil {
		node.next.prev = node.prev
	}
	if c.tail == node {
		c.tail = node.prev
	}
	node.prev = nil
	node.next = c.head
	if c.head != nil {
		c.head.prev = node
	}
	c.head = node
}

func (c *radixCache) pushFront(node *radixNode) {
	node.prev = nil
	node.next = c.head
	if c.head != nil {
		c.head.prev = node
	}
	c.head = node
	if c.tail == nil {
		c.tail = node
	}
	c.len++
}

func (c *radixCache) remove(node *radixNode) {
	if c.head != node && node.prev == nil {
		node.evicted = true
		return
	}
	if node.prev != nil {
		node.prev.next = node.next
	} else {
		c.head = node.next
	}
	if node.next != nil {
		node.next.prev = node.prev
	} else {
		c.tail = node.prev
	}
	node.prev = nil
	node.next = nil
	node.evicted = true
	c.len--
}

func (c *radixCache) evictOne() ValueType {
	node := c.tail
	if node == nil {
		return nil
	}
	node.evicted = true
	return c.eraseInternal(node)
}

////////////////////////////////////////////////////////////////////////
// Cache interface
////////////////////////////////////////////////////////////////////////

func (c *radixCache) insertNodeAndAdjust(key string, value ValueType, valueSize uint64) ([]ValueType, error) {
	if node, oldValue := c.insertNode(key, value); oldValue != nil {
		delta := int64(valueSize) - int64(node.size)
		node.size = valueSize
		c.currentSize = uint64(int64(c.currentSize) + delta)
		if c.globalCurrentSize != nil {
			c.globalCurrentSize.Add(delta)
		}
		c.moveToFront(node)
		node.accessTime.Store(time.Now().UnixNano())
	} else {
		node.size = valueSize
		c.pushFront(node)
		c.currentSize += valueSize
		if c.globalCurrentSize != nil {
			c.globalCurrentSize.Add(int64(valueSize))
		}
		node.accessTime.Store(time.Now().UnixNano())
	}

	var evictedValues []ValueType
	// Evict until we're at or below maxSize
	for c.currentSize > c.maxSize && c.tail != nil {
		evictedValues = append(evictedValues, c.evictOne())
	}

	return evictedValues, nil
}

// Insert the supplied value into the cache, overwriting any previous entry for
// the given key. The value must be non-nil.
// Also returns a slice of ValueType evicted by the new inserted entry.
func (c *radixCache) Insert(key string, value ValueType) ([]ValueType, error) {
	if value == nil {
		return nil, ErrInvalidEntry
	}

	valueSize := value.Size()
	if valueSize > c.maxSize {
		return nil, ErrInvalidEntrySize
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.flushPromotions()
	return c.insertNodeAndAdjust(key, value, valueSize)
}

// eraseInternal removes any entry for the supplied key from the cache without acquiring locks.
// It returns the value of the erased key, or nil if not present.
// LOCKS_REQUIRED(c.mu)
func (c *radixCache) eraseInternal(node *radixNode) (value ValueType) {
	deletedEntry := node.value
	deletedSize := node.size
	c.currentSize -= deletedSize
	if c.globalCurrentSize != nil {
		c.globalCurrentSize.Add(-int64(deletedSize))
	}

	node.evicted = true
	c.remove(node)
	c.deleteNode(node)

	return deletedEntry
}

// Erase any entry for the supplied key, also returns the value of erased key.
func (c *radixCache) Erase(key string) (value ValueType) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.flushPromotions()
	node, ok := c.getNode(key)
	if !ok {
		return nil
	}

	return c.eraseInternal(node)
}

// LookUp a previously-inserted value for the given key. Return nil if no
// value is present.
func (c *radixCache) LookUp(key string) (value ValueType) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	node, ok := c.getNode(key)
	if !ok {
		return nil
	}

	node.accessTime.Store(time.Now().UnixNano())

	if !c.closed.Load() && c.promoChan != nil {
		select {
		case c.promoChan <- node:
		default:
		}
	}

	return node.value
}

// LookUpWithoutChangingOrder looks up previously-inserted value for a given key
// without changing the order of entries in the cache. Return nil if no value
// is present.
//
// Note: Because this look up doesn't change the order, it only acquires and
// releases read lock.
func (c *radixCache) LookUpWithoutChangingOrder(key string) (value ValueType) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	node, ok := c.getNode(key)
	if !ok {
		return nil
	}

	return node.value
}

// UpdateWithoutChangingOrder updates entry with the given key in cache with
// given value without changing order of entries in cache, returning error if an
// entry with given key doesn't exist. Also, the size of value for entry
// shouldn't be updated with this method (use c.Insert for updating size).
func (c *radixCache) UpdateWithoutChangingOrder(key string, value ValueType) error {
	if value == nil {
		return ErrInvalidEntry
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.flushPromotions()
	node, ok := c.getNode(key)
	if !ok {
		return ErrEntryNotExist
	}

	if value.Size() != node.size {
		return ErrInvalidUpdateEntrySize
	}

	node.value = value
	node.size = value.Size()
	return nil
}

// UpdateSize updates the currentSize accounting when an entry's size has changed.
// This is needed for entries whose size grows incrementally (e.g., sparse files).
// The entry's order in the LRU is not changed.
func (c *radixCache) UpdateSize(key string, sizeDelta uint64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.flushPromotions()
	node, ok := c.getNode(key)
	if !ok {
		return ErrEntryNotExist
	}

	// Update currentSize accounting
	node.size += sizeDelta
	c.currentSize += sizeDelta
	if c.globalCurrentSize != nil {
		c.globalCurrentSize.Add(int64(sizeDelta))
	}

	// Evict until we're at or below maxSize to maintain invariants
	for c.currentSize > c.maxSize && c.tail != nil {
		c.evictOne()
	}

	return nil
}

func (c *radixCache) EraseEntriesWithGivenPrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.flushPromotions()
	c.eraseEntriesWithGivenPrefixInternal(prefix)
}

func (c *radixCache) eraseEntriesWithGivenPrefixInternal(prefix string) {
	// this check was done to comply with the existing behaviour of the EraseEntriesWithGivenPrefix function
	// the mapCache uses strings.HasPrefix which returns true for any string compared with ""
	if prefix == "" {
		if c.globalCurrentSize != nil && c.currentSize > 0 {
			c.globalCurrentSize.Add(-int64(c.currentSize))
		}
		for curr := c.head; curr != nil; curr = curr.next {
			curr.evicted = true
		}
		c.root = &radixNode{} // Reset the tree
		c.head = nil
		c.tail = nil
		c.currentSize = 0
		c.len = 0
		return
	}

	node := c.root
	search := prefix

	for len(search) > 0 {
		child := node.getChild(search[0])
		if child == nil {
			return // Prefix doesn't exist
		}

		lcp := longestCommonPrefix(search, child.prefix)

		if lcp == len(search) {
			// We found the exact node where the prefix ends.
			// Sever it entirely from the tree structure
			node.removeChild(child)

			// Now sweep the detached subtree to fix LRU and currentSize
			c.sweepAndUnlink(child)
			c.compressPathUpwards(node)
			return
		}

		if lcp == len(child.prefix) {
			search = search[len(child.prefix):]
			node = child
			continue
		}

		return
	}
}

// sweepAndUnlink recursively cleans up a detached subtree without triggering tree merges
func (c *radixCache) sweepAndUnlink(node *radixNode) {
	curr := node
	for curr != nil {

		if curr.value != nil {
			c.currentSize -= curr.size
			if c.globalCurrentSize != nil {
				c.globalCurrentSize.Add(-int64(curr.size))
			}
			c.remove(curr)
			curr.evicted = true
			curr.value = nil
			curr.size = 0
		}
		// Advance to next node in pre-order traversal
		if curr.child != nil {
			curr = curr.child
			continue
		}
		// Backtrack up parent chain until we find an ancestor with an unvisited sibling,
		// or we reach the root of the detached subtree.
		for curr != node && curr.sibling == nil {
			curr = curr.parent
		}
		if curr == node {
			return
		}
		curr = curr.sibling
	}
}
