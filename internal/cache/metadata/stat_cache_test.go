// Copyright 2023 Google Inc. All Rights Reserved.
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
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/internal/mount"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/gcs"
	. "github.com/jacobsa/ogletest"
)

func TestStatCache(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Test-helper cache wrappers
////////////////////////////////////////////////////////////////////////

type testHelperCache struct {
	wrapped metadata.StatCache
}

func (c *testHelperCache) Insert(
	minObj *gcs.MinObject,
	expiration time.Time) {
	c.wrapped.Insert(minObj, expiration)
}

func (c *testHelperCache) AddNegativeEntry(
	name string,
	expiration time.Time) {
	c.wrapped.AddNegativeEntry(name, expiration)
}

func (c *testHelperCache) Erase(name string) {
	c.wrapped.Erase(name)
}

func (c *testHelperCache) LookUp(
	name string,
	now time.Time) (hit bool, o *gcs.MinObject) {
	hit, o = c.wrapped.LookUp(name, now)
	return
}

func (c *testHelperCache) LookUpOrNil(
	name string,
	now time.Time) (o *gcs.MinObject) {
	_, o = c.LookUp(name, now)
	return
}

func (c *testHelperCache) Hit(
	name string,
	now time.Time) (hit bool) {
	hit, _ = c.LookUp(name, now)
	return
}

func (c *testHelperCache) NegativeEntry(
	name string,
	now time.Time) (negative bool) {
	hit, o := c.LookUp(name, now)
	negative = hit && o == nil
	return
}

type testMultiBucketCacheHelper struct {
	fruits testHelperCache
	spices testHelperCache
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const capacity = 3

var someTime = time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local)
var expiration = someTime.Add(time.Second)

type StatCacheTest struct {
	cache testHelperCache
}

type MultiBucketStatCacheTest struct {
	multiBucketCache testMultiBucketCacheHelper
}

func init() {
	RegisterTestSuite(&StatCacheTest{})
	RegisterTestSuite(&MultiBucketStatCacheTest{})
}

func (t *StatCacheTest) SetUp(ti *TestInfo) {
	cache := lru.NewCache(uint64(mount.AverageSizeOfPositiveStatCacheEntry * capacity))
	t.cache.wrapped = metadata.NewStatCacheBucketView(cache, "") // this demonstrates
	// that if you are using a cache for a single bucket, then
	// its prepending bucketName can be left empty("") without any problem.
}

func (t *MultiBucketStatCacheTest) SetUp(ti *TestInfo) {
	sharedCache := lru.NewCache(uint64(mount.AverageSizeOfPositiveStatCacheEntry * capacity))
	t.multiBucketCache.fruits = testHelperCache{wrapped: metadata.NewStatCacheBucketView(sharedCache, "fruits")}
	t.multiBucketCache.spices = testHelperCache{wrapped: metadata.NewStatCacheBucketView(sharedCache, "spices")}
}

////////////////////////////////////////////////////////////////////////
// Test functions
////////////////////////////////////////////////////////////////////////

func (t *StatCacheTest) LookUpInEmptyCache() {
	ExpectFalse(t.cache.Hit("", someTime))
	ExpectFalse(t.cache.Hit("taco", someTime))
}

func (t *StatCacheTest) LookUpUnknownKey() {
	o0 := &gcs.MinObject{Name: "burrito"}
	o1 := &gcs.MinObject{Name: "taco"}

	t.cache.Insert(o0, someTime.Add(time.Second))
	t.cache.Insert(o1, someTime.Add(time.Second))

	ExpectFalse(t.cache.Hit("", someTime))
	ExpectFalse(t.cache.Hit("enchilada", someTime))
}

func (t *StatCacheTest) KeysPresentButEverythingIsExpired() {
	o0 := &gcs.MinObject{Name: "burrito"}
	o1 := &gcs.MinObject{Name: "taco"}

	t.cache.Insert(o0, someTime.Add(-time.Second))
	t.cache.Insert(o1, someTime.Add(-time.Second))

	ExpectFalse(t.cache.Hit("burrito", someTime))
	ExpectFalse(t.cache.Hit("taco", someTime))
}

func (t *StatCacheTest) FillUpToCapacity() {
	AssertEq(3, capacity) // maxSize = 3 * 2400 = 7200 bytes

	o0 := &gcs.MinObject{Name: "burrito"}
	o1 := &gcs.MinObject{Name: "taco"}
	o2 := &gcs.MinObject{Name: "quesadilla"}

	t.cache.Insert(o0, expiration)                    // size = 1886 bytes
	t.cache.Insert(o1, expiration)                    // size = 1874 bytes (cumulative = 3760 bytes)
	t.cache.AddNegativeEntry("enchilada", expiration) // size = 178 bytes (cumulative = 3938 bytes)
	t.cache.Insert(o2, expiration)                    // size = 1898 bytes (cumulative = 5836 bytes)
	t.cache.AddNegativeEntry("fajita", expiration)    // size = 172 bytes (cumulative = 6008 bytes)

	// Before expiration
	justBefore := expiration.Add(-time.Nanosecond)
	ExpectEq(o0, t.cache.LookUpOrNil("burrito", justBefore))
	ExpectEq(o1, t.cache.LookUpOrNil("taco", justBefore))
	ExpectTrue(t.cache.NegativeEntry("enchilada", justBefore))
	ExpectEq(o2, t.cache.LookUpOrNil("quesadilla", justBefore))
	ExpectTrue(t.cache.NegativeEntry("fajita", justBefore))

	// At expiration
	ExpectEq(o0, t.cache.LookUpOrNil("burrito", expiration))
	ExpectEq(o1, t.cache.LookUpOrNil("taco", expiration))
	ExpectTrue(t.cache.NegativeEntry("enchilada", justBefore))
	ExpectEq(o2, t.cache.LookUpOrNil("quesadilla", justBefore))
	ExpectTrue(t.cache.NegativeEntry("fajita", justBefore))

	// After expiration
	justAfter := expiration.Add(time.Nanosecond)
	ExpectFalse(t.cache.Hit("burrito", justAfter))
	ExpectFalse(t.cache.Hit("taco", justAfter))
	ExpectFalse(t.cache.Hit("enchilada", justAfter))
	ExpectFalse(t.cache.Hit("quesadilla", justAfter))
	ExpectFalse(t.cache.Hit("fajita", justAfter))
}

func (t *StatCacheTest) ExpiresLeastRecentlyUsed() {
	AssertEq(3, capacity) // maxSize = 3 * 2400 = 7200 bytes

	o0 := &gcs.MinObject{Name: "burrito"}
	o1 := &gcs.MinObject{Name: "taco"}
	o2 := &gcs.MinObject{Name: "quesadilla"}

	t.cache.Insert(o0, expiration)                         // size = 1886 bytes
	t.cache.Insert(o1, expiration)                         // Least recent, size = 1874 bytes (cumulative = 3760 bytes)
	t.cache.AddNegativeEntry("enchilada", expiration)      // Third most recent, size = 178 bytes (cumulative = 3938 bytes)
	t.cache.Insert(o2, expiration)                         // Second most recent, size = 1898 bytes (cumulative = 5836 bytes)
	AssertEq(o0, t.cache.LookUpOrNil("burrito", someTime)) // Most recent

	// Insert another.
	o3 := &gcs.MinObject{Name: "queso"}
	t.cache.Insert(o3, expiration) // size = 1878 bytes (cumulative = 7714 bytes)
	// This would evict the least recent entry i.e o1/"taco".

	// See what's left.
	ExpectFalse(t.cache.Hit("taco", someTime))
	ExpectEq(o0, t.cache.LookUpOrNil("burrito", someTime))
	ExpectTrue(t.cache.NegativeEntry("enchilada", someTime))
	ExpectEq(o2, t.cache.LookUpOrNil("quesadilla", someTime))
	ExpectEq(o3, t.cache.LookUpOrNil("queso", someTime))
}

func (t *StatCacheTest) Overwrite_NewerGeneration() {
	o0 := &gcs.MinObject{Name: "taco", Generation: 17, MetaGeneration: 5}
	o1 := &gcs.MinObject{Name: "taco", Generation: 19, MetaGeneration: 1}

	t.cache.Insert(o0, expiration)
	t.cache.Insert(o1, expiration)

	ExpectEq(o1, t.cache.LookUpOrNil("taco", someTime))

	// The overwritten entry shouldn't count toward capacity.
	AssertEq(3, capacity)

	t.cache.Insert(&gcs.MinObject{Name: "burrito"}, expiration)
	t.cache.Insert(&gcs.MinObject{Name: "enchilada"}, expiration)

	ExpectNe(nil, t.cache.LookUpOrNil("taco", someTime))
	ExpectNe(nil, t.cache.LookUpOrNil("burrito", someTime))
	ExpectNe(nil, t.cache.LookUpOrNil("enchilada", someTime))
}

func (t *StatCacheTest) Overwrite_SameGeneration_NewerMetadataGen() {
	o0 := &gcs.MinObject{Name: "taco", Generation: 17, MetaGeneration: 5}
	o1 := &gcs.MinObject{Name: "taco", Generation: 17, MetaGeneration: 7}

	t.cache.Insert(o0, expiration)
	t.cache.Insert(o1, expiration)

	ExpectEq(o1, t.cache.LookUpOrNil("taco", someTime))

	// The overwritten entry shouldn't count toward capacity.
	AssertEq(3, capacity)

	t.cache.Insert(&gcs.MinObject{Name: "burrito"}, expiration)
	t.cache.Insert(&gcs.MinObject{Name: "enchilada"}, expiration)

	ExpectNe(nil, t.cache.LookUpOrNil("taco", someTime))
	ExpectNe(nil, t.cache.LookUpOrNil("burrito", someTime))
	ExpectNe(nil, t.cache.LookUpOrNil("enchilada", someTime))
}

func (t *StatCacheTest) Overwrite_SameGeneration_SameMetadataGen() {
	o0 := &gcs.MinObject{Name: "taco", Generation: 17, MetaGeneration: 5}
	o1 := &gcs.MinObject{Name: "taco", Generation: 17, MetaGeneration: 5}

	t.cache.Insert(o0, expiration)
	t.cache.Insert(o1, expiration)

	ExpectEq(o1, t.cache.LookUpOrNil("taco", someTime))
}

func (t *StatCacheTest) Overwrite_SameGeneration_OlderMetadataGen() {
	o0 := &gcs.MinObject{Name: "taco", Generation: 17, MetaGeneration: 5}
	o1 := &gcs.MinObject{Name: "taco", Generation: 17, MetaGeneration: 3}

	t.cache.Insert(o0, expiration)
	t.cache.Insert(o1, expiration)

	ExpectEq(o0, t.cache.LookUpOrNil("taco", someTime))
}

func (t *StatCacheTest) Overwrite_OlderGeneration() {
	o0 := &gcs.MinObject{Name: "taco", Generation: 17, MetaGeneration: 5}
	o1 := &gcs.MinObject{Name: "taco", Generation: 13, MetaGeneration: 7}

	t.cache.Insert(o0, expiration)
	t.cache.Insert(o1, expiration)

	ExpectEq(o0, t.cache.LookUpOrNil("taco", someTime))
}

func (t *StatCacheTest) Overwrite_NegativeWithPositive() {
	const name = "taco"
	o1 := &gcs.MinObject{Name: name, Generation: 13, MetaGeneration: 7}

	t.cache.AddNegativeEntry(name, expiration)
	t.cache.Insert(o1, expiration)

	ExpectEq(o1, t.cache.LookUpOrNil(name, someTime))
}

func (t *StatCacheTest) Overwrite_PositiveWithNegative() {
	const name = "taco"
	o0 := &gcs.MinObject{Name: name, Generation: 13, MetaGeneration: 7}

	t.cache.Insert(o0, expiration)
	t.cache.AddNegativeEntry(name, expiration)

	ExpectTrue(t.cache.NegativeEntry(name, someTime))
}

func (t *StatCacheTest) Overwrite_NegativeWithNegative() {
	const name = "taco"

	t.cache.AddNegativeEntry(name, expiration)
	t.cache.AddNegativeEntry(name, expiration)

	ExpectTrue(t.cache.NegativeEntry(name, someTime))
}

// ///////////////////////////////////////////////////////////////
// ////// Tests for multi-bucket cache scenarios /////////////////
// ///////////////////////////////////////////////////////////////
var (
	apple    = &gcs.MinObject{Name: "apple"}
	orange   = &gcs.MinObject{Name: "orange"}
	cardamom = &gcs.MinObject{Name: "cardamom"}
)

func (t *MultiBucketStatCacheTest) CreateEntriesWithSameNameInDifferentBuckets() {
	AssertEq(3, capacity)

	cache := &t.multiBucketCache
	fruits := &cache.fruits
	spices := &cache.spices

	spices.AddNegativeEntry("apple", expiration)
	// the following should not overwrite the previous entry.
	fruits.Insert(apple, expiration)

	// Before expiration
	justBefore := expiration.Add(-time.Nanosecond)
	ExpectEq(apple, fruits.LookUpOrNil("apple", justBefore))
	ExpectTrue(spices.NegativeEntry("apple", justBefore))
}

func (t *MultiBucketStatCacheTest) FillUpToCapacity() {
	AssertEq(3, capacity) // maxSize = 3 * 2400 = 7200 bytes

	cache := &t.multiBucketCache
	fruits := &cache.fruits
	spices := &cache.spices

	fruits.Insert(apple, expiration)               // size = 1892 bytes
	fruits.Insert(orange, expiration)              // size = 1896 bytes (cumulative = 3788 bytes)
	spices.Insert(cardamom, expiration)            // size = 1904 bytes (cumulative = 5692 bytes)
	fruits.AddNegativeEntry("papaya", expiration)  // size = 186 bytes (cumulative = 5878 bytes)
	spices.AddNegativeEntry("saffron", expiration) // size = 188 bytes (cumulative = 6066 bytes)
	spices.AddNegativeEntry("pepper", expiration)  // size = 186 bytes (cumulative = 6252 bytes)

	// Before expiration
	justBefore := expiration.Add(-time.Nanosecond)
	ExpectEq(apple, fruits.LookUpOrNil("apple", justBefore))
	ExpectEq(orange, fruits.LookUpOrNil("orange", justBefore))
	ExpectEq(cardamom, spices.LookUpOrNil("cardamom", justBefore))
	ExpectTrue(fruits.NegativeEntry("papaya", justBefore))
	ExpectTrue(spices.NegativeEntry("saffron", justBefore))
	ExpectTrue(spices.NegativeEntry("pepper", justBefore))

	// At expiration
	ExpectEq(apple, fruits.LookUpOrNil("apple", expiration))
	ExpectEq(orange, fruits.LookUpOrNil("orange", expiration))
	ExpectEq(cardamom, spices.LookUpOrNil("cardamom", justBefore))
	ExpectTrue(fruits.NegativeEntry("papaya", justBefore))
	ExpectTrue(spices.NegativeEntry("saffron", justBefore))
	ExpectTrue(spices.NegativeEntry("pepper", justBefore))

	// After expiration
	justAfter := expiration.Add(time.Nanosecond)
	ExpectFalse(fruits.Hit("apple", justAfter))
	ExpectFalse(fruits.Hit("orange", justAfter))
	ExpectFalse(spices.Hit("cardamom", justAfter))
	ExpectFalse(fruits.Hit("papaya", justAfter))
	ExpectFalse(spices.Hit("saffron", justAfter))
	ExpectFalse(spices.Hit("pepper", justAfter))
}

func (t *MultiBucketStatCacheTest) ExpiresLeastRecentlyUsed() {
	AssertEq(3, capacity) // maxSize = 3 * 2400 = 7200 bytes

	cache := &t.multiBucketCache
	fruits := &cache.fruits
	spices := &cache.spices

	fruits.Insert(apple, expiration)                       // size = 1892 bytes
	fruits.Insert(orange, expiration)                      // Least recent, size = 1896 bytes (cumulative = 3788 bytes)
	spices.Insert(cardamom, expiration)                    // Second most recent, size = 1904 bytes (cumulative = 5692 bytes)
	AssertEq(apple, fruits.LookUpOrNil("apple", someTime)) // Most recent

	// Insert another.
	saffron := &gcs.MinObject{Name: "saffron"}
	spices.Insert(saffron, expiration) // size = 1900 bytes (cumulative = 7592 bytes)
	// This will evict the least recent entry, i.e. orange.

	// See what's left.
	ExpectFalse(fruits.Hit("orange", someTime))
	ExpectEq(apple, fruits.LookUpOrNil("apple", someTime))
	ExpectEq(cardamom, spices.LookUpOrNil("cardamom", someTime))
	ExpectEq(saffron, spices.LookUpOrNil("saffron", someTime))
}
