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

package lrucache_test

import (
	"bytes"
	"encoding/gob"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/internal/cache/lrucache"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

func TestCache(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Invariant-checking cache
////////////////////////////////////////////////////////////////////////

type invariantsCache struct {
	Wrapped lrucache.Cache
}

func (c *invariantsCache) Insert(
		key string,
		value lrucache.ValueType) []lrucache.ValueType {
	c.Wrapped.CheckInvariants()
	defer c.Wrapped.CheckInvariants()

	return c.Wrapped.Insert(key, value)
}

func (c *invariantsCache) Erase(key string) lrucache.ValueType {
	c.Wrapped.CheckInvariants()
	defer c.Wrapped.CheckInvariants()

	return c.Wrapped.Erase(key)
}

func (c *invariantsCache) LookUp(key string) lrucache.ValueType {
	c.Wrapped.CheckInvariants()
	defer c.Wrapped.CheckInvariants()

	return c.Wrapped.LookUp(key)
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const maxWeight = 50

type CacheTest struct {
	cache invariantsCache
}

func init() { RegisterTestSuite(&CacheTest{}) }

func (t *CacheTest) SetUp(ti *TestInfo) {
	t.cache.Wrapped = lrucache.New(maxWeight)
}

type testData struct {
	Value    int64
	DataSize uint64
}

func (td testData) Size() uint64 {
	return td.DataSize
}

////////////////////////////////////////////////////////////////////////
// Test functions
////////////////////////////////////////////////////////////////////////

func (t *CacheTest) LookUpInEmptyCache() {
	ExpectEq(nil, t.cache.LookUp(""))
	ExpectEq(nil, t.cache.LookUp("taco"))
}

func (t *CacheTest) InsertNilValue() {
	ExpectThat(
		func() { t.cache.Insert("taco", nil) },
		Panics(HasSubstr("nil value")),
	)
}

func (t *CacheTest) LookUpUnknownKey() {
	t.cache.Insert("burrito", testData{Value: 23, DataSize: 4})
	t.cache.Insert("taco", testData{Value: 23, DataSize: 8})

	ExpectEq(nil, t.cache.LookUp(""))
	ExpectEq(nil, t.cache.LookUp("enchilada"))
}

func (t *CacheTest) FillUpToCapacity() {
	// Initialize the cache.
	ret := t.cache.Insert("burrito", testData{Value: 23, DataSize: 4})
	AssertEq(len(ret), 0)

	ret = t.cache.Insert("taco", testData{Value: 26, DataSize: 20})
	AssertEq(len(ret), 0)

	ret = t.cache.Insert("enchilada", testData{Value: 28, DataSize: 26})
	AssertEq(len(ret), 0)

	ExpectEq(23, t.cache.LookUp("burrito").(testData).Value)
	ExpectEq(26, t.cache.LookUp("taco").(testData).Value)
	ExpectEq(28, t.cache.LookUp("enchilada").(testData).Value)
}

func (t *CacheTest) ExpiresLeastRecentlyUsed() {
	// Cache initialization.
	ret := t.cache.Insert("burrito", testData{Value: 23, DataSize: 4})
	AssertEq(len(ret), 0)

	ret = t.cache.Insert("taco", testData{Value: 26, DataSize: 20}) // Least recent
	AssertEq(len(ret), 0)

	ret = t.cache.Insert("enchilada", testData{Value: 28, DataSize: 26}) // Second most recent
	AssertEq(len(ret), 0)

	AssertEq(23, t.cache.LookUp("burrito").(testData).Value) // Most recent

	// Insert another.
	ret = t.cache.Insert("queso", testData{Value: 34, DataSize: 5})
	AssertEq(len(ret), 1)
	ExpectEq(26, ret[0].(testData).Value)

	// See what's left.
	ExpectEq(nil, t.cache.LookUp("taco"))
	ExpectEq(23, t.cache.LookUp("burrito").(testData).Value)
	ExpectEq(28, t.cache.LookUp("enchilada").(testData).Value)
	ExpectEq(34, t.cache.LookUp("queso").(testData).Value)
}

func (t *CacheTest) Overwrite() {
	ret := t.cache.Insert("burrito", testData{Value: 23, DataSize: 4})
	AssertEq(len(ret), 0)

	ret = t.cache.Insert("taco", testData{Value: 26, DataSize: 20})
	AssertEq(len(ret), 0)

	ret = t.cache.Insert("enchilada", testData{Value: 28, DataSize: 20})
	AssertEq(len(ret), 0)

	ret = t.cache.Insert("burrito", testData{Value: 33, DataSize: 6})
	AssertEq(len(ret), 0)

	// Increase the DataSize while modifying, so eviction should happen
	ret = t.cache.Insert("burrito", testData{Value: 33, DataSize: 12})
	AssertEq(len(ret), 1)
	ExpectEq(ret[0].(testData).Value, 26)

	ExpectEq(nil, t.cache.LookUp("taco"))
	ExpectEq(33, t.cache.LookUp("burrito").(testData).Value)
	ExpectEq(28, t.cache.LookUp("enchilada").(testData).Value)
}

func (t *CacheTest) TestMultipleEviction() {
	ret := t.cache.Insert("burrito", testData{Value: 23, DataSize: 4})
	AssertEq(len(ret), 0)

	ret = t.cache.Insert("taco", testData{Value: 26, DataSize: 20})
	AssertEq(len(ret), 0)

	ret = t.cache.Insert("enchilada", testData{Value: 28, DataSize: 20})
	AssertEq(len(ret), 0)

	// Increase the DataSize while modifying, so eviction should happen
	ret = t.cache.Insert("large_data", testData{Value: 33, DataSize: 45})
	AssertEq(len(ret), 3)
	ExpectEq(ret[0].(testData).Value, 23)
	ExpectEq(ret[1].(testData).Value, 26)
	ExpectEq(ret[2].(testData).Value, 28)

	ExpectEq(nil, t.cache.LookUp("taco"))
	ExpectEq(nil, t.cache.LookUp("burrito"))
	ExpectEq(nil, t.cache.LookUp("enchilada"))
	ExpectEq(33, t.cache.LookUp("large_data").(testData).Value)
}

func (t *CacheTest) Encode_EmptyCache() {
	// Encode
	buf := new(bytes.Buffer)
	encoder := gob.NewEncoder(buf)
	AssertEq(nil, encoder.Encode(&t.cache))

	// Decode
	decoder := gob.NewDecoder(buf)
	var decoded invariantsCache
	AssertEq(nil, decoder.Decode(&decoded))

	ExpectEq(nil, decoded.LookUp(""))
	ExpectEq(nil, decoded.LookUp("taco"))
}

func (t *CacheTest) Encode_PreservesLRUOrderAndCapacity() {
	gob.Register(testData{Value: 30, DataSize: 23})

	// Contents
	AssertEq(50, maxWeight)

	ret := t.cache.Insert("burrito", testData{Value: 23, DataSize: 4})
	AssertEq(len(ret), 0)

	ret = t.cache.Insert("taco", testData{Value: 26, DataSize: 20}) // Least recent
	AssertEq(len(ret), 0)

	ret = t.cache.Insert("enchilada", testData{Value: 28, DataSize: 26}) // Second most recent
	AssertEq(len(ret), 0)

	AssertEq(23, t.cache.LookUp("burrito").(testData).Value) // Most recent

	// Encode
	buf := new(bytes.Buffer)
	encoder := gob.NewEncoder(buf)
	AssertEq(nil, encoder.Encode(&t.cache))

	// Decode
	decoder := gob.NewDecoder(buf)
	var decoded invariantsCache
	AssertEq(nil, decoder.Decode(&decoded))

	// Insert another.
	evictedValue := decoded.Insert("queso", testData{Value: 33, DataSize: 26})
	AssertEq(2, len(evictedValue))

	ExpectEq(26, evictedValue[0].(testData).Value)
	ExpectEq(28, evictedValue[1].(testData).Value)

	// See what's left.
	ExpectEq(nil, decoded.LookUp("taco"))
	ExpectEq(nil, decoded.LookUp("enchilada"))
	ExpectEq(23, decoded.LookUp("burrito").(testData).Value)
	ExpectEq(33, decoded.LookUp("queso").(testData).Value)
}
