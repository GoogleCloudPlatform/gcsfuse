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
	"sort"
	"strings"
	"sync"
	"time"
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

// Data returns the data associated with the file.
func (fi *FileInfo) Data() interface{} {
	if fi == nil {
		return nil
	}
	return fi.data
}

// NewFileInfo creates a new FileInfo.
func NewFileInfo() *FileInfo {
	return &FileInfo{
		mu: new(sync.RWMutex),
	}
}

func NewFileInfoWithData(data interface{}) *FileInfo {
	return &FileInfo{
		data: data,
		mu:   new(sync.RWMutex),
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
	root        *TrieNode
	mu          *sync.RWMutex
	ttl         int        // ttl in minutes.
	ttl_long    int        // long ttl in minutes.
	lwm_size_mb int64      // low water mark for size in MB.
	hwm_size_mb int64      // high water mark for size in MB.
	leaf_counts int64      // leaf nodes count
	total_size  int64      // total size of all files in bytes.
	is_evicting bool       // A flag to prevent concurrent eviction runs.
	eviction_mu sync.Mutex // Mutex to protect the is_evicting flag
}

// NewTrie creates and returns a new, empty Trie.
func NewTrie() *Trie {
	return &Trie{
		mu:          new(sync.RWMutex),
		root:        newTrieNode(),
		ttl:         -1,
		ttl_long:    -1,
		lwm_size_mb: -1,
		hwm_size_mb: -1,
	}
}

// NewTrie creates and returns a new, empty Trie.
func NewTrieWithTTL(ttl int, ttl_long int,
	lwm_size_mb int64, hwm_size_mb int64) *Trie {
	return &Trie{
		mu:          new(sync.RWMutex),
		root:        newTrieNode(),
		ttl:         ttl,
		ttl_long:    ttl_long,
		lwm_size_mb: lwm_size_mb,
		hwm_size_mb: hwm_size_mb,
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
				t.total_size += fileInfo.size
				t.mu.Unlock()
			} else if node.file != nil && fileInfo != nil {
				t.mu.Lock()
				t.total_size -= node.file.size
				t.total_size += fileInfo.size
				t.mu.Unlock()
			} else if node.file != nil && fileInfo == nil {
				t.mu.Lock()
				t.total_size -= node.file.size
				t.mu.Unlock()
			}
			if fileInfo != nil && fileInfo.mu == nil {
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
	needsEviction := t.hwm_size_mb > 0 && t.total_size > (t.hwm_size_mb*1024*1024)
	t.mu.RUnlock()

	if needsEviction {
		t.eviction_mu.Lock()
		if !t.is_evicting {
			t.is_evicting = true
			go func() {
				defer func() { t.eviction_mu.Lock(); t.is_evicting = false; t.eviction_mu.Unlock() }()
				t.evict()
			}()
		}
		t.eviction_mu.Unlock()
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
			t.total_size -= node.file.size
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
	// This is an approximation as we don't have file_counts anymore.
	// We can iterate and count, but that is slow.
	// Returning leaf_counts is a reasonable substitute if needed.
	return int(t.leaf_counts)
}

type fileEvictionInfo struct {
	path  string
	atime time.Time
	size  int64
}

func (t *Trie) collectFilesForEviction(node *TrieNode, currentPath string, files *[]fileEvictionInfo) {
	node.mu.RLock()
	isLeaf := node.isLeaf
	file := node.file
	children := make(map[string]*TrieNode, len(node.children))
	for name, child := range node.children {
		children[name] = child
	}
	node.mu.RUnlock()

	if isLeaf && file != nil {
		file.mu.RLock()
		atime := file.atime
		size := file.size
		file.mu.RUnlock()
		*files = append(*files, fileEvictionInfo{path: currentPath, atime: atime, size: size})
	}

	for name, child := range children {
		newPath := currentPath
		if newPath != "" && !strings.HasSuffix(newPath, "/") {
			newPath += "/"
		} else if newPath == "" {
			newPath = "/"
		}
		newPath += name
		t.collectFilesForEviction(child, newPath, files)
	}
}

// evict traverses the trie and removes files based on size and access time.
func (t *Trie) evict() {
	if t.hwm_size_mb <= 0 || t.lwm_size_mb <= 0 {
		return
	}

	var filesToEvict []fileEvictionInfo
	t.collectFilesForEviction(t.root, "", &filesToEvict)

	// Sort files by access time, oldest first.
	sort.Slice(filesToEvict, func(i, j int) bool {
		return filesToEvict[i].atime.Before(filesToEvict[j].atime)
	})

	lwmBytes := t.lwm_size_mb * 1024 * 1024

	t.mu.RLock()
	currentSize := t.total_size
	t.mu.RUnlock()

	if currentSize <= lwmBytes {
		return
	}

	var evictedSize int64
	for _, file := range filesToEvict {
		if currentSize-evictedSize <= lwmBytes {
			break
		}
		t.Delete(file.path)
		evictedSize += file.size
	}
}
