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
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/file"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/file/downloader"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

const (
	TestObject                = "testObject"
	sequentialReadSizeInMb    = 22
	sequentialReadSizeInBytes = sequentialReadSizeInMb * MB
	CacheMaxSize              = 2 * sequentialReadSizeInMb * util.MiB
)

type FileCacheReaderTest struct {
	suite.Suite
	ctx             context.Context
	object          *gcs.MinObject
	mockBucket      *storage.TestifyMockBucket
	cacheDir        string
	jobManager      *downloader.JobManager
	cacheHandler    *MockFileCacheHandler
	mockCacheHandle *MockFileCacheHandle
	metricHandle    *common.MockMetricHandle
}

// MockFileCacheHandle is a mock type for the FileCacheMetricHandle type
type MockFileCacheHandle struct {
	mock.Mock
}

func (m *MockFileCacheHandle) Read(ctx context.Context, bucket gcs.Bucket, object *gcs.MinObject, offset int64, dst []byte) (int, bool, error) {
	args := m.Called(ctx, bucket, object, offset, dst)

	n := args.Int(0)
	cacheHit := args.Bool(1)
	err := args.Error(2)

	return n, cacheHit, err
}

func (m *MockFileCacheHandle) IsSequential(currentOffset int64) bool {
	args := m.Called(currentOffset)
	return args.Bool(0)
}

func (m *MockFileCacheHandle) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockFileCacheHandler is a mock type for the FileCacheMetricHandle type
type MockFileCacheHandler struct {
	mock.Mock
}

func (m *MockFileCacheHandler) GetCacheHandle(obj *gcs.MinObject, bucket gcs.Bucket, cacheFileForRangeRead bool, initialOffset int64) (file.CacheHandleInterface, error) {
	args := m.Called(obj, bucket, cacheFileForRangeRead, initialOffset)
	handle, _ := args.Get(0).(file.CacheHandleInterface) // Allow returning nil handle
	return handle, args.Error(1)
}

func (m *MockFileCacheHandler) InvalidateCache(objectName string, bucketName string) error {
	args := m.Called(objectName, bucketName)
	return args.Error(0)
}

func TestFileCacheReaderTestSuite(t *testing.T) {
	suite.Run(t, new(FileCacheReaderTest))
}

func (t *FileCacheReaderTest) SetupTest() {
	t.object = &gcs.MinObject{
		Name:       TestObject,
		Size:       17,
		Generation: 1234,
	}
	t.mockBucket = new(storage.TestifyMockBucket)
	t.cacheDir = path.Join(os.Getenv("HOME"), "test_cache_dir")
	lruCache := lru.NewCache(util.CacheMaxSize)
	t.jobManager = downloader.NewJobManager(lruCache, util.DefaultFilePerm, util.DefaultDirPerm, t.cacheDir, util.SequentialReadSizeInMb, &cfg.FileCacheConfig{EnableCrc: false}, nil)
	t.cacheHandler = new(MockFileCacheHandler)
	readOp := &fuseops.ReadFileOp{
		Handle: fuseops.HandleID(123),
		Offset: 0,
		Size:   10,
	}
	t.ctx = context.WithValue(context.Background(), ReadOp, readOp)
}

func (t *FileCacheReaderTest) TearDown() {
	err := os.RemoveAll(t.cacheDir)
	if err != nil {
		t.T().Logf("Failed to clean up test cache directory '%s': %v", t.cacheDir, err)
	}
}

func (t *FileCacheReaderTest) TestNewFileCacheReader() {
	reader := NewFileCacheReader(t.object, t.mockBucket, t.cacheHandler, true, nil)

	assert.NotNil(t.T(), reader)
	assert.Equal(t.T(), t.object, reader.obj)
	assert.Equal(t.T(), t.mockBucket, reader.bucket)
	assert.Equal(t.T(), t.cacheHandler, reader.fileCacheHandler)
	assert.True(t.T(), reader.cacheFileForRangeRead)
	assert.Nil(t.T(), reader.metricHandle)
	assert.Nil(t.T(), reader.fileCacheHandle)
}

func (t *FileCacheReaderTest) TestTryReadingFromFileCache_NilHandler() {
	reader := NewFileCacheReader(t.object, t.mockBucket, nil, true, nil)
	n, hit, err := reader.tryReadingFromFileCache(context.Background(), make([]byte, 10), 0)
	assert.NoError(t.T(), err)
	assert.Zero(t.T(), n)
	assert.False(t.T(), hit)
}

func (t *FileCacheReaderTest) TestTryReadingFromFileCache_ErrorScenarios() {
	type testCase struct {
		name        string
		mockErr     error
		expectedErr bool
		expectedHit bool
		expectedN   int
	}

	cases := []testCase{
		{
			name:        "InvalidEntrySize - should skip cache read",
			mockErr:     lru.ErrInvalidEntrySize,
			expectedErr: false,
			expectedHit: false,
			expectedN:   0,
		},
		{
			name:        "CacheHandleNotRequiredForRandomRead - should skip cache read",
			mockErr:     util.ErrCacheHandleNotRequiredForRandomRead,
			expectedErr: false,
			expectedHit: false,
			expectedN:   0,
		},
		{
			name:        "Generic handle creation error - should return error",
			mockErr:     fmt.Errorf("mock creation error"),
			expectedErr: true,
			expectedHit: false,
			expectedN:   0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func() {
			mockCacheHandler := new(MockFileCacheHandler)
			mockCacheHandler.On("GetCacheHandle", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, tc.mockErr)
			mockBucket := new(storage.TestifyMockBucket)
			mockBucket.On("Name").Return("test-bucket")
			mockMetricHandle := new(common.MockMetricHandle)
			mockMetricHandle.On("FileCacheReadCount", mock.Anything, int64(1), mock.Anything).Return()
			mockMetricHandle.On("FileCacheReadBytesCount", mock.Anything, mock.AnythingOfType("int64"), mock.Anything).Return()
			mockMetricHandle.On("FileCacheReadLatency", mock.Anything, mock.AnythingOfType("float64"), mock.Anything).Return()
			reader := NewFileCacheReader(t.object, mockBucket, mockCacheHandler, true, mockMetricHandle)

			n, hit, err := reader.tryReadingFromFileCache(t.ctx, make([]byte, 10), 0)

			if tc.expectedErr {
				assert.Error(t.T(), err)
			} else {
				assert.NoError(t.T(), err)
			}
			assert.Equal(t.T(), tc.expectedHit, hit)
			assert.Equal(t.T(), tc.expectedN, n)

			// Verify mocks
			mockCacheHandler.AssertExpectations(t.T())
			mockBucket.AssertExpectations(t.T())
			mockMetricHandle.AssertExpectations(t.T())
		})
	}
}

func (t *FileCacheReaderTest) TestReadFail() {
	// Arrange
	p1 := make([]byte, 100)
	offset1 := int64(0)

	// Bucket
	mockBucket := new(storage.TestifyMockBucket)
	mockBucket.On("Name").Return("test-bucket")

	// Cache Handle
	mockCacheHandle := new(MockFileCacheHandle)
	mockCacheHandle.On("Read", mock.Anything, mock.Anything, mock.Anything, offset1, p1).
		Return(0, false, util.ErrFallbackToGCS).Once()

	// Cache Handler
	mockCacheHandler := new(MockFileCacheHandler)
	mockCacheHandler.On("GetCacheHandle", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(mockCacheHandle, nil)

	// Metric Handle
	mockMetricHandle := new(common.MockMetricHandle)
	mockMetricHandle.On("FileCacheReadCount", mock.Anything, int64(1), mock.Anything).Return()
	mockMetricHandle.On("FileCacheReadBytesCount", mock.Anything, mock.AnythingOfType("int64"), mock.Anything).Return()
	mockMetricHandle.On("FileCacheReadLatency", mock.Anything, mock.AnythingOfType("float64"), mock.Anything).Return()

	// Reader
	reader := NewFileCacheReader(t.object, mockBucket, mockCacheHandler, true, mockMetricHandle)

	// Act - first read
	n1, hit1, err1 := reader.tryReadingFromFileCache(t.ctx, p1, offset1)

	// Assert - first read
	assert.NoError(t.T(), err1)
	assert.Equal(t.T(), 0, n1)
	assert.False(t.T(), hit1)

	mockCacheHandle.AssertExpectations(t.T())
	mockCacheHandler.AssertExpectations(t.T())
	mockMetricHandle.AssertExpectations(t.T())
	mockBucket.AssertExpectations(t.T())
}
