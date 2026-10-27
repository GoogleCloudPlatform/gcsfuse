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

package metadata_test

import (
	"fmt"
	"math"
	"math/rand"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sizeRecorder keeps the metadata_cache/size series it is given and ignores
// every other metric.
type sizeRecorder struct {
	metrics.MetricHandle
	positive, negative, other int64
}

func newSizeRecorder() *sizeRecorder {
	return &sizeRecorder{MetricHandle: metrics.NewNoopMetrics()}
}

func (r *sizeRecorder) MetadataCacheSize(inc int64, entryStatus metrics.EntryStatus) {
	switch entryStatus {
	case metrics.EntryStatusPositiveAttr:
		r.positive += inc
	case metrics.EntryStatusNegativeAttr:
		r.negative += inc
	default:
		r.other += inc
	}
}

// sizeOf returns the size the shared cache accounts for the entries of keys,
// which must all be cached.
func sizeOf(t *testing.T, shared *lru.Cache, keys ...string) (size int64) {
	t.Helper()
	for _, k := range keys {
		v := shared.LookUpWithoutChangingOrder(k)
		require.NotNil(t, v, "no entry for %q", k)
		size += int64(v.Size())
	}
	return size
}

// requireSizes checks that the series are exactly the sizes of the given
// positive and negative entries of the shared cache.
func (r *sizeRecorder) requireSizes(t *testing.T, shared *lru.Cache, positive, negative []string) {
	t.Helper()
	require.Equal(t, sizeOf(t, shared, positive...), r.positive, "entry_status=positive")
	require.Equal(t, sizeOf(t, shared, negative...), r.negative, "entry_status=negative")
	require.Zero(t, r.other, "unexpected entry_status")
}

func TestNewSharedStatCache_RecordsSize(t *testing.T) {
	now := time.Now()
	later := now.Add(time.Hour)
	past := now.Add(-time.Second)
	object := func(name string, generation int64) *gcs.MinObject {
		return &gcs.MinObject{Name: name, Generation: generation, MetaGeneration: 1, Size: 10}
	}
	bigObject := func(name string, generation int64) *gcs.MinObject {
		o := object(name, generation)
		o.Metadata = map[string]string{"key": strings.Repeat("v", 100)}
		return o
	}
	testCases := []struct {
		name string
		ops  func(sc metadata.StatCache)
		// Keys cached afterwards, as positive and as negative entries.
		positive, negative []string
	}{
		{
			name:     "object is positive",
			ops:      func(sc metadata.StatCache) { sc.Insert(object("a", 1), later) },
			positive: []string{"a"},
		},
		{
			name:     "implicit directory is positive",
			ops:      func(sc metadata.StatCache) { sc.InsertImplicitDir("d/", later) },
			positive: []string{"d/"},
		},
		{
			name:     "folder is positive",
			ops:      func(sc metadata.StatCache) { sc.InsertFolder(&gcs.Folder{Name: "f/"}, later) },
			positive: []string{"f/"},
		},
		{
			name:     "negative entry",
			ops:      func(sc metadata.StatCache) { sc.AddNegativeEntry("a", later) },
			negative: []string{"a"},
		},
		{
			name:     "negative folder entry",
			ops:      func(sc metadata.StatCache) { sc.AddNegativeEntryForFolder("f/", later) },
			negative: []string{"f/"},
		},
		{
			name: "negative entry replaced by object",
			ops: func(sc metadata.StatCache) {
				sc.AddNegativeEntry("a", later)
				sc.Insert(object("a", 1), later)
			},
			positive: []string{"a"},
		},
		{
			name: "object replaced by negative entry",
			ops: func(sc metadata.StatCache) {
				sc.Insert(object("a", 1), later)
				sc.AddNegativeEntry("a", later)
			},
			negative: []string{"a"},
		},
		{
			name: "object replaced by newer generation",
			ops: func(sc metadata.StatCache) {
				sc.Insert(object("a", 1), later)
				sc.Insert(bigObject("a", 2), later)
			},
			positive: []string{"a"},
		},
		{
			name: "older generation is not cached",
			ops: func(sc metadata.StatCache) {
				sc.Insert(object("a", 2), later)
				sc.Insert(bigObject("a", 1), later)
			},
			positive: []string{"a"},
		},
		{
			name: "implicit directory does not replace explicit one",
			ops: func(sc metadata.StatCache) {
				sc.Insert(bigObject("d/", 1), later)
				sc.InsertImplicitDir("d/", later)
			},
			positive: []string{"d/"},
		},
		{
			name: "erase",
			ops: func(sc metadata.StatCache) {
				sc.Insert(object("a", 1), later)
				sc.AddNegativeEntry("b", later)
				sc.Erase("a")
				sc.Erase("b")
			},
		},
		{
			name: "look-ups erase expired entries",
			ops: func(sc metadata.StatCache) {
				sc.Insert(object("a", 1), past)
				sc.AddNegativeEntry("b", past)
				sc.InsertFolder(&gcs.Folder{Name: "f/"}, past)
				sc.LookUp("a", now)
				sc.LookUp("b", now)
				sc.LookUpFolder("f/", now)
			},
		},
		{
			name: "erase with prefix",
			ops: func(sc metadata.StatCache) {
				sc.Insert(object("dir/a", 1), later)
				sc.AddNegativeEntry("dir/b", later)
				sc.InsertImplicitDir("dir/c/", later)
				sc.Insert(object("other", 1), later)
				sc.EraseEntriesWithGivenPrefix("dir/")
			},
			positive: []string{"other"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := newSizeRecorder()
			shared := metadata.NewSharedStatCache(1<<20, r)

			tc.ops(metadata.NewStatCacheBucketView(shared, ""))

			r.requireSizes(t, shared, tc.positive, tc.negative)
		})
	}
}

// Whatever the mix of buckets, entry kinds, replacements, erasures and
// evictions, metadata_cache/size is exactly the size of the cached entries.
func TestNewSharedStatCache_SizeMatchesCachedEntries(t *testing.T) {
	r := newSizeRecorder()
	// Room for a few dozen entries, so that inserts keep evicting, possibly
	// the entries of the other bucket.
	shared := metadata.NewSharedStatCache(30*cfg.AverageSizeOfPositiveStatCacheEntry, r)
	bucketNames := []string{"fruits", "spices"}
	views := map[string]metadata.StatCache{}
	for _, b := range bucketNames {
		views[b] = metadata.NewStatCacheBucketView(shared, b)
	}
	var names []string
	for i := range 60 {
		names = append(names, fmt.Sprintf("dir%d/object%d", i%4, i))
	}
	// Status of the entry last written for each shared-cache key.
	written := map[string]metrics.EntryStatus{}
	later := time.Now().Add(time.Hour)
	rnd := rand.New(rand.NewSource(1))
	generation := int64(0)

	for range 5000 {
		b := bucketNames[rnd.Intn(len(bucketNames))]
		sc := views[b]
		name := names[rnd.Intn(len(names))]
		dir := name[:strings.Index(name, "/")+1]
		switch rnd.Intn(6) {
		case 0, 1:
			generation++
			sc.Insert(&gcs.MinObject{Name: name, Generation: generation}, later)
			written[b+"/"+name] = metrics.EntryStatusPositiveAttr
		case 2:
			sc.AddNegativeEntry(name, later)
			written[b+"/"+name] = metrics.EntryStatusNegativeAttr
		case 3:
			sc.InsertImplicitDir(dir, later)
			written[b+"/"+dir] = metrics.EntryStatusPositiveAttr
		case 4:
			sc.Erase(name)
			delete(written, b+"/"+name)
		case 5:
			sc.EraseEntriesWithGivenPrefix(dir)
			for k := range written {
				if strings.HasPrefix(k, b+"/"+dir) {
					delete(written, k)
				}
			}
		}

		var positive, negative []string
		for k, status := range written {
			if shared.LookUpWithoutChangingOrder(k) == nil { // Evicted.
				delete(written, k)
				continue
			}
			if status == metrics.EntryStatusPositiveAttr {
				positive = append(positive, k)
			} else {
				negative = append(negative, k)
			}
		}
		r.requireSizes(t, shared, positive, negative)
	}
}

func TestNewSharedStatCache_NilMetricHandle(t *testing.T) {
	shared := metadata.NewSharedStatCache(1<<20, nil)
	sc := metadata.NewStatCacheBucketView(shared, "")

	sc.Insert(&gcs.MinObject{Name: "a"}, time.Now().Add(time.Hour))

	hit, m := sc.LookUp("a", time.Now())
	assert.True(t, hit)
	assert.Equal(t, "a", m.Name)
}

func TestNewSharedStatCache_MemoryPressureEviction(t *testing.T) {
	prevLimit := debug.SetMemoryLimit(math.MaxInt64)
	defer debug.SetMemoryLimit(prevLimit)

	r := newSizeRecorder()
	const numEntries = 10
	// Create a cache whose capacity holds all 10 positive entries comfortably.
	maxSize := uint64(numEntries) * cfg.AverageSizeOfPositiveStatCacheEntry
	shared := metadata.NewSharedStatCache(maxSize, r)
	sc := metadata.NewStatCacheBucketView(shared, "")
	later := time.Now().Add(time.Hour)

	// The first write samples pressure at math.MaxInt64 (0.0 pressure).
	var allKeys []string
	for i := range numEntries {
		name := fmt.Sprintf("obj-%02d", i)
		allKeys = append(allKeys, name)
		sc.Insert(&gcs.MinObject{Name: name, Generation: 1}, later)
	}
	r.requireSizes(t, shared, allKeys, nil)

	// Set a 1-byte GOMEMLIMIT so the runtime pressure probe reports critical pressure (>= 0.90),
	// and advance past the 256-write amortized sampling window so foreground writes re-sample pressure.
	debug.SetMemoryLimit(1)
	for i := range 256 {
		name := fmt.Sprintf("obj-%02d", i%numEntries)
		sc.Insert(&gcs.MinObject{Name: name, Generation: int64(2 + i)}, later)
	}

	var remaining []string
	for _, k := range allKeys {
		if shared.LookUpWithoutChangingOrder(k) != nil {
			remaining = append(remaining, k)
		}
	}
	// Critical pressure sheds LRU entries down to <= 50% of maxSize.
	assert.Less(t, len(remaining), len(allKeys))
	assert.LessOrEqual(t, r.positive, int64(maxSize/2))
	r.requireSizes(t, shared, remaining, nil)
}
