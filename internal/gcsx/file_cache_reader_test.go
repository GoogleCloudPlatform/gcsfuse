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
	"io"
	"os"
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/file"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/file/downloader"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	testutil "github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	testObject                = "testObject"
	sequentialReadSizeInMb    = 22
	sequentialReadSizeInBytes = sequentialReadSizeInMb * MB
	cacheMaxSize              = 2 * sequentialReadSizeInMb * util.MiB
)

type fileCacheReaderTest struct {
	suite.Suite
	ctx          context.Context
	object       *gcs.MinObject
	mockBucket   *storage.TestifyMockBucket
	cacheDir     string
	jobManager   *downloader.JobManager
	cacheHandler *file.CacheHandler
	reader       *FileCacheReader
}

func TestFileCacheReaderTestSuite(t *testing.T) {
	suite.Run(t, new(fileCacheReaderTest))
}

func (t *fileCacheReaderTest) SetupTest() {
	t.object = &gcs.MinObject{
		Name:       testObject,
		Size:       17,
		Generation: 1234,
	}
	t.mockBucket = new(storage.TestifyMockBucket)
	t.cacheDir = path.Join(os.Getenv("HOME"), "test_cache_dir")
	lruCache := lru.NewCache(cacheMaxSize)
	t.jobManager = downloader.NewJobManager(lruCache, util.DefaultFilePerm, util.DefaultDirPerm, t.cacheDir, sequentialReadSizeInMb, &cfg.FileCacheConfig{EnableCrc: false}, common.NewNoopMetrics())
	t.cacheHandler = file.NewCacheHandler(lruCache, t.jobManager, t.cacheDir, util.DefaultFilePerm, util.DefaultDirPerm)
	t.reader = NewFileCacheReader(t.object, t.mockBucket, t.cacheHandler, true, common.NewNoopMetrics())
	t.ctx = context.Background()
}

func (t *fileCacheReaderTest) TearDown() {
	err := os.RemoveAll(t.cacheDir)
	if err != nil {
		t.T().Logf("Failed to clean up test cache directory '%s': %v", t.cacheDir, err)
	}
}

func (t *fileCacheReaderTest) mockNewReaderWithHandleCallForTestBucket(start uint64, limit uint64, rd gcs.StorageReader) {
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(rg *gcs.ReadObjectRequest) bool {
		return rg != nil && (*rg.Range).Start == start && (*rg.Range).Limit == limit
	})).Return(rd, nil)
}

func (t *fileCacheReaderTest) TestNewFileCacheReader() {
	reader := NewFileCacheReader(t.object, t.mockBucket, t.cacheHandler, true, nil)

	assert.NotNil(t.T(), reader)
	assert.Equal(t.T(), t.object, reader.object)
	assert.Equal(t.T(), t.mockBucket, reader.bucket)
	assert.Equal(t.T(), t.cacheHandler, reader.fileCacheHandler)
	assert.True(t.T(), reader.cacheFileForRangeRead)
	assert.Nil(t.T(), reader.metricHandle)
	assert.Nil(t.T(), reader.fileCacheHandle)
}

func (t *fileCacheReaderTest) Test_ReadAt_NilFileCacheHandlerThrowFallBackError() {
	reader := NewFileCacheReader(t.object, t.mockBucket, nil, true, nil)

	readerResponse, err := reader.ReadAt(t.ctx, make([]byte, 10), 0)

	assert.True(t.T(), errors.Is(err, FallbackToAnotherReader))
	assert.Zero(t.T(), readerResponse.Size)
}

func (t *fileCacheReaderTest) Test_ReadAt_FileSizeIsGreaterThanCacheSize() {
	t.reader.object.Size = 45 * MB
	t.mockBucket.On("Name").Return("test-bucket")

	readerResponse, err := t.reader.ReadAt(t.ctx, make([]byte, t.reader.object.Size), 0)

	assert.True(t.T(), errors.Is(err, FallbackToAnotherReader))
	assert.Zero(t.T(), readerResponse.Size)
}

func (t *fileCacheReaderTest) Test_ReadAt_OffsetGreaterThanFileSizeWillReturnEOF() {
	offset := t.reader.object.Size + 10

	readerResponse, err := t.reader.ReadAt(t.ctx, make([]byte, 10), int64(offset))

	assert.True(t.T(), errors.Is(err, io.EOF), "expected %v error got %v", io.EOF, err)
	assert.Zero(t.T(), readerResponse.Size)
}

func (t *fileCacheReaderTest) Test_tryReadingFromFileCache_CacheHit() {
	testContent := testutil.GenerateRandomBytes(int(t.object.Size))
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, t.object.Size, rd)
	t.mockBucket.On("Name").Return("test-bucket")
	buf := make([]byte, t.object.Size)
	// First read will be a cache miss.
	n, cacheHit, err := t.reader.tryReadingFromFileCache(t.ctx, buf, 0)
	assert.NoError(t.T(), err)
	assert.False(t.T(), cacheHit)
	assert.Equal(t.T(), n, len(buf))

	// Second read will be a cache hit.
	n, cacheHit, err = t.reader.tryReadingFromFileCache(t.ctx, buf, 0)

	assert.NoError(t.T(), err)
	assert.True(t.T(), cacheHit)
	assert.Equal(t.T(), n, len(buf))
	t.mockBucket.AssertExpectations(t.T())
}

func (t *fileCacheReaderTest) Test_tryReadingFromFileCache_SequentialSubsequentReadOffsetLessThanReadChunkSize() {
	t.object.Size = 20 * util.MiB
	testContent := testutil.GenerateRandomBytes(int(t.object.Size))
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, t.object.Size, rd)
	t.mockBucket.On("Name").Return("test-bucket")
	start1 := 0
	end1 := util.MiB
	require.Less(t.T(), end1, int(t.object.Size))
	// First call from offset 0 - sequential read
	buf := make([]byte, end1-start1)
	_, cacheHit, err := t.reader.tryReadingFromFileCache(t.ctx, buf, int64(start1))
	assert.NoError(t.T(), err)
	assert.False(t.T(), cacheHit)
	assert.Equal(t.T(), buf, testContent[start1:end1])
	start2 := 3*util.MiB + 4
	end2 := start2 + util.MiB
	buf2 := make([]byte, end2-start2)

	// Assuming start2 offset download in progress
	_, cacheHit, err = t.reader.tryReadingFromFileCache(t.ctx, buf2, int64(start2))

	assert.NoError(t.T(), err)
	assert.True(t.T(), cacheHit)
	assert.Equal(t.T(), buf2, testContent[start2:end2])
	t.mockBucket.AssertExpectations(t.T())
}

func (t *fileCacheReaderTest) Test_ReadAt_SequentialRangeRead() {
	testContent := testutil.GenerateRandomBytes(int(t.object.Size))
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, t.object.Size, rd)
	t.mockBucket.On("Name").Return("test-bucket")
	start := 0
	end := 10
	require.Less(t.T(), end, int(t.object.Size))
	buf := make([]byte, end-start)

	readerResponse, err := t.reader.ReadAt(t.ctx, buf, int64(start))

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), readerResponse.DataBuf, testContent[start:end])
	t.mockBucket.AssertExpectations(t.T())
}

func (t *fileCacheReaderTest) Test_ReadAt_RandomReadNotStartWithZeroOffsetWhenCacheForRangeReadIsFalse() {
	t.reader.cacheFileForRangeRead = false
	start := 5
	end := 10
	t.mockBucket.On("Name").Return("test-bucket")
	buf := make([]byte, end-start)
	readerResponse, err := t.reader.ReadAt(t.ctx, buf, int64(start))
	assert.True(t.T(), errors.Is(err, FallbackToAnotherReader))
	assert.Zero(t.T(), readerResponse.Size)
	job := t.jobManager.CreateJobIfNotExists(t.object, t.mockBucket)
	jobStatus := job.GetStatus()
	assert.True(t.T(), jobStatus.Name == downloader.NotStarted)

	readerResponse, err = t.reader.ReadAt(t.ctx, buf, int64(start))

	assert.True(t.T(), errors.Is(err, FallbackToAnotherReader))
	assert.Zero(t.T(), readerResponse.Size)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *fileCacheReaderTest) Test_ReadAt_RandomReadNotStartWithZeroOffsetWhenCacheForRangeReadIsTrue() {
	t.reader.cacheFileForRangeRead = true
	testContent := testutil.GenerateRandomBytes(int(t.object.Size))
	start := 5
	end := 10
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	// Mock for download job's NewReader call
	t.mockNewReaderWithHandleCallForTestBucket(0, t.object.Size, rd)
	t.mockBucket.On("Name").Return("test-bucket")
	buf := make([]byte, end-start)

	readerResponse, err := t.reader.ReadAt(t.ctx, buf, int64(start))

	assert.True(t.T(), errors.Is(err, FallbackToAnotherReader))
	assert.Zero(t.T(), readerResponse.Size)
	job := t.jobManager.GetJob(t.object.Name, t.mockBucket.Name())
	assert.True(t.T(), job == nil || job.GetStatus().Name == downloader.Downloading)
	assert.NotNil(t.T(), t.reader.fileCacheHandle)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *fileCacheReaderTest) Test_ReadAt_SequentialToRandomSubsequentReadOffsetMoreThanReadChunkSize() {
	t.object.Size = 20 * util.MiB
	testContent := testutil.GenerateRandomBytes(int(t.object.Size))
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	// Mock for download job's NewReader call
	t.mockNewReaderWithHandleCallForTestBucket(0, t.object.Size, rd)
	t.mockBucket.On("Name").Return("test-bucket")
	start1 := 0
	end1 := util.MiB
	require.Less(t.T(), end1, int(t.object.Size))
	// First call from offset 0 - sequential read
	buf := make([]byte, end1-start1)
	readerResponse, err := t.reader.ReadAt(t.ctx, buf, int64(start1))
	// Served from file cache
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), readerResponse.DataBuf, testContent[start1:end1])
	start2 := 16*util.MiB + 4
	end2 := start2 + util.MiB
	buf2 := make([]byte, end2-start2)

	readerResponse, err = t.reader.ReadAt(t.ctx, buf2, int64(start2))

	// Assuming a download with a start offset of start2 is in progress, a fallback to another reader will be required.
	assert.True(t.T(), errors.Is(err, FallbackToAnotherReader))
	assert.Zero(t.T(), readerResponse.Size)
	job := t.jobManager.GetJob(t.object.Name, t.mockBucket.Name())
	assert.True(t.T(), job == nil || job.GetStatus().Name == downloader.Downloading)
	assert.NotNil(t.T(), t.reader.fileCacheHandle)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *fileCacheReaderTest) Test_ReadAt_SequentialToRandomSubsequentReadOffsetLessThanPrevious() {
	t.object.Size = 20 * util.MiB
	testContent := testutil.GenerateRandomBytes(int(t.object.Size))
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, t.object.Size, rd)
	t.mockBucket.On("Name").Return("test-bucket")
	start1 := 0
	end1 := util.MiB
	require.Less(t.T(), end1, int(t.object.Size))
	// First call from offset 0 - sequential read
	buf := make([]byte, end1-start1)
	readerResponse, err := t.reader.ReadAt(t.ctx, buf, int64(start1))
	// Served from file cache
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), readerResponse.DataBuf, testContent[start1:end1])
	start2 := 16*util.MiB + 4
	end2 := start2 + util.MiB
	buf2 := make([]byte, end2-start2)
	// Assuming a download with a start offset of start2 is in progress, a fallback to another reader will be required.
	readerResponse, err = t.reader.ReadAt(t.ctx, buf2, int64(start2))
	assert.True(t.T(), errors.Is(err, FallbackToAnotherReader))
	assert.Zero(t.T(), readerResponse.Size)
	// Assuming start3 offset is downloaded
	start3 := 4 * util.MiB
	end3 := start3 + util.MiB
	buf3 := make([]byte, end3-start3)

	readerResponse, err = t.reader.ReadAt(t.ctx, buf3, int64(start3))

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), readerResponse.DataBuf, testContent[start3:end3])
	t.mockBucket.AssertExpectations(t.T())
}

func (t *fileCacheReaderTest) Test_ReadAt_CacheMissDueToInvalidJob() {
	testContent := testutil.GenerateRandomBytes(int(t.object.Size))
	rc1 := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, t.object.Size, rc1)
	t.mockBucket.On("Name").Return("test-bucket")
	buf := make([]byte, t.object.Size)
	readerResponse, err := t.reader.ReadAt(t.ctx, buf, 0)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), readerResponse.DataBuf, testContent)
	job := t.jobManager.GetJob(t.object.Name, t.mockBucket.Name())
	if job != nil {
		jobStatus := job.GetStatus().Name
		assert.True(t.T(), jobStatus == downloader.Downloading || jobStatus == downloader.Completed, fmt.Sprintf("the actual status is %v", jobStatus))
	}
	err = t.reader.fileCacheHandler.InvalidateCache(t.object.Name, t.mockBucket.Name())
	assert.NoError(t.T(), err)

	readerResponse, err = t.reader.ReadAt(t.ctx, buf, 0)

	// As job is invalidated Need to get served from GCS reader
	assert.True(t.T(), errors.Is(err, FallbackToAnotherReader))
	assert.Zero(t.T(), readerResponse.Size)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *fileCacheReaderTest) Test_ReadAt_CachePopulatedAndThenCacheMissDueToInvalidJob() {
	testContent := testutil.GenerateRandomBytes(int(t.object.Size))
	rd1 := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, t.object.Size, rd1)
	t.mockBucket.On("Name").Return("test-bucket")
	buf := make([]byte, t.object.Size)
	readerResponse, err := t.reader.ReadAt(t.ctx, buf, 0)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), readerResponse.DataBuf, testContent)
	job := t.jobManager.GetJob(t.object.Name, t.mockBucket.Name())
	if job != nil {
		jobStatus := job.GetStatus().Name
		assert.True(t.T(), jobStatus == downloader.Downloading || jobStatus == downloader.Completed, fmt.Sprintf("the actual status is %v", jobStatus))
	}
	assert.NotNil(t.T(), t.reader.fileCacheHandle)
	err = t.reader.fileCacheHandler.InvalidateCache(t.object.Name, t.mockBucket.Name())
	assert.NoError(t.T(), err)
	// Second reader (rc2) is required, since first reader (rc) is completely read.
	// Reading again will return EOF.
	fmt.Println("Invalidated")
	readerResponse, err = t.reader.ReadAt(t.ctx, buf, 0)
	// As job is invalidated Need to get served from GCS reader
	assert.True(t.T(), errors.Is(err, FallbackToAnotherReader))
	assert.Zero(t.T(), readerResponse.Size)
	assert.Nil(t.T(), t.reader.fileCacheHandle)
	fmt.Println("Invalidated22")
	rd1 = &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, t.object.Size, rd1)

	readerResponse, err = t.reader.ReadAt(t.ctx, buf, 0)

	time.Sleep(2 * time.Second)
	// As job is invalidated Need to get served from GCS reader
	assert.True(t.T(), errors.Is(err, FallbackToAnotherReader))
	assert.Zero(t.T(), readerResponse.Size)
	//if job != nil {
	//	jobStatus := job.GetStatus().Name
	//	assert.True(t.T(), jobStatus == downloader.Downloading, fmt.Sprintf("the actual status is %v", jobStatus))
	//}
	assert.NotNil(t.T(), t.reader.fileCacheHandle)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *fileCacheReaderTest) Test_ReadAt_CachePopulatedAndThenCacheMissDueToInvalidFileHandle() {
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd)
	t.mockBucket.On("Name").Return("test-bucket")
	buf := make([]byte, objectSize)
	readerResponse, err := t.reader.ReadAt(t.ctx, buf, 0)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), readerResponse.DataBuf, testContent)
	assert.NotNil(t.T(), t.reader.fileCacheHandle)
	err = t.reader.fileCacheHandle.Close()
	assert.NoError(t.T(), err)
	readerResponse, err = t.reader.ReadAt(t.ctx, buf, 0)
	assert.True(t.T(), errors.Is(err, FallbackToAnotherReader))
	assert.Zero(t.T(), readerResponse.Size)
	assert.Nil(t.T(), t.reader.fileCacheHandle)
	rc3 := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rc3)

	readerResponse, err = t.reader.ReadAt(t.ctx, buf, 0)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), readerResponse.DataBuf, testContent)
	assert.NotNil(t.T(), t.reader.fileCacheHandle)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *fileCacheReaderTest) Test_ReadAt_IfCacheFileGetsDeleted() {
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd)
	t.mockBucket.On("Name").Return("test-bucket")
	buf := make([]byte, objectSize)
	readerResponse, err := t.reader.ReadAt(t.ctx, buf, 0)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), readerResponse.DataBuf, testContent)
	assert.NotNil(t.T(), t.reader.fileCacheHandle)
	err = t.reader.fileCacheHandle.Close()
	assert.NoError(t.T(), err)
	t.reader.fileCacheHandle = nil
	// Delete the local cache file.
	filePath := util.GetDownloadPath(t.cacheDir, util.GetObjectPath(t.mockBucket.Name(), t.object.Name))
	err = os.Remove(filePath)
	assert.NoError(t.T(), err)

	readerResponse, err = t.reader.ReadAt(t.ctx, buf, 0)

	assert.True(t.T(), errors.Is(err, util.ErrFileNotPresentInCache))
	assert.Zero(t.T(), readerResponse.Size)
}

func (t *fileCacheReaderTest) Test_ReadAt_IfCacheFileGetsDeletedWithCacheHandleOpen() {
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd)
	t.mockBucket.On("Name").Return("test-bucket")
	buf := make([]byte, objectSize)
	readerResponse, err := t.reader.ReadAt(t.ctx, buf, 0)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), readerResponse.DataBuf, testContent)
	assert.NotNil(t.T(), t.reader.fileCacheHandle)
	// Delete the local cache file.
	filePath := util.GetDownloadPath(t.cacheDir, util.GetObjectPath(t.mockBucket.Name(), t.object.Name))
	err = os.Remove(filePath)
	assert.NoError(nil, err)

	// Read via cache only, as we have old fileHandle open and linux
	// doesn't delete the file until the fileHandle count for the file is zero.
	readerResponse, err = t.reader.ReadAt(t.ctx, buf, 0)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), readerResponse.DataBuf, testContent)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *fileCacheReaderTest) Test_ReadAt_FailedJobRestartAndCacheHit() {
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	//rc := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	// First call goes to file cache succeeded by next call to random reader.
	// First NewReaderWithReadHandle-call throws error, hence async job fails.
	// Later NewReader-call returns a valid readCloser object hence fallback to
	// GCS read will succeed.
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(nil, errors.New(""))
	t.mockBucket.On("Name").Return("test-bucket")
	buf := make([]byte, objectSize)
	readerResponse, err := t.reader.ReadAt(t.ctx, buf, 0)
	assert.True(t.T(), errors.Is(err, FallbackToAnotherReader))
	assert.Zero(t.T(), readerResponse.Size)
	job := t.jobManager.GetJob(t.object.Name, t.mockBucket.Name())
	assert.True(t.T(), job == nil || job.GetStatus().Name == downloader.Failed)
	// This call will populate the cache again.
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd)
	readerResponse, err = t.reader.ReadAt(t.ctx, buf, 0)
	assert.True(t.T(), errors.Is(err, FallbackToAnotherReader))
	assert.Zero(t.T(), readerResponse.Size)
	assert.Nil(t.T(), t.reader.fileCacheHandle)

	readerResponse, err = t.reader.ReadAt(t.ctx, buf, 0)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), readerResponse.DataBuf, testContent)
	assert.NotNil(t.T(), t.reader.fileCacheHandle)
	t.mockBucket.AssertExpectations(t.T())
}
