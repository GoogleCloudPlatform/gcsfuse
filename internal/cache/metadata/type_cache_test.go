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
	"github.com/stretchr/testify/require"
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
// Helpers
////////////////////////////////////////////////////////////////////////

func createNewTypeCache(t *testing.T, maxSizeMB int64, ttl time.Duration) *typeCache {
	tc := NewTypeCache(maxSizeMB, ttl)

	require.NotNil(t, tc)
	tcImpl, ok := tc.(*typeCache)
	require.True(t, ok)

	return tcImpl
}

////////////////////////////////////////////////////////////////////////
// Tests for regular TypeCache
////////////////////////////////////////////////////////////////////////

func TestTypeCache_New(t *testing.T) {
	inputs := []struct {
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
		},
	}

	for _, input := range inputs {
		tc := createNewTypeCache(t, input.maxSizeMB, input.ttl)
		if input.entriesShouldBeNil {
			assert.Nil(t, tc.entries)
		} else {
			assert.NotNil(t, tc.entries)
		}
	}
}

func TestTypeCache_GetFromEmpty(t *testing.T) {
	cache := createNewTypeCache(t, TypeCacheMaxSizeMB, TTL)
	assert.Equal(t, UnknownType, cache.Get(now, "abc"))
}

func TestTypeCache_GetUninserted(t *testing.T) {
	cache := createNewTypeCache(t, TypeCacheMaxSizeMB, TTL)
	cache.Insert(now, "abcd", RegularFileType)

	assert.Equal(t, UnknownType, cache.Get(beforeExpiration, "abc"))
}

func TestTypeCache_GetOverwritten(t *testing.T) {
	cache := createNewTypeCache(t, TypeCacheMaxSizeMB, TTL)
	cache.Insert(now, "abcd", RegularFileType)
	cache.Insert(now, "abcd", ExplicitDirType)

	assert.Equal(t, ExplicitDirType, cache.Get(beforeExpiration, "abcd"))
}

func TestTypeCache_GetBeforeTtlExpiration(t *testing.T) {
	cache := createNewTypeCache(t, TypeCacheMaxSizeMB, TTL)
	cache.Insert(now, "abcd", RegularFileType)

	assert.Equal(t, RegularFileType, cache.Get(beforeExpiration, "abcd"))
}

func TestTypeCache_GetAfterTtlExpiration(t *testing.T) {
	cache := createNewTypeCache(t, TypeCacheMaxSizeMB, TTL)
	cache.Insert(now, "abcd", RegularFileType)

	assert.Equal(t, UnknownType, cache.Get(afterExpiration, "abcd"))
}

func TestTypeCache_GetAfterSizeExpiration(t *testing.T) {
	cache := createNewTypeCache(t, TypeCacheMaxSizeMB, TTL)
	sizePerEntry := cacheEntry{key: "abcde"}.Size()
	entriesToBeInserted := int(util.MiBsToBytes(TypeCacheMaxSizeMB) / sizePerEntry)
	nameOfIthFile := func(i int) string {
		return fmt.Sprintf("%05d", i)
	}

	// adding 1 entry more than can be fit in the cache.
	for i := 0; i <= entriesToBeInserted; i++ {
		cache.Insert(now, nameOfIthFile(i), RegularFileType)
	}

	// Verify that Get works, by accessing the last entry inserted.
	assert.Equal(t, RegularFileType, cache.Get(beforeExpiration, nameOfIthFile(entriesToBeInserted-1)))

	// The first inserted entry should have been evicted by all the later insertions.
	assert.Equal(t, UnknownType, cache.Get(beforeExpiration, nameOfIthFile(0)))

	// The second entry should not have been evicted
	assert.Equal(t, RegularFileType, cache.Get(beforeExpiration, nameOfIthFile(1)))
}

func TestTypeCache_GetErased(t *testing.T) {
	cache := createNewTypeCache(t, TypeCacheMaxSizeMB, TTL)
	cache.Insert(now, "abcd", RegularFileType)
	cache.Erase("abcd")

	assert.Equal(t, UnknownType, cache.Get(beforeExpiration, "abcd"))
}

func TestTypeCache_GetReinserted(t *testing.T) {
	cache := createNewTypeCache(t, TypeCacheMaxSizeMB, TTL)
	cache.Insert(now, "abcd", RegularFileType)
	cache.Erase("abcd")
	cache.Insert(now2, "abcd", ExplicitDirType)

	assert.Equal(t, ExplicitDirType, cache.Get(beforeExpiration2, "abcd"))
}

////////////////////////////////////////////////////////////////////////
// Tests for TypeCache created with size=0
////////////////////////////////////////////////////////////////////////

func TestZeroSizeTypeCache_GetFromEmpty(t *testing.T) {
	cache := createNewTypeCache(t, 0, TTL)
	assert.Equal(t, UnknownType, cache.Get(now, "abc"))
}

func TestZeroSizeTypeCache_GetInserted(t *testing.T) {
	cache := createNewTypeCache(t, 0, TTL)
	cache.Insert(now, "abcd", RegularFileType)

	assert.Equal(t, UnknownType, cache.Get(beforeExpiration, "abcd"))
}

////////////////////////////////////////////////////////////////////////
// Tests for TypeCache created with ttl=0
////////////////////////////////////////////////////////////////////////

func TestZeroTtlTypeCache_GetFromEmpty(t *testing.T) {
	cache := createNewTypeCache(t, TypeCacheMaxSizeMB, 0)
	assert.Equal(t, UnknownType, cache.Get(now, "abc"))
}

func TestZeroTtlTypeCache_GetInserted(t *testing.T) {
	cache := createNewTypeCache(t, TypeCacheMaxSizeMB, 0)
	cache.Insert(now, "abcd", RegularFileType)

	assert.Equal(t, UnknownType, cache.Get(beforeExpiration, "abcd"))
}
