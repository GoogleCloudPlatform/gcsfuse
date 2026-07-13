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
	"sync"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type shardedTestData struct {
	Value    int64
	DataSize uint64
}

func (d shardedTestData) Size() uint64 {
	return d.DataSize
}

func TestShardedRadixCache_BasicOperations(t *testing.T) {
	cache := lru.NewShardedRadixCache(100)
	defer func() {
		if c, ok := cache.(*lru.ShardedRadixCache); ok {
			c.Close()
		}
	}()

	// LookUp empty
	assert.Nil(t, cache.LookUp("key1"))

	// Insert
	evicted, err := cache.Insert("key1", shardedTestData{Value: 10, DataSize: 20})
	require.NoError(t, err)
	assert.Empty(t, evicted)

	// LookUp hit
	val := cache.LookUp("key1")
	require.NotNil(t, val)
	assert.Equal(t, int64(10), val.(shardedTestData).Value)

	// Update size
	err = cache.UpdateSize("key1", 10)
	require.NoError(t, err)

	// Erase
	erasedVal := cache.Erase("key1")
	require.NotNil(t, erasedVal)
	assert.Equal(t, int64(10), erasedVal.(shardedTestData).Value)
	assert.Nil(t, cache.LookUp("key1"))
}

func TestShardedRadixCache_PrefixErasure(t *testing.T) {
	cache := lru.NewShardedRadixCache(1000)
	defer func() {
		if c, ok := cache.(*lru.ShardedRadixCache); ok {
			c.Close()
		}
	}()

	for i := 0; i < 50; i++ {
		_, err := cache.Insert(fmt.Sprintf("dir1/file_%d.txt", i), shardedTestData{Value: int64(i), DataSize: 10})
		require.NoError(t, err)
		_, err = cache.Insert(fmt.Sprintf("dir2/file_%d.txt", i), shardedTestData{Value: int64(i), DataSize: 10})
		require.NoError(t, err)
	}

	cache.EraseEntriesWithGivenPrefix("dir1/")

	for i := 0; i < 50; i++ {
		assert.Nil(t, cache.LookUp(fmt.Sprintf("dir1/file_%d.txt", i)))
		assert.NotNil(t, cache.LookUp(fmt.Sprintf("dir2/file_%d.txt", i)))
	}
}

func TestShardedRadixCache_ConcurrentReadWriteRace(t *testing.T) {
	cache := lru.NewShardedRadixCache(500)
	defer func() {
		if c, ok := cache.(*lru.ShardedRadixCache); ok {
			c.Close()
		}
	}()

	var wg sync.WaitGroup
	workers := 16
	ops := 200

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < ops; i++ {
				key := fmt.Sprintf("dir_%d/file_%d", i%5, (workerID*10)+i)
				switch i % 4 {
				case 0:
					_, _ = cache.Insert(key, shardedTestData{Value: int64(i), DataSize: 10})
				case 1:
					_ = cache.LookUp(key)
				case 2:
					_ = cache.Erase(key)
				case 3:
					if i%20 == 0 {
						cache.EraseEntriesWithGivenPrefix(fmt.Sprintf("dir_%d/", i%5))
					}
				}
			}
		}(w)
	}

	wg.Wait()
}
