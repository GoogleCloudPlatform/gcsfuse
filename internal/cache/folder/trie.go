// Copyright 2025 Google LLC
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

package folder2

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	"unsafe"
)

// FileInfo represents a leaf node data in the trie.
type FileInfo struct {
	mu    sync.RWMutex
	atime time.Time
	mtime time.Time
	ctime time.Time
	size  int64
	data  interface{} //can be used for caching file content
}

// TrieNode represents a node in the trie. Each node corresponds to a
// component in a path.
type TrieNode struct {
	mu       sync.RWMutex
	children map[string]*TrieNode
	isLeaf   bool
	name     string
	file     *FileInfo
}

// newTrieNode creates a new trie node with the given name.
func newTrieNode(name string) *TrieNode {
	return &TrieNode{
		children: make(map[string]*TrieNode),
		name:     name,
	}
}

// ToString returns a string representation of the TrieNode for debugging.
func (n *TrieNode) ToString() string {
	n.mu.RLock()
	defer n.mu.RUnlock()

	var childrenKeys []string
	for k := range n.children {
		childrenKeys = append(childrenKeys, k)
	}
	sort.Strings(childrenKeys)

	fileInfoStr := "nil"
	if n.file != nil {
		fileInfoStr = fmt.Sprintf("&{size: %d}", n.file.size)
	}

	return fmt.Sprintf("TrieNode{name: %q, isLeaf: %v, file: %s, children: %v}", n.name, n.isLeaf, fileInfoStr, childrenKeys)
}

// Trie is a thread-safe prefix tree (trie) data structure suitable for storing
// hierarchical data, such as file system paths.
type Trie struct {
	root        *TrieNode
	mu          sync.RWMutex
	leaf_counts int64
}

// NewTrie creates and returns a new, empty Trie.
func NewTrie() *Trie {
	return &Trie{
		root: newTrieNode(""),
	}
}

// splitPath splits a path string into its components. It filters out any
// empty strings that result from splitting, for example, from leading/trailing
// or consecutive slashes.
func splitPath(path string) []string {
	return strings.FieldsFunc(path, func(r rune) bool {
		return r == '/'
	})
}

// Insert adds a path to the trie. The last part of the path is a leaf node
// representing a file. Intermediate parts are directory nodes.
func (t *Trie) Insert(path string, fileInfo *FileInfo) {
	parts := splitPath(path)
	if len(parts) == 0 {
		return
	}

	node := t.root
	for i, part := range parts {
		node.mu.RLock()
		child, ok := node.children[part]
		node.mu.RUnlock()

		if !ok {
			node.mu.Lock()
			// Re-check in case another goroutine created it in the meantime.
			if child, ok = node.children[part]; !ok {
				child = newTrieNode(part)
				node.children[part] = child
			}
			node.mu.Unlock()
		}

		// If it's the last part, it's a leaf node (file).
		if i == len(parts)-1 {
			child.mu.Lock()
			if !child.isLeaf {
				t.mu.Lock()
				t.leaf_counts++
				t.mu.Unlock()
				child.isLeaf = true
			}
			child.file = fileInfo
			child.mu.Unlock()
		}

		node = child
	}
}

// InsertDir adds a directory path to the trie. It creates the necessary
// intermediate nodes. The final node will not be marked as a leaf.
func (t *Trie) InsertDir(path string) {
	parts := splitPath(path)
	if len(parts) == 0 {
		return
	}

	node := t.root
	for _, part := range parts {
		node.mu.RLock()
		child, ok := node.children[part]
		node.mu.RUnlock()

		if !ok {
			node.mu.Lock()
			// Re-check in case another goroutine created it in the meantime.
			if child, ok = node.children[part]; !ok {
				child = newTrieNode(part)
				node.children[part] = child
			}
			node.mu.Unlock()
		}
		node = child
	}
	// The final node represents a directory, so we don't mark it as a leaf.
}

// Get retrieves the FileInfo for a given path from the trie.
// It returns the FileInfo and true if the path corresponds to a file,
// otherwise it returns nil and false.
func (t *Trie) Get(path string) (*FileInfo, bool) {
	node := t.traverse(path)
	if node == nil {
		return nil, false
	}
	node.mu.RLock()
	defer node.mu.RUnlock()

	if node.isLeaf && node.file != nil {
		return node.file, true
	}

	return nil, false
}

// PathExists checks if a given path exists in the trie, either as a file or a directory.
func (t *Trie) PathExists(path string) bool {
	node := t.traverse(path)
	return node != nil
}

// Delete removes a path and its associated file information from the trie.
// If the node at the given path becomes empty (no children and no data),
// it and any empty parent nodes will be pruned from the trie.
func (t *Trie) Delete(path string) {
	parts := splitPath(path)
	if len(parts) == 0 {
		return
	}

	var nodes []*TrieNode
	node := t.root
	nodes = append(nodes, node)

	for _, part := range parts {
		node.mu.RLock()
		child, ok := node.children[part]
		node.mu.RUnlock()
		if !ok {
			return
		}
		node = child
		nodes = append(nodes, node)
	}

	// Clear the file info from the target node.
	node.mu.Lock()
	if node.isLeaf {
		t.mu.Lock()
		t.leaf_counts--
		t.mu.Unlock()
		node.isLeaf = false
		node.file = nil
	}
	hasChildren := len(node.children) > 0
	node.mu.Unlock()

	// If the node has no children, we can attempt to prune it and its parents.
	if !hasChildren {
		t.pruneEmptyPath(nodes, parts)
	}
}

// pruneEmptyPath traverses up from a deleted node, removing any parent
// directories that have become empty as a result.
func (t *Trie) pruneEmptyPath(nodes []*TrieNode, parts []string, lockedNodes ...*TrieNode) {
	// We lock each parent individually to perform a safe check-and-delete.
	for i := len(parts) - 1; i >= 0; i-- {
		parent := nodes[i]
		isLocked := false
		for _, ln := range lockedNodes {
			if parent == ln {
				isLocked = true
				break
			}
		}

		if !isLocked {
			parent.mu.Lock()
		}
		childName := parts[i]

		child, ok := parent.children[childName]
		if !ok {
			// Should not happen if traversal was correct, but as a safeguard.
			if !isLocked {
				parent.mu.Unlock()
			}
			break
		}

		// Check if the child node is empty and can be deleted.
		if !child.isLeaf && len(child.children) == 0 {
			delete(parent.children, childName)
		}

		// If the parent is now empty and not a leaf, the loop will continue to prune it. Otherwise, we can stop.
		if parent.isLeaf || len(parent.children) > 0 {
			if !isLocked {
				parent.mu.Unlock()
			}
			break
		}
		if !isLocked {
			parent.mu.Unlock()
		}
	}
}

// DeleteFile removes a leaf node at a given path without pruning parent nodes.
// It returns the FileInfo of the deleted file and a boolean indicating if the
// file was found and deleted.
func (t *Trie) DeleteFile(path string) (*FileInfo, bool) {
	node := t.traverse(path)
	if node == nil {
		return nil, false
	}

	node.mu.Lock()
	defer node.mu.Unlock()

	if !node.isLeaf {
		return nil, false
	}

	fileInfo := node.file
	node.isLeaf = false
	node.file = nil
	t.mu.Lock()
	t.leaf_counts--
	t.mu.Unlock()

	return fileInfo, true
}

// ListPathsWithPrefix returns all the file paths in the trie that start with the given prefix.
func (t *Trie) ListPathsWithPrefix(prefix string) []string {
	node := t.traverse(prefix)
	if node == nil {
		return nil // Prefix does not exist.
	}

	// Collect all paths from this node downwards.
	var paths []string
	t.collectPaths(node, prefix, &paths)
	return paths
}

// collectPaths is a helper function to recursively find all paths for leaf nodes
// starting from a given node.
func (t *Trie) collectPaths(node *TrieNode, currentPath string, paths *[]string) {
	// Lock the node only long enough to read its state.
	node.mu.RLock()
	isLeaf := node.isLeaf
	// Make a copy of the children map to iterate over after releasing the lock.
	children := make(map[string]*TrieNode, len(node.children))
	for name, child := range node.children {
		children[name] = child
	}
	node.mu.RUnlock()

	if isLeaf {
		*paths = append(*paths, currentPath)
	}

	for name, child := range children {
		newPath := currentPath
		if newPath != "" && !strings.HasSuffix(newPath, "/") {
			newPath += "/" // Add separator if not present
		} else if newPath == "" {
			newPath = "/" // Start with a separator for root-level children
		}
		newPath += name

		t.collectPaths(child, newPath, paths)
	}
}

// traverse finds the node corresponding to the given path using read locks.
// It returns nil if the path does not exist.
func (t *Trie) traverse(path string) *TrieNode {
	parts := splitPath(path)
	node := t.root

	for _, part := range parts {
		node.mu.RLock()
		child, ok := node.children[part]
		node.mu.RUnlock()

		if !ok {
			return nil
		}
		node = child
	}

	return node
}

// CountFiles returns the total number of files (leaf nodes) stored in the trie.
func (t *Trie) CountFiles() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return int(t.leaf_counts)
}

// // countRecursive is a helper function to recursively count all leaf nodes
// // starting from a given node.
// func (t *Trie) countRecursive(node *TrieNode) int {
// 	// Lock the node only long enough to read its state.
// 	node.mu.RLock()
// 	count := 0
// 	if node.isLeaf {
// 		count = 1
// 	}

// 	// Make a copy of the children map to iterate over after releasing the lock.
// 	children := make(map[string]*TrieNode, len(node.children))
// 	for name, child := range node.children {
// 		children[name] = child
// 	}
// 	node.mu.RUnlock()

// 	// Recursively count files in child nodes.
// 	for _, child := range children {
// 		count += t.countRecursive(child)
// 	}

// 	return count
// }

// Move renames or moves a path, pruning the old path if it becomes empty.
// It moves the entire subtree from sourcePath to destPath.
// It returns false if the source does not exist, the destination already exists,
// or the move is invalid (e.g., moving a directory into itself).
func (t *Trie) Move(sourcePath, destPath string) bool {
	if sourcePath == destPath || sourcePath == "" || destPath == "" || strings.HasPrefix(destPath, sourcePath+"/") {
		return false // Invalid move request.
	}

	sourceParts := splitPath(sourcePath)
	destParts := splitPath(destPath)
	if len(sourceParts) == 0 || len(destParts) == 0 {
		return false
	}

	// 1. Traverse to find source parent and destination parent.
	var sourceParent, destParent *TrieNode
	var sourcePruneNodes []*TrieNode

	// Find source parent and collect nodes for pruning.
	node := t.root
	sourcePruneNodes = append(sourcePruneNodes, node)
	for i := 0; i < len(sourceParts)-1; i++ {
		node.mu.RLock()
		child, ok := node.children[sourceParts[i]]
		node.mu.RUnlock()
		if !ok {
			return false // Source path does not exist.
		}
		node = child
		sourcePruneNodes = append(sourcePruneNodes, node)
	}
	sourceParent = node

	// Find destination parent.
	node = t.root
	for i := 0; i < len(destParts)-1; i++ {
		node.mu.RLock()
		child, ok := node.children[destParts[i]]
		node.mu.RUnlock()
		if !ok {
			// If destination parent path does not exist, create it.
			node.mu.Lock()
			// Re-check in case another goroutine created it.
			if child, ok = node.children[destParts[i]]; !ok {
				child = newTrieNode(destParts[i])
				node.children[destParts[i]] = child
			}
			node.mu.Unlock()
		}
		node = child
	}
	destParent = node

	// Disallow moving to be a sibling of the root.
	if sourceParent == destParent && len(sourceParts) == 1 {
		return false
	}

	// 2. Lock nodes in a consistent order to prevent deadlocks.
	sourceName := sourceParts[len(sourceParts)-1]
	destName := destParts[len(destParts)-1]

	// Lock parents in a consistent order (by memory address) to prevent deadlocks.
	if sourceParent == destParent {
		sourceParent.mu.Lock()
		defer sourceParent.mu.Unlock()
	} else if uintptr(unsafe.Pointer(sourceParent)) < uintptr(unsafe.Pointer(destParent)) {
		sourceParent.mu.Lock()
		destParent.mu.Lock()
		defer sourceParent.mu.Unlock()
		defer destParent.mu.Unlock()
	} else {
		destParent.mu.Lock()
		sourceParent.mu.Lock()
		defer destParent.mu.Unlock()
		defer sourceParent.mu.Unlock()
	}

	sourceNode, ok := sourceParent.children[sourceName]
	if !ok {
		return false // Source disappeared before we could lock.
	}

	// Check if destination already exists.
	if _, ok := destParent.children[destName]; ok {
		return false
	}

	// 3. Perform the move.
	delete(sourceParent.children, sourceName)
	sourceNode.mu.Lock()
	sourceNode.name = destName
	sourceNode.mu.Unlock()
	destParent.children[destName] = sourceNode

	// 4. Prune the old path.
	if len(sourceParent.children) == 0 && !sourceParent.isLeaf {
		t.pruneEmptyPath(sourcePruneNodes, sourceParts[:len(sourceParts)-1], sourceParent, destParent)
	}

	return true
}
