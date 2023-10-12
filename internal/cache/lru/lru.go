// Copyright 2023 Google Inc. All Rights Reserved.
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
	"errors"
	"fmt"
	"reflect"
	"sync"
)

// Predefined error message returned by the Cache.
const InvalidEntrySizeErrorMsg = "size of the entry is more than the cache's maxSize"
const InvalidEntryErrorMsg = "nil values are not supported"

// Cache is a LRU cache for any lru.ValueType indexed by string keys.
// That means entry's value should be a lru.ValueType.
type Cache struct {
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

	// Index of elements by name.
	//
	// INVARIANT: For each k, v: v.Value.(entry).Key == k
	// INVARIANT: Contains all and only the elements of entries
	index map[string]*list.Element

	// All public methods of this Cache uses this mutex while accessing/updating
	// Cache's data. This means when one method is accessing/updating Cache's data,
	// other methods will be blocked until the method in execution completes.
	mu sync.Mutex
}

type ValueType interface {
	Size() uint64
}

type entry struct {
	Key   string
	Value ValueType
}

// New initialize a cache with the supplied maxSize, which must be greater than
// zero.
func NewCache(maxSize uint64) (c Cache) {
	c.maxSize = maxSize
	c.index = make(map[string]*list.Element)
	return
}

// CheckInvariants panic if any internal invariants have been violated.
// The careful user can arrange to call this at crucial moments.
func (c *Cache) CheckInvariants() {
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

	// INVARIANT: For each k, v: v.Value.(entry).Key == k
	// INVARIANT: Contains all and only the elements of entries
	if c.entries.Len() != len(c.index) {
		panic(fmt.Sprintf(
			"Length mismatch: %v vs. %v",
			c.entries.Len(),
			len(c.index)))
	}

	for e := c.entries.Front(); e != nil; e = e.Next() {
		if c.index[e.Value.(entry).Key] != e {
			panic(fmt.Sprintf("Mismatch for key %v", e.Value.(entry).Key))
		}
	}
}

func (c *Cache) evictOne() ValueType {
	e := c.entries.Back()
	key := e.Value.(entry).Key

	evictedEntry := e.Value.(entry).Value
	c.currentSize -= evictedEntry.Size()

	c.entries.Remove(e)
	delete(c.index, key)

	return evictedEntry
}

////////////////////////////////////////////////////////////////////////
// Cache interface
////////////////////////////////////////////////////////////////////////

// Insert the supplied value into the cache, overwriting any previous entry for
// the given key. The value must be non-nil.
// Also returns a slice of ValueType evicted by the new inserted entry.
// LOCK_EXCLUDED(c.mu)
func (c *Cache) Insert(
	key string,
	value ValueType) ([]ValueType, error) {
	if value == nil {
		return nil, errors.New(InvalidEntryErrorMsg)
	}

	if value.Size() > c.maxSize {
		return nil, errors.New(InvalidEntrySizeErrorMsg)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.index[key]
	if ok {
		// Update an entry if already exist.
		c.currentSize -= e.Value.(entry).Value.Size()
		c.currentSize += value.Size()
		e.Value = entry{key, value}
		c.entries.MoveToFront(e)
	} else {
		// Add the entry if already doesn't exist.
		e := c.entries.PushFront(entry{key, value})
		c.index[key] = e
		c.currentSize += value.Size()
	}

	var evictedValues []ValueType
	// Evict until we're at or below maxSize.
	for c.currentSize > c.maxSize {
		evictedValues = append(evictedValues, c.evictOne())
	}

	return evictedValues, nil
}

// Erase any entry for the supplied key, also returns the value of erased key.
// LOCK_EXCLUDED(c.mu)
func (c *Cache) Erase(key string) (value ValueType) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.index[key]
	if !ok {
		return
	}

	deletedEntry := e.Value.(entry).Value
	c.currentSize -= deletedEntry.Size()

	delete(c.index, key)
	c.entries.Remove(e)

	return deletedEntry
}

// LookUp a previously-inserted value for the given key. Return nil if no
// value is present.
// LOCK_EXCLUDED(c.mu)
func (c *Cache) LookUp(key string) (value ValueType) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Consult the index.
	e, ok := c.index[key]
	if !ok {
		return
	}
	// This is now the most recently used entry.
	c.entries.MoveToFront(e)

	// Return the value.
	return e.Value.(entry).Value
}
