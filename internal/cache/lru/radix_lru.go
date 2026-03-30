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

	"github.com/armon/go-radix"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/locker"
)

// radixCache is an LRU cache implementation that uses a radix tree for indexing.
type radixCache struct {
	maxSize uint64

	currentSize uint64

	entries list.List

	index *radix.Tree

	mu locker.RWLocker
}

// NewRadixCache creates a new radix-tree based LRU cache.
func NewRadixCache(maxSize uint64) Cache {
	c := &radixCache{
		maxSize: maxSize,
		index:   radix.New(),
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
		case entry:
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
		val, ok := c.index.Get(e.Value.(entry).Key)
		if !ok || val.(*list.Element) != e {
			panic(fmt.Sprintf("Mismatch for key %v", e.Value.(entry).Key))
		}
	}
}

func (c *radixCache) evictOne() ValueType {
	e := c.entries.Back()
	key := e.Value.(entry).Key

	evictedEntry := e.Value.(entry).Value
	c.currentSize -= evictedEntry.Size()

	c.entries.Remove(e)
	c.index.Delete(key)

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

	val, ok := c.index.Get(key)
	if ok {
		e := val.(*list.Element)
		c.currentSize -= e.Value.(entry).Value.Size()
		c.currentSize += valueSize
		e.Value = entry{key, value}
		c.entries.MoveToFront(e)
	} else {
		e := c.entries.PushFront(entry{key, value})
		c.index.Insert(key, e)
		c.currentSize += valueSize
	}

	var evictedValues []ValueType
	for c.currentSize > c.maxSize {
		evictedValues = append(evictedValues, c.evictOne())
	}

	return evictedValues, nil
}

func (c *radixCache) eraseInternal(key string) (value ValueType) {
	val, ok := c.index.Get(key)
	if !ok {
		return
	}

	e := val.(*list.Element)
	deletedEntry := e.Value.(entry).Value
	c.currentSize -= deletedEntry.Size()

	c.index.Delete(key)
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

	val, ok := c.index.Get(key)
	if !ok {
		return
	}
	e := val.(*list.Element)
	c.entries.MoveToFront(e)

	return e.Value.(entry).Value
}

func (c *radixCache) LookUpWithoutChangingOrder(key string) (value ValueType) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	val, ok := c.index.Get(key)
	if !ok {
		return
	}
	e := val.(*list.Element)

	return e.Value.(entry).Value
}

func (c *radixCache) UpdateWithoutChangingOrder(key string, value ValueType) error {
	if value == nil {
		return ErrInvalidEntry
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	val, ok := c.index.Get(key)
	if !ok {
		return ErrEntryNotExist
	}

	e := val.(*list.Element)
	if value.Size() != e.Value.(entry).Value.Size() {
		return ErrInvalidUpdateEntrySize
	}

	e.Value = entry{key, value}
	// The radix index holds the pointer to the list element.
	// We've mutated the element in place, so no need to update index.
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
	var keysToDelete []string
	c.index.WalkPrefix(prefix, func(key string, value interface{}) bool {
		keysToDelete = append(keysToDelete, key)
		return false
	})
	c.mu.RUnlock()

	if len(keysToDelete) > 0 {
		c.mu.Lock()
		defer c.mu.Unlock()
		for _, key := range keysToDelete {
			c.eraseInternal(key)
		}
	}
}
