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

package caching_test

import (
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/storage/caching"
	. "github.com/jacobsa/ogletest"
)

func TestStatCache(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Invariant-checking cache
////////////////////////////////////////////////////////////////////////

type invariantsCache struct {
	wrapped caching.StatCache
}

func (c *invariantsCache) Insert(
	o *object.Object,
	expiration time.Time) {
	c.wrapped.CheckInvariants()
	defer c.wrapped.CheckInvariants()

	c.wrapped.Insert(o, expiration)
}

func (c *invariantsCache) AddNegativeEntry(
	name string,
	expiration time.Time) {
	c.wrapped.CheckInvariants()
	defer c.wrapped.CheckInvariants()

	c.wrapped.AddNegativeEntry(name, expiration)
}

func (c *invariantsCache) Erase(name string) {
	c.wrapped.CheckInvariants()
	defer c.wrapped.CheckInvariants()

	c.wrapped.Erase(name)
}

func (c *invariantsCache) LookUp(
	name string,
	now time.Time) (hit bool, o *object.Object) {
	c.wrapped.CheckInvariants()
	defer c.wrapped.CheckInvariants()

	hit, o = c.wrapped.LookUp(name, now)
	return
}

func (c *invariantsCache) LookUpOrNil(
	name string,
	now time.Time) (o *object.Object) {
	_, o = c.LookUp(name, now)
	return
}

func (c *invariantsCache) Hit(
	name string,
	now time.Time) (hit bool) {
	hit, _ = c.LookUp(name, now)
	return
}

func (c *invariantsCache) NegativeEntry(
	name string,
	now time.Time) (negative bool) {
	hit, o := c.LookUp(name, now)
	negative = hit && o == nil
	return
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const capacity = 3

var someTime = time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local)
var expiration = someTime.Add(time.Second)

type StatCacheTest struct {
	cache invariantsCache
}

func init() { RegisterTestSuite(&StatCacheTest{}) }

func (t *StatCacheTest) SetUp(ti *TestInfo) {
	t.cache.wrapped = caching.NewStatCache(capacity)
}

////////////////////////////////////////////////////////////////////////
// Test functions
////////////////////////////////////////////////////////////////////////

func (t *StatCacheTest) LookUpInEmptyCache() {
	ExpectFalse(t.cache.Hit("", someTime))
	ExpectFalse(t.cache.Hit("taco", someTime))
}

func (t *StatCacheTest) LookUpUnknownKey() {
	o0 := &object.Object{Name: "burrito"}
	o1 := &object.Object{Name: "taco"}

	t.cache.Insert(o0, someTime.Add(time.Second))
	t.cache.Insert(o1, someTime.Add(time.Second))

	ExpectFalse(t.cache.Hit("", someTime))
	ExpectFalse(t.cache.Hit("enchilada", someTime))
}

func (t *StatCacheTest) KeysPresentButEverythingIsExpired() {
	o0 := &object.Object{Name: "burrito"}
	o1 := &object.Object{Name: "taco"}

	t.cache.Insert(o0, someTime.Add(-time.Second))
	t.cache.Insert(o1, someTime.Add(-time.Second))

	ExpectFalse(t.cache.Hit("burrito", someTime))
	ExpectFalse(t.cache.Hit("taco", someTime))
}

func (t *StatCacheTest) FillUpToCapacity() {
	AssertEq(3, capacity)

	o0 := &object.Object{Name: "burrito"}
	o1 := &object.Object{Name: "taco"}

	t.cache.Insert(o0, expiration)
	t.cache.Insert(o1, expiration)
	t.cache.AddNegativeEntry("enchilada", expiration)

	// Before expiration
	justBefore := expiration.Add(-time.Nanosecond)
	ExpectEq(o0, t.cache.LookUpOrNil("burrito", justBefore))
	ExpectEq(o1, t.cache.LookUpOrNil("taco", justBefore))
	ExpectTrue(t.cache.NegativeEntry("enchilada", justBefore))

	// At expiration
	ExpectEq(o0, t.cache.LookUpOrNil("burrito", expiration))
	ExpectEq(o1, t.cache.LookUpOrNil("taco", expiration))
	ExpectTrue(t.cache.NegativeEntry("enchilada", justBefore))

	// After expiration
	justAfter := expiration.Add(time.Nanosecond)
	ExpectFalse(t.cache.Hit("burrito", justAfter))
	ExpectFalse(t.cache.Hit("taco", justAfter))
	ExpectFalse(t.cache.Hit("enchilada", justAfter))
}

func (t *StatCacheTest) ExpiresLeastRecentlyUsed() {
	AssertEq(3, capacity)

	o0 := &object.Object{Name: "burrito"}
	o1 := &object.Object{Name: "taco"}

	t.cache.Insert(o0, expiration)
	t.cache.Insert(o1, expiration)                         // Least recent
	t.cache.AddNegativeEntry("enchilada", expiration)      // Second most recent
	AssertEq(o0, t.cache.LookUpOrNil("burrito", someTime)) // Most recent

	// Insert another.
	o3 := &object.Object{Name: "queso"}
	t.cache.Insert(o3, expiration)

	// See what's left.
	ExpectFalse(t.cache.Hit("taco", someTime))
	ExpectEq(o0, t.cache.LookUpOrNil("burrito", someTime))
	ExpectTrue(t.cache.NegativeEntry("enchilada", someTime))
	ExpectEq(o3, t.cache.LookUpOrNil("queso", someTime))
}

func (t *StatCacheTest) Overwrite_NewerGeneration() {
	o0 := &object.Object{Name: "taco", Generation: 17, MetaGeneration: 5}
	o1 := &object.Object{Name: "taco", Generation: 19, MetaGeneration: 1}

	t.cache.Insert(o0, expiration)
	t.cache.Insert(o1, expiration)

	ExpectEq(o1, t.cache.LookUpOrNil("taco", someTime))

	// The overwritten entry shouldn't count toward capacity.
	AssertEq(3, capacity)

	t.cache.Insert(&object.Object{Name: "burrito"}, expiration)
	t.cache.Insert(&object.Object{Name: "enchilada"}, expiration)

	ExpectNe(nil, t.cache.LookUpOrNil("taco", someTime))
	ExpectNe(nil, t.cache.LookUpOrNil("burrito", someTime))
	ExpectNe(nil, t.cache.LookUpOrNil("enchilada", someTime))
}

func (t *StatCacheTest) Overwrite_SameGeneration_NewerMetadataGen() {
	o0 := &object.Object{Name: "taco", Generation: 17, MetaGeneration: 5}
	o1 := &object.Object{Name: "taco", Generation: 17, MetaGeneration: 7}

	t.cache.Insert(o0, expiration)
	t.cache.Insert(o1, expiration)

	ExpectEq(o1, t.cache.LookUpOrNil("taco", someTime))

	// The overwritten entry shouldn't count toward capacity.
	AssertEq(3, capacity)

	t.cache.Insert(&object.Object{Name: "burrito"}, expiration)
	t.cache.Insert(&object.Object{Name: "enchilada"}, expiration)

	ExpectNe(nil, t.cache.LookUpOrNil("taco", someTime))
	ExpectNe(nil, t.cache.LookUpOrNil("burrito", someTime))
	ExpectNe(nil, t.cache.LookUpOrNil("enchilada", someTime))
}

func (t *StatCacheTest) Overwrite_SameGeneration_SameMetadataGen() {
	o0 := &object.Object{Name: "taco", Generation: 17, MetaGeneration: 5}
	o1 := &object.Object{Name: "taco", Generation: 17, MetaGeneration: 5}

	t.cache.Insert(o0, expiration)
	t.cache.Insert(o1, expiration)

	ExpectEq(o1, t.cache.LookUpOrNil("taco", someTime))
}

func (t *StatCacheTest) Overwrite_SameGeneration_OlderMetadataGen() {
	o0 := &object.Object{Name: "taco", Generation: 17, MetaGeneration: 5}
	o1 := &object.Object{Name: "taco", Generation: 17, MetaGeneration: 3}

	t.cache.Insert(o0, expiration)
	t.cache.Insert(o1, expiration)

	ExpectEq(o0, t.cache.LookUpOrNil("taco", someTime))
}

func (t *StatCacheTest) Overwrite_OlderGeneration() {
	o0 := &object.Object{Name: "taco", Generation: 17, MetaGeneration: 5}
	o1 := &object.Object{Name: "taco", Generation: 13, MetaGeneration: 7}

	t.cache.Insert(o0, expiration)
	t.cache.Insert(o1, expiration)

	ExpectEq(o0, t.cache.LookUpOrNil("taco", someTime))
}

func (t *StatCacheTest) Overwrite_NegativeWithPositive() {
	const name = "taco"
	o1 := &object.Object{Name: name, Generation: 13, MetaGeneration: 7}

	t.cache.AddNegativeEntry(name, expiration)
	t.cache.Insert(o1, expiration)

	ExpectEq(o1, t.cache.LookUpOrNil(name, someTime))
}

func (t *StatCacheTest) Overwrite_PositiveWithNegative() {
	const name = "taco"
	o0 := &object.Object{Name: name, Generation: 13, MetaGeneration: 7}

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
