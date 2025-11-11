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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/file"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/file/downloader"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
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

const (
	testObject                = "testObject"
	testObject_unfinalized    = "testObject_unfinalized"
	sequentialReadSizeInMb    = 22
	sequentialReadSizeInBytes = sequentialReadSizeInMb * MiB
	cacheMaxSize              = 2 * sequentialReadSizeInMb * MiB
)

type fileCacheReaderTest struct {
	suite.Suite
	ctx                       context.Context
	object                    *gcs.MinObject
	unfinalized_object        *gcs.MinObject
	mockBucket                *storage.TestifyMockBucket
	cacheDir                  string
	jobManager                *downloader.JobManager
	cacheHandler              *file.CacheHandler
	reader                    *FileCacheReader
	reader_unfinalized_object *FileCacheReader
	bucketType                gcs.BucketType
}

func TestNonZonalBucketFileCacheReaderTestSuite(t *testing.T) {
	nonZonalBucketFileCacheReaderTestSuite := &fileCacheReaderTest{
		bucketType: gcs.BucketType{}}
	suite.Run(t, nonZonalBucketFileCacheReaderTestSuite)
}

func TestZonalFileCacheReaderTestSuite(t *testing.T) {
	zonalBucketFileCacheReaderTestSuite := &fileCacheReaderTest{
		bucketType: gcs.BucketType{Zonal: true, Hierarchical: true}}
	suite.Run(t, zonalBucketFileCacheReaderTestSuite)
}

func (t *fileCacheReaderTest) SetupTest() {
	t.object = &gcs.MinObject{
		Name:       testObject,
		Size:       17,
		Generation: 1234,
		Finalized:  time.Date(2025, time.June, 27, 07, 22, 30, 0, time.UTC),
	}
	t.unfinalized_object = &gcs.MinObject{
		Name:       testObject_unfinalized,
		Size:       17,
		Generation: 1234,
	}
	t.mockBucket = new(storage.TestifyMockBucket)
	t.cacheDir = path.Join(os.Getenv("HOME"), "test_cache_dir")
	lruCache := lru.NewCache(cacheMaxSize)
	t.jobManager = downloader.NewJobManager(lruCache, util.DefaultFilePerm, util.DefaultDirPerm, t.cacheDir, sequentialReadSizeInMb, &cfg.FileCacheConfig{EnableCrc: false}, metrics.NewNoopMetrics())
	t.cacheHandler = file.NewCacheHandler(lruCache, t.jobManager, t.cacheDir, util.DefaultFilePerm, util.DefaultDirPerm, "", "")
	t.reader = NewFileCacheReader(t.object, t.mockBucket, t.cacheHandler, true, metrics.NewNoopMetrics())
	t.reader_unfinalized_object = NewFileCacheReader(t.unfinalized_object, t.mockBucket, t.cacheHandler, true, metrics.NewNoopMetrics())
	t.ctx = context.Background()
}

func (t *fileCacheReaderTest) TearDownTest() {
	err := os.RemoveAll(t.cacheDir)
	if err != nil {
		t.T().Logf("Failed to clean up test cache directory '%s': %v", t.cacheDir, err)
	}
	t.reader.Destroy()
}

func (t *fileCacheReaderTest) mockNewReaderWithHandleCallForTestBucket(limit uint64, rd gcs.StorageReader) {
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(rg *gcs.ReadObjectRequest) bool {
		return rg != nil && (*rg.Range).Start == 0 && (*rg.Range).Limit == limit
	})).Return(rd, nil).Once()
}

func getReadCloser(content []byte) io.ReadCloser {
	r := bytes.NewReader(content)
	rc := io.NopCloser(r)
	return rc
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

	readResponse, err := reader.ReadAt(t.ctx, make([]byte, 10), 0)

	assert.True(t.T(), errors.Is(err, FallbackToAnotherReader), "expected %v error got %v", FallbackToAnotherReader, err)
	assert.Zero(t.T(), readResponse.Size)
}

func (t *fileCacheReaderTest) Test_ReadAt_FileSizeIsGreaterThanCacheSize() {
	t.object.Size = cacheMaxSize + 5
	t.mockBucket.On("Name").Return("test-bucket")

	readResponse, err := t.reader.ReadAt(t.ctx, make([]byte, t.object.Size), 0)

	assert.True(t.T(), errors.Is(err, FallbackToAnotherReader), "expected %v error got %v", FallbackToAnotherReader, err)
	assert.Zero(t.T(), readResponse.Size)
}

func (t *fileCacheReaderTest) Test_ReadAt_OffsetGreaterThanFileSizeWillReturnEOF() {
	offset := t.object.Size + 10

	readResponse, err := t.reader.ReadAt(t.ctx, make([]byte, 10), int64(offset))

	assert.True(t.T(), errors.Is(err, io.EOF), "expected %v error got %v", io.EOF, err)
	assert.Zero(t.T(), readResponse.Size)
}

func (t *fileCacheReaderTest) Test_tryReadingFromFileCache_CacheHit() {
	testContent := testutil.GenerateRandomBytes(int(t.object.Size))
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(t.object.Size, rd)
	t.mockBucket.On("Name").Return("test-bucket")
	t.mockBucket.On("BucketType").Return(t.bucketType)
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
	t.mockNewReaderWithHandleCallForTestBucket(t.object.Size, rd)
	t.mockBucket.On("Name").Return("test-bucket")
	t.mockBucket.On("BucketType").Return(t.bucketType)
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
	t.mockNewReaderWithHandleCallForTestBucket(t.object.Size, rd)
	t.mockBucket.On("Name").Return("test-bucket")
	t.mockBucket.On("BucketType").Return(t.bucketType)
	start := 0
	end := 10
	require.Less(t.T(), end, int(t.object.Size))
	buf := make([]byte, end-start)

	readResponse, err := t.reader.ReadAt(t.ctx, buf, int64(start))

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), testContent[start:end], buf[:readResponse.Size])
	t.mockBucket.AssertExpectations(t.T())
}

func (t *fileCacheReaderTest) Test_ReadAt_RandomReadNotStartWithZeroOffsetWhenCacheForRangeReadIsFalse() {
	t.reader.cacheFileForRangeRead = false
	start := 5
	end := 10
	t.mockBucket.On("Name").Return("test-bucket")
	buf := make([]byte, end-start)
	readResponse, err := t.reader.ReadAt(t.ctx, buf, int64(start))
	assert.True(t.T(), errors.Is(err, FallbackToAnotherReader), "expected %v error got %v", FallbackToAnotherReader, err)
	assert.Zero(t.T(), readResponse.Size)
	job := t.jobManager.CreateJobIfNotExists(t.object, t.mockBucket)
	jobStatus := job.GetStatus()
	assert.True(t.T(), jobStatus.Name == downloader.NotStarted)

	readResponse, err = t.reader.ReadAt(t.ctx, buf, int64(start))

	assert.True(t.T(), errors.Is(err, FallbackToAnotherReader), "expected %v error got %v", FallbackToAnotherReader, err)
	assert.Zero(t.T(), readResponse.Size)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *fileCacheReaderTest) Test_ReadAt_RandomReadNotStartWithZeroOffsetWhenCacheForRangeReadIsTrue() {
	t.reader.cacheFileForRangeRead = true
	testContent := testutil.GenerateRandomBytes(int(t.object.Size))
	start := 5
	end := 10
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	// Mock for download job's NewReader call
	t.mockNewReaderWithHandleCallForTestBucket(t.object.Size, rd)
	t.mockBucket.On("Name").Return("test-bucket")
	t.mockBucket.On("BucketType").Return(t.bucketType)
	buf := make([]byte, end-start)

	readResponse, err := t.reader.ReadAt(t.ctx, buf, int64(start))

	assert.True(t.T(), errors.Is(err, FallbackToAnotherReader), "expected %v error got %v", FallbackToAnotherReader, err)
	assert.Zero(t.T(), readResponse.Size)
	job := t.jobManager.GetJob(t.object.Name, t.mockBucket.Name())
	assert.True(t.T(), job == nil || job.GetStatus().Name == downloader.Downloading)
	assert.NotNil(t.T(), t.reader.fileCacheHandle)
}

func (t *fileCacheReaderTest) Test_ReadAt_SequentialToRandomSubsequentReadOffsetMoreThanReadChunkSize() {
	t.object.Size = 20 * util.MiB
	testContent := testutil.GenerateRandomBytes(int(t.object.Size))
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	// Mock for download job's NewReader call
	t.mockNewReaderWithHandleCallForTestBucket(t.object.Size, rd)
	t.mockBucket.On("Name").Return("test-bucket")
	t.mockBucket.On("BucketType").Return(t.bucketType)
	start1 := 0
	end1 := util.MiB
	require.Less(t.T(), end1, int(t.object.Size))
	// First call from offset 0 - sequential read
	buf := make([]byte, end1-start1)
	readResponse, err := t.reader.ReadAt(t.ctx, buf, int64(start1))
	// Served from file cache
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), testContent[start1:end1], buf[:readResponse.Size])
	start2 := 16*util.MiB + 4
	end2 := start2 + util.MiB
	buf2 := make([]byte, end2-start2)

	readResponse, err = t.reader.ReadAt(t.ctx, buf2, int64(start2))

	// Assuming a download with a start offset of start2 is in progress, a fallback to another reader will be required.
	assert.True(t.T(), errors.Is(err, FallbackToAnotherReader), "expected %v error got %v", FallbackToAnotherReader, err)
	assert.Zero(t.T(), readResponse.Size)
	job := t.jobManager.GetJob(t.object.Name, t.mockBucket.Name())
	assert.True(t.T(), job == nil || job.GetStatus().Name == downloader.Downloading)
	assert.NotNil(t.T(), t.reader.fileCacheHandle)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *fileCacheReaderTest) Test_ReadAt_SequentialToRandomSubsequentReadOffsetLessThanPrevious() {
	t.object.Size = 20 * util.MiB
	testContent := testutil.GenerateRandomBytes(int(t.object.Size))
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(t.object.Size, rd)
	t.mockBucket.On("Name").Return("test-bucket")
	t.mockBucket.On("BucketType").Return(t.bucketType)
	start1 := 0
	end1 := util.MiB
	require.Less(t.T(), end1, int(t.object.Size))
	// First call from offset 0 - sequential read
	buf := make([]byte, end1-start1)
	readResponse, err := t.reader.ReadAt(t.ctx, buf, int64(start1))
	// Served from file cache
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), testContent[start1:end1], buf[:readResponse.Size])
	start2 := 16*util.MiB + 4
	end2 := start2 + util.MiB
	buf2 := make([]byte, end2-start2)
	// Assuming a download with a start offset of start2 is in progress, a fallback to another reader will be required.
	readResponse, err = t.reader.ReadAt(t.ctx, buf2, int64(start2))
	assert.True(t.T(), errors.Is(err, FallbackToAnotherReader), "expected %v error got %v", FallbackToAnotherReader, err)
	assert.Zero(t.T(), readResponse.Size)
	// Assuming start3 offset is downloaded
	start3 := 4 * util.MiB
	end3 := start3 + util.MiB
	buf3 := make([]byte, end3-start3)

	readResponse, err = t.reader.ReadAt(t.ctx, buf3, int64(start3))

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), testContent[start3:end3], buf3)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *fileCacheReaderTest) Test_ReadAt_CacheMissDueToInvalidJob() {
	testContent := testutil.GenerateRandomBytes(int(t.object.Size))
	rc1 := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(t.object.Size, rc1)
	t.mockBucket.On("Name").Return("test-bucket")
	t.mockBucket.On("BucketType").Return(t.bucketType)
	buf := make([]byte, t.object.Size)
	readResponse, err := t.reader.ReadAt(t.ctx, buf, 0)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), testContent, buf[:readResponse.Size])
	job := t.jobManager.GetJob(t.object.Name, t.mockBucket.Name())
	if job != nil {
		jobStatus := job.GetStatus().Name
		assert.True(t.T(), jobStatus == downloader.Downloading || jobStatus == downloader.Completed, fmt.Sprintf("the actual status is %v", jobStatus))
	}
	err = t.reader.fileCacheHandler.InvalidateCache(t.object.Name, t.mockBucket.Name())
	assert.NoError(t.T(), err)

	readResponse, err = t.reader.ReadAt(t.ctx, buf, 0)

	// As job is invalidated Need to get served from GCS reader
	assert.True(t.T(), errors.Is(err, FallbackToAnotherReader), "expected %v error got %v", FallbackToAnotherReader, err)
	assert.Zero(t.T(), readResponse.Size)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *fileCacheReaderTest) Test_ReadAt_CachePopulatedAndThenCacheMissDueToInvalidJob() {
	testContent := testutil.GenerateRandomBytes(int(t.object.Size))
	// First successful read with cache
	rd1 := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(t.object.Size, rd1)
	t.mockBucket.On("Name").Return("test-bucket")
	t.mockBucket.On("BucketType").Return(t.bucketType)
	buf := make([]byte, t.object.Size)
	readResponse, err := t.reader.ReadAt(t.ctx, buf, 0)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), testContent, buf)
	job := t.jobManager.GetJob(t.object.Name, t.mockBucket.Name())
	if job != nil {
		jobStatus := job.GetStatus().Name
		assert.True(t.T(), jobStatus == downloader.Downloading || jobStatus == downloader.Completed, fmt.Sprintf("the actual status is %v", jobStatus))
	}
	assert.NotNil(t.T(), t.reader.fileCacheHandle)
	// Invalidate the cache to simulate cache miss
	err = t.reader.fileCacheHandler.InvalidateCache(t.object.Name, t.mockBucket.Name())
	assert.NoError(t.T(), err)
	readResponse, err = t.reader.ReadAt(t.ctx, make([]byte, t.object.Size), 0)
	assert.True(t.T(), errors.Is(err, FallbackToAnotherReader), "expected %v error got %v", FallbackToAnotherReader, err)
	assert.Zero(t.T(), readResponse.Size)
	assert.Nil(t.T(), t.reader.fileCacheHandle)
	rd2 := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(t.object.Size, rd2)
	clear(buf)

	readResponse, err = t.reader.ReadAt(t.ctx, buf, 0)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), testContent, buf[:readResponse.Size])
	assert.NotNil(t.T(), t.reader.fileCacheHandle)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *fileCacheReaderTest) Test_ReadAt_CachePopulatedAndThenCacheMissDueToInvalidFileHandleAfterThenCacheHitWithNewFileCacheHandle() {
	testContent := testutil.GenerateRandomBytes(int(t.object.Size))
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(t.object.Size, rd)
	t.mockBucket.On("Name").Return("test-bucket")
	t.mockBucket.On("BucketType").Return(t.bucketType)
	buf := make([]byte, t.object.Size)
	readResponse, err := t.reader.ReadAt(t.ctx, buf, 0)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), testContent, buf[:readResponse.Size])
	assert.NotNil(t.T(), t.reader.fileCacheHandle)
	err = t.reader.fileCacheHandle.Close()
	assert.NoError(t.T(), err)
	readResponse, err = t.reader.ReadAt(t.ctx, make([]byte, t.object.Size), 0)
	assert.True(t.T(), errors.Is(err, FallbackToAnotherReader), "expected %v error got %v", FallbackToAnotherReader, err)
	assert.Zero(t.T(), readResponse.Size)
	assert.Nil(t.T(), t.reader.fileCacheHandle)
	clear(buf)

	readResponse, err = t.reader.ReadAt(t.ctx, buf, 0)

	// Reading from file cache with new file cache handle.
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), testContent, buf[:readResponse.Size])
	assert.NotNil(t.T(), t.reader.fileCacheHandle)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *fileCacheReaderTest) Test_ReadAt_IfCacheFileGetsDeleted() {
	testContent := testutil.GenerateRandomBytes(int(t.object.Size))
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(t.object.Size, rd)
	t.mockBucket.On("Name").Return("test-bucket")
	t.mockBucket.On("BucketType").Return(t.bucketType)
	buf := make([]byte, t.object.Size)
	readResponse, err := t.reader.ReadAt(t.ctx, buf, 0)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), testContent, buf[:readResponse.Size])
	assert.NotNil(t.T(), t.reader.fileCacheHandle)
	err = t.reader.fileCacheHandle.Close()
	assert.NoError(t.T(), err)
	t.reader.fileCacheHandle = nil
	// Delete the local cache file.
	filePath := util.GetDownloadPath(t.cacheDir, util.GetObjectPath(t.mockBucket.Name(), t.object.Name))
	err = os.Remove(filePath)
	assert.NoError(t.T(), err)

	readResponse, err = t.reader.ReadAt(t.ctx, make([]byte, t.object.Size), 0)

	assert.True(t.T(), errors.Is(err, util.ErrFileNotPresentInCache))
	assert.Zero(t.T(), readResponse.Size)
}

func (t *fileCacheReaderTest) Test_ReadAt_IfCacheFileGetsDeletedWithCacheHandleOpen() {
	testContent := testutil.GenerateRandomBytes(int(t.object.Size))
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(t.object.Size, rd)
	t.mockBucket.On("Name").Return("test-bucket")
	t.mockBucket.On("BucketType").Return(t.bucketType)
	buf := make([]byte, t.object.Size)
	readResponse, err := t.reader.ReadAt(t.ctx, buf, 0)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), testContent, buf[:readResponse.Size])
	assert.NotNil(t.T(), t.reader.fileCacheHandle)
	// Delete the local cache file.
	filePath := util.GetDownloadPath(t.cacheDir, util.GetObjectPath(t.mockBucket.Name(), t.object.Name))
	err = os.Remove(filePath)
	assert.NoError(nil, err)
	clear(buf)

	// Read via cache only, as we have old fileHandle open and linux
	// doesn't delete the file until the fileHandle count for the file is zero.
	readResponse, err = t.reader.ReadAt(t.ctx, buf, 0)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), testContent, buf[:readResponse.Size])
	t.mockBucket.AssertExpectations(t.T())
}

func (t *fileCacheReaderTest) Test_ReadAt_FailedJobNextReadCreatesNewJobAndCacheHit() {
	testContent := testutil.GenerateRandomBytes(int(t.object.Size))
	// First NewReaderWithReadHandle call fails, simulating a failed attempt to read from GCS.
	// This triggers a fallback to GCS reader.
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(nil, errors.New("")).Once()
	t.mockBucket.On("Name").Return("test-bucket")
	t.mockBucket.On("BucketType").Return(t.bucketType)
	// First ReadAt call:
	// - Should result in a FallbackToAnotherReader error.
	// - No data should be returned.
	// - The job should be marked as failed (if jobManager is functioning correctly).
	readResponse, err := t.reader.ReadAt(t.ctx, make([]byte, t.object.Size), 0)
	assert.True(t.T(), errors.Is(err, FallbackToAnotherReader), "expected %v error got %v", FallbackToAnotherReader, err)
	assert.Zero(t.T(), readResponse.Size)
	job := t.jobManager.GetJob(t.object.Name, t.mockBucket.Name())
	assert.True(t.T(), job == nil || job.GetStatus().Name == downloader.Failed)
	rc := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(t.object.Size, rc)
	buf := make([]byte, t.object.Size)
	// Second ReadAt call: The file cache should be populated as a result of this successful read.
	readResponse, err = t.reader.ReadAt(t.ctx, buf, 0)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), testContent, buf[:readResponse.Size])
	assert.NotNil(t.T(), t.reader.fileCacheHandle)
	clear(buf)

	// Third ReadAt call: Should be served directly from the file cache.
	readResponse, err = t.reader.ReadAt(t.ctx, buf, 0)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), testContent, buf[:readResponse.Size])
	assert.NotNil(t.T(), t.reader.fileCacheHandle)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *fileCacheReaderTest) Test_ReadAt_NegativeOffsetShouldThrowError() {
	t.mockBucket.On("Name").Return("test-bucket")

	readResponse, err := t.reader.ReadAt(t.ctx, make([]byte, 10), -1)

	assert.Error(t.T(), err)
	assert.Zero(t.T(), readResponse.Size)
}

func (t *fileCacheReaderTest) Test_ReadAt_OffsetBeyondObjectSizeShouldThrowEOFError() {
	readResponse, err := t.reader.ReadAt(t.ctx, make([]byte, 10), int64(t.object.Size)+1)

	assert.Error(t.T(), err)
	assert.Zero(t.T(), readResponse.Size)
	assert.ErrorIs(t.T(), err, io.EOF)
}

func (t *fileCacheReaderTest) skipForNonZonalBucket() {
	t.T().Helper()

	if !t.bucketType.Zonal {
		t.T().Skipf("Skipping test for non-zonal bucket type")
	}
}

func (t *fileCacheReaderTest) fullyReadOriginalSizeOfUnfinalizedObject(origObjectSize uint64) {
	t.T().Helper()
	t.mockBucket.On("Name").Return("test-bucket")
	testContent := testutil.GenerateRandomBytes(int(t.unfinalized_object.Size))
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(t.unfinalized_object.Size, rd)
	t.mockBucket.On("BucketType").Return(t.bucketType)
	readResponse, err := t.reader_unfinalized_object.ReadAt(t.ctx, make([]byte, origObjectSize), 0)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), int(origObjectSize), readResponse.Size)
}

func (t *fileCacheReaderTest) waitForDownloadJobToFinish(obj *gcs.MinObject) {
	t.T().Helper()

	job := t.jobManager.GetJob(obj.Name, t.mockBucket.Name())
	if job != nil {
		for job.GetStatus().Name == downloader.Downloading {
			time.Sleep(10 * time.Millisecond) // Poll the job status.
		}
	}
}

func (t *fileCacheReaderTest) Test_ReadAt_UnfinalizedObjectReadFromOffsetBeyondCachedSizeAfterSizeIncreasedShouldThrowFallbackError() {
	t.skipForNonZonalBucket()
	origObjectSize := t.unfinalized_object.Size
	// First read, which may start a background download job.
	t.fullyReadOriginalSizeOfUnfinalizedObject(origObjectSize)
	// Wait for the background download job to complete to avoid a data race.
	// This is needed to avoid the race condition on the size of t.unfinalized_object later on.
	t.waitForDownloadJobToFinish(t.unfinalized_object)

	// Resize the object, and read beyond previously cached size and within new object size.
	cachedObjectSize := int64(origObjectSize)
	objectSizeIncrease := uint64(10)
	newObjectSize := origObjectSize + objectSizeIncrease
	t.unfinalized_object.Size = newObjectSize
	readResponse, err := t.reader_unfinalized_object.ReadAt(t.ctx, make([]byte, objectSizeIncrease), cachedObjectSize)

	assert.Error(t.T(), err)
	assert.Zero(t.T(), readResponse.Size)
	assert.ErrorIs(t.T(), err, FallbackToAnotherReader)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *fileCacheReaderTest) Test_ReadAt_UnfinalizedObjectReadFromOffsetBeyondObjectSizeAfterSizeIncreasedShouldThrowEOFError() {
	t.skipForNonZonalBucket()
	origObjectSize := t.unfinalized_object.Size
	// First read, which may start a background download job.
	t.fullyReadOriginalSizeOfUnfinalizedObject(origObjectSize)
	// Wait for the background download job to complete to avoid a data race.
	// This is needed to avoid the race condition on the size of t.unfinalized_object later on.
	t.waitForDownloadJobToFinish(t.unfinalized_object)

	// Resize the object, and read beyond new object size.
	objectSizeIncrease := uint64(10)
	newObjectSize := origObjectSize + objectSizeIncrease
	t.unfinalized_object.Size = newObjectSize
	readResponse, err := t.reader_unfinalized_object.ReadAt(t.ctx, make([]byte, 1), int64(newObjectSize))

	assert.Error(t.T(), err)
	assert.Zero(t.T(), readResponse.Size)
	assert.ErrorIs(t.T(), err, io.EOF)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *fileCacheReaderTest) Test_ReadAt_UnfinalizedObjectReadFromOffsetBelowCachedSizeAndReadBeyondCachedSizeWithIncreasedObjectSizeShouldThrowFallbackError() {
	t.skipForNonZonalBucket()
	origObjectSize := t.unfinalized_object.Size
	// First read, which may start a background download job.
	t.fullyReadOriginalSizeOfUnfinalizedObject(origObjectSize)
	// Wait for the background download job to complete to avoid a data race.
	// This is needed to avoid the race condition on the size of t.unfinalized_object later on.
	t.waitForDownloadJobToFinish(t.unfinalized_object)

	// Resize the object, and read from below previously cached size and within new object size.
	cachedObjectSize := int64(origObjectSize)
	objectSizeIncrease := uint64(10)
	newObjectSize := origObjectSize + objectSizeIncrease
	t.unfinalized_object.Size = newObjectSize
	readResponse, err := t.reader_unfinalized_object.ReadAt(t.ctx, make([]byte, cachedObjectSize+1), cachedObjectSize/2)

	assert.Error(t.T(), err)
	assert.Zero(t.T(), readResponse.Size)
	assert.ErrorIs(t.T(), err, FallbackToAnotherReader)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *fileCacheReaderTest) Test_ReadAt_UnfinalizedObjectReadFromOffsetBelowCachedSizeAndReadBeyondObjectSizeWithIncreasedObjectSizeShouldThrowFallbackError() {
	t.skipForNonZonalBucket()
	origObjectSize := t.unfinalized_object.Size
	// First read, which may start a background download job.
	t.fullyReadOriginalSizeOfUnfinalizedObject(origObjectSize)
	// Wait for the background download job to complete to avoid a data race.
	// This is needed to avoid the race condition on the size of t.unfinalized_object later on.
	t.waitForDownloadJobToFinish(t.unfinalized_object)

	// Resize the object, and read from below cached size and beyond this new object size.
	cachedObjectSize := int64(origObjectSize)
	objectSizeIncrease := uint64(10)
	newObjectSize := origObjectSize + objectSizeIncrease
	t.unfinalized_object.Size = newObjectSize
	readResponse, err := t.reader_unfinalized_object.ReadAt(t.ctx, make([]byte, newObjectSize), cachedObjectSize/2)

	assert.Error(t.T(), err)
	assert.Zero(t.T(), readResponse.Size)
	assert.ErrorIs(t.T(), err, FallbackToAnotherReader)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *fileCacheReaderTest) Test_ReadAt_UnfinalizedObjectReadFromOffsetBelowCachedSizeAndReadBeyondCachedSizeShouldNotThrowError() {
	t.skipForNonZonalBucket()
	origObjectSize := t.unfinalized_object.Size
	t.fullyReadOriginalSizeOfUnfinalizedObject(origObjectSize)

	cachedObjectSize := int64(origObjectSize)
	readResponse, err := t.reader_unfinalized_object.ReadAt(t.ctx, make([]byte, cachedObjectSize+1), cachedObjectSize/2)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), int(cachedObjectSize-cachedObjectSize/2), readResponse.Size)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *fileCacheReaderTest) Test_ReadAt_FinalizedObjectReadFromOffsetBelowCachedSizeAndReadBeyondCachedSizeShouldNotThrowError() {
	t.mockBucket.On("Name").Return("test-bucket")
	testContent := testutil.GenerateRandomBytes(int(t.object.Size))
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(t.object.Size, rd)
	t.mockBucket.On("BucketType").Return(t.bucketType)
	readResponse, err := t.reader.ReadAt(t.ctx, make([]byte, t.object.Size), 0)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), int(t.object.Size), readResponse.Size)

	readResponse, err = t.reader.ReadAt(t.ctx, make([]byte, t.object.Size+1), int64(t.object.Size)/2)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), int(t.object.Size-t.object.Size/2), readResponse.Size)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *fileCacheReaderTest) Test_Destroy_NonNilCacheHandle() {
	testContent := testutil.GenerateRandomBytes(int(t.object.Size))
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(t.object.Size, rd)
	t.mockBucket.On("Name").Return("test-bucket")
	t.mockBucket.On("BucketType").Return(t.bucketType)
	buf := make([]byte, t.object.Size)
	readResponse, err := t.reader.ReadAt(t.ctx, buf, 0)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), testContent, buf[:readResponse.Size])
	assert.NotNil(t.T(), t.reader.fileCacheHandle)

	t.reader.Destroy()

	assert.Nil(t.T(), t.reader.fileCacheHandle)
}

func (t *fileCacheReaderTest) Test_Destroy_NilCacheHandle() {
	t.reader.fileCacheHandler = nil

	t.reader.Destroy()

	assert.Nil(nil, t.reader.fileCacheHandle)
}
