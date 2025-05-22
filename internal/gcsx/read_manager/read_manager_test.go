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

package read_manager

import (
	"context"
	"errors"
	"io"
	"os"
	"path"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/file"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/file/downloader"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/fs/gcsfuse_errors"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx"
	clientReaders "github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx/client_readers"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

const (
	MiB                    = 1024 * 1024
	sequentialReadSizeInMb = 22
	cacheMaxSize           = 2 * sequentialReadSizeInMb * MiB
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (t *readManagerTest) readManagerConfig(fileCacheEnable bool) *ReadManagerConfig {
	config := &ReadManagerConfig{
		Object:                t.object,
		Bucket:                t.mockBucket,
		SequentialReadSizeMB:  sequentialReadSizeInMb,
		CacheFileForRangeRead: false,
		MetricHandle:          common.NewNoopMetrics(),
		MrdWrapper:            nil,
	}
	if fileCacheEnable {
		cacheDir := path.Join(os.Getenv("HOME"), "test_cache_dir")
		lruCache := lru.NewCache(cacheMaxSize)
		jobManager := downloader.NewJobManager(lruCache, util.DefaultFilePerm, util.DefaultDirPerm, cacheDir, sequentialReadSizeInMb, &cfg.FileCacheConfig{EnableCrc: false}, common.NewNoopMetrics())
		config.FileCacheHandler = file.NewCacheHandler(lruCache, jobManager, cacheDir, util.DefaultFilePerm, util.DefaultDirPerm)
	} else {
		config.FileCacheHandler = nil
	}
	return config
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type readManagerTest struct {
	suite.Suite
	object      *gcs.MinObject
	mockBucket  *storage.TestifyMockBucket
	readManager *ReadManager
	ctx         context.Context
}

func TestReadManagerTestSuite(t *testing.T) {
	suite.Run(t, new(readManagerTest))
}

func (t *readManagerTest) SetupTest() {
	t.object = &gcs.MinObject{
		Name:       "testObject",
		Size:       17,
		Generation: 1234,
	}
	t.mockBucket = new(storage.TestifyMockBucket)
	t.ctx = context.Background()
	t.readManager = NewReadManager(t.readManagerConfig(true))
}

func (t *readManagerTest) TearDownTest() {
	t.readManager.Destroy()
}

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////
func (t *readManagerTest) Test_NewReadManager_WithFileCacheHandler() {
	config := t.readManagerConfig(true)

	rm := NewReadManager(config)

	assert.Equal(t.T(), t.object, rm.Object())
	assert.Len(t.T(), rm.readers, 2)
	_, ok1 := rm.readers[0].(*gcsx.FileCacheReader)
	_, ok2 := rm.readers[1].(*clientReaders.GCSReader)
	assert.True(t.T(), ok1, "First reader should be FileCacheReader")
	assert.True(t.T(), ok2, "Second reader should be GCSReader")
}

func (t *readManagerTest) Test_NewReadManager_WithoutFileCacheHandler() {
	config := t.readManagerConfig(false)

	rm := NewReadManager(config)

	assert.Equal(t.T(), t.object, rm.Object())
	assert.Len(t.T(), rm.readers, 1)
	_, ok := rm.readers[0].(*clientReaders.GCSReader)
	assert.True(t.T(), ok, "Only reader should be GCSReader")
}

func (t *readManagerTest) Test_ReadAt_EmptyRead() {
	// Nothing should happen.
	readerResponse, err := t.readManager.ReadAt(t.ctx, make([]byte, 0), 0)

	assert.NoError(t.T(), err)
	assert.Zero(t.T(), readerResponse.Size)
}

func (t *readManagerTest) Test_ReadAt_InvalidOffset() {
	tests := []struct {
		name   string
		offset int64
	}{
		{
			name:   "ReadAtEndOfObject",
			offset: int64(t.object.Size),
		},
		{
			name:   "ReadPastEndOfObject",
			offset: int64(t.object.Size) + 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func() {
			readerResponse, err := t.readManager.ReadAt(t.ctx, make([]byte, 1), tc.offset)

			assert.Zero(t.T(), readerResponse.Size)
			assert.True(t.T(), errors.Is(err, io.EOF), "expected %v error got %v", io.EOF, err)
		})
	}
}

func (t *readManagerTest) Test_ReadAt_NoExistingReader() {
	// The bucket should be called to set up a new reader.
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(nil, errors.New("network error"))
	t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{}).Times(1)
	t.mockBucket.On("Name").Return("test-bucket")

	_, err := t.readManager.ReadAt(t.ctx, make([]byte, 1), 0)

	assert.Error(t.T(), err)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *readManagerTest) Test_ReadAt_ReaderFailsWithTimeout() {
	t.readManager = NewReadManager(t.readManagerConfig(false))
	r := iotest.OneByteReader(iotest.TimeoutReader(strings.NewReader("xxx")))
	rc := &fake.FakeReader{ReadCloser: io.NopCloser(r)}
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(rc, nil).Once()
	t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{}).Times(1)

	_, err := t.readManager.ReadAt(t.ctx, make([]byte, 3), 0)

	assert.Error(t.T(), err)
	assert.Contains(t.T(), err.Error(), "timeout")
	t.mockBucket.AssertExpectations(t.T())
}

func (t *readManagerTest) Test_ReadAt_FileClobbered() {
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(nil, &gcs.NotFoundError{})
	t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{}).Times(1)
	t.mockBucket.On("Name").Return("test-bucket")

	_, err := t.readManager.ReadAt(t.ctx, make([]byte, 3), 1)

	assert.Error(t.T(), err)
	var clobberedErr *gcsfuse_errors.FileClobberedError
	assert.True(t.T(), errors.As(err, &clobberedErr))
	t.mockBucket.AssertExpectations(t.T())
}
