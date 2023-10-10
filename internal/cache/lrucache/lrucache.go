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

package lrucache

import (
	"bytes"
	"container/list"
	"encoding/gob"
	"fmt"
	"reflect"
	"sync"
)

// Cache is a LRU cache for any lrucache.ValueType indexed by string keys.
// External synchronization is required. Gob encoding/decoding is supported as long as
// all values are registered using gob.Register.
//
// May be used directly as a field in a larger struct. Must be created with New
// or initialized using gob decoding.
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

	// All exposed methods are guarded by this Mutex. This means that only one
	// method of this cache will be executed at a time, and other methods will
	// be blocked until the current method completes.
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
func New(maxSize uint64) (c Cache) {
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

	// INVARIANT: entries.Len() <= maxSize
	if !(c.currentSize <= c.maxSize) {
		panic(fmt.Sprintf("Length %v over maxSize %v", c.entries.Len(), c.maxSize))
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
	value ValueType) []ValueType {
	if value == nil {
		panic("nil values are not supported")
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

	return evictedValues
}

// Erase any entry for the supplied key, also returns the value of erased key.
// LOCK_EXCLUDED(c.mu)
func (c *Cache) Erase(key string) (value ValueType) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e := c.index[key]
	if e == nil {
		return
	}

	deletedEntry := e.Value.(entry).Value
	c.currentSize -= value.Size()

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
	e := c.index[key]
	if e == nil {
		return
	}

	// This is now the most recently used entry.
	c.entries.MoveToFront(e)

	// Return the value.
	return e.Value.(entry).Value
}

////////////////////////////////////////////////////////////////////////
// Gob encoding
////////////////////////////////////////////////////////////////////////

func (c *Cache) GobEncode() (b []byte, err error) {
	// Implementation note: we have a custom gob encoding method because it's not
	// clear from encoding/gob's documentation that its flattening process won't
	// ruin our list and map values. Even if that works out fine, we don't need
	// the redundant index on the wire.

	// Make sure no inflight cache operation.
	c.mu.Lock()
	defer c.mu.Unlock()

	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)

	// Create a slice containing all of our entries, in order by recency of use.
	entrySlice := make([]entry, 0, c.entries.Len())
	for e := c.entries.Front(); e != nil; e = e.Next() {
		entrySlice = append(entrySlice, e.Value.(entry))
	}

	// Encode the maxSize.
	if err = encoder.Encode(c.maxSize); err != nil {
		err = fmt.Errorf("encoding maxSize: %v", err)
		return
	}

	// Encoding the currentSize.
	if err = encoder.Encode(c.currentSize); err != nil {
		err = fmt.Errorf("encoding currentSize: %v", err)
		return
	}

	// Encode the entries.
	if err = encoder.Encode(entrySlice); err != nil {
		err = fmt.Errorf("encoding entries: %v", err)
		return
	}

	b = buf.Bytes()
	return
}

func (c *Cache) GobDecode(b []byte) (err error) {
	buf := bytes.NewBuffer(b)
	decoder := gob.NewDecoder(buf)

	// Decode the maxSize.
	var maxSize uint64
	if err = decoder.Decode(&maxSize); err != nil {
		err = fmt.Errorf("decoding maxSize: %v", err)
		return
	}

	// Decode currentSize.
	var currentSize uint64
	if err = decoder.Decode(&currentSize); err != nil {
		err = fmt.Errorf("decoding currentSize: %v", err)
		return
	}

	*c = New(maxSize)

	// Decode the entries.
	var entrySlice []entry
	if err = decoder.Decode(&entrySlice); err != nil {
		err = fmt.Errorf("decoding entries: %v", err)
		return
	}

	// Store each.
	for _, entry := range entrySlice {
		e := c.entries.PushBack(entry)
		c.index[entry.Key] = e
	}

	c.currentSize = currentSize
	return
}
