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

package inode

import (
	"fmt"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/util"
	. "github.com/jacobsa/ogletest"
)

const (
	TTL             time.Duration = time.Millisecond
	TypeCacheSizeMb               = 1
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

func init() {
	RegisterTestSuite(&TypeCacheTest{})
	RegisterTestSuite(&ZeroSizeTypeCacheTest{})
	RegisterTestSuite(&ZeroTtlTypeCacheTest{})
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

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func createNewTypeCache(sizeInMB int, ttl time.Duration) *typeCache {
	tc := newTypeCache(sizeInMB, ttl)
	AssertNe(nil, &tc)
	return &tc
}

////////////////////////////////////////////////////////////////////////
// Tests for regulat TypeCache - TypeCacheTest
////////////////////////////////////////////////////////////////////////

func (t *TypeCacheTest) TestNewTypeCache() {
	input := []struct {
		sizeInMb           int
		ttl                time.Duration
		entriesShouldBeNil bool
	}{
		{
			sizeInMb:           0,
			ttl:                time.Second,
			entriesShouldBeNil: true,
		},
		{
			sizeInMb:           1,
			ttl:                0,
			entriesShouldBeNil: true,
		},
		{
			sizeInMb: -1,
			ttl:      time.Second,
		},
		{
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

	// The first inserted entry should have been evicted by all the later insertions.
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
