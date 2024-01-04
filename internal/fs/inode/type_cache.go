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

package inode

import (
	"fmt"
	"math"
	"time"
	unsafe "unsafe"

	"github.com/googlecloudplatform/gcsfuse/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/internal/util"
)

type cacheEntry struct {
	expiry    time.Time
	inodeType Type
}

func (ce cacheEntry) Size() uint64 {
	return uint64(unsafe.Sizeof(cacheEntry{}))
}

// A cache that maps from a name to information about the type of the object
// with that name. Each name N is in one of the following states:
//
//   - Nothing is known about N.
//   - We have recorded that N is a file.
//   - We have recorded that N is a directory.
//   - We have recorded that N is both a file and a directory.
//
// Must be created with newTypeCache. May be contained in a larger struct.
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
	//
	// INVARIANT: entries.CheckInvariants() does not panic
	// INVARIANT: Each value is of type cacheEntry
	entries *lru.Cache
}

// Create a cache whose information expires with the supplied TTL. If the TTL
// is zero, nothing will ever be cached.
func newTypeCache(sizeInMB int, ttl time.Duration) typeCache {
	if ttl > 0 && sizeInMB != 0 {
		if sizeInMB < -1 {
			panic("unhandled scenario: type-cache-max-size-mb-per-dir < -1")
		}
		var lruSizeInBytesToUse uint64 = math.MaxUint64 // default for when sizeInMb = -1, increasing
		if sizeInMB > 0 {
			lruSizeInBytesToUse = util.MiBsToBytes(uint64(sizeInMB))
		}
		return typeCache{
			ttl:     ttl,
			entries: lru.NewCache(lruSizeInBytesToUse),
		}
	}
	return typeCache{}
}

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

// Insert inserts a record to the cache.
func (tc *typeCache) Insert(now time.Time, name string, it Type) {
	if tc.entries != nil {
		_, err := tc.entries.Insert(name, cacheEntry{
			expiry:    now.Add(tc.ttl),
			inodeType: it,
		})
		if err != nil {
			panic(fmt.Errorf("failed to insert entry in typeCache: %v", err))
		}
		logger.Debugf("TypeCache: Inserted %s as %s", name, it.String())
	}
}

// Erase erases all information about the supplied name.
func (tc *typeCache) Erase(name string) {
	if tc.entries != nil {
		tc.entries.Erase(name)
		logger.Debugf("TypeCache: Erased entry for %s", name)
	}
}

// Get gets the record for the given name.
func (tc *typeCache) Get(now time.Time, name string) Type {
	if tc.entries == nil {
		return UnknownType
	}

	logger.Debugf("TypeCache: Fetching entry for %s ...", name)

	val := tc.entries.LookUp(name)
	if val == nil {
		logger.Debugf("                                     ... Not found!")
		return UnknownType
	}

	entry := val.(cacheEntry)

	logger.Debugf("                                     ... Found as %s", entry.inodeType.String())

	// Has the entry expired?
	if entry.expiry.Before(now) {
		logger.Debugf("TypeCache: Erasing entry for %s because of TTL expiration", name)
		tc.entries.Erase(name)
		return UnknownType
	}
	return entry.inodeType
}
