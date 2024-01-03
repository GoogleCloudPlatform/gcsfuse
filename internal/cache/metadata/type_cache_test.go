// Copyright 2024 Google Inc. All Rights Reserved.
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

package metadata

import (
	"fmt"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/util"
	. "github.com/jacobsa/ogletest"
)

const (
	TTL             time.Duration = time.Millisecond
	TypeCacheSizeMb               = 1
	pathToObject                  = "path/to/object"
)

var (
	now               time.Time = time.Now()
	expiration        time.Time = now.Add(TTL)
	beforeExpiration  time.Time = expiration.Add(-time.Nanosecond)
	afterExpiration   time.Time = expiration.Add(time.Nanosecond)
	now2              time.Time = now.Add(TTL / 2)
	expiration2       time.Time = now2.Add(TTL)
	beforeExpiration2 time.Time = expiration2.Add(-time.Nanosecond)
)

func TestTypeCache(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type TypeCacheTest struct {
	cache *typeCache
	ttl   time.Duration
}

type ZeroSizeTypeCacheTest struct {
	cache *typeCache
	ttl   time.Duration
}

type ZeroTtlTypeCacheTest struct {
	cache *typeCache
}

type TypeCacheBucketViewTest struct {
	// named bucket views
	// useful for dynamic-mount tests.
	typeCacheBucketViews map[string]*typeCacheBucketView

	// bucket-view with bucket passed as ""
	// useful for static-mount tests.
	defaultView *typeCacheBucketView

	ttl time.Duration
}

func init() {
	RegisterTestSuite(&TypeCacheTest{})
	RegisterTestSuite(&ZeroSizeTypeCacheTest{})
	RegisterTestSuite(&ZeroTtlTypeCacheTest{})
	RegisterTestSuite(&TypeCacheBucketViewTest{})
}

func (t *TypeCacheTest) SetUp(ti *TestInfo) {
	t.ttl = TTL
	t.cache = createNewTypeCache(TypeCacheSizeMb, t.ttl)
}

func (t *ZeroSizeTypeCacheTest) SetUp(ti *TestInfo) {
	t.ttl = TTL
	t.cache = createNewTypeCache(0, t.ttl)
}

func (t *ZeroTtlTypeCacheTest) SetUp(ti *TestInfo) {
	t.cache = createNewTypeCache(TypeCacheSizeMb, 0)
}

func (t *TypeCacheTest) TearDown() {
}

func (t *ZeroSizeTypeCacheTest) TearDown() {
}

func (t *ZeroTtlTypeCacheTest) TearDown() {
}

func (t *TypeCacheBucketViewTest) SetUp(ti *TestInfo) {
	t.ttl = TTL
	sharedCache := createNewTypeCache(1, t.ttl)
	t.typeCacheBucketViews = map[string]*typeCacheBucketView{}
	names := []string{"a", "b"}
	for _, name := range names {
		t.typeCacheBucketViews[name] = createNewTypeCacheBucketView(sharedCache, name)
	}

	t.defaultView = createNewTypeCacheBucketView(sharedCache, "")
}

func (t *TypeCacheBucketViewTest) TearDown() {
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func createNewTypeCache(sizeInMB int, ttl time.Duration) *typeCache {
	tc := NewTypeCache(sizeInMB, ttl)
	AssertNe(nil, tc)
	AssertNe(nil, tc.(*typeCache))
	return tc.(*typeCache)
}

func createNewTypeCacheBucketView(tc TypeCache, name string) *typeCacheBucketView {
	tcbv := NewTypeCacheBucketView(tc, name)
	AssertNe(nil, tcbv)
	AssertNe(nil, tcbv.(*typeCacheBucketView))
	return tcbv.(*typeCacheBucketView)
}

////////////////////////////////////////////////////////////////////////
// Tests for regulat TypeCache - TypeCacheTest
////////////////////////////////////////////////////////////////////////

func (t *TypeCacheTest) TestNewTypeCache() {
	input := []struct {
		sizeInMb           int
		ttl                time.Duration
		entriesShouldBeNil bool
	}{{
		sizeInMb:           0,
		ttl:                time.Second,
		entriesShouldBeNil: true,
	}, {
		sizeInMb:           1,
		ttl:                0,
		entriesShouldBeNil: true,
	}, {
		sizeInMb: -1,
		ttl:      time.Second,
	}, {
		sizeInMb: 1,
		ttl:      time.Second,
	}}

	for _, input := range input {
		tc := createNewTypeCache(input.sizeInMb, input.ttl)
		AssertEq(input.entriesShouldBeNil, tc.entries == nil)
	}
}

func (t *TypeCacheTest) TestGetFromEmptyTypeCache() {
	ExpectEq(UnknownType, t.cache.Get(now, "abc"))
}

func (t *TypeCacheTest) TestGetUninsertedEntry() {
	t.cache.Insert(now, "abcd", RegularFileType)
	ExpectEq(UnknownType, t.cache.Get(beforeExpiration, "abc"))
}

func (t *TypeCacheTest) TestGetOverwrittenEntry() {
	t.cache.Insert(now, "abcd", RegularFileType)
	t.cache.Insert(now, "abcd", ExplicitDirType)
	ExpectEq(ExplicitDirType, t.cache.Get(beforeExpiration, "abcd"))
}

func (t *TypeCacheTest) TestGetBeforeTtlExpiration() {
	t.cache.Insert(now, "abcd", RegularFileType)
	ExpectEq(RegularFileType, t.cache.Get(beforeExpiration, "abcd"))
}

func (t *TypeCacheTest) TestGetAfterTtlExpiration() {
	t.cache.Insert(now, "abcd", RegularFileType)
	ExpectEq(UnknownType, t.cache.Get(afterExpiration, "abcd"))
}

func (t *TypeCacheTest) TestGetAfterSizeExpiration() {
	entriesToBeInserted := int(util.MiBsToBytes(TypeCacheSizeMb)/cacheEntry{}.Size()) + 1

	for i := 0; i < entriesToBeInserted; i++ {
		t.cache.Insert(now, fmt.Sprint(i), RegularFileType)
	}

	// Verify that Get works, by accessing the last entry inserted.
	ExpectEq(RegularFileType, t.cache.Get(beforeExpiration, fmt.Sprint(entriesToBeInserted-1)))

	// The first inserted entry should have been evicted by the later insertions.
	ExpectEq(UnknownType, t.cache.Get(beforeExpiration, fmt.Sprint(0)))
}

func (t *TypeCacheTest) TestGetErasedEntry() {
	t.cache.Insert(now, "abcd", RegularFileType)
	t.cache.Erase("abcd")
	ExpectEq(UnknownType, t.cache.Get(beforeExpiration, "abcd"))
}

func (t *TypeCacheTest) TestGetReinsertedEntry() {
	t.cache.Insert(now, "abcd", RegularFileType)
	t.cache.Erase("abcd")
	t.cache.Insert(now2, "abcd", ExplicitDirType)
	ExpectEq(ExplicitDirType, t.cache.Get(beforeExpiration2, "abcd"))
}

////////////////////////////////////////////////////////////////////////
// Tests for TypeCache created with size=0 - ZeroSizeTypeCacheTest
////////////////////////////////////////////////////////////////////////

func (t *ZeroSizeTypeCacheTest) TestGetFromEmptyTypeCache() {
	ExpectEq(UnknownType, t.cache.Get(now, "abc"))
}

func (t *ZeroSizeTypeCacheTest) TestGetInsertedEntry() {
	t.cache.Insert(now, "abcd", RegularFileType)
	ExpectEq(UnknownType, t.cache.Get(beforeExpiration, "abcd"))
}

////////////////////////////////////////////////////////////////////////
// Tests for TypeCache created with ttl=0 - ZeroTtlTypeCacheTest
////////////////////////////////////////////////////////////////////////

func (t *ZeroTtlTypeCacheTest) TestGetFromEmptyTypeCache() {
	ExpectEq(UnknownType, t.cache.Get(now, "abc"))
}

func (t *ZeroTtlTypeCacheTest) TestGetInsertedEntry() {
	t.cache.Insert(now, "abcd", RegularFileType)
	ExpectEq(UnknownType, t.cache.Get(beforeExpiration, "abcd"))
}

//////////////////////////////////////////////////////////////////////////
// Tests for TypeCache created with bucket-views - TypeCacheBucketViewTest
//////////////////////////////////////////////////////////////////////////

func (t *TypeCacheBucketViewTest) TestInsertGetEntry() {
	// This tests for the multi-bucket scenario (dynamic-mount).
	// Here t.a is created by passing bucket-name as "a".
	// This test expects in such a case that the object is stored in the
	// shared type-cache with the bucket's name "a" prepended to the
	// original name "path/to/object".

	a := t.typeCacheBucketViews["a"]
	a.Insert(now, pathToObject, RegularFileType)

	AssertEq(UnknownType, a.sharedTypeCache.Get(beforeExpiration, pathToObject))
	AssertEq(RegularFileType, a.sharedTypeCache.Get(beforeExpiration, "a/path/to/object"))
	AssertEq(RegularFileType, a.Get(beforeExpiration, pathToObject))
}

func (t *TypeCacheBucketViewTest) TestInsertGetEntryForUnnamedView() {
	// This tests for the single-bucket scenario (static-mount).
	// Here t.defaultView is created by passing bucket-name as "".
	// This test expects in such a case that the object is stored in the
	// shared type-cache with the same name as the original name.
	t.defaultView.Insert(now, pathToObject, RegularFileType)

	AssertEq(RegularFileType, t.defaultView.sharedTypeCache.Get(beforeExpiration, pathToObject))
	AssertEq(RegularFileType, t.defaultView.Get(beforeExpiration, pathToObject))
}

func (t *TypeCacheBucketViewTest) TestInsertSameEntryToMultipleBuckets() {
	a := t.typeCacheBucketViews["a"]
	b := t.typeCacheBucketViews["b"]

	// add same entries in two bucket views to the same shared-type-cache
	a.Insert(now, pathToObject, RegularFileType)
	b.Insert(now, pathToObject, ExplicitDirType)
	// verify that both entries co-exist
	AssertEq(RegularFileType, a.Get(beforeExpiration, pathToObject))
	AssertEq(ExplicitDirType, b.Get(beforeExpiration, pathToObject))

	// verify that deletion of one of the entries doesn't affect the other
	a.Erase(pathToObject)
	// verify that the deleted entry is gone, but the other survived.
	AssertEq(UnknownType, a.Get(beforeExpiration, pathToObject))
	AssertEq(ExplicitDirType, b.Get(beforeExpiration, pathToObject))

	// reinsert the deleted entry at a later time
	a.Insert(now2, pathToObject, SymlinkType)
	// verify that just before the original ttl expiration, both entries co-exist
	AssertEq(SymlinkType, a.Get(beforeExpiration, pathToObject))
	AssertEq(ExplicitDirType, b.Get(beforeExpiration, pathToObject))
	// verify that just before the ttl expiration of the reinserted entry, the reinserted entry is there, but the other got evicted.
	AssertEq(SymlinkType, a.Get(beforeExpiration2, pathToObject))
	AssertEq(UnknownType, b.Get(beforeExpiration2, pathToObject))
}

func (t *TypeCacheBucketViewTest) TestTypeCacheBucketViewsSizeSharing() {
	a := t.typeCacheBucketViews["a"]
	b := t.typeCacheBucketViews["b"]
	wg := sync.WaitGroup{}
	maxEntriesInSharedCache := float64(util.MiBsToBytes(1)) / float64(cacheEntry{}.Size())
	maxEntriesPerView := int(math.Ceil(maxEntriesInSharedCache/2.0)) + 1 // +1 to purposely force the total entry-count to go up by at least 2.

	// Adding the '0' entries in both the views first out-of-loop, to ensure that these two are the earliest entries added in the shared-cache.
	a.Insert(now, fmt.Sprint(0), RegularFileType)
	b.Insert(now, fmt.Sprint(0), ExplicitDirType)

	// adding entries in the two views through parallel go-routines to ensure that
	// the entries are added in random/interleaved fashion.
	wg.Add(1)
	go func() {
		defer wg.Done()

		for i := 1; i < maxEntriesPerView-1; i++ {
			a.Insert(now, fmt.Sprint(i), RegularFileType)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		for i := 1; i < maxEntriesPerView-1; i++ {
			b.Insert(now, fmt.Sprint(i), ExplicitDirType)
		}
	}()

	wg.Wait()

	// Adding the 'maxEntriesPerView-1' entries in both the views last out-of-loop,
	// to ensure that these two are the last entries added in the shared-cache,
	// to avoid  race-conditions
	a.Insert(now, fmt.Sprint(maxEntriesPerView-1), RegularFileType)
	b.Insert(now, fmt.Sprint(maxEntriesPerView-1), ExplicitDirType)

	// verify that the last entries of both the views co-exist
	AssertEq(RegularFileType, a.Get(beforeExpiration, fmt.Sprint(maxEntriesPerView-1)))
	AssertEq(ExplicitDirType, b.Get(beforeExpiration, fmt.Sprint(maxEntriesPerView-1)))

	// verify that the first entries of both the views got evicted.
	AssertEq(UnknownType, a.Get(beforeExpiration, fmt.Sprint(0)))
	AssertEq(UnknownType, b.Get(beforeExpiration, fmt.Sprint(0)))
}
