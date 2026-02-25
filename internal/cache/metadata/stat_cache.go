// Copyright 2023 Google LLC
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
	"math"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
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
	Insert(m *gcs.MinObject, expiration time.Time)

	InsertImplicitDir(objectName string, expiration time.Time)

	// Set up a negative entry for the given name, indicating that the name
	// doesn't exist. Overwrite any existing entry for the name, positive or
	// negative.
	AddNegativeEntry(name string, expiration time.Time)

	// Erase the entry for the given object name, if any.
	Erase(name string)

	// Return the current object entry for the given name, or nil if there is a negative
	// entry. Return hit == false when there is neither a positive nor a negative
	// entry, or the entry has expired according to the supplied current time.
	LookUp(name string, now time.Time) (hit bool, m *gcs.MinObject)

	// Insert an entry for the given folder resource.
	//
	// In order to help cope with caching of arbitrarily out of date (i.e.
	// inconsistent) object listings, entry will not replace any positive entry
	// with a newer meta generation number.
	//
	// The entry will expire after the supplied time.
	InsertFolder(f *gcs.Folder, expiration time.Time)

	// Return the current folder entry for the given name, or nil if there is a negative
	// entry. Return hit == false when there is neither a positive nor a negative
	// entry, or the entry has expired according to the supplied current time.
	LookUpFolder(folderName string, now time.Time) (bool, *gcs.Folder)

	// Set up a negative entry for the given folder name, indicating that the name
	// doesn't exist. Overwrite any existing entry for the name, positive or
	// negative.
	AddNegativeEntryForFolder(folderName string, expiration time.Time)

	// Invalidate cache for all the entries with given prefix
	// e.g. If cache contains below objects
	// a
	// a/b
	// a/d/c
	// d
	// Then it will invalidate entries a, a/b, a/d/c
	// Entry d will remain in cache.
	EraseEntriesWithGivenPrefix(prefix string)
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
	m          *gcs.MinObject
	f          *gcs.Folder
	expiration time.Time
	key        string
	// Set to true only for implicit directory entries. This flag will always remain false for negative entries and explicit objects.
	implicitDir bool
}

// Size returns the memory-size (resident set size) of the receiver entry.
// The size calculated by the unsafe.Sizeof calls, and
// NestedSizeOfGcsMinObject etc. does not account for
// hidden members in data structures like maps, slices, linked-lists etc.
// To account for those, we are adding a fixed constant of 515 bytes (deduced from
// benchmark runs) to heap-size per positive stat-cache entry
// to calculate a size closer to the actual memory utilization.
func (e entry) Size() (size uint64) {
	// First, calculate size on heap (including folder size also in case of hns buckets, in case of non-hns buckets 0 will be added as e.f will be Nil ).
	// Additional 2*util.UnsafeSizeOf(&e.key) is to account for the copies of string
	// struct stored in the cache map and in the cache linked-list.
	size = uint64(util.UnsafeSizeOf(&e) + len(e.key) + 2*util.UnsafeSizeOf(&e.key) + util.NestedSizeOfGcsMinObject(e.m))
	if e.m != nil {
		size += 515
	}

	if e.f != nil {
		size += uint64(util.UnsafeSizeOf(&e.f))
	}

	// Convert heap-size to RSS (resident set size).
	size = uint64(math.Ceil(util.HeapSizeToRssConversionFactor * float64(size)))

	return
}

// Should the supplied object for a new positive entry replace the given
// existing entry?
func shouldReplace(m *gcs.MinObject, existing entry) bool {
	// Negative entries should always be replaced with positive entries.
	if existing.m == nil {
		return true
	}

	// Compare first on generation.
	if m.Generation != existing.m.Generation {
		return m.Generation > existing.m.Generation
	}

	// Break ties on metadata generation.
	if m.MetaGeneration != existing.m.MetaGeneration {
		return m.MetaGeneration > existing.m.MetaGeneration
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

func (sc *statCacheBucketView) Insert(m *gcs.MinObject, expiration time.Time) {
	name := sc.key(m.Name)

	// Is there already a better entry?
	if existing := sc.sharedCache.LookUp(name); existing != nil {
		if !shouldReplace(m, existing.(entry)) {
			return
		}
	}

	// Insert an entry.
	e := entry{
		m:          m,
		expiration: expiration,
		key:        name,
	}

	if _, err := sc.sharedCache.Insert(name, e); err != nil {
		panic(err)
	}
}

func (sc *statCacheBucketView) InsertImplicitDir(objectName string, expiration time.Time) {
	name := sc.key(objectName)

	// Is there already a better entry?
	if existing := sc.sharedCache.LookUp(name); existing != nil {
		e := existing.(entry)
		// The ListObjects response handles directories in two ways:
		// 1. 'MinObject' returns explicit directory objects containing full metadata.
		// 2. 'CollapseRun' generates placeholders for these same directories; if no
		//    explicit object exists, it treats them as "implicit" (inferred).
		//
		// We attempt to create implicit directories for all entries in 'CollapseRun'.
		// However, since 'ListObject' returns explicit directories in the 'MinObject'
		// list as well, this could result in redundant implicit entries for
		// every explicit directory already processed.
		//
		// To prevent this, we check if an entry with the same name already exists
		// with non-nil metadata. If metadata is present, we skip the implicit
		// creation to avoid overwriting a real, explicit object with an inferred
		// placeholder (which would lack metadata and have 'Generation 0').
		if e.m != nil {
			return
		}
	}

	// Insert an entry.
	e := entry{
		implicitDir: true,
		expiration:  expiration,
		key:         name,
	}

	if _, err := sc.sharedCache.Insert(name, e); err != nil {
		logger.Errorf("Failed to insert implicit dir stat cache entry for %q: %v", name, err)
	}
}

func (sc *statCacheBucketView) AddNegativeEntry(objectName string, expiration time.Time) {
	name := sc.key(objectName)

	// Insert a negative entry.
	e := entry{
		m:          nil,
		expiration: expiration,
		key:        name,
	}

	if _, err := sc.sharedCache.Insert(name, e); err != nil {
		panic(err)
	}
}

func (sc *statCacheBucketView) AddNegativeEntryForFolder(folderName string, expiration time.Time) {
	name := sc.key(folderName)

	// Insert a negative entry.
	e := entry{
		f:          nil,
		expiration: expiration,
		key:        name,
	}

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
	now time.Time) (bool, *gcs.MinObject) {
	// Look up in the LRU cache.
	hit, entry := sc.sharedCacheLookup(objectName, now)
	if hit {
		if entry.implicitDir {
			return true, &gcs.MinObject{Name: objectName}
		}
		return hit, entry.m
	}

	return false, nil
}

func (sc *statCacheBucketView) LookUpFolder(
	folderName string,
	now time.Time) (bool, *gcs.Folder) {
	// Look up in the LRU cache.
	hit, entry := sc.sharedCacheLookup(folderName, now)

	if hit {
		return hit, entry.f
	}

	return false, nil
}

func (sc *statCacheBucketView) sharedCacheLookup(key string, now time.Time) (bool, *entry) {
	value := sc.sharedCache.LookUp(sc.key(key))
	if value == nil {
		return false, nil
	}

	e := value.(entry)

	// Has this entry expired?
	if e.expiration.Before(now) {
		sc.Erase(key)
		return false, nil
	}

	return true, &e
}

func (sc *statCacheBucketView) InsertFolder(f *gcs.Folder, expiration time.Time) {
	name := sc.key(f.Name)

	e := entry{
		f:          f,
		expiration: expiration,
		key:        name,
	}

	if _, err := sc.sharedCache.Insert(name, e); err != nil {
		panic(err)
	}
}

// Invalidate cache for all the entries with given prefix.
func (sc *statCacheBucketView) EraseEntriesWithGivenPrefix(prefix string) {
	prefix = sc.key(prefix)
	sc.sharedCache.EraseEntriesWithGivenPrefix(prefix)
}
