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
	"container/list"
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

// trieNode represents a single node in the file system tree.
type trieNode struct {
	name     string
	parent   *trieNode
	children map[string]*trieNode
	// value is nil for directories, and points to the LRU element for files.
	value *list.Element
}

// isDir returns true if this node represents a directory (has no value).
func (n *trieNode) isDir() bool {
	return n.value == nil
}

// fullPath reconstructs the full path of the node by traversing up to the root.
func (n *trieNode) fullPath() string {
	if n.parent == nil {
		return n.name
	}
	parentPath := n.parent.fullPath()
	if parentPath == "" {
		return n.name
	}
	return filepath.Join(parentPath, n.name)
}

// fileCacheTrieIndexer implements the indexer interface using a bidirectional Trie.
// It is NOT thread-safe on its own; it relies on the parent LRU Cache mutex.
type fileCacheTrieIndexer struct {
	root       *trieNode
	totalFiles int
	onEmptyDir func(dirPath string)
}

// newFileCacheTrieIndexer creates a new fileCacheTrieIndexer.
// onEmptyDir is a callback invoked synchronously when a directory node
// is removed from the trie because it became empty.
func NewFileCacheTrieIndexer(onEmptyDir func(dirPath string)) *fileCacheTrieIndexer {
	return &fileCacheTrieIndexer{
		root: &trieNode{
			name:     "",
			children: make(map[string]*trieNode),
		},
		onEmptyDir: onEmptyDir,
	}
}

// splitPath splits a path into segments, ignoring empty segments.
func splitPath(p string) []string {
	// Clean the path to remove redundant slashes
	p = path.Clean(p)
	if p == "." || p == "/" || p == "" {
		return nil
	}
	// Remove leading slash for consistency
	if strings.HasPrefix(p, "/") {
		p = p[1:]
	}
	return strings.Split(p, "/")
}

// Get retrieves the list element for the given key, if it exists and is a file.
func (ti *fileCacheTrieIndexer) Get(key string) (*list.Element, bool) {
	segments := splitPath(key)
	if len(segments) == 0 {
		return nil, false
	}

	curr := ti.root
	for i, segment := range segments {
		next, ok := curr.children[segment]
		if !ok {
			return nil, false
		}
		if i < len(segments)-1 && !next.isDir() {
			panic(fmt.Sprintf("fileCacheTrieIndexer.Get: intermediate segment %q in path %q is a file", next.fullPath(), key))
		}
		curr = next
	}

	// Only return the value if it's a file node
	if curr.isDir() {
		return nil, false
	}
	return curr.value, true
}

// Set inserts or updates the value for the given key.
func (ti *fileCacheTrieIndexer) Set(key string, value *list.Element) {
	segments := splitPath(key)
	if len(segments) == 0 {
		return
	}

	curr := ti.root
	for i, segment := range segments {
		next, ok := curr.children[segment]
		if !ok {
			next = &trieNode{
				name:     segment,
				parent:   curr,
				children: make(map[string]*trieNode),
			}
			curr.children[segment] = next
		} else if i < len(segments)-1 && !next.isDir() {
			panic(fmt.Sprintf("fileCacheTrieIndexer.Set: intermediate segment %q in path %q is already a known file", next.fullPath(), key))
		}
		curr = next

		// If this is the last segment, it's a file
		if i == len(segments)-1 {
			if curr.isDir() {
				ti.totalFiles++
			}
			curr.value = value
		}
	}
}

// Delete removes the file node at the given key.
// If its parent directory becomes empty, the onEmptyDir callback is invoked.
// Crucially, it DOES NOT delete the parent directory node from the trie.
// The directory node is only deleted via DeleteDir after successful disk removal.
func (ti *fileCacheTrieIndexer) Delete(key string) {
	segments := splitPath(key)
	if len(segments) == 0 {
		return
	}

	curr := ti.root
	for i, segment := range segments {
		next, ok := curr.children[segment]
		if !ok {
			return // Key not found
		}
		if i < len(segments)-1 && !next.isDir() {
			panic(fmt.Sprintf("fileCacheTrieIndexer.Delete: intermediate segment %q in path %q is a file", next.fullPath(), key))
		}
		curr = next
	}

	if curr.isDir() {
		return // Can only delete files via this interface
	}

	// Remove the file node from its parent
	parent := curr.parent
	if parent != nil {
		delete(parent.children, curr.name)
		ti.totalFiles--

		// If the parent directory is now empty, signal it for background deletion.
		if len(parent.children) == 0 && parent.isDir() && ti.onEmptyDir != nil {
			// DO NOT delete the parent node here. Just signal.
			ti.onEmptyDir(parent.fullPath())
		}
	}
}

// DeleteDirIfEmpty removes an empty directory node from the trie.
// This is called AFTER the background worker successfully deletes the directory from disk.
// If deleting this node makes its parent empty, it signals the parent for deletion.
func (ti *fileCacheTrieIndexer) DeleteDirIfEmpty(dirPath string) {
	segments := splitPath(dirPath)
	if len(segments) == 0 {
		return
	}

	curr := ti.root
	for i, segment := range segments {
		next, ok := curr.children[segment]
		if !ok {
			return // Dir not found (maybe already deleted or renamed concurrently)
		}
		if i < len(segments)-1 && !next.isDir() {
			// Should not happen if filesystem is sane, but be safe
			return
		}
		curr = next
	}

	// Double check: Is it still a directory, and is it STILL empty?
	// A concurrent Insert might have added a file while os.Remove was running.
	if !curr.isDir() || len(curr.children) > 0 {
		return
	}

	// Remove the empty directory node from its parent
	parent := curr.parent
	if parent != nil {
		delete(parent.children, curr.name)

		// Recursion Trigger: If removing this directory made its parent empty,
		// signal the parent for background deletion.
		if len(parent.children) == 0 && parent.isDir() && ti.onEmptyDir != nil && parent != ti.root {
			ti.onEmptyDir(parent.fullPath())
		}
	}
}

func (ti *fileCacheTrieIndexer) SetOnEmptyDirCallback(f func(dirPath string)) {
	ti.onEmptyDir = f
}

// KeysWithPrefix returns all file keys that start with the given prefix.
func (ti *fileCacheTrieIndexer) KeysWithPrefix(prefix string) []string {
	var result []string

	// Handle empty prefix - return all files
	if prefix == "" {
		ti.collectFiles(ti.root, "", &result)
		return result
	}

	segments := splitPath(prefix)
	curr := ti.root

	// Traverse to the prefix node
	for i, segment := range segments {
		// If it's the last segment, it might be a partial match (e.g., prefix "a/b/c" matching "a/b/c1.txt")
		if i == len(segments)-1 {
			for childName, childNode := range curr.children {
				if strings.HasPrefix(childName, segment) {
					ti.collectFiles(childNode, childNode.fullPath(), &result)
				}
			}
			return result
		}

		next, ok := curr.children[segment]
		if !ok {
			return result // Prefix not found
		}
		if !next.isDir() {
			panic(fmt.Sprintf("fileCacheTrieIndexer.KeysWithPrefix: intermediate segment %q in prefix %q is a file", next.fullPath(), prefix))
		}
		curr = next
	}

	// If we perfectly matched a directory path, collect all files under it
	ti.collectFiles(curr, curr.fullPath(), &result)
	return result
}

// collectFiles recursively collects all file paths under the given node.
func (ti *fileCacheTrieIndexer) collectFiles(node *trieNode, currentPath string, result *[]string) {
	if !node.isDir() {
		*result = append(*result, currentPath)
	}

	for _, child := range node.children {
		childPath := child.name
		if currentPath != "" {
			childPath = filepath.Join(currentPath, child.name)
		}
		ti.collectFiles(child, childPath, result)
	}
}

func (ti *fileCacheTrieIndexer) Len() int {
	return ti.totalFiles
}
