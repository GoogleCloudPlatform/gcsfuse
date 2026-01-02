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

package gcsx

import (
	"strconv"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type MrdInstanceTest struct {
	suite.Suite
	object      *gcs.MinObject
	bucket      *storage.TestifyMockBucket
	cache       *lru.Cache
	inodeID     fuseops.InodeID
	mrdConfig   cfg.MrdConfig
	mrdInstance *MrdInstance
}

func TestMrdInstanceTestSuite(t *testing.T) {
	suite.Run(t, new(MrdInstanceTest))
}

func (t *MrdInstanceTest) SetupTest() {
	t.object = &gcs.MinObject{
		Name:       "foo",
		Size:       1024 * MiB,
		Generation: 1234,
	}
	t.bucket = new(storage.TestifyMockBucket)
	t.cache = lru.NewCache(2) // Small cache size for testing eviction
	t.inodeID = 100
	t.mrdConfig = cfg.MrdConfig{PoolSize: 1}

	t.mrdInstance = NewMrdInstance(t.object, t.bucket, t.cache, t.inodeID, t.mrdConfig)
}

func (t *MrdInstanceTest) TestNewMrdInstance() {
	assert.Equal(t.T(), t.object, t.mrdInstance.object)
	assert.Equal(t.T(), t.bucket, t.mrdInstance.bucket)
	assert.Equal(t.T(), t.cache, t.mrdInstance.mrdCache)
	assert.Equal(t.T(), t.inodeID, t.mrdInstance.inodeId)
	assert.Equal(t.T(), t.mrdConfig, t.mrdInstance.mrdConfig)
	assert.Nil(t.T(), t.mrdInstance.mrdPool)
	assert.Equal(t.T(), int64(0), t.mrdInstance.refCount)
}

func (t *MrdInstanceTest) TestEnsureMrdInstance() {
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	assert.Nil(t.T(), t.mrdInstance.mrdPool)

	err := t.mrdInstance.EnsureMrdInstance()

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), t.mrdInstance.mrdPool)
	t.bucket.AssertExpectations(t.T())
}

func (t *MrdInstanceTest) TestEnsureMrdInstance_AlreadyExists() {
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	err := t.mrdInstance.EnsureMrdInstance()
	assert.NoError(t.T(), err)
	pool := t.mrdInstance.mrdPool

	// Call again
	err = t.mrdInstance.EnsureMrdInstance()

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), pool, t.mrdInstance.mrdPool)
	t.bucket.AssertExpectations(t.T()) // Should only be called once
}

func (t *MrdInstanceTest) TestGetMRDEntry() {
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	err := t.mrdInstance.EnsureMrdInstance()
	assert.NoError(t.T(), err)

	entry := t.mrdInstance.GetMRDEntry()

	assert.NotNil(t.T(), entry)
	assert.Equal(t.T(), fakeMRD, entry.mrd)
}

func (t *MrdInstanceTest) TestGetMRDEntry_NilPool() {
	entry := t.mrdInstance.GetMRDEntry()

	assert.Nil(t.T(), entry)
}

func (t *MrdInstanceTest) TestRecreateMRDEntry() {
	fakeMRD1 := fake.NewFakeMultiRangeDownloader(t.object, nil)
	fakeMRD2 := fake.NewFakeMultiRangeDownloader(t.object, nil)
	// Initial creation
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD1, nil).Once()
	err := t.mrdInstance.EnsureMrdInstance()
	assert.NoError(t.T(), err)
	entry := t.mrdInstance.GetMRDEntry()
	assert.Equal(t.T(), fakeMRD1, entry.mrd)

	// Recreate
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD2, nil).Once()
	t.mrdInstance.RecreateMRDEntry(entry)

	assert.Equal(t.T(), fakeMRD2, entry.mrd)
}

func (t *MrdInstanceTest) TestRecreateMRD() {
	fakeMRD1 := fake.NewFakeMultiRangeDownloader(t.object, nil)
	fakeMRD2 := fake.NewFakeMultiRangeDownloader(t.object, nil)
	// Initial creation
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD1, nil).Once()
	err := t.mrdInstance.EnsureMrdInstance()
	assert.NoError(t.T(), err)
	pool1 := t.mrdInstance.mrdPool

	// Recreate
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD2, nil).Once()
	t.mrdInstance.RecreateMRD(t.object)

	assert.NotNil(t.T(), t.mrdInstance.mrdPool)
	assert.NotEqual(t.T(), pool1, t.mrdInstance.mrdPool)
}

func (t *MrdInstanceTest) TestDestroy() {
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	t.mrdInstance.EnsureMrdInstance()
	assert.NotNil(t.T(), t.mrdInstance.mrdPool)

	t.mrdInstance.Destroy()

	assert.Nil(t.T(), t.mrdInstance.mrdPool)
}

func (t *MrdInstanceTest) TestIncrementRefCount() {
	// Setup: Put something in cache first to verify removal
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	err := t.mrdInstance.EnsureMrdInstance()
	assert.NoError(t.T(), err)
	// Manually insert into cache to simulate it being inactive
	key := strconv.FormatUint(uint64(t.inodeID), 10)
	_, err = t.cache.Insert(key, t.mrdInstance)
	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), t.cache.LookUpWithoutChangingOrder(key))

	t.mrdInstance.IncrementRefCount()

	assert.Equal(t.T(), int64(1), t.mrdInstance.refCount)
	assert.Nil(t.T(), t.cache.LookUpWithoutChangingOrder(key))
}

func (t *MrdInstanceTest) TestDecrementRefCount() {
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	err := t.mrdInstance.EnsureMrdInstance()
	assert.NoError(t.T(), err)
	t.mrdInstance.refCount = 1

	t.mrdInstance.DecrementRefCount()

	assert.Equal(t.T(), int64(0), t.mrdInstance.refCount)
	key := strconv.FormatUint(uint64(t.inodeID), 10)
	assert.NotNil(t.T(), t.cache.LookUpWithoutChangingOrder(key))
}

func (t *MrdInstanceTest) TestDecrementRefCount_Eviction() {
	// Fill cache with other items
	localMrdInstance := &MrdInstance{mrdPool: &MRDPool{}}
	localMrdInstance.mrdPool.currentSize.Store(1)
	_, err := t.cache.Insert("other1", localMrdInstance)
	assert.NoError(t.T(), err)
	_, err = t.cache.Insert("other2", localMrdInstance)
	assert.NoError(t.T(), err)
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	err = t.mrdInstance.EnsureMrdInstance()
	assert.NoError(t.T(), err)
	t.mrdInstance.refCount = 1

	// This should trigger eviction of "other1" (LRU)
	t.mrdInstance.DecrementRefCount()

	assert.Equal(t.T(), int64(0), t.mrdInstance.refCount)
	key := strconv.FormatUint(uint64(t.inodeID), 10)
	assert.NotNil(t.T(), t.cache.LookUpWithoutChangingOrder(key))
	assert.Nil(t.T(), t.cache.LookUpWithoutChangingOrder("other1"))
	assert.NotNil(t.T(), t.cache.LookUpWithoutChangingOrder("other2"))
}
