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
	object       *gcs.MinObject
	mockBucket   *storage.TestifyMockBucket
	cacheDir     string
	jobManager   *downloader.JobManager
	cacheHandler *MockFileCacheHandler
	metricHandle *MockFileCacheHandler
}

// MockFileCacheHandler is a mock type for the FileCacheMetricHandle type
type MockFileCacheHandler struct {
	mock.Mock
}

func (m *MockFileCacheHandler) GetCacheHandle(obj *gcs.MinObject, bucket gcs.Bucket, cacheFileForRangeRead bool, initialOffset int64) (*file.CacheHandle, error) {
	args := m.Called(obj, bucket, cacheFileForRangeRead, initialOffset)
	handle, _ := args.Get(0).(*file.CacheHandle) // Allow returning nil handle
	return handle, args.Error(1)
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
			readOp := &fuseops.ReadFileOp{
				Handle: fuseops.HandleID(123),
				Offset: 0,
				Size:   10,
			}
			ctx := context.WithValue(context.Background(), ReadOp, readOp)

			mockHandler := new(MockFileCacheHandler)
			mockHandler.On("GetCacheHandle", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(nil, tc.mockErr)

			mockBucket := new(storage.TestifyMockBucket)
			mockBucket.On("Name").Return("test-bucket")

			mockMetricHandle := new(common.MockMetricHandle)
			mockMetricHandle.On("FileCacheReadCount", mock.Anything, int64(1), mock.Anything).Return()
			mockMetricHandle.On("FileCacheReadBytesCount", mock.Anything, mock.AnythingOfType("int64"), mock.Anything).Return()
			mockMetricHandle.On("FileCacheReadLatency", mock.Anything, mock.AnythingOfType("float64"), mock.Anything).Return()

			reader := NewFileCacheReader(t.object, mockBucket, mockHandler, true, mockMetricHandle)

			n, hit, err := reader.tryReadingFromFileCache(ctx, make([]byte, 10), 0)

			if tc.expectedErr {
				assert.Error(t.T(), err)
			} else {
				assert.NoError(t.T(), err)
			}
			assert.Equal(t.T(), tc.expectedHit, hit)
			assert.Equal(t.T(), tc.expectedN, n)
		})
	}
}
