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
	"github.com/googlecloudplatform/gcsfuse/internal/storage/gcs"
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

const (
	foo  = "foo"
	taco = "taco"
)

var (
	// The following should be configured for different tests
	// differently inside SetUpTestSuite as these need to
	// set for mount itself.

	// ttlInSeconds is equivalent of metadata-cache:ttl-secs in config-file.
	ttlInSeconds int64

	// typeCacheMaxSizeMb is equivalent of metadata-cache:type-cache-max-entries in config-file.
	typeCacheMaxSizeMb int

	contentInBytes []byte

	fi  fs.FileInfo
	err error
)

func (t *typeCacheTestCommon) SetUpTestSuite() {
	t.serverCfg.MountConfig = config.NewMountConfig()
	t.serverCfg.MountConfig.MetadataCacheConfig = config.MetadataCacheConfig{
		TypeCacheMaxSizeMB: typeCacheMaxSizeMb,
		TtlInSeconds:       ttlInSeconds,
	}

	// Fill server-cfg from mount-config.
	func(mountConfig *config.MountConfig, serverCfg *gcsfusefs.ServerConfig) {
		serverCfg.DirTypeCacheTTL = mount.ResolveMetadataCacheTTL(mount.DefaultStatOrTypeCacheTTL, mount.DefaultStatOrTypeCacheTTL,
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

type TypeCacheTestWithZeroSize struct {
	typeCacheTestCommon
}

func (t *TypeCacheTestWithZeroSize) SetUpTestSuite() {
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
	RegisterTestSuite(&TypeCacheTestWithZeroSize{})
	RegisterTestSuite(&TypeCacheTestWithZeroTTL{})
	RegisterTestSuite(&TypeCacheTestWithInfiniteTTL{})

	const contents string = "taco"
	contentInBytes = []byte(contents)
}

// //////////////////////////////////////////////////////////////////////
// helpers
// //////////////////////////////////////////////////////////////////////
func (t *typeCacheTestCommon) createObjectOnGCS(name string) *gcs.Object {
	// Create a file/directory object in a fake-bucket.
	fileObject, err := storageutil.CreateObject(
		ctx,
		bucket,
		name,
		contentInBytes)

	ExpectEq(nil, err)
	AssertNe(nil, fileObject)

	return fileObject
}

func (t *typeCacheTestCommon) statAndConfirmIsDir(name string, isDir bool) {
	fi, err = os.Stat(name)

	ExpectEq(nil, err)
	AssertNe(nil, fi)
	ExpectEq(isDir, fi.IsDir())
}

func (t *typeCacheTestCommon) statAndExpectNotADirectoryError(name string) {
	_, err = os.Stat(name)

	ExpectNe(nil, err)
	ExpectThat(err, oglematchers.Error(oglematchers.HasSubstr("not a directory")))
}

func (t *typeCacheTestCommon) testNoInsertionSupported() {
	// Create a file object in GCS.
	t.createObjectOnGCS(foo)

	// Stat-call with file object. It should
	// pass stat call, skip type-cache, and return type as file.
	t.statAndConfirmIsDir(path.Join(mntDir, foo), false)

	// Create a directory object in GCS with same name as the file object.
	t.createObjectOnGCS(foo + "/")

	// Stat-call with directory object. It should
	// pass stat call, skip type-cache, and return type as directory.
	t.statAndConfirmIsDir(path.Join(mntDir, foo)+"/", true)
}

// //////////////////////////////////////////////////////////////////////
// Tests for TypeCacheTestWithMaxSize1MB
// //////////////////////////////////////////////////////////////////////
func (t *TypeCacheTestWithMaxSize1MB) TestNoEntryInitially() {
	// Initially, without any existing object, type-cache
	// should not contain any entry and os.Stat should fail.
	_, err = os.Stat(path.Join(mntDir, foo))

	ExpectNe(nil, err)
	ExpectThat(err, oglematchers.Error(oglematchers.HasSubstr("no such file or directory")))
}

func (t *TypeCacheTestWithMaxSize1MB) TestStatBehavior_DirHiddenByFile() {
	// Create a file object in GCS.
	t.createObjectOnGCS(foo)

	// Stat-call with the file object. It should
	// pass stat call, hit type-cache and return type as file.
	t.statAndConfirmIsDir(path.Join(mntDir, foo), false)

	// Create a directory object in GCS with same name as the file object.
	t.createObjectOnGCS(foo + "/")

	// Stat-call with the directory object path. It should
	// fail the stat call, as type-cache currently contains a file entry with the same name and
	// type is returned as file which is not compatible with the passed trailing "/".
	// So, this stat call should fail and return "not a directory error"
	t.statAndExpectNotADirectoryError(path.Join(mntDir, foo) + "/")
}

func (t *TypeCacheTestWithMaxSize1MB) TestStatBehavior_FileHiddenByDir() {
	// Create a directory object in GCS.
	t.createObjectOnGCS(foo + "/")

	// Stat-call with the directory object. It should
	// pass and return type as directory.
	t.statAndConfirmIsDir(path.Join(mntDir, foo)+"/", true)

	// Create a file object in GCS with same name as the directory object.
	t.createObjectOnGCS(foo)

	// Stat-call with the file object should
	// pass but it should still report it as a directory
	// because of its old type-cache entry which says type is directory.
	t.statAndConfirmIsDir(path.Join(mntDir, foo), true)
}

func (t *TypeCacheTestWithMaxSize1MB) TestSizeBasedEviction() {
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

	// Create a file object in GCS.
	t.createObjectOnGCS(foo)

	// Stat-call with first file object. It should
	// pass stat call, through type-cache, and return type as file.
	t.statAndConfirmIsDir(path.Join(mntDir, foo), false)

	// Create a directory object in GCS with same name as the first file object.
	t.createObjectOnGCS(foo + "/")

	// Stat-call with the directory object path. It should
	// fail the stat call, as type-cache currently contains a file entry with the same name and
	// type is returned as file which is not compatible with the passed trailing "/".
	// So, this stat call should fail and return "not a directory error"
	t.statAndExpectNotADirectoryError(path.Join(mntDir, foo) + "/")

	// type-cache-max-size-mb = 1MiB.
	// let's add another 1MiB/perEntrySize entries to evict
	// the first file object ("foo") entry from the type-cache.
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
			t.createObjectOnGCS(name)

			// Stat-call will insert the new file objects into the type-cache.
			// As a side-effect of all these insertions,
			// the first file object (foo) will be evicted from type-cache
			// because of type-cache-max-size-mb=1 .
			t.statAndConfirmIsDir(path.Join(mntDir, name), false)
		}
	}

	// Create and execute batches of object.
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
	// type-cache entry got evicted, and this time type-cache inserts a directory entry
	// and stat returns a directory type successfully.
	t.statAndConfirmIsDir(path.Join(mntDir, foo)+"/", true)
}

func (t *TypeCacheTestWithMaxSize1MB) TestTTLBasedEviction() {
	// Create a file object in GCS.
	t.createObjectOnGCS(foo)

	// Stat-call with existing object, found in type-cache and returned as type file.
	t.statAndConfirmIsDir(path.Join(mntDir, foo), false)

	// Create a directory object in GCS with same name as the file object.
	t.createObjectOnGCS(foo + "/")

	// Stat-call with the directory object path. It should
	// fail the stat call, as type-cache currently contains a file entry with the same name and
	// type is returned as file which is not compatible with the passed trailing "/".
	// So, this stat call should fail and return "not a directory error"
	t.statAndExpectNotADirectoryError(path.Join(mntDir, foo) + "/")

	// Doubly confirming that the type-cache still has
	// the entry for the file object.
	t.statAndConfirmIsDir(path.Join(mntDir, foo), false)

	// Advance time to cross TTL to let the file-object entry be
	// removed from type-cache.
	cacheClock.AdvanceTime(time.Duration(ttlInSeconds)*time.Second + time.Nanosecond)

	// Stat-call with directory object to verify that the file object's
	// type-cache entry got removed.
	t.statAndConfirmIsDir(path.Join(mntDir, foo)+"/", true)
}

// //////////////////////////////////////////////////////////////////////
// Tests for TypeCacheTestWithZeroSize
// //////////////////////////////////////////////////////////////////////
func (t *TypeCacheTestWithZeroSize) TestNoInsertionSupported() {
	t.typeCacheTestCommon.testNoInsertionSupported()
}

// //////////////////////////////////////////////////////////////////////
// Tests for TypeCacheTestWithZeroTTL
// //////////////////////////////////////////////////////////////////////
func (t *TypeCacheTestWithZeroTTL) TestNoInsertionSupported() {
	t.typeCacheTestCommon.testNoInsertionSupported()
}

// //////////////////////////////////////////////////////////////////////
// Tests for TypeCacheTestWithInfiniteTTL
// //////////////////////////////////////////////////////////////////////
func (t *TypeCacheTestWithInfiniteTTL) TestNoTTLExpiryEver() {
	// Create a file object in GCS.
	t.createObjectOnGCS(foo)

	// Stat-call with file object. It should
	// pass stat call, as a file.
	t.statAndConfirmIsDir(path.Join(mntDir, foo), false)

	// Let 100 years pass in the type-cache's simulated clock.
	// Surely, type-cache won't forget about the file entry.
	cacheClock.AdvanceTime(100 * 365.2425 * 24 * time.Hour)

	// Create a directory object in GCS with same name as the file object.
	t.createObjectOnGCS(foo + "/")

	// Stat-call with the directory object path. It should
	// fail the stat call, as type-cache currently contains a file entry with the same name and
	// type is returned as file which is not compatible with the passed trailing "/".
	// So, this stat call should fail and return "not a directory error"
	t.statAndExpectNotADirectoryError(path.Join(mntDir, foo) + "/")

	// Doubly confirming that the type-cache still has
	// the entry for the file object.
	t.statAndConfirmIsDir(path.Join(mntDir, foo), false)
}
