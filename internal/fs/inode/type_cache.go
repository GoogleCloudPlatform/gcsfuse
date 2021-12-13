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

// A cache that maps from a name to information about the type of the object
// with that name. Each name N is in one of the following states:
//
//  *  Nothing is known about N.
//  *  We have recorded that N is a file.
//  *  We have recorded that N is a directory.
//  *  We have recorded that N is both a file and a directory.
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

	// A cache mapping file names to the time at which the entry should expire.
	//
	// INVARIANT: files.CheckInvariants() does not panic
	// INVARIANT: Each value is of type time.Time
	files lrucache.Cache

	// A cache mapping directory names to the time at which the entry should
	// expire.
	//
	// INVARIANT: dirs.CheckInvariants() does not panic
	// INVARIANT: Each value is of type time.Time
	dirs lrucache.Cache

	// A cache mapping implicit directory names to the time at which the entry
	// should expire.
	//
	// INVARIANT: implicitDirs.CheckInvariants() does not panic
	// INVARIANT: Each value is of type time.Time
	// INVARIANT: must be a subset of dirs
	implicitDirs lrucache.Cache
}

// Create a cache whose information expires with the supplied TTL. If the TTL
// is zero, nothing will ever be cached.
func newTypeCache(
	perTypeCapacity int,
	ttl time.Duration) (tc typeCache) {
	tc = typeCache{
		ttl:          ttl,
		files:        lrucache.New(perTypeCapacity),
		dirs:         lrucache.New(perTypeCapacity),
		implicitDirs: lrucache.New(perTypeCapacity),
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

// Panic if any internal invariants have been violated. The careful user can
// arrange to call this at crucial moments.
func (tc *typeCache) CheckInvariants() {
	// INVARIANT: files.CheckInvariants() does not panic
	tc.files.CheckInvariants()

	// INVARIANT: dirs.CheckInvariants() does not panic
	tc.dirs.CheckInvariants()

	// INVARIANT: dirs.CheckInvariants() does not panic
	tc.implicitDirs.CheckInvariants()
}

// Record that the supplied name is a file. It may still also be a directory.
func (tc *typeCache) NoteFile(now time.Time, name string) {
	// Are we disabled?
	if tc.ttl == 0 {
		return
	}

	tc.files.Insert(name, now.Add(tc.ttl))
}

// Record that the supplied name is a directory. It may still also be a file.
func (tc *typeCache) NoteDir(now time.Time, name string) {
	// Are we disabled?
	if tc.ttl == 0 {
		return
	}

	tc.dirs.Insert(name, now.Add(tc.ttl))
}

// Record that the supplied name is an implicit directory. It may still also be
// a file, but must not be a regular directory.
func (tc *typeCache) NoteImplicitDir(now time.Time, name string) {
	// Are we disabled?
	if tc.ttl == 0 {
		return
	}

	tc.implicitDirs.Insert(name, now.Add(tc.ttl))
}

// Erase all information about the supplied name.
func (tc *typeCache) Erase(name string) {
	tc.files.Erase(name)
	tc.dirs.Erase(name)
	tc.implicitDirs.Erase(name)
}

// Do we currently think the given name is a file?
func (tc *typeCache) IsFile(now time.Time, name string) (res bool) {
	// Is there an entry?
	val := tc.files.LookUp(name)
	if val == nil {
		res = false
		return
	}

	expiration := val.(time.Time)

	// Has the entry expired?
	if expiration.Before(now) {
		tc.files.Erase(name)
		res = false
		return
	}

	res = true
	return
}

// Do we currently think the given name is a directory?
func (tc *typeCache) IsDir(now time.Time, name string) (res bool) {
	// Is there an entry?
	val := tc.dirs.LookUp(name)
	if val == nil {
		res = false
		return
	}

	expiration := val.(time.Time)

	// Has the entry expired?
	if expiration.Before(now) {
		tc.dirs.Erase(name)
		res = false
		return
	}

	res = true
	return
}

// Do we currently think the given name is an implicit directory?
func (tc *typeCache) IsImplicitDir(now time.Time, name string) (res bool) {
	// Is there an entry?
	val := tc.implicitDirs.LookUp(name)
	if val == nil {
		res = false
		return
	}

	expiration := val.(time.Time)

	// Has the entry expired?
	if expiration.Before(now) {
		tc.implicitDirs.Erase(name)
		res = false
		return
	}

	res = true
	return
}
