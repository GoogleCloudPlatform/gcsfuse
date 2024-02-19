// Copyright 2015 Google Inc. All Rights Reserved.
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
	"math"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/internal/util"
)

type Type int

const (
	UnknownType     Type = 0
	SymlinkType     Type = 1
	RegularFileType Type = 2
	ExplicitDirType Type = 3
	ImplicitDirType Type = 4
	NonexistentType Type = 5
)

// TypeCache is a (name -> Type) map.
// It maintains TTL for each entry for supporting
// TTL-based expiration.
// Sample usage:
//
//	tc := NewTypeCache(size, ttl)
//	tc.Insert(time.Now(), "file", RegularFileType)
//	tc.Insert(time.Now(), "dir", ExplicitDirType)
//	tc.Get(time.Now(),"file") -> RegularFileType
//	tc.Get(time.Now(),"dir") -> ExplicitDirType
//	tc.Get(time.Now()+ttl+1ns, "file") -> internally tc.Erase("file") -> UnknownType
//	tc.Erase("dir")
//	tc.Get(time.Now(),"dir") -> UnknownType
type TypeCache interface {
	// Insert inserts the given entry (name -> type)
	// with the entry-expiration at now+ttl.
	Insert(now time.Time, name string, it Type)
	// Erase removes the entry with the given name.
	Erase(name string)
	// Get returns the entry with given name, and also
	// records this entry as latest accessed in the cache.
	// If now > expiration, then entry is removed from cache, and
	// UnknownType is returned.
	// If entry doesn't exist in the cache, then
	// UnknownType is returned.
	Get(now time.Time, name string) Type
}

type cacheEntry struct {
	expiry    time.Time
	inodeType Type
	// Copy of key string (internally only
	// points to the original
	// string data, so size overhead is a fixed 16 bytes, i.e. O(1),
	// irrespective of actual key value.
	// This is needed to calculate the
	// accurate size of the type-cache entry on heap.
	key string
}

// Size returns the size of cacheEntry on RSS (Resident Set Size), which is
// approximated as twice as calculated heap-size.
// The calculated heap-size for each entry is
// 80-bytes (24 for expiry, 8 for inodeType, 3*16 for 3 copies of
// string structure for the key ) + length of the key itself,
// for each entry (including negative ones).
// This size is specific
// to 64-bit linux machines, and might differ on other platforms.
func (ce cacheEntry) Size() (size uint64) {
	// First calculate size on heap.
	// Additional 2*util.UnsafeSizeOf(&e.key) is to account for the 2 other copies of string
	// struct stored in the cache map and in the cache linked-list.
	size = uint64(util.UnsafeSizeOf(&ce) + 2*util.UnsafeSizeOf(&ce.key) + len(ce.key))

	// Convert heap-size to rss (resident set size).
	size = uint64(math.Ceil(util.HeapSizeToRssConversionFactor * float64(size)))

	return
}

// Return the size (rss) of a type-cache entry
// for a given key-string.
func SizeOfTypeCacheEntry(key string) uint64 {
	ce := cacheEntry{}
	ce.key = key
	return ce.Size()
}

// A cache that maps from a name to information about the type of the object
// with that name. Each name N is in one of the following states:
//
//   - Nothing is known about N.
//   - We have recorded that N is a file.
//   - We have recorded that N is a directory.
//   - We have recorded that N is both a file and a directory.
//
// Must be created with NewTypeCache. May be contained in a larger struct.
// External synchronization is required.
type typeCache struct {
	/////////////////////////
	// Constant data
	/////////////////////////

	ttl time.Duration

	/////////////////////////
	// Mutable state
	/////////////////////////

	// A cache mapping names to the cache entry.
	// INVARIANT: entries.CheckInvariants() does not panic
	// INVARIANT: Each value is of type cacheEntry
	entries *lru.Cache
}

// NewTypeCache creates an LRU-policy-based cache with given parameters.
// Any entry whose TTL has expired, is removed from the cache on next access (Get).
// When insertion of next entry would cause size of cache > maxSizeMB,
// older entries are evicted according to the LRU-policy.
// If either of TTL or maxSizeMB is zero, nothing is ever cached.
func NewTypeCache(maxSizeMB int, ttl time.Duration) TypeCache {
	if ttl > 0 && maxSizeMB != 0 {
		var lruSizeInBytesToUse uint64 = math.MaxUint64 // default for when maxSizeMB = -1
		if maxSizeMB > 0 {
			lruSizeInBytesToUse = util.MiBsToBytes(uint64(maxSizeMB))
		}
		return &typeCache{
			ttl:     ttl,
			entries: lru.NewCache(lruSizeInBytesToUse),
		}
	}
	return &typeCache{}
}

func (tc *typeCache) Insert(now time.Time, name string, it Type) {
	if tc.entries != nil { // only if caching is enabled
		_, err := tc.entries.Insert(name, cacheEntry{
			expiry:    now.Add(tc.ttl),
			inodeType: it,
			key:       name,
		})
		if err != nil {
			panic(fmt.Errorf("failed to insert entry in typeCache: %v", err))
		}
	}
}

func (tc *typeCache) Erase(name string) {
	if tc.entries != nil { // only if caching is enabled
		tc.entries.Erase(name)
	}
}

func (tc *typeCache) Get(now time.Time, name string) Type {
	if tc.entries == nil { // if caching is not enabled
		return UnknownType
	}

	val := tc.entries.LookUp(name)
	if val == nil {
		return UnknownType
	}

	entry := val.(cacheEntry)
	// Has the entry expired?
	if entry.expiry.Before(now) {
		tc.entries.Erase(name)
		return UnknownType
	}
	return entry.inodeType
}
