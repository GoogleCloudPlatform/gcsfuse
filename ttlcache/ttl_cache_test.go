// Copyright 2025 Google LLC
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

package ttlcache

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/clock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTTLCache(t *testing.T) {
	startTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	simClock := clock.NewSimulatedClock(startTime)
	ttl := 10 * time.Minute

	cache := newWithClock[string, string](ttl, 2*ttl, simClock)

	require.NotNil(t, cache)
	assert.Equal(t, ttl, cache.ttl)
	assert.Equal(t, simClock, cache.clock)
	assert.NotNil(t, cache.items)
}

func TestTTLCache_SetAndGet(t *testing.T) {
	startTime := time.Now()
	simClock := clock.NewSimulatedClock(startTime)
	cache := newWithClock[string, string](5*time.Minute, 10*time.Minute, simClock)

	key := "myKey"
	value := "myValue"

	cache.Set(key, value)

	// Get immediately, should exist
	retrievedValue, ok := cache.Get(key)
	assert.True(t, ok)
	assert.Equal(t, value, retrievedValue)
}

func TestTTLCache_Get_NotFound(t *testing.T) {
	startTime := time.Now()
	simClock := clock.NewSimulatedClock(startTime)
	cache := newWithClock[string, string](5*time.Minute, 10*time.Minute, simClock)

	retrievedValue, ok := cache.Get("nonExistentKey")

	assert.False(t, ok)
	assert.Equal(t, "", retrievedValue)
}

func TestTTLCache_Get_Expired(t *testing.T) {
	startTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	simClock := clock.NewSimulatedClock(startTime)
	ttl := 10 * time.Minute
	cache := newWithClock[string, string](ttl, 2*ttl, simClock)

	key := "myKey"
	value := "myValue"
	cache.Set(key, value)

	// Advance time just before expiration
	simClock.AdvanceTime(ttl - 1*time.Second)
	retrievedValue, ok := cache.Get(key)
	assert.True(t, ok)
	assert.Equal(t, value, retrievedValue)

	// Advance time to the point of expiration. `expireAt` is `startTime.Add(ttl)`.
	// `time.After` is true if `t > u`. So if `now == expireAt`, it's not expired.
	simClock.AdvanceTime(1 * time.Second) // Now at exactly TTL from start
	retrievedValue, ok = cache.Get(key)
	assert.True(t, ok, "item should not be expired at the exact expiration time")
	assert.Equal(t, value, retrievedValue)

	// Advance time just past expiration
	simClock.AdvanceTime(1 * time.Nanosecond) // Now just after TTL
	retrievedValue, ok = cache.Get(key)
	assert.False(t, ok, "item should be expired after expiration time")
	assert.Equal(t, "", retrievedValue)
}

func TestTTLCache_Set_Update(t *testing.T) {
	startTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	simClock := clock.NewSimulatedClock(startTime)
	ttl := 10 * time.Minute
	cache := newWithClock[string, string](ttl, 2*ttl, simClock)

	key := "myKey"
	value1 := "value1"
	value2 := "value2"

	// Set initial value
	cache.Set(key, value1)

	// Advance time, but not enough to expire
	simClock.AdvanceTime(ttl / 2)

	// Update value
	cache.Set(key, value2)

	// Advance time again, past the original expiration time but not the new one
	simClock.AdvanceTime(ttl / 2)

	// The item should still be there because its TTL was reset
	retrievedValue, ok := cache.Get(key)
	assert.True(t, ok)
	assert.Equal(t, value2, retrievedValue)

	// Advance time past the new expiration time
	simClock.AdvanceTime(ttl/2 + 1*time.Nanosecond)
	retrievedValue, ok = cache.Get(key)
	assert.False(t, ok)
	assert.Equal(t, "", retrievedValue)
}

func TestTTLCache_Set_ZeroOrNegativeTTL(t *testing.T) {
	testCases := []struct {
		name string
		ttl  time.Duration
	}{
		{"ZeroTTL", 0},
		{"NegativeTTL", -5 * time.Minute},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			startTime := time.Now()
			simClock := clock.NewSimulatedClock(startTime)
			cache := newWithClock[string, string](tc.ttl, 2*tc.ttl, simClock)
			cache.Set("someKey", "someValue")

			val, ok := cache.Get("someKey")

			assert.True(t, ok)
			assert.Equal(t, "someValue", val)
		})
	}
}

func TestTTLCache_Delete(t *testing.T) {
	startTime := time.Now()
	simClock := clock.NewSimulatedClock(startTime)
	cache := newWithClock[string, string](5*time.Minute, 10*time.Minute, simClock)

	key := "myKey"
	value := "myValue"

	cache.Set(key, value)
	retrievedValue, ok := cache.Get(key)
	require.True(t, ok)
	require.Equal(t, value, retrievedValue)

	cache.Delete(key)

	retrievedValue, ok = cache.Get(key)
	assert.False(t, ok)
	assert.Equal(t, "", retrievedValue)
}

func TestTTLCache_Concurrency(t *testing.T) {
	simClock := clock.NewSimulatedClock(time.Now())
	cache := newWithClock[string, int](5*time.Minute, 10*time.Minute, simClock)
	var wg sync.WaitGroup
	numGoroutines := 50
	itemsPerGoroutine := 20

	// Concurrent Set and Get
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for j := 0; j < itemsPerGoroutine; j++ {
				key := fmt.Sprintf("key-%d-%d", g, j)
				value := g*itemsPerGoroutine + j

				cache.Set(key, value)

				retrievedValue, ok := cache.Get(key)
				if assert.True(t, ok) {
					assert.Equal(t, value, retrievedValue)
				}
			}
		}(i)
	}
	wg.Wait()

	// Advance time to expire everything
	simClock.AdvanceTime(10*time.Minute + 1*time.Nanosecond)

	// Concurrent reads of expired items
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for j := 0; j < itemsPerGoroutine; j++ {
				key := fmt.Sprintf("key-%d-%d", g, j)
				_, ok := cache.Get(key)
				assert.False(t, ok, "key should be expired")
			}
		}(i)
	}
	wg.Wait()
}
