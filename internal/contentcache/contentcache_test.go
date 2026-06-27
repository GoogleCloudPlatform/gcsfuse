// Copyright 2022 Google LLC
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
	"os"
	"path"
	"sync"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/contentcache"
	"github.com/jacobsa/fuse/fsutil"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const numConcurrentGoRoutines = 100
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
	assert.True(t, cacheObject.ValidateGeneration(testGeneration, testMetaGeneration))
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
	assert.False(t, cacheObject.ValidateGeneration(testGenerationOld, testMetaGenerationOld))
	assert.False(t, cacheObject.ValidateGeneration(testGenerationOld, testMetaGeneration))
	assert.False(t, cacheObject.ValidateGeneration(testGeneration, testMetaGenerationOld))
}

func TestReadWriteMetadataCheckpointFile(t *testing.T) {
	tempDir := t.TempDir()
	mtimeClock := timeutil.RealClock()
	contentCache := contentcache.New(tempDir, mtimeClock)
	f, err := fsutil.AnonymousFile(tempDir)
	require.NoError(t, err)
	defer f.Close()

	objectMetadata := contentcache.CacheFileObjectMetadata{
		CacheFileNameOnDisk: f.Name(),
		BucketName:          "foo",
		ObjectName:          "baz",
		Generation:          testGeneration,
		MetaGeneration:      testMetaGeneration,
	}
	// Write the checkpoint file to the temp directory instead of the working directory
	metadataFileName, err := contentCache.WriteMetadataCheckpointFile(path.Join(tempDir, objectMetadata.ObjectName), &objectMetadata)
	require.NoError(t, err)
	newObjectMetadata := contentcache.CacheFileObjectMetadata{}
	contents, err := os.ReadFile(metadataFileName)
	require.NoError(t, err)
	err = json.Unmarshal(contents, &newObjectMetadata)
	require.NoError(t, err)

	assert.Equal(t, objectMetadata.BucketName, newObjectMetadata.BucketName)
	assert.Equal(t, objectMetadata.CacheFileNameOnDisk, newObjectMetadata.CacheFileNameOnDisk)
	assert.Equal(t, objectMetadata.Generation, newObjectMetadata.Generation)
	assert.Equal(t, objectMetadata.MetaGeneration, newObjectMetadata.MetaGeneration)
	assert.Equal(t, objectMetadata.ObjectName, newObjectMetadata.ObjectName)
}

// TestContentCacheAddOrReplace should panic on concurrent map access
func TestContentCacheAddOrReplace(t *testing.T) {
	tempDir := t.TempDir()
	var wg sync.WaitGroup
	mtimeClock := timeutil.RealClock()
	contentCache := contentcache.New(tempDir, mtimeClock)
	cacheObjectKey := &contentcache.CacheObjectKey{
		BucketName: "foo",
		ObjectName: "baz",
	}
	for i := 1; i <= numConcurrentGoRoutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := contentCache.AddOrReplace(cacheObjectKey, 1000, 1, nil)
			assert.NoError(t, err)
		}()
	}
	wg.Wait()
}

func TestContentCacheGet(t *testing.T) {
	tempDir := t.TempDir()
	var wg sync.WaitGroup
	mtimeClock := timeutil.RealClock()
	contentCache := contentcache.New(tempDir, mtimeClock)
	cacheObjectKey := &contentcache.CacheObjectKey{
		BucketName: "foo",
		ObjectName: "baz",
	}
	cacheObject, err := contentCache.AddOrReplace(cacheObjectKey, 1000, 1, nil)
	require.NoError(t, err)
	for i := 1; i <= numConcurrentGoRoutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			retrievedCacheObject, exists := contentCache.Get(cacheObjectKey)
			assert.True(t, exists)
			assert.Equal(t, cacheObject.CacheFile, retrievedCacheObject.CacheFile)
			assert.Equal(t, cacheObject.CacheFileObjectMetadata, retrievedCacheObject.CacheFileObjectMetadata)
			assert.Equal(t, cacheObject.MetadataFileName, retrievedCacheObject.MetadataFileName)
		}()
	}
	wg.Wait()
}

func TestContentCacheRemove(t *testing.T) {
	tempDir := t.TempDir()
	var wg sync.WaitGroup
	mtimeClock := timeutil.RealClock()
	contentCache := contentcache.New(tempDir, mtimeClock)
	for i := 1; i <= numConcurrentGoRoutines; i++ {
		cacheObjectKey := &contentcache.CacheObjectKey{
			BucketName: "foo",
			ObjectName: fmt.Sprintf("baz%d", i),
		}
		_, err := contentCache.AddOrReplace(cacheObjectKey, 1000, 1, nil)
		require.NoError(t, err)
	}
	for i := 1; i <= numConcurrentGoRoutines; i++ {
		wg.Add(1)
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
	assert.Equal(t, 0, contentCache.Size())
}
