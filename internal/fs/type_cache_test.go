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

// A collection of tests for a file system where we do not attempt to write to
// the file system at all. Rather we set up contents in a GCS bucket out of
// band, wait for them to be available, and then read them via the file system.

package fs_test

import (
	"fmt"
	"io/fs"
	"math"
	"os"
	"path"
	"sync"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/internal/config"
	gcsfusefs "github.com/googlecloudplatform/gcsfuse/internal/fs"
	"github.com/googlecloudplatform/gcsfuse/internal/mount"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/internal/util"

	"github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

// The following is the control-flow of an os.Stat(name) call in case of GCSFuse,
// for understanding how the tests work. The stat call
// 1. comes as fs.LookUpInode() call to gcsfuse, which
// 2. queries type-cache of the parent directory, without the suffix '/'.
// 2.1 If entry is found in type-cache, then that is returned as type.
// 2.2 If entry is not found in type-cache, then its type is queried from GCS, and the returned type is stored in type-cache and is also returned as cache.
// 3. If the input to os.Stat() had a suffix '/' and the return type is not ExplicitDir or ImplicitDir, then an error containing 'not a directory' is returned.

////////////////////////////////////////////////////////////////////////
// Common
////////////////////////////////////////////////////////////////////////

type typeCacheTestCommon struct {
	fsTest
}

var (
	// The following should be configured for different tests
	// differently inside SetUpTestSuite as these need to
	// set for mount itself.

	// ttlInSeconds is equivalent of metadata-cache:ttl-secs in config-file.
	ttlInSeconds int64

	// typeCacheMaxSizeMb is equivalent of metadata-cache:type-cache-max-entries in config-file.
	typeCacheMaxSizeMb int
)

func (t *typeCacheTestCommon) SetUpTestSuite() {
	t.serverCfg.MountConfig = config.NewMountConfig()
	t.serverCfg.MountConfig.MetadataCacheConfig = config.MetadataCacheConfig{
		TypeCacheMaxSizeMB: typeCacheMaxSizeMb,
		TtlInSeconds:       ttlInSeconds,
	}

	// Fill server-cfg from mount-config.
	func(mountConfig *config.MountConfig, serverCfg *gcsfusefs.ServerConfig) {
		serverCfg.DirTypeCacheTTL = mount.MetadataCacheTTL(mount.DefaultStatOrTypeCacheTTL, mount.DefaultStatOrTypeCacheTTL,
			mountConfig.TtlInSeconds)
		serverCfg.InodeAttributeCacheTTL = serverCfg.DirTypeCacheTTL
		// We can add more logic here to fill other fileds in serverCfg
		// from mountConfig here as needed.
	}(t.serverCfg.MountConfig, &t.serverCfg)

	// Call through.
	t.fsTest.SetUpTestSuite()
}

////////////////////////////////////////////////////////////////////////
// Specific test classes
////////////////////////////////////////////////////////////////////////

type TypeCacheTestWithMaxSize1MB struct {
	typeCacheTestCommon
}

func (t *TypeCacheTestWithMaxSize1MB) SetUpTestSuite() {
	ttlInSeconds = 30
	typeCacheMaxSizeMb = 1

	t.typeCacheTestCommon.SetUpTestSuite()
}

type TypeCacheTestWithZeroCapacity struct {
	typeCacheTestCommon
}

func (t *TypeCacheTestWithZeroCapacity) SetUpTestSuite() {
	ttlInSeconds = 30
	typeCacheMaxSizeMb = 0

	t.typeCacheTestCommon.SetUpTestSuite()
}

type TypeCacheTestWithZeroTTL struct {
	typeCacheTestCommon
}

func (t *TypeCacheTestWithZeroTTL) SetUpTestSuite() {
	ttlInSeconds = 0
	typeCacheMaxSizeMb = 1

	t.typeCacheTestCommon.SetUpTestSuite()
}

type TypeCacheTestWithInfiniteTTL struct {
	typeCacheTestCommon
}

func (t *TypeCacheTestWithInfiniteTTL) SetUpTestSuite() {
	ttlInSeconds = -1
	typeCacheMaxSizeMb = 1

	t.typeCacheTestCommon.SetUpTestSuite()
}

func init() {
	RegisterTestSuite(&TypeCacheTestWithMaxSize1MB{})
	RegisterTestSuite(&TypeCacheTestWithZeroCapacity{})
	RegisterTestSuite(&TypeCacheTestWithZeroTTL{})
	RegisterTestSuite(&TypeCacheTestWithInfiniteTTL{})
}

// //////////////////////////////////////////////////////////////////////
// helpers
// //////////////////////////////////////////////////////////////////////
func (t *typeCacheTestCommon) testNoInsertion() {
	const name1 = "foo"
	const contents = "taco"
	var fi fs.FileInfo
	var err error

	// Create a file object in GCS.
	fileObject, err := storageutil.CreateObject(
		ctx,
		bucket,
		name1,
		[]byte(contents))

	ExpectEq(nil, err)
	ExpectNe(nil, fileObject)

	// Stat-call with file object. It should
	// pass stat call, bypassing type-cache, as a file.
	fi, err = os.Stat(path.Join(mntDir, name1))

	ExpectEq(nil, err)
	AssertNe(nil, fi)
	ExpectFalse(fi.IsDir())

	// Create a directory object in GCS with same name as the file object.
	dirObject, err := storageutil.CreateObject(
		ctx,
		bucket,
		name1+"/",
		[]byte(contents))

	ExpectEq(nil, err)
	ExpectNe(nil, dirObject)

	// Stat-call with directory object. It should
	// pass stat call, bypassing type-cache, as a directory.
	// It works because no entries are inserted in type-cache
	// in this case.
	fi, err = os.Stat(path.Join(mntDir, name1) + "/")

	ExpectEq(nil, err)
	AssertNe(nil, fi)
	ExpectTrue(fi.IsDir())
}

// //////////////////////////////////////////////////////////////////////
// Tests for TypeCacheTestWithMaxEntries1
// //////////////////////////////////////////////////////////////////////
func (t *TypeCacheTestWithMaxSize1MB) TestSizeBasedEviction() {
	const name1 = "foo"
	const contents = "taco"
	contentInBytes := []byte(contents)
	var fi fs.FileInfo
	var err error
	objectNameTemplate := "type_cache_test_object_%06d" // Will create a 29 character string.
	// Increase the object-name size to increase per-entry-size (max allowed is 1024)
	// to decrease count of objects being created for this,
	// to reduce the runtime.
	for i := 0; i < 99; i++ {
		objectNameTemplate += "abcdefjhij" // This makes it length+=10.
	}
	nameOfIthObject := func(i int) string {
		return fmt.Sprintf(objectNameTemplate, i)
	}
	objectNameSample := nameOfIthObject(0)
	perEntrySize := int(metadata.SizeOfTypeCacheEntry(objectNameSample))

	// Initially, without any existing object, type-cache
	// should not contain any entry and os.Stat should fail.
	_, err = os.Stat(path.Join(mntDir, name1))

	ExpectNe(nil, err)

	// Create a file object in GCS.
	fileObject, err := storageutil.CreateObject(
		ctx,
		bucket,
		name1,
		[]byte(contents))

	ExpectEq(nil, err)
	ExpectNe(nil, fileObject)

	// Stat-call with first file object. It should
	// pass stat call, through type-cache as a file.
	fi, err = os.Stat(path.Join(mntDir, name1))

	ExpectEq(nil, err)
	AssertNe(nil, fi)
	ExpectFalse(fi.IsDir())

	// Create a directory object in GCS with same name as the first file object.
	dirObject, err := storageutil.CreateObject(
		ctx,
		bucket,
		name1+"/",
		contentInBytes)

	ExpectEq(nil, err)
	ExpectNe(nil, dirObject)

	// Stat-call with the directory object. It should
	// fail as there is currently an entry for the first
	// file object, which has the same name.
	_, err = os.Stat(path.Join(mntDir, name1) + "/")

	ExpectNe(nil, err)
	ExpectThat(err, oglematchers.Error(oglematchers.HasSubstr("not a directory")))

	// type-cache-max-size-mb = 1MiB.
	// let's add another 1MiB/perEntrySize entries to evict
	// the first file object ("foo") entry.
	numObjectsToBeInserted := int(util.MiBsToBytes(1)) / perEntrySize

	// If we run a single for-loop over all numObjectsToBeInserted,
	// then object creation might take a long time.
	// Let us parallelize it to reduce runtime, by breaking it into batches.
	wg := sync.WaitGroup{}
	createAndStatBatchOfObjects := func(batchOffset, batchSize int) {
		defer wg.Done()

		for i := batchOffset; i < batchOffset+batchSize; i++ {
			name := nameOfIthObject(i)

			// Create a new file object in GCS.
			fileObject, err = storageutil.CreateObject(
				ctx,
				bucket,
				name,
				contentInBytes)

			ExpectEq(nil, err)
			ExpectNe(nil, fileObject)

			// Stat-call will insert the new file objects into the type-cache.
			// As a side-effect of all these insertions,
			// the first file object (foo) will be evicted from type-cache
			// because of type-cache-max-size-mb=1 .
			fi, err = os.Stat(path.Join(mntDir, name))

			ExpectEq(nil, err)
			AssertNe(nil, fi)
			ExpectFalse(fi.IsDir())
		}
	}
	maxNumParallelBatches := 8
	maxNumObjectsPerBatch := int(math.Ceil(float64(numObjectsToBeInserted) / float64(maxNumParallelBatches)))
	AssertGt(maxNumObjectsPerBatch, 0) // To avoid an infinite for-loop.
	var batchOffset int
	for remainingObjectsToBeInserted := numObjectsToBeInserted; remainingObjectsToBeInserted > 0; {
		objectsInsertedInThisBatch := maxNumObjectsPerBatch
		if objectsInsertedInThisBatch > remainingObjectsToBeInserted {
			objectsInsertedInThisBatch = remainingObjectsToBeInserted
		}

		wg.Add(1)
		go createAndStatBatchOfObjects(batchOffset, objectsInsertedInThisBatch)

		remainingObjectsToBeInserted -= objectsInsertedInThisBatch
		batchOffset += objectsInsertedInThisBatch
	}
	wg.Wait()

	// Stat-call with directory object again, to verify that the first file's
	// type-cache entry got removed, and this time type-cache inserts a directory entry
	// and stat returns a directory successfully.
	fi, err = os.Stat(path.Join(mntDir, name1) + "/")

	ExpectEq(nil, err)
	AssertNe(nil, fi)
	ExpectTrue(fi.IsDir())
}

func (t *TypeCacheTestWithMaxSize1MB) TestTTLBasedEviction() {
	const name1 = "foo"
	const contents = "taco"
	var fi fs.FileInfo
	var err error

	// Create a file object in GCS.
	fileObject, err := storageutil.CreateObject(
		ctx,
		bucket,
		name1,
		[]byte(contents))

	ExpectEq(nil, err)
	ExpectNe(nil, fileObject)

	// Stat-call with existing object, found in type-cache.
	fi, err = os.Stat(path.Join(mntDir, name1))

	ExpectEq(nil, err)
	AssertNe(nil, fi)
	ExpectFalse(fi.IsDir())

	// Create a directory object in GCS with same name as the file object.
	dirObject, err := storageutil.CreateObject(
		ctx,
		bucket,
		name1+"/",
		[]byte(contents))

	ExpectEq(nil, err)
	ExpectNe(nil, dirObject)

	// Stat-call with the directory object. It should
	// fail as there is already an entry for the
	// file object, which has the same name.
	_, err = os.Stat(path.Join(mntDir, name1) + "/")

	ExpectNe(nil, err)
	ExpectThat(err, oglematchers.Error(oglematchers.HasSubstr("not a directory")))

	// Doubly confirming that the type-cache still has
	// the entry for the file object.
	fi, err = os.Stat(path.Join(mntDir, name1))

	ExpectEq(nil, err)
	AssertNe(nil, fi)
	ExpectFalse(fi.IsDir())

	// Advance time to cross TTL to let the file-object entry be
	// removed from type-cache.
	cacheClock.AdvanceTime(time.Duration(ttlInSeconds)*time.Second + time.Nanosecond)

	// Stat-call with directory object to verify that the file object's
	// type-cache entry got removed.
	fi, err = os.Stat(path.Join(mntDir, name1) + "/")

	ExpectEq(nil, err)
	AssertNe(nil, fi)
	ExpectTrue(fi.IsDir())
}

// //////////////////////////////////////////////////////////////////////
// Tests for TypeCacheTestWithZeroCapacity
// //////////////////////////////////////////////////////////////////////
func (t *TypeCacheTestWithZeroCapacity) TestNoInsertion() {
	t.typeCacheTestCommon.testNoInsertion()
}

// //////////////////////////////////////////////////////////////////////
// Tests for TypeCacheTestWithZeroTTL
// //////////////////////////////////////////////////////////////////////
func (t *TypeCacheTestWithZeroTTL) TestNoInsertion() {
	t.typeCacheTestCommon.testNoInsertion()
}

// //////////////////////////////////////////////////////////////////////
// Tests for TypeCacheTestWithInfiniteTTL
// //////////////////////////////////////////////////////////////////////
func (t *TypeCacheTestWithInfiniteTTL) TestNoTTLExpiryEver() {
	const name1 = "foo"
	const contents = "taco"
	var fi fs.FileInfo
	var err error

	// Create a file object in GCS.
	fileObject, err := storageutil.CreateObject(
		ctx,
		bucket,
		name1,
		[]byte(contents))

	ExpectEq(nil, err)
	ExpectNe(nil, fileObject)

	// Stat-call with file object. It should
	// pass stat call, as a file.
	fi, err = os.Stat(path.Join(mntDir, name1))

	ExpectEq(nil, err)
	AssertNe(nil, fi)
	ExpectFalse(fi.IsDir())

	// Let 100 years pass in the type-cache's simulated clock.
	// Surely, type-cache won't forget about the file entry.
	cacheClock.AdvanceTime(100 * 365.2425 * 24 * time.Hour)

	// Create a directory object in GCS with same name as the file object.
	dirObject, err := storageutil.CreateObject(
		ctx,
		bucket,
		name1+"/",
		[]byte(contents))

	ExpectEq(nil, err)
	ExpectNe(nil, dirObject)

	// Stat-call with the directory object. It should
	// fail as there is already a type-cache entry for the
	// file object, which has the same name.
	_, err = os.Stat(path.Join(mntDir, name1) + "/")

	ExpectNe(nil, err)
	ExpectThat(err, oglematchers.Error(oglematchers.HasSubstr("not a directory")))

	// Doubly confirming that the type-cache still has
	// the entry for the file object.
	fi, err = os.Stat(path.Join(mntDir, name1))

	ExpectEq(nil, err)
	AssertNe(nil, fi)
	ExpectFalse(fi.IsDir())
}
