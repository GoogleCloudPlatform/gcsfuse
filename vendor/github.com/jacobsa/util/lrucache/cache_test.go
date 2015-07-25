// Copyright 2015 Aaron Jacobs. All Rights Reserved.
// Author: aaronjjacobs@gmail.com (Aaron Jacobs)
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

	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/util/lrucache"
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
	value interface{}) {
	c.Wrapped.CheckInvariants()
	defer c.Wrapped.CheckInvariants()

	c.Wrapped.Insert(key, value)
	return
}

func (c *invariantsCache) Erase(key string) {
	c.Wrapped.CheckInvariants()
	defer c.Wrapped.CheckInvariants()

	c.Wrapped.Erase(key)
	return
}

func (c *invariantsCache) LookUp(key string) (v interface{}) {
	c.Wrapped.CheckInvariants()
	defer c.Wrapped.CheckInvariants()

	v = c.Wrapped.LookUp(key)
	return
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const capacity = 3

type CacheTest struct {
	cache invariantsCache
}

func init() { RegisterTestSuite(&CacheTest{}) }

func (t *CacheTest) SetUp(ti *TestInfo) {
	t.cache.Wrapped = lrucache.New(capacity)
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
	t.cache.Insert("burrito", 17)
	t.cache.Insert("taco", 19)

	ExpectEq(nil, t.cache.LookUp(""))
	ExpectEq(nil, t.cache.LookUp("enchilada"))
}

func (t *CacheTest) FillUpToCapacity() {
	AssertEq(3, capacity)

	t.cache.Insert("burrito", 17)
	t.cache.Insert("taco", 19)
	t.cache.Insert("enchilada", []byte{0x23, 0x29})

	ExpectEq(17, t.cache.LookUp("burrito"))
	ExpectEq(19, t.cache.LookUp("taco"))
	ExpectThat(t.cache.LookUp("enchilada"), DeepEquals([]byte{0x23, 0x29}))
}

func (t *CacheTest) ExpiresLeastRecentlyUsed() {
	AssertEq(3, capacity)

	t.cache.Insert("burrito", 17)
	t.cache.Insert("taco", 19)              // Least recent
	t.cache.Insert("enchilada", 23)         // Second most recent
	AssertEq(17, t.cache.LookUp("burrito")) // Most recent

	// Insert another.
	t.cache.Insert("queso", 29)

	// See what's left.
	ExpectEq(nil, t.cache.LookUp("taco"))
	ExpectEq(17, t.cache.LookUp("burrito"))
	ExpectEq(23, t.cache.LookUp("enchilada"))
	ExpectEq(29, t.cache.LookUp("queso"))
}

func (t *CacheTest) Overwrite() {
	// Write several times
	t.cache.Insert("taco", 17)
	t.cache.Insert("taco", 19)
	t.cache.Insert("taco", 23)

	// Look up
	ExpectEq(23, t.cache.LookUp("taco"))

	// The overwritten entries shouldn't count toward capacity.
	AssertEq(3, capacity)

	t.cache.Insert("burrito", 29)
	t.cache.Insert("enchilada", 31)

	ExpectEq(23, t.cache.LookUp("taco"))
	ExpectEq(29, t.cache.LookUp("burrito"))
	ExpectEq(31, t.cache.LookUp("enchilada"))
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
	// Contents
	AssertEq(3, capacity)

	t.cache.Insert("burrito", 17)
	t.cache.Insert("taco", 19)                      // Least recent
	t.cache.Insert("enchilada", []byte{0x23, 0x29}) // Second most recent
	AssertEq(17, t.cache.LookUp("burrito"))         // Most recent

	// Encode
	buf := new(bytes.Buffer)
	encoder := gob.NewEncoder(buf)
	AssertEq(nil, encoder.Encode(&t.cache))

	// Decode
	decoder := gob.NewDecoder(buf)
	var decoded invariantsCache
	AssertEq(nil, decoder.Decode(&decoded))

	// Insert another.
	decoded.Insert("queso", 29)

	// See what's left.
	ExpectEq(nil, decoded.LookUp("taco"))
	ExpectEq(17, decoded.LookUp("burrito"))
	ExpectThat(t.cache.LookUp("enchilada"), DeepEquals([]byte{0x23, 0x29}))
	ExpectEq(29, decoded.LookUp("queso"))
}
