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
	assert.Equal(t.T(), t.object, t.mrdInstance.object.Load())
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
	assert.NotEqual(t.T(), pool1, t.mrdInstance.mrdPool)
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
