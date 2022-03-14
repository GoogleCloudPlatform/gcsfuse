// Copyright 2022 Google Inc. All Rights Reserved.
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

package contentcache_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/internal/contentcache"
	"github.com/jacobsa/fuse/fsutil"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
)

const numConcurrentGoRoutines = 100
const testTempDir = "/tmp"
const testGeneration = 10002022
const testGenerationOld = 10002020
const testMetaGeneration = 1
const testMetaGenerationOld = 2

func TestValidateGeneration(t *testing.T) {
	objectMetadata := contentcache.CacheFileObjectMetadata{
		CacheFileNameOnDisk: "foobar",
		BucketName:          "foo",
		ObjectName:          "baz",
		Generation:          testGeneration,
		MetaGeneration:      testMetaGeneration,
	}
	cacheObject := contentcache.CacheObject{CacheFileObjectMetadata: &objectMetadata}
	ExpectTrue(cacheObject.ValidateGeneration(testGeneration, testMetaGeneration))
}

func TestValidateGenerationNegative(t *testing.T) {
	objectMetadata := contentcache.CacheFileObjectMetadata{
		CacheFileNameOnDisk: "foobar",
		BucketName:          "foo",
		ObjectName:          "baz",
		Generation:          testGeneration,
		MetaGeneration:      testMetaGeneration,
	}
	cacheObject := contentcache.CacheObject{CacheFileObjectMetadata: &objectMetadata}
	ExpectFalse(cacheObject.ValidateGeneration(testGenerationOld, testMetaGenerationOld))
	ExpectFalse(cacheObject.ValidateGeneration(testGenerationOld, testMetaGeneration))
	ExpectFalse(cacheObject.ValidateGeneration(testGeneration, testMetaGenerationOld))
}

func TestReadWriteMetadataCheckpointFile(t *testing.T) {
	mtimeClock := timeutil.RealClock()
	contentCache := contentcache.New(testTempDir, mtimeClock)
	f, err := fsutil.AnonymousFile(testTempDir)
	AssertEq(err, nil)
	objectMetadata := contentcache.CacheFileObjectMetadata{
		CacheFileNameOnDisk: f.Name(),
		BucketName:          "foo",
		ObjectName:          "baz",
		Generation:          testGeneration,
		MetaGeneration:      testMetaGeneration,
	}
	metadataFileName, err := contentCache.WriteMetadataCheckpointFile(objectMetadata.ObjectName, &objectMetadata)
	AssertEq(err, nil)
	newObjectMetadata := contentcache.CacheFileObjectMetadata{}
	contents, err := ioutil.ReadFile(metadataFileName)
	AssertEq(err, nil)
	err = json.Unmarshal(contents, &newObjectMetadata)
	AssertEq(err, nil)
	// There is no struct equality support in ExpectEq
	ExpectEq(objectMetadata.BucketName, newObjectMetadata.BucketName)
	ExpectEq(objectMetadata.CacheFileNameOnDisk, newObjectMetadata.CacheFileNameOnDisk)
	ExpectEq(objectMetadata.Generation, newObjectMetadata.Generation)
	ExpectEq(objectMetadata.MetaGeneration, newObjectMetadata.MetaGeneration)
	ExpectEq(objectMetadata.ObjectName, newObjectMetadata.ObjectName)
	os.Remove(metadataFileName)
}

// TestContentCacheAddOrReplace should panic on concurrent map access
func TestContentCacheAddOrReplace(t *testing.T) {
	var wg sync.WaitGroup
	mtimeClock := timeutil.RealClock()
	contentCache := contentcache.New(testTempDir, mtimeClock)
	cacheObjectKey := &contentcache.CacheObjectKey{
		BucketName: "foo",
		ObjectName: "baz",
	}
	for i := 1; i <= numConcurrentGoRoutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := contentCache.AddOrReplace(cacheObjectKey, 1000, 1, nil)
			ExpectEq(err, nil)
		}()
	}
	wg.Wait()
}

func TestContentCacheGet(t *testing.T) {
	var wg sync.WaitGroup
	mtimeClock := timeutil.RealClock()
	contentCache := contentcache.New(testTempDir, mtimeClock)
	cacheObjectKey := &contentcache.CacheObjectKey{
		BucketName: "foo",
		ObjectName: "baz",
	}
	cacheObject, err := contentCache.AddOrReplace(cacheObjectKey, 1000, 1, nil)
	ExpectEq(err, nil)
	for i := 1; i <= numConcurrentGoRoutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			retrievedCacheObject, exists := contentCache.Get(cacheObjectKey)
			ExpectTrue(exists)
			ExpectEq(cacheObject.CacheFile, retrievedCacheObject.CacheFile)
			ExpectEq(cacheObject.CacheFileObjectMetadata, retrievedCacheObject.CacheFileObjectMetadata)
			ExpectEq(cacheObject.MetadataFileName, retrievedCacheObject.MetadataFileName)
		}()
	}
	wg.Wait()
}

func TestContentCacheRemove(t *testing.T) {
	var wg sync.WaitGroup
	mtimeClock := timeutil.RealClock()
	contentCache := contentcache.New(testTempDir, mtimeClock)
	for i := 1; i <= numConcurrentGoRoutines; i++ {
		cacheObjectKey := &contentcache.CacheObjectKey{
			BucketName: "foo",
			ObjectName: fmt.Sprintf("baz%d", i),
		}
		_, err := contentCache.AddOrReplace(cacheObjectKey, 1000, 1, nil)
		ExpectEq(err, nil)
	}
	for i := 1; i <= numConcurrentGoRoutines; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			cacheObjectKey := &contentcache.CacheObjectKey{
				BucketName: "foo",
				ObjectName: fmt.Sprintf("baz%d", i),
			}
			contentCache.Remove(cacheObjectKey)
		}()
	}
	wg.Wait()
	ExpectEq(contentCache.Size(), 0)
}
