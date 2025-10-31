// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file  except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an  "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ttlcache

import (
	"math"
	"sync"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/clock"
)

// item represents an item in the cache.
type item[V any] struct {
	value      V
	expiration int64 // Unix nano timestamp
}

// Cache is a generic, thread-safe cache with a fixed TTL for each entry.
// It is optimized for reads and periodically cleans up expired items.
type Cache[K comparable, V any] struct {
	items    map[K]item[V]
	mu       sync.RWMutex
	ttl      time.Duration
	stopChan chan struct{}
	clock    clock.Clock
}

// New creates a new TTL cache.
// ttl: The time-to-live for items in the cache. If zero or negative, the items never expire.
// cleanupInterval: The interval at which expired items are cleaned up. If zero or negative, the cleanup won't happen.
func New[K comparable, V any](ttl, cleanupInterval time.Duration) *Cache[K, V] {
	return newWithClock[K, V](ttl, cleanupInterval, &clock.RealClock{})
}

func newWithClock[K comparable, V any](ttl, cleanupInterval time.Duration, clk clock.Clock) *Cache[K, V] {
	if ttl <= 0 {
		// No TTL means items never expire, which might not be what the user wants
		// for a TTL cache, but we can support  it. The cleanup goroutine won't run.
		return &Cache[K, V]{
			items: make(map[K]item[V]),
			ttl:   0, // No expiration
			clock: clk,
		}
	}

	c := &Cache[K, V]{
		items:    make(map[K]item[V]),
		ttl:      ttl,
		stopChan: make(chan struct{}),
		clock:    clk,
	}

	if cleanupInterval <= 0 {
		return c
	}

	go c.runCleanup(cleanupInterval)

	return c
}

// Set adds an item to the cache. If the key already exists, its value and
// expiration are updated.
func (c *Cache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	expiration := int64(math.MaxInt64)
	if c.ttl > 0 {
		expiration = c.clock.Now().Add(c.ttl).UnixNano()
	}

	c.items[key] = item[V]{
		value:      value,
		expiration: expiration,
	}
}

// Get retrieves an item from the cache. It returns the value and a boolean
// indicating whether the key  was found and not expired.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	it, found := c.items[key]
	if !found || it.expiration < c.clock.Now().UnixNano() {
		var zero V
		return zero, false
	}

	return it.value, true
}

// Delete removes an item from the cache.
func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Stop terminates the background cleanup goroutine. It should be called when
// the cache is no longer needed to prevent goroutine leaks.
func (c *Cache[K, V]) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stopChan != nil {
		close(c.stopChan)
		c.stopChan = nil
	}
}

// runCleanup is  the background goroutine that periodically removes expired items.
func (c *Cache[K, V]) runCleanup(interval time.Duration) {
	timer := c.clock.After(interval)

	for {
		select {
		case <-timer:
			c.deleteExpired()
			timer = c.clock.After(interval)
		case <-c.stopChan:
			return
		}
	}
}

// deleteExpired iterates through the cache and removes expired items.
// It also recreates the map in order to reclaim the memory since Golang maps don't ever contract.
func (c *Cache[K, V]) deleteExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.clock.Now().UnixNano()
	newItems := make(map[K]item[V], len(c.items))
	for k, v := range c.items {
		if v.expiration >= now {
			newItems[k] = v
		}
	}
	c.items = newItems
}
