// Copyright 2026 Google LLC
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

	"github.com/googlecloudplatform/gcsfuse/v3/internal/locker"
)

// NewArenaRadixCache returns the reference of cache object by initialising the cache with
// the supplied maxSize, which must be greater than zero.
func NewArenaRadixCache(maxSize uint64) Cache {
	if maxSize == 0 {
		panic("Invalid maxSize")
	}

	c := &arenaRadix{
		maxSize:  maxSize,
		freeHead: nilNode,
		head:     nilNode,
		tail:     nilNode,
		nodeMap:  make(map[uint64]uint32),
	}

	c.root = c.allocateNode()
	c.mu = locker.New("ArenaRadixCache", c.checkInvariants)
	return c
}

func (c *arenaRadix) checkInvariants() {
	// INVARIANT: currentSize <= maxSize
	if c.currentSize > c.maxSize {
		panic(fmt.Sprintf("CurrentSize %v over maxSize %v", c.currentSize, c.maxSize))
	}

	// INVARIANT: Each element in the LRU list must have a valid value
	lruCount := 0
	for currID := c.head; currID != nilNode; currID = c.nodes[currID].next {
		lruCount++
		if c.nodes[currID].value == nil {
			panic(fmt.Sprintf("Unexpected empty value in LRU list for prefix: %v", c.nodes[currID].prefix))
		}
	}

	if lruCount != c.len {
		panic(fmt.Sprintf("LRU list actual count %v does not match c.len %v", lruCount, c.len))
	}

	// INVARIANT: Every value-bearing node in the tree must exist in the LRU list exactly once
	treeCount := 0

	// Iterative pre-order traversal using parent/sibling pointers (O(1) space) to prevent stack overflows
	currID := c.root
	for currID != nilNode {
		if c.nodes[currID].value != nil {
			treeCount++
			// A node is verifiably in the LRU list if it is the head, or if it has a predecessor
			inLRU := c.head == currID || c.nodes[currID].prev != nilNode
			if !inLRU {
				panic(fmt.Sprintf("Mismatch: Node with prefix '%v' has a value but is missing from LRU list", c.nodes[currID].prefix))
			}
		}

		// Advance to next node
		if c.nodes[currID].child != nilNode {
			currID = c.nodes[currID].child
			continue
		}

		// Backtrack up parent chain
		for currID != c.root && c.nodes[currID].sibling == nilNode {
			currID = c.nodes[currID].parent
		}
		if currID == c.root {
			break
		}
		currID = c.nodes[currID].sibling
	}

	if treeCount != c.len {
		panic(fmt.Sprintf("Tree actual value count %v does not match LRU length %v", treeCount, c.len))
	}
}

////////////////////////////////////////////////////////////////////////
// Cache interface
////////////////////////////////////////////////////////////////////////

// Insert the supplied value into the cache, overwriting any previous entry for
// the given key. The value must be non-nil.
// Also returns a slice of ValueType evicted by the new inserted entry.
func (c *arenaRadix) Insert(key string, value ValueType) ([]ValueType, error) {
	if value == nil {
		return nil, ErrInvalidEntry
	}

	valueSize := value.Size()
	if valueSize > c.maxSize {
		return nil, ErrInvalidEntrySize
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	var evictedValues []ValueType

	// A single insert can allocate up to 2 nodes (one routing node, one leaf).
	// If the slice has reached its physical uint32 maximum, we must rely entirely on the freelist.
	// Since evicting one LRU item guarantees at least 1 leaf node (and potentially 1 routing node)
	// is freed to the freelist, we proactively evict until we have at least 2 free nodes.
	for uint32(len(c.nodes)) >= nilNode-2 && c.tail != nilNode && (c.freeHead == nilNode || c.nodes[c.freeHead].next == nilNode) {
		evictedValues = append(evictedValues, c.evictOne())
	}

	if nodeID, oldValue := c.insertNode(key, value); oldValue != nil {
		c.currentSize += valueSize - oldValue.Size()
		c.nodeMap[hashString(key)] = nodeID
		c.moveToFront(nodeID)
	} else {
		c.pushFront(nodeID)
		c.currentSize += valueSize
		c.nodeMap[hashString(key)] = nodeID
	}

	// Evict until we're at or below maxSize
	for c.currentSize > c.maxSize && c.tail != nilNode {
		evictedValues = append(evictedValues, c.evictOne())
	}

	return evictedValues, nil
}

// Erase any entry for the supplied key, also returns the value of erased key.
func (c *arenaRadix) Erase(key string) (value ValueType) {
	c.mu.Lock()
	defer c.mu.Unlock()

	nodeID, ok := c.getNodeKey(key)
	if !ok {
		return nil
	}

	return c.eraseInternal(nodeID)
}

// LookUp a previously-inserted value for the given key. Return nil if no
// value is present.
func (c *arenaRadix) LookUp(key string) (value ValueType) {
	c.mu.Lock()
	defer c.mu.Unlock()

	nodeID, ok := c.getNodeKey(key)
	if !ok {
		return nil
	}
	c.moveToFront(nodeID)

	return c.nodes[nodeID].value
}

// LookUpWithoutChangingOrder looks up previously-inserted value for a given key
// without changing the order of entries in the cache. Return nil if no value
// is present.
//
// Note: Even though this lookup doesn't change the MRU order, we must acquire a
// write lock because getNodeKey can lazily mutate the internal nodeMap cache.
func (c *arenaRadix) LookUpWithoutChangingOrder(key string) (value ValueType) {
	c.mu.Lock()
	defer c.mu.Unlock()

	nodeID, ok := c.getNodeKey(key)
	if !ok {
		return nil
	}

	return c.nodes[nodeID].value
}

// UpdateWithoutChangingOrder updates entry with the given key in cache with
// given value without changing order of entries in cache, returning error if an
// entry with given key doesn't exist. Also, the size of value for entry
// shouldn't be updated with this method (use c.Insert for updating size).
func (c *arenaRadix) UpdateWithoutChangingOrder(key string, value ValueType) error {
	if value == nil {
		return ErrInvalidEntry
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	nodeID, ok := c.getNodeKey(key)
	if !ok {
		return ErrEntryNotExist
	}

	if value.Size() != c.nodes[nodeID].value.Size() {
		return ErrInvalidUpdateEntrySize
	}

	c.nodes[nodeID].value = value
	return nil
}

// UpdateSize updates the currentSize accounting when an entry's size has changed.
// This is needed for entries whose size grows incrementally (e.g., sparse files).
// The entry's order in the LRU is not changed.
func (c *arenaRadix) UpdateSize(key string, sizeDelta uint64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, ok := c.getNodeKey(key)
	if !ok {
		return ErrEntryNotExist
	}

	// Update currentSize accounting
	c.currentSize += sizeDelta

	// Evict until we're at or below maxSize to maintain invariants
	for c.currentSize > c.maxSize && c.tail != nilNode {
		c.evictOne()
	}

	return nil
}

func (c *arenaRadix) EraseEntriesWithGivenPrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if prefix == "" {
		c.nodes = nil
		c.freeHead = nilNode

		c.root = c.allocateNode()
		c.head = nilNode
		c.tail = nilNode
		c.currentSize = 0
		c.len = 0
		clear(c.nodeMap)
		return
	}

	nodeID := c.root
	search := prefix

	for len(search) > 0 {
		childID := c.getChild(nodeID, search[0])
		if childID == nilNode {
			return // Prefix doesn't exist
		}

		lcp := longestCommonPrefix(search, c.nodes[childID].prefix)

		if lcp == len(search) {
			// We found the exact node where the prefix ends.
			// Sever it entirely from the tree structure
			c.removeChild(nodeID, childID)

			// temporarily restore upward parent pointer
			c.nodes[childID].parent = nodeID

			// Now sweep the detached subtree to fix LRU and currentSize
			c.freeSubtree(childID)
			c.compressPathUpwards(nodeID)
			return
		}

		if lcp == len(c.nodes[childID].prefix) {
			search = search[len(c.nodes[childID].prefix):]
			nodeID = childID
			continue
		}

		return
	}
}

// freeSubtree recursively cleans up a detached subtree and correctly frees every node
func (c *arenaRadix) freeSubtree(nodeID uint32) {
	if nodeID == nilNode {
		return
	}
	currID := nodeID
	for currID != nilNode {
		if c.nodes[currID].value != nil {
			c.currentSize -= c.nodes[currID].value.Size()
			c.remove(currID)
			hash := hashString(c.getFullKey(currID))
			if c.nodeMap[hash] == currID {
				delete(c.nodeMap, hash)
			}
			c.nodes[currID].value = nil
		}

		if c.nodes[currID].child != nilNode {
			currID = c.nodes[currID].child
			continue
		}

		for currID != nodeID && c.nodes[currID].sibling == nilNode {
			parentID := c.nodes[currID].parent
			c.freeNode(currID)
			currID = parentID
		}

		if currID == nodeID {
			c.freeNode(currID)
			return
		}

		siblingID := c.nodes[currID].sibling
		c.freeNode(currID)
		currID = siblingID
	}
}
