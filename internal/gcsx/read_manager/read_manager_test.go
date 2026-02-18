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
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/bufferedread"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/file"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/file/downloader"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/gcsfuse_errors"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	clientReaders "github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx/client_readers"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	testUtil "github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/workerpool"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/googlecloudplatform/gcsfuse/v3/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/semaphore"
)

const (
	MiB                    = 1024 * 1024
	sequentialReadSizeInMb = 22
	cacheMaxSize           = 2 * sequentialReadSizeInMb * MiB
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (t *readManagerTest) readManagerConfig(fileCacheEnable bool, bufferedReadEnable bool) *ReadManagerConfig {
	config := &ReadManagerConfig{
		SequentialReadSizeMB:  sequentialReadSizeInMb,
		CacheFileForRangeRead: false,
		MetricHandle:          metrics.NewNoopMetrics(),
		MrdWrapper:            nil,
		Config: &cfg.Config{
			Read: cfg.ReadConfig{
				EnableBufferedRead:   bufferedReadEnable,
				MaxBlocksPerHandle:   10,
				BlockSizeMb:          1,
				StartBlocksPerHandle: 2,
				MinBlocksPerHandle:   2,
			},
		},
		GlobalMaxBlocksSem: semaphore.NewWeighted(20),
		InitialOffset:      0,
	}
	if bufferedReadEnable {
		t.workerPool, _ = workerpool.NewStaticWorkerPool(5, 20, 25)
		t.workerPool.Start()
		config.WorkerPool = t.workerPool
	}

	if fileCacheEnable {
		cacheDir := path.Join(os.Getenv("HOME"), "test_cache_dir")
		lruCache := lru.NewCache(cacheMaxSize)
		fileCacheConfig := &cfg.FileCacheConfig{EnableCrc: false}
		jobManager := downloader.NewJobManager(lruCache, util.DefaultFilePerm, util.DefaultDirPerm, cacheDir, sequentialReadSizeInMb, fileCacheConfig, metrics.NewNoopMetrics(), tracing.NewNoopTracer())
		config.FileCacheHandler = file.NewCacheHandler(lruCache, jobManager, cacheDir, util.DefaultFilePerm, util.DefaultDirPerm, "", "", false)
	} else {
		config.FileCacheHandler = nil
	}
	return config
}

func (t *readManagerTest) mockNewReaderWithHandleCallForTestBucket(start uint64, limit uint64, rd gcs.StorageReader) {
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(rg *gcs.ReadObjectRequest) bool {
		return rg != nil && (*rg.Range).Start == start && (*rg.Range).Limit == limit
	})).Return(rd, nil).Once()
}

func getReadCloser(content []byte) io.ReadCloser {
	r := bytes.NewReader(content)
	rc := io.NopCloser(r)
	return rc
}

func (t *readManagerTest) readAt(dst []byte, offset int64) (gcsx.ReadResponse, error) {
	t.readManager.CheckInvariants()
	defer t.readManager.CheckInvariants()
	return t.readManager.ReadAt(t.ctx, &gcsx.ReadRequest{
		Buffer: dst,
		Offset: offset,
	})
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
	bucketType  gcs.BucketType
	workerPool  workerpool.WorkerPool
}

func TestNonZonalBucketReadManagerTestSuite(t *testing.T) {
	suite.Run(t, &readManagerTest{bucketType: gcs.BucketType{}})
}

func TestZonalBucketReadManagerTestSuite(t *testing.T) {
	suite.Run(t, &readManagerTest{bucketType: gcs.BucketType{Zonal: true, Hierarchical: true}})
}

func (t *readManagerTest) SetupTest() {
	t.object = &gcs.MinObject{
		Name:       "testObject",
		Size:       17,
		Generation: 1234,
	}
	t.mockBucket = new(storage.TestifyMockBucket)
	t.ctx = context.Background()
	t.readManager = NewReadManager(t.object, t.mockBucket, t.readManagerConfig(true, false))
}

func (t *readManagerTest) TearDownTest() {
	t.readManager.Destroy()
	if t.workerPool != nil {
		t.workerPool.Stop()
		t.workerPool = nil
	}
}

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////
func (t *readManagerTest) Test_NewReadManager_WithFileCacheHandlerOnly() {
	config := t.readManagerConfig(true, false)

	rm := NewReadManager(t.object, t.mockBucket, config)

	assert.Equal(t.T(), t.object, rm.Object())
	assert.Len(t.T(), rm.readers, 2)
	_, ok1 := rm.readers[0].(*gcsx.FileCacheReader)
	_, ok2 := rm.readers[1].(*clientReaders.GCSReader)
	assert.True(t.T(), ok1, "First reader should be FileCacheReader")
	assert.True(t.T(), ok2, "Second reader should be GCSReader")
}

func (t *readManagerTest) Test_NewReadManager_WithoutFileCacheAndBufferedRead() {
	config := t.readManagerConfig(false, false)

	rm := NewReadManager(t.object, t.mockBucket, config)

	assert.Equal(t.T(), t.object, rm.Object())
	assert.Len(t.T(), rm.readers, 1)
	_, ok := rm.readers[0].(*clientReaders.GCSReader)
	assert.True(t.T(), ok, "Only reader should be GCSReader")
}

func (t *readManagerTest) Test_NewReadManager_WithBufferedRead() {
	config := t.readManagerConfig(false, true)

	rm := NewReadManager(t.object, t.mockBucket, config)

	assert.Equal(t.T(), t.object, rm.Object())
	assert.Len(t.T(), rm.readers, 2) // BufferedReader and GCSReader
	_, ok1 := rm.readers[0].(*bufferedread.BufferedReader)
	_, ok2 := rm.readers[1].(*clientReaders.GCSReader)
	assert.True(t.T(), ok1, "First reader should be BufferedReader")
	assert.True(t.T(), ok2, "Second reader should be GCSReader")
}

func (t *readManagerTest) Test_NewReadManager_WithFileCacheAndBufferedRead() {
	config := t.readManagerConfig(true, true)
	defer os.RemoveAll(path.Join(os.Getenv("HOME"), "test_cache_dir"))

	rm := NewReadManager(t.object, t.mockBucket, config)

	assert.Equal(t.T(), t.object, rm.Object())
	assert.Len(t.T(), rm.readers, 3) // FileCacheReader, BufferedReader, GCSReader
	_, ok1 := rm.readers[0].(*gcsx.FileCacheReader)
	_, ok2 := rm.readers[1].(*bufferedread.BufferedReader)
	_, ok3 := rm.readers[2].(*clientReaders.GCSReader)
	assert.True(t.T(), ok1, "First reader should be FileCacheReader")
	assert.True(t.T(), ok2, "Second reader should be BufferedReader")
	assert.True(t.T(), ok3, "Third reader should be GCSReader")
}

func (t *readManagerTest) Test_NewReadManager_BufferedReaderCreationFails() {
	config := t.readManagerConfig(false, true)
	// Exhaust the semaphore
	config.GlobalMaxBlocksSem = semaphore.NewWeighted(0)

	rm := NewReadManager(t.object, t.mockBucket, config)

	assert.Equal(t.T(), t.object, rm.Object())
	assert.Len(t.T(), rm.readers, 1) // Only GCSReader
	_, ok := rm.readers[0].(*clientReaders.GCSReader)
	assert.True(t.T(), ok, "Only reader should be GCSReader")
}

func (t *readManagerTest) Test_ReadAt_EmptyRead() {
	// Nothing should happen.
	readResponse, err := t.readAt(make([]byte, 0), 0)

	assert.NoError(t.T(), err)
	assert.Zero(t.T(), readResponse.Size)
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
			readResponse, err := t.readAt(make([]byte, 1), tc.offset)

			assert.Zero(t.T(), readResponse.Size)
			assert.True(t.T(), errors.Is(err, io.EOF), "expected %v error got %v", io.EOF, err)
		})
	}
}

func (t *readManagerTest) Test_ReadAt_NoExistingReader() {
	// The bucket should be called to set up a new reader.
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(nil, errors.New("network error"))
	t.mockBucket.On("BucketType", mock.Anything).Return(t.bucketType)
	t.mockBucket.On("Name").Return("test-bucket")

	_, err := t.readAt(make([]byte, 1), 0)

	assert.Error(t.T(), err)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *readManagerTest) Test_ReadAt_ReaderFailsWithTimeout() {
	t.readManager = NewReadManager(t.object, t.mockBucket, t.readManagerConfig(false, false))
	r := iotest.OneByteReader(iotest.TimeoutReader(strings.NewReader("xxx")))
	rc := &fake.FakeReader{ReadCloser: io.NopCloser(r)}
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(rc, nil).Once()
	t.mockBucket.On("BucketType", mock.Anything).Return(t.bucketType).Times(2)

	_, err := t.readAt(make([]byte, 3), 0)

	assert.Error(t.T(), err)
	assert.Contains(t.T(), err.Error(), "timeout")
	t.mockBucket.AssertExpectations(t.T())
}

func (t *readManagerTest) Test_ReadAt_FileClobbered() {
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(nil, &gcs.NotFoundError{})
	t.mockBucket.On("BucketType", mock.Anything).Return(t.bucketType).Times(3)
	t.mockBucket.On("Name").Return("test-bucket")

	_, err := t.readAt(make([]byte, 3), 0)

	assert.Error(t.T(), err)
	var clobberedErr *gcsfuse_errors.FileClobberedError
	assert.True(t.T(), errors.As(err, &clobberedErr))
	t.mockBucket.AssertExpectations(t.T())
}

func (t *readManagerTest) Test_ReadAt_FullObjectFromCache() {
	objectSize := int(t.object.Size)
	expectedData := testUtil.GenerateRandomBytes(objectSize)
	fakeReader := &fake.FakeReader{
		ReadCloser: getReadCloser(expectedData),
	}
	// Mock the reader that returns full object data
	t.mockNewReaderWithHandleCallForTestBucket(0, t.object.Size, fakeReader)
	t.mockBucket.On("Name").Return("test-bucket").Maybe()
	t.mockBucket.On("BucketType").Return(t.bucketType)
	buf := make([]byte, objectSize)

	// Act: First read (expected to be served via GCS, populating the cache)
	firstResp, err := t.readAt(buf, 0)

	// Assert: First read succeeds and returns expected data
	assert.NoError(t.T(), err, "First read should not return an error")
	assert.Equal(t.T(), objectSize, firstResp.Size)
	assert.Equal(t.T(), expectedData, buf, "First read should return expected data")

	clear(buf)

	// Act: Second read (should be served from cache)
	secondResp, err := t.readAt(buf, 0)

	// Assert: Second read also succeeds and returns the same cached data
	assert.NoError(t.T(), err, "Second read (from cache) should not return an error")
	assert.Equal(t.T(), objectSize, secondResp.Size)
	assert.Equal(t.T(), expectedData, buf, "Second read should return cached data")
	// Verify that bucket mock expectations are met
	t.mockBucket.AssertExpectations(t.T())
}

func (t *readManagerTest) Test_ReadAt_R1FailsR2Succeeds() {
	offset := int64(0)
	buf := make([]byte, 10)
	expectedResp := gcsx.ReadResponse{Size: 10}
	mockReader1 := new(gcsx.MockReader)
	mockReader2 := new(gcsx.MockReader)
	rm := &ReadManager{
		object:             t.object,
		readers:            []gcsx.Reader{mockReader1, mockReader2},
		readTypeClassifier: gcsx.NewReadTypeClassifier(sequentialReadSizeInMb, 0),
	}
	mockReader1.On("ReadAt", t.ctx, mock.AnythingOfType("*gcsx.ReadRequest")).Return(gcsx.ReadResponse{}, gcsx.FallbackToAnotherReader).Once()
	mockReader1.On("Destroy").Once()
	mockReader2.On("ReadAt", t.ctx, mock.AnythingOfType("*gcsx.ReadRequest")).Return(expectedResp, nil).Once()
	mockReader2.On("Destroy").Once()

	resp, err := rm.ReadAt(t.ctx, &gcsx.ReadRequest{
		Buffer: buf,
		Offset: offset,
	})
	rm.Destroy()

	assert.NoError(t.T(), err, "expected no error when second reader succeeds")
	assert.Equal(t.T(), expectedResp, resp, "expected response from second reader")
	mockReader1.AssertExpectations(t.T())
	mockReader2.AssertExpectations(t.T())
}

func (t *readManagerTest) Test_ReadAt_BufferedReaderFallsBack() {
	offset := int64(0)
	buf := make([]byte, 10)
	mockBufferedReader := new(gcsx.MockReader)
	mockGCSReader := new(gcsx.MockReader)
	rm := &ReadManager{
		object:             t.object,
		readers:            []gcsx.Reader{mockBufferedReader, mockGCSReader},
		readTypeClassifier: gcsx.NewReadTypeClassifier(sequentialReadSizeInMb, 0),
	}
	mockBufferedReader.On("ReadAt", t.ctx, mock.AnythingOfType("*gcsx.ReadRequest")).Return(gcsx.ReadResponse{}, gcsx.FallbackToAnotherReader).Once()
	mockBufferedReader.On("Destroy").Once()
	mockGCSReader.On("ReadAt", t.ctx, mock.AnythingOfType("*gcsx.ReadRequest")).Return(gcsx.ReadResponse{Size: 10}, nil).Once()
	mockGCSReader.On("Destroy").Once()

	resp, err := rm.ReadAt(t.ctx, &gcsx.ReadRequest{
		Buffer: buf,
		Offset: offset,
	})
	rm.Destroy()

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), gcsx.ReadResponse{Size: 10}, resp)
	mockBufferedReader.AssertExpectations(t.T())
	mockGCSReader.AssertExpectations(t.T())
}
