// Copyright 2026 Google LLC
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

package lru_test

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stressTestData struct {
	val  int64
	size uint64
}

func (d stressTestData) Size() uint64 {
	return d.size
}

// TestStress_HighConcurrencyReadsAndEvictions tests parallel reads and inserts with heavy evictions.
func TestStress_HighConcurrencyReadsAndEvictions(t *testing.T) {
	const totalMaxSize = 1000
	cache := lru.NewShardedRadixCache(totalMaxSize)
	defer func() {
		if c, ok := cache.(*lru.ShardedRadixCache); ok {
			c.Close()
		}
	}()

	var wg sync.WaitGroup
	numWorkers := 32
	opsPerWorker := 1000

	start := make(chan struct{})

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			<-start
			r := rand.New(rand.NewSource(int64(workerID)))

			for i := 0; i < opsPerWorker; i++ {
				keyID := r.Intn(100)
				key := fmt.Sprintf("dir_%d/file_%d.txt", keyID%10, keyID)
				size := uint64(r.Intn(50) + 1)

				if r.Float32() < 0.7 {
					// Read operation
					_ = cache.LookUp(key)
				} else {
					// Insert operation (drives evictions)
					_, _ = cache.Insert(key, stressTestData{val: int64(i), size: size})
				}
			}
		}(w)
	}

	close(start)
	wg.Wait()

	// Verify invariant: total size in cache must not exceed totalMaxSize
	var calculatedSize uint64
	for i := 0; i < 100; i++ {
		for d := 0; d < 10; d++ {
			key := fmt.Sprintf("dir_%d/file_%d.txt", d, i)
			v := cache.LookUpWithoutChangingOrder(key)
			if v != nil {
				calculatedSize += v.Size()
			}
		}
	}
	assert.LessOrEqual(t, calculatedSize, uint64(totalMaxSize), "Calculated total size of remaining items exceeds totalMaxSize")
}

// TestStress_ConcurrentPrefixEraseWithReadsAndWrites tests EraseEntriesWithGivenPrefix running concurrently with LookUp and Insert.
func TestStress_ConcurrentPrefixEraseWithReadsAndWrites(t *testing.T) {
	const totalMaxSize = 2000
	cache := lru.NewShardedRadixCache(totalMaxSize)
	defer func() {
		if c, ok := cache.(*lru.ShardedRadixCache); ok {
			c.Close()
		}
	}()

	var wg sync.WaitGroup
	numWriters := 8
	numReaders := 16
	numErasers := 4
	duration := 2 * time.Second
	stop := make(chan struct{})

	// Writers
	for w := 0; w < numWriters; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			r := rand.New(rand.NewSource(int64(id)))
			i := 0
			for {
				select {
				case <-stop:
					return
				default:
					dir := r.Intn(5)
					file := r.Intn(50)
					key := fmt.Sprintf("tenant_%d/dir_%d/file_%d", id%2, dir, file)
					_, _ = cache.Insert(key, stressTestData{val: int64(i), size: 10})
					i++
				}
			}
		}(w)
	}

	// Readers
	for rIdx := 0; rIdx < numReaders; rIdx++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			r := rand.New(rand.NewSource(int64(id + 100)))
			for {
				select {
				case <-stop:
					return
				default:
					dir := r.Intn(5)
					file := r.Intn(50)
					key := fmt.Sprintf("tenant_%d/dir_%d/file_%d", id%2, dir, file)
					_ = cache.LookUp(key)
				}
			}
		}(rIdx)
	}

	// Erasers
	for e := 0; e < numErasers; e++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			r := rand.New(rand.NewSource(int64(id + 200)))
			for {
				select {
				case <-stop:
					return
				default:
					time.Sleep(1 * time.Millisecond)
					switch r.Intn(3) {
					case 0:
						cache.EraseEntriesWithGivenPrefix(fmt.Sprintf("tenant_%d/dir_%d/", r.Intn(2), r.Intn(5)))
					case 1:
						cache.EraseEntriesWithGivenPrefix(fmt.Sprintf("tenant_%d/", r.Intn(2)))
					case 2:
						cache.EraseEntriesWithGivenPrefix("") // Erase all
					}
				}
			}
		}(e)
	}

	time.Sleep(duration)
	close(stop)
	wg.Wait()
}

func fnv1a(key string) uint32 {
	var hash uint32 = 2166136261
	for i := 0; i < len(key); i++ {
		hash ^= uint32(key[i])
		hash *= 16777619
	}
	return hash
}

// TestAdversarial_UpdateSizeExceedingShardCapacity tests UpdateSize when sizeDelta forces eviction across shard boundaries.
func TestAdversarial_UpdateSizeExceedingShardCapacity(t *testing.T) {
	// Custom sharded cache with 4 shards, total maxSize = 100
	cache := lru.NewShardedRadixCacheWithCustomShards(100, 4)
	defer cache.Close()

	// Find keys that hash to different shards
	var keysShard0 []string
	var keysShard1 []string

	for i := 0; i < 1000; i++ {
		k := fmt.Sprintf("key_%d", i)
		h := fnv1a(k) & 3
		if h == 0 && len(keysShard0) < 2 {
			keysShard0 = append(keysShard0, k)
		} else if h == 1 && len(keysShard1) < 2 {
			keysShard1 = append(keysShard1, k)
		}
		if len(keysShard0) >= 2 && len(keysShard1) >= 2 {
			break
		}
	}

	require.Len(t, keysShard0, 2)
	require.Len(t, keysShard1, 2)

	// Populate Shard 0 with 10 bytes and Shard 1 with 80 bytes (Total global size = 90)
	_, err := cache.Insert(keysShard0[0], stressTestData{val: 1, size: 10})
	require.NoError(t, err)

	_, err = cache.Insert(keysShard1[0], stressTestData{val: 2, size: 80})
	require.NoError(t, err)

	// Increase size of keysShard0[0] in Shard 0 by 50 bytes.
	// Total global size would become 90 + 50 = 140 > 100.
	// Shard 0 has only 10 bytes to evict (itself). Shard 1 has 80 bytes.
	err = cache.UpdateSize(keysShard0[0], 50)
	require.NoError(t, err)

	// Verify global current size invariant
	// If global size exceeds 100, then global eviction in UpdateSize failed!
	var remainingSize uint64
	if v := cache.LookUpWithoutChangingOrder(keysShard0[0]); v != nil {
		remainingSize += v.Size()
	}
	if v := cache.LookUpWithoutChangingOrder(keysShard1[0]); v != nil {
		remainingSize += v.Size()
	}

	t.Logf("Remaining size after UpdateSize: %d (maxSize: 100)", remainingSize)
	assert.LessOrEqual(t, remainingSize, uint64(100), "UpdateSize failed to enforce global capacity limit across shards")
}

// TestAdversarial_InsertStaleSizeDeltaUnderConcurrentModification tests if concurrent Inserts causing global eviction preserve global size invariants.
func TestAdversarial_InsertStaleSizeDeltaUnderConcurrentModification(t *testing.T) {
	const totalMaxSize = 200
	cache := lru.NewShardedRadixCacheWithCustomShards(totalMaxSize, 4)
	defer cache.Close()

	var wg sync.WaitGroup
	workers := 16
	iterations := 500
	var panicCount atomic.Int64

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panicCount.Add(1)
					t.Errorf("Panic caught during concurrent insert: %v", r)
				}
			}()

			r := rand.New(rand.NewSource(int64(id)))
			for i := 0; i < iterations; i++ {
				key := fmt.Sprintf("shared_key_%d", r.Intn(10))
				size := uint64(r.Intn(80) + 10)

				switch r.Intn(4) {
				case 0, 1:
					_, _ = cache.Insert(key, stressTestData{val: int64(i), size: size})
				case 2:
					_ = cache.Erase(key)
				case 3:
					_ = cache.LookUp(key)
				}
			}
		}(w)
	}

	wg.Wait()
	assert.Equal(t, int64(0), panicCount.Load(), "Zero panics expected")
}

// TestAdversarial_ZeroSizeEntries tests behavior with entries that have Size() == 0.
func TestAdversarial_ZeroSizeEntries(t *testing.T) {
	cache := lru.NewShardedRadixCache(100)
	defer func() {
		if c, ok := cache.(*lru.ShardedRadixCache); ok {
			c.Close()
		}
	}()

	zeroEntries := make([]*stressTestData, 50)
	// Insert zero-size entries
	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("zero_%d", i)
		zeroEntries[i] = &stressTestData{val: int64(i), size: 0}
		evicted, err := cache.Insert(key, zeroEntries[i])
		require.NoError(t, err)
		assert.Empty(t, evicted)
	}

	// Lookup zero-size entries
	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("zero_%d", i)
		val := cache.LookUp(key)
		require.NotNil(t, val)
		assert.Equal(t, uint64(0), val.Size())
	}

	// Update size from 0 to 20
	zeroEntries[0].size = 20
	err := cache.UpdateSize("zero_0", 20)
	require.NoError(t, err)
	assert.Equal(t, uint64(20), cache.LookUp("zero_0").Size())

	// Erase prefix
	cache.EraseEntriesWithGivenPrefix("zero_")
	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("zero_%d", i)
		assert.Nil(t, cache.LookUp(key))
	}
}

// TestAdversarial_EmptyKey tests behavior when key is the empty string "".
func TestAdversarial_EmptyKey(t *testing.T) {
	cache := lru.NewShardedRadixCache(100)
	defer func() {
		if c, ok := cache.(*lru.ShardedRadixCache); ok {
			c.Close()
		}
	}()

	// Insert empty key
	evicted, err := cache.Insert("", stressTestData{val: 42, size: 30})
	require.NoError(t, err)
	assert.Empty(t, evicted)

	// Lookup empty key
	val := cache.LookUp("")
	require.NotNil(t, val)
	assert.Equal(t, int64(42), val.(stressTestData).val)

	// Update empty key without changing order
	err = cache.UpdateWithoutChangingOrder("", stressTestData{val: 99, size: 30})
	require.NoError(t, err)

	val = cache.LookUp("")
	require.NotNil(t, val)
	assert.Equal(t, int64(99), val.(stressTestData).val)

	// Erase empty key
	erased := cache.Erase("")
	require.NotNil(t, erased)
	assert.Nil(t, cache.LookUp(""))
}

// TestAdversarial_BoundaryCapacity tests exact maxSize limits and oversized entries.
func TestAdversarial_BoundaryCapacity(t *testing.T) {
	const maxSize = 100
	cache := lru.NewShardedRadixCache(maxSize)
	defer func() {
		if c, ok := cache.(*lru.ShardedRadixCache); ok {
			c.Close()
		}
	}()

	// 1. Entry size > totalMaxSize should fail with ErrInvalidEntrySize
	_, err := cache.Insert("huge", stressTestData{val: 1, size: 101})
	assert.ErrorIs(t, err, lru.ErrInvalidEntrySize)

	// 2. Entry size == totalMaxSize should succeed and evict everything else
	_, err = cache.Insert("elem1", stressTestData{val: 1, size: 50})
	require.NoError(t, err)
	_, err = cache.Insert("elem2", stressTestData{val: 2, size: 50})
	require.NoError(t, err)

	evicted, err := cache.Insert("full", stressTestData{val: 3, size: 100})
	require.NoError(t, err)
	assert.Len(t, evicted, 2)
	assert.Nil(t, cache.LookUp("elem1"))
	assert.Nil(t, cache.LookUp("elem2"))
	assert.NotNil(t, cache.LookUp("full"))
}
