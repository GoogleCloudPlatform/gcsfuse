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
	"errors"
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
	reader           FileCacheReader
	ctx              context.Context
	object           *gcs.MinObject
	mockBucket       *storage.TestifyMockBucket
	cacheDir         string
	jobManager       *downloader.JobManager
	mockCacheHandler *file.MockCacheHandler
	mockCacheHandle  *file.MockCacheHandle
	cacheHandler     file.CacheHandlerInterface
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
	lruCache := lru.NewCache(util.CacheMaxSize)
	t.jobManager = downloader.NewJobManager(lruCache, util.DefaultFilePerm, util.DefaultDirPerm, t.cacheDir, util.SequentialReadSizeInMb, &cfg.FileCacheConfig{EnableCrc: false}, nil)
	t.mockCacheHandler = new(file.MockCacheHandler)
	readOp := &fuseops.ReadFileOp{
		Handle: fuseops.HandleID(123),
		Offset: 0,
		Size:   10,
	}
	t.mockCacheHandle = new(file.MockCacheHandle)
	t.cacheHandler = file.NewCacheHandler(lruCache, t.jobManager, t.cacheDir, util.DefaultFilePerm, util.DefaultDirPerm)
	t.ctx = context.WithValue(context.Background(), ReadOp, readOp)
	t.reader = NewFileCacheReader(t.object, t.mockBucket, t.cacheHandler, true, nil)
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
	assert.Nil(t.T(), reader.fileCacheHandle)
}

func (t *FileCacheReaderTest) TestTReadAt_ryReadingFromFileCache_NilHandler() {
	reader := NewFileCacheReader(t.object, t.mockBucket, nil, true, nil)
	readerResponse, err := reader.ReadAt(t.ctx, make([]byte, 10), 0)

	assert.True(t.T(), errors.Is(err, FallbackToAnotherReader))
	assert.Zero(t.T(), readerResponse.Size)
}

//func (t *FileCacheReaderTest) mockNewReaderWithHandleCallForTestBucket(start uint64, limit uint64, rd gcs.StorageReader) {
//	ExpectCall(t.mockBucket, "NewReaderWithReadHandle")(
//		Any(), AllOf(rangeStartIs(start), rangeLimitIs(limit))).
//		WillRepeatedly(Return(rd, nil))
//}

//func (t *FileCacheReaderTest) Test_ReadAt_SequentialRangeRead() {
//	t.reader.fileCacheHandler = t.cacheHandler
//	objectSize := t.object.Size
//	testContent := testutil.GenerateRandomBytes(int(objectSize))
//	//rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
//	//t.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd)
//	t.mockBucket.On("Name").Return("test-bucket")
//	start := 0
//	end := 10 // not included
//	AssertLt(end, objectSize)
//	buf := make([]byte, end-start)
//
//	objectData, err := t.rr.ReadAt(buf, int64(start))
//
//	ExpectFalse(objectData.CacheHit)
//	ExpectEq(nil, err)
//	ExpectTrue(reflect.DeepEqual(testContent[start:end], buf))
//}

func (t *FileCacheReaderTest) TestReadAt_TryReadingFromFileCache_NotAbleToCreateCacheHandle() {
	t.mockCacheHandler.On("GetCacheHandle", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("mock error"))
	t.mockBucket.On("Name").Return("test-bucket")
	reader := NewFileCacheReader(t.object, t.mockBucket, t.mockCacheHandler, true, common.NewNoopMetrics())

	readerResponse, err := reader.ReadAt(t.ctx, make([]byte, 10), 0)

	assert.Error(t.T(), err)
	assert.Zero(t.T(), readerResponse.Size)
	// Verify mocks
	t.mockCacheHandle.AssertExpectations(t.T())
	t.mockBucket.AssertExpectations(t.T())
	t.mockCacheHandler.AssertExpectations(t.T())
}

func (t *FileCacheReaderTest) TestReadAt_TryReadingFromFileCache_ErrorScenarios() {
	type testCase struct {
		name        string
		mockErr     error
		expectedErr error
	}

	cases := []testCase{
		{
			name:        "InvalidEntrySize - should skip cache read",
			mockErr:     lru.ErrInvalidEntrySize,
			expectedErr: FallbackToAnotherReader,
		},
		{
			name:        "CacheHandleNotRequiredForRandomRead - should skip cache read",
			mockErr:     util.ErrCacheHandleNotRequiredForRandomRead,
			expectedErr: FallbackToAnotherReader,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func() {
			t.SetupTest()
			t.mockCacheHandler.On("GetCacheHandle", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, tc.mockErr)
			t.mockBucket.On("Name").Return("test-bucket")
			reader := NewFileCacheReader(t.object, t.mockBucket, t.mockCacheHandler, true, common.NewNoopMetrics())

			readerResponse, err := reader.ReadAt(t.ctx, make([]byte, 10), 0)

			assert.Equal(t.T(), tc.expectedErr, err)
			assert.Zero(t.T(), readerResponse.Size)
			// Verify mocks
			t.mockCacheHandle.AssertExpectations(t.T())
			t.mockBucket.AssertExpectations(t.T())
			t.mockCacheHandler.AssertExpectations(t.T())
		})
	}
}

func (t *FileCacheReaderTest) Test_ReadAt_TryReadingFromFileCache_fallsBackToGCS() {
	p1 := make([]byte, 100)
	offset1 := int64(0)
	t.mockBucket.On("Name").Return("test-bucket")
	t.mockCacheHandle.On("Read", mock.Anything, mock.Anything, mock.Anything, offset1, p1).Return(0, false, util.ErrFallbackToGCS).Once()
	t.mockCacheHandle.On("IsSequential", offset1).Return(true)
	t.mockCacheHandler.On("GetCacheHandle", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(t.mockCacheHandle, nil)
	reader := NewFileCacheReader(t.object, t.mockBucket, t.mockCacheHandler, true, common.NewNoopMetrics())

	readerResponse, err := reader.ReadAt(t.ctx, p1, offset1)

	assert.True(t.T(), errors.Is(err, FallbackToAnotherReader))
	assert.Zero(t.T(), readerResponse.Size)
	// Verify mocks
	t.mockCacheHandle.AssertExpectations(t.T())
	t.mockCacheHandler.AssertExpectations(t.T())
	t.mockBucket.AssertExpectations(t.T())
}

func (t *FileCacheReaderTest) Test_ReadAt_TryReadingFromFileCache_NotFallsBackToGCS() {
	p1 := make([]byte, 100)
	offset1 := int64(0)
	t.mockBucket.On("Name").Return("test-bucket")
	t.mockCacheHandle.On("Read", mock.Anything, mock.Anything, mock.Anything, offset1, p1).Return(0, false, fmt.Errorf("mock error")).Once()
	t.mockCacheHandle.On("IsSequential", offset1).Return(true)
	t.mockCacheHandler.On("GetCacheHandle", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(t.mockCacheHandle, nil)
	reader := NewFileCacheReader(t.object, t.mockBucket, t.mockCacheHandler, true, common.NewNoopMetrics())

	readerResponse, err := reader.ReadAt(t.ctx, p1, offset1)

	assert.Error(t.T(), err)
	assert.Zero(t.T(), readerResponse.Size)
	// Verify mocks
	t.mockCacheHandle.AssertExpectations(t.T())
	t.mockCacheHandler.AssertExpectations(t.T())
	t.mockBucket.AssertExpectations(t.T())
}

func (t *FileCacheReaderTest) Test_ReadAt_TryReadingFromFileCache_HandleInvalidStatesGracefully() {
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
			reader := NewFileCacheReader(t.object, t.mockBucket, t.mockCacheHandler, true, common.NewNoopMetrics())

			readerResponse, err := reader.ReadAt(t.ctx, buffer, offset)

			assert.True(t.T(), errors.Is(err, FallbackToAnotherReader))
			assert.Zero(t.T(), readerResponse.Size)
			assert.Nil(t.T(), reader.fileCacheHandle)
			// Verify mocks
			t.mockCacheHandle.AssertExpectations(t.T())
			t.mockBucket.AssertExpectations(t.T())
			t.mockCacheHandler.AssertExpectations(t.T())
		})
	}
}

func (t *FileCacheReaderTest) Test_ReadAt_Success() {
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
			reader := NewFileCacheReader(t.object, t.mockBucket, t.mockCacheHandler, true, common.NewNoopMetrics())

			readerResponse, err := reader.ReadAt(t.ctx, buffer, tc.offset)

			assert.NoError(t.T(), err)
			assert.Equal(t.T(), readerResponse.Size, len(buffer))
			assert.NotNil(t.T(), reader.fileCacheHandle)
			t.mockCacheHandle.AssertExpectations(t.T())
			t.mockBucket.AssertExpectations(t.T())
			t.mockCacheHandler.AssertExpectations(t.T())
		})
	}
}
