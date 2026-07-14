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
	"math"
	"strings"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/locker"
)

const nilNode uint32 = math.MaxUint32

// arenaRadixNode represents a node in the custom radix tree.
// It uses a Left-Child Right-Sibling (LCRS) representation and 32-bit indices to avoid slice allocations and pointer overhead.
type arenaRadixNode struct {
	prefix  string
	value   ValueType
	parent  uint32
	child   uint32
	sibling uint32
	// LRU Linked List pointers
	prev uint32
	next uint32
}

// arenaRadix encapsulates the core tree structure and implements the lru.Cache interface.
type arenaRadix struct {
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

	nodes    []arenaRadixNode
	freeHead uint32

	nodeMap map[uint64]uint32

	root uint32

	// Head and tail of the LRU Doubly Linked List
	head uint32
	tail uint32

	// len is the number of nodes currently linked in the LRU list.
	len int

	// All public methods of this Cache uses this RW mutex based locker while
	// accessing/updating Cache's data.
	mu locker.RWLocker
}

// FNV-1a 64-bit hash
func hashString(s string) uint64 {
	var h uint64 = 14695981039346656037
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= 1099511628211
	}
	return h
}

func (c *arenaRadix) getFullKey(nodeID uint32) string {
	var parts []string
	curr := nodeID
	for curr != c.root && curr != nilNode {
		parts = append(parts, c.nodes[curr].prefix)
		curr = c.nodes[curr].parent
	}
	var sb strings.Builder
	for i := len(parts) - 1; i >= 0; i-- {
		sb.WriteString(parts[i])
	}
	return sb.String()
}

func (c *arenaRadix) allocateNode() uint32 {
	if c.freeHead != nilNode {
		id := c.freeHead
		c.freeHead = c.nodes[id].next
		c.nodes[id] = arenaRadixNode{
			parent:  nilNode,
			child:   nilNode,
			sibling: nilNode,
			prev:    nilNode,
			next:    nilNode,
		}
		return id
	}
	id := uint32(len(c.nodes))
	c.nodes = append(c.nodes, arenaRadixNode{
		parent:  nilNode,
		child:   nilNode,
		sibling: nilNode,
		prev:    nilNode,
		next:    nilNode,
	})
	return id
}

func (c *arenaRadix) freeNode(id uint32) {
	n := &c.nodes[id]
	n.prefix = ""
	n.value = nil
	n.parent = nilNode
	n.child = nilNode
	n.sibling = nilNode
	n.prev = nilNode

	n.next = c.freeHead
	c.freeHead = id
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

// getChild finds a child node whose prefix starts with the given byte.
func (c *arenaRadix) getChild(nID uint32, b byte) uint32 {
	n := &c.nodes[nID]
	for curr := n.child; curr != nilNode; curr = c.nodes[curr].sibling {
		if c.nodes[curr].prefix[0] == b {
			return curr
		}
		//as the siblings are arranged in lexicographical order we can exit early
		if c.nodes[curr].prefix[0] > b {
			return nilNode
		}
	}
	return nilNode
}

// addChild adds a child node, maintaining sorted order by the first byte of the prefix.
func (c *arenaRadix) addChild(nID uint32, newChildID uint32) {
	c.nodes[newChildID].parent = nID
	pcurr := &c.nodes[nID].child
	for *pcurr != nilNode && c.nodes[*pcurr].prefix[0] < c.nodes[newChildID].prefix[0] {
		pcurr = &c.nodes[*pcurr].sibling
	}
	c.nodes[newChildID].sibling = *pcurr
	*pcurr = newChildID
}

// removeChild directly removes a child node by reference from the sibling list.
func (c *arenaRadix) removeChild(nID uint32, childToRemoveID uint32) {
	for pcurr := &c.nodes[nID].child; *pcurr != nilNode; pcurr = &c.nodes[*pcurr].sibling {
		if *pcurr == childToRemoveID {
			*pcurr = c.nodes[childToRemoveID].sibling

			c.nodes[childToRemoveID].sibling = nilNode
			c.nodes[childToRemoveID].parent = nilNode

			return
		}
	}
}

// replaceChild finds oldChild in the sibling linked-list and replaces it with newChild,
// seamlessly preserving the rest of the sibling chain.
func (c *arenaRadix) replaceChild(nID uint32, oldChildID uint32, newChildID uint32) {
	for pcurr := &c.nodes[nID].child; *pcurr != nilNode; pcurr = &c.nodes[*pcurr].sibling {
		if *pcurr == oldChildID {
			c.nodes[newChildID].sibling = c.nodes[oldChildID].sibling
			*pcurr = newChildID

			c.nodes[oldChildID].sibling = nilNode
			c.nodes[oldChildID].parent = nilNode

			return
		}
	}
}

// insertNode inserts a new key into the radix tree and returns the leaf node ID.
func (c *arenaRadix) insertNode(key string, value ValueType) (uint32, ValueType) {
	if value == nil {
		return nilNode, nil
	}

	nodeID := c.root
	search := key

	for {
		if len(search) == 0 {
			oldValue := c.nodes[nodeID].value
			c.nodes[nodeID].value = value
			return nodeID, oldValue
		}

		childID := c.getChild(nodeID, search[0])
		if childID == nilNode {
			// clone the substring to prevent memory leaks otherwise, the slice pins the entire original string in memory
			newLeafID := c.allocateNode()
			c.nodes[newLeafID].prefix = strings.Clone(search)
			c.nodes[newLeafID].value = value
			c.addChild(nodeID, newLeafID)
			return newLeafID, nil
		}

		lcp := longestCommonPrefix(search, c.nodes[childID].prefix)

		if lcp == len(c.nodes[childID].prefix) {
			search = search[lcp:]
			nodeID = childID
			continue
		}

		splitNodeID := c.allocateNode()
		c.nodes[splitNodeID].prefix = strings.Clone(c.nodes[childID].prefix[:lcp])
		c.nodes[splitNodeID].parent = nodeID

		c.replaceChild(nodeID, childID, splitNodeID)

		c.nodes[childID].prefix = strings.Clone(c.nodes[childID].prefix[lcp:])
		c.addChild(splitNodeID, childID)

		if lcp == len(search) {
			oldValue := c.nodes[splitNodeID].value
			c.nodes[splitNodeID].value = value
			return splitNodeID, oldValue
		}

		newLeafID := c.allocateNode()
		c.nodes[newLeafID].prefix = strings.Clone(search[lcp:])
		c.nodes[newLeafID].value = value
		c.addChild(splitNodeID, newLeafID)
		return newLeafID, nil
	}
}

// verifyKey checks if the node's full path matches the given key with ZERO allocations.
func (c *arenaRadix) verifyKey(nodeID uint32, key string) bool {
	curr := nodeID
	end := len(key)

	for curr != c.root && curr != nilNode {
		prefix := c.nodes[curr].prefix
		prefixLen := len(prefix)

		if end < prefixLen {
			return false
		}
		if key[end-prefixLen:end] != prefix {
			return false
		}
		end -= prefixLen
		curr = c.nodes[curr].parent
	}

	return end == 0
}

// getNodeKey finds a leaf node ID by key.
func (c *arenaRadix) getNodeKey(key string) (uint32, bool) {
	nodeID, ok := c.nodeMap[hashString(key)]
	if ok && c.nodes[nodeID].value != nil {
		if c.verifyKey(nodeID, key) {
			return nodeID, true
		}
	}
	return nilNode, false
}

// deleteNode removes a leaf node directly using its parent pointers.
func (c *arenaRadix) deleteNode(nodeID uint32) {
	if nodeID == nilNode || c.nodes[nodeID].value == nil {
		return
	}

	c.nodes[nodeID].value = nil
	c.compressPathUpwards(nodeID)
}

// compressPathUpwards walks up the tree, pruning empty leaves and compressing single-child routing nodes.
func (c *arenaRadix) compressPathUpwards(currID uint32) {
	for currID != nilNode && currID != c.root {
		if c.nodes[currID].value != nil {
			break
		}

		if c.nodes[currID].child == nilNode {
			parentID := c.nodes[currID].parent
			c.removeChild(parentID, currID)
			c.freeNode(currID)
			currID = parentID
			continue
		}

		if c.nodes[c.nodes[currID].child].sibling == nilNode {
			onlyChildID := c.nodes[currID].child
			c.nodes[onlyChildID].prefix = c.nodes[currID].prefix + c.nodes[onlyChildID].prefix
			c.nodes[onlyChildID].parent = c.nodes[currID].parent

			parentID := c.nodes[currID].parent
			c.replaceChild(parentID, currID, onlyChildID)

			c.freeNode(currID)
			currID = parentID
			continue
		}
		break
	}
}
