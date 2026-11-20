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

	golru "github.com/google/go-lru"
	"github.com/googlecloudplatform/gcsfuse/v4/internal/locker"
)

// Predefined errors returned by the Cache.
var (
	ErrInvalidEntrySize       = golru.ErrInvalidEntrySize
	ErrInvalidEntry           = errors.New("nil values are not supported")
	ErrInvalidUpdateEntrySize = errors.New("size of entry to be updated is not same as existing size")
	ErrEntryNotExist          = golru.ErrEntryNotExist
)

// minGolruMaxSize ensures the underlying golru.Cache capacity is at least 1 GiB
// in scaled units so that zero-size entries never trigger golru's zero-weight
// entry count cap (which only activates when zeroSizeCount >= golru.maxSize).
const minGolruMaxSize = uint64(1) << 30

// Cache is a LRU cache for any lru.ValueType indexed by string keys.
// That means entry's value should be a lru.ValueType.
type Cache struct {
	/////////////////////////
	// Constant data
	/////////////////////////

	// Maximum capacity of the cache in bytes.
	maxSize uint64

	// Scale factor applied to entry weights in the underlying golru.Cache.
	scale uint64

	// Notified of every value entering or leaving the cache, if not nil.
	sizeObserver SizeObserver

	/////////////////////////
	// Mutable state
	/////////////////////////

	// Sum of entry.Value.Size() of all the entries in the cache.
	currentSize uint64

	// Underlying go-lru cache instance.
	//
	// INVARIANT: currentSize <= maxSize
	// INVARIANT: Each element is of type entry
	cache golru.Cache[entry]

	// Tracks entries whose UpdateSize growth temporarily exceeded maxSize and
	// whose weight update in golru is deferred until the next Insert/Erase.
	hasDeferredOverflow bool
	deferredGolruSize   map[string]uint64

	// All public methods of this Cache uses this RW mutex based locker while
	// accessing/updating Cache's data.
	mu locker.RWLocker
}

type ValueType interface {
	Size() uint64
}

// SizeObserver is notified of every change to the contents of a Cache: with
// delta = +size when a value enters the cache, and with delta = -size when it
// leaves it, whether erased, evicted or replaced by an Insert of the same key.
// size is the one the cache accounts for the value, captured when the value
// was inserted, so the deltas always add up to the cache's current size.
//
// It is called with the cache's lock held: it must be fast, must not block and
// must not call into the cache.
type SizeObserver func(value ValueType, delta int64)

// Option configures a Cache when it is created.
type Option func(*Cache)

// WithSizeObserver makes the cache notify o of every change to its contents.
func WithSizeObserver(o SizeObserver) Option {
	return func(c *Cache) {
		c.sizeObserver = o
	}
}

type entry struct {
	value ValueType
	size  uint64
}

func (c *Cache) weigh(key string, ent entry) uint64 {
	if c.hasDeferredOverflow {
		if golruSize, ok := c.deferredGolruSize[key]; ok {
			return golruSize * c.scale
		}
	}
	return ent.size * c.scale
}

func (c *Cache) initGolruCache(enableInvariantsCheck bool) {
	c.scale = 1
	golruMaxSize := c.maxSize
	if c.maxSize == 0 {
		golruMaxSize = minGolruMaxSize
	} else if c.maxSize < minGolruMaxSize {
		c.scale = minGolruMaxSize
		golruMaxSize = c.maxSize * c.scale
	}

	golruOpts := []golru.Option{
		golru.WithWeigher(c.weigh),
		golru.WithCompactionThreshold(math.MaxFloat64),
		golru.WithEvictionThreshold(math.MaxFloat64),
		golru.WithInvariantChecking(enableInvariantsCheck),
	}

	c.cache = golru.NewArenaRadixCache[entry](golruMaxSize, golruOpts...)
}

// NewCache returns the reference of cache object by initialising the cache with
// the supplied maxSize.
func NewCache(maxSize uint64, opts ...Option) *Cache {
	c := &Cache{
		maxSize: maxSize,
	}
	for _, opt := range opts {
		opt(c)
	}
	c.initGolruCache(false)

	// Set up invariant checking.
	c.mu = locker.NewRW("LRUCache", c.checkInvariants)
	return c
}

// NewCacheWithOptions returns a Cache instance initialized with the
// supplied maxSize and lock options.
func NewCacheWithOptions(maxSize uint64, lockOpts locker.Options, opts ...Option) *Cache {
	c := &Cache{
		maxSize: maxSize,
	}
	for _, opt := range opts {
		opt(c)
	}
	c.initGolruCache(lockOpts.EnableInvariantsCheck)

	c.mu = locker.NewRWWithOptions("LRUCache", c.checkInvariants, lockOpts)
	return c
}

// notifySizeChange tells the size observer, if any, that value entered
// (delta > 0) or left (delta < 0) the cache.
//
// LOCKS_REQUIRED(c.mu)
func (c *Cache) notifySizeChange(value ValueType, delta int64) {
	if c.sizeObserver != nil {
		c.sizeObserver(value, delta)
	}
}

// reconcileDeferred synchronizes the underlying golru.Cache weights for any
// entries whose UpdateSize growth was deferred while currentSize exceeded maxSize.
//
// LOCKS_REQUIRED(c.mu)
func (c *Cache) reconcileDeferred() {
	for k := range c.deferredGolruSize {
		delete(c.deferredGolruSize, k)
		if ent, ok := c.cache.Peek(k); ok {
			_ = c.cache.Replace(k, ent)
		}
	}
	c.hasDeferredOverflow = false
}

// checkInvariants panic if any internal invariants have been violated.
func (c *Cache) checkInvariants() {
	st := c.cache.Stats()
	golruCurrentSize := st.CurrentSize / c.scale
	currentSize := c.currentSize
	if c.sizeObserver == nil && !c.hasDeferredOverflow {
		currentSize = golruCurrentSize
	} else if !c.hasDeferredOverflow && currentSize != golruCurrentSize {
		panic(fmt.Sprintf("Size mismatch: currentSize %v vs golru CurrentSize %v", currentSize, golruCurrentSize))
	}

	// INVARIANT: currentSize <= maxSize (unless deferred overflow from UpdateSize is pending)
	if !c.hasDeferredOverflow && currentSize > c.maxSize {
		panic(fmt.Sprintf("CurrentSize %v over maxSize %v", currentSize, c.maxSize))
	}

	// INVARIANT: Each element is of type entry and sum of entry sizes matches currentSize
	var sumSize uint64
	var count int
	for ent := range c.cache.Values() {
		if ent.value == nil {
			panic("Unexpected nil entry value")
		}
		sumSize += ent.size
		count++
	}

	if sumSize != currentSize {
		panic(fmt.Sprintf("Size mismatch: sum of entry sizes %v vs currentSize %v", sumSize, currentSize))
	}

	if count != st.Len {
		panic(fmt.Sprintf(
			"Length mismatch: %v vs. %v",
			count,
			st.Len))
	}
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

	valueSize := value.Size()
	if valueSize > c.maxSize {
		return nil, ErrInvalidEntrySize
	}

	if c.sizeObserver == nil && !c.hasDeferredOverflow {
		evictedEntries, _ := c.cache.Put(key, entry{value: value, size: valueSize})
		if len(evictedEntries) == 0 {
			return nil, nil
		}
		evictedValues := make([]ValueType, len(evictedEntries))
		for i, ev := range evictedEntries {
			evictedValues[i] = ev.value
		}
		return evictedValues, nil
	}

	oldEnt, hasOld := c.cache.Peek(key)
	if hasOld {
		// Update an entry if already exist.
		c.currentSize -= oldEnt.size
		c.notifySizeChange(oldEnt.value, -int64(oldEnt.size))
		if c.hasDeferredOverflow {
			delete(c.deferredGolruSize, key)
		}
	}

	c.currentSize += valueSize
	c.notifySizeChange(value, int64(valueSize))

	var evictedValues []ValueType
	if c.hasDeferredOverflow {
		if hasOld {
			_, _ = c.cache.Delete(key)
		}
		if c.currentSize > c.maxSize {
			var keys []string
			for k := range c.cache.Keys() {
				keys = append(keys, k)
			}
			for i := len(keys) - 1; i >= 0 && c.currentSize > c.maxSize; i-- {
				if ev, ok := c.cache.Delete(keys[i]); ok {
					delete(c.deferredGolruSize, keys[i])
					c.currentSize -= ev.size
					c.notifySizeChange(ev.value, -int64(ev.size))
					evictedValues = append(evictedValues, ev.value)
				}
			}
		}
		c.reconcileDeferred()
	}

	evictedEntries, _ := c.cache.Put(key, entry{value: value, size: valueSize})

	if len(evictedEntries) > 0 {
		if evictedValues == nil {
			evictedValues = make([]ValueType, 0, len(evictedEntries))
		}
		for _, ev := range evictedEntries {
			c.currentSize -= ev.size
			c.notifySizeChange(ev.value, -int64(ev.size))
			evictedValues = append(evictedValues, ev.value)
		}
	}

	return evictedValues, nil
}

// eraseInternal removes any entry for the supplied key from the cache without acquiring locks.
// It returns the value of the erased key, or nil if not present.
// LOCKS_REQUIRED(c.mu)
func (c *Cache) eraseInternal(key string) (value ValueType) {
	deletedEntry, ok := c.cache.Delete(key)
	if !ok {
		return nil
	}

	if c.sizeObserver != nil || c.hasDeferredOverflow {
		c.currentSize -= deletedEntry.size
	}
	if c.hasDeferredOverflow {
		delete(c.deferredGolruSize, key)
		if c.currentSize <= c.maxSize {
			c.reconcileDeferred()
		}
	}
	c.notifySizeChange(deletedEntry.value, -int64(deletedEntry.size))

	return deletedEntry.value
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

	ent, ok := c.cache.Get(key)
	if !ok {
		return nil
	}

	return ent.value
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

	ent, ok := c.cache.Peek(key)
	if !ok {
		return nil
	}

	return ent.value
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

	ent, ok := c.cache.Peek(key)
	if !ok {
		return ErrEntryNotExist
	}

	if value.Size() != ent.size {
		return ErrInvalidUpdateEntrySize
	}

	// The size stays the same, but the value it is accounted to changes.
	c.notifySizeChange(ent.value, -int64(ent.size))
	c.notifySizeChange(value, int64(ent.size))
	_ = c.cache.Replace(key, entry{value: value, size: ent.size})

	return nil
}

// UpdateSize updates the currentSize accounting when an entry's size has changed.
// This is needed for entries whose size grows incrementally (e.g., sparse files).
// Eviction is deferred until the next Insert() call.
// The entry's order in the LRU is not changed.
func (c *Cache) UpdateSize(key string, sizeDelta uint64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	ent, ok := c.cache.Peek(key)
	if !ok {
		return ErrEntryNotExist
	}

	oldSize := ent.size
	if c.sizeObserver == nil && !c.hasDeferredOverflow {
		c.currentSize = c.cache.Stats().CurrentSize / c.scale
	}
	prevTotalSize := c.currentSize
	ent.size += sizeDelta

	// Update currentSize accounting
	// Note: This may temporarily violate currentSize <= maxSize invariant
	// Eviction will happen on the next Insert() call
	c.currentSize += sizeDelta
	c.notifySizeChange(ent.value, int64(sizeDelta))

	if !c.hasDeferredOverflow && sizeDelta <= c.maxSize-prevTotalSize {
		_ = c.cache.Replace(key, ent)
	} else {
		if !c.hasDeferredOverflow {
			c.hasDeferredOverflow = true
			if c.deferredGolruSize == nil {
				c.deferredGolruSize = make(map[string]uint64)
			}
		}
		if _, exists := c.deferredGolruSize[key]; !exists {
			c.deferredGolruSize[key] = oldSize
		}
		_ = c.cache.Replace(key, ent)
	}

	return nil
}

func (c *Cache) EraseEntriesWithGivenPrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.sizeObserver == nil && !c.hasDeferredOverflow {
		c.cache.DeletePrefix(prefix)
		return
	}

	matched := false
	for key, ent := range c.cache.All() {
		if strings.HasPrefix(key, prefix) {
			matched = true
			if c.hasDeferredOverflow {
				delete(c.deferredGolruSize, key)
			}
			c.currentSize -= ent.size
			c.notifySizeChange(ent.value, -int64(ent.size))
		}
	}
	if !matched {
		return
	}

	c.cache.DeletePrefix(prefix)
	if c.hasDeferredOverflow && c.currentSize <= c.maxSize {
		c.reconcileDeferred()
	}
}
