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

	"github.com/googlecloudplatform/gcsfuse/internal/lrucache"
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

const capacity = 50

type CacheTest struct {
	cache invariantsCache
}

func init() { RegisterTestSuite(&CacheTest{}) }

func (t *CacheTest) SetUp(ti *TestInfo) {
	t.cache.Wrapped = lrucache.New(capacity)
}

type testData struct {
	lrucache.ValueType
	value int64
	size  uint64
}

func (td testData) Size() uint64 {
	return td.size
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
	t.cache.Insert("burrito", testData{value: 23, size: 4})
	t.cache.Insert("taco", testData{value: 23, size: 8})

	ExpectEq(nil, t.cache.LookUp(""))
	ExpectEq(nil, t.cache.LookUp("enchilada"))
}

func (t *CacheTest) FillUpToCapacity() {
	AssertEq(50, capacity)

	t.cache.Insert("burrito", testData{value: 23, size: 4})
	t.cache.Insert("taco", testData{value: 26, size: 20})
	t.cache.Insert("enchilada", testData{value: 28, size: 26})

	ExpectEq(23, t.cache.LookUp("burrito").(testData).value)
	ExpectEq(26, t.cache.LookUp("taco").(testData).value)
	ExpectEq(28, t.cache.LookUp("enchilada").(testData).value)
}

func (t *CacheTest) ExpiresLeastRecentlyUsed() {
	AssertEq(50, capacity)

	t.cache.Insert("burrito", testData{value: 23, size: 4})
	t.cache.Insert("taco", testData{value: 26, size: 20})      // Least recent
	t.cache.Insert("enchilada", testData{value: 28, size: 26}) // Second most recent
	AssertEq(23, t.cache.LookUp("burrito").(testData).value)   // Most recent

	// Insert another.
	t.cache.Insert("queso", testData{value: 34, size: 5})

	// See what's left.
	ExpectEq(nil, t.cache.LookUp("taco"))
	ExpectEq(23, t.cache.LookUp("burrito").(testData).value)
	ExpectEq(28, t.cache.LookUp("enchilada").(testData).value)
	ExpectEq(34, t.cache.LookUp("queso").(testData).value)
}

func (t *CacheTest) Overwrite() {
	ret := t.cache.Insert("burrito", testData{value: 23, size: 4})
	AssertEq(len(ret), 0)

	ret = t.cache.Insert("taco", testData{value: 26, size: 20})
	AssertEq(len(ret), 0)

	ret = t.cache.Insert("enchilada", testData{value: 28, size: 20})
	AssertEq(len(ret), 0)

	ret = t.cache.Insert("burrito", testData{value: 33, size: 6})
	AssertEq(len(ret), 0)

	// Increase the size while modifying, so eviction should happen
	ret = t.cache.Insert("burrito", testData{value: 33, size: 12})
	AssertEq(len(ret), 1)
	ExpectEq(ret[0].(testData).value, 26)

	ExpectEq(nil, t.cache.LookUp("taco"))
	ExpectEq(33, t.cache.LookUp("burrito").(testData).value)
	ExpectEq(28, t.cache.LookUp("enchilada").(testData).value)
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
	// This test is failing with - Encoding entries: gob: type not registered for interface: lrucache_test.testData
	// Please don't look at this.

	// Contents
	AssertEq(50, capacity)

	t.cache.Insert("burrito", testData{value: 23, size: 4})
	t.cache.Insert("taco", testData{value: 26, size: 20})      // Least recent
	t.cache.Insert("enchilada", testData{value: 28, size: 26}) // Second most recent
	AssertEq(23, t.cache.LookUp("burrito").(testData).value)   // Most recent

	// Encode
	buf := new(bytes.Buffer)
	encoder := gob.NewEncoder(buf)
	AssertEq(nil, encoder.Encode(&t.cache))

	// Decode
	decoder := gob.NewDecoder(buf)
	var decoded invariantsCache
	AssertEq(nil, decoder.Decode(&decoded))

	// Insert another.
	evictedValue := decoded.Insert("queso", testData{value: 33, size: 26})
	AssertEq(2, len(evictedValue))

	ExpectEq(26, evictedValue[0].(testData).value)
	ExpectEq(28, evictedValue[1].(testData).value)

	// See what's left.
	ExpectEq(nil, decoded.LookUp("taco"))
	ExpectEq(nil, decoded.LookUp("enchilada"))
	ExpectEq(23, decoded.LookUp("burrito").(testData).value)
	ExpectEq(33, decoded.LookUp("queso").(testData).value)
}
