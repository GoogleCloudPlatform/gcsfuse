// Copyright 2023 Google LLC
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
	"strings"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/locker"
)

// Predefined errors returned by the Cache.
var (
	ErrInvalidEntrySize       = errors.New("size of the entry is more than the cache's maxSize")
	ErrInvalidEntry           = errors.New("nil values are not supported")
	ErrInvalidUpdateEntrySize = errors.New("size of entry to be updated is not same as existing size")
	ErrEntryNotExist          = errors.New("entry with given key does not exist")
)

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

	// Sum of size of all the entries in the cache.
	currentSize uint64

	// Function to calculate the actual cost of a single entry towards currentSize
	sizeCalcFunc func(ValueType) uint64

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

	// All public methods of this Cache uses this RW mutex based locker while
	// accessing/updating Cache's data.
	mu locker.RWLocker
}

type ValueType interface {
	Size() uint64
}

type entry struct {
	Key   string
	Value ValueType
}

func defaultSizeCalc(v ValueType) uint64 {
	return v.Size()
}

// NewCache returns the reference of cache object by initialising the cache with
// the supplied maxSize, which must be greater than zero.
func NewCache(maxSize uint64) *Cache {
	return NewCacheWithCustomSizeCalc(maxSize, defaultSizeCalc)
}

// NewCacheWithCustomSizeCalc returns a cache object that uses a custom
// function to calculate the size footprint of each entry in the cache.
func NewCacheWithCustomSizeCalc(maxSize uint64, sizeCalcFunc func(ValueType) uint64) *Cache {
	c := &Cache{
		maxSize:      maxSize,
		index:        make(map[string]*list.Element),
		sizeCalcFunc: sizeCalcFunc,
	}

	// Set up invariant checking.
	c.mu = locker.NewRW("LRUCache", c.checkInvariants)
	return c
}

// checkInvariants panic if any internal invariants have been violated.
func (c *Cache) checkInvariants() {
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
	c.currentSize -= c.sizeCalcFunc(evictedEntry)

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
func (c *Cache) Insert(
	key string,
	value ValueType) ([]ValueType, error) {
	if value == nil {
		return nil, ErrInvalidEntry
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	valueSize := c.sizeCalcFunc(value)
	if valueSize > c.maxSize {
		return nil, ErrInvalidEntrySize
	}

	e, ok := c.index[key]
	if ok {
		// Update an entry if already exist.
		c.currentSize -= c.sizeCalcFunc(e.Value.(entry).Value)
		c.currentSize += valueSize
		e.Value = entry{key, value}
		c.entries.MoveToFront(e)
	} else {
		// Add the entry if already doesn't exist.
		e := c.entries.PushFront(entry{key, value})
		c.index[key] = e
		c.currentSize += valueSize
	}

	var evictedValues []ValueType
	// Evict until we're at or below maxSize.
	for c.currentSize > c.maxSize {
		evictedValues = append(evictedValues, c.evictOne())
	}

	return evictedValues, nil
}

// eraseInternal removes any entry for the supplied key from the cache without acquiring locks.
// It returns the value of the erased key, or nil if not present.
// LOCKS_REQUIRED(c.mu)
func (c *Cache) eraseInternal(key string) (value ValueType) {
	e, ok := c.index[key]
	if !ok {
		return
	}

	deletedEntry := e.Value.(entry).Value
	c.currentSize -= c.sizeCalcFunc(deletedEntry)

	delete(c.index, key)
	c.entries.Remove(e)

	return deletedEntry
}

// eraseKeys removes a list of keys from the cache.
func (c *Cache) eraseKeys(keys []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, key := range keys {
		c.eraseInternal(key)
	}
}

// Erase any entry for the supplied key, also returns the value of erased key.
func (c *Cache) Erase(key string) (value ValueType) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.eraseInternal(key)
}

// LookUp a previously-inserted value for the given key. Return nil if no
// value is present.
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

// LookUpWithoutChangingOrder looks up previously-inserted value for a given key
// without changing the order of entries in the cache. Return nil if no value
// is present.
//
// Note: Because this look up doesn't change the order, it only acquires and
// releases read lock.
func (c *Cache) LookUpWithoutChangingOrder(key string) (value ValueType) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Consult the index.
	e, ok := c.index[key]
	if !ok {
		return
	}

	// Return the value.
	return e.Value.(entry).Value
}

// UpdateWithoutChangingOrder updates entry with the given key in cache with
// given value without changing order of entries in cache, returning error if an
// entry with given key doesn't exist. Also, the size of value for entry
// shouldn't be updated with this method (use c.Insert for updating size).
func (c *Cache) UpdateWithoutChangingOrder(
	key string,
	value ValueType) error {
	if value == nil {
		return ErrInvalidEntry
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.index[key]
	if !ok {
		return ErrEntryNotExist
	}

	if c.sizeCalcFunc(value) != c.sizeCalcFunc(e.Value.(entry).Value) {
		return ErrInvalidUpdateEntrySize
	}

	e.Value = entry{key, value}
	c.index[key] = e

	return nil
}

// UpdateSize updates the currentSize accounting when an entry's size has changed.
// This is needed for entries whose size grows incrementally (e.g., sparse files).
// Eviction is deferred until the next Insert() call.
// The entry's order in the LRU is not changed.
func (c *Cache) UpdateSize(key string, sizeDelta uint64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, ok := c.index[key]
	if !ok {
		return ErrEntryNotExist
	}

	// Update currentSize accounting
	// Note: This may temporarily violate currentSize <= maxSize invariant
	// Eviction will happen on the next Insert() call
	c.currentSize += sizeDelta

	return nil
}

func (c *Cache) EraseEntriesWithGivenPrefix(prefix string) {
	c.mu.RLock()
	var keysToDelete []string
	for key := range c.index {
		if strings.HasPrefix(key, prefix) {
			keysToDelete = append(keysToDelete, key)
		}
	}
	c.mu.RUnlock()

	if len(keysToDelete) > 0 {
		c.eraseKeys(keysToDelete)
	}
}

// SetSizeCalcFunc overrides the default size calculation logic for the LRU entries.
func (c *Cache) SetSizeCalcFunc(sizeCalcFunc func(ValueType) uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sizeCalcFunc = sizeCalcFunc
}
