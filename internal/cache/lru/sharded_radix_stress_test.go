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
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stressVal struct {
	val  int64
	size uint64
}

func (s stressVal) Size() uint64 {
	return s.size
}

// TestShardedRadixCache_Bug1_EmptyPrefixUnderflow reproduces Bug 1:
// EraseEntriesWithGivenPrefix("") clears s.currentSize=0 and s.sieveHead=nil BEFORE calling
// processDetachedSubtreesBatchLocked. When old nodes are subsequently processed, subtracting
// their size underflows s.currentSize to ~2^64, causing all subsequent Inserts to be evicted immediately.
func TestShardedRadixCache_Bug1_EmptyPrefixUnderflow(t *testing.T) {
	cache := lru.NewShardedRadixCache(1000000)
	defer cache.Close()

	// Insert entries
	for i := 0; i < 500; i++ {
		key := fmt.Sprintf("key_%d", i)
		_, err := cache.Insert(key, stressVal{val: int64(i), size: 10})
		require.NoError(t, err)
	}

	// Erase all with empty prefix
	cache.EraseEntriesWithGivenPrefix("")

	// Allow background worker to process detached nodes
	time.Sleep(50 * time.Millisecond)

	// Now insert 10 new keys. Due to underflow, local/global eviction will evict them!
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("new_key_%d", i)
		_, err := cache.Insert(key, stressVal{val: int64(i), size: 10})
		require.NoError(t, err)
	}

	// At least the most recently inserted or some keys should be present, but due to underflow, all get evicted.
	var found int
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("new_key_%d", i)
		if cache.LookUp(key) != nil {
			found++
		}
	}
	assert.Equal(t, 10, found, "All 10 newly inserted keys should be found, but Bug 1 (size underflow) evicts them")
}

// TestShardedRadixCache_Bug2_DetachedSubtreeSiblingSkipping reproduces Bug 2:
// When a detached tree has > 64 nodes, processDetachedSubtreesBatchLocked saves curr to detachedQueue[0].
// On the next batch, subRoot is curr, which causes the upward traversal termination check (curr == subRoot)
// to exit prematurely, silently skipping siblings and parent-siblings of curr and leaking memory/size.
func TestShardedRadixCache_Bug2_DetachedSubtreeSiblingSkipping(t *testing.T) {
	cache := lru.NewShardedRadixCache(100000000)
	defer cache.Close()

	// Insert 30,000 keys under "dir/" so that multiple shards receive subtrees with > 64 nodes.
	// Use branching paths to ensure siblings exist at multiple levels.
	const numKeys = 30000
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("dir/branch_%d/sub_%d/file_%d", i%50, i%20, i)
		_, err := cache.Insert(key, stressVal{val: int64(i), size: 10})
		require.NoError(t, err)
	}

	// Erase prefix "dir/"
	cache.EraseEntriesWithGivenPrefix("dir/")

	// Wait enough time for background worker to sweep all batches across all shards
	time.Sleep(500 * time.Millisecond)

	// Check that no keys remain accessible
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("dir/branch_%d/sub_%d/file_%d", i%50, i%20, i)
		assert.Nil(t, cache.LookUp(key), "key %q should be erased", key)
	}

	// Verify size leak from skipped siblings:
	// We insert "baseline" of size 1000, then "huge" of size maxSize-1000.
	// If all 300,000 bytes from "dir/" were properly reclaimed, total size is exactly maxSize (no eviction).
	// Due to Bug 2 (sibling skipping in partial batches), thousands of bytes leaked in currentSize,
	// so inserting "huge" will cause "baseline" (or "huge") to be evicted!
	_, err := cache.Insert("baseline", stressVal{val: 1, size: 1000})
	require.NoError(t, err)

	_, err = cache.Insert("huge", stressVal{val: 2, size: 100000000 - 1000})
	require.NoError(t, err)

	assert.NotNil(t, cache.LookUp("baseline"), "baseline should NOT be evicted if all erased subtree size was reclaimed, but Bug 2 leaked currentSize")
	assert.NotNil(t, cache.LookUp("huge"), "huge should NOT be evicted")
}

// TestShardedRadixCache_Stress_Differential against mapCache (Oracle) without eviction.
func TestShardedRadixCache_Stress_Differential(t *testing.T) {
	const maxSize = 10000000 // Large enough to prevent eviction
	oracle := lru.NewCache(maxSize)
	defer oracle.Close()
	target := lru.NewShardedRadixCache(maxSize)
	defer target.Close()

	rng := rand.New(rand.NewSource(42))
	keys := []string{
		"", "a", "b", "c",
		"foo", "foo/bar", "foo/baz", "foo/bar/1", "foo/bar/2",
		"bar", "bar/1", "bar/2", "baz/1/2/3",
	}

	var history []string
	for step := 0; step < 5000; step++ {
		op := rng.Intn(6)
		key := keys[rng.Intn(len(keys))]

		switch op {
		case 0: // Insert
			val := stressVal{val: int64(step), size: uint64(rng.Intn(50) + 1)}
			history = append(history, fmt.Sprintf("step %d: Insert(%q, %v)", step, key, val))
			_, err1 := oracle.Insert(key, val)
			_, err2 := target.Insert(key, val)
			require.Equal(t, err1, err2, "Insert error mismatch at step %d for key %q\nHistory:\n%s", step, key, strings.Join(history, "\n"))

		case 1: // LookUp
			history = append(history, fmt.Sprintf("step %d: LookUp(%q)", step, key))
			v1 := oracle.LookUp(key)
			v2 := target.LookUp(key)
			require.Equal(t, v1, v2, "LookUp mismatch at step %d for key %q\nHistory:\n%s", step, key, strings.Join(history, "\n"))

		case 2: // Erase
			history = append(history, fmt.Sprintf("step %d: Erase(%q)", step, key))
			v1 := oracle.Erase(key)
			v2 := target.Erase(key)
			require.Equal(t, v1, v2, "Erase mismatch at step %d for key %q\nHistory:\n%s", step, key, strings.Join(history, "\n"))

		case 3: // EraseEntriesWithGivenPrefix
			prefix := keys[rng.Intn(len(keys))]
			history = append(history, fmt.Sprintf("step %d: EraseEntriesWithGivenPrefix(%q)", step, prefix))
			oracle.EraseEntriesWithGivenPrefix(prefix)
			target.EraseEntriesWithGivenPrefix(prefix)
			// Small sleep if empty prefix was used, to allow background sweep
			time.Sleep(2 * time.Millisecond)

		case 4: // LookUpWithoutChangingOrder
			history = append(history, fmt.Sprintf("step %d: LookUpWithoutChangingOrder(%q)", step, key))
			v1 := oracle.LookUpWithoutChangingOrder(key)
			v2 := target.LookUpWithoutChangingOrder(key)
			require.Equal(t, v1, v2, "LookUpWithoutChangingOrder mismatch at step %d for key %q\nHistory:\n%s", step, key, strings.Join(history, "\n"))

		case 5: // UpdateWithoutChangingOrder
			val := stressVal{val: int64(step + 100000), size: 10} // fixed size to avoid size mismatch
			history = append(history, fmt.Sprintf("step %d: UpdateWithoutChangingOrder(%q, %v)", step, key, val))
			// Ensure it has size 10 in both if present
			v1 := oracle.LookUpWithoutChangingOrder(key)
			if v1 != nil && v1.(stressVal).size == 10 {
				err1 := oracle.UpdateWithoutChangingOrder(key, val)
				err2 := target.UpdateWithoutChangingOrder(key, val)
				require.Equal(t, err1, err2, "UpdateWithoutChangingOrder error mismatch at step %d for key %q\nHistory:\n%s", step, key, strings.Join(history, "\n"))
			}
		}
	}

	// Final verification across all keys
	for _, key := range keys {
		require.Equal(t, oracle.LookUp(key), target.LookUp(key), "Final state mismatch for key %q", key)
	}
}

// TestShardedRadixCache_Stress_Differential_NoBug1 tests differential fuzzing avoiding empty prefix erasure
// to see if any other routing/eviction/concurrency invariants are broken.
func TestShardedRadixCache_Stress_Differential_NoBug1(t *testing.T) {
	const maxSize = 10000000 // Large enough to prevent eviction
	oracle := lru.NewCache(maxSize)
	defer oracle.Close()
	target := lru.NewShardedRadixCache(maxSize)
	defer target.Close()

	rng := rand.New(rand.NewSource(999))
	keys := []string{
		"a", "b", "c",
		"foo", "foo/bar", "foo/baz", "foo/bar/1", "foo/bar/2",
		"bar", "bar/1", "bar/2", "baz/1/2/3",
	}

	var history []string
	for step := 0; step < 5000; step++ {
		op := rng.Intn(6)
		key := keys[rng.Intn(len(keys))]

		switch op {
		case 0: // Insert
			val := stressVal{val: int64(step), size: uint64(rng.Intn(50) + 1)}
			history = append(history, fmt.Sprintf("step %d: Insert(%q, %v)", step, key, val))
			_, err1 := oracle.Insert(key, val)
			_, err2 := target.Insert(key, val)
			require.Equal(t, err1, err2, "Insert error mismatch at step %d for key %q\nHistory:\n%s", step, key, strings.Join(history, "\n"))

		case 1: // LookUp
			history = append(history, fmt.Sprintf("step %d: LookUp(%q)", step, key))
			v1 := oracle.LookUp(key)
			v2 := target.LookUp(key)
			require.Equal(t, v1, v2, "LookUp mismatch at step %d for key %q\nHistory:\n%s", step, key, strings.Join(history, "\n"))

		case 2: // Erase
			history = append(history, fmt.Sprintf("step %d: Erase(%q)", step, key))
			v1 := oracle.Erase(key)
			v2 := target.Erase(key)
			require.Equal(t, v1, v2, "Erase mismatch at step %d for key %q\nHistory:\n%s", step, key, strings.Join(history, "\n"))

		case 3: // EraseEntriesWithGivenPrefix (non-empty only)
			prefix := keys[rng.Intn(len(keys))]
			history = append(history, fmt.Sprintf("step %d: EraseEntriesWithGivenPrefix(%q)", step, prefix))
			oracle.EraseEntriesWithGivenPrefix(prefix)
			target.EraseEntriesWithGivenPrefix(prefix)
			time.Sleep(1 * time.Millisecond)

		case 4: // LookUpWithoutChangingOrder
			history = append(history, fmt.Sprintf("step %d: LookUpWithoutChangingOrder(%q)", step, key))
			v1 := oracle.LookUpWithoutChangingOrder(key)
			v2 := target.LookUpWithoutChangingOrder(key)
			require.Equal(t, v1, v2, "LookUpWithoutChangingOrder mismatch at step %d for key %q\nHistory:\n%s", step, key, strings.Join(history, "\n"))

		case 5: // UpdateWithoutChangingOrder
			val := stressVal{val: int64(step + 100000), size: 10}
			history = append(history, fmt.Sprintf("step %d: UpdateWithoutChangingOrder(%q, %v)", step, key, val))
			v1 := oracle.LookUpWithoutChangingOrder(key)
			if v1 != nil && v1.(stressVal).size == 10 {
				err1 := oracle.UpdateWithoutChangingOrder(key, val)
				err2 := target.UpdateWithoutChangingOrder(key, val)
				require.Equal(t, err1, err2, "UpdateWithoutChangingOrder error mismatch at step %d for key %q\nHistory:\n%s", step, key, strings.Join(history, "\n"))
			}
		}
	}

	for _, key := range keys {
		require.Equal(t, oracle.LookUp(key), target.LookUp(key), "Final state mismatch for key %q", key)
	}
}

// TestShardedRadixCache_Stress_Concurrency tests heavy concurrent operations with -race enabled.
func TestShardedRadixCache_Stress_Concurrency(t *testing.T) {
	cache := lru.NewShardedRadixCache(500000)
	defer cache.Close()

	var wg sync.WaitGroup
	const numRoutines = 20
	const opsPerRoutine = 500

	keys := []string{
		"usr/local/bin/file1", "usr/local/bin/file2",
		"usr/local/lib/lib1", "usr/local/lib/lib2",
		"home/user/doc1", "home/user/doc2",
		"var/log/syslog", "var/log/auth",
		"etc/passwd", "etc/hosts", "",
	}

	var stop int32

	// Readers
	for i := 0; i < numRoutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(id)))
			for atomic.LoadInt32(&stop) == 0 {
				key := keys[rng.Intn(len(keys))]
				_ = cache.LookUp(key)
				_ = cache.LookUpWithoutChangingOrder(key)
				time.Sleep(time.Microsecond)
			}
		}(i)
	}

	// Writers (Insert & Update)
	for i := 0; i < numRoutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(id + 100)))
			for j := 0; j < opsPerRoutine; j++ {
				key := keys[rng.Intn(len(keys))]
				val := stressVal{val: int64(j), size: uint64(rng.Intn(50) + 1)}
				_, _ = cache.Insert(key, val)
				_ = cache.UpdateWithoutChangingOrder(key, val)
				_ = cache.UpdateSize(key, uint64(rng.Intn(10)))
			}
		}(i)
	}

	// Erasers (Erase & Prefix Erase)
	for i := 0; i < numRoutines/2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(id + 200)))
			for j := 0; j < opsPerRoutine/5; j++ {
				key := keys[rng.Intn(len(keys))]
				_ = cache.Erase(key)
				if rng.Intn(10) == 0 {
					prefixes := []string{"usr/", "home/", "var/", "etc/", ""}
					cache.EraseEntriesWithGivenPrefix(prefixes[rng.Intn(len(prefixes))])
				}
				time.Sleep(2 * time.Millisecond)
			}
		}(i)
	}

	// Pruner
	wg.Add(1)
	go func() {
		defer wg.Done()
		for atomic.LoadInt32(&stop) == 0 {
			if sc, ok := cache.(*lru.ShardedRadixCache); ok {
				sc.PruneAllEmptyLeaves()
			}
			time.Sleep(5 * time.Millisecond)
		}
	}()

	// Wait for writers and erasers to finish
	time.Sleep(1 * time.Second)
	atomic.StoreInt32(&stop, 1)
	wg.Wait()
}
