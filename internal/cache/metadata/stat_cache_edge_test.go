// Copyright 2026 Google LLC
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

package metadata_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type StatCacheEdgeTest struct {
	suite.Suite
	statCache metadata.StatCache
}

func TestStatCacheEdge(t *testing.T) {
	suite.Run(t, new(StatCacheEdgeTest))
}

func (t *StatCacheEdgeTest) SetupTest() {
	capacity := 100
	cache := lru.NewCache(uint64(cfg.AverageSizeOfPositiveStatCacheEntry+cfg.AverageSizeOfNegativeStatCacheEntry) * uint64(capacity))
	t.statCache = metadata.NewStatCacheBucketView(cache, "")
}

// 1. Implicit Directory Hit Paths & Zero-Value Ambiguity
func (t *StatCacheEdgeTest) Test_ImplicitDir_HitPath_NonEmptyName() {
	now := time.Now()
	exp := now.Add(time.Hour)
	const dirName = "testdir/"

	t.statCache.InsertImplicitDir(dirName, exp)

	hit, minObj := t.statCache.LookUp(dirName, now)
	assert.True(t.T(), hit)
	assert.Equal(t.T(), dirName, minObj.Name)
	assert.Equal(t.T(), int64(0), minObj.Generation)
	assert.Equal(t.T(), uint64(0), minObj.Size)
}

func (t *StatCacheEdgeTest) Test_ImplicitDir_EmptyName_ZeroValueAmbiguity() {
	now := time.Now()
	exp := now.Add(time.Hour)

	// Insert implicit directory with empty name ""
	t.statCache.InsertImplicitDir("", exp)

	hit, minObj := t.statCache.LookUp("", now)
	assert.True(t.T(), hit)
	// MinObject returned for implicit dir with "" has Name:"" and Generation:0
	// which is indistinguishable from zero-value MinObject returned for negative cache entries.
	assert.Equal(t.T(), gcs.MinObject{}, minObj)
}

// 2. Negative Cache Hits
func (t *StatCacheEdgeTest) Test_NegativeCacheHits_MinObject() {
	now := time.Now()
	exp := now.Add(time.Hour)
	const objName = "nonexistent.txt"

	t.statCache.AddNegativeEntry(objName, exp)

	hit, minObj := t.statCache.LookUp(objName, now)
	assert.True(t.T(), hit)
	assert.Equal(t.T(), gcs.MinObject{}, minObj)
}

func (t *StatCacheEdgeTest) Test_NegativeCacheHits_Folder() {
	now := time.Now()
	exp := now.Add(time.Hour)
	const folderName = "nonexistent_folder/"

	t.statCache.AddNegativeEntryForFolder(folderName, exp)

	hit, folder := t.statCache.LookUpFolder(folderName, now)
	assert.True(t.T(), hit)
	assert.Equal(t.T(), gcs.Folder{}, folder)
}

// 3. Expired Cache Entries Boundary Conditions
func (t *StatCacheEdgeTest) Test_ExpiredCacheEntries_ExactBoundaries() {
	startTime := time.Date(2026, 7, 11, 10, 0, 0, 0, time.UTC)
	expTime := startTime.Add(10 * time.Second)

	// Insert positive object, folder, implicit dir, and negative entry
	t.statCache.Insert(&gcs.MinObject{Name: "pos_obj", Generation: 1}, expTime)
	t.statCache.InsertFolder(&gcs.Folder{Name: "pos_folder/"}, expTime)
	t.statCache.InsertImplicitDir("impl_dir/", expTime)
	t.statCache.AddNegativeEntry("neg_obj", expTime)

	// 1. Before expiration: all must be hits
	beforeExp := expTime.Add(-time.Nanosecond)
	hitPos, _ := t.statCache.LookUp("pos_obj", beforeExp)
	hitFolder, _ := t.statCache.LookUpFolder("pos_folder/", beforeExp)
	hitImpl, _ := t.statCache.LookUp("impl_dir/", beforeExp)
	hitNeg, _ := t.statCache.LookUp("neg_obj", beforeExp)
	assert.True(t.T(), hitPos)
	assert.True(t.T(), hitFolder)
	assert.True(t.T(), hitImpl)
	assert.True(t.T(), hitNeg)

	// 2. Exactly at expiration time: e.expiration.Before(expTime) is FALSE (not strictly before).
	hitAtExp, _ := t.statCache.LookUp("pos_obj", expTime)
	assert.True(t.T(), hitAtExp)

	// 3. 1 ns after expiration time: e.expiration.Before(expTime + 1ns) is TRUE -> must expire and return false
	afterExp := expTime.Add(time.Nanosecond)
	hitPosAfter, objAfter := t.statCache.LookUp("pos_obj", afterExp)
	hitFolderAfter, folderAfter := t.statCache.LookUpFolder("pos_folder/", afterExp)
	hitImplAfter, implAfter := t.statCache.LookUp("impl_dir/", afterExp)
	hitNegAfter, negAfter := t.statCache.LookUp("neg_obj", afterExp)

	assert.False(t.T(), hitPosAfter)
	assert.Equal(t.T(), gcs.MinObject{}, objAfter)

	assert.False(t.T(), hitFolderAfter)
	assert.Equal(t.T(), gcs.Folder{}, folderAfter)

	assert.False(t.T(), hitImplAfter)
	assert.Equal(t.T(), gcs.MinObject{}, implAfter)

	assert.False(t.T(), hitNegAfter)
	assert.Equal(t.T(), gcs.MinObject{}, negAfter)
}

// 4. Folder Lookups & Zero-Value Ambiguity
func (t *StatCacheEdgeTest) Test_FolderLookup_Positive() {
	now := time.Now()
	exp := now.Add(time.Hour)
	updateTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	folder := &gcs.Folder{Name: "myfolder/", UpdateTime: updateTime}
	t.statCache.InsertFolder(folder, exp)

	hit, fetchedFolder := t.statCache.LookUpFolder("myfolder/", now)
	assert.True(t.T(), hit)
	assert.Equal(t.T(), "myfolder/", fetchedFolder.Name)
	assert.Equal(t.T(), updateTime, fetchedFolder.UpdateTime)
}

func (t *StatCacheEdgeTest) Test_FolderLookup_ZeroValueAmbiguity() {
	now := time.Now()
	exp := now.Add(time.Hour)

	// Inserting zero-value folder
	zeroFolder := &gcs.Folder{}
	t.statCache.InsertFolder(zeroFolder, exp)

	hit, fetchedFolder := t.statCache.LookUpFolder("", now)
	assert.True(t.T(), hit)
	// Notice: zeroFolder is gcs.Folder{}, which is identical to the negative entry marker.
	assert.Equal(t.T(), gcs.Folder{}, fetchedFolder)
}

// 5. Stress Harness & Concurrency Safety
func (t *StatCacheEdgeTest) Test_ConcurrentOperations_StressHarness() {
	capacity := 1000
	cache := lru.NewCache(uint64(cfg.AverageSizeOfPositiveStatCacheEntry+cfg.AverageSizeOfNegativeStatCacheEntry) * uint64(capacity))
	sharedStatCache := metadata.NewStatCacheBucketView(cache, "bucket_stress")

	var wg sync.WaitGroup
	numGoroutines := 20
	iterations := 500
	now := time.Now()
	exp := now.Add(10 * time.Minute)

	// Concurrency stress: multi-threaded operations on value-return methods
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		workerID := i
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				keyName := fmt.Sprintf("obj_%d_%d", workerID, j%20)
				folderName := fmt.Sprintf("folder_%d_%d/", workerID, j%20)

				switch j % 5 {
				case 0:
					sharedStatCache.Insert(&gcs.MinObject{Name: keyName, Generation: int64(j)}, exp)
				case 1:
					sharedStatCache.InsertFolder(&gcs.Folder{Name: folderName}, exp)
				case 2:
					sharedStatCache.AddNegativeEntry(keyName, exp)
					sharedStatCache.AddNegativeEntryForFolder(folderName, exp)
				case 3:
					hit, m := sharedStatCache.LookUp(keyName, now)
					if hit && m.Name != "" {
						assert.Equal(t.T(), keyName, m.Name)
					}
					hitF, f := sharedStatCache.LookUpFolder(folderName, now)
					if hitF && f.Name != "" {
						assert.Equal(t.T(), folderName, f.Name)
					}
				case 4:
					sharedStatCache.Erase(keyName)
				}
			}
		}()
	}

	wg.Wait()
}
