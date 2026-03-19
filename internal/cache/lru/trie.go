// Copyright 2024 Google LLC
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
	"container/list"
	"strings"
)

// trieNode represents a node in the Radix Trie.
type trieNode struct {
	// children stores the edges to child nodes
	children []trieEdge
	// element points to the LRU list element, if this node represents a valid key
	element *list.Element
}

// trieEdge represents a directed edge to a child node with a string prefix.
type trieEdge struct {
	prefix string
	node   *trieNode
}

// pathTrie is a Radix Trie tailored for replacing map[string]*list.Element.
// It achieves memory savings via prefix deduplication, and fast O(1) matching subtree deletions.
type pathTrie struct {
	root *trieNode
	size int
}

func newPathTrie() *pathTrie {
	return &pathTrie{
		root: &trieNode{},
	}
}

// longestCommonPrefix returns the length of the longest common prefix between two strings.
func longestCommonPrefix(a, b string) int {
	i := 0
	max := len(a)
	if len(b) < max {
		max = len(b)
	}
	for i < max && a[i] == b[i] {
		i++
	}
	return i
}

// insert inserts a key and its corresponding list element into the trie.
func (t *pathTrie) insert(key string, element *list.Element) {
	node := t.root
	search := key

	for {
		// If search string is empty, we reached the target node.
		if len(search) == 0 {
			if node.element == nil {
				t.size++
			}
			node.element = element
			return
		}

		// Look for a matching edge
		var matchEdgeIdx = -1
		var lcp = 0
		for i, edge := range node.children {
			l := longestCommonPrefix(search, edge.prefix)
			if l > 0 {
				matchEdgeIdx = i
				lcp = l
				break
			}
		}

		if matchEdgeIdx == -1 {
			// No matching edge, create a new one for the remainder of the search string.
			newNode := &trieNode{
				element: element,
			}
			node.children = append(node.children, trieEdge{prefix: search, node: newNode})
			t.size++
			return
		}

		edge := node.children[matchEdgeIdx]

		if lcp == len(edge.prefix) {
			// Search string fully contains the edge prefix. Go down this edge.
			search = search[lcp:]
			node = edge.node
			continue
		}

		// The common prefix is shorter than the edge prefix. We must split the edge.
		// For example, if edge is "abc", and search is "abd", lcp is 2 ("ab").
		// We split "abc" into "ab" -> "c", and add a new branch "ab" -> "d".
		splitNode := &trieNode{
			children: []trieEdge{
				{prefix: edge.prefix[lcp:], node: edge.node},
			},
		}

		// The edge from the current node now points to the split node with the common prefix.
		node.children[matchEdgeIdx] = trieEdge{prefix: search[:lcp], node: splitNode}

		// Now add the remainder of the search string as a new child of the split node.
		if lcp == len(search) {
			// The search string ends exactly at the split point.
			splitNode.element = element
			t.size++
			return
		}

		newNode := &trieNode{
			element: element,
		}
		splitNode.children = append(splitNode.children, trieEdge{prefix: search[lcp:], node: newNode})
		t.size++
		return
	}
}

// lookup searches for the given key and returns the element and true if found.
func (t *pathTrie) lookup(key string) (*list.Element, bool) {
	node := t.root
	search := key

	for len(search) > 0 {
		found := false
		for _, edge := range node.children {
			if strings.HasPrefix(search, edge.prefix) {
				search = search[len(edge.prefix):]
				node = edge.node
				found = true
				break
			}
		}
		if !found {
			return nil, false
		}
	}

	if node.element != nil {
		return node.element, true
	}
	return nil, false
}

// delete removes the element associated with the given key.
// It also prunes empty nodes bottom-up to save memory.
// Returns the deleted element, or nil if not found.
func (t *pathTrie) delete(key string) *list.Element {
	// We need to keep track of the path to prune empty nodes bottom-up.
	type pathItem struct {
		parent *trieNode
		idx    int
	}
	var path []pathItem

	node := t.root
	search := key

	for len(search) > 0 {
		found := false
		for i, edge := range node.children {
			if strings.HasPrefix(search, edge.prefix) {
				path = append(path, pathItem{parent: node, idx: i})
				search = search[len(edge.prefix):]
				node = edge.node
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}

	if node.element == nil {
		return nil
	}

	el := node.element
	node.element = nil
	t.size--

	// Prune bottom-up
	curr := node
	for i := len(path) - 1; i >= 0; i-- {
		p := path[i]
		if curr.element == nil && len(curr.children) == 0 {
			// Remove current node from its parent
			last := len(p.parent.children) - 1
			p.parent.children[p.idx] = p.parent.children[last]
			p.parent.children = p.parent.children[:last]
		} else if curr.element == nil && len(curr.children) == 1 {
			// Merge current node with its single child to compress the tree.
			childEdge := curr.children[0]
			p.parent.children[p.idx].prefix += childEdge.prefix
			p.parent.children[p.idx].node = childEdge.node
		}
		curr = p.parent
	}

	return el
}

// collectElements recursively collects all valid elements in the subtree.
func (t *pathTrie) collectElements(node *trieNode, elements *[]*list.Element) {
	if node.element != nil {
		*elements = append(*elements, node.element)
		t.size--
	}
	for _, edge := range node.children {
		t.collectElements(edge.node, elements)
	}
}

// erasePrefix removes all elements whose keys start with the given prefix,
// and returns all deleted elements.
func (t *pathTrie) erasePrefix(prefix string) []*list.Element {
	// Track path for pruning
	type pathItem struct {
		parent *trieNode
		idx    int
	}
	var path []pathItem

	node := t.root
	search := prefix

	// Walk down to the node representing the prefix, or where the prefix ends.
	for len(search) > 0 {
		found := false
		for i, edge := range node.children {
			lcp := longestCommonPrefix(search, edge.prefix)
			if lcp > 0 {
				path = append(path, pathItem{parent: node, idx: i})
				if lcp == len(search) {
					// The prefix ends inside this edge.
					// This node's entire subtree matches the prefix.
					node = edge.node
					search = ""
					found = true
					break
				} else if lcp == len(edge.prefix) {
					// The edge is fully consumed by the prefix.
					search = search[lcp:]
					node = edge.node
					found = true
					break
				} else {
					// The prefix diverges from the edge. This means no keys in the trie
					// can possibly have this prefix (because they follow this edge which differs).
					return nil
				}
			}
		}
		if !found {
			return nil
		}
	}

	var deleted []*list.Element
	t.collectElements(node, &deleted)

	// Prune the subtree
	node.children = nil
	node.element = nil

	// Prune bottom-up
	curr := node
	for i := len(path) - 1; i >= 0; i-- {
		p := path[i]
		if curr.element == nil && len(curr.children) == 0 {
			// Remove current node from its parent
			last := len(p.parent.children) - 1
			p.parent.children[p.idx] = p.parent.children[last]
			p.parent.children = p.parent.children[:last]
		} else if curr.element == nil && len(curr.children) == 1 {
			// Merge current node with its single child.
			childEdge := curr.children[0]
			p.parent.children[p.idx].prefix += childEdge.prefix
			p.parent.children[p.idx].node = childEdge.node
		}
		curr = p.parent
	}

	return deleted
}

// len returns the number of valid elements in the trie.
func (t *pathTrie) len() int {
	return t.size
}
