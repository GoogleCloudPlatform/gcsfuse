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
	"fmt"
	"reflect"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/locker"
)

// TrieCache is a LRU cache for any lru.ValueType indexed by string keys.
// It uses a Trie data structure for efficient prefix-based operations.
type TrieCache struct {
	/////////////////////////
	// Constant data
	/////////////////////////

	// INVARIANT: maxSize > 0
	maxSize uint64

	/////////////////////////
	// Mutable state
	/////////////////////////

	// Sum of entry.Value.Size() of all the entries in the cache.
	currentSize uint64

	// List of cache entries, with least recently used at the tail.
	//
	// INVARIANT: currentSize <= maxSize
	// INVARIANT: Each element is of type entry
	entries list.List

	// Trie root node.
	root *trieNode

	// All public methods of this Cache uses this RW mutex based locker while
	// accessing/updating Cache's data.
	mu locker.RWLocker
}

type trieNode struct {
	children map[byte]*trieNode
	// element points to the list element if this node represents a key in the cache.
	// nil otherwise.
	element *list.Element
}

func newTrieNode() *trieNode {
	return &trieNode{
		children: make(map[byte]*trieNode),
	}
}

// NewTrieCache returns the reference of cache object by initialising the cache with
// the supplied maxSize, which must be greater than zero.
func NewTrieCache(maxSize uint64) *TrieCache {
	c := &TrieCache{
		maxSize: maxSize,
		root:    newTrieNode(),
	}

	// Set up invariant checking.
	c.mu = locker.NewRW("TrieCache", c.checkInvariants)
	return c
}

// checkInvariants panic if any internal invariants have been violated.
func (c *TrieCache) checkInvariants() {
	// INVARIANT: maxSize > 0
	if !(c.maxSize > 0) {
		panic(fmt.Sprintf("Invalid maxSize: %v", c.maxSize))
	}

	// INVARIANT: currentSize <= maxSize
	if !(c.currentSize <= c.maxSize) {
		panic(fmt.Sprintf("CurrentSize %v over maxSize %v", c.currentSize, c.maxSize))
	}

	// INVARIANT: Each element is of type entry
	for e := c.entries.Front(); e != nil; e = e.Next() {
		switch e.Value.(type) {
		case entry:
		default:
			panic(fmt.Sprintf("Unexpected element type: %v", reflect.TypeOf(e.Value)))
		}
	}

	// Verify Trie consistency could be expensive, so skipping full traversal for now.
	// But we can check count if we maintained it.
}

func (c *TrieCache) evictOne() ValueType {
	e := c.entries.Back()
	key := e.Value.(entry).Key

	evictedEntry := e.Value.(entry).Value
	c.currentSize -= evictedEntry.Size()

	c.entries.Remove(e)
	c.deleteFromTrie(key)

	return evictedEntry
}

func (c *TrieCache) insertIntoTrie(key string, e *list.Element) {
	node := c.root
	for i := 0; i < len(key); i++ {
		b := key[i]
		if node.children[b] == nil {
			node.children[b] = newTrieNode()
		}
		node = node.children[b]
	}
	node.element = e
}

func (c *TrieCache) lookUpInTrie(key string) *list.Element {
	node := c.root
	for i := 0; i < len(key); i++ {
		b := key[i]
		if node.children[b] == nil {
			return nil
		}
		node = node.children[b]
	}
	return node.element
}

func (c *TrieCache) deleteFromTrie(key string) {
	// We need to remove the element reference and potentially prune unused nodes.
	// Recursive approach is easier for pruning.
	c.deleteFromTrieRecursive(c.root, key, 0)
}

func (c *TrieCache) deleteFromTrieRecursive(node *trieNode, key string, index int) bool {
	if index == len(key) {
		node.element = nil
		return len(node.children) == 0 // Return true if node can be deleted
	}

	b := key[index]
	child, ok := node.children[b]
	if !ok {
		return false // Key not found
	}

	shouldDeleteChild := c.deleteFromTrieRecursive(child, key, index+1)
	if shouldDeleteChild {
		delete(node.children, b)
		return len(node.children) == 0 && node.element == nil
	}

	return false
}

// Insert the supplied value into the cache, overwriting any previous entry for
// the given key. The value must be non-nil.
// Also returns a slice of ValueType evicted by the new inserted entry.
func (c *TrieCache) Insert(
	key string,
	value ValueType) ([]ValueType, error) {
	if value == nil {
		return nil, ErrInvalidEntry
	}

	valueSize := value.Size()
	if valueSize > c.maxSize {
		return nil, ErrInvalidEntrySize
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	e := c.lookUpInTrie(key)
	if e != nil {
		// Update an entry if already exist.
		c.currentSize -= e.Value.(entry).Value.Size()
		c.currentSize += valueSize
		e.Value = entry{key, value}
		c.entries.MoveToFront(e)
	} else {
		// Add the entry if already doesn't exist.
		e := c.entries.PushFront(entry{key, value})
		c.insertIntoTrie(key, e)
		c.currentSize += valueSize
	}

	var evictedValues []ValueType
	// Evict until we're at or below maxSize.
	for c.currentSize > c.maxSize {
		evictedValues = append(evictedValues, c.evictOne())
	}

	return evictedValues, nil
}

// Erase any entry for the supplied key, also returns the value of erased key.
func (c *TrieCache) Erase(key string) (value ValueType) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e := c.lookUpInTrie(key)
	if e == nil {
		return
	}

	deletedEntry := e.Value.(entry).Value
	c.currentSize -= deletedEntry.Size()

	c.entries.Remove(e)
	c.deleteFromTrie(key)

	return deletedEntry
}

// LookUp a previously-inserted value for the given key. Return nil if no
// value is present.
func (c *TrieCache) LookUp(key string) (value ValueType) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e := c.lookUpInTrie(key)
	if e == nil {
		return
	}
	// This is now the most recently used entry.
	c.entries.MoveToFront(e)

	// Return the value.
	return e.Value.(entry).Value
}

// EraseEntriesWithGivenPrefix erases all entries with the given prefix.
func (c *TrieCache) EraseEntriesWithGivenPrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Find the node corresponding to the prefix.
	node := c.root
	for i := 0; i < len(prefix); i++ {
		b := prefix[i]
		if node.children[b] == nil {
			return // Prefix not found
		}
		node = node.children[b]
	}

	// Collect all entries under this node and remove them.
	// We can just traverse and collect keys, then remove them.
	// Or we can remove them directly.
	// Removing directly might be tricky with the list.
	// Let's collect keys first to be safe and simple, or just list elements.
	var elements []*list.Element
	c.collectElements(node, &elements)

	for _, e := range elements {
		val := e.Value.(entry).Value
		c.currentSize -= val.Size()
		c.entries.Remove(e)
		// We don't need to call deleteFromTrie for each key if we prune the subtree.
		// But we might only be pruning a subtree, so we need to be careful.
		// Actually, if we erase the prefix, we can just remove the child pointer from the parent of the prefix node?
		// Yes, if the prefix matches exactly a node, we can just cut it off.
		// BUT, the prefix node itself might have an element (if prefix is also a key).
		// And we need to update currentSize and entries list.
	}

	// Now efficiently remove from Trie.
	// If prefix is empty, clear everything.
	if prefix == "" {
		c.root = newTrieNode()
		return
	}

	// Otherwise, find parent of the prefix node and remove the child.
	// Wait, we already traversed to `node` which corresponds to `prefix`.
	// We need to remove `node` from its parent.
	// Let's re-traverse to find parent.
	parentNode := c.root
	for i := 0; i < len(prefix)-1; i++ {
		parentNode = parentNode.children[prefix[i]]
	}
	lastByte := prefix[len(prefix)-1]
	
	// If the prefix node itself has an element, it was already added to `elements` by `collectElements`.
	// So we can just delete the child from parent.
	delete(parentNode.children, lastByte)
	
	// We also need to prune upwards if parent becomes empty and has no element.
	// This is handled by deleteFromTrieRecursive usually, but here we did a bulk delete.
	// For simplicity, we can leave potentially empty nodes, or do a cleanup.
	// Given this is an optimization, leaving empty nodes is fine for correctness, but might waste memory.
	// Let's try to cleanup upwards.
	c.cleanupUpwards(prefix)
}

func (c *TrieCache) collectElements(node *trieNode, elements *[]*list.Element) {
	if node.element != nil {
		*elements = append(*elements, node.element)
	}
	for _, child := range node.children {
		c.collectElements(child, elements)
	}
}

func (c *TrieCache) cleanupUpwards(key string) {
	// Similar to deleteFromTrieRecursive but we know we just deleted the end.
	// Actually, we can just call deleteFromTrieRecursive with the prefix, 
	// but we need to make sure it knows the child is already gone? 
	// Or just use deleteFromTrieRecursive logic to remove the node if it exists?
	// Wait, we already deleted the child from parent.
	// So we just need to check if parent is now empty and has no element.
	
	// Let's just use a simplified loop going backwards? Trie doesn't have parent pointers.
	// So we have to use recursion or stack.
	// Let's just use deleteFromTrieRecursive logic but for the prefix string, 
	// effectively "deleting" the prefix key but treating it as if we want to remove the node.
	// Actually, `deleteFromTrieRecursive` removes the element at the end. 
	// Here we want to remove the whole subtree at the end.
	// We already removed the subtree by `delete(parentNode.children, lastByte)`.
	// Now we just need to prune parents if they are empty.
	
	if len(key) == 0 {
		return
	}
	
	// We can call a helper that goes down to (len-1) and checks if it can prune.
	c.pruneEmptyNodes(c.root, key, 0)
}

func (c *TrieCache) pruneEmptyNodes(node *trieNode, key string, index int) bool {
	if index == len(key) {
		// We reached the end of key. This node (corresponding to key) was already removed/detached?
		// No, we are traversing `key` which is the prefix.
		// Wait, in EraseEntriesWithGivenPrefix, we detached the node at `key`.
		// So `node` here is the PARENT of the detached node?
		// No, `key` is the prefix.
		// If we call pruneEmptyNodes(root, prefix, 0), we will reach the node for prefix.
		// BUT we already deleted that node from its parent!
		// So we should call it with `prefix` but stop at `len(prefix)-1`.
		return false
	}
	
	// If we are at the parent of the deleted node
	if index == len(key)-1 {
		// This node contains the pointer to the deleted child.
		// We already did `delete(node.children, key[index])`.
		// Now check if this node is empty.
		return len(node.children) == 0 && node.element == nil
	}

	b := key[index]
	child, ok := node.children[b]
	if !ok {
		return false // Should not happen if we just deleted it
	}

	shouldDeleteChild := c.pruneEmptyNodes(child, key, index+1)
	if shouldDeleteChild {
		delete(node.children, b)
		return len(node.children) == 0 && node.element == nil
	}

	return false
}

// LookUpWithoutChangingOrder looks up previously-inserted value for a given key
// without changing the order of entries in the cache. Return nil if no value
// is present.
func (c *TrieCache) LookUpWithoutChangingOrder(key string) (value ValueType) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	e := c.lookUpInTrie(key)
	if e == nil {
		return
	}

	return e.Value.(entry).Value
}

// UpdateWithoutChangingOrder updates entry with the given key in cache with
// given value without changing order of entries in cache, returning error if an
// entry with given key doesn't exist.
func (c *TrieCache) UpdateWithoutChangingOrder(
	key string,
	value ValueType) error {
	if value == nil {
		return ErrInvalidEntry
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	e := c.lookUpInTrie(key)
	if e == nil {
		return ErrEntryNotExist
	}

	if value.Size() != e.Value.(entry).Value.Size() {
		return ErrInvalidUpdateEntrySize
	}

	e.Value = entry{key, value}
	// No need to update Trie as element pointer is same.

	return nil
}

// Helper to print trie for debugging
func (c *TrieCache) PrintTrie() {
	c.printTrieRecursive(c.root, "")
}

func (c *TrieCache) printTrieRecursive(node *trieNode, prefix string) {
	if node.element != nil {
		fmt.Printf("Key: %s, Value: %v\n", prefix, node.element.Value.(entry).Value)
	}
	for b, child := range node.children {
		c.printTrieRecursive(child, prefix+string(b))
	}
}

// Keys returns all keys in the cache.
func (c *TrieCache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var keys []string
	c.collectKeys(c.root, "", &keys)
	return keys
}

func (c *TrieCache) collectKeys(node *trieNode, prefix string, keys *[]string) {
	if node.element != nil {
		*keys = append(*keys, prefix)
	}
	for b, child := range node.children {
		c.collectKeys(child, prefix+string(b), keys)
	}
}

// Check if the cache contains a key.
func (c *TrieCache) Contains(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lookUpInTrie(key) != nil
}

// Len returns the number of items in the cache.
func (c *TrieCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.entries.Len()
}
