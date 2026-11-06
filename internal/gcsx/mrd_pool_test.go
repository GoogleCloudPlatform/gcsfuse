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
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/gcsfuse_errors"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type mrdPoolTest struct {
	suite.Suite
	object     *gcs.MinObject
	bucket     *storage.TestifyMockBucket
	poolConfig *MRDPoolConfig
}

func TestMRDPoolTestSuite(t *testing.T) {
	suite.Run(t, new(mrdPoolTest))
}

func (t *mrdPoolTest) SetupTest() {
	t.object = &gcs.MinObject{
		Name:       "foo",
		Size:       1024 * MiB,
		Generation: 1234,
	}
	t.bucket = new(storage.TestifyMockBucket)
	t.poolConfig = &MRDPoolConfig{
		PoolSize: 4,
		object:   t.object,
		bucket:   t.bucket,
	}
}

func (t *mrdPoolTest) TestNewMRDPool_SmallFile() {
	t.object.Size = 100 * MiB
	t.poolConfig.PoolSize = 4
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	// A single MultiRangeDownloader with MinConnections=2, MaxConnections=2 is created for [100MiB, 500MiB) files.
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.MatchedBy(func(req *gcs.MultiRangeDownloaderRequest) bool {
		return req.MinConnections == 2 && req.MaxConnections == 2
	})).Return(fakeMRD, nil).Once()

	pool, err := NewMRDPool(t.poolConfig, nil)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 2, pool.poolConfig.PoolSize)
	assert.Equal(t.T(), uint64(2), pool.Size())
	assert.Len(t.T(), pool.entries, 1)
	assert.NotNil(t.T(), pool.entries[0].mrd)
	t.bucket.AssertExpectations(t.T())
}

func (t *mrdPoolTest) TestNewMRDPool_LargeFile() {
	t.object.Size = 1024 * MiB
	t.poolConfig.PoolSize = 4
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.MatchedBy(func(req *gcs.MultiRangeDownloaderRequest) bool {
		return req.MinConnections == 4 && req.MaxConnections == 4
	})).Return(fakeMRD, nil).Once()

	pool, err := NewMRDPool(t.poolConfig, nil)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 4, pool.poolConfig.PoolSize)
	assert.Equal(t.T(), uint64(4), pool.Size())
	assert.Len(t.T(), pool.entries, 1)
	assert.NotNil(t.T(), pool.entries[0].mrd)
	t.bucket.AssertExpectations(t.T())
}

func (t *mrdPoolTest) TestNewMRDPool_PassesReadHandle() {
	t.object.Size = 1024 * MiB
	t.poolConfig.PoolSize = 4
	expectedHandle := []byte("cached-handle")
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.MatchedBy(func(req *gcs.MultiRangeDownloaderRequest) bool {
		return string(req.ReadHandle) == "cached-handle" && req.MinConnections == 4 && req.MaxConnections == 4
	})).Return(fakeMRD, nil).Once()

	pool, err := NewMRDPool(t.poolConfig, expectedHandle)

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), pool.entries[0].mrd)
	t.bucket.AssertExpectations(t.T())
}

func (t *mrdPoolTest) TestNewMRDPool_FileClobbered() {
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(nil, &gcs.NotFoundError{Err: fmt.Errorf("not found")}).Once()

	pool, err := NewMRDPool(t.poolConfig, nil)

	require.Error(t.T(), err)
	assert.Nil(t.T(), pool)
	var clobberedErr *gcsfuse_errors.FileClobberedError
	assert.ErrorAs(t.T(), err, &clobberedErr)
}

func (t *mrdPoolTest) TestNewMRDPool_NilConfig() {
	pool, err := NewMRDPool(nil, nil)

	assert.ErrorContains(t.T(), err, "config cannot be nil")

	assert.Nil(t.T(), pool)
}

func (t *mrdPoolTest) TestNewMRDPool_Error() {
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("error")).Once()

	pool, err := NewMRDPool(t.poolConfig, nil)

	assert.Error(t.T(), err)
	assert.Nil(t.T(), pool)
}

func (t *mrdPoolTest) TestNext() {
	t.poolConfig.PoolSize = 3
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.MatchedBy(func(req *gcs.MultiRangeDownloaderRequest) bool {
		return req.MinConnections == 3 && req.MaxConnections == 3
	})).Return(fakeMRD, nil).Once()
	pool, err := NewMRDPool(t.poolConfig, nil)
	assert.NoError(t.T(), err)

	e1 := pool.Next()
	e2 := pool.Next()

	assert.Same(t.T(), e1.mrd, fakeMRD)
	assert.Same(t.T(), e2.mrd, fakeMRD)
}

func (t *mrdPoolTest) TestDeterminePoolSize() {
	testCases := []struct {
		name             string
		objectSize       uint64
		initialPoolSize  int
		expectedPoolSize int
	}{
		{
			name:             "SmallFile_BelowThreshold",
			objectSize:       50 * MiB,
			initialPoolSize:  4,
			expectedPoolSize: 1,
		},
		{
			name:             "SmallFile_AtThreshold",
			objectSize:       smallFileThresholdMiB * MiB,
			initialPoolSize:  4,
			expectedPoolSize: 2,
		},
		{
			name:             "MediumFile_BetweenThresholds",
			objectSize:       300 * MiB,
			initialPoolSize:  4,
			expectedPoolSize: 2,
		},
		{
			name:             "MediumFile_JustBelowThreshold",
			objectSize:       (mediumFileThresholdMiB - 1) * MiB,
			initialPoolSize:  4,
			expectedPoolSize: 2,
		},
		{
			name:             "LargeFile_AtThreshold",
			objectSize:       mediumFileThresholdMiB * MiB,
			initialPoolSize:  4,
			expectedPoolSize: 4,
		},
		{
			name:             "LargeFile_AboveThreshold",
			objectSize:       2048 * MiB,
			initialPoolSize:  4,
			expectedPoolSize: 4,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.object.Size = tc.objectSize
			t.poolConfig.PoolSize = tc.initialPoolSize

			t.poolConfig.determinePoolSize()

			assert.Equal(t.T(), tc.expectedPoolSize, t.poolConfig.PoolSize)
		})
	}
}

func (t *mrdPoolTest) TestRecreateMRD() {
	t.poolConfig.PoolSize = 4
	existingHandle := []byte("existing_handle")
	fakeMRD1 := fake.NewFakeMultiRangeDownloaderWithHandle(t.object, nil, existingHandle)
	fakeMRD2 := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.MatchedBy(func(req *gcs.MultiRangeDownloaderRequest) bool {
		return req.ReadHandle == nil && req.MinConnections == 4 && req.MaxConnections == 4
	})).Return(fakeMRD1, nil).Once()
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.MatchedBy(func(req *gcs.MultiRangeDownloaderRequest) bool {
		return string(req.ReadHandle) == "existing_handle" && req.MinConnections == 4 && req.MaxConnections == 4
	})).Return(fakeMRD2, nil).Once()
	pool, err := NewMRDPool(t.poolConfig, nil)
	assert.NoError(t.T(), err)
	entry := pool.Next()
	oldMRD := entry.mrd

	err = pool.RecreateMRD(entry, nil)

	assert.NoError(t.T(), err)
	assert.NotSame(t.T(), oldMRD, entry.mrd)
	t.bucket.AssertExpectations(t.T())
}

func (t *mrdPoolTest) TestRecreateMRD_UsesFallbackHandle() {
	t.poolConfig.PoolSize = 2
	// Initial creation
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	pool, err := NewMRDPool(t.poolConfig, nil)
	require.NoError(t.T(), err)
	entry := pool.Next()
	// Simulate invalid entry
	entry.mu.Lock()
	entry.mrd = nil
	entry.mu.Unlock()
	fallbackHandle := []byte("fallback")
	// Expectation: Recreate uses fallback handle and configured pool size
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.MatchedBy(func(req *gcs.MultiRangeDownloaderRequest) bool {
		return string(req.ReadHandle) == "fallback" && req.MinConnections == 2 && req.MaxConnections == 2
	})).Return(fakeMRD, nil).Once()

	err = pool.RecreateMRD(entry, fallbackHandle)

	assert.NoError(t.T(), err)
	t.bucket.AssertExpectations(t.T())
}

func (t *mrdPoolTest) TestRecreateMRD_Error() {
	t.poolConfig.PoolSize = 1
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	pool, err := NewMRDPool(t.poolConfig, nil)
	assert.NoError(t.T(), err)
	entry := pool.Next()
	oldMRD := entry.mrd
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("recreate error")).Once() // Fail the recreation

	err = pool.RecreateMRD(entry, nil)

	assert.Error(t.T(), err)
	assert.Same(t.T(), oldMRD, entry.mrd) // Should remain unchanged on error
}

func (t *mrdPoolTest) TestClose() {
	t.poolConfig.PoolSize = 2
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	pool, err := NewMRDPool(t.poolConfig, nil)
	assert.NoError(t.T(), err)

	pool.Close()

	// Verify entries are cleared
	for i := 0; i < len(pool.entries); i++ {
		assert.Nil(t.T(), pool.entries[i].mrd)
	}
}

func (t *mrdPoolTest) TestClose_ReturnsHandle() {
	t.poolConfig.PoolSize = 1
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	pool, err := NewMRDPool(t.poolConfig, nil)
	require.NoError(t.T(), err)
	// Inject mock to return handle on Close/GetHandle check
	expectedHandle := []byte("handle")
	mockMRD := fake.NewFakeMultiRangeDownloaderWithHandle(t.object, nil, expectedHandle)
	pool.entries[0].mu.Lock()
	pool.entries[0].mrd = mockMRD
	pool.entries[0].mu.Unlock()

	handle := pool.Close()

	assert.Equal(t.T(), expectedHandle, handle)
}

func (t *mrdPoolTest) TestCloseDoesNotCancelDownloaderContext() {
	t.poolConfig.PoolSize = 1
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, nil)
	ctxCh := make(chan context.Context, 1)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		ctxCh <- args.Get(0).(context.Context)
	}).Return(fakeMRD, nil).Once()
	pool, err := NewMRDPool(t.poolConfig, nil)
	require.NoError(t.T(), err)
	capturedCtx := <-ctxCh

	pool.Close()

	// context.Background() never gets canceled and has no Done channel
	require.NotNil(t.T(), capturedCtx)
	assert.Nil(t.T(), capturedCtx.Done())
	assert.NoError(t.T(), capturedCtx.Err())
}
