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
	"time"

	"github.com/jacobsa/util/lrucache"
)

type cacheEntry struct {
	expiry    time.Time
	inodeType Type
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
	entries lrucache.Cache
}

// Create a cache whose information expires with the supplied TTL. If the TTL
// is zero, nothing will ever be cached.
func newTypeCache(perTypeCapacity int, ttl time.Duration) typeCache {
	return typeCache{
		ttl:     ttl,
		entries: lrucache.New(perTypeCapacity),
	}
}

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

// Panic if any internal invariants have been violated. The careful user can
// arrange to call this at crucial moments.
func (tc *typeCache) CheckInvariants() {
	tc.entries.CheckInvariants()
}

// Insert inserts a record to the cache.
func (tc *typeCache) Insert(now time.Time, name string, it Type) {
	// Are we disabled?
	if tc.ttl == 0 {
		return
	}
	tc.entries.Insert(name, cacheEntry{
		expiry:    now.Add(tc.ttl),
		inodeType: it,
	})
}

// Erase erases all information about the supplied name.
func (tc *typeCache) Erase(name string) {
	tc.entries.Erase(name)
}

// Get gets the record for the given name.
func (tc *typeCache) Get(now time.Time, name string) Type {
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
