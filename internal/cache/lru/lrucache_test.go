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

package lru_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/internal/cache/lru"
	. "github.com/jacobsa/ogletest"
)

func TestCache(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Invariant-checking cache
////////////////////////////////////////////////////////////////////////

type invariantsCache struct {
	Wrapped lru.Cache
}

func (c *invariantsCache) Insert(key string, value lru.ValueType) ([]lru.ValueType, error) {
	c.Wrapped.CheckInvariants()
	defer c.Wrapped.CheckInvariants()

	return c.Wrapped.Insert(key, value)
}

func (c *invariantsCache) Erase(key string) lru.ValueType {
	c.Wrapped.CheckInvariants()
	defer c.Wrapped.CheckInvariants()

	return c.Wrapped.Erase(key)
}

func (c *invariantsCache) LookUp(key string) lru.ValueType {
	c.Wrapped.CheckInvariants()
	defer c.Wrapped.CheckInvariants()

	return c.Wrapped.LookUp(key)
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const MaxSize = 50

type CacheTest struct {
	cache invariantsCache
}

func init() { RegisterTestSuite(&CacheTest{}) }

func (t *CacheTest) SetUp(*TestInfo) {
	t.cache.Wrapped = lru.New(MaxSize)
}

type testData struct {
	Value    int64
	DataSize uint64
}

func (td testData) Size() uint64 {
	return td.DataSize
}

// insertAndAssert inserts the given key,value in the cache and assert based on
// the expected eviction and error.
func (t *CacheTest) insertAndAssert(key string, val lru.ValueType, expectedEviction int, evictedValues []int64, expectedError error) {
	ret, err := t.cache.Insert(key, val)

	if err == nil || expectedError == nil {
		AssertEq(err, expectedError)
	} else {
		AssertEq(expectedError.Error(), err.Error())
	}
	AssertEq(expectedEviction, len(ret))
	AssertEq(len(evictedValues), expectedEviction)
	for index, value := range ret {
		ExpectEq(evictedValues[index], value.(testData).Value)
	}
}

////////////////////////////////////////////////////////////////////////
// Test functions
////////////////////////////////////////////////////////////////////////

func (t *CacheTest) LookUpInEmptyCache() {
	ExpectEq(nil, t.cache.LookUp(""))
	ExpectEq(nil, t.cache.LookUp("taco"))
}

func (t *CacheTest) InsertNilValue() {
	t.insertAndAssert("taco", nil, 0, []int64{}, errors.New(lru.InvalidEntryErrorMsg))
}

func (t *CacheTest) LookUpUnknownKey() {
	t.insertAndAssert("burrito", testData{Value: 23, DataSize: 4}, 0, []int64{}, nil)
	t.insertAndAssert("taco", testData{Value: 23, DataSize: 8}, 0, []int64{}, nil)

	ExpectEq(nil, t.cache.LookUp(""))
	ExpectEq(nil, t.cache.LookUp("enchilada"))
}

func (t *CacheTest) FillUpToCapacity() {
	t.insertAndAssert("burrito", testData{Value: 23, DataSize: 4}, 0, []int64{}, nil)
	t.insertAndAssert("taco", testData{Value: 26, DataSize: 20}, 0, []int64{}, nil)
	t.insertAndAssert("enchilada", testData{Value: 28, DataSize: 26}, 0, []int64{}, nil)

	ExpectEq(23, t.cache.LookUp("burrito").(testData).Value)
	ExpectEq(26, t.cache.LookUp("taco").(testData).Value)
	ExpectEq(28, t.cache.LookUp("enchilada").(testData).Value)
}

func (t *CacheTest) ExpiresLeastRecentlyUsed() {
	t.insertAndAssert("burrito", testData{Value: 23, DataSize: 4}, 0, []int64{}, nil)

	// Least recent.
	t.insertAndAssert("taco", testData{Value: 26, DataSize: 20}, 0, []int64{}, nil)

	// Second most recent.
	t.insertAndAssert("enchilada", testData{Value: 28, DataSize: 26}, 0, []int64{}, nil)

	AssertEq(23, t.cache.LookUp("burrito").(testData).Value) // Most recent

	// Insert another.
	t.insertAndAssert("queso", testData{Value: 34, DataSize: 5}, 1, []int64{26}, nil)

	// See what's left.
	ExpectEq(nil, t.cache.LookUp("taco"))
	ExpectEq(23, t.cache.LookUp("burrito").(testData).Value)
	ExpectEq(28, t.cache.LookUp("enchilada").(testData).Value)
	ExpectEq(34, t.cache.LookUp("queso").(testData).Value)
}

func (t *CacheTest) Overwrite() {
	t.insertAndAssert("burrito", testData{Value: 23, DataSize: 4}, 0, []int64{}, nil)
	t.insertAndAssert("taco", testData{Value: 26, DataSize: 20}, 0, []int64{}, nil)
	t.insertAndAssert("enchilada", testData{Value: 28, DataSize: 20}, 0, []int64{}, nil)
	t.insertAndAssert("burrito", testData{Value: 33, DataSize: 6}, 0, []int64{}, nil)

	// Increase the DataSize while modifying, so eviction should happen
	t.insertAndAssert("burrito", testData{Value: 33, DataSize: 12}, 1, []int64{26}, nil)

	ExpectEq(nil, t.cache.LookUp("taco"))
	ExpectEq(33, t.cache.LookUp("burrito").(testData).Value)
	ExpectEq(28, t.cache.LookUp("enchilada").(testData).Value)
}

func (t *CacheTest) TestMultipleEviction() {
	t.insertAndAssert("burrito", testData{Value: 23, DataSize: 4}, 0, []int64{}, nil)
	t.insertAndAssert("taco", testData{Value: 26, DataSize: 20}, 0, []int64{}, nil)
	t.insertAndAssert("enchilada", testData{Value: 28, DataSize: 20}, 0, []int64{}, nil)

	// Increase the DataSize while modifying, so eviction should happen
	t.insertAndAssert("large_data", testData{Value: 33, DataSize: 45}, 3, []int64{23, 26, 28}, nil)

	ExpectEq(nil, t.cache.LookUp("taco"))
	ExpectEq(nil, t.cache.LookUp("burrito"))
	ExpectEq(nil, t.cache.LookUp("enchilada"))
	ExpectEq(33, t.cache.LookUp("large_data").(testData).Value)
}

func (t *CacheTest) TestWhenEntrySizeMoreThanCacheMaxSize() {
	t.insertAndAssert("burrito", testData{Value: 23, DataSize: 4}, 0, []int64{}, nil)

	// Insert entry with size greater than maxSize of cache.
	t.insertAndAssert("taco", testData{Value: 26, DataSize: MaxSize + 1}, 0, []int64{}, errors.New(lru.InvalidEntrySizeErrorMsg))

	ExpectEq(23, t.cache.LookUp("burrito").(testData).Value)
}

func (t *CacheTest) TestEraseWhenKeyPresent() {
	t.insertAndAssert("burrito", testData{Value: 23, DataSize: 4}, 0, []int64{}, nil)

	deletedEntry := t.cache.Erase("burrito")

	ExpectEq(23, deletedEntry.(testData).Value)
	ExpectEq(nil, t.cache.LookUp("burrito"))
}

func (t *CacheTest) TestEraseWhenKeyNotPresent() {
	t.insertAndAssert("burrito", testData{Value: 23, DataSize: 4}, 0, []int64{}, nil)

	deletedEntry := t.cache.Erase("taco")
	ExpectEq(nil, deletedEntry)

	ExpectEq(23, t.cache.LookUp("burrito").(testData).Value)
}

// This will detect race if we run the test with `-race` flag.
// We get the race condition failure if we remove lock from Insert or Erase method.
func (t *CacheTest) TestRaceCondition() {
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		for i := 0; i < MaxSize; i++ {
			t.cache.Wrapped.Insert("key", testData{
				Value:    int64(i),
				DataSize: uint64(i),
			})
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < MaxSize; i++ {
			t.cache.Wrapped.Erase("key")
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < MaxSize; i++ {
			t.cache.Wrapped.LookUp("key")
		}
	}()

	wg.Wait()
}
