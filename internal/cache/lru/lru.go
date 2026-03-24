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
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/locker"
	hashicorplru "github.com/hashicorp/golang-lru/v2"
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

	// Sum of entry.Value.Size() of all the entries in the cache.
	currentSize uint64

	// hashicorp/golang-lru cache
	lru *hashicorplru.Cache[string, *entry]

	// All public methods of this Cache uses this RW mutex based locker while
	// accessing/updating Cache's data.
	mu locker.RWLocker
}

type ValueType interface {
	Size() uint64
}

type entry struct {
	Value ValueType
	Size  uint64
}

// NewCache returns the reference of cache object by initialising the cache with
// the supplied maxSize, which must be greater than zero.
func NewCache(maxSize uint64) *Cache {
	// We use an extremely large capacity because we manage eviction manually by byte size.
	// Hashicorp LRU is managed by entry count, but we need byte size limit.
	l, err := hashicorplru.New[string, *entry](math.MaxInt32)
	if err != nil {
		panic(fmt.Errorf("failed to create lru cache: %w", err))
	}

	c := &Cache{
		maxSize: maxSize,
		lru:     l,
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
}

func (c *Cache) evictOne() ValueType {
	_, e, ok := c.lru.RemoveOldest()
	if !ok {
		panic("evictOne called on empty cache")
	}

	c.currentSize -= e.Size
	// The key is already removed from c.lru by RemoveOldest
	return e.Value
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

	valueSize := value.Size()
	if valueSize > c.maxSize {
		return nil, ErrInvalidEntrySize
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.lru.Peek(key)
	if ok {
		// Update an entry if already exist.
		c.currentSize -= e.Size
		c.currentSize += valueSize
		c.lru.Add(key, &entry{Value: value, Size: valueSize})
	} else {
		// Add the entry if already doesn't exist.
		c.lru.Add(key, &entry{Value: value, Size: valueSize})
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
func (c *Cache) Erase(key string) (value ValueType) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.lru.Peek(key)
	if !ok {
		return nil
	}

	c.currentSize -= e.Size
	c.lru.Remove(key)

	return e.Value
}

// LookUp a previously-inserted value for the given key. Return nil if no
// value is present.
func (c *Cache) LookUp(key string) (value ValueType) {
	// golang-lru Get is thread-safe, but we use our RWLock to ensure
	// consistency with currentSize checks if any, though not strictly needed here.
	// We keep Lock to preserve previous semantics where LookUp changes order and is protected.
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.lru.Get(key)
	if !ok {
		return nil
	}
	return e.Value
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

	e, ok := c.lru.Peek(key)
	if !ok {
		return nil
	}

	return e.Value
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

	valueSize := value.Size()

	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.lru.Peek(key)
	if !ok {
		return ErrEntryNotExist
	}

	if valueSize != e.Size {
		return ErrInvalidUpdateEntrySize
	}

	// Update the underlying value without changing the LRU order.
	// Since e is a pointer, we can safely mutate it here while under Lock.
	e.Value = value
	e.Size = valueSize

	return nil
}

// UpdateSize updates the currentSize accounting when an entry's size has changed.
// This is needed for entries whose size grows incrementally (e.g., sparse files).
// Eviction is deferred until the next Insert() call.
// The entry's order in the LRU is not changed.
func (c *Cache) UpdateSize(key string, sizeDelta uint64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.lru.Peek(key)
	if !ok {
		return ErrEntryNotExist
	}

	// Update currentSize accounting
	c.currentSize += sizeDelta
	// We need to update the size of the entry.
	e.Size += sizeDelta

	return nil
}

// EraseEntriesWithGivenPrefix erases all entries that match a given prefix.
func (c *Cache) EraseEntriesWithGivenPrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	keys := c.lru.Keys()

	for _, key := range keys {
		if strings.HasPrefix(key, prefix) {
			if e, ok := c.lru.Peek(key); ok {
				c.currentSize -= e.Size
				c.lru.Remove(key)
			}
		}
	}
}
