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
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, meither express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metadata_test

import (
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 1. Boundary Condition: SizeOfTypeCacheEntry helper and TypeCache zero-capacities
func TestAdv_TypeCacheBoundaryCoverage(t *testing.T) {
	// Directly test SizeOfTypeCacheEntry (addressing 0% coverage gap)
	sz := metadata.SizeOfTypeCacheEntry("test/path/file.txt")
	assert.Greater(t, sz, uint64(0), "SizeOfTypeCacheEntry must return non-zero RSS calculation")

	now := time.Now()
	ttl := 5 * time.Minute

	// Disabled TypeCache (maxSizeMB = 0 or ttl = 0)
	disabledCache1 := metadata.NewTypeCache(0, ttl)
	disabledCache1.Insert(now, "file1", metadata.RegularFileType)
	assert.Equal(t, metadata.UnknownType, disabledCache1.Get(now, "file1"), "TypeCache with 0 maxSize must store nothing")

	disabledCache2 := metadata.NewTypeCache(10, 0)
	disabledCache2.Insert(now, "file2", metadata.RegularFileType)
	assert.Equal(t, metadata.UnknownType, disabledCache2.Get(now, "file2"), "TypeCache with 0 TTL must store nothing")

	// Enabled TypeCache
	tc := metadata.NewTypeCache(1, ttl)
	tc.Insert(now, "file3", metadata.RegularFileType)
	assert.Equal(t, metadata.RegularFileType, tc.Get(now, "file3"))

	// Erase functionality
	tc.Erase("file3")
	assert.Equal(t, metadata.UnknownType, tc.Get(now, "file3"))

	// Expired entry
	tc.Insert(now, "file4", metadata.ExplicitDirType)
	assert.Equal(t, metadata.UnknownType, tc.Get(now.Add(ttl+time.Second), "file4"), "Expired entry must return UnknownType")
}

// 2. Boundary Condition: StatCache Bucket Views (Dynamic vs Static Mounts) and Prefix Invalidation
func TestAdv_StatCacheBucketViewAndPrefix(t *testing.T) {
	sharedLRU := lru.NewRadixCache(100000)

	// View for bucket A and view for bucket B sharing same cache
	viewA := metadata.NewStatCacheBucketView(sharedLRU, "bucket-a")
	viewB := metadata.NewStatCacheBucketView(sharedLRU, "bucket-b")
	viewStatic := metadata.NewStatCacheBucketView(sharedLRU, "") // Static mount (no bucket prefix)

	now := time.Now()
	future := now.Add(1 * time.Hour)

	// Insert identical object names into bucket A and bucket B
	viewA.Insert(&gcs.MinObject{Name: "dir/file.txt", Generation: 1}, future)
	viewB.Insert(&gcs.MinObject{Name: "dir/file.txt", Generation: 2}, future)
	viewStatic.Insert(&gcs.MinObject{Name: "dir/file.txt", Generation: 3}, future)

	// LookUp verification: Bucket views isolate entries
	hitA, minObjA := viewA.LookUp("dir/file.txt", now)
	require.True(t, hitA)
	assert.Equal(t, int64(1), minObjA.Generation)

	hitB, minObjB := viewB.LookUp("dir/file.txt", now)
	require.True(t, hitB)
	assert.Equal(t, int64(2), minObjB.Generation)

	hitStatic, minObjStatic := viewStatic.LookUp("dir/file.txt", now)
	require.True(t, hitStatic)
	assert.Equal(t, int64(3), minObjStatic.Generation)

	// EraseEntriesWithGivenPrefix on bucket A should NOT affect bucket B or static view
	viewA.EraseEntriesWithGivenPrefix("dir/")

	hitAAfter, _ := viewA.LookUp("dir/file.txt", now)
	assert.False(t, hitAAfter, "Bucket A entry should be erased by prefix invalidation")

	hitBAfter, _ := viewB.LookUp("dir/file.txt", now)
	assert.True(t, hitBAfter, "Bucket B entry must NOT be affected by bucket A prefix invalidation")

	hitStaticAfter, _ := viewStatic.LookUp("dir/file.txt", now)
	assert.True(t, hitStaticAfter, "Static view entry must NOT be affected by bucket A prefix invalidation")
}

// 3. Boundary Condition: Implicit Dir vs Explicit Object Collisions
func TestAdv_ImplicitDirVsExplicitCollision(t *testing.T) {
	sharedLRU := lru.NewRadixCache(10000)
	sc := metadata.NewStatCacheBucketView(sharedLRU, "")

	now := time.Now()
	exp := now.Add(10 * time.Minute)

	// Insert explicit directory object first
	explicitObj := &gcs.MinObject{
		Name:           "my-dir/",
		Generation:     100,
		MetaGeneration: 1,
	}
	sc.Insert(explicitObj, exp)

	// Now try to insert an implicit dir placeholder for the same name
	sc.InsertImplicitDir("my-dir/", exp)

	// Lookup should still return the explicit object with full metadata (Generation 100)
	hit, obj := sc.LookUp("my-dir/", now)
	require.True(t, hit)
	assert.Equal(t, int64(100), obj.Generation, "Explicit directory object must not be overwritten by implicit dir placeholder")
}
