// Copyright 2024 Google LLC
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

	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
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

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type TypeCacheTest struct {
	suite.Suite
	cache *typeCache
	ttl   time.Duration
}

type ZeroSizeTypeCacheTest struct {
	suite.Suite
	cache *typeCache
	ttl   time.Duration
}

type ZeroTtlTypeCacheTest struct {
	suite.Suite
	cache *typeCache
}

func TestTypeCacheTestSuite(t *testing.T) {
	suite.Run(t, new(TypeCacheTest))
}

func TestZeroSizeTypeCacheTestSuite(t *testing.T) {
	suite.Run(t, new(ZeroSizeTypeCacheTest))
}

func TestZeroTtlTypeCacheTestSuite(t *testing.T) {
	suite.Run(t, new(ZeroTtlTypeCacheTest))
}

func (t *TypeCacheTest) SetupTest() {
	t.ttl = TTL
	t.cache = createNewTypeCache(t.T(), TypeCacheMaxSizeMB, t.ttl)
	if t.cache.IsDeprecated() {
		t.T().Skip("Skipping TypeCacheTest because EnableTypeCacheDeprecation is true")
	}
}

func (t *ZeroSizeTypeCacheTest) SetupTest() {
	t.ttl = TTL
	t.cache = createNewTypeCache(t.T(), 0, t.ttl)
	if t.cache.IsDeprecated() {
		t.T().Skip("Skipping ZeroSizeTypeCacheTest because EnableTypeCacheDeprecation is true")
	}
}

func (t *ZeroTtlTypeCacheTest) SetupTest() {
	t.cache = createNewTypeCache(t.T(), TypeCacheMaxSizeMB, 0)
	if t.cache.IsDeprecated() {
		t.T().Skip("Skipping ZeroTtlTypeCacheTest because EnableTypeCacheDeprecation is true")
	}
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func createNewTypeCache(t *testing.T, maxSizeMB int64, ttl time.Duration) *typeCache {
	tc := NewTypeCache(maxSizeMB, ttl, false)

	assert.NotNil(t, tc)
	assert.NotNil(t, tc.(*typeCache))

	return tc.(*typeCache)
}

////////////////////////////////////////////////////////////////////////
// Tests for regular TypeCache - TypeCacheTest
////////////////////////////////////////////////////////////////////////

func (t *TypeCacheTest) TestNewTypeCache() {
	input := []struct {
		maxSizeMB          int64
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
		tc := createNewTypeCache(t.T(), input.maxSizeMB, input.ttl)
		t.Equal(input.entriesShouldBeNil, tc.entries == nil)
	}
}

func (t *TypeCacheTest) TestGetFromEmptyTypeCache() {
	t.Equal(UnknownType, t.cache.Get(now, "abc"))
}

func (t *TypeCacheTest) TestGetUninsertedEntry() {
	t.cache.Insert(now, "abcd", RegularFileType)

	t.Equal(UnknownType, t.cache.Get(beforeExpiration, "abc"))
}

func (t *TypeCacheTest) TestGetOverwrittenEntry() {
	t.cache.Insert(now, "abcd", RegularFileType)
	t.cache.Insert(now, "abcd", ExplicitDirType)

	t.Equal(ExplicitDirType, t.cache.Get(beforeExpiration, "abcd"))
}

func (t *TypeCacheTest) TestGetBeforeTtlExpiration() {
	t.cache.Insert(now, "abcd", RegularFileType)

	t.Equal(RegularFileType, t.cache.Get(beforeExpiration, "abcd"))
}

func (t *TypeCacheTest) TestGetAfterTtlExpiration() {
	t.cache.Insert(now, "abcd", RegularFileType)

	t.Equal(UnknownType, t.cache.Get(afterExpiration, "abcd"))
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
	t.Equal(RegularFileType, t.cache.Get(beforeExpiration, nameOfIthFile(entriesToBeInserted-1)))

	// The first inserted entry should have been evicted by all the later insertions.
	t.Equal(UnknownType, t.cache.Get(beforeExpiration, nameOfIthFile(0)))

	// The second entry should not have been evicted
	t.Equal(RegularFileType, t.cache.Get(beforeExpiration, nameOfIthFile(1)))
}

func (t *TypeCacheTest) TestGetErasedEntry() {
	t.cache.Insert(now, "abcd", RegularFileType)
	t.cache.Erase("abcd")

	t.Equal(UnknownType, t.cache.Get(beforeExpiration, "abcd"))
}

func (t *TypeCacheTest) TestGetReinsertedEntry() {
	t.cache.Insert(now, "abcd", RegularFileType)
	t.cache.Erase("abcd")
	t.cache.Insert(now2, "abcd", ExplicitDirType)

	t.Equal(ExplicitDirType, t.cache.Get(beforeExpiration2, "abcd"))
}

////////////////////////////////////////////////////////////////////////
// Tests for TypeCache created with size=0 - ZeroSizeTypeCacheTest
////////////////////////////////////////////////////////////////////////

func (t *ZeroSizeTypeCacheTest) TestGetFromEmptyTypeCache() {
	t.Equal(UnknownType, t.cache.Get(now, "abc"))
}

func (t *ZeroSizeTypeCacheTest) TestGetInsertedEntry() {
	t.cache.Insert(now, "abcd", RegularFileType)

	t.Equal(UnknownType, t.cache.Get(beforeExpiration, "abcd"))
}

////////////////////////////////////////////////////////////////////////
// Tests for TypeCache created with ttl=0 - ZeroTtlTypeCacheTest
////////////////////////////////////////////////////////////////////////

func (t *ZeroTtlTypeCacheTest) TestGetFromEmptyTypeCache() {
	t.Equal(UnknownType, t.cache.Get(now, "abc"))
}

func (t *ZeroTtlTypeCacheTest) TestGetInsertedEntry() {
	t.cache.Insert(now, "abcd", RegularFileType)

	t.Equal(UnknownType, t.cache.Get(beforeExpiration, "abcd"))
}
