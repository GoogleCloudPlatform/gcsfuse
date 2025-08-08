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
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCache_SetAndGet(t *testing.T) {
	cache := New[string, string](100*time.Millisecond, 10*time.Millisecond)
	defer cache.Stop()

	cache.Set("key1", "value1")
	val, found := cache.Get("key1")

	assert.True(t, found)
	assert.Equal(t, "value1", val)
}

func TestCache_GetExpired(t *testing.T) {
	ttl := 50 * time.Millisecond
	cache := New[string, int](ttl, 10*time.Millisecond)
	defer cache.Stop()

	cache.Set("key1", 123)

	// Wait for item to expire
	time.Sleep(ttl + 10*time.Millisecond)

	val, found := cache.Get("key1")

	assert.False(t, found)
	assert.Equal(t, 0, val) // zero  value for int
}

func TestCache_GetNonExistent(t *testing.T) {
	cache := New[string, int](time.Minute, time.Second)
	defer cache.Stop()

	val, found := cache.Get("non-existent-key")

	assert.False(t, found)
	assert.Equal(t, 0, val)
}

func TestCache_SetOverrides(t *testing.T) {
	cache := New[string, string](time.Minute, time.Second)
	defer cache.Stop()

	cache.Set("key1", "value1")
	cache.Set("key1", "value2")

	val, found := cache.Get("key1")

	assert.True(t, found)
	assert.Equal(t, "value2", val)
}

func TestCache_Delete(t *testing.T) {
	cache := New[string, string](time.Minute, time.Second)
	defer cache.Stop()

	cache.Set("key1", "value1")
	cache.Delete("key1")

	_, found := cache.Get("key1")
	assert.False(t, found)
}

func TestCache_Cleanup(t *testing.T) {
	ttl := 50 * time.Millisecond
	cleanupInterval := 10 * time.Millisecond
	cache := New[string, int](ttl, cleanupInterval)
	defer cache.Stop()

	cache.Set("key1", 123)
	cache.Set("key2", 456)

	// Wait for cleanup to run
	time.Sleep(ttl + cleanupInterval*2)

	cache.mu.RLock()
	_, foundInMap := cache.items["key1"]
	cache.mu.RUnlock()

	assert.False(t, foundInMap, "Expired item should be removed by cleanup goroutine")
}

func TestCache_Concurrency(t *testing.T) {
	cache := New[string, int](100*time.Millisecond, 20*time.Millisecond)
	defer cache.Stop()

	var wg sync.WaitGroup
	numGoroutines := 100
	itemsPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for j := 0; j < itemsPerGoroutine; j++ {
				key := "key-" + strconv.Itoa(g) + "-" + strconv.Itoa(j)
				cache.Set(key, g*itemsPerGoroutine+j)
				_, _ = cache.Get(key)
			}
		}(i)
	}

	wg.Wait()

	// Check one item to see if it's there
	val, found := cache.Get("key-50-50")
	assert.True(t, found)
	assert.Equal(t, 50*itemsPerGoroutine+50, val)
}

func TestCache_Stop(t *testing.T) {
	// This test is tricky to write  perfectly without inspecting runtime internals.
	// We can check that the cache stops functioning as expected after Stop.
	// A more robust test would involve checking goroutine count, but that's flaky.
	ttl := 50 * time.Millisecond
	cleanupInterval := 10 * time.Millisecond
	cache := New[string, int](ttl, cleanupInterval)

	cache.Set("key1", 123)
	cache.Stop()

	// Wait for a potential cleanup cycle
	time.Sleep(cleanupInterval * 2)

	// After stopping, the cleanup should  not run.
	// But Get still checks for expiration.
	// Let's check if we can still set items.
	cache.Set("key2", 456)
	val, found := cache.Get("key2")
	assert.True(t, found)
	assert.Equal(t, 456, val)

	// Wait for key1 to expire
	time.Sleep(ttl)
	_, found = cache.Get("key1")
	assert.False(t, found, "Get should still respect expiration even if cleanup is stopped")
}

func TestCache_NoTTL(t *testing.T) {
	cache := New[string, string](0, 0) // No TTL
	defer cache.Stop()

	cache.Set("key1", "value1")
	time.Sleep(50 * time.Millisecond) //  Wait a bit

	val, found := cache.Get("key1")
	assert.True(t, found)
	assert.Equal(t, "value1", val)
}
