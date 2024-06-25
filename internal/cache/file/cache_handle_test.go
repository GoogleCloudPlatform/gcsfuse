// Copyright 2023 Google Inc. All Rights Reserved.
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

package file

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/file/downloader"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/semaphore"
)

const CacheMaxSize = 100 * util.MiB
const ReadContentSize = 1 * util.MiB

const TestObjectSize = 16 * util.MiB
const TestObjectName = "foo.txt"
const DefaultSequentialReadSizeMb = 17

type cacheHandleTest struct {
	suite.Suite
	bucket      gcs.Bucket
	fakeStorage storage.FakeStorage
	object      *gcs.MinObject
	cache       *lru.Cache
	cacheHandle *CacheHandle
	cacheDir    string
	fileSpec    data.FileSpec
}

func TestBucketHandleTestSuite(testSuite *testing.T) {
	suite.Run(testSuite, new(cacheHandleTest))
}

func (cht *cacheHandleTest) addTestFileInfoEntryInCache() {
	// Add an entry into
	fileInfoKey := data.FileInfoKey{
		BucketName: storage.TestBucketName,
		ObjectName: TestObjectName,
	}
	fileInfo := data.FileInfo{
		Key:              fileInfoKey,
		ObjectGeneration: cht.object.Generation,
		FileSize:         cht.object.Size,
		Offset:           0,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	assert.Nil(cht.T(), err)

	_, err = cht.cache.Insert(fileInfoKeyName, fileInfo)
	assert.Nil(cht.T(), err)
}

func (cht *cacheHandleTest) verifyContentRead(readStartOffset int64, expectedContent []byte) {
	fileStat, fileErr := os.Stat(cht.fileSpec.Path)
	assert.Nil(cht.T(), fileErr)
	assert.Equal(cht.T(), cht.fileSpec.FilePerm, fileStat.Mode())
	dirStat, dirErr := os.Stat(filepath.Dir(cht.fileSpec.Path))
	assert.Nil(cht.T(), dirErr)
	assert.Equal(cht.T(), cht.fileSpec.DirPerm, dirStat.Mode().Perm())

	// Create a byte buffer of same len as expectedContent.
	buf := make([]byte, len(expectedContent))

	// Read from file and compare with expectedContent.
	_, err := cht.cacheHandle.fileHandle.Seek(readStartOffset, 0)
	assert.Nil(cht.T(), err)
	_, err = io.ReadFull(cht.cacheHandle.fileHandle, buf)
	assert.Nil(cht.T(), err)
	assert.True(cht.T(), reflect.DeepEqual(expectedContent, buf[:len(expectedContent)]))
}

func (cht *cacheHandleTest) SetupTest() {
	locker.EnableInvariantsCheck()
	cht.cacheDir = path.Join(os.Getenv("HOME"), "cache/dir")

	// Create bucket in fake storage.
	cht.fakeStorage = storage.NewFakeStorage()
	storageHandle := cht.fakeStorage.CreateStorageHandle()
	cht.bucket = storageHandle.BucketHandle(storage.TestBucketName, "")

	// Create test object in the bucket.
	ctx := context.Background()
	testObjectContent := make([]byte, TestObjectSize)
	n, err := rand.Read(testObjectContent)
	assert.Equal(cht.T(), TestObjectSize, n)
	assert.Nil(cht.T(), err)
	objects := map[string][]byte{TestObjectName: testObjectContent}
	err = storageutil.CreateObjects(ctx, cht.bucket, objects)
	assert.Nil(cht.T(), err)

	minObject, _, err := cht.bucket.StatObject(ctx, &gcs.StatObjectRequest{Name: TestObjectName,
		ForceFetchFromGcs: true})
	assert.Nil(cht.T(), err)
	assert.NotEqual(cht.T(), nil, minObject)
	cht.object = minObject

	// fileInfoCache with testFileInfoEntry
	cht.cache = lru.NewCache(CacheMaxSize)
	cht.addTestFileInfoEntryInCache()

	localDownloadedPath := path.Join(cht.cacheDir, cht.bucket.Name(), cht.object.Name)
	cht.fileSpec = data.FileSpec{Path: localDownloadedPath, FilePerm: util.DefaultFilePerm, DirPerm: util.DefaultDirPerm}

	readLocalFileHandle, err := util.CreateFile(cht.fileSpec, os.O_RDONLY)
	assert.Nil(cht.T(), err)

	fileDownloadJob := downloader.NewJob(cht.object, cht.bucket, cht.cache, DefaultSequentialReadSizeMb, cht.fileSpec, func() {}, &config.FileCacheConfig{EnableCRC: true, EnableParallelDownloads: false}, semaphore.NewWeighted(math.MaxInt64))

	cht.cacheHandle = NewCacheHandle(readLocalFileHandle, fileDownloadJob, cht.cache, false, 0)
}

func (cht *cacheHandleTest) TearDownTest() {
	cht.fakeStorage.ShutDown()

	err := cht.cacheHandle.Close()
	assert.Nil(cht.T(), err)

	operations.RemoveDir(cht.cacheDir)
}

func (cht *cacheHandleTest) Test_validateCacheHandle_WithNilFileHandle() {
	cht.cacheHandle.fileHandle = nil

	err := cht.cacheHandle.validateCacheHandle()

	assert.Equal(cht.T(), util.InvalidFileHandleErrMsg, err.Error())
}

func (cht *cacheHandleTest) Test_validateCacheHandle_WithNilFileDownloadJob() {
	cht.cacheHandle.fileDownloadJob = nil

	err := cht.cacheHandle.validateCacheHandle()

	assert.Nil(cht.T(), err)
}

func (cht *cacheHandleTest) Test_validateCacheHandle_WithNilFileInfoCache() {
	cht.cacheHandle.fileInfoCache = nil

	err := cht.cacheHandle.validateCacheHandle()

	assert.Equal(cht.T(), util.InvalidFileInfoCacheErrMsg, err.Error())
}

func (cht *cacheHandleTest) Test_validateCacheHandle_WithNonNilMemberAttributes() {
	err := cht.cacheHandle.validateCacheHandle()

	assert.Nil(cht.T(), err)
}

func (cht *cacheHandleTest) Test_Close_WithNonNilFileHandle() {
	err := cht.cacheHandle.Close()
	assert.Nil(cht.T(), err)

	assert.Nil(cht.T(), cht.cacheHandle.fileHandle)
}

func (cht *cacheHandleTest) Test_Close_WithNilFileHandle() {
	cht.cacheHandle.fileHandle = nil

	err := cht.cacheHandle.Close()
	assert.Nil(cht.T(), err)

	assert.Nil(cht.T(), cht.cacheHandle.fileHandle)
}

func (cht *cacheHandleTest) Test_IsSequential_WhenReadTypeIsNotSequential() {
	cht.cacheHandle.isSequential = false
	currentOffset := int64(3)

	isSeq := cht.cacheHandle.IsSequential(currentOffset)

	assert.False(cht.T(), isSeq)
}

func (cht *cacheHandleTest) Test_IsSequential_WhenPrevOffsetGreaterThanCurrent() {
	cht.cacheHandle.isSequential = true
	cht.cacheHandle.prevOffset = 5
	currentOffset := int64(3)

	isSeq := cht.cacheHandle.IsSequential(currentOffset)

	assert.False(cht.T(), isSeq)
}

func (cht *cacheHandleTest) Test_IsSequential_WhenOffsetDiffIsMoreThanMaxAllowed() {
	cht.cacheHandle.isSequential = true
	cht.cacheHandle.prevOffset = 5
	currentOffset := int64(8 + downloader.ReadChunkSize)

	isSeq := cht.cacheHandle.IsSequential(currentOffset)

	assert.False(cht.T(), isSeq)
}

func (cht *cacheHandleTest) Test_IsSequential_WhenOffsetDiffIsLessThanMaxAllowed() {
	cht.cacheHandle.isSequential = true
	cht.cacheHandle.prevOffset = 5
	currentOffset := int64(10)

	isSeq := cht.cacheHandle.IsSequential(currentOffset)

	assert.True(cht.T(), isSeq)
}

func (cht *cacheHandleTest) Test_IsSequential_WhenOffsetDiffIsEqualToMaxAllowed() {
	cht.cacheHandle.isSequential = true
	cht.cacheHandle.prevOffset = 5
	currentOffset := int64(5 + downloader.ReadChunkSize)

	isSeq := cht.cacheHandle.IsSequential(currentOffset)

	assert.True(cht.T(), isSeq)
}

func (cht *cacheHandleTest) Test_shouldReadFromCache_WithJobStateIsNotStarted() {
	requiredOffset := int64(downloader.ReadChunkSize + util.MiB)
	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	assert.Equal(cht.T(), downloader.NotStarted, jobStatus.Name)

	err := cht.cacheHandle.shouldReadFromCache(&jobStatus, requiredOffset)

	assert.NotNil(cht.T(), err)
	assert.True(cht.T(), strings.Contains(err.Error(), util.FallbackToGCSErrMsg))
}

func (cht *cacheHandleTest) Test_shouldReadFromCache_WithJobStateIsFailed() {
	requiredOffset := int64(downloader.ReadChunkSize + util.MiB)
	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	jobStatus.Name = downloader.Failed

	err := cht.cacheHandle.shouldReadFromCache(&jobStatus, requiredOffset)

	assert.NotNil(cht.T(), err)
	assert.True(cht.T(), strings.Contains(err.Error(), util.InvalidFileDownloadJobErrMsg))
}

func (cht *cacheHandleTest) Test_shouldReadFromCache_WithJobStateIsInvalid() {
	requiredOffset := int64(downloader.ReadChunkSize + util.MiB)
	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	jobStatus.Name = downloader.Invalid

	err := cht.cacheHandle.shouldReadFromCache(&jobStatus, requiredOffset)

	assert.NotNil(cht.T(), err)
	assert.True(cht.T(), strings.Contains(err.Error(), util.InvalidFileDownloadJobErrMsg))
}

func (cht *cacheHandleTest) Test_shouldReadFromCache_WithJobStateIsCompleted() {
	requiredOffset := int64(downloader.ReadChunkSize + util.MiB)
	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	jobStatus.Name = downloader.Completed
	jobStatus.Offset = int64(cht.object.Size)

	err := cht.cacheHandle.shouldReadFromCache(&jobStatus, requiredOffset)

	assert.Nil(cht.T(), err)
}

func (cht *cacheHandleTest) Test_shouldReadFromCache_WithJobDownloadedOffsetIsLessThanRequiredOffset() {
	requiredOffset := int64(downloader.ReadChunkSize + util.MiB)
	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	jobStatus.Name = downloader.Downloading
	jobStatus.Offset = requiredOffset - 1

	err := cht.cacheHandle.shouldReadFromCache(&jobStatus, requiredOffset)

	assert.NotNil(cht.T(), err)
	assert.True(cht.T(), strings.Contains(err.Error(), util.FallbackToGCSErrMsg))
}

func (cht *cacheHandleTest) Test_shouldReadFromCache_WithJobDownloadedOffsetSameAsRequiredOffset() {
	requiredOffset := int64(downloader.ReadChunkSize + util.MiB)
	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	jobStatus.Name = downloader.Downloading
	jobStatus.Offset = requiredOffset

	err := cht.cacheHandle.shouldReadFromCache(&jobStatus, requiredOffset)

	assert.Nil(cht.T(), err)
}

func (cht *cacheHandleTest) Test_shouldReadFromCache_WithJobDownloadedOffsetIsGreaterThanRequiredOffset() {
	requiredOffset := int64(util.MiB)
	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	jobStatus.Name = downloader.Downloading
	jobStatus.Offset = requiredOffset + 1

	err := cht.cacheHandle.shouldReadFromCache(&jobStatus, requiredOffset)

	assert.Nil(cht.T(), err)
}

func (cht *cacheHandleTest) Test_shouldReadFromCache_WithNonNilJobStatusErr() {
	requiredOffset := int64(downloader.ReadChunkSize + util.MiB)
	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	jobStatus.Name = downloader.Downloading
	jobStatus.Offset = requiredOffset + 1
	jobStatus.Err = errors.New("job error")

	err := cht.cacheHandle.shouldReadFromCache(&jobStatus, requiredOffset)

	assert.NotNil(cht.T(), err)
	assert.True(cht.T(), strings.Contains(err.Error(), util.InvalidFileDownloadJobErrMsg))
}

func (cht *cacheHandleTest) Test_validateEntryInFileInfoCache_FileInfoPresent() {
	fileInfoKey := data.FileInfoKey{
		BucketName: cht.bucket.Name(),
		ObjectName: cht.object.Name,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	assert.Nil(cht.T(), err)
	fileInfo := data.FileInfo{
		Key:              fileInfoKey,
		ObjectGeneration: cht.object.Generation,
		FileSize:         cht.object.Size,
		Offset:           cht.object.Size,
	}
	_, err = cht.cache.Insert(fileInfoKeyName, fileInfo)
	assert.Nil(cht.T(), err)

	err = cht.cacheHandle.validateEntryInFileInfoCache(cht.bucket, cht.object, cht.object.Size, false)

	assert.Nil(cht.T(), err)
}

func (cht *cacheHandleTest) Test_validateEntryInFileInfoCache_FileInfoNotPresent() {
	fileInfoKey := data.FileInfoKey{
		BucketName: cht.bucket.Name(),
		ObjectName: cht.object.Name,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	assert.Nil(cht.T(), err)

	_ = cht.cache.Erase(fileInfoKeyName)
	err = cht.cacheHandle.validateEntryInFileInfoCache(cht.bucket, cht.object, 0, false)

	expectedErr := fmt.Errorf("%v: no entry found in file info cache for key %v", util.InvalidFileInfoCacheErrMsg, fileInfoKeyName)
	assert.True(cht.T(), strings.Contains(err.Error(), expectedErr.Error()))
}

func (cht *cacheHandleTest) Test_validateEntryInFileInfoCache_FileInfoGenerationChanged() {
	fileInfoKey := data.FileInfoKey{
		BucketName: cht.bucket.Name(),
		ObjectName: cht.object.Name,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	assert.Nil(cht.T(), err)
	fileInfo := data.FileInfo{
		Key:              fileInfoKey,
		ObjectGeneration: cht.object.Generation + 1,
		FileSize:         cht.object.Size,
		Offset:           cht.object.Size,
	}
	_, err = cht.cache.Insert(fileInfoKeyName, fileInfo)
	assert.Nil(cht.T(), err)

	err = cht.cacheHandle.validateEntryInFileInfoCache(cht.bucket, cht.object, cht.object.Size-1, true)

	expectedErr := fmt.Errorf("%v: generation of cached object: %v is different from required generation: ", util.InvalidFileInfoCacheErrMsg, fileInfo.ObjectGeneration)
	assert.True(cht.T(), strings.Contains(err.Error(), expectedErr.Error()))
}

func (cht *cacheHandleTest) Test_validateEntryInFileInfoCache_FileInfoOffsetLessThanRequired() {
	fileInfoKey := data.FileInfoKey{
		BucketName: cht.bucket.Name(),
		ObjectName: cht.object.Name,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	assert.Nil(cht.T(), err)
	fileInfo := data.FileInfo{
		Key:              fileInfoKey,
		ObjectGeneration: cht.object.Generation,
		FileSize:         cht.object.Size,
		Offset:           10, // Insert offset less than required
	}
	_, err = cht.cache.Insert(fileInfoKeyName, fileInfo)
	assert.Nil(cht.T(), err)

	err = cht.cacheHandle.validateEntryInFileInfoCache(cht.bucket, cht.object, 11, true)

	assert.NotNil(cht.T(), err)
	expectedErr := fmt.Errorf("%v offset of cached object: %v is less than required offset %v", util.InvalidFileInfoCacheErrMsg, 10, 11)
	assert.Equal(cht.T(), expectedErr.Error(), err.Error())
}

func (cht *cacheHandleTest) Test_validateEntryInFileInfoCache_changeCacheOrderIsTrue() {
	// Adding one more entry to file info cache other than the one already added
	// by cht.addTestFileInfoEntryInCache, such that the file info cache becomes
	// full
	newObjectName := "new_test_object"
	fileInfoKey := data.FileInfoKey{
		BucketName: cht.bucket.Name(),
		ObjectName: newObjectName,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	assert.Nil(cht.T(), err)
	fileInfo := data.FileInfo{
		Key:              fileInfoKey,
		ObjectGeneration: 1,                              // Adding random generation.
		FileSize:         CacheMaxSize - cht.object.Size, // This makes cache size full.
		Offset:           1,                              // Insert offset less than required
	}
	evictedEntries, err := cht.cache.Insert(fileInfoKeyName, fileInfo)
	assert.Nil(cht.T(), err)
	assert.Equal(cht.T(), 0, len(evictedEntries))

	// Because changeCacheOrder is true, the entry corresponding to cht.object.Size
	// should come on top
	err = cht.cacheHandle.validateEntryInFileInfoCache(cht.bucket, cht.object, 0, true)

	assert.Nil(cht.T(), err)
	// Inserting new entry should evict the newObjectName
	fileInfoKey = data.FileInfoKey{
		BucketName: cht.bucket.Name(),
		ObjectName: "one more object",
	}
	fileInfoKeyName, err = fileInfoKey.Key()
	assert.Nil(cht.T(), err)
	fileInfo = data.FileInfo{
		Key:              fileInfoKey,
		ObjectGeneration: 1,
		FileSize:         1,
		Offset:           1,
	}
	evictedEntries, err = cht.cache.Insert(fileInfoKeyName, fileInfo)
	assert.Nil(cht.T(), err)
	assert.Equal(cht.T(), 1, len(evictedEntries))
	assert.Equal(cht.T(), newObjectName, evictedEntries[0].(data.FileInfo).Key.ObjectName)
}

func (cht *cacheHandleTest) Test_validateEntryInFileInfoCache_changeCacheOrderIsFalse() {
	// Adding one more entry to file info cache other than the one already added
	// by cht.addTestFileInfoEntryInCache, such that the file info cache becomes
	// full
	newObjectName := "new_test_object"
	fileInfoKey := data.FileInfoKey{
		BucketName: cht.bucket.Name(),
		ObjectName: newObjectName,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	assert.Nil(cht.T(), err)
	fileInfo := data.FileInfo{
		Key:              fileInfoKey,
		ObjectGeneration: 1,                              // Adding random generation.
		FileSize:         CacheMaxSize - cht.object.Size, // This makes cache size full.
		Offset:           1,                              // Insert offset less than required
	}
	evictedEntries, err := cht.cache.Insert(fileInfoKeyName, fileInfo)
	assert.Nil(cht.T(), err)
	assert.Equal(cht.T(), 0, len(evictedEntries))

	// Because changeCacheOrder is false, the new object entry should remain on top.
	err = cht.cacheHandle.validateEntryInFileInfoCache(cht.bucket, cht.object, 0, false)

	assert.Nil(cht.T(), err)
	// Inserting new entry should evict the entry corresponding to cht.object.
	fileInfoKey = data.FileInfoKey{
		BucketName: cht.bucket.Name(),
		ObjectName: "one more object",
	}
	fileInfoKeyName, err = fileInfoKey.Key()
	assert.Nil(cht.T(), err)
	fileInfo = data.FileInfo{
		Key:              fileInfoKey,
		ObjectGeneration: 1,
		FileSize:         1,
		Offset:           1,
	}
	evictedEntries, err = cht.cache.Insert(fileInfoKeyName, fileInfo)
	assert.Nil(cht.T(), err)
	assert.Equal(cht.T(), 1, len(evictedEntries))
	assert.Equal(cht.T(), cht.object.Name, evictedEntries[0].(data.FileInfo).Key.ObjectName)
}

func (cht *cacheHandleTest) Test_Read_RequestingMoreOffsetThanSize() {
	dst := make([]byte, ReadContentSize)
	offset := int64(cht.object.Size + 1)

	n, cacheHit, err := cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, offset, dst)

	assert.NotNil(cht.T(), err)
	assert.Equal(cht.T(), 0, n)
	assert.False(cht.T(), cacheHit)
	assert.True(cht.T(), strings.Contains(err.Error(), "wrong offset requested"))
}

func (cht *cacheHandleTest) Test_Read_WithNilFileHandle() {
	dst := make([]byte, ReadContentSize)
	offset := int64(5)
	cht.cacheHandle.fileHandle = nil

	n, cacheHit, err := cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, offset, dst)

	assert.NotNil(cht.T(), err)
	assert.Equal(cht.T(), 0, n)
	assert.False(cht.T(), cacheHit)
	assert.Equal(cht.T(), util.InvalidFileHandleErrMsg, err.Error())
}

func (cht *cacheHandleTest) Test_Read_WithNilFileDownloadJobAndCacheMiss() {
	dst := make([]byte, ReadContentSize)
	offset := int64(5)
	cht.cacheHandle.fileDownloadJob = nil

	// The file info entry added by cht.addTestFileInfoEntryInCache() has offset 0.
	// This means file info entry is there but download job and hence this should
	// throw.
	n, cacheHit, err := cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, offset, dst)

	assert.NotNil(cht.T(), err)
	assert.Equal(cht.T(), 0, n)
	assert.False(cht.T(), cacheHit)
	assert.True(cht.T(), strings.Contains(err.Error(), util.InvalidFileInfoCacheErrMsg))
}

func (cht *cacheHandleTest) Test_Read_WithNilFileDownloadJobAndCacheHit() {
	ctx := context.Background()
	// Download the complete object via job.
	jobStatus, err := cht.cacheHandle.fileDownloadJob.Download(ctx, int64(cht.object.Size), true)
	assert.Nil(cht.T(), err)
	assert.Equal(cht.T(), downloader.Downloading, jobStatus.Name)
	dst := make([]byte, cht.object.Size)
	offset := int64(0)
	cht.cacheHandle.isSequential = false
	cht.cacheHandle.fileDownloadJob = nil

	// Because the whole object is downloaded into the cache, file info cache
	// should have offset equal to object.Size, so even with nil download job,
	// read should be served from cache.
	n, cacheHit, err := cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, offset, dst)

	assert.Nil(cht.T(), err)
	assert.Equal(cht.T(), int64(cht.object.Size), int64(n))
	assert.True(cht.T(), cacheHit)
}

func (cht *cacheHandleTest) Test_Read_Random() {
	dst := make([]byte, ReadContentSize)
	offset := int64(cht.object.Size - ReadContentSize)
	cht.cacheHandle.isSequential = false
	cht.cacheHandle.cacheFileForRangeRead = true

	// Since, it's a random read hence will not wait to download till requested offset.
	n, cacheHit, err := cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, offset, dst)

	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	assert.Less(cht.T(), jobStatus.Offset, offset)
	assert.Equal(cht.T(), downloader.Downloading, jobStatus.Name)
	assert.Equal(cht.T(), 0, n)
	assert.False(cht.T(), cacheHit)
	assert.NotNil(cht.T(), err)
	assert.True(cht.T(), strings.Contains(err.Error(), util.FallbackToGCSErrMsg))
}

func (cht *cacheHandleTest) Test_Read_RandomWithNoRandomDownload() {
	dst := make([]byte, ReadContentSize)
	offset := int64(cht.object.Size - ReadContentSize)
	cht.cacheHandle.isSequential = false

	// Since, it's a random read hence will not wait to download till requested offset.
	n, cacheHit, err := cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, offset, dst)

	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	assert.Equal(cht.T(), downloader.NotStarted, jobStatus.Name)
	assert.Less(cht.T(), jobStatus.Offset, offset)
	assert.Equal(cht.T(), n, 0)
	assert.False(cht.T(), cacheHit)
	assert.NotNil(cht.T(), err)
	assert.True(cht.T(), strings.Contains(err.Error(), util.FallbackToGCSErrMsg))
}

func (cht *cacheHandleTest) Test_Read_RandomWithNoRandomDownloadButCacheHit() {
	ctx := context.Background()
	// Download the job till util.MiB
	jobStatus, err := cht.cacheHandle.fileDownloadJob.Download(ctx, int64(2*util.MiB), true)
	assert.Nil(cht.T(), err)
	assert.Equal(cht.T(), downloader.Downloading, jobStatus.Name)
	dst := make([]byte, ReadContentSize)
	offset := int64(1)
	cht.cacheHandle.isSequential = false

	// Since, it's a random read hence will not wait to download till requested offset.
	_, cacheHit, err := cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, offset, dst)

	jobStatus = cht.cacheHandle.fileDownloadJob.GetStatus()
	assert.True(cht.T(), jobStatus.Name == downloader.Downloading || jobStatus.Name == downloader.Completed)
	assert.GreaterOrEqual(cht.T(), jobStatus.Offset, offset)
	assert.True(cht.T(), cacheHit)
	assert.Nil(cht.T(), err)
	cht.verifyContentRead(offset, dst)
}

func (cht *cacheHandleTest) Test_Read_Sequential() {
	dst := make([]byte, ReadContentSize)
	offset := int64(cht.object.Size - ReadContentSize)
	cht.cacheHandle.isSequential = true
	cht.cacheHandle.prevOffset = offset - util.MiB
	cht.cacheHandle.cacheFileForRangeRead = false

	// Since, it's a sequential read, hence will wait to download till requested offset.
	_, cacheHit, err := cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, offset, dst)

	assert.Nil(cht.T(), err)
	assert.False(cht.T(), cacheHit)
	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	assert.GreaterOrEqual(cht.T(), jobStatus.Offset, offset)
	cht.verifyContentRead(offset, dst)
}

func (cht *cacheHandleTest) Test_Read_ChangeCacheOrder() {
	// Adding one more entry to file info cache other than the one already added
	// by cht.addTestFileInfoEntryInCache, such that the file info cache becomes
	// full
	newObjectName := "new_test_object"
	fileInfoKey := data.FileInfoKey{
		BucketName: cht.bucket.Name(),
		ObjectName: newObjectName,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	assert.Nil(cht.T(), err)
	fileInfo := data.FileInfo{
		Key:              fileInfoKey,
		ObjectGeneration: 1,                              // Adding random generation.
		FileSize:         CacheMaxSize - cht.object.Size, // This makes cache size full.
		Offset:           1,                              // Insert offset less than required
	}
	evictedEntries, err := cht.cache.Insert(fileInfoKeyName, fileInfo)
	assert.Nil(cht.T(), err)
	assert.Equal(cht.T(), 0, len(evictedEntries))
	dst := make([]byte, ReadContentSize)
	offset := int64(cht.object.Size - ReadContentSize)
	cht.cacheHandle.isSequential = true
	cht.cacheHandle.prevOffset = offset - util.MiB
	cht.cacheHandle.cacheFileForRangeRead = false

	// Read should change the order in cache and bring cht.Object to most recently
	// used.
	_, cacheHit, err := cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, offset, dst)

	assert.Nil(cht.T(), err)
	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	assert.False(cht.T(), cacheHit)
	assert.GreaterOrEqual(cht.T(), jobStatus.Offset, offset)
	cht.verifyContentRead(offset, dst)
	// Inserting new entry should evict the newObjectName
	fileInfoKey = data.FileInfoKey{
		BucketName: cht.bucket.Name(),
		ObjectName: "one more object",
	}
	fileInfoKeyName, err = fileInfoKey.Key()
	assert.Nil(cht.T(), err)
	fileInfo = data.FileInfo{
		Key:              fileInfoKey,
		ObjectGeneration: 1,
		FileSize:         1,
		Offset:           1,
	}
	evictedEntries, err = cht.cache.Insert(fileInfoKeyName, fileInfo)
	assert.Nil(cht.T(), err)
	assert.Equal(cht.T(), 1, len(evictedEntries))
	assert.Equal(cht.T(), newObjectName, evictedEntries[0].(data.FileInfo).Key.ObjectName)
}

func (cht *cacheHandleTest) Test_Read_SequentialToRandom() {
	dst := make([]byte, ReadContentSize)
	firstReqOffset := int64(0)
	cht.cacheHandle.isSequential = true
	cht.cacheHandle.cacheFileForRangeRead = true
	// Since, it's a sequential read, hence will wait to download till requested offset.
	_, cacheHit, err := cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, firstReqOffset, dst)
	assert.Nil(cht.T(), nil, err)
	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	assert.GreaterOrEqual(cht.T(), jobStatus.Offset, firstReqOffset)
	assert.False(cht.T(), cacheHit)
	assert.True(cht.T(), cht.cacheHandle.isSequential)

	secondReqOffset := int64(cht.object.Size - ReadContentSize) // type will change to random.
	_, cacheHit, err = cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, secondReqOffset, dst)

	assert.NotNil(cht.T(), err)
	assert.True(cht.T(), strings.Contains(err.Error(), util.FallbackToGCSErrMsg))
	assert.False(cht.T(), cacheHit)
	assert.False(cht.T(), cht.cacheHandle.isSequential)
	jobStatus = cht.cacheHandle.fileDownloadJob.GetStatus()
	assert.LessOrEqual(cht.T(), jobStatus.Offset, secondReqOffset)
}

func (cht *cacheHandleTest) Test_Read_WhenDstBufferIsMoreContentToBeRead() {
	// Add extra buffer.
	extraBuffer := 2
	dst := make([]byte, ReadContentSize+extraBuffer)
	offset := int64(cht.object.Size - ReadContentSize)
	cht.cacheHandle.isSequential = true
	cht.cacheHandle.prevOffset = offset - util.MiB
	cht.cacheHandle.cacheFileForRangeRead = true

	// Since, it's a sequential read, hence will wait to download till requested offset.
	_, cacheHit, err := cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, offset, dst)

	assert.Nil(cht.T(), err)
	assert.False(cht.T(), cacheHit)
	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	assert.GreaterOrEqual(cht.T(), jobStatus.Offset, offset)
	cht.verifyContentRead(offset, dst[:len(dst)-extraBuffer])
}

func (cht *cacheHandleTest) Test_Read_FileInfoRemoved() {
	dst := make([]byte, ReadContentSize)
	cht.cacheHandle.isSequential = true
	cht.cacheHandle.cacheFileForRangeRead = true
	// First let the cache populated (we are doing this so that we can externally
	// modify file info cache for this unit test without hampering download job).
	_, cacheHit, err := cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, 0, dst)
	assert.Nil(cht.T(), err)
	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	assert.GreaterOrEqual(cht.T(), jobStatus.Offset, int64(ReadContentSize))
	fileInfoKey := data.FileInfoKey{
		BucketName: cht.bucket.Name(),
		ObjectName: cht.object.Name,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	assert.Nil(cht.T(), err)
	assert.False(cht.T(), cacheHit)

	// Delete the file info entry and again perform read
	_ = cht.cache.Erase(fileInfoKeyName)
	_, cacheHit, err = cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, 0, dst)

	assert.NotNil(cht.T(), err)
	expectedErr := fmt.Errorf("%v: no entry found in file info cache for key %v", util.InvalidFileInfoCacheErrMsg, fileInfoKeyName)
	assert.True(cht.T(), strings.Contains(err.Error(), expectedErr.Error()))
	assert.False(cht.T(), cacheHit)
}

func (cht *cacheHandleTest) Test_Read_FileInfoGenerationChanged() {
	dst := make([]byte, ReadContentSize)
	cht.cacheHandle.isSequential = true
	cht.cacheHandle.cacheFileForRangeRead = true
	// First let the cache populated (we are doing this so that we can externally
	// modify file info cache for this unit test without hampering download job).
	_, cacheHit, err := cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, 0, dst)
	assert.Nil(cht.T(), err)
	assert.False(cht.T(), cacheHit)
	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	assert.GreaterOrEqual(cht.T(), jobStatus.Offset, int64(ReadContentSize))
	fileInfoKey := data.FileInfoKey{
		BucketName: cht.bucket.Name(),
		ObjectName: cht.object.Name,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	assert.Nil(cht.T(), err)
	fileInfo := cht.cache.LookUp(fileInfoKeyName)
	fileInfoData := fileInfo.(data.FileInfo)

	// Update the file info entry generation and again perform read
	fileInfoData.ObjectGeneration = 1
	err = cht.cache.UpdateWithoutChangingOrder(fileInfoKeyName, fileInfoData)
	assert.Nil(cht.T(), err)
	_, cacheHit, err = cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, 0, dst)

	assert.NotNil(cht.T(), err)
	expectedErr := fmt.Errorf("%v: generation of cached object: %v is different from required generation: ", util.InvalidFileInfoCacheErrMsg, fileInfoData.ObjectGeneration)
	assert.True(cht.T(), strings.Contains(err.Error(), expectedErr.Error()))
	assert.False(cht.T(), cacheHit)
}

func (cht *cacheHandleTest) Test_MultipleReads_CacheHitShouldBeFalseThenTrue() {
	dst := make([]byte, ReadContentSize)
	offset := int64(cht.object.Size - ReadContentSize)
	cht.cacheHandle.isSequential = true
	cht.cacheHandle.prevOffset = offset - util.MiB
	cht.cacheHandle.cacheFileForRangeRead = true
	// First read should be cache miss.
	n, cacheHit, err := cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, offset, dst)
	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	assert.GreaterOrEqual(cht.T(), jobStatus.Offset, offset)
	assert.Equal(cht.T(), n, ReadContentSize)
	cht.verifyContentRead(offset, dst[:n])
	assert.False(cht.T(), cacheHit)
	assert.Nil(cht.T(), err)

	// Second read should be cache hit.
	n, cacheHit, err = cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, offset, dst)

	assert.Equal(cht.T(), n, ReadContentSize)
	assert.True(cht.T(), cacheHit)
	assert.Nil(cht.T(), err)
}

func (cht *cacheHandleTest) Test_Read_Sequential_Parallel_Download_True() {
	dst := make([]byte, ReadContentSize)
	offset := int64(cht.object.Size - ReadContentSize)
	cht.cacheHandle.isSequential = true
	cht.cacheHandle.prevOffset = offset - util.MiB
	cht.cacheHandle.cacheFileForRangeRead = false
	fileDownloadJob := downloader.NewJob(
		cht.object,
		cht.bucket,
		cht.cache,
		DefaultSequentialReadSizeMb,
		cht.fileSpec,
		func() {},
		&config.FileCacheConfig{EnableCRC: true, EnableParallelDownloads: true, ParallelDownloadsPerFile: 2, DownloadChunkSizeMB: 2},
		semaphore.NewWeighted(math.MaxInt64),
	)
	cht.cacheHandle.fileDownloadJob = fileDownloadJob

	n, cacheHit, err := cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, offset, dst)

	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	assert.Less(cht.T(), jobStatus.Offset, offset)
	assert.Equal(cht.T(), downloader.Downloading, jobStatus.Name)
	assert.Equal(cht.T(), 0, n)
	assert.False(cht.T(), cacheHit)
	assert.ErrorContains(cht.T(), err, util.FallbackToGCSErrMsg)
}

func (cht *cacheHandleTest) Test_Read_Random_Parallel_Download_True() {
	dst := make([]byte, ReadContentSize)
	offset := int64(cht.object.Size - ReadContentSize)
	cht.cacheHandle.isSequential = false
	cht.cacheHandle.cacheFileForRangeRead = true
	fileDownloadJob := downloader.NewJob(
		cht.object,
		cht.bucket,
		cht.cache,
		DefaultSequentialReadSizeMb,
		cht.fileSpec,
		func() {},
		&config.FileCacheConfig{EnableCRC: true, EnableParallelDownloads: true, ParallelDownloadsPerFile: 5, DownloadChunkSizeMB: 2},
		semaphore.NewWeighted(math.MaxInt64),
	)
	cht.cacheHandle.fileDownloadJob = fileDownloadJob

	n, cacheHit, err := cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, offset, dst)

	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	assert.Less(cht.T(), jobStatus.Offset, offset)
	assert.Equal(cht.T(), downloader.Downloading, jobStatus.Name)
	assert.Equal(cht.T(), 0, n)
	assert.False(cht.T(), cacheHit)
	assert.ErrorContains(cht.T(), err, util.FallbackToGCSErrMsg)
}
