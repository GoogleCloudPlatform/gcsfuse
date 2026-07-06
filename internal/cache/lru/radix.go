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
}

// this radixTree struct will be replaced by a RadixCache struct afterwards with additional necessary fields
// radixTree encapsulates the core tree structure (LRU logic to be added in next PR).
type radixTree struct {
	root *radixNode
	size int
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
	}
	return nil
}

// addChild adds a child node, maintaining sorted order by the first byte of the prefix.
func (n *radixNode) addChild(newChild *radixNode) {
	newChild.parent = n
	newChild.sibling = nil

	if n.child == nil {
		n.child = newChild
		return
	}

	if newChild.prefix[0] < n.child.prefix[0] {
		newChild.sibling = n.child
		n.child = newChild
		return
	}

	curr := n.child
	for curr.sibling != nil && curr.sibling.prefix[0] < newChild.prefix[0] {
		curr = curr.sibling
	}

	newChild.sibling = curr.sibling
	curr.sibling = newChild
}

// removeChild directly removes a child node by reference from the sibling list.
func (n *radixNode) removeChild(childToRemove *radixNode) {
	if n.child == childToRemove {
		n.child = childToRemove.sibling
		return
	}

	curr := n.child
	for curr != nil && curr.sibling != childToRemove {
		curr = curr.sibling
	}

	if curr != nil {
		curr.sibling = childToRemove.sibling
	}
}

// insertNode inserts a new key into the radix tree and returns the leaf node.
func (t *radixTree) insertNode(key string, value ValueType) (*radixNode, bool) {
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

		if node.child == child {
			splitNode.sibling = child.sibling
			node.child = splitNode
		} else {
			curr := node.child
			for curr != nil && curr.sibling != child {
				curr = curr.sibling
			}
			if curr != nil {
				splitNode.sibling = child.sibling
				curr.sibling = splitNode
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

			if curr.parent.child == curr {
				onlyChild.sibling = curr.sibling
				curr.parent.child = onlyChild
			} else {
				prev := curr.parent.child
				for prev != nil && prev.sibling != curr {
					prev = prev.sibling
				}
				if prev != nil {
					onlyChild.sibling = curr.sibling
					prev.sibling = onlyChild
				}
			}
			curr = curr.parent
			continue
		}
		break
	}
}
