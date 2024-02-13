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
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/util"
	. "github.com/jacobsa/ogletest"
)

const (
	TTL                time.Duration = time.Millisecond
	TypeCacheMaxSizeMB               = 1
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
	cache TypeCache
	ttl   time.Duration
}

type ZeroSizeTypeCacheTest struct {
	cache TypeCache
	ttl   time.Duration
}

type ZeroTtlTypeCacheTest struct {
	cache TypeCache
}

func init() {
	RegisterTestSuite(&TypeCacheTest{})
	RegisterTestSuite(&ZeroSizeTypeCacheTest{})
	RegisterTestSuite(&ZeroTtlTypeCacheTest{})
}

func (t *TypeCacheTest) SetUp(ti *TestInfo) {
	t.ttl = TTL
	t.cache = createNewTypeCache(TypeCacheMaxSizeMB, t.ttl)
}

func (t *ZeroSizeTypeCacheTest) SetUp(ti *TestInfo) {
	t.ttl = TTL
	t.cache = createNewTypeCache(0, t.ttl)
}

func (t *ZeroTtlTypeCacheTest) SetUp(ti *TestInfo) {
	t.cache = createNewTypeCache(TypeCacheMaxSizeMB, 0)
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func createNewTypeCache(maxSizeMB int, ttl time.Duration) *typeCache {
	tc := NewTypeCache(maxSizeMB, ttl)

	AssertNe(nil, tc)
	AssertNe(nil, tc.(*typeCache))

	return tc.(*typeCache)
}

////////////////////////////////////////////////////////////////////////
// Tests for regulat TypeCache - TypeCacheTest
////////////////////////////////////////////////////////////////////////

func (t *TypeCacheTest) TestNewTypeCache() {
	input := []struct {
		maxSizeMB          int
		ttl                time.Duration
		entriesShouldBeNil bool
	}{
		{
			maxSizeMB:          0,
			ttl:                time.Second,
			entriesShouldBeNil: true,
		},
		{
			maxSizeMB:          1,
			ttl:                0,
			entriesShouldBeNil: true,
		},
		{
			maxSizeMB: -1,
			ttl:       time.Second,
		},
		{
			maxSizeMB: 1,
			ttl:       time.Second,
		}}

	for _, input := range input {
		tc := createNewTypeCache(input.maxSizeMB, input.ttl)

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
	sizePerEntry := cacheEntry{key: "abcde"}.Size()
	entriesToBeInserted := int(util.MiBsToBytes(TypeCacheMaxSizeMB) / sizePerEntry)
	nameOfIthFile := func(i int) string {
		return fmt.Sprintf("%05d", i)
	}

	// adding 1 entry more than can be fit in the cache.
	for i := 0; i <= entriesToBeInserted; i++ {
		t.cache.Insert(now, nameOfIthFile(i), RegularFileType)
	}

	// Verify that Get works, by accessing the last entry inserted.
	ExpectEq(RegularFileType, t.cache.Get(beforeExpiration, nameOfIthFile(entriesToBeInserted-1)))

	// The first inserted entry should have been evicted by all the later insertions.
	ExpectEq(UnknownType, t.cache.Get(beforeExpiration, nameOfIthFile(0)))

	// The second entry should not have been evicted
	ExpectEq(RegularFileType, t.cache.Get(beforeExpiration, nameOfIthFile(1)))
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
