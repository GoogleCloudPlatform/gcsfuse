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
	"strings"
)

// radixNode represents a node in the custom radix tree.
// It uses a Left-Child Right-Sibling (LCRS) representation to avoid slice allocations.
type radixNode struct {
	prefix  string
	value   ValueType
	parent  *radixNode
	child   *radixNode
	sibling *radixNode
	// LRU Linked List pointers
	prev *radixNode
	next *radixNode
}

// this radixTree struct will be replaced by a RadixCache struct afterwards with additional necessary fields
// radixTree encapsulates the core tree structure (LRU logic to be added in next PR).
type radixTree struct {
	root *radixNode
	size int
	// Head and tail of the LRU Doubly Linked List
	head *radixNode
	tail *radixNode
	len  int
}

func newRadixTree() *radixTree {
	return &radixTree{
		root: &radixNode{},
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

// insertNode inserts a new key into the radix tree and returns the leaf node.
func (t *radixTree) insertNode(key string, value ValueType) (*radixNode, bool) {
	if value == nil {
		return nil, false
	}

	node := t.root
	search := key

	for {
		if len(search) == 0 {
			isNew := node.value == nil
			if isNew {
				t.size++
			}
			node.value = value
			return node, isNew
		}

		child := node.getChild(search[0])
		if child == nil {
			// clone the substring to prevent memory leaks otherwise, the slice pins the entire original string in memory
			newLeaf := &radixNode{
				prefix: strings.Clone(search),
				value:  value,
			}
			node.addChild(newLeaf)
			t.size++
			return newLeaf, true
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

		for pcurr := &node.child; *pcurr != nil; pcurr = &(*pcurr).sibling {
			if *pcurr == child {
				splitNode.sibling = child.sibling
				*pcurr = splitNode
				break
			}
		}

		child.prefix = strings.Clone(child.prefix[lcp:])
		child.sibling = nil
		splitNode.addChild(child)

		if lcp == len(search) {
			splitNode.value = value
			t.size++
			return splitNode, true
		}

		newLeaf := &radixNode{
			prefix: strings.Clone(search[lcp:]),
			value:  value,
		}
		splitNode.addChild(newLeaf)
		t.size++
		return newLeaf, true
	}
}

// getNode finds a leaf node by key.
func (t *radixTree) getNode(key string) (*radixNode, bool) {
	node := t.root
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
func (t *radixTree) deleteNode(node *radixNode) {
	if node == nil || node.value == nil {
		return
	}
	t.remove(node) // Unlink from LRU list

	node.value = nil
	t.size--
	t.compressPathUpwards(node)
}

// compressPathUpwards walks up the tree, pruning empty leaves and compressing single-child routing nodes.
func (t *radixTree) compressPathUpwards(curr *radixNode) {
	for curr != nil && curr != t.root {
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

			for pcurr := &curr.parent.child; *pcurr != nil; pcurr = &(*pcurr).sibling {
				if *pcurr == curr {
					onlyChild.sibling = curr.sibling
					*pcurr = onlyChild
					break
				}
			}
			curr = curr.parent
			continue
		}
		break
	}
}

// --- LRU Logic ---
func (t *radixTree) moveToFront(node *radixNode) {
	if t.head == node {
		return
	}
	if node.prev != nil {
		node.prev.next = node.next
	}
	if node.next != nil {
		node.next.prev = node.prev
	}
	if t.tail == node {
		t.tail = node.prev
	}
	node.prev = nil
	node.next = t.head
	if t.head != nil {
		t.head.prev = node
	}
	t.head = node
	if t.tail == nil {
		t.tail = node
	}
}

func (t *radixTree) pushFront(node *radixNode) {
	node.prev = nil
	node.next = t.head
	if t.head != nil {
		t.head.prev = node
	}
	t.head = node
	if t.tail == nil {
		t.tail = node
	}
	t.len++
}

func (t *radixTree) remove(node *radixNode) {
	if t.head != node && node.prev == nil {
		return
	}
	if node.prev != nil {
		node.prev.next = node.next
	} else {
		t.head = node.next
	}
	if node.next != nil {
		node.next.prev = node.prev
	} else {
		t.tail = node.prev
	}
	node.prev = nil
	node.next = nil
	t.len--
}

func (t *radixTree) evictOne() ValueType {
	node := t.tail
	if node == nil {
		return nil
	}
	evictedEntry := node.value
	// Remove from LRU list and then fully delete from the tree
	t.remove(node)
	t.deleteNode(node)
	return evictedEntry
}
