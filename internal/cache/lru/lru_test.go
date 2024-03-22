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
	"math/rand"
	"strings"
	"sync"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/locker"
	. "github.com/jacobsa/ogletest"
)

func TestCache(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const MaxSize = 50
const OperationCount = 100

type CacheTest struct {
	cache *lru.Cache
}

func init() { RegisterTestSuite(&CacheTest{}) }

func (t *CacheTest) SetUp(*TestInfo) {
	locker.EnableInvariantsCheck()
	t.cache = lru.NewCache(MaxSize)
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
func (t *CacheTest) insertAndAssert(key string, val lru.ValueType, evictedValues []int64, expectedError error) {
	ret, err := t.cache.Insert(key, val)

	if err == nil || expectedError == nil {
		AssertEq(err, expectedError)
	} else {
		AssertEq(expectedError.Error(), err.Error())
	}
	AssertEq(len(evictedValues), len(ret))
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
	t.insertAndAssert("taco", nil, []int64{}, errors.New(lru.InvalidEntryErrorMsg))
}

func (t *CacheTest) LookUpUnknownKey() {
	t.insertAndAssert("burrito", testData{Value: 23, DataSize: 4}, []int64{}, nil)
	t.insertAndAssert("taco", testData{Value: 23, DataSize: 8}, []int64{}, nil)

	ExpectEq(nil, t.cache.LookUp(""))
	ExpectEq(nil, t.cache.LookUp("enchilada"))
}

func (t *CacheTest) FillUpToCapacity() {
	t.insertAndAssert("burrito", testData{Value: 23, DataSize: 4}, []int64{}, nil)
	t.insertAndAssert("taco", testData{Value: 26, DataSize: 20}, []int64{}, nil)
	t.insertAndAssert("enchilada", testData{Value: 28, DataSize: 26}, []int64{}, nil)

	ExpectEq(23, t.cache.LookUp("burrito").(testData).Value)
	ExpectEq(26, t.cache.LookUp("taco").(testData).Value)
	ExpectEq(28, t.cache.LookUp("enchilada").(testData).Value)
}

func (t *CacheTest) ExpiresLeastRecentlyUsed() {
	t.insertAndAssert("burrito", testData{Value: 23, DataSize: 4}, []int64{}, nil)

	// Least recent.
	t.insertAndAssert("taco", testData{Value: 26, DataSize: 20}, []int64{}, nil)

	// Second most recent.
	t.insertAndAssert("enchilada", testData{Value: 28, DataSize: 26}, []int64{}, nil)

	AssertEq(23, t.cache.LookUp("burrito").(testData).Value) // Most recent

	// Insert another.
	t.insertAndAssert("queso", testData{Value: 34, DataSize: 5}, []int64{26}, nil)

	// See what's left.
	ExpectEq(nil, t.cache.LookUp("taco"))
	ExpectEq(23, t.cache.LookUp("burrito").(testData).Value)
	ExpectEq(28, t.cache.LookUp("enchilada").(testData).Value)
	ExpectEq(34, t.cache.LookUp("queso").(testData).Value)
}

func (t *CacheTest) Overwrite() {
	t.insertAndAssert("burrito", testData{Value: 23, DataSize: 4}, []int64{}, nil)
	t.insertAndAssert("taco", testData{Value: 26, DataSize: 20}, []int64{}, nil)
	t.insertAndAssert("enchilada", testData{Value: 28, DataSize: 20}, []int64{}, nil)
	t.insertAndAssert("burrito", testData{Value: 33, DataSize: 6}, []int64{}, nil)

	// Increase the DataSize while modifying, so eviction should happen
	t.insertAndAssert("burrito", testData{Value: 33, DataSize: 12}, []int64{26}, nil)

	ExpectEq(nil, t.cache.LookUp("taco"))
	ExpectEq(33, t.cache.LookUp("burrito").(testData).Value)
	ExpectEq(28, t.cache.LookUp("enchilada").(testData).Value)
}

func (t *CacheTest) TestMultipleEviction() {
	t.insertAndAssert("burrito", testData{Value: 23, DataSize: 4}, []int64{}, nil)
	t.insertAndAssert("taco", testData{Value: 26, DataSize: 20}, []int64{}, nil)
	t.insertAndAssert("enchilada", testData{Value: 28, DataSize: 20}, []int64{}, nil)

	// Increase the DataSize while modifying, so eviction should happen
	t.insertAndAssert("large_data", testData{Value: 33, DataSize: 45}, []int64{23, 26, 28}, nil)

	ExpectEq(nil, t.cache.LookUp("taco"))
	ExpectEq(nil, t.cache.LookUp("burrito"))
	ExpectEq(nil, t.cache.LookUp("enchilada"))
	ExpectEq(33, t.cache.LookUp("large_data").(testData).Value)
}

func (t *CacheTest) TestWhenEntrySizeMoreThanCacheMaxSize() {
	t.insertAndAssert("burrito", testData{Value: 23, DataSize: 4}, []int64{}, nil)

	// Insert entry with size greater than maxSize of cache.
	t.insertAndAssert("taco", testData{Value: 26, DataSize: MaxSize + 1}, []int64{}, errors.New(lru.InvalidEntrySizeErrorMsg))

	ExpectEq(23, t.cache.LookUp("burrito").(testData).Value)
}

func (t *CacheTest) TestEraseWhenKeyPresent() {
	t.insertAndAssert("burrito", testData{Value: 23, DataSize: 4}, []int64{}, nil)

	deletedEntry := t.cache.Erase("burrito")

	ExpectEq(23, deletedEntry.(testData).Value)
	ExpectEq(nil, t.cache.LookUp("burrito"))
}

func (t *CacheTest) TestEraseWhenKeyNotPresent() {
	t.insertAndAssert("burrito", testData{Value: 23, DataSize: 4}, []int64{}, nil)

	deletedEntry := t.cache.Erase("taco")
	ExpectEq(nil, deletedEntry)

	ExpectEq(23, t.cache.LookUp("burrito").(testData).Value)
}

func (t *CacheTest) TestUpdateWhenKeyPresent() {
	key := "burrito"
	data := testData{Value: 23, DataSize: 4}
	t.insertAndAssert(key, data, []int64{}, nil)
	newData := testData{Value: 2, DataSize: 4}

	err := t.cache.UpdateWithoutChangingOrder(key, newData)

	ExpectEq(nil, err)
	ExpectEq(2, t.cache.LookUp(key).(testData).Value)
}

func (t *CacheTest) TestUpdateWhenKeyNotPresent() {
	key := "burrito"
	data := testData{Value: 23, DataSize: 4}

	err := t.cache.UpdateWithoutChangingOrder(key, data)

	ExpectNe(nil, err)
	ExpectTrue(strings.Contains(err.Error(), lru.EntryNotExistErrMsg))
}

func (t *CacheTest) TestUpdateWhenSizeIsDifferent() {
	key := "burrito"
	data := testData{Value: 23, DataSize: 4}
	t.insertAndAssert(key, data, []int64{}, nil)
	newData := testData{Value: 2, DataSize: 3}

	err := t.cache.UpdateWithoutChangingOrder(key, newData)

	ExpectNe(nil, err)
	ExpectTrue(strings.Contains(err.Error(), lru.InvalidUpdateEntrySizeErrorMsg))
}

func (t *CacheTest) TestUpdateNotChangeOrder() {
	key1 := "burrito1"
	data1 := testData{Value: 23, DataSize: 10}
	t.insertAndAssert(key1, data1, []int64{}, nil)
	key2 := "burrito2"
	data2 := testData{Value: 2, DataSize: 40}
	t.insertAndAssert(key2, data2, []int64{}, nil)

	newData := testData{Value: 7, DataSize: 10}
	err := t.cache.UpdateWithoutChangingOrder(key1, newData)

	ExpectEq(nil, err)
	// inserting again which should evict key1 because key1 is updated without
	// changing order
	key3 := "burrito3"
	data3 := testData{Value: 3, DataSize: 5}
	t.insertAndAssert(key3, data3, []int64{7}, nil)
}

func (t *CacheTest) TestLookUpWithoutChangingOrder_WhenKeyPresent() {
	key := "burrito"
	data := testData{Value: 23, DataSize: 4}
	t.insertAndAssert(key, data, []int64{}, nil)

	value := t.cache.LookUpWithoutChangingOrder(key)

	ExpectEq(23, value.(testData).Value)
}

func (t *CacheTest) TestLookUpWithoutChangingOrder_WhenKeyNotPresent() {
	key := "burrito"

	value := t.cache.LookUpWithoutChangingOrder(key)

	ExpectEq(nil, value)
}

func (t *CacheTest) TestLookUpWithoutChangingOrder_NotChangeOrder() {
	key1 := "burrito1"
	data1 := testData{Value: 23, DataSize: 10}
	t.insertAndAssert(key1, data1, []int64{}, nil)
	key2 := "burrito2"
	data2 := testData{Value: 2, DataSize: 40}
	t.insertAndAssert(key2, data2, []int64{}, nil)

	value := t.cache.LookUpWithoutChangingOrder(key1)

	ExpectEq(23, value.(testData).Value)
	// inserting again which should evict key1 because key1 is looked up without
	// changing order
	key3 := "burrito3"
	data3 := testData{Value: 3, DataSize: 5}
	t.insertAndAssert(key3, data3, []int64{23}, nil)
}

// This will detect race if we run the test with `-race` flag.
// We get the race condition failure if we remove lock from Insert or Erase method.
func (t *CacheTest) TestRaceCondition() {
	var wg sync.WaitGroup
	wg.Add(5)

	go func() {
		defer wg.Done()
		for i := 0; i < OperationCount; i++ {
			_, err := t.cache.Insert("key", testData{
				Value:    int64(i),
				DataSize: uint64(rand.Intn(MaxSize)),
			})

			AssertEq(nil, err)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < OperationCount; i++ {
			t.cache.Erase("key")
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < OperationCount; i++ {
			t.cache.LookUp("key")
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < OperationCount; i++ {
			t.cache.LookUpWithoutChangingOrder("key")
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < OperationCount; i++ {
			_ = t.cache.UpdateWithoutChangingOrder("key", testData{
				Value:    int64(i),
				DataSize: uint64(rand.Intn(MaxSize)),
			})
		}
	}()

	wg.Wait()
}
