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
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	testutil "github.com/googlecloudplatform/gcsfuse/v2/internal/util"
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
	ctx          context.Context
	object       *gcs.MinObject
	mockBucket   *storage.TestifyMockBucket
	cacheDir     string
	jobManager   *downloader.JobManager
	cacheHandler *file.CacheHandler
	reader       FileCacheReader
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
	lruCache := lru.NewCache(CacheMaxSize)
	t.jobManager = downloader.NewJobManager(lruCache, util.DefaultFilePerm, util.DefaultDirPerm, t.cacheDir, sequentialReadSizeInMb, &cfg.FileCacheConfig{EnableCrc: false}, common.NewNoopMetrics())
	t.cacheHandler = file.NewCacheHandler(lruCache, t.jobManager, t.cacheDir, util.DefaultFilePerm, util.DefaultDirPerm)
	t.reader = NewFileCacheReader(t.object, t.mockBucket, t.cacheHandler, true, common.NewNoopMetrics())
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

func (t *FileCacheReaderTest) mockNewReaderWithHandleCallForTestBucket(start uint64, limit uint64, rd gcs.StorageReader) {
	t.mockBucket.
		On("NewReaderWithReadHandle", mock.Anything, mock.MatchedBy(func(rg *gcs.ReadObjectRequest) bool {
			return rg != nil && (*rg.Range).Start == start && (*rg.Range).Limit == limit
		})).Return(rd, nil).Maybe()
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

func (t *FileCacheReaderTest) TestReadWithNilFileCacheHandler() {
	reader := NewFileCacheReader(t.object, t.mockBucket, nil, true, nil)

	readerResponse, err := reader.ReadAt(t.ctx, make([]byte, 10), 0)

	assert.Error(t.T(), err)
	assert.Zero(t.T(), readerResponse.Size)
}

func (t *FileCacheReaderTest) TestReadWith_CacheForRangeReadFalsError() {
	t.mockBucket.On("Name").Return("test-bucket")
	reader := NewFileCacheReader(t.object, t.mockBucket, t.cacheHandler, false, common.NewNoopMetrics())

	readerResponse, err := reader.ReadAt(t.ctx, make([]byte, 10), 10)

	assert.True(t.T(), errors.Is(err, FallbackToAnotherReader))
	assert.Zero(t.T(), readerResponse.Size)
}

func (t *FileCacheReaderTest) TestReadWith_GeneticError() {
	t.mockBucket.On("Name").Return("test-bucket")
	reader := NewFileCacheReader(t.object, t.mockBucket, t.cacheHandler, false, common.NewNoopMetrics())

	readerResponse, err := reader.ReadAt(t.ctx, make([]byte, 10), 10)

	assert.True(t.T(), errors.Is(err, FallbackToAnotherReader))
	assert.Zero(t.T(), readerResponse.Size)
}

// Writing unit tests on tryReadingFromFileCache to check if cache hit is getting populated correctly.
func (t *FileCacheReaderTest) Test_tryReadingFromFileCache_CacheHit() {
	t.reader.fileCacheHandler = t.cacheHandler
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd)
	t.mockBucket.On("Name").Return("test-bucket")
	buf := make([]byte, objectSize)
	// First read will be a cache miss.
	_, cacheHit, err := t.reader.tryReadingFromFileCache(t.ctx, buf, 0)
	assert.False(t.T(), cacheHit)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), testContent, buf)

	// Second read will be a cache hit.
	_, cacheHit, err = t.reader.tryReadingFromFileCache(t.ctx, buf, 0)
	assert.True(t.T(), cacheHit)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), testContent, buf)
}

func (t *FileCacheReaderTest) Test_ReadAt_SequentialFullObject() {
	t.reader.fileCacheHandler = t.cacheHandler
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd)
	t.mockBucket.On("Name").Return("test-bucket")
	buf := make([]byte, objectSize)
	n, cacheHit, err := t.reader.tryReadingFromFileCache(t.ctx, buf, 0)

	assert.NoError(t.T(), err)
	assert.False(t.T(), cacheHit)
	assert.Equal(t.T(), n, len(buf))

	n, cacheHit, err = t.reader.tryReadingFromFileCache(t.ctx, buf, 0)

	assert.NoError(t.T(), err)
	assert.True(t.T(), cacheHit)
	assert.Equal(t.T(), n, len(buf))
}

func (t *FileCacheReaderTest) Test_ReadAt_SequentialRangeRead() {
	t.reader.fileCacheHandler = t.cacheHandler
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd)
	t.mockBucket.On("Name").Return("test-bucket")
	start := 0
	end := 10
	assert.Less(t.T(), end, int(objectSize))
	buf := make([]byte, end-start)

	readerResponse, err := t.reader.ReadAt(t.ctx, buf, int64(start))

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), readerResponse.DataBuf, testContent[start:end])
}

func (t *FileCacheReaderTest) Test_ReadAt_SequentialSubsequentReadOffsetLessThanReadChunkSize() {
	t.reader.fileCacheHandler = t.cacheHandler
	t.object.Size = 20 * util.MiB
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd)
	t.mockBucket.On("Name").Return("test-bucket")
	start1 := 0
	end1 := util.MiB // not included
	assert.Less(t.T(), end1, int(objectSize))
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
}

func (t *FileCacheReaderTest) Test_ReadAt_RandomReadNotStartWithZeroOffsetWhenCacheForRangeReadIsFalse() {
	t.reader.fileCacheHandler = t.cacheHandler
	objectSize := t.object.Size
	t.reader.cacheFileForRangeRead = false
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	start := 5
	end := 10 // not included
	rc := &fake.FakeReader{ReadCloser: getReadCloser(testContent[start:])}
	t.mockNewReaderWithHandleCallForTestBucket(uint64(start), objectSize, rc)
	t.mockBucket.On("Name").Return("test-bucket")
	t.mockBucket.On("BucketType").Return(gcs.BucketType{})
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
}

func (t *FileCacheReaderTest) Test_ReadAt_RandomReadNotStartWithZeroOffsetWhenCacheForRangeReadIsTrue() {
	t.reader.fileCacheHandler = t.cacheHandler
	t.reader.cacheFileForRangeRead = true
	testContent := testutil.GenerateRandomBytes(int(t.object.Size))
	start := 5
	end := 10 // not included
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent[start:])}
	// Mock for random-reader's NewReader call
	t.mockNewReaderWithHandleCallForTestBucket(uint64(start), t.object.Size, rd)
	rd1 := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	// Mock for download job's NewReader call
	t.mockNewReaderWithHandleCallForTestBucket(0, t.object.Size, rd1)
	t.mockBucket.On("Name").Return("test-bucket")
	t.mockBucket.On("BucketType").Return(gcs.BucketType{})
	buf := make([]byte, end-start)

	readerResponse, err := t.reader.ReadAt(t.ctx, buf, int64(start))

	assert.True(t.T(), errors.Is(err, FallbackToAnotherReader))
	assert.Zero(t.T(), readerResponse.Size)
	job := t.jobManager.GetJob(t.object.Name, t.mockBucket.Name())
	assert.True(t.T(), job == nil || job.GetStatus().Name == downloader.Downloading)
}

func (t *FileCacheReaderTest) Test_ReadAt_SequentialToRandomSubsequentReadOffsetMoreThanReadChunkSize() {
	t.reader.fileCacheHandler = t.cacheHandler
	t.object.Size = 20 * util.MiB
	testContent := testutil.GenerateRandomBytes(int(t.object.Size))
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	// Mock for download job's NewReader call
	t.mockNewReaderWithHandleCallForTestBucket(0, t.object.Size, rd)
	t.mockBucket.On("Name").Return("test-bucket")
	t.mockBucket.On("BucketType").Return(gcs.BucketType{})
	start1 := 0
	end1 := util.MiB // not included
	assert.Less(t.T(), end1, int(t.object.Size))
	// First call from offset 0 - sequential read
	buf := make([]byte, end1-start1)
	readerResponse, err := t.reader.ReadAt(t.ctx, buf, int64(start1))
	// Served from file cache
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), readerResponse.DataBuf, testContent[start1:end1])
	start2 := 16*util.MiB + 4
	end2 := start2 + util.MiB
	rd2 := &fake.FakeReader{ReadCloser: getReadCloser(testContent[start2:])}
	// Mock for random-reader's NewReader call
	t.mockNewReaderWithHandleCallForTestBucket(uint64(start2), t.object.Size, rd2)
	buf2 := make([]byte, end2-start2)

	readerResponse, err = t.reader.ReadAt(t.ctx, buf2, int64(start2))

	// Assuming start2 offset download in progress
	// require to fall back on GCS reader
	assert.True(t.T(), errors.Is(err, FallbackToAnotherReader))
	assert.Zero(t.T(), readerResponse.Size)
}

func (t *FileCacheReaderTest) Test_ReadAt_SequentialToRandomSubsequentReadOffsetLessThanPrevious() {
	t.reader.fileCacheHandler = t.cacheHandler
	t.object.Size = 20 * util.MiB
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rd := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rd)
	t.mockBucket.On("Name").Return("test-bucket")
	t.mockBucket.On("BucketType").Return(gcs.BucketType{})
	start1 := 0
	end1 := util.MiB // not included
	assert.Less(t.T(), end1, int(t.object.Size))
	// First call from offset 0 - sequential read
	buf := make([]byte, end1-start1)
	readerResponse, err := t.reader.ReadAt(t.ctx, buf, int64(start1))
	// Served from file cache
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), readerResponse.DataBuf, testContent[start1:end1])
	start2 := 16*util.MiB + 4
	end2 := start2 + util.MiB
	rc2 := &fake.FakeReader{ReadCloser: getReadCloser(testContent[start2:])}
	t.mockNewReaderWithHandleCallForTestBucket(uint64(start2), objectSize, rc2)
	buf2 := make([]byte, end2-start2)
	// Assuming start2 offset download in progress
	readerResponse, err = t.reader.ReadAt(t.ctx, buf2, int64(start2))
	// Need to get served from GCS reader
	assert.True(t.T(), errors.Is(err, FallbackToAnotherReader))
	assert.Zero(t.T(), readerResponse.Size)
	start3 := util.MiB
	end3 := start3 + util.MiB
	buf3 := make([]byte, end3-start3)

	readerResponse, err = t.reader.ReadAt(t.ctx, buf3, int64(start3))

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), readerResponse.DataBuf, testContent[start3:end3])
}

func (t *FileCacheReaderTest) Test_ReadAt_CacheMissDueToInvalidJob() {
	t.reader.fileCacheHandler = t.cacheHandler
	objectSize := t.object.Size
	testContent := testutil.GenerateRandomBytes(int(objectSize))
	rc1 := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.mockNewReaderWithHandleCallForTestBucket(0, objectSize, rc1)
	t.mockBucket.On("Name").Return("test-bucket")
	t.mockBucket.On("BucketType").Return(gcs.BucketType{})
	buf := make([]byte, objectSize)
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
}
