// Copyright 2023 Google LLC
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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/vipnydav/gcsfuse/v3/cfg"
	"github.com/vipnydav/gcsfuse/v3/internal/cache/lru"
	"github.com/vipnydav/gcsfuse/v3/internal/cache/metadata"
	"github.com/vipnydav/gcsfuse/v3/internal/storage/gcs"
)

func TestStatCache(testSuite *testing.T) {
	suite.Run(testSuite, new(StatCacheTest))
	suite.Run(testSuite, new(MultiBucketStatCacheTest))
}

////////////////////////////////////////////////////////////////////////
// Test-helper cache wrappers
////////////////////////////////////////////////////////////////////////

type testHelperCache struct {
	wrapped metadata.StatCache
}

func (c *testHelperCache) Insert(
	m *gcs.MinObject,
	expiration time.Time) {
	c.wrapped.Insert(m, expiration)
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
	now time.Time) (hit bool, m *gcs.MinObject) {
	hit, m = c.wrapped.LookUp(name, now)
	return
}

func (c *testHelperCache) LookUpOrNil(
	name string,
	now time.Time) (m *gcs.MinObject) {
	_, m = c.LookUp(name, now)
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
	suite.Suite
	cache testHelperCache
	//t.cache is the wrapper class on top of metadata.StatCache.
	//This approach tests wrapper methods instead of directly testing actual functionality, compromising the safety net.
	//For instance: If the helper class changes internally and stops calling stat cache methods, tests won't fail,
	//hence tests are not being safety net capturing behaviour change of actual functionality.
	//added stat cache to test metadata.StatCache directly, removing unnecessary wrappers for accurate unit testing.
	//Changing every test will increase the scope and actual hns work will be affected, so taking cautious call to just add new test in refactored way in first go,
	//we can update rest of tests slowly later.
	statCache metadata.StatCache
}

type MultiBucketStatCacheTest struct {
	suite.Suite
	multiBucketCache testMultiBucketCacheHelper
}

func (t *StatCacheTest) SetupTest() {
	cache := lru.NewCache(uint64((cfg.AverageSizeOfPositiveStatCacheEntry + cfg.AverageSizeOfNegativeStatCacheEntry) * capacity))
	t.cache.wrapped = metadata.NewStatCacheBucketView(cache, "") // this demonstrates
	t.statCache = metadata.NewStatCacheBucketView(cache, "")     // this demonstrates
	// that if you are using a cache for a single bucket, then
	// its prepending bucketName can be left empty("") without any problem.
}

func (t *MultiBucketStatCacheTest) SetupTest() {
	sharedCache := lru.NewCache(uint64((cfg.AverageSizeOfPositiveStatCacheEntry + cfg.AverageSizeOfNegativeStatCacheEntry) * capacity))
	t.multiBucketCache.fruits = testHelperCache{wrapped: metadata.NewStatCacheBucketView(sharedCache, "fruits")}
	t.multiBucketCache.spices = testHelperCache{wrapped: metadata.NewStatCacheBucketView(sharedCache, "spices")}
}

////////////////////////////////////////////////////////////////////////
// Test functions
////////////////////////////////////////////////////////////////////////

func (t *StatCacheTest) Test_LookUpInEmptyCache() {
	assert.False(t.T(), t.cache.Hit("", someTime))
	assert.False(t.T(), t.cache.Hit("taco", someTime))
}

func (t *StatCacheTest) Test_LookUpUnknownKey() {
	m0 := &gcs.MinObject{Name: "burrito"}
	m1 := &gcs.MinObject{Name: "taco"}

	t.cache.Insert(m0, someTime.Add(time.Second))
	t.cache.Insert(m1, someTime.Add(time.Second))

	assert.False(t.T(), t.cache.Hit("", someTime))
	assert.False(t.T(), t.cache.Hit("enchilada", someTime))
}

func (t *StatCacheTest) Test_KeysPresentButEverythingIsExpired() {
	m0 := &gcs.MinObject{Name: "burrito"}
	m1 := &gcs.MinObject{Name: "taco"}

	t.cache.Insert(m0, someTime.Add(-time.Second))
	t.cache.Insert(m1, someTime.Add(-time.Second))

	assert.False(t.T(), t.cache.Hit("burrito", someTime))
	assert.False(t.T(), t.cache.Hit("taco", someTime))
}

func (t *StatCacheTest) Test_FillUpToCapacity() {
	assert.Equal(t.T(), 3, capacity) // maxSize = 3 * 1640 = 4920 bytes

	m0 := &gcs.MinObject{Name: "burrito"}
	m1 := &gcs.MinObject{Name: "taco"}
	m2 := &gcs.MinObject{Name: "quesadilla"}

	t.cache.Insert(m0, expiration)                    // size = 1410 bytes
	t.cache.Insert(m1, expiration)                    // size = 1398 bytes (cumulative = 2808 bytes)
	t.cache.AddNegativeEntry("enchilada", expiration) // size = 178 bytes (cumulative = 2986 bytes)
	t.cache.Insert(m2, expiration)                    // size = 1422 bytes (cumulative = 4408 bytes)
	t.cache.AddNegativeEntry("fajita", expiration)    // size = 172 bytes (cumulative = 4580 bytes)
	t.cache.AddNegativeEntry("salsa", expiration)     // size = 170 bytes (cumulative = 4750 bytes)

	// Before expiration
	justBefore := expiration.Add(-time.Nanosecond)
	assert.Equal(t.T(), m0, t.cache.LookUpOrNil("burrito", justBefore))
	assert.Equal(t.T(), m1, t.cache.LookUpOrNil("taco", justBefore))
	assert.True(t.T(), t.cache.NegativeEntry("enchilada", justBefore))
	assert.Equal(t.T(), m2, t.cache.LookUpOrNil("quesadilla", justBefore))
	assert.True(t.T(), t.cache.NegativeEntry("fajita", justBefore))
	assert.True(t.T(), t.cache.NegativeEntry("salsa", justBefore))

	// At expiration
	assert.Equal(t.T(), m0, t.cache.LookUpOrNil("burrito", expiration))
	assert.Equal(t.T(), m1, t.cache.LookUpOrNil("taco", expiration))
	assert.True(t.T(), t.cache.NegativeEntry("enchilada", expiration))
	assert.Equal(t.T(), m2, t.cache.LookUpOrNil("quesadilla", expiration))
	assert.True(t.T(), t.cache.NegativeEntry("fajita", expiration))
	assert.True(t.T(), t.cache.NegativeEntry("salsa", expiration))

	// After expiration
	justAfter := expiration.Add(time.Nanosecond)
	assert.False(t.T(), t.cache.Hit("burrito", justAfter))
	assert.False(t.T(), t.cache.Hit("taco", justAfter))
	assert.False(t.T(), t.cache.Hit("enchilada", justAfter))
	assert.False(t.T(), t.cache.Hit("quesadilla", justAfter))
	assert.False(t.T(), t.cache.Hit("fajita", justAfter))
	assert.False(t.T(), t.cache.Hit("salsa", justAfter))
}

func (t *StatCacheTest) Test_ExpiresLeastRecentlyUsed() {
	assert.Equal(t.T(), 3, capacity) // maxSize = 3 * 1640 = 4920 bytes

	o0 := &gcs.MinObject{Name: "burrito"}
	o1 := &gcs.MinObject{Name: "taco"}
	o2 := &gcs.MinObject{Name: "quesadilla"}

	t.cache.Insert(o0, expiration)                                    // size = 1410 bytes
	t.cache.Insert(o1, expiration)                                    // Least recent, size = 1398 bytes (cumulative = 2808 bytes)
	t.cache.AddNegativeEntry("enchilada", expiration)                 // Third most recent, size = 178 bytes (cumulative = 2986 bytes)
	t.cache.Insert(o2, expiration)                                    // Second most recent, size = 1422 bytes (cumulative = 4408 bytes)
	assert.Equal(t.T(), o0, t.cache.LookUpOrNil("burrito", someTime)) // Most recent

	// Insert another.
	o3 := &gcs.MinObject{Name: "queso"}
	t.cache.Insert(o3, expiration) // size = 1402 bytes (cumulative = 5810 bytes)
	// This would evict the least recent entry i.e o1/"taco".

	// See what's left.
	assert.False(t.T(), t.cache.Hit("taco", someTime))
	assert.Equal(t.T(), o0, t.cache.LookUpOrNil("burrito", someTime))
	assert.True(t.T(), t.cache.NegativeEntry("enchilada", someTime))
	assert.Equal(t.T(), o2, t.cache.LookUpOrNil("quesadilla", someTime))
	assert.Equal(t.T(), o3, t.cache.LookUpOrNil("queso", someTime))
}

func (t *StatCacheTest) Test_Overwrite_NewerGeneration() {
	m0 := &gcs.MinObject{Name: "taco", Generation: 17, MetaGeneration: 5}
	m1 := &gcs.MinObject{Name: "taco", Generation: 19, MetaGeneration: 1}

	t.cache.Insert(m0, expiration)
	t.cache.Insert(m1, expiration)

	assert.Equal(t.T(), m1, t.cache.LookUpOrNil("taco", someTime))

	// The overwritten entry shouldn't count toward capacity.
	assert.Equal(t.T(), 3, capacity)

	t.cache.Insert(&gcs.MinObject{Name: "burrito"}, expiration)
	t.cache.Insert(&gcs.MinObject{Name: "enchilada"}, expiration)

	assert.NotEqual(t.T(), nil, t.cache.LookUpOrNil("taco", someTime))
	assert.NotEqual(t.T(), nil, t.cache.LookUpOrNil("burrito", someTime))
	assert.NotEqual(t.T(), nil, t.cache.LookUpOrNil("enchilada", someTime))
}

func (t *StatCacheTest) Test_Overwrite_SameGeneration_NewerMetadataGen() {
	m0 := &gcs.MinObject{Name: "taco", Generation: 17, MetaGeneration: 5}
	m1 := &gcs.MinObject{Name: "taco", Generation: 17, MetaGeneration: 7}

	t.cache.Insert(m0, expiration)
	t.cache.Insert(m1, expiration)

	assert.Equal(t.T(), m1, t.cache.LookUpOrNil("taco", someTime))

	// The overwritten entry shouldn't count toward capacity.
	assert.Equal(t.T(), 3, capacity)

	t.cache.Insert(&gcs.MinObject{Name: "burrito"}, expiration)
	t.cache.Insert(&gcs.MinObject{Name: "enchilada"}, expiration)

	assert.NotEqual(t.T(), nil, t.cache.LookUpOrNil("taco", someTime))
	assert.NotEqual(t.T(), nil, t.cache.LookUpOrNil("burrito", someTime))
	assert.NotEqual(t.T(), nil, t.cache.LookUpOrNil("enchilada", someTime))
}

func (t *StatCacheTest) Test_Overwrite_SameGeneration_SameMetadataGen() {
	m0 := &gcs.MinObject{Name: "taco", Generation: 17, MetaGeneration: 5}
	m1 := &gcs.MinObject{Name: "taco", Generation: 17, MetaGeneration: 5}

	t.cache.Insert(m0, expiration)
	t.cache.Insert(m1, expiration)

	assert.Equal(t.T(), m1, t.cache.LookUpOrNil("taco", someTime))
}

func (t *StatCacheTest) Test_Overwrite_SameGeneration_OlderMetadataGen() {
	m0 := &gcs.MinObject{Name: "taco", Generation: 17, MetaGeneration: 5}
	m1 := &gcs.MinObject{Name: "taco", Generation: 17, MetaGeneration: 3}

	t.cache.Insert(m0, expiration)
	t.cache.Insert(m1, expiration)

	assert.Equal(t.T(), m0, t.cache.LookUpOrNil("taco", someTime))
}

func (t *StatCacheTest) Test_Overwrite_OlderGeneration() {
	m0 := &gcs.MinObject{Name: "taco", Generation: 17, MetaGeneration: 5}
	m1 := &gcs.MinObject{Name: "taco", Generation: 13, MetaGeneration: 7}

	t.cache.Insert(m0, expiration)
	t.cache.Insert(m1, expiration)

	assert.Equal(t.T(), m0, t.cache.LookUpOrNil("taco", someTime))
}

func (t *StatCacheTest) Test_Overwrite_NegativeWithPositive() {
	const name = "taco"
	m1 := &gcs.MinObject{Name: name, Generation: 13, MetaGeneration: 7}

	t.cache.AddNegativeEntry(name, expiration)
	t.cache.Insert(m1, expiration)

	assert.Equal(t.T(), m1, t.cache.LookUpOrNil(name, someTime))
}

func (t *StatCacheTest) Test_Overwrite_PositiveWithNegative() {
	const name = "taco"
	m0 := &gcs.MinObject{Name: name, Generation: 13, MetaGeneration: 7}

	t.cache.Insert(m0, expiration)
	t.cache.AddNegativeEntry(name, expiration)

	assert.True(t.T(), t.cache.NegativeEntry(name, someTime))
}

func (t *StatCacheTest) Test_Overwrite_NegativeWithNegative() {
	const name = "taco"

	t.cache.AddNegativeEntry(name, expiration)
	t.cache.AddNegativeEntry(name, expiration)

	assert.True(t.T(), t.cache.NegativeEntry(name, someTime))
}

// ///////////////////////////////////////////////////////////////
// ////// Tests for multi-bucket cache scenarios /////////////////
// ///////////////////////////////////////////////////////////////
var (
	apple    = &gcs.MinObject{Name: "apple"}
	orange   = &gcs.MinObject{Name: "orange"}
	cardamom = &gcs.MinObject{Name: "cardamom"}
)

func (t *MultiBucketStatCacheTest) Test_CreateEntriesWithSameNameInDifferentBuckets() {
	assert.Equal(t.T(), 3, capacity)

	cache := &t.multiBucketCache
	fruits := &cache.fruits
	spices := &cache.spices

	spices.AddNegativeEntry("apple", expiration)
	// the following should not overwrite the previous entry.
	fruits.Insert(apple, expiration)

	// Before expiration
	justBefore := expiration.Add(-time.Nanosecond)
	assert.Equal(t.T(), apple, fruits.LookUpOrNil("apple", justBefore))
	assert.True(t.T(), spices.NegativeEntry("apple", justBefore))
}

func (t *MultiBucketStatCacheTest) Test_FillUpToCapacity() {
	assert.Equal(t.T(), 3, capacity) // maxSize = 3 * 1640 = 4920 bytes

	cache := &t.multiBucketCache
	fruits := &cache.fruits
	spices := &cache.spices

	fruits.Insert(apple, expiration)               // size = 1416 bytes
	fruits.Insert(orange, expiration)              // size = 1420 bytes (cumulative = 2836 bytes)
	spices.Insert(cardamom, expiration)            // size = 1428 bytes (cumulative = 4264 bytes)
	fruits.AddNegativeEntry("papaya", expiration)  // size = 186 bytes (cumulative = 4450 bytes)
	spices.AddNegativeEntry("saffron", expiration) // size = 188 bytes (cumulative = 4638 bytes)
	spices.AddNegativeEntry("pepper", expiration)  // size = 186 bytes (cumulative = 4824 bytes)

	// Before expiration
	justBefore := expiration.Add(-time.Nanosecond)
	assert.Equal(t.T(), apple, fruits.LookUpOrNil("apple", justBefore))
	assert.Equal(t.T(), orange, fruits.LookUpOrNil("orange", justBefore))
	assert.Equal(t.T(), cardamom, spices.LookUpOrNil("cardamom", justBefore))
	assert.True(t.T(), fruits.NegativeEntry("papaya", justBefore))
	assert.True(t.T(), spices.NegativeEntry("saffron", justBefore))
	assert.True(t.T(), spices.NegativeEntry("pepper", justBefore))

	// At expiration
	assert.Equal(t.T(), apple, fruits.LookUpOrNil("apple", expiration))
	assert.Equal(t.T(), orange, fruits.LookUpOrNil("orange", expiration))
	assert.Equal(t.T(), cardamom, spices.LookUpOrNil("cardamom", expiration))
	assert.True(t.T(), fruits.NegativeEntry("papaya", expiration))
	assert.True(t.T(), spices.NegativeEntry("saffron", expiration))
	assert.True(t.T(), spices.NegativeEntry("pepper", expiration))

	// After expiration
	justAfter := expiration.Add(time.Nanosecond)
	assert.False(t.T(), fruits.Hit("apple", justAfter))
	assert.False(t.T(), fruits.Hit("orange", justAfter))
	assert.False(t.T(), spices.Hit("cardamom", justAfter))
	assert.False(t.T(), fruits.Hit("papaya", justAfter))
	assert.False(t.T(), spices.Hit("saffron", justAfter))
	assert.False(t.T(), spices.Hit("pepper", justAfter))
}

func (t *MultiBucketStatCacheTest) Test_ExpiresLeastRecentlyUsed() {
	assert.Equal(t.T(), 3, capacity) // maxSize = 3 * 1640 = 4920 bytes

	cache := &t.multiBucketCache
	fruits := &cache.fruits
	spices := &cache.spices

	fruits.Insert(apple, expiration)                                  // size = 1416 bytes
	fruits.Insert(orange, expiration)                                 // Least recent, size = 1420 bytes (cumulative = 2836 bytes)
	spices.Insert(cardamom, expiration)                               // Second most recent, size = 1428 bytes (cumulative = 4264 bytes)
	assert.Equal(t.T(), apple, fruits.LookUpOrNil("apple", someTime)) // Most recent

	// Insert another.
	saffron := &gcs.MinObject{Name: "saffron"}
	spices.Insert(saffron, expiration) // size = 1424 bytes (cumulative = 5688 bytes)
	// This will evict the least recent entry, i.e. orange.

	// See what's left.
	assert.False(t.T(), fruits.Hit("orange", someTime))
	assert.Equal(t.T(), apple, fruits.LookUpOrNil("apple", someTime))
	assert.Equal(t.T(), cardamom, spices.LookUpOrNil("cardamom", someTime))
	assert.Equal(t.T(), saffron, spices.LookUpOrNil("saffron", someTime))
}

func (t *StatCacheTest) Test_InsertFolderCreateEntryWhenNoEntryIsPresent() {
	const name = "key1/"
	newEntry := &gcs.Folder{
		Name: name,
	}

	t.statCache.InsertFolder(newEntry, expiration)

	hit, entry := t.statCache.LookUpFolder(name, someTime)
	assert.True(t.T(), hit)
	assert.Equal(t.T(), name, entry.Name)
}

func (t *StatCacheTest) Test_InsertFolderOverrideEntryOldEntryIsAlreadyPresent() {
	const name = "key1/"
	existingEntry := &gcs.Folder{
		Name: name,
	}
	t.statCache.InsertFolder(existingEntry, expiration)
	newEntry := &gcs.Folder{
		Name: name,
	}

	t.statCache.InsertFolder(newEntry, expiration)

	hit, entry := t.statCache.LookUpFolder(name, someTime)
	assert.True(t.T(), hit)
	assert.Equal(t.T(), name, entry.Name)
}

func (t *StatCacheTest) Test_LookupReturnFalseIfExpirationIsPassed() {
	const name = "key1/"
	entry := &gcs.Folder{
		Name: name,
	}
	t.statCache.InsertFolder(entry, expiration)

	hit, result := t.statCache.LookUpFolder(name, expiration.Add(time.Second))

	assert.False(t.T(), hit)
	assert.Nil(t.T(), result)
}

func (t *StatCacheTest) Test_LookupReturnFalseWhenIsNotPresent() {
	const name = "key1/"

	hit, result := t.statCache.LookUpFolder(name, expiration.Add(time.Second))

	assert.False(t.T(), hit)
	assert.Nil(t.T(), result)
}

func (t *StatCacheTest) Test_InsertFolderShouldNotOverrideEntryIfMetagenerationIsOld() {
	const name = "key1/"
	existingEntry := &gcs.Folder{
		Name: name,
	}
	t.statCache.InsertFolder(existingEntry, expiration)
	newEntry := &gcs.Folder{
		Name: name,
	}

	t.statCache.InsertFolder(newEntry, expiration)

	hit, entry := t.statCache.LookUpFolder(name, someTime)
	assert.True(t.T(), hit)
	assert.Equal(t.T(), name, entry.Name)
}

func (t *StatCacheTest) Test_AddNegativeEntryForFolderShouldAddNegativeEntryForFolder() {
	const name = "key1/"
	existingEntry := &gcs.Folder{
		Name: name,
	}
	t.statCache.InsertFolder(existingEntry, expiration)

	t.statCache.AddNegativeEntryForFolder(name, expiration)

	hit, entry := t.statCache.LookUpFolder(name, someTime)
	assert.True(t.T(), hit)
	assert.Nil(t.T(), entry)
}

func (t *StatCacheTest) Test_ShouldReturnHitTrueWhenOnlyObjectAlreadyHasEntry() {
	const name = "key1"
	existingEntry := &gcs.MinObject{
		Name:           name,
		MetaGeneration: 2,
	}
	t.statCache.Insert(existingEntry, expiration)

	hit, entry := t.statCache.LookUpFolder(name, someTime)

	// If "key1" object exist then corresponding folder entry will be nil, but hit will be true as key have entry in the cache for object.
	assert.True(t.T(), hit)
	assert.Nil(t.T(), entry)
}

func (t *StatCacheTest) Test_ShouldEvictEntryOnFullCapacityIncludingFolderSize() {
	localCache := lru.NewCache(uint64(3000))
	t.statCache = metadata.NewStatCacheBucketView(localCache, "local_bucket")
	objectEntry1 := &gcs.MinObject{Name: "1"}
	objectEntry2 := &gcs.MinObject{Name: "2"}
	folderEntry := &gcs.Folder{
		Name: "3/",
	}
	t.statCache.Insert(objectEntry1, expiration) // adds size of 1428
	t.statCache.Insert(objectEntry2, expiration) // adds size of 1428

	hit1, entry1 := t.statCache.LookUp("1", someTime)
	hit2, entry2 := t.statCache.LookUp("2", someTime)

	assert.True(t.T(), hit1)
	assert.Equal(t.T(), "1", entry1.Name)
	assert.True(t.T(), hit2)
	assert.Equal(t.T(), "2", entry2.Name)

	t.statCache.InsertFolder(folderEntry, expiration) //adds size of 220 and exceeds capacity

	hit1, entry1 = t.statCache.LookUp("1", someTime)
	hit2, entry2 = t.statCache.LookUp("2", someTime)
	hit3, entry3 := t.statCache.LookUpFolder("3/", someTime)

	assert.False(t.T(), hit1)
	assert.Nil(t.T(), entry1)
	assert.True(t.T(), hit2)
	assert.Equal(t.T(), "2", entry2.Name)
	assert.True(t.T(), hit3)
	assert.Equal(t.T(), "3/", entry3.Name)
}

func (t *StatCacheTest) Test_ShouldEvictAllEntriesWithPrefixFolder() {
	localCache := lru.NewCache(uint64(10000))
	t.statCache = metadata.NewStatCacheBucketView(localCache, "local_bucket")
	folderEntry1 := &gcs.Folder{
		Name: "a",
	}
	objectEntry1 := &gcs.MinObject{Name: "a/b"}
	objectEntry2 := &gcs.MinObject{Name: "a/b/c"}
	objectEntry3 := &gcs.MinObject{Name: "d"}
	folderEntry2 := &gcs.Folder{
		Name: "a/d",
	}
	folderEntry3 := &gcs.Folder{
		Name: "b",
	}
	t.statCache.InsertFolder(folderEntry1, expiration) //adds size of 220 and exceeds capacity
	t.statCache.Insert(objectEntry1, expiration)       // adds size of 1428
	t.statCache.Insert(objectEntry2, expiration)       // adds size of 1428
	t.statCache.InsertFolder(folderEntry2, expiration) //adds size of 220 and exceeds capacity
	t.statCache.InsertFolder(folderEntry3, expiration) //adds size of 220 and exceeds capacity
	t.statCache.Insert(objectEntry3, expiration)       // adds size of 1428

	t.statCache.EraseEntriesWithGivenPrefix("a")

	hit1, entry1 := t.statCache.LookUpFolder("a", someTime)
	assert.False(t.T(), hit1)
	assert.Nil(t.T(), entry1)
	hit2, entry2 := t.statCache.LookUpFolder("a/b", someTime)
	assert.False(t.T(), hit2)
	assert.Nil(t.T(), entry2)
	hit3, entry3 := t.statCache.LookUp("a/b/c", someTime)
	assert.False(t.T(), hit3)
	assert.Nil(t.T(), entry3)
	hit4, entry4 := t.statCache.LookUpFolder("a/d", someTime)
	assert.False(t.T(), hit4)
	assert.Nil(t.T(), entry4)
	hit5, entry5 := t.statCache.LookUpFolder("b", someTime)
	assert.True(t.T(), hit5)
	assert.Equal(t.T(), "b", entry5.Name)
	hit6, entry6 := t.statCache.LookUp("d", someTime)
	assert.True(t.T(), hit6)
	assert.Equal(t.T(), "d", entry6.Name)
}
