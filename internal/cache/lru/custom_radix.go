package lru

import (
	"strings"
)

// radixNode represents a node in the custom radix tree.
// It embeds the LRU doubly linked list pointers to avoid separate allocations.
type radixNode struct {
	// prefix is the string fragment stored at this node.
	prefix string

	// value stores the cache entry value. If nil, this is an internal routing node.
	value ValueType

	// Tree pointers (Left-Child Right-Sibling representation to avoid slice allocations)
	parent  *radixNode
	child   *radixNode // points to the first child
	sibling *radixNode // points to the next sibling

	// LRU Linked List pointers
	prev *radixNode
	next *radixNode
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
// We iterate over the sibling linked list.
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
	newChild.sibling = nil // ensure it's clean

	if n.child == nil {
		n.child = newChild
		return
	}

	// Insert before first child
	if newChild.prefix[0] < n.child.prefix[0] {
		newChild.sibling = n.child
		n.child = newChild
		return
	}

	// Insert after a sibling
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

// Insert inserts a new key into the radix tree and returns the leaf node.
// If the key already exists, it updates the value and returns the existing leaf node.
func (t *radixTree) Insert(key string, value ValueType) (*radixNode, bool) {
	node := t.root
	search := key

	for {
		if len(search) == 0 {
			// Exact match with an existing node
			isNew := node.value == nil
			if isNew {
				t.size++
			}
			node.value = value
			return node, isNew
		}

		child := node.getChild(search[0])
		if child == nil {
			// No matching child, create a new leaf
			newLeaf := &radixNode{
				prefix: search,
				value:  value,
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

		// Replace the child in the parent's sibling list with the split node.
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

		// Adjust the old child
		child.prefix = child.prefix[lcp:]
		child.sibling = nil // it is now the ONLY child of splitNode temporarily
		splitNode.addChild(child)

		if lcp == len(search) {
			// The search key ends exactly at the split point.
			// The split node becomes the leaf.
			splitNode.value = value
			t.size++
			return splitNode, true
		}

		// The search key continues past the split point.
		// Create a new leaf node for the remaining search key.
		newLeaf := &radixNode{
			prefix: search[lcp:],
			value:  value,
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

// numChildren counts the number of children a node has by walking the sibling list.
func (n *radixNode) numChildren() int {
	count := 0
	for curr := n.child; curr != nil; curr = curr.sibling {
		count++
	}
	return count
}

// DeleteNode removes a leaf node directly using its parent pointers.
// It merges the parent with its remaining child if the parent becomes empty and has only 1 child.
func (t *radixTree) DeleteNode(node *radixNode) {
	if node == nil || node.value == nil {
		// Not a leaf node, or already deleted.
		return
	}

	// Mark as an internal node (or deleted leaf)
	node.value = nil
	t.size--

	// If the node has children, it must remain as an internal routing node.
	if node.child != nil {
		// If it has exactly 1 child, we can merge it.
		if node.child.sibling == nil && node != t.root {
			child := node.child
			child.prefix = node.prefix + child.prefix
			child.parent = node.parent

			// Replace node with child in node's parent's sibling list
			if node.parent.child == node {
				child.sibling = node.sibling
				node.parent.child = child
			} else {
				curr := node.parent.child
				for curr != nil && curr.sibling != node {
					curr = curr.sibling
				}
				if curr != nil {
					child.sibling = node.sibling
					curr.sibling = child
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
		parent.removeChild(curr)

		// Check if the parent needs to be merged or kept.
		if parent.value != nil {
			// Parent is a valid leaf, stop pruning.
			break
		}

		// Check if parent has more than 1 branch
		if parent.child != nil && parent.child.sibling != nil {
			// Parent has multiple branches, stop pruning.
			break
		}

		if parent.child != nil && parent.child.sibling == nil && parent != t.root {
			// Parent has exactly 1 child left and is not a leaf.
			// Merge parent with its remaining child.
			onlyChild := parent.child
			onlyChild.prefix = parent.prefix + onlyChild.prefix
			onlyChild.parent = parent.parent

			// Replace parent with onlyChild in parent's parent's sibling list
			if parent.parent.child == parent {
				onlyChild.sibling = parent.sibling
				parent.parent.child = onlyChild
			} else {
				currParentSibling := parent.parent.child
				for currParentSibling != nil && currParentSibling.sibling != parent {
					currParentSibling = currParentSibling.sibling
				}
				if currParentSibling != nil {
					onlyChild.sibling = parent.sibling
					currParentSibling.sibling = onlyChild
				}
			}
			break
		}

		if parent.child == nil {
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
		child := node.getChild(search[0])
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
	if node.value != nil {
		if fn(node) {
			return true
		}
	}

	// Create a safe copy of the children list because fn might delete the current child
	// from the radix tree, which mutates the sibling links!
	var children []*radixNode
	for curr := node.child; curr != nil; curr = curr.sibling {
		children = append(children, curr)
	}

	for _, child := range children {
		if walk(child, fn) {
			return true
		}
	}
	return false
}
