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

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/locker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sizeChange is one notification received by a SizeObserver: the
// testData.Value of the value notified, and the delta.
type sizeChange struct {
	value int64
	delta int64
}

// sizeLog is a SizeObserver that records what it is notified of.
type sizeLog struct {
	changes []sizeChange
}

func (l *sizeLog) observe(v lru.ValueType, delta int64) {
	l.changes = append(l.changes, sizeChange{v.(testData).Value, delta})
}

func newObservedCache(t *testing.T) (*lru.Cache, *sizeLog) {
	t.Helper()
	log := &sizeLog{}
	cache := lru.NewCacheWithOptions(MaxSize, locker.Options{EnableInvariantsCheck: true}, lru.WithSizeObserver(log.observe))
	return cache, log
}

func TestSizeObserver(t *testing.T) {
	testCases := []struct {
		name string
		ops  func(t *testing.T, c *lru.Cache)
		want []sizeChange
	}{
		{
			name: "insert",
			ops: func(t *testing.T, c *lru.Cache) {
				insertAndAssert(t, c, "a", testData{Value: 1, DataSize: 10}, []int64{}, nil)
			},
			want: []sizeChange{{1, 10}},
		},
		{
			name: "replace notifies the replaced value with its accounted size",
			ops: func(t *testing.T, c *lru.Cache) {
				insertAndAssert(t, c, "a", testData{Value: 1, DataSize: 10}, []int64{}, nil)
				insertAndAssert(t, c, "a", testData{Value: 2, DataSize: 4}, []int64{}, nil)
			},
			want: []sizeChange{{1, 10}, {1, -10}, {2, 4}},
		},
		{
			name: "erase",
			ops: func(t *testing.T, c *lru.Cache) {
				insertAndAssert(t, c, "a", testData{Value: 1, DataSize: 10}, []int64{}, nil)
				c.Erase("a")
			},
			want: []sizeChange{{1, 10}, {1, -10}},
		},
		{
			name: "erase of an absent key notifies nothing",
			ops: func(t *testing.T, c *lru.Cache) {
				c.Erase("a")
			},
		},
		{
			name: "eviction",
			ops: func(t *testing.T, c *lru.Cache) {
				insertAndAssert(t, c, "a", testData{Value: 1, DataSize: 20}, []int64{}, nil)
				insertAndAssert(t, c, "b", testData{Value: 2, DataSize: 20}, []int64{}, nil)
				insertAndAssert(t, c, "c", testData{Value: 3, DataSize: 20}, []int64{1}, nil)
			},
			want: []sizeChange{{1, 20}, {2, 20}, {3, 20}, {1, -20}},
		},
		{
			name: "erase with prefix",
			ops: func(t *testing.T, c *lru.Cache) {
				insertAndAssert(t, c, "p/a", testData{Value: 1, DataSize: 5}, []int64{}, nil)
				insertAndAssert(t, c, "p/b", testData{Value: 2, DataSize: 6}, []int64{}, nil)
				insertAndAssert(t, c, "q", testData{Value: 3, DataSize: 7}, []int64{}, nil)
				c.EraseEntriesWithGivenPrefix("p/")
			},
			want: []sizeChange{{1, 5}, {2, 6}, {3, 7}, {1, -5}, {2, -6}},
		},
		{
			name: "update without changing order moves the size to the new value",
			ops: func(t *testing.T, c *lru.Cache) {
				insertAndAssert(t, c, "a", testData{Value: 1, DataSize: 10}, []int64{}, nil)
				require.NoError(t, c.UpdateWithoutChangingOrder("a", testData{Value: 2, DataSize: 10}))
			},
			want: []sizeChange{{1, 10}, {1, -10}, {2, 10}},
		},
		{
			name: "update size",
			ops: func(t *testing.T, c *lru.Cache) {
				insertAndAssert(t, c, "a", testData{Value: 1, DataSize: 10}, []int64{}, nil)
				require.NoError(t, c.UpdateSize("a", 5))
				c.Erase("a")
			},
			want: []sizeChange{{1, 10}, {1, 5}, {1, -15}},
		},
		{
			name: "rejected changes notify nothing",
			ops: func(t *testing.T, c *lru.Cache) {
				insertAndAssert(t, c, "a", nil, []int64{}, lru.ErrInvalidEntry)
				insertAndAssert(t, c, "a", testData{Value: 1, DataSize: MaxSize + 1}, []int64{}, lru.ErrInvalidEntrySize)
				require.ErrorIs(t, c.UpdateWithoutChangingOrder("a", testData{Value: 1, DataSize: 1}), lru.ErrEntryNotExist)
				require.ErrorIs(t, c.UpdateSize("a", 1), lru.ErrEntryNotExist)
			},
		},
		{
			name: "look-ups notify nothing",
			ops: func(t *testing.T, c *lru.Cache) {
				insertAndAssert(t, c, "a", testData{Value: 1, DataSize: 10}, []int64{}, nil)
				c.LookUp("a")
				c.LookUpWithoutChangingOrder("a")
			},
			want: []sizeChange{{1, 10}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cache, log := newObservedCache(t)

			tc.ops(t, cache)

			// EraseEntriesWithGivenPrefix erases in map order, so compare as a multiset.
			assert.ElementsMatch(t, tc.want, log.changes)
		})
	}
}

// presentSize returns the size of the values cached for keys.
func presentSize(c *lru.Cache, keys []string) (size int64) {
	for _, k := range keys {
		if v := c.LookUpWithoutChangingOrder(k); v != nil {
			size += int64(v.Size())
		}
	}
	return size
}

// The deltas notified always add up to the size of the cached values, which
// the cache's invariant checks keep equal to its accounted current size.
func TestSizeObserver_DeltasAddUpToCachedSize(t *testing.T) {
	var total int64
	cache := lru.NewCacheWithOptions(MaxSize, locker.Options{EnableInvariantsCheck: true},
		lru.WithSizeObserver(func(_ lru.ValueType, delta int64) { total += delta }))
	keys := make([]string, 10)
	for i := range keys {
		keys[i] = fmt.Sprintf("d%d/key-%d", i%3, i)
	}
	r := rand.New(rand.NewSource(1))

	for range 10000 {
		k := keys[r.Intn(len(keys))]
		switch r.Intn(4) {
		case 0, 1:
			_, err := cache.Insert(k, testData{Value: r.Int63(), DataSize: uint64(1 + r.Intn(15))})
			require.NoError(t, err)
		case 2:
			cache.Erase(k)
		case 3:
			// Erases the whole "directory" of k.
			cache.EraseEntriesWithGivenPrefix(k[:3])
		}

		require.Equal(t, presentSize(cache, keys), total)
	}
}

// Concurrent changes are notified exactly once each.
func TestSizeObserver_Concurrent(t *testing.T) {
	var total atomic.Int64
	cache := lru.NewCache(1000, lru.WithSizeObserver(func(_ lru.ValueType, delta int64) { total.Add(delta) }))
	keys := make([]string, 100)
	for i := range keys {
		keys[i] = fmt.Sprintf("key-%d", i)
	}

	var wg sync.WaitGroup
	for g := range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := rand.New(rand.NewSource(int64(g)))
			for range 2000 {
				k := keys[r.Intn(len(keys))]
				if r.Intn(3) == 0 {
					cache.Erase(k)
				} else {
					_, _ = cache.Insert(k, testData{DataSize: uint64(1 + r.Intn(30))})
				}
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, presentSize(cache, keys), total.Load())
}
