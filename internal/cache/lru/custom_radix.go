package lru

import (
	"container/list"
	"strings"
)

// radixNode represents a node in the custom radix tree.
type radixNode struct {
	// prefix is the string fragment stored at this node.
	prefix string

	// children contains the child nodes, sorted by their prefix's first byte.
	children []*radixNode

	// parent points to the parent node to facilitate O(1) bottom-up deletion.
	parent *radixNode

	// element is the pointer to the LRU linked list element.
	// If element is nil, this is an internal routing node, not a stored key.
	element *list.Element
}

// radixTree is a custom implementation of a radix tree (compressed trie).
type radixTree struct {
	root *radixNode
	size int
}

func newRadixTree() *radixTree {
	return &radixTree{
		root: &radixNode{},
	}
}

func (t *radixTree) Len() int {
	return t.size
}

// longestCommonPrefix finds the length of the longest common prefix of a and b.
func longestCommonPrefix(a, b string) int {
	i := 0
	maxLen := len(a)
	if len(b) < maxLen {
		maxLen = len(b)
	}
	for i < maxLen && a[i] == b[i] {
		i++
	}
	return i
}

// getChild finds a child node whose prefix starts with the given byte.
// It also returns the index of the child in the children slice.
func (n *radixNode) getChild(b byte) (*radixNode, int) {
	// We can use binary search if children are sorted, but linear search
	// is usually fast enough for a small number of children (max 256).
	for i, child := range n.children {
		if child.prefix[0] == b {
			return child, i
		}
	}
	return nil, -1
}

// addChild adds a child node, maintaining the sorted order by the first byte of the prefix.
func (n *radixNode) addChild(child *radixNode) {
	child.parent = n
	// Insert in sorted order
	for i, c := range n.children {
		if child.prefix[0] < c.prefix[0] {
			// Insert at i
			n.children = append(n.children[:i], append([]*radixNode{child}, n.children[i:]...)...)
			return
		}
	}
	n.children = append(n.children, child)
}

// removeChild removes a child node at the given index.
func (n *radixNode) removeChild(index int) {
	n.children = append(n.children[:index], n.children[index+1:]...)
}

// Insert inserts a new key into the radix tree and returns the leaf node.
// If the key already exists, it updates the element and returns the existing leaf node.
func (t *radixTree) Insert(key string, element *list.Element) (*radixNode, bool) {
	node := t.root
	search := key

	for {
		if len(search) == 0 {
			// Exact match with an existing node
			isNew := node.element == nil
			if isNew {
				t.size++
			}
			node.element = element
			return node, isNew
		}

		child, _ := node.getChild(search[0])
		if child == nil {
			// No matching child, create a new leaf
			newLeaf := &radixNode{
				prefix:  search,
				element: element,
			}
			node.addChild(newLeaf)
			t.size++
			return newLeaf, true
		}

		// Found a matching child, find the longest common prefix
		lcp := longestCommonPrefix(search, child.prefix)

		if lcp == len(child.prefix) {
			// The search key contains the entire child prefix.
			// Consume the prefix and continue down the tree.
			search = search[lcp:]
			node = child
			continue
		}

		// We need to split the child node.
		// Create a new split node.
		splitNode := &radixNode{
			prefix: child.prefix[:lcp],
			parent: node,
		}

		// Replace the child in the parent's children slice with the split node.
		for i, c := range node.children {
			if c == child {
				node.children[i] = splitNode
				break
			}
		}

		// Adjust the old child
		child.prefix = child.prefix[lcp:]
		splitNode.addChild(child)

		if lcp == len(search) {
			// The search key ends exactly at the split point.
			// The split node becomes the leaf.
			splitNode.element = element
			t.size++
			return splitNode, true
		}

		// The search key continues past the split point.
		// Create a new leaf node for the remaining search key.
		newLeaf := &radixNode{
			prefix:  search[lcp:],
			element: element,
		}
		splitNode.addChild(newLeaf)
		t.size++
		return newLeaf, true
	}
}

// Get finds a leaf node by key.
func (t *radixTree) Get(key string) (*radixNode, bool) {
	node := t.root
	search := key

	for {
		if len(search) == 0 {
			if node.element != nil {
				return node, true
			}
			return nil, false
		}

		child, _ := node.getChild(search[0])
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

// DeleteNode removes a leaf node directly using its parent pointers.
// It merges the parent with its remaining child if the parent becomes empty and has only 1 child.
func (t *radixTree) DeleteNode(node *radixNode) {
	if node == nil || node.element == nil {
		// Not a leaf node, or already deleted.
		return
	}

	// Mark as an internal node (or deleted leaf)
	node.element = nil
	t.size--

	// If the node has children, it must remain as an internal routing node.
	if len(node.children) > 0 {
		// If it has exactly 1 child, we can merge it.
		if len(node.children) == 1 && node != t.root {
			child := node.children[0]
			child.prefix = node.prefix + child.prefix
			child.parent = node.parent
			for i, c := range node.parent.children {
				if c == node {
					node.parent.children[i] = child
					break
				}
			}
		}
		return
	}

	// The node has no children. It can be completely removed.
	// We need to traverse up and prune empty internal nodes.
	curr := node
	for curr != t.root {
		parent := curr.parent

		// Find curr in parent's children and remove it
		for i, c := range parent.children {
			if c == curr {
				parent.removeChild(i)
				break
			}
		}

		// Check if the parent needs to be merged or kept.
		if parent.element != nil {
			// Parent is a valid leaf, stop pruning.
			break
		}

		if len(parent.children) > 1 {
			// Parent has other branches, stop pruning.
			break
		}

		if len(parent.children) == 1 && parent != t.root {
			// Parent has exactly 1 child left and is not a leaf.
			// Merge parent with its remaining child.
			onlyChild := parent.children[0]
			onlyChild.prefix = parent.prefix + onlyChild.prefix
			onlyChild.parent = parent.parent
			for i, c := range parent.parent.children {
				if c == parent {
					parent.parent.children[i] = onlyChild
					break
				}
			}
			break
		}

		if len(parent.children) == 0 {
			// Parent is now empty (no children, no value).
			// It should be pruned in the next iteration.
			curr = parent
		} else {
			break
		}
	}
}

// WalkPrefix traverses the tree and calls fn for every leaf node that starts with the given prefix.
func (t *radixTree) WalkPrefix(prefix string, fn func(*radixNode) bool) {
	node := t.root
	search := prefix

	// Find the node where the prefix ends.
	for len(search) > 0 {
		child, _ := node.getChild(search[0])
		if child == nil {
			return // Prefix not found
		}

		lcp := longestCommonPrefix(search, child.prefix)

		if lcp == len(child.prefix) {
			search = search[len(child.prefix):]
			node = child
			continue
		}

		if lcp == len(search) {
			walk(child, fn)
		}

		return
	}

	// The prefix matched exactly at this node boundary. Walk all descendants.
	walk(node, fn)
}

// walk recursively visits all leaf nodes in the subtree rooted at node.
func walk(node *radixNode, fn func(*radixNode) bool) bool {
	if node.element != nil {
		if fn(node) {
			return true
		}
	}

	// Create a safe copy of the children slice because fn might delete the current child
	// from the radix tree, which mutates the parent's children slice.
	// Wait, walk is called under an RLock! We cannot mutate the tree here!
	// So we don't need a copy for safety against tree mutation during Walk.
	// However, we DO need a copy if fn modifies the list. But WalkPrefix only populates nodesToDelete.

	for _, child := range node.children {
		if walk(child, fn) {
			return true
		}
	}
	return false
}
