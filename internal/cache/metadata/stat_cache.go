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

package metadata

import (
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/gcs"
)

// A cache mapping from name to most recent known record for the object of that
// name. External synchronization must be provided.
type StatCache interface {
	// Insert an entry for the given object record.
	//
	// In order to help cope with caching of arbitrarily out of date (i.e.
	// inconsistent) object listings, entry will not replace any positive entry
	// with a newer generation number, or with an equivalent generation number
	// but newer metadata generation number. We have no choice, however, but to
	// replace negative entries.
	//
	// The entry will expire after the supplied time.
	Insert(o *gcs.Object, expiration time.Time)

	// Set up a negative entry for the given name, indicating that the name
	// doesn't exist. Overwrite any existing entry for the name, positive or
	// negative.
	AddNegativeEntry(name string, expiration time.Time)

	// Erase the entry for the given object name, if any.
	Erase(name string)

	// Return the current entry for the given name, or nil if there is a negative
	// entry. Return hit == false when there is neither a positive nor a negative
	// entry, or the entry has expired according to the supplied current time.
	LookUp(name string, now time.Time) (hit bool, o *gcs.Object)
}

// Create a new bucket-view to the passed shared-cache object.
// For dynamic-mount (mount for multiple buckets), pass bn as bucket-name.
// For static-mout (mount for single bucket), pass bn as "".
func NewStatCacheBucketView(sc *lru.Cache, bn string) StatCache {
	return &statCacheBucketView{
		sharedCache: sc,
		bucketName:  bn,
	}
}

// statCacheBucketView is a special type of StatCache which
// shares its underlying cache map object with other
// statCacheBucketView objects (for dynamically mounts) through
// a specific bucket-name. It does so by prepending its
// bucket-name to its entry keys to make them unique
// to it.
type statCacheBucketView struct {
	sharedCache *lru.Cache
	// bucketName is the unique identifier for this
	// statCache object among all statCache objects
	// using the same shared lru.Cache object.
	// It can be empty ("").
	bucketName string
}

// An entry in the cache, pairing an object with the expiration time for the
// entry. Nil object means negative entry.
type entry struct {
	o          *gcs.Object
	expiration time.Time
}

// Size returns the size of entry.
// It is currently set to dummy value 1 to avoid
// the unnecessary actual size calculation.
func (e entry) Size() uint64 {
	return 1
}

// Should the supplied object for a new positive entry replace the given
// existing entry?
func shouldReplace(o *gcs.Object, existing entry) bool {
	// Negative entries should always be replaced with positive entries.
	if existing.o == nil {
		return true
	}

	// Compare first on generation.
	if o.Generation != existing.o.Generation {
		return o.Generation > existing.o.Generation
	}

	// Break ties on metadata generation.
	if o.MetaGeneration != existing.o.MetaGeneration {
		return o.MetaGeneration > existing.o.MetaGeneration
	}

	// Break ties by preferring fresher entries.
	return true
}

func (sc *statCacheBucketView) key(objectName string) string {
	// path.Join(sc.bucketName, objectName) does not work
	// because that normalizes the trailing "/"
	// which breaks functionality by removing
	// differentiation between files and directories.
	if sc.bucketName != "" {
		return sc.bucketName + "/" + objectName
	}
	return objectName
}

func (sc *statCacheBucketView) Insert(o *gcs.Object, expiration time.Time) {
	name := sc.key(o.Name)

	// Is there already a better entry?
	if existing := sc.sharedCache.LookUp(name); existing != nil {
		if !shouldReplace(o, existing.(entry)) {
			return
		}
	}

	// Insert an entry.
	e := entry{
		o:          o,
		expiration: expiration,
	}

	if _, err := sc.sharedCache.Insert(name, e); err != nil {
		panic(err)
	}
}

func (sc *statCacheBucketView) AddNegativeEntry(objectName string, expiration time.Time) {
	// Insert a negative entry.
	e := entry{
		o:          nil,
		expiration: expiration,
	}

	name := sc.key(objectName)
	if _, err := sc.sharedCache.Insert(name, e); err != nil {
		panic(err)
	}
}

func (sc *statCacheBucketView) Erase(objectName string) {
	name := sc.key(objectName)
	sc.sharedCache.Erase(name)
}

func (sc *statCacheBucketView) LookUp(
	objectName string,
	now time.Time) (hit bool, o *gcs.Object) {
	// Look up in the LRU cache.
	value := sc.sharedCache.LookUp(sc.key(objectName))
	if value == nil {
		return
	}

	e := value.(entry)

	// Has this entry expired?
	if e.expiration.Before(now) {
		sc.Erase(objectName)
		return
	}

	hit = true
	o = e.o

	return
}
