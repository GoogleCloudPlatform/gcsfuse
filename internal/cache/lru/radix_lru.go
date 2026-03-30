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
	"fmt"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/locker"
)

// radixCache is an LRU cache implementation that uses a highly optimized custom radix tree.
// The tree's nodes double as the LRU doubly linked list elements, reducing overhead.
type radixCache struct {
	maxSize     uint64
	currentSize uint64

	// Head and tail of the LRU Doubly Linked List
	head *radixNode
	tail *radixNode
	len  int

	index *radixTree
	mu    locker.RWLocker
}

// NewRadixCache creates a new radix-tree based LRU cache.
func NewRadixCache(maxSize uint64) Cache {
	c := &radixCache{
		maxSize: maxSize,
		index:   newRadixTree(),
	}
	c.mu = locker.NewRW("RadixLRUCache", c.checkInvariants)
	return c
}

func (c *radixCache) checkInvariants() {
	if !(c.maxSize > 0) {
		panic(fmt.Sprintf("Invalid maxSize: %v", c.maxSize))
	}

	if !(c.currentSize <= c.maxSize) {
		panic(fmt.Sprintf("CurrentSize %v over maxSize %v", c.currentSize, c.maxSize))
	}

	count := 0
	for curr := c.head; curr != nil; curr = curr.next {
		count++
		if curr.value == nil {
			panic("LRU List contains an internal routing node instead of a cache entry")
		}
	}

	if count != c.len {
		panic(fmt.Sprintf("LRU List length %v does not match c.len %v", count, c.len))
	}

	if c.len != c.index.Len() {
		panic(fmt.Sprintf(
			"Length mismatch: %v vs. %v",
			c.len,
			c.index.Len()))
	}
}

// Internal LRU Linked List operations
func (c *radixCache) moveToFront(node *radixNode) {
	if c.head == node {
		return
	}
	// Detach
	if node.prev != nil {
		node.prev.next = node.next
	}
	if node.next != nil {
		node.next.prev = node.prev
	}
	if c.tail == node {
		c.tail = node.prev
	}

	// Insert at front
	node.prev = nil
	node.next = c.head
	if c.head != nil {
		c.head.prev = node
	}
	c.head = node
	if c.tail == nil {
		c.tail = node
	}
}

func (c *radixCache) pushFront(node *radixNode) {
	node.prev = nil
	node.next = c.head
	if c.head != nil {
		c.head.prev = node
	}
	c.head = node
	if c.tail == nil {
		c.tail = node
	}
	c.len++
}

func (c *radixCache) remove(node *radixNode) {
	if node.prev != nil {
		node.prev.next = node.next
	} else {
		c.head = node.next
	}
	if node.next != nil {
		node.next.prev = node.prev
	} else {
		c.tail = node.prev
	}
	node.prev = nil
	node.next = nil
	c.len--
}

func (c *radixCache) evictOne() ValueType {
	node := c.tail
	evictedEntry := node.value

	c.currentSize -= evictedEntry.Size()

	c.remove(node)
	c.index.DeleteNode(node)

	return evictedEntry
}

func (c *radixCache) Insert(key string, value ValueType) ([]ValueType, error) {
	if value == nil {
		return nil, ErrInvalidEntry
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	valueSize := value.Size()
	if valueSize > c.maxSize {
		return nil, ErrInvalidEntrySize
	}

	node, isNew := c.index.Insert(key, value)
	if !isNew {
		c.currentSize -= node.value.Size()
		c.currentSize += valueSize
		node.value = value
		c.moveToFront(node)
	} else {
		c.pushFront(node)
		c.currentSize += valueSize
	}

	var evictedValues []ValueType
	for c.currentSize > c.maxSize {
		evictedValues = append(evictedValues, c.evictOne())
	}

	return evictedValues, nil
}

func (c *radixCache) eraseInternal(node *radixNode) (value ValueType) {
	deletedEntry := node.value
	c.currentSize -= deletedEntry.Size()

	c.remove(node)
	c.index.DeleteNode(node)

	return deletedEntry
}

func (c *radixCache) Erase(key string) (value ValueType) {
	c.mu.Lock()
	defer c.mu.Unlock()

	node, ok := c.index.Get(key)
	if !ok {
		return
	}

	return c.eraseInternal(node)
}

func (c *radixCache) LookUp(key string) (value ValueType) {
	c.mu.Lock()
	defer c.mu.Unlock()

	node, ok := c.index.Get(key)
	if !ok {
		return
	}
	c.moveToFront(node)

	return node.value
}

func (c *radixCache) LookUpWithoutChangingOrder(key string) (value ValueType) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	node, ok := c.index.Get(key)
	if !ok {
		return
	}

	return node.value
}

func (c *radixCache) UpdateWithoutChangingOrder(key string, value ValueType) error {
	if value == nil {
		return ErrInvalidEntry
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	node, ok := c.index.Get(key)
	if !ok {
		return ErrEntryNotExist
	}

	if value.Size() != node.value.Size() {
		return ErrInvalidUpdateEntrySize
	}

	node.value = value
	return nil
}

func (c *radixCache) UpdateSize(key string, sizeDelta uint64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, ok := c.index.Get(key)
	if !ok {
		return ErrEntryNotExist
	}

	c.currentSize += sizeDelta

	return nil
}

func (c *radixCache) EraseEntriesWithGivenPrefix(prefix string) {
	c.mu.RLock()
	var nodesToDelete []*radixNode
	c.index.WalkPrefix(prefix, func(node *radixNode) bool {
		nodesToDelete = append(nodesToDelete, node)
		return false
	})
	c.mu.RUnlock()

	if len(nodesToDelete) > 0 {
		c.mu.Lock()
		defer c.mu.Unlock()
		for _, node := range nodesToDelete {
			if node.value == nil { // Double check it hasn't been deleted
				continue
			}
			c.eraseInternal(node)
		}
	}
}
