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

package folder

import (
	"strings"
	"sync"
	"time"
	"unsafe"
)

// FileInfo represents a leaf node data in the trie.
type FileInfo struct {
	mu    *sync.RWMutex
	atime time.Time
	mtime time.Time
	ctime time.Time
	size  int64
	data  interface{} //can be used for caching file content
}

// NewFileInfo creates a new FileInfo.
func NewFileInfo() *FileInfo {
	return &FileInfo{
		mu: new(sync.RWMutex),
	}
}

// TrieNode represents a node in the trie. Each node corresponds to a
// component in a path.
type TrieNode struct {
	mu       *sync.RWMutex
	children map[string]*TrieNode
	isLeaf   bool
	file     *FileInfo
}

// newTrieNode creates a new trie node with the given name.
func newTrieNode() *TrieNode {
	return &TrieNode{
		mu: new(sync.RWMutex),
		// children map is lazily initialized to save memory on leaf nodes.
	}
}

// Trie is a thread-safe prefix tree (trie) data structure suitable for storing
// hierarchical data, such as file system paths.
type Trie struct {
	root           *TrieNode
	mu             *sync.RWMutex
	ttl            int        // ttl in minutes.
	ttl_long       int        // ttl_long in minutes.
	lwm_file_count int64      // low water mark when the thread will check for ttl_long/
	hwm_file_count int64      // high water mark beyond which aggresive ttl will be used.
	leaf_counts    int64      // leaf nodes count
	file_counts    int64      // file counts
	is_evicting    bool       // A flag to prevent concurrent eviction runs.
	eviction_mu    sync.Mutex // Mutex to protect the is_evicting flag.
}

// NewTrie creates and returns a new, empty Trie.
func NewTrie() *Trie {
	return &Trie{
		mu:             new(sync.RWMutex),
		root:           newTrieNode(),
		ttl:            -1,
		ttl_long:       -1,
		lwm_file_count: -1,
		hwm_file_count: -1,
	}
}

// NewTrie creates and returns a new, empty Trie.
func NewTrieWithTTL(ttl int, ttl_long int,
	lwm_file_count int64, hwm_file_count int64) *Trie {
	return &Trie{
		mu:             new(sync.RWMutex),
		root:           newTrieNode(),
		ttl:            ttl,
		ttl_long:       ttl_long,
		lwm_file_count: lwm_file_count,
		hwm_file_count: hwm_file_count,
	}
}

// Insert adds a path to the trie. The last part of the path is a leaf node
// representing a file. Intermediate parts are directory nodes.
// This method is not a thread safe option but used for init direct load.
func (t *Trie) InsertDirect(path *string, fileInfo *FileInfo) {
	parts := strings.Split(*path, "/")
	if len(parts) == 0 {
		return
	}

	node := t.root

	for i, part := range parts {
		child, ok := node.children[part]

		if !ok {
			if node.children == nil {
				node.children = make(map[string]*TrieNode)
			}
			child = newTrieNode()
			node.children[part] = child
		}

		node = child // Move to the next level.

		// If it's the last part, it's a leaf node (file).
		if i == len(parts)-1 {
			if !node.isLeaf {
				t.leaf_counts++
				node.isLeaf = true
			}
			if node.file == nil && fileInfo != nil {
				t.file_counts++
			}
			if fileInfo.mu == nil {
				fileInfo.mu = new(sync.RWMutex)
			}
			node.file = fileInfo

		}
	}
}

// Insert adds a path to the trie. The last part of the path is a leaf node
// representing a file. Intermediate parts are directory nodes.
func (t *Trie) Insert(path string, fileInfo *FileInfo) {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return
	}

	node := t.root
	node.mu.Lock() // Lock the root initially.

	for i, part := range parts {
		child, ok := node.children[part]

		if !ok {
			if node.children == nil {
				node.children = make(map[string]*TrieNode)
			}
			child = newTrieNode()
			node.children[part] = child
		}

		child.mu.Lock()  // Lock the child.
		node.mu.Unlock() // Unlock the parent.
		node = child     // Move to the next level.

		// If it's the last part, it's a leaf node (file).
		if i == len(parts)-1 {
			if !node.isLeaf {
				t.mu.Lock()
				t.leaf_counts++
				t.mu.Unlock()
				node.isLeaf = true
			}
			if node.file == nil && fileInfo != nil {
				t.mu.Lock()
				t.file_counts++
				t.mu.Unlock()
			}
			if fileInfo.mu == nil {
				fileInfo.mu = new(sync.RWMutex)
			}
			node.file = fileInfo
			if fileInfo != nil && node.file.atime.IsZero() {
				node.file.atime = time.Now()
			}
		}
	}
	node.mu.Unlock() // Unlock the final node.

	// Check if eviction is needed and not already running.
	t.mu.RLock()
	needsEviction := t.hwm_file_count > 0 && t.file_counts > t.hwm_file_count
	t.mu.RUnlock()

	if needsEviction {
		t.eviction_mu.Lock()
		if !t.is_evicting {
			t.is_evicting = true
			go func() {
				defer func() { t.eviction_mu.Lock(); t.is_evicting = false; t.eviction_mu.Unlock() }()
				t.EvictOlderAccessFiles()
			}()
		}
		t.eviction_mu.Unlock()
	}

}

// InsertDir adds a directory path to the trie. It creates the necessary
// intermediate nodes. The final node will not be marked as a leaf.
func (t *Trie) InsertDir(path string) {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return
	}
	node := t.root
	for _, part := range parts {
		child, ok := node.children[part]

		if !ok {
			if node.children == nil {
				node.children = make(map[string]*TrieNode)
			}
			child = newTrieNode()
			node.children[part] = child
		}
		node = child
	}
}

// Get retrieves the FileInfo for a given path from the trie.
// It returns the FileInfo and true if the path corresponds to a file,
// otherwise it returns nil and false.
func (t *Trie) Get(path string) (*FileInfo, bool) {
	node := t.traverse(&path)
	if node == nil {
		return nil, false
	}
	node.mu.RLock()
	defer node.mu.RUnlock()

	if node.isLeaf && node.file != nil {
		node.file.atime = time.Now()
		return node.file, true
	}

	return nil, false
}

// PathExists checks if a given path exists in the trie, either as a file or a directory.
func (t *Trie) PathExists(path string) bool {
	node := t.traverse(&path)
	return node != nil
}

// Delete removes a path and its associated file information from the trie.
// If the node at the given path becomes empty (no children and no data),
// it and any empty parent nodes will be pruned from the trie.
func (t *Trie) Delete(path string) {
	parts := strings.Split(path, "/")
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
		if node.file != nil {
			t.file_counts--
		}
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
	node := t.traverse(&path)
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
	if node.file != nil {
		t.file_counts--
	}
	t.mu.Unlock()

	return fileInfo, true
}

// ListPathsWithPrefix returns all the file paths in the trie that start with the given prefix.
func (t *Trie) ListPathsWithPrefix(prefix string) []string {
	node := t.traverse(&prefix)
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
func (t *Trie) traverse(path *string) *TrieNode {
	parts := strings.Split(*path, "/")
	node := t.root
	node.mu.RLock() // Lock the root node initially.

	for _, part := range parts {
		child, ok := node.children[part]
		if !ok {
			node.mu.RUnlock()
			return nil
		}

		child.mu.RLock()  // Lock the child before unlocking the parent.
		node.mu.RUnlock() // Unlock the parent.
		node = child
	}
	node.mu.RUnlock() // Unlock the final node.

	return node
}

// CountLeafs returns the total number of files (leaf nodes) stored in the trie.
func (t *Trie) CountLeafs() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return int(t.leaf_counts)
}

// CountFiles returns the total number of files (files nodes) stored in the trie.
func (t *Trie) CountFiles() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return int(t.file_counts)
}

// Move renames or moves a path, pruning the old path if it becomes empty.
// It moves the entire subtree from sourcePath to destPath.
// It returns false if the source does not exist, the destination already exists,
// or the move is invalid (e.g., moving a directory into itself).
func (t *Trie) Move(sourcePath, destPath string) bool {
	if sourcePath == destPath || sourcePath == "" || destPath == "" || strings.HasPrefix(destPath, sourcePath+"/") {
		return false // Invalid move request.
	}

	sourceParts := strings.Split(sourcePath, "/")
	destParts := strings.Split(destPath, "/")
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
				if node.children == nil {
					node.children = make(map[string]*TrieNode)
				}
				child = newTrieNode()
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
	if destParent.children == nil {
		destParent.children = make(map[string]*TrieNode)
	}
	destParent.children[destName] = sourceNode

	// 4. Prune the old path.
	if len(sourceParent.children) == 0 && !sourceParent.isLeaf {
		t.pruneEmptyPath(sourcePruneNodes, sourceParts[:len(sourceParts)-1], sourceParent, destParent)
	}

	return true
}

// EvictOlderThan traverses the trie and removes files with an access time
// older than the specified duration. It prunes empty parent directories.
func (t *Trie) EvictOlderAccessFiles() {
	if t.hwm_file_count <= 0 || t.lwm_file_count <= 0 {
		return
	}
	now := time.Now()

	// Phase 1: Evict based on the long TTL.
	if t.ttl_long > 0 {
		t.evictRecursive(t.root, "", now, time.Duration(t.ttl_long)*time.Minute)
	}

	// Phase 2: If still over low watermark, evict based on the short TTL.
	t.mu.RLock()
	isOverLWM := t.file_counts > t.lwm_file_count
	t.mu.RUnlock()
	if isOverLWM && t.ttl > 0 {
		t.evictRecursive(t.root, "", now, time.Duration(t.ttl)*time.Minute)
	}
}

// evictRecursive is a helper function to recursively traverse the trie and
// evict old files.
func (t *Trie) evictRecursive(node *TrieNode, currentPath string, now time.Time, ttl time.Duration) {
	node.mu.Lock()

	if node.isLeaf && node.file != nil {
		node.file.mu.RLock()
		atime := node.file.atime
		node.file.mu.RUnlock()

		if now.Sub(atime) > ttl {
			// Evict this file.
			node.isLeaf = false
			node.file = nil
			t.mu.Lock()
			t.leaf_counts--
			t.file_counts--
			t.mu.Unlock()
		}
	}

	// Make a copy of children to iterate over after potentially modifying node.children
	childrenToVisit := make(map[string]*TrieNode, len(node.children))
	for name, child := range node.children {
		childrenToVisit[name] = child
	}

	// Unlock parent before traversing to children to allow for better concurrency.
	node.mu.Unlock()

	for name, child := range childrenToVisit {
		var childPath string
		if currentPath == "" {
			childPath = "/" + name
		} else {
			childPath = currentPath + "/" + name
		}
		t.evictRecursive(child, childPath, now, ttl)
	}

	// After visiting children, re-lock and check if this node became empty and can be pruned.
	node.mu.Lock()
	defer node.mu.Unlock()

	// Prune children that are now empty leaves without files.
	for name, child := range node.children {
		if !child.isLeaf && len(child.children) == 0 {
			delete(node.children, name)
		}
	}
}
