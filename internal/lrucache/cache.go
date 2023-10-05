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

// An LRU cache for arbitrary values indexed by string keys. External
// synchronization is required. Gob encoding/decoding is supported as long as
// all values are registered using gob.Register.
//
// May be used directly as a field in a larger struct. Must be created with New
// or initialized using gob decoding.
type Cache struct {
	/////////////////////////
	// Constant data
	/////////////////////////

	// INVARIANT: capacity > 0
	capacity uint64

	/////////////////////////
	// Mutable state
	/////////////////////////

	// List of cache entries, with least recently used at the tail.
	//
	// INVARIANT: entries.Len() <= capacity
	// INVARIANT: Each element is of type entry
	entries list.List

	// Index of elements by name.
	//
	// INVARIANT: For each k, v: v.Value.(entry).Key == k
	// INVARIANT: Contains all and only the elements of entries
	index map[string]*list.Element

	// Guards all the methods of this cache.
	mu sync.Mutex

	usedSize uint64
}

type ValueType interface {
	Size() uint64
}

type entry struct {
	Key   string
	Value ValueType
}

// Initialize a cache with the supplied capacity, which must be greater than
// zero.
func New(capacity uint64) (c Cache) {
	c.capacity = capacity
	c.index = make(map[string]*list.Element)
	return
}

// Panic if any internal invariants have been violated. The careful user can
// arrange to call this at crucial moments.
func (c *Cache) CheckInvariants() {
	// INVARIANT: capacity > 0
	if !(c.capacity > 0) {
		panic(fmt.Sprintf("Invalid capacity: %v", c.capacity))
	}

	// INVARIANT: entries.Len() <= capacity
	if !(c.usedSize <= c.capacity) {
		panic(fmt.Sprintf("Length %v over capacity %v", c.entries.Len(), c.capacity))
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
	c.usedSize -= evictedEntry.Size()

	c.entries.Remove(e)
	delete(c.index, key)

	return evictedEntry
}

////////////////////////////////////////////////////////////////////////
// Cache interface
////////////////////////////////////////////////////////////////////////

// Insert the supplied value into the cache, overwriting any previous entry for
// the given key. The value must be non-nil.
func (c *Cache) Insert(
	key string,
	value ValueType) []ValueType {
	if value == nil {
		panic("nil values are not supported")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Erase any existing element for this key.
	// If entry already exist
	e, ok := c.index[key]
	if ok {
		c.usedSize -= e.Value.(entry).Value.Size()
		c.usedSize += value.Size()
		e.Value = entry{key, value}
		c.entries.MoveToFront(e)
	} else {
		// Add a new element.
		e := c.entries.PushFront(entry{key, value})
		c.index[key] = e
		c.usedSize += value.Size()
	}

	var evictedEntries []ValueType
	// Evict until we're at or below capacity.
	for c.usedSize > c.capacity {
		evictedEntries = append(evictedEntries, c.evictOne())
	}

	return evictedEntries
}

// Erase any entry for the supplied key.
func (c *Cache) Erase(key string) (value ValueType) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e := c.index[key]
	if e == nil {
		return
	}

	deletedEntry := e.Value.(entry).Value
	c.usedSize -= value.Size()

	delete(c.index, key)
	c.entries.Remove(e)

	return deletedEntry
}

// Look up a previously-inserted value for the given key. Return nil if no
// value is present.
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

	// Encode the capacity.
	if err = encoder.Encode(c.capacity); err != nil {
		err = fmt.Errorf("Encoding capacity: %v", err)
		return
	}

	// Encoding the usedSize.
	if err = encoder.Encode(c.usedSize); err != nil {
		err = fmt.Errorf("Encoding usedSize: %v", err)
		return
	}

	// Encode the entries.
	if err = encoder.Encode(entrySlice); err != nil {
		err = fmt.Errorf("Encoding entries: %v", err)
		return
	}

	b = buf.Bytes()
	return
}

func (c *Cache) GobDecode(b []byte) (err error) {
	buf := bytes.NewBuffer(b)
	decoder := gob.NewDecoder(buf)

	// Decode the capacity.
	var capacity uint64
	if err = decoder.Decode(&capacity); err != nil {
		err = fmt.Errorf("Decoding capacity: %v", err)
		return
	}

	// Decode usedSize.
	var usedSize uint64
	if err = decoder.Decode(&usedSize); err != nil {
		err = fmt.Errorf("Decoding usedSize: %v", err)
		return
	}

	*c = New(capacity)

	// Decode the entries.
	var entrySlice []entry
	if err = decoder.Decode(&entrySlice); err != nil {
		err = fmt.Errorf("Decoding entries: %v", err)
		return
	}

	// Store each.
	for _, entry := range entrySlice {
		e := c.entries.PushBack(entry)
		c.index[entry.Key] = e
	}

	c.usedSize = usedSize
	return
}
