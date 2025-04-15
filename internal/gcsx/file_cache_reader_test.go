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
	ctx              context.Context
	object           *gcs.MinObject
	mockBucket       *storage.TestifyMockBucket
	cacheDir         string
	jobManager       *downloader.JobManager
	mockCacheHandler *file.MockCacheHandler
	mockCacheHandle  *file.MockCacheHandle
	mockMetricHandle *common.MockMetricHandle
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
	//lruCache := lru.NewCache(util.CacheMaxSize)
	//t.jobManager = downloader.NewJobManager(lruCache, util.DefaultFilePerm, util.DefaultDirPerm, t.cacheDir, util.SequentialReadSizeInMb, &cfg.FileCacheConfig{EnableCrc: false}, nil)
	t.mockCacheHandler = new(file.MockCacheHandler)
	readOp := &fuseops.ReadFileOp{
		Handle: fuseops.HandleID(123),
		Offset: 0,
		Size:   10,
	}
	t.mockCacheHandle = new(file.MockCacheHandle)
	t.mockMetricHandle = new(common.MockMetricHandle)
	t.ctx = context.WithValue(context.Background(), ReadOp, readOp)
}

func (t *FileCacheReaderTest) TearDown() {
	err := os.RemoveAll(t.cacheDir)
	if err != nil {
		t.T().Logf("Failed to clean up test cache directory '%s': %v", t.cacheDir, err)
	}
}

func (t *FileCacheReaderTest) TestNewFileCacheReader() {
	reader := NewFileCacheReader(t.object, t.mockBucket, t.mockCacheHandler, true, nil)

	assert.NotNil(t.T(), reader)
	assert.Equal(t.T(), t.object, reader.obj)
	assert.Equal(t.T(), t.mockBucket, reader.bucket)
	assert.Equal(t.T(), t.mockCacheHandler, reader.fileCacheHandler)
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
	}

	cases := []testCase{
		{
			name:        "InvalidEntrySize - should skip cache read",
			mockErr:     lru.ErrInvalidEntrySize,
			expectedErr: false,
		},
		{
			name:        "CacheHandleNotRequiredForRandomRead - should skip cache read",
			mockErr:     util.ErrCacheHandleNotRequiredForRandomRead,
			expectedErr: false,
		},
		{
			name:        "Generic handle creation error - should return error",
			mockErr:     fmt.Errorf("mock creation error"),
			expectedErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func() {
			t.SetupTest()
			t.mockCacheHandler.On("GetCacheHandle", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, tc.mockErr)
			t.mockBucket.On("Name").Return("test-bucket")
			t.mockMetricHandle.On("FileCacheReadCount", mock.Anything, int64(1), mock.Anything).Return()
			t.mockMetricHandle.On("FileCacheReadBytesCount", mock.Anything, mock.AnythingOfType("int64"), mock.Anything).Return()
			t.mockMetricHandle.On("FileCacheReadLatency", mock.Anything, mock.AnythingOfType("float64"), mock.Anything).Return()
			reader := NewFileCacheReader(t.object, t.mockBucket, t.mockCacheHandler, true, t.mockMetricHandle)

			n, hit, err := reader.tryReadingFromFileCache(t.ctx, make([]byte, 10), 0)

			if tc.expectedErr {
				assert.Error(t.T(), err)
			} else {
				assert.NoError(t.T(), err)
			}
			assert.False(t.T(), hit)
			assert.Zero(t.T(), n)
			// Verify mocks
			t.mockCacheHandle.AssertExpectations(t.T())
			t.mockBucket.AssertExpectations(t.T())
			t.mockCacheHandler.AssertExpectations(t.T())
		})
	}
}

func (t *FileCacheReaderTest) Test_TryReadingFromFileCache_fallsBackToGCS() {
	p1 := make([]byte, 100)
	offset1 := int64(0)
	t.mockBucket.On("Name").Return("test-bucket")
	t.mockCacheHandle.On("Read", mock.Anything, mock.Anything, mock.Anything, offset1, p1).Return(0, false, util.ErrFallbackToGCS).Once()
	t.mockCacheHandle.On("IsSequential", offset1).Return(true)
	t.mockCacheHandler.On("GetCacheHandle", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(t.mockCacheHandle, nil)
	t.mockMetricHandle.On("FileCacheReadCount", mock.Anything, int64(1), mock.Anything).Return()
	t.mockMetricHandle.On("FileCacheReadBytesCount", mock.Anything, mock.AnythingOfType("int64"), mock.Anything).Return()
	t.mockMetricHandle.On("FileCacheReadLatency", mock.Anything, mock.AnythingOfType("float64"), mock.Anything).Return()
	reader := NewFileCacheReader(t.object, t.mockBucket, t.mockCacheHandler, true, t.mockMetricHandle)

	n, hit, err := reader.tryReadingFromFileCache(t.ctx, p1, offset1)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 0, n)
	assert.False(t.T(), hit)
	// Verify mocks
	t.mockCacheHandle.AssertExpectations(t.T())
	t.mockCacheHandler.AssertExpectations(t.T())
	t.mockMetricHandle.AssertExpectations(t.T())
	t.mockBucket.AssertExpectations(t.T())
}

func (t *FileCacheReaderTest) Test_TryReadingFromFileCache_NotfallsBackToGCS() {
	p1 := make([]byte, 100)
	offset1 := int64(0)
	t.mockBucket.On("Name").Return("test-bucket")
	t.mockCacheHandle.On("Read", mock.Anything, mock.Anything, mock.Anything, offset1, p1).Return(0, false, fmt.Errorf("mock error")).Once()
	t.mockCacheHandle.On("IsSequential", offset1).Return(true)
	t.mockCacheHandler.On("GetCacheHandle", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(t.mockCacheHandle, nil)
	t.mockMetricHandle.On("FileCacheReadCount", mock.Anything, int64(1), mock.Anything).Return()
	t.mockMetricHandle.On("FileCacheReadBytesCount", mock.Anything, mock.AnythingOfType("int64"), mock.Anything).Return()
	t.mockMetricHandle.On("FileCacheReadLatency", mock.Anything, mock.AnythingOfType("float64"), mock.Anything).Return()
	reader := NewFileCacheReader(t.object, t.mockBucket, t.mockCacheHandler, true, t.mockMetricHandle)

	n, hit, err := reader.tryReadingFromFileCache(t.ctx, p1, offset1)

	assert.Error(t.T(), err)
	assert.Equal(t.T(), 0, n)
	assert.False(t.T(), hit)
	// Verify mocks
	t.mockCacheHandle.AssertExpectations(t.T())
	t.mockCacheHandler.AssertExpectations(t.T())
	t.mockMetricHandle.AssertExpectations(t.T())
	t.mockBucket.AssertExpectations(t.T())
}

func (t *FileCacheReaderTest) Test_TryReadingFromFileCache_HandleInvalidStatesGracefully() {
	type testCase struct {
		name            string
		readErr         error
		closeFileHandle error
	}

	testCases := []testCase{
		{
			name:            "InReadingFileHandle - should skip cache read without error",
			readErr:         util.ErrInReadingFileHandle,
			closeFileHandle: nil,
		},
		{
			name:            "InvalidFileHandle - should skip cache read without error",
			readErr:         util.ErrInvalidFileHandle,
			closeFileHandle: nil,
		},
		{
			name:            "InvalidFileInfoCache - should return error",
			readErr:         util.ErrInvalidFileInfoCache,
			closeFileHandle: fmt.Errorf("mock error"),
		},
		{
			name:            "InvalidFileDownloadJob - should return error",
			readErr:         util.ErrInvalidFileDownloadJob,
			closeFileHandle: nil,
		},
		{
			name:            "InSeekingFileHandle - should return error",
			readErr:         util.ErrInSeekingFileHandle,
			closeFileHandle: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.SetupTest()
			offset := int64(0)
			buffer := make([]byte, 100)
			// Set up mocks
			t.mockBucket.On("Name").Return("test-bucket")
			t.mockCacheHandler.On("GetCacheHandle", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(t.mockCacheHandle, nil)
			t.mockCacheHandle.On("Read", mock.Anything, mock.Anything, mock.Anything, offset, buffer).Return(0, false, tc.readErr).Once()
			t.mockCacheHandle.On("Close").Return(tc.closeFileHandle)
			t.mockMetricHandle.On("FileCacheReadCount", mock.Anything, int64(1), mock.Anything).Return()
			t.mockMetricHandle.On("FileCacheReadBytesCount", mock.Anything, mock.AnythingOfType("int64"), mock.Anything).Return()
			t.mockMetricHandle.On("FileCacheReadLatency", mock.Anything, mock.AnythingOfType("float64"), mock.Anything).Return()
			reader := NewFileCacheReader(t.object, t.mockBucket, t.mockCacheHandler, true, t.mockMetricHandle)

			n, hit, err := reader.tryReadingFromFileCache(t.ctx, buffer, offset)

			assert.NoError(t.T(), err)
			assert.False(t.T(), hit)
			assert.Zero(t.T(), n)
			assert.Nil(t.T(), reader.fileCacheHandle)
			// Verify mocks
			t.mockCacheHandle.AssertExpectations(t.T())
			t.mockBucket.AssertExpectations(t.T())
			t.mockCacheHandler.AssertExpectations(t.T())
		})
	}
}

func (t *FileCacheReaderTest) Test_TryReadingFromFileCache_ReadSuccessfully() {
	type testCase struct {
		name   string
		offset int64
	}

	testCases := []testCase{
		{
			name:   "Sequential read at offset 0",
			offset: 0,
		},
		{
			name:   "Sequential read at offset 50",
			offset: 50,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.SetupTest()

			buffer := make([]byte, 100)

			// Arrange
			t.mockBucket.On("Name").Return("test-bucket")

			t.mockCacheHandler.On("GetCacheHandle", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(t.mockCacheHandle, nil)
			t.mockCacheHandle.On("Read", mock.Anything, mock.Anything, mock.Anything, tc.offset, buffer).Return(len(buffer), true, nil).Once()
			t.mockCacheHandle.On("IsSequential", tc.offset).Return(true)
			t.mockMetricHandle.On("FileCacheReadCount", mock.Anything, int64(1), mock.Anything).Return()
			t.mockMetricHandle.On("FileCacheReadBytesCount", mock.Anything, mock.AnythingOfType("int64"), mock.Anything).Return()
			t.mockMetricHandle.On("FileCacheReadLatency", mock.Anything, mock.AnythingOfType("float64"), mock.Anything).Return()
			reader := NewFileCacheReader(t.object, t.mockBucket, t.mockCacheHandler, true, t.mockMetricHandle)

			n, hit, err := reader.tryReadingFromFileCache(t.ctx, buffer, tc.offset)

			assert.NoError(t.T(), err)
			assert.Equal(t.T(), len(buffer), n)
			assert.True(t.T(), hit)
			assert.NotNil(t.T(), reader.fileCacheHandle)
			t.mockCacheHandle.AssertExpectations(t.T())
			t.mockBucket.AssertExpectations(t.T())
			t.mockCacheHandler.AssertExpectations(t.T())
		})
	}
}
