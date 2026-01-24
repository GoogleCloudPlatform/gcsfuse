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
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
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
	config      *cfg.Config
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
	t.config = &cfg.Config{Mrd: cfg.MrdConfig{PoolSize: 1}}

	t.mrdInstance = NewMrdInstance(t.object, t.bucket, t.cache, t.inodeID, t.config)
}

func (t *MrdInstanceTest) TestNewMrdInstance() {
	assert.Equal(t.T(), t.object, t.mrdInstance.object)
	assert.Equal(t.T(), t.bucket, t.mrdInstance.bucket)
	assert.Equal(t.T(), t.cache, t.mrdInstance.mrdCache)
	assert.Equal(t.T(), t.inodeID, t.mrdInstance.inodeId)
	assert.Equal(t.T(), t.config, t.mrdInstance.config)
	assert.Nil(t.T(), t.mrdInstance.mrdPool)
	assert.Equal(t.T(), int64(0), t.mrdInstance.refCount)
}

func (t *MrdInstanceTest) TestRead_Success() {
	data := []byte("hello world")
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, data)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	buf := make([]byte, 5)

	n, err := t.mrdInstance.Read(context.Background(), buf, 0, metrics.NewNoopMetrics())

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 5, n)
	assert.Equal(t.T(), "hello", string(buf))
}

func (t *MrdInstanceTest) TestRead_InitializesPool() {
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	assert.Nil(t.T(), t.mrdInstance.mrdPool)
	buf := make([]byte, 1)

	_, err := t.mrdInstance.Read(context.Background(), buf, 0, metrics.NewNoopMetrics())

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), t.mrdInstance.mrdPool)
}

func (t *MrdInstanceTest) TestRead_RecreatesInvalidEntry() {
	fakeMRD1 := fake.NewFakeMultiRangeDownloader(t.object, nil)
	fakeMRD2 := fake.NewFakeMultiRangeDownloader(t.object, nil)
	// Initial creation
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD1, nil).Once()
	buf := make([]byte, 1)
	_, err := t.mrdInstance.Read(context.Background(), buf, 0, metrics.NewNoopMetrics())
	assert.NoError(t.T(), err)

	// Manually invalidate the entry to simulate a failure
	entry := &t.mrdInstance.mrdPool.entries[0]
	entry.mrd.Close() // Close it.
	// Replace the entry's MRD with one that returns an error.
	entry.mu.Lock()
	entry.mrd = fake.NewFakeMultiRangeDownloaderWithStatusError(t.object, nil, fmt.Errorf("broken"))
	entry.mu.Unlock()

	// Expect recreation
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD2, nil).Once()

	_, err = t.mrdInstance.Read(context.Background(), buf, 0, metrics.NewNoopMetrics())

	assert.NoError(t.T(), err)
	entry.mu.RLock()
	assert.Equal(t.T(), fakeMRD2, entry.mrd)
	entry.mu.RUnlock()
}

func (t *MrdInstanceTest) TestRead_EnsureFails() {
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("init error")).Once()
	assert.Nil(t.T(), t.mrdInstance.mrdPool)
	buf := make([]byte, 1)

	n, err := t.mrdInstance.Read(context.Background(), buf, 0, metrics.NewNoopMetrics())

	assert.Error(t.T(), err)
	assert.Contains(t.T(), err.Error(), "init error")
	assert.Equal(t.T(), 0, n)
}

func (t *MrdInstanceTest) TestRead_RecreationFails() {
	fakeMRD1 := fake.NewFakeMultiRangeDownloader(t.object, nil)
	// Initial creation.
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD1, nil).Once()
	buf := make([]byte, 1)
	_, err := t.mrdInstance.Read(context.Background(), buf, 0, metrics.NewNoopMetrics())
	assert.NoError(t.T(), err)

	// Manually invalidate the entry to simulate a failure.
	entry := &t.mrdInstance.mrdPool.entries[0]
	entry.mrd.Close() // Close it.
	// Replace the entry's MRD with one that returns an error.
	entry.mu.Lock()
	entry.mrd = fake.NewFakeMultiRangeDownloaderWithStatusError(t.object, nil, fmt.Errorf("broken"))
	entry.mu.Unlock()

	// Expect recreation failure.
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("recreate error")).Once()

	n, err := t.mrdInstance.Read(context.Background(), buf, 0, metrics.NewNoopMetrics())

	assert.Error(t.T(), err)
	assert.Contains(t.T(), err.Error(), "recreate error")
	assert.Equal(t.T(), 0, n)
}

func (t *MrdInstanceTest) TestRead_EmptyBuffer() {
	n, err := t.mrdInstance.Read(context.Background(), []byte{}, 0, metrics.NewNoopMetrics())

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 0, n)
}

func (t *MrdInstanceTest) TestRead_ContextCancelled() {
	data := []byte("hello world")
	fakeMRD := fake.NewFakeMultiRangeDownloaderWithSleep(t.object, data, 100*time.Millisecond)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	ctx, cancel := context.WithCancel(context.Background())
	buf := make([]byte, 5)

	cancel()
	n, err := t.mrdInstance.Read(ctx, buf, 0, metrics.NewNoopMetrics())

	assert.Error(t.T(), err)
	assert.Equal(t.T(), context.Canceled, err)
	assert.Equal(t.T(), 0, n)
}

func (t *MrdInstanceTest) TestRead_AddError() {
	fakeMRD := fake.NewFakeMultiRangeDownloaderWithSleepAndDefaultError(t.object, []byte("data"), 0, fmt.Errorf("read error"))
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	buf := make([]byte, 5)

	n, err := t.mrdInstance.Read(context.Background(), buf, 0, metrics.NewNoopMetrics())

	assert.Error(t.T(), err)
	assert.Contains(t.T(), err.Error(), "read error")
	assert.Equal(t.T(), 0, n)
}

func (t *MrdInstanceTest) TestGetMRDEntry_Success() {
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()

	entry, err := t.mrdInstance.getMRDEntry()

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), entry)
	assert.Equal(t.T(), fakeMRD, entry.mrd)
}

func (t *MrdInstanceTest) TestGetMRDEntry_EnsureFails() {
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("init error")).Once()

	entry, err := t.mrdInstance.getMRDEntry()

	assert.Error(t.T(), err)
	assert.Nil(t.T(), entry)
	assert.Contains(t.T(), err.Error(), "init error")
}

func (t *MrdInstanceTest) TestGetMRDEntry_RecreatesInvalidMRD() {
	fakeMRD1 := fake.NewFakeMultiRangeDownloader(t.object, nil)
	fakeMRD2 := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD1, nil).Once()
	entry1, err := t.mrdInstance.getMRDEntry()
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), fakeMRD1, entry1.mrd)
	entry1.mrd.Close()
	entry1.mu.Lock()
	entry1.mrd = fake.NewFakeMultiRangeDownloaderWithStatusError(t.object, nil, fmt.Errorf("broken"))
	entry1.mu.Unlock()
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD2, nil).Once()

	// This should force recreation of the MRD.
	entry2, err := t.mrdInstance.getMRDEntry()

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), fakeMRD2, entry2.mrd)
}

func (t *MrdInstanceTest) TestRecreateMRD() {
	fakeMRD1 := fake.NewFakeMultiRangeDownloader(t.object, nil)
	fakeMRD2 := fake.NewFakeMultiRangeDownloader(t.object, nil)
	// Initial creation
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD1, nil).Once()
	buf := make([]byte, 1)
	_, err := t.mrdInstance.Read(context.Background(), buf, 0, metrics.NewNoopMetrics())
	assert.NoError(t.T(), err)
	pool1 := t.mrdInstance.mrdPool
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD2, nil).Once()

	// Recreate
	err = t.mrdInstance.RecreateMRD()

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), t.mrdInstance.mrdPool)
	assert.NotSame(t.T(), pool1, t.mrdInstance.mrdPool)
}

func (t *MrdInstanceTest) TestDestroy() {
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	buf := make([]byte, 1)
	_, err := t.mrdInstance.Read(context.Background(), buf, 0, metrics.NewNoopMetrics())
	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), t.mrdInstance.mrdPool)

	t.mrdInstance.Destroy()

	assert.Nil(t.T(), t.mrdInstance.mrdPool)
}

func (t *MrdInstanceTest) TestIncrementRefCount() {
	// Setup: Put something in cache first to verify removal
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	buf := make([]byte, 1)
	_, err := t.mrdInstance.Read(context.Background(), buf, 0, metrics.NewNoopMetrics())
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
	buf := make([]byte, 1)
	_, err := t.mrdInstance.Read(context.Background(), buf, 0, metrics.NewNoopMetrics())
	assert.NoError(t.T(), err)
	t.mrdInstance.refCount = 1

	t.mrdInstance.DecrementRefCount()

	assert.Equal(t.T(), int64(0), t.mrdInstance.refCount)
	key := strconv.FormatUint(uint64(t.inodeID), 10)
	assert.NotNil(t.T(), t.cache.LookUpWithoutChangingOrder(key))
}

func (t *MrdInstanceTest) TestDecrementRefCount_Eviction() {
	// Fill cache with other items
	localMrdInstance := &MrdInstance{mrdPool: &MRDPool{poolConfig: &MRDPoolConfig{PoolSize: 1}}}
	_, err := t.cache.Insert("other1", localMrdInstance)
	assert.NoError(t.T(), err)
	_, err = t.cache.Insert("other2", localMrdInstance)
	assert.NoError(t.T(), err)
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	buf := make([]byte, 1)
	_, err = t.mrdInstance.Read(context.Background(), buf, 0, metrics.NewNoopMetrics())
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

func (t *MrdInstanceTest) TestGetKey() {
	testCases := []struct {
		inodeID  fuseops.InodeID
		expected string
	}{
		{0, "0"},
		{123, "123"},
		{18446744073709551615, "18446744073709551615"}, // Max uint64
	}

	for _, tc := range testCases {
		assert.Equal(t.T(), tc.expected, getKey(tc.inodeID))
	}
}

func (t *MrdInstanceTest) TestEnsureMRDPool_Success() {
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	assert.Nil(t.T(), t.mrdInstance.mrdPool)

	err := t.mrdInstance.ensureMRDPool()

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), t.mrdInstance.mrdPool)
}

func (t *MrdInstanceTest) TestEnsureMRDPool_AlreadyExists() {
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	err := t.mrdInstance.ensureMRDPool()
	assert.NoError(t.T(), err)
	pool := t.mrdInstance.mrdPool

	// Call again
	err = t.mrdInstance.ensureMRDPool()

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), pool, t.mrdInstance.mrdPool)
	t.bucket.AssertExpectations(t.T()) // Should only be called once
}

func (t *MrdInstanceTest) TestEnsureMRDPool_Failure() {
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("init error")).Once()
	assert.Nil(t.T(), t.mrdInstance.mrdPool)

	err := t.mrdInstance.ensureMRDPool()

	assert.Error(t.T(), err)
	assert.Nil(t.T(), t.mrdInstance.mrdPool)
	assert.Contains(t.T(), err.Error(), "init error")
}

func (t *MrdInstanceTest) TestSize() {
	assert.Equal(t.T(), uint64(0), t.mrdInstance.Size())
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	buf := make([]byte, 1)
	_, err := t.mrdInstance.Read(context.Background(), buf, 0, metrics.NewNoopMetrics())
	assert.NoError(t.T(), err)

	poolSize := t.mrdInstance.Size()

	// Pool size is 1 based on SetupTest config (PoolSize: 1)
	assert.Equal(t.T(), uint64(1), poolSize)
	t.mrdInstance.Destroy()
	assert.Equal(t.T(), uint64(0), t.mrdInstance.Size())
}

func (t *MrdInstanceTest) TestDestroy_RemovesFromCache() {
	// Manually insert into cache
	key := getKey(t.inodeID)
	_, err := t.cache.Insert(key, t.mrdInstance)
	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), t.cache.LookUpWithoutChangingOrder(key))

	t.mrdInstance.Destroy()

	assert.Nil(t.T(), t.cache.LookUpWithoutChangingOrder(key))
}

func (t *MrdInstanceTest) TestDestroy_WithRefCount() {
	t.mrdInstance.refCount = 1
	// Should log warning but proceed to destroy
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	err := t.mrdInstance.ensureMRDPool()
	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), t.mrdInstance.mrdPool)
	// Capture logs to verify error message
	var buf bytes.Buffer
	logger.SetOutput(&buf)
	defer logger.SetOutput(os.Stdout)

	t.mrdInstance.Destroy()

	assert.Nil(t.T(), t.mrdInstance.mrdPool)
	assert.Contains(t.T(), buf.String(), "MrdInstance::Destroy called on an instance with refCount 1")
}

func (t *MrdInstanceTest) TestDecrementRefCount_Negative() {
	t.mrdInstance.refCount = 0
	// Capture logs to verify error message
	var buf bytes.Buffer
	logger.SetOutput(&buf)
	defer logger.SetOutput(os.Stdout)

	// Should log error and return, not panic. RefCount should remain 0.
	t.mrdInstance.DecrementRefCount()

	assert.Equal(t.T(), int64(0), t.mrdInstance.refCount)
	assert.Contains(t.T(), buf.String(), "MrdInstance::DecrementRefCount: Refcount cannot be negative")
}

func (t *MrdInstanceTest) TestDecrementRefCount_CacheInsertFailure() {
	// Create a cache with capacity 1
	smallCache := lru.NewCache(1)
	// Create instance with pool size 2 (so Size() returns 2).
	config := &cfg.Config{Mrd: cfg.MrdConfig{PoolSize: 2}}
	mi := NewMrdInstance(t.object, t.bucket, smallCache, t.inodeID, config)
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil)
	// Initialize pool.
	err := mi.ensureMRDPool()
	assert.NoError(t.T(), err)
	mi.refCount = 1
	// Capture logs to verify error message
	var buf bytes.Buffer
	logger.SetOutput(&buf)
	defer logger.SetOutput(os.Stdout)

	// This should fail to insert into cache (Size 2 > Cap 1) and should close the pool instantly.
	mi.DecrementRefCount()

	assert.Equal(t.T(), int64(0), mi.refCount)
	mi.poolMu.RLock()
	assert.Nil(t.T(), mi.mrdPool)
	mi.poolMu.RUnlock()
	assert.Contains(t.T(), buf.String(), "Failed to insert MrdInstance")
}

func (t *MrdInstanceTest) TestClosePool() {
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	err := t.mrdInstance.ensureMRDPool()
	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), t.mrdInstance.mrdPool)

	t.mrdInstance.closePool()

	t.mrdInstance.poolMu.RLock()
	assert.Nil(t.T(), t.mrdInstance.mrdPool)
	t.mrdInstance.poolMu.RUnlock()
}

func (t *MrdInstanceTest) TestHandleEviction_Resurrected() {
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	// Initialize pool.
	err := t.mrdInstance.ensureMRDPool()
	assert.NoError(t.T(), err)
	// Simulate resurrection (refCount > 0).
	t.mrdInstance.refCount = 1

	t.mrdInstance.handleEviction()

	// Pool should still exist because refCount > 0.
	t.mrdInstance.poolMu.RLock()
	assert.NotNil(t.T(), t.mrdInstance.mrdPool)
	t.mrdInstance.poolMu.RUnlock()
}

func (t *MrdInstanceTest) TestHandleEviction_ReAddedToCache() {
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	// Initialize pool.
	err := t.mrdInstance.ensureMRDPool()
	assert.NoError(t.T(), err)
	// Add to cache to simulate it being re-added concurrently.
	key := getKey(t.inodeID)
	_, err = t.cache.Insert(key, t.mrdInstance)
	assert.NoError(t.T(), err)
	// refCount is 0, but it is in the cache.
	t.mrdInstance.refCount = 0

	t.mrdInstance.handleEviction()

	// Pool should still exist because it's in the cache.
	t.mrdInstance.poolMu.RLock()
	assert.NotNil(t.T(), t.mrdInstance.mrdPool)
	t.mrdInstance.poolMu.RUnlock()
}

func (t *MrdInstanceTest) TestHandleEviction_SafeToClose() {
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	// Initialize pool.
	err := t.mrdInstance.ensureMRDPool()
	assert.NoError(t.T(), err)
	// Ensure not in cache.
	key := getKey(t.inodeID)
	t.cache.Erase(key)
	// refCount is 0.
	t.mrdInstance.refCount = 0

	t.mrdInstance.handleEviction()

	// Pool should be closed (nil).
	t.mrdInstance.poolMu.RLock()
	assert.Nil(t.T(), t.mrdInstance.mrdPool)
	t.mrdInstance.poolMu.RUnlock()
}

func (t *MrdInstanceTest) TestCreateAndSwapPool_Success() {
	// Setup initial state
	initialObj := &gcs.MinObject{Name: "old", Generation: 1}
	t.mrdInstance.object = initialObj
	// Create an initial pool
	fakeMRD1 := fake.NewFakeMultiRangeDownloader(initialObj, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD1, nil).Once()
	err := t.mrdInstance.ensureMRDPool()
	assert.NoError(t.T(), err)
	oldPool := t.mrdInstance.mrdPool
	assert.NotNil(t.T(), oldPool)
	// Prepare for new pool creation
	newObj := &gcs.MinObject{Name: "new", Generation: 2}
	fakeMRD2 := fake.NewFakeMultiRangeDownloader(newObj, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD2, nil).Once()

	// Call createAndSwapPool
	t.mrdInstance.poolMu.Lock()
	err = t.mrdInstance.createAndSwapPool(newObj)
	t.mrdInstance.poolMu.Unlock()

	assert.NoError(t.T(), err)
	assert.NotSame(t.T(), oldPool, t.mrdInstance.mrdPool)
	assert.Equal(t.T(), newObj, t.mrdInstance.object)
	assert.Equal(t.T(), fakeMRD2, t.mrdInstance.mrdPool.entries[0].mrd)
}

func (t *MrdInstanceTest) TestCreateAndSwapPool_Failure() {
	// Setup initial state
	initialObj := &gcs.MinObject{Name: "old", Generation: 1}
	t.mrdInstance.object = initialObj
	// Create an initial pool
	fakeMRD1 := fake.NewFakeMultiRangeDownloader(initialObj, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD1, nil).Once()
	err := t.mrdInstance.ensureMRDPool()
	assert.NoError(t.T(), err)
	oldPool := t.mrdInstance.mrdPool
	assert.NotNil(t.T(), oldPool)
	// Prepare for new pool creation failure
	newObj := &gcs.MinObject{Name: "new", Generation: 2}
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("creation failed")).Once()

	// Call createAndSwapPool
	t.mrdInstance.poolMu.Lock()
	err = t.mrdInstance.createAndSwapPool(newObj)
	t.mrdInstance.poolMu.Unlock()

	assert.Error(t.T(), err)
	assert.Contains(t.T(), err.Error(), "creation failed")
	// Verify state remains unchanged
	assert.Equal(t.T(), oldPool, t.mrdInstance.mrdPool)
	assert.Equal(t.T(), initialObj, t.mrdInstance.object)
}

func (t *MrdInstanceTest) TestSetMinObject_NilObject() {
	err := t.mrdInstance.SetMinObject(nil)

	assert.Error(t.T(), err)

	assert.Contains(t.T(), err.Error(), "Missing MinObject")
}

func (t *MrdInstanceTest) TestSetMinObject_SameGeneration() {
	// Setup
	initialObj := t.mrdInstance.GetMinObject()
	// Ensure pool exists.
	fakeMRD1 := fake.NewFakeMultiRangeDownloader(initialObj, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD1, nil).Once()
	err := t.mrdInstance.ensureMRDPool()
	assert.NoError(t.T(), err)
	t.mrdInstance.poolMu.RLock()
	initialPool := t.mrdInstance.mrdPool
	t.mrdInstance.poolMu.RUnlock()
	// Same generation update (e.g. size change).
	newObj := &gcs.MinObject{
		Name:       initialObj.Name,
		Generation: initialObj.Generation,
		Size:       initialObj.Size + 100,
	}

	err = t.mrdInstance.SetMinObject(newObj)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), newObj, t.mrdInstance.GetMinObject())
	// Pool should not change for same generation.
	t.mrdInstance.poolMu.RLock()
	assert.Equal(t.T(), initialPool, t.mrdInstance.mrdPool)
	t.mrdInstance.poolMu.RUnlock()
}

func (t *MrdInstanceTest) TestSetMinObject_DifferentGeneration() {
	// Setup
	initialObj := t.mrdInstance.GetMinObject()
	// Ensure pool exists.
	fakeMRD1 := fake.NewFakeMultiRangeDownloader(initialObj, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD1, nil).Once()
	err := t.mrdInstance.ensureMRDPool()
	assert.NoError(t.T(), err)
	t.mrdInstance.poolMu.RLock()
	initialPool := t.mrdInstance.mrdPool
	t.mrdInstance.poolMu.RUnlock()
	// New generation
	newObj := &gcs.MinObject{
		Name:       initialObj.Name,
		Generation: initialObj.Generation + 1,
		Size:       initialObj.Size,
	}
	// Mock creation of new pool
	fakeMRD2 := fake.NewFakeMultiRangeDownloader(newObj, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD2, nil).Once()

	err = t.mrdInstance.SetMinObject(newObj)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), newObj, t.mrdInstance.GetMinObject())
	t.mrdInstance.poolMu.RLock()
	assert.NotSame(t.T(), initialPool, t.mrdInstance.mrdPool)
	assert.NotNil(t.T(), t.mrdInstance.mrdPool)
	t.mrdInstance.poolMu.RUnlock()
}

func (t *MrdInstanceTest) TestGetMinObject() {
	obj := t.mrdInstance.GetMinObject()

	assert.Equal(t.T(), t.object, obj)
}

func (t *MrdInstanceTest) TestClosePoolWithTimeout_LogWarningOnTimeout() {
	// 1. Capture logs.
	var buf logBuffer
	logger.SetOutput(&buf)
	defer logger.SetOutput(os.Stdout)
	// 2. Create a pool that blocks on Close().
	// MRDPool.Close() waits on creationWg. We increment it to block Close().
	pool := &MRDPool{
		poolConfig: &MRDPoolConfig{
			object: t.object,
		},
	}
	pool.creationWg.Add(1)

	// 3. Call the function.
	closePoolWithTimeout(pool, "TestCaller", 10*time.Millisecond)

	// 4. Wait enough time for timeout to trigger.
	time.Sleep(50 * time.Millisecond)
	// 5. Verify log.
	assert.Contains(t.T(), buf.String(), "TestCaller: MRDPool.Close() timed out")
	assert.Contains(t.T(), buf.String(), t.object.Name)
	// 7. Cleanup: Unblock the pool closure to avoid goroutine leak.
	pool.creationWg.Done()
}

// logBuffer is a thread-safe buffer for capturing logs in tests.
type logBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *logBuffer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *logBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func (t *MrdInstanceTest) Test_Cache_AddAndRemove() {
	key := getKey(t.inodeID)
	// Setup: Ensure pool is created
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	err := t.mrdInstance.ensureMRDPool()
	assert.NoError(t.T(), err)

	// Act: Open, close, and reopen file
	t.mrdInstance.IncrementRefCount()
	t.mrdInstance.DecrementRefCount()
	assert.NotNil(t.T(), t.cache.LookUpWithoutChangingOrder(key), "Instance should be in cache.")
	t.mrdInstance.IncrementRefCount()

	// Assert: Instance reused and removed from cache on reopen
	assert.Equal(t.T(), int64(1), t.mrdInstance.refCount)
	assert.Nil(t.T(), t.cache.LookUpWithoutChangingOrder(key), "Instance should be removed from cache")
	t.mrdInstance.poolMu.RLock()
	assert.NotNil(t.T(), t.mrdInstance.mrdPool, "MRD Pool should still exist (reused)")
	t.mrdInstance.poolMu.RUnlock()
}

func (t *MrdInstanceTest) Test_DecrementRefCount_ParallelUpdates() {
	// Arrange
	const finalRefCount int64 = 0
	maxRefCount := 10
	wg := sync.WaitGroup{}
	key := getKey(t.inodeID)
	// Ensure pool exists so it can be cached
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	err := t.mrdInstance.ensureMRDPool()
	assert.NoError(t.T(), err)

	// Act: Increment refcount in parallel
	for i := 0; i < maxRefCount; i++ {
		wg.Add(1)
		go func() {
			t.mrdInstance.IncrementRefCount()
			wg.Done()
		}()
	}
	wg.Wait()
	// Act: Decrement refcount in parallel
	for i := 0; i < maxRefCount; i++ {
		wg.Add(1)
		go func() {
			t.mrdInstance.DecrementRefCount()
			wg.Done()
		}()
	}
	wg.Wait()

	// Assert: Final state is refCount=0, MRD pooled in cache
	assert.Equal(t.T(), finalRefCount, t.mrdInstance.refCount)
	t.mrdInstance.poolMu.RLock()
	assert.NotNil(t.T(), t.mrdInstance.mrdPool, "MRD Pool should be pooled in cache")
	t.mrdInstance.poolMu.RUnlock()
	assert.NotNil(t.T(), t.cache.LookUpWithoutChangingOrder(key), "Instance should be in cache")
}

func (t *MrdInstanceTest) Test_Cache_EvictionOnOverflow() {
	// Arrange: Create 3 instances (cache max is 2)
	instances := make([]*MrdInstance, 3)
	for i := 0; i < 3; i++ {
		obj := &gcs.MinObject{
			Name:       fmt.Sprintf("file%d", i),
			Size:       100,
			Generation: int64(1000 + i),
		}
		instance := NewMrdInstance(obj, t.bucket, t.cache, fuseops.InodeID(100+i), t.config)

		// Setup mock for pool creation
		fakeMRD := fake.NewFakeMultiRangeDownloader(obj, nil)
		t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
		err := instance.ensureMRDPool()
		assert.NoError(t.T(), err)

		instances[i] = instance
	}

	// Act: Open and close all 3 instances (triggers eviction on 3rd)
	for i := 0; i < 3; i++ {
		instances[i].IncrementRefCount()
		instances[i].DecrementRefCount()
	}

	// Assert: First instance evicted (LRU), last 2 remain in cache
	instances[0].poolMu.RLock()
	assert.Nil(t.T(), instances[0].mrdPool, "First instance's MRD Pool should be closed (evicted)")
	instances[0].poolMu.RUnlock()
	for i := 1; i < 3; i++ {
		key := getKey(instances[i].inodeId)
		assert.NotNil(t.T(), t.cache.LookUpWithoutChangingOrder(key), "Instance %d should be in cache", i)
		instances[i].poolMu.RLock()
		assert.NotNil(t.T(), instances[i].mrdPool, "Instance %d MRD Pool should exist (pooled)", i)
		instances[i].poolMu.RUnlock()
	}
}

func (t *MrdInstanceTest) Test_Cache_DeletedIfReopened() {
	// Arrange: Create 2 instances and fill cache (size 2)
	instances := make([]*MrdInstance, 2)
	for i := 0; i < 2; i++ {
		obj := &gcs.MinObject{
			Name:       fmt.Sprintf("file%d", i),
			Size:       100,
			Generation: int64(1000 + i),
		}
		instance := NewMrdInstance(obj, t.bucket, t.cache, fuseops.InodeID(100+i), t.config)

		fakeMRD := fake.NewFakeMultiRangeDownloader(obj, nil)
		t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
		err := instance.ensureMRDPool()
		assert.NoError(t.T(), err)

		instance.IncrementRefCount()
		instance.DecrementRefCount()
		instances[i] = instance
	}

	// Act: Reopen instance 0 -> should remove it from cache
	instances[0].IncrementRefCount()

	// Assert: instance 0 will be deleted from cache.
	assert.Nil(t.T(), t.cache.LookUpWithoutChangingOrder(getKey(instances[0].inodeId)), "Instance 0 should not be in cache")
}

func (t *MrdInstanceTest) Test_Cache_ConcurrentAddRemove() {
	// Arrange
	const numGoroutines = 10
	const numIterations = 100
	wg := sync.WaitGroup{}
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	err := t.mrdInstance.ensureMRDPool()
	assert.NoError(t.T(), err)

	// Act: Concurrent open/close cycles from multiple goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				t.mrdInstance.IncrementRefCount()
				t.mrdInstance.DecrementRefCount()
			}
		}()
	}
	wg.Wait()

	// Assert: Final state is refCount=0 (no deadlocks or panics)
	assert.Equal(t.T(), int64(0), t.mrdInstance.refCount, "RefCount should be 0 after all operations")
	assert.NotNil(t.T(), t.cache.LookUpWithoutChangingOrder(getKey(t.inodeID)), "Instance should be in cache")
}

func (t *MrdInstanceTest) Test_Cache_Disabled() {
	// Arrange: Create instance with nil cache (disabled)
	instance := NewMrdInstance(t.object, t.bucket, nil, t.inodeID, t.config)
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	err := instance.ensureMRDPool()
	assert.NoError(t.T(), err)

	// Act: Open and close file
	instance.IncrementRefCount()
	instance.DecrementRefCount()

	// Assert: MRD Pool should be open forever since cache is disabled.
	instance.poolMu.RLock()
	assert.NotNil(t.T(), instance.mrdPool, "MRD Pool should be open when cache disabled")
	instance.poolMu.RUnlock()
}

func (t *MrdInstanceTest) Test_Cache_EvictionRaceWithRepool() {
	// Arrange: Add instance to cache then fill with 2 more to trigger eviction (cache size 2)
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	err := t.mrdInstance.ensureMRDPool()
	assert.NoError(t.T(), err)
	t.mrdInstance.IncrementRefCount()
	t.mrdInstance.DecrementRefCount()
	assert.NoError(t.T(), err)
	for i := 0; i < 2; i++ {
		obj := &gcs.MinObject{
			Name:       fmt.Sprintf("file%d", i),
			Size:       100,
			Generation: int64(1000 + i),
		}
		instance := NewMrdInstance(obj, t.bucket, t.cache, fuseops.InodeID(200+i), t.config)

		fakeMRD := fake.NewFakeMultiRangeDownloader(obj, nil)
		t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
		err := instance.ensureMRDPool()
		assert.NoError(t.T(), err)

		instance.IncrementRefCount()
		instance.DecrementRefCount()
	}
	// Verify it was evicted
	t.mrdInstance.poolMu.RLock()
	assert.Nil(t.T(), t.mrdInstance.mrdPool)
	t.mrdInstance.poolMu.RUnlock()
	buf := make([]byte, 10)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(
		fake.NewFakeMultiRangeDownloader(t.object, nil),
		nil,
	).Once()

	// Act: Access evicted instance (should recreate MRD Pool). Read should recreate it
	_, err = t.mrdInstance.Read(context.Background(), buf, 0, metrics.NewNoopMetrics())

	// Assert: MRD Pool recreated successfully after eviction
	assert.NoError(t.T(), err)
	t.mrdInstance.poolMu.RLock()
	assert.NotNil(t.T(), t.mrdInstance.mrdPool, "MRD Pool should be recreated after eviction")
	t.mrdInstance.poolMu.RUnlock()
}

func (t *MrdInstanceTest) Test_Cache_MultipleEvictions() {
	// Arrange: Create small cache (size 2) and 5 instances
	smallCache := lru.NewCache(2)
	instances := make([]*MrdInstance, 5)
	for i := 0; i < 5; i++ {
		obj := &gcs.MinObject{
			Name:       fmt.Sprintf("file%d", i),
			Size:       100,
			Generation: int64(1000 + i),
		}
		instance := NewMrdInstance(obj, t.bucket, smallCache, fuseops.InodeID(300+i), t.config)

		fakeMRD := fake.NewFakeMultiRangeDownloader(obj, nil)
		t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
		err := instance.ensureMRDPool()
		assert.NoError(t.T(), err)

		instances[i] = instance
	}

	// Act: Add all 5 instances (triggers batch eviction of 3)
	for i := 0; i < 5; i++ {
		instances[i].IncrementRefCount()
		instances[i].DecrementRefCount()
	}

	// Assert: First 3 evicted, last 2 remain in cache
	for i := 0; i < 3; i++ {
		instances[i].poolMu.RLock()
		assert.Nil(t.T(), instances[i].mrdPool, "Instance %d should be evicted", i)
		instances[i].poolMu.RUnlock()
	}
	for i := 3; i < 5; i++ {
		instances[i].poolMu.RLock()
		assert.NotNil(t.T(), instances[i].mrdPool, "Instance %d should be in cache", i)
		instances[i].poolMu.RUnlock()
	}
}
