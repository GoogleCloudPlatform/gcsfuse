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

package lru

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathSegmentInterner_BoundedCapacity100kUniquePaths(t *testing.T) {
	interner := NewPathSegmentInterner()

	var msBefore, msAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&msBefore)

	const count = 150000
	for i := 0; i < count; i++ {
		s := fmt.Sprintf("tenant_%d/dataset_%d/sub/file_%d.png", i%100, i%500, i)
		res := interner.Intern(s)
		assert.Equal(t, s, res)
	}

	runtime.ReadMemStats(&msAfter)

	totalEntries := 0
	maxShardLen := 0
	for i := 0; i < 64; i++ {
		interner.shards[i].mu.RLock()
		l := len(interner.shards[i].m)
		interner.shards[i].mu.RUnlock()
		totalEntries += l
		if l > maxShardLen {
			maxShardLen = l
		}
	}

	heapDeltaMB := float64(msAfter.HeapAlloc-msBefore.HeapAlloc) / (1024 * 1024)
	t.Logf("Interner total entries across 64 shards after %d unique paths: %d (max shard len: %d)", count, totalEntries, maxShardLen)
	t.Logf("HeapAlloc delta after interning %d unique strings: %.2f MB", count, heapDeltaMB)

	assert.LessOrEqual(t, maxShardLen, maxCapacityPerShard, "No individual shard should exceed maxCapacityPerShard (8192)")
	assert.LessOrEqual(t, totalEntries, 64*maxCapacityPerShard, "Total entries across all 64 shards must stay strictly bounded")
}

func TestPathSegmentInterner_ShardResetThreshold1MillionPaths(t *testing.T) {
	interner := NewPathSegmentInterner()

	const count = 1000000 // 1 Million unique strings
	for i := 0; i < count; i++ {
		s := fmt.Sprintf("high_cardinality_path/segment_%d/file_%d.dat", i%50, i)
		res := interner.Intern(s)
		assert.Equal(t, s, res)
	}

	totalEntries := 0
	maxShardLen := 0
	for i := 0; i < 64; i++ {
		interner.shards[i].mu.RLock()
		l := len(interner.shards[i].m)
		interner.shards[i].mu.RUnlock()
		totalEntries += l
		if l > maxShardLen {
			maxShardLen = l
		}
	}

	t.Logf("Interner total entries after %d unique paths: %d (max shard len: %d)", count, totalEntries, maxShardLen)
	assert.LessOrEqual(t, maxShardLen, maxCapacityPerShard, "No individual shard should exceed maxCapacityPerShard (8192)")
	assert.LessOrEqual(t, totalEntries, 64*maxCapacityPerShard, "Total stored entries must remain bounded strictly below 524,288")
}
