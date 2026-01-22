// Copyright 2025 Google LLC
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
	"context"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/gcsfuse_errors"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	testutil "github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type mrdWrapperTest struct {
	suite.Suite
	object     *gcs.MinObject
	objectData []byte
	mockBucket *storage.TestifyMockBucket
	mrdWrapper *MultiRangeDownloaderWrapper
}

func TestMRDWrapperTestSuite(t *testing.T) {
	suite.Run(t, new(mrdWrapperTest))
}

func (t *mrdWrapperTest) SetupTest() {
	var err error
	t.object = &gcs.MinObject{
		Name:       "foo",
		Size:       100,
		Generation: 1234,
	}
	t.objectData = testutil.GenerateRandomBytes(int(t.object.Size))
	// Create the bucket.
	t.mockBucket = new(storage.TestifyMockBucket)
	t.mrdWrapper, err = NewMultiRangeDownloaderWrapper(t.mockBucket, t.object, &cfg.Config{}, nil)
	assert.Nil(t.T(), err, "Error in creating MRDWrapper")
	t.mrdWrapper.Wrapped = fake.NewFakeMultiRangeDownloaderWithSleep(t.object, t.objectData, time.Microsecond)
	t.mrdWrapper.refCount = 0
}

func (t *mrdWrapperTest) Test_IncrementRefCount_ParallelUpdates() {
	const finalRefCount int = 1
	wg := sync.WaitGroup{}
	for range finalRefCount {
		wg.Add(1)
		go func() {
			t.mrdWrapper.IncrementRefCount()
			wg.Done()
		}()
	}
	wg.Wait()

	assert.Equal(t.T(), finalRefCount, t.mrdWrapper.refCount)
}

func (t *mrdWrapperTest) Test_DecrementRefCount_ParallelUpdates() {
	const finalRefCount int = 0
	maxRefCount := 10
	wg := sync.WaitGroup{}
	// Incrementing refcount in parallel.
	for range maxRefCount {
		wg.Add(1)
		go func() {
			t.mrdWrapper.IncrementRefCount()
			wg.Done()
		}()
	}
	wg.Wait()
	// Decrementing refcount in parallel.
	for range maxRefCount {
		wg.Add(1)
		go func() {
			err := t.mrdWrapper.DecrementRefCount()
			assert.Nil(t.T(), err)
			wg.Done()
		}()
	}
	wg.Wait()

	assert.Equal(t.T(), finalRefCount, t.mrdWrapper.GetRefCount())
	assert.NotNil(t.T(), t.mrdWrapper.Wrapped)
}

func (t *mrdWrapperTest) Test_DecrementRefCount_InvalidUse() {
	errMsg := "MultiRangeDownloaderWrapper DecrementRefCount: Refcount cannot be negative"
	assert.ErrorContains(t.T(), t.mrdWrapper.DecrementRefCount(), errMsg)
}

func (t *mrdWrapperTest) Test_Read() {
	testCases := []struct {
		name  string
		start int
		end   int
	}{
		{
			name:  "ReadFull",
			start: 0,
			end:   int(t.object.Size),
		},
		{
			name:  "ReadChunk",
			start: 10,
			end:   10 + int(t.object.Size)/2,
		},
		{
			name:  "ReadEmpty",
			start: 10,
			end:   10,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			buf := make([]byte, tc.end-tc.start)
			t.mrdWrapper.Wrapped = nil
			t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, t.objectData, time.Microsecond))

			bytesRead, err := t.mrdWrapper.Read(context.Background(), buf, int64(tc.start), int64(tc.end), metrics.NewNoopMetrics(), false)

			assert.NoError(t.T(), err)
			assert.Equal(t.T(), tc.end-tc.start, bytesRead)
			assert.Equal(t.T(), t.objectData[tc.start:tc.end], buf[:bytesRead])
		})
	}
}

func (t *mrdWrapperTest) Test_Read_ErrorInCreatingMRD() {
	t.mrdWrapper.Wrapped = nil
	t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("Error in creating MRD")).Once()

	bytesRead, err := t.mrdWrapper.Read(context.Background(), make([]byte, t.object.Size), 0, int64(t.object.Size), metrics.NewNoopMetrics(), false)

	assert.ErrorContains(t.T(), err, "MultiRangeDownloaderWrapper::Read: Error in creating MultiRangeDownloader")
	assert.Equal(t.T(), 0, bytesRead)
}

func (t *mrdWrapperTest) Test_Read_ShortRead() {
	t.mrdWrapper.Wrapped = nil
	// Configure the fake MRD to return a short read.
	fakeMRD := fake.NewFakeMultiRangeDownloaderWithShortRead(t.object, t.objectData)
	t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()

	bytesRead, err := t.mrdWrapper.Read(context.Background(), make([]byte, t.object.Size), 0, int64(t.object.Size), metrics.NewNoopMetrics(), false)

	assert.ErrorIs(t.T(), err, io.EOF)
	assert.Less(t.T(), bytesRead, int(t.object.Size))
}

func (t *mrdWrapperTest) TestReadContextCancelledWithInterruptsEnabled() {
	t.mrdWrapper.Wrapped = nil
	t.mrdWrapper.config = &cfg.Config{FileSystem: cfg.FileSystemConfig{IgnoreInterrupts: false}}
	t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, t.objectData, time.Microsecond), nil).Once()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	bytesRead, err := t.mrdWrapper.Read(ctx, make([]byte, t.object.Size), 0, int64(t.object.Size), metrics.NewNoopMetrics(), false)

	require.Error(t.T(), err)
	assert.ErrorContains(t.T(), err, "context canceled")
	assert.Equal(t.T(), 0, bytesRead)
}

func (t *mrdWrapperTest) TestReadContextCancelledWithInterruptsDisabled() {
	t.mrdWrapper.config = &cfg.Config{FileSystem: cfg.FileSystemConfig{IgnoreInterrupts: true}}
	t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, t.objectData, time.Microsecond), nil).Once()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	bytesRead, err := t.mrdWrapper.Read(ctx, make([]byte, t.object.Size), 0, int64(t.object.Size), metrics.NewNoopMetrics(), false)

	require.NoError(t.T(), err)
	assert.Equal(t.T(), 100, bytesRead)
}

func (t *mrdWrapperTest) Test_Read_EOF() {
	t.mrdWrapper.Wrapped = nil
	t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleepAndDefaultError(t.object, t.objectData, time.Microsecond, io.EOF), nil).Once()

	_, err := t.mrdWrapper.Read(context.Background(), make([]byte, t.object.Size), 0, int64(t.object.Size), metrics.NewNoopMetrics(), false)

	assert.ErrorIs(t.T(), err, io.EOF)
}

func (t *mrdWrapperTest) Test_Read_Error() {
	t.mrdWrapper.Wrapped = nil
	t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleepAndDefaultError(t.object, t.objectData, time.Microsecond, fmt.Errorf("Error")), nil).Once()

	bytesRead, err := t.mrdWrapper.Read(context.Background(), make([]byte, t.object.Size), 0, int64(t.object.Size), metrics.NewNoopMetrics(), false)

	assert.ErrorContains(t.T(), err, "error in Add call")
	assert.Equal(t.T(), 0, bytesRead)
}

func (t *mrdWrapperTest) Test_NewMultiRangeDownloaderWrapper() {
	testCases := []struct {
		name   string
		bucket gcs.Bucket
		obj    *gcs.MinObject
		err    error
	}{
		{
			name:   "ValidParameters",
			bucket: t.mockBucket,
			obj:    t.object,
			err:    nil,
		},
		{
			name:   "NilMinObject",
			bucket: t.mockBucket,
			obj:    nil,
			err:    fmt.Errorf("NewMultiRangeDownloaderWrapper: Missing MinObject"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			_, err := NewMultiRangeDownloaderWrapper(tc.bucket, tc.obj, &cfg.Config{}, nil)
			if tc.err == nil {
				assert.NoError(t.T(), err)
			} else {
				assert.Error(t.T(), err)
				assert.EqualError(t.T(), err, tc.err.Error())
			}
		})
	}
}

func (t *mrdWrapperTest) Test_SetMinObject() {
	testCases := []struct {
		name string
		obj  *gcs.MinObject
		err  error
	}{
		{
			name: "ValidMinObject",
			obj:  t.object,
			err:  nil,
		},
		{
			name: "NilMinObject",
			obj:  nil,
			err:  fmt.Errorf("MultiRangeDownloaderWrapper::SetMinObject: Missing MinObject"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			err := t.mrdWrapper.SetMinObject(tc.obj)
			if tc.err == nil {
				assert.NoError(t.T(), err)
			} else {
				assert.Error(t.T(), err)
				assert.EqualError(t.T(), err, tc.err.Error())
			}
		})
	}
}

func (t *mrdWrapperTest) Test_EnsureMultiRangeDownloader() {
	testCases := []struct {
		name   string
		obj    *gcs.MinObject
		bucket gcs.Bucket
		err    error
	}{
		{
			name:   "ValidMinObject",
			obj:    t.object,
			bucket: t.mockBucket,
			err:    nil,
		},
		{
			name:   "NilMinObject",
			obj:    nil,
			bucket: t.mockBucket,
			err:    fmt.Errorf("ensureMultiRangeDownloader error: Missing minObject or bucket"),
		},
		{
			name:   "NilBucket",
			obj:    t.object,
			bucket: nil,
			err:    fmt.Errorf("ensureMultiRangeDownloader error: Missing minObject or bucket"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.mrdWrapper.bucket = tc.bucket
			t.mrdWrapper.object = tc.obj
			t.mrdWrapper.Wrapped = nil
			t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, t.objectData, time.Microsecond))
			t.mrdWrapper.mu.RLock()
			defer t.mrdWrapper.mu.RUnlock()
			err := t.mrdWrapper.ensureMultiRangeDownloader(false)
			if tc.err == nil {
				assert.NoError(t.T(), err)
				assert.NotNil(t.T(), t.mrdWrapper.Wrapped)
			} else {
				assert.Error(t.T(), err)
				assert.EqualError(t.T(), err, tc.err.Error())
				assert.Nil(t.T(), t.mrdWrapper.Wrapped)
			}
		})
	}
}

func (t *mrdWrapperTest) Test_EnsureMultiRangeDownloader_UnusableExistingMRDTriggersRecreation() {
	t.mrdWrapper.bucket = t.mockBucket
	t.mrdWrapper.object = t.object
	t.mrdWrapper.Wrapped = fake.NewFakeMultiRangeDownloaderWithStatusError(t.object, t.objectData, fmt.Errorf("MRD is unusable..."))

	t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, t.objectData, time.Microsecond))
	t.mrdWrapper.mu.RLock()
	defer t.mrdWrapper.mu.RUnlock()

	err := t.mrdWrapper.ensureMultiRangeDownloader(false)

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), t.mrdWrapper.Wrapped)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *mrdWrapperTest) Test_EnsureMultiRangeDownloader_UsableExistingMRDPreventsRecreation() {
	t.mrdWrapper.bucket = t.mockBucket
	t.mrdWrapper.object = t.object
	t.mrdWrapper.Wrapped = fake.NewFakeMultiRangeDownloaderWithStatusError(t.object, t.objectData, nil)
	t.mrdWrapper.mu.RLock()
	defer t.mrdWrapper.mu.RUnlock()

	err := t.mrdWrapper.ensureMultiRangeDownloader(false)

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), t.mrdWrapper.Wrapped)
	t.mockBucket.AssertNotCalled(t.T(), "NewMultiRangeDownloader")
}

func (t *mrdWrapperTest) Test_EnsureMultiRangeDownloader_ForceRecreateMRD() {
	t.mrdWrapper.bucket = t.mockBucket
	t.mrdWrapper.object = t.object
	t.mrdWrapper.Wrapped = nil
	// First call to create an MRD.
	t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, t.objectData, time.Microsecond), nil).Once()
	t.mrdWrapper.mu.RLock()
	err := t.mrdWrapper.ensureMultiRangeDownloader(false)
	t.mrdWrapper.mu.RUnlock()
	require.NoError(t.T(), err)
	initialMRD := t.mrdWrapper.Wrapped
	require.NotNil(t.T(), initialMRD)

	// Second call with forceRecreateMRD=true should create a new MRD.
	t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, t.objectData, time.Microsecond), nil).Once()
	t.mrdWrapper.mu.RLock()
	err = t.mrdWrapper.ensureMultiRangeDownloader(true)
	t.mrdWrapper.mu.RUnlock()

	require.NoError(t.T(), err)
	assert.NotNil(t.T(), t.mrdWrapper.Wrapped)
	assert.NotSame(t.T(), initialMRD, t.mrdWrapper.Wrapped, "A new MRD instance should have been created")
	t.mockBucket.AssertExpectations(t.T())
}

func (t *mrdWrapperTest) Test_EnsureMultiRangeDownloader_FileClobbered() {
	t.mrdWrapper.Wrapped = nil
	notFoundErr := &gcs.NotFoundError{Err: fmt.Errorf("not found")}
	t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(nil, notFoundErr).Once()

	t.mrdWrapper.mu.RLock()
	defer t.mrdWrapper.mu.RUnlock()
	err := t.mrdWrapper.ensureMultiRangeDownloader(false)

	require.Error(t.T(), err)
	var clobberedErr *gcsfuse_errors.FileClobberedError
	assert.ErrorAs(t.T(), err, &clobberedErr)
	assert.Nil(t.T(), t.mrdWrapper.Wrapped)
}

// mrdWrapperCacheTest inherits from mrdWrapperTest and adds cache functionality.
type mrdWrapperCacheTest struct {
	mrdWrapperTest
	cache *lru.Cache
}

func TestMRDWrapperCacheTestSuite(t *testing.T) {
	suite.Run(t, new(mrdWrapperCacheTest))
}

func (t *mrdWrapperCacheTest) SetupTest() {
	t.mrdWrapperTest.SetupTest()

	// Recreate wrapper with cache enabled
	t.cache = lru.NewCache(3)
	var err error
	t.mrdWrapper, err = NewMultiRangeDownloaderWrapper(
		t.mockBucket,
		t.object,
		&cfg.Config{},
		t.cache,
	)
	assert.Nil(t.T(), err, "Error in creating MRDWrapper with cache")
	t.mrdWrapper.Wrapped = fake.NewFakeMultiRangeDownloaderWithSleep(t.object, t.objectData, time.Microsecond)
	t.mrdWrapper.refCount = 0
}

func (t *mrdWrapperCacheTest) Test_Cache_AddAndRemove() {
	key := wrapperKey(t.mrdWrapper)

	// Act: Open, close, and reopen file
	t.mrdWrapper.IncrementRefCount()
	err := t.mrdWrapper.DecrementRefCount()
	assert.NotNil(t.T(), t.cache.LookUpWithoutChangingOrder(key), "Wrapper should be in cache.")
	t.mrdWrapper.IncrementRefCount()

	// Assert: MRD reused and removed from cache on reopen
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 1, t.mrdWrapper.refCount)
	assert.Nil(t.T(), t.cache.LookUpWithoutChangingOrder(key), "Wrapper should be removed from cache")
	assert.NotNil(t.T(), t.mrdWrapper.Wrapped, "MRD should still exist (reused)")
}

// Override parent test - with cache enabled, MRD stays pooled
func (t *mrdWrapperCacheTest) Test_DecrementRefCount_ParallelUpdates() {
	// Arrange
	const finalRefCount int = 0
	maxRefCount := 10
	wg := sync.WaitGroup{}
	key := wrapperKey(t.mrdWrapper)

	// Act: Increment refcount in parallel
	for range maxRefCount {
		wg.Add(1)
		go func() {
			t.mrdWrapper.IncrementRefCount()
			wg.Done()
		}()
	}
	wg.Wait()

	// Act: Decrement refcount in parallel
	for range maxRefCount {
		wg.Add(1)
		go func() {
			err := t.mrdWrapper.DecrementRefCount()
			assert.Nil(t.T(), err)
			wg.Done()
		}()
	}
	wg.Wait()

	// Assert: Final state is refCount=0, MRD pooled in cache
	assert.Equal(t.T(), finalRefCount, t.mrdWrapper.GetRefCount())
	assert.NotNil(t.T(), t.mrdWrapper.Wrapped, "MRD should be pooled in cache")
	assert.NotNil(t.T(), t.cache.LookUpWithoutChangingOrder(key), "Wrapper should be in cache")
}

func (t *mrdWrapperCacheTest) Test_Cache_EvictionOnOverflow() {
	// Arrange: Create 4 wrappers (cache max is 3)
	wrappers := make([]*MultiRangeDownloaderWrapper, 4)
	for i := range 4 {
		obj := &gcs.MinObject{
			Name:       fmt.Sprintf("file%d", i),
			Size:       100,
			Generation: int64(1000 + i),
		}
		wrapper, err := NewMultiRangeDownloaderWrapper(
			t.mockBucket,
			obj,
			&cfg.Config{},
			t.cache,
		)
		assert.NoError(t.T(), err)
		wrapper.Wrapped = fake.NewFakeMultiRangeDownloaderWithSleep(obj, t.objectData, time.Microsecond)
		wrappers[i] = wrapper
	}

	// Act: Open and close all 4 wrappers (triggers eviction on 4th)
	for i := range 4 {
		wrappers[i].IncrementRefCount()
		err := wrappers[i].DecrementRefCount()
		assert.NoError(t.T(), err)
	}

	// Assert: First wrapper evicted (LRU), last 3 remain in cache
	assert.Nil(t.T(), wrappers[0].Wrapped, "First wrapper's MRD should be closed (evicted)")
	for i := range wrappers[1:] {
		key := wrapperKey(wrappers[i+1])
		assert.NotNil(t.T(), t.cache.LookUpWithoutChangingOrder(key), "Wrapper %d should be in cache", i+1)
		assert.NotNil(t.T(), wrappers[i+1].Wrapped, "Wrapper %d MRD should exist (pooled)", i+1)
	}
}

func (t *mrdWrapperCacheTest) Test_Cache_DeletedIfReopened() {
	// Arrange: Create 3 wrappers and fill cache
	wrappers := make([]*MultiRangeDownloaderWrapper, 3)
	for i := range 3 {
		obj := &gcs.MinObject{
			Name:       fmt.Sprintf("file%d", i),
			Size:       100,
			Generation: int64(1000 + i),
		}
		wrapper, err := NewMultiRangeDownloaderWrapper(
			t.mockBucket,
			obj,
			&cfg.Config{},
			t.cache,
		)
		assert.NoError(t.T(), err)
		wrapper.Wrapped = fake.NewFakeMultiRangeDownloaderWithSleep(obj, t.objectData, time.Microsecond)
		wrappers[i] = wrapper
		wrappers[i].IncrementRefCount()
		err = wrappers[i].DecrementRefCount()
		assert.NoError(t.T(), err)
	}

	// Act: Reopen wrapper 0 -> should remove it from cache
	wrappers[0].IncrementRefCount()

	// Assert: wrapper 0 will be deleted from cache.
	assert.Nil(t.T(), t.cache.LookUpWithoutChangingOrder(wrapperKey(wrappers[0])), "Wrapper 0 should not be in cache")
}

func (t *mrdWrapperCacheTest) Test_Cache_ConcurrentAddRemove() {
	// Arrange
	const numGoroutines = 10
	const numIterations = 100
	wg := sync.WaitGroup{}

	// Act: Concurrent open/close cycles from multiple goroutines
	for range numGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range numIterations {
				t.mrdWrapper.IncrementRefCount()
				err := t.mrdWrapper.DecrementRefCount()
				assert.NoError(t.T(), err)
			}
		}()
	}
	wg.Wait()

	// Assert: Final state is refCount=0 (no deadlocks or panics)
	assert.Equal(t.T(), 0, t.mrdWrapper.refCount, "RefCount should be 0 after all operations")
	assert.NotNil(t.T(), t.cache.LookUpWithoutChangingOrder(wrapperKey(t.mrdWrapper)), "Wrapper should be in cache")
}

func (t *mrdWrapperCacheTest) Test_Cache_Disabled() {
	// Arrange: Create wrapper with nil cache (disabled)
	wrapper, err := NewMultiRangeDownloaderWrapper(
		t.mockBucket,
		t.object,
		&cfg.Config{},
		nil, // Cache disabled
	)
	assert.NoError(t.T(), err)
	wrapper.Wrapped = fake.NewFakeMultiRangeDownloaderWithSleep(t.object, t.objectData, time.Microsecond)

	// Act: Open and close file
	wrapper.IncrementRefCount()
	err = wrapper.DecrementRefCount()
	assert.NoError(t.T(), err)

	// Assert: MRD will be open forever since cache is disabled.
	assert.NotNil(t.T(), wrapper.Wrapped, "MRD should be open when cache disabled")
}

func (t *mrdWrapperCacheTest) Test_Cache_EvictionRaceWithRepool() {
	// Arrange: Add wrapper to cache then fill with 3 more to trigger eviction
	t.mrdWrapper.IncrementRefCount()
	err := t.mrdWrapper.DecrementRefCount()
	assert.NoError(t.T(), err)
	for i := range 3 {
		obj := &gcs.MinObject{
			Name:       fmt.Sprintf("file%d", i),
			Size:       100,
			Generation: int64(1000 + i),
		}
		wrapper, err := NewMultiRangeDownloaderWrapper(
			t.mockBucket,
			obj,
			&cfg.Config{},
			t.cache,
		)
		assert.NoError(t.T(), err)
		wrapper.Wrapped = fake.NewFakeMultiRangeDownloaderWithSleep(obj, t.objectData, time.Microsecond)
		wrapper.IncrementRefCount()
		err = wrapper.DecrementRefCount()
		assert.NoError(t.T(), err)
	}

	// Act: Access evicted wrapper (should recreate MRD)
	buf := make([]byte, 10)
	t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(
		fake.NewFakeMultiRangeDownloaderWithSleep(t.object, t.objectData, time.Microsecond),
		nil,
	).Once()
	bytesRead, err := t.mrdWrapper.Read(context.Background(), buf, 0, 10, metrics.NewNoopMetrics(), false)

	// Assert: MRD recreated successfully after eviction
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 10, bytesRead)
	assert.NotNil(t.T(), t.mrdWrapper.Wrapped, "MRD should be recreated after eviction")
}

func (t *mrdWrapperCacheTest) Test_Cache_MultipleEvictions() {
	// Arrange: Create small cache (size 2) and 5 wrappers
	smallCache := lru.NewCache(2)
	wrappers := make([]*MultiRangeDownloaderWrapper, 5)
	for i := range 5 {
		obj := &gcs.MinObject{
			Name:       fmt.Sprintf("file%d", i),
			Size:       100,
			Generation: int64(1000 + i),
		}
		wrapper, err := NewMultiRangeDownloaderWrapper(
			t.mockBucket,
			obj,
			&cfg.Config{},
			smallCache,
		)
		assert.NoError(t.T(), err)
		wrapper.Wrapped = fake.NewFakeMultiRangeDownloaderWithSleep(obj, t.objectData, time.Microsecond)
		wrappers[i] = wrapper
	}

	// Act: Add all 5 wrappers (triggers batch eviction of 3)
	for i := range 5 {
		wrappers[i].IncrementRefCount()
		err := wrappers[i].DecrementRefCount()
		assert.NoError(t.T(), err)
	}

	// Assert: First 3 evicted, last 2 remain in cache
	for i := range wrappers[:3] {
		assert.Nil(t.T(), wrappers[i].Wrapped, "Wrapper %d should be evicted", i)
	}
	for i := range wrappers[3:] {
		assert.NotNil(t.T(), wrappers[i+3].Wrapped, "Wrapper %d should be in cache", i+3)
	}
}
