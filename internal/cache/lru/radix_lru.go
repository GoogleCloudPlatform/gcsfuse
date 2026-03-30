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

// radixListEntry is stored inside the LRU linked list's element.
type radixListEntry struct {
	node  *radixNode
	value ValueType
}

// radixCache is an LRU cache implementation that uses a radix tree for indexing.
type radixCache struct {
	maxSize uint64

	currentSize uint64

	entries list.List

	index *radixTree

	mu locker.RWLocker
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

	for e := c.entries.Front(); e != nil; e = e.Next() {
		switch e.Value.(type) {
		case radixListEntry:
		default:
			panic(fmt.Sprintf("Unexpected element type: %v", reflect.TypeOf(e.Value)))
		}
	}

	if c.entries.Len() != c.index.Len() {
		panic(fmt.Sprintf(
			"Length mismatch: %v vs. %v",
			c.entries.Len(),
			c.index.Len()))
	}

	for e := c.entries.Front(); e != nil; e = e.Next() {
		node := e.Value.(radixListEntry).node
		if node.element != e {
			panic("Mismatch for internal radix node element mapping")
		}
	}
}

func (c *radixCache) evictOne() ValueType {
	e := c.entries.Back()
	node := e.Value.(radixListEntry).node

	evictedEntry := e.Value.(radixListEntry).value
	c.currentSize -= evictedEntry.Size()

	c.entries.Remove(e)
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

	node, ok := c.index.Get(key)
	if ok {
		e := node.element
		c.currentSize -= e.Value.(radixListEntry).value.Size()
		c.currentSize += valueSize
		e.Value = radixListEntry{node: node, value: value}
		c.entries.MoveToFront(e)
	} else {
		// Insert temporarily to get the node, then update element.
		node, _ = c.index.Insert(key, nil)
		e := c.entries.PushFront(radixListEntry{node: node, value: value})
		node.element = e
		c.currentSize += valueSize
	}

	var evictedValues []ValueType
	for c.currentSize > c.maxSize {
		evictedValues = append(evictedValues, c.evictOne())
	}

	return evictedValues, nil
}

func (c *radixCache) eraseInternal(key string) (value ValueType) {
	node, ok := c.index.Get(key)
	if !ok {
		return
	}

	e := node.element
	deletedEntry := e.Value.(radixListEntry).value
	c.currentSize -= deletedEntry.Size()

	c.index.DeleteNode(node)
	c.entries.Remove(e)

	return deletedEntry
}

func (c *radixCache) Erase(key string) (value ValueType) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.eraseInternal(key)
}

func (c *radixCache) LookUp(key string) (value ValueType) {
	c.mu.Lock()
	defer c.mu.Unlock()

	node, ok := c.index.Get(key)
	if !ok {
		return
	}
	e := node.element
	c.entries.MoveToFront(e)

	return e.Value.(radixListEntry).value
}

func (c *radixCache) LookUpWithoutChangingOrder(key string) (value ValueType) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	node, ok := c.index.Get(key)
	if !ok {
		return
	}
	e := node.element

	return e.Value.(radixListEntry).value
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

	e := node.element
	if value.Size() != e.Value.(radixListEntry).value.Size() {
		return ErrInvalidUpdateEntrySize
	}

	e.Value = radixListEntry{node: node, value: value}
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
			if node.element == nil {
				continue
			}
			e := node.element
			deletedEntry := e.Value.(radixListEntry).value
			c.currentSize -= deletedEntry.Size()

			c.index.DeleteNode(node)
			c.entries.Remove(e)
		}
	}
}
