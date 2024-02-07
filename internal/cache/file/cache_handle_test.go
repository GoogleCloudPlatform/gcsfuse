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
	"os"
	"path"
	"reflect"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/file/downloader"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	. "github.com/jacobsa/ogletest"
)

const CacheMaxSize = 100 * util.MiB
const ReadContentSize = 1 * util.MiB

const TestObjectSize = 16 * util.MiB
const TestObjectName = "foo.txt"
const DefaultSequentialReadSizeMb = 17

func TestCacheHandle(t *testing.T) { RunTests(t) }

type cacheHandleTest struct {
	bucket        gcs.Bucket
	fakeStorage   storage.FakeStorage
	object        *gcs.MinObject
	cache         *lru.Cache
	cacheHandle   *CacheHandle
	cacheLocation string
	fileSpec      data.FileSpec
}

func init() {
	RegisterTestSuite(&cacheHandleTest{})
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
	AssertEq(nil, err)

	_, err = cht.cache.Insert(fileInfoKeyName, fileInfo)
	AssertEq(nil, err)
}

func (cht *cacheHandleTest) verifyContentRead(readStartOffset int64, expectedContent []byte) {
	fileStat, err := os.Stat(cht.fileSpec.Path)
	AssertEq(nil, err)
	AssertEq(cht.fileSpec.FilePerm, fileStat.Mode())
	AssertEq(cht.fileSpec.DirPerm, 0700)

	// Create a byte buffer of same len as expectedContent.
	buf := make([]byte, len(expectedContent))

	// Read from file and compare with expectedContent.
	_, err = cht.cacheHandle.fileHandle.Seek(readStartOffset, 0)
	AssertEq(nil, err)
	_, err = io.ReadFull(cht.cacheHandle.fileHandle, buf)
	AssertEq(nil, err)
	AssertTrue(reflect.DeepEqual(expectedContent, buf[:len(expectedContent)]))
}

func (cht *cacheHandleTest) SetUp(*TestInfo) {
	locker.EnableInvariantsCheck()
	cht.cacheLocation = path.Join(os.Getenv("HOME"), "cache/location")

	// Create bucket in fake storage.
	cht.fakeStorage = storage.NewFakeStorage()
	storageHandle := cht.fakeStorage.CreateStorageHandle()
	cht.bucket = storageHandle.BucketHandle(storage.TestBucketName, "")

	// Create test object in the bucket.
	ctx := context.Background()
	testObjectContent := make([]byte, TestObjectSize)
	n, err := rand.Read(testObjectContent)
	AssertEq(TestObjectSize, n)
	AssertEq(nil, err)
	objects := map[string][]byte{TestObjectName: testObjectContent}
	err = storageutil.CreateObjects(ctx, cht.bucket, objects)
	AssertEq(nil, err)

	gcsObj, err := cht.bucket.StatObject(ctx, &gcs.StatObjectRequest{Name: TestObjectName,
		ForceFetchFromGcs: true})
	AssertEq(nil, err)
	minObject := storageutil.ConvertObjToMinObject(gcsObj)
	cht.object = &minObject

	// fileInfoCache with testFileInfoEntry
	cht.cache = lru.NewCache(CacheMaxSize)
	cht.addTestFileInfoEntryInCache()

	localDownloadedPath := path.Join(cht.cacheLocation, cht.bucket.Name(), cht.object.Name)
	cht.fileSpec = data.FileSpec{Path: localDownloadedPath, FilePerm: util.DefaultFilePerm, DirPerm: util.DefaultDirPerm}

	readLocalFileHandle, err := util.CreateFile(cht.fileSpec, os.O_RDONLY)
	AssertEq(nil, err)

	fileDownloadJob := downloader.NewJob(cht.object, cht.bucket, cht.cache, DefaultSequentialReadSizeMb, cht.fileSpec)

	cht.cacheHandle = NewCacheHandle(readLocalFileHandle, fileDownloadJob, cht.cache, false, 0)
}

func (cht *cacheHandleTest) TearDown() {
	cht.fakeStorage.ShutDown()

	err := cht.cacheHandle.Close()
	AssertEq(nil, err)

	operations.RemoveDir(cht.cacheLocation)
}

func (cht *cacheHandleTest) Test_validateCacheHandle_WithNilFileHandle() {
	cht.cacheHandle.fileHandle = nil

	err := cht.cacheHandle.validateCacheHandle()

	ExpectEq(util.InvalidFileHandleErrMsg, err.Error())
}

func (cht *cacheHandleTest) Test_validateCacheHandle_WithNilFileDownloadJob() {
	cht.cacheHandle.fileDownloadJob = nil

	err := cht.cacheHandle.validateCacheHandle()

	ExpectEq(util.InvalidFileDownloadJobErrMsg, err.Error())
}

func (cht *cacheHandleTest) Test_validateCacheHandle_WithNilFileInfoCache() {
	cht.cacheHandle.fileInfoCache = nil

	err := cht.cacheHandle.validateCacheHandle()

	ExpectEq(util.InvalidFileInfoCacheErrMsg, err.Error())
}

func (cht *cacheHandleTest) Test_validateCacheHandle_WithNonNilMemberAttributes() {
	err := cht.cacheHandle.validateCacheHandle()

	ExpectEq(nil, err)
}

func (cht *cacheHandleTest) Test_Close_WithNonNilFileHandle() {
	err := cht.cacheHandle.Close()
	AssertEq(nil, err)

	ExpectEq(nil, cht.cacheHandle.fileHandle)
}

func (cht *cacheHandleTest) Test_Close_WithNilFileHandle() {
	cht.cacheHandle.fileHandle = nil

	err := cht.cacheHandle.Close()
	AssertEq(nil, err)

	ExpectEq(nil, cht.cacheHandle.fileHandle)
}

func (cht *cacheHandleTest) Test_IsSequential_WhenReadTypeIsNotSequential() {
	cht.cacheHandle.isSequential = false
	currentOffset := int64(3)

	isSeq := cht.cacheHandle.IsSequential(currentOffset)

	ExpectEq(false, isSeq)
}

func (cht *cacheHandleTest) Test_IsSequential_WhenPrevOffsetGreaterThanCurrent() {
	cht.cacheHandle.isSequential = true
	cht.cacheHandle.prevOffset = 5
	currentOffset := int64(3)

	isSeq := cht.cacheHandle.IsSequential(currentOffset)

	ExpectEq(false, isSeq)
}

func (cht *cacheHandleTest) Test_IsSequential_WhenOffsetDiffIsMoreThanMaxAllowed() {
	cht.cacheHandle.isSequential = true
	cht.cacheHandle.prevOffset = 5
	currentOffset := int64(8 + downloader.ReadChunkSize)

	isSeq := cht.cacheHandle.IsSequential(currentOffset)

	ExpectEq(false, isSeq)
}

func (cht *cacheHandleTest) Test_IsSequential_WhenOffsetDiffIsLessThanMaxAllowed() {
	cht.cacheHandle.isSequential = true
	cht.cacheHandle.prevOffset = 5
	currentOffset := int64(10)

	isSeq := cht.cacheHandle.IsSequential(currentOffset)

	ExpectEq(true, isSeq)
}

func (cht *cacheHandleTest) Test_IsSequential_WhenOffsetDiffIsEqualToMaxAllowed() {
	cht.cacheHandle.isSequential = true
	cht.cacheHandle.prevOffset = 5
	currentOffset := int64(5 + downloader.ReadChunkSize)

	isSeq := cht.cacheHandle.IsSequential(currentOffset)

	ExpectEq(true, isSeq)
}

func (cht *cacheHandleTest) Test_shouldReadFromCache_WithJobStateIsNotStarted() {
	requiredOffset := int64(downloader.ReadChunkSize + util.MiB)
	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	AssertEq(downloader.NOT_STARTED, jobStatus.Name)

	err := cht.cacheHandle.shouldReadFromCache(&jobStatus, requiredOffset)

	ExpectNe(nil, err)
	ExpectTrue(strings.Contains(err.Error(), util.FallbackToGCSErrMsg))
}

func (cht *cacheHandleTest) Test_shouldReadFromCache_WithJobStateIsFailed() {
	requiredOffset := int64(downloader.ReadChunkSize + util.MiB)
	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	jobStatus.Name = downloader.FAILED

	err := cht.cacheHandle.shouldReadFromCache(&jobStatus, requiredOffset)

	ExpectNe(nil, err)
	ExpectTrue(strings.Contains(err.Error(), util.InvalidFileDownloadJobErrMsg))
}

func (cht *cacheHandleTest) Test_shouldReadFromCache_WithJobStateIsInvalid() {
	requiredOffset := int64(downloader.ReadChunkSize + util.MiB)
	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	jobStatus.Name = downloader.INVALID

	err := cht.cacheHandle.shouldReadFromCache(&jobStatus, requiredOffset)

	ExpectNe(nil, err)
	ExpectTrue(strings.Contains(err.Error(), util.InvalidFileDownloadJobErrMsg))
}

func (cht *cacheHandleTest) Test_shouldReadFromCache_WithJobStateIsCompleted() {
	requiredOffset := int64(downloader.ReadChunkSize + util.MiB)
	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	jobStatus.Name = downloader.COMPLETED
	jobStatus.Offset = int64(cht.object.Size)

	err := cht.cacheHandle.shouldReadFromCache(&jobStatus, requiredOffset)

	ExpectEq(nil, err)
}

func (cht *cacheHandleTest) Test_shouldReadFromCache_WithJobDownloadedOffsetIsLessThanRequiredOffset() {
	requiredOffset := int64(downloader.ReadChunkSize + util.MiB)
	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	jobStatus.Name = downloader.DOWNLOADING
	jobStatus.Offset = requiredOffset - 1

	err := cht.cacheHandle.shouldReadFromCache(&jobStatus, requiredOffset)

	ExpectNe(nil, err)
	ExpectTrue(strings.Contains(err.Error(), util.FallbackToGCSErrMsg))
}

func (cht *cacheHandleTest) Test_shouldReadFromCache_WithJobDownloadedOffsetSameAsRequiredOffset() {
	requiredOffset := int64(downloader.ReadChunkSize + util.MiB)
	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	jobStatus.Name = downloader.DOWNLOADING
	jobStatus.Offset = requiredOffset

	err := cht.cacheHandle.shouldReadFromCache(&jobStatus, requiredOffset)

	ExpectEq(nil, err)
}

func (cht *cacheHandleTest) Test_shouldReadFromCache_WithCancelledJobOffsetIsGreaterThanRequiredOffset() {
	requiredOffset := int64(util.MiB)
	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	jobStatus.Name = downloader.CANCELLED
	jobStatus.Offset = requiredOffset + 1

	err := cht.cacheHandle.shouldReadFromCache(&jobStatus, requiredOffset)

	ExpectEq(nil, err)
}

func (cht *cacheHandleTest) Test_shouldReadFromCache_WithCancelledJobOffsetIsLessThanRequiredOffset() {
	requiredOffset := int64(downloader.ReadChunkSize + util.MiB)
	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	jobStatus.Name = downloader.CANCELLED
	jobStatus.Offset = requiredOffset - 1

	err := cht.cacheHandle.shouldReadFromCache(&jobStatus, requiredOffset)

	ExpectNe(nil, err)
	ExpectTrue(strings.Contains(err.Error(), util.FallbackToGCSErrMsg))
}

func (cht *cacheHandleTest) Test_shouldReadFromCache_WithCancelledJobOffsetSameAsRequiredOffset() {
	requiredOffset := int64(downloader.ReadChunkSize + util.MiB)
	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	jobStatus.Name = downloader.CANCELLED
	jobStatus.Offset = requiredOffset

	err := cht.cacheHandle.shouldReadFromCache(&jobStatus, requiredOffset)

	ExpectEq(nil, err)
}

func (cht *cacheHandleTest) Test_shouldReadFromCache_WithJobDownloadedOffsetIsGreaterThanRequiredOffset() {
	requiredOffset := int64(util.MiB)
	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	jobStatus.Name = downloader.DOWNLOADING
	jobStatus.Offset = requiredOffset + 1

	err := cht.cacheHandle.shouldReadFromCache(&jobStatus, requiredOffset)

	ExpectEq(nil, err)
}

func (cht *cacheHandleTest) Test_shouldReadFromCache_WithJobDownloadedOffsetIsMoreThanRequiredOffset() {
	requiredOffset := int64(downloader.ReadChunkSize + util.MiB)
	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	jobStatus.Name = downloader.DOWNLOADING
	jobStatus.Offset = requiredOffset + 1

	err := cht.cacheHandle.shouldReadFromCache(&jobStatus, requiredOffset)

	ExpectEq(nil, err)
}

func (cht *cacheHandleTest) Test_shouldReadFromCache_WithNonNilJobStatusErr() {
	requiredOffset := int64(downloader.ReadChunkSize + util.MiB)
	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	jobStatus.Name = downloader.DOWNLOADING
	jobStatus.Offset = requiredOffset + 1
	jobStatus.Err = errors.New("job error")

	err := cht.cacheHandle.shouldReadFromCache(&jobStatus, requiredOffset)

	ExpectNe(nil, err)
	ExpectTrue(strings.Contains(err.Error(), util.InvalidFileDownloadJobErrMsg))
}

func (cht *cacheHandleTest) Test_checkEntryInFileInfoCache_FileInfoPresent() {
	fileInfoKey := data.FileInfoKey{
		BucketName: cht.bucket.Name(),
		ObjectName: cht.object.Name,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	AssertEq(nil, err)
	fileInfo := data.FileInfo{
		Key:              fileInfoKey,
		ObjectGeneration: cht.object.Generation,
		FileSize:         cht.object.Size,
		Offset:           cht.object.Size,
	}
	_, err = cht.cache.Insert(fileInfoKeyName, fileInfo)
	AssertEq(nil, err)

	err = cht.cacheHandle.checkEntryInFileInfoCache(cht.bucket, cht.object, 0)

	AssertEq(nil, err)
}

func (cht *cacheHandleTest) Test_checkEntryInFileInfoCache_FileInfoNotPresent() {
	fileInfoKey := data.FileInfoKey{
		BucketName: cht.bucket.Name(),
		ObjectName: cht.object.Name,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	AssertEq(nil, err)

	_ = cht.cache.Erase(fileInfoKeyName)
	err = cht.cacheHandle.checkEntryInFileInfoCache(cht.bucket, cht.object, 0)

	expectedErr := fmt.Errorf("%v: no entry found in file info cache for key %v", util.InvalidFileInfoCacheErrMsg, fileInfoKeyName)
	AssertTrue(strings.Contains(err.Error(), expectedErr.Error()))
}

func (cht *cacheHandleTest) Test_checkEntryInFileInfoCache_FileInfoGenerationChanged() {
	fileInfoKey := data.FileInfoKey{
		BucketName: cht.bucket.Name(),
		ObjectName: cht.object.Name,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	AssertEq(nil, err)
	fileInfo := data.FileInfo{
		Key:              fileInfoKey,
		ObjectGeneration: cht.object.Generation + 1,
		FileSize:         cht.object.Size,
		Offset:           cht.object.Size,
	}
	_, err = cht.cache.Insert(fileInfoKeyName, fileInfo)
	AssertEq(nil, err)

	err = cht.cacheHandle.checkEntryInFileInfoCache(cht.bucket, cht.object, int64(cht.object.Size-1))

	expectedErr := fmt.Errorf("%v: generation of cached object: %v is different from required generation: ", util.InvalidFileInfoCacheErrMsg, fileInfo.ObjectGeneration)
	AssertTrue(strings.Contains(err.Error(), expectedErr.Error()))
}

func (cht *cacheHandleTest) Test_Read_RequestingMoreOffsetThanSize() {
	dst := make([]byte, ReadContentSize)
	offset := int64(cht.object.Size + 1)

	n, cacheHit, err := cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, offset, dst)

	ExpectNe(nil, err)
	ExpectEq(0, n)
	ExpectFalse(cacheHit)
	ExpectTrue(strings.Contains(err.Error(), "wrong offset requested"))
}

func (cht *cacheHandleTest) Test_Read_WithNilFileHandle() {
	dst := make([]byte, ReadContentSize)
	offset := int64(5)
	cht.cacheHandle.fileHandle = nil

	n, cacheHit, err := cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, offset, dst)
	ExpectNe(nil, err)
	ExpectEq(0, n)
	ExpectFalse(cacheHit)
	ExpectEq(util.InvalidFileHandleErrMsg, err.Error())
}

func (cht *cacheHandleTest) Test_Read_Random() {
	dst := make([]byte, ReadContentSize)
	offset := int64(cht.object.Size - ReadContentSize)
	cht.cacheHandle.isSequential = false
	cht.cacheHandle.cacheFileForRangeRead = true

	// Since, it's a random read hence will not wait to download till requested offset.
	n, cacheHit, err := cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, offset, dst)

	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	ExpectLt(jobStatus.Offset, offset)
	ExpectEq(n, 0)
	ExpectFalse(cacheHit)
	ExpectNe(nil, err)
	ExpectTrue(strings.Contains(err.Error(), util.FallbackToGCSErrMsg))
}

func (cht *cacheHandleTest) Test_Read_RandomWithNoRandomDownload() {
	dst := make([]byte, ReadContentSize)
	offset := int64(cht.object.Size - ReadContentSize)
	cht.cacheHandle.isSequential = false

	// Since, it's a random read hence will not wait to download till requested offset.
	n, cacheHit, err := cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, offset, dst)

	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	ExpectEq(jobStatus.Name, downloader.NOT_STARTED)
	ExpectLt(jobStatus.Offset, offset)
	ExpectEq(n, 0)
	ExpectFalse(cacheHit)
	ExpectNe(nil, err)
	ExpectTrue(strings.Contains(err.Error(), util.FallbackToGCSErrMsg))
}

func (cht *cacheHandleTest) Test_Read_RandomWithNoRandomDownloadButCacheHit() {
	ctx := context.Background()
	// Download the job till util.MiB
	jobStatus, err := cht.cacheHandle.fileDownloadJob.Download(ctx, int64(2*util.MiB), true)
	AssertEq(nil, err)
	AssertEq(jobStatus.Name, downloader.DOWNLOADING)
	dst := make([]byte, ReadContentSize)
	offset := int64(1)
	cht.cacheHandle.isSequential = false

	// Since, it's a random read hence will not wait to download till requested offset.
	_, cacheHit, err := cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, offset, dst)

	jobStatus = cht.cacheHandle.fileDownloadJob.GetStatus()
	ExpectTrue(jobStatus.Name == downloader.DOWNLOADING || jobStatus.Name == downloader.COMPLETED)
	ExpectGe(jobStatus.Offset, offset)
	ExpectEq(cacheHit, true)
	ExpectEq(nil, err)
	cht.verifyContentRead(offset, dst)
}

func (cht *cacheHandleTest) Test_Read_RandomWithNoRandomDownloadButCacheHitInCancelledState() {
	ctx := context.Background()
	// Download the job till util.MiB
	jobStatus, err := cht.cacheHandle.fileDownloadJob.Download(ctx, int64(2*util.MiB), true)
	AssertEq(nil, err)
	AssertEq(jobStatus.Name, downloader.DOWNLOADING)
	cht.cacheHandle.fileDownloadJob.Cancel()
	jobStatus = cht.cacheHandle.fileDownloadJob.GetStatus()
	AssertEq(jobStatus.Name, downloader.CANCELLED)
	dst := make([]byte, ReadContentSize)
	offset := int64(1)
	cht.cacheHandle.isSequential = false

	// Since, it's a random read hence will not wait to download till requested offset.
	_, cacheHit, err := cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, offset, dst)

	jobStatus = cht.cacheHandle.fileDownloadJob.GetStatus()
	ExpectTrue(jobStatus.Name == downloader.CANCELLED)
	ExpectGe(jobStatus.Offset, offset)
	ExpectEq(nil, err)
	ExpectTrue(cacheHit)
	cht.verifyContentRead(offset, dst)
}

func (cht *cacheHandleTest) Test_Read_Sequential() {
	dst := make([]byte, ReadContentSize)
	offset := int64(cht.object.Size - ReadContentSize)
	cht.cacheHandle.isSequential = true
	cht.cacheHandle.prevOffset = offset - util.MiB
	cht.cacheHandle.cacheFileForRangeRead = true

	// Since, it's a sequential read, hence will wait to download till requested offset.
	_, cacheHit, err := cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, offset, dst)

	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	ExpectFalse(cacheHit)
	ExpectGe(jobStatus.Offset, offset)
	cht.verifyContentRead(offset, dst)
	ExpectEq(nil, err)
}

func (cht *cacheHandleTest) Test_Read_SequentialToRandom() {
	dst := make([]byte, ReadContentSize)
	firstReqOffset := int64(0)
	cht.cacheHandle.isSequential = true
	cht.cacheHandle.cacheFileForRangeRead = true
	// Since, it's a sequential read, hence will wait to download till requested offset.
	_, cacheHit, err := cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, firstReqOffset, dst)
	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	AssertGe(jobStatus.Offset, firstReqOffset)
	AssertEq(nil, err)
	ExpectFalse(cacheHit)
	AssertEq(cht.cacheHandle.isSequential, true)

	secondReqOffset := int64(cht.object.Size - ReadContentSize) // type will change to random.
	_, cacheHit, err = cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, secondReqOffset, dst)

	jobStatus = cht.cacheHandle.fileDownloadJob.GetStatus()
	ExpectLe(jobStatus.Offset, secondReqOffset)
	ExpectNe(nil, err)
	ExpectFalse(cacheHit)
	ExpectTrue(strings.Contains(err.Error(), util.FallbackToGCSErrMsg))
	ExpectEq(cht.cacheHandle.isSequential, false)
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

	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	ExpectGe(jobStatus.Offset, offset)
	ExpectFalse(cacheHit)
	cht.verifyContentRead(offset, dst[:len(dst)-extraBuffer])
	ExpectEq(nil, err)
}

func (cht *cacheHandleTest) Test_Read_FileInfoRemoved() {
	dst := make([]byte, ReadContentSize)
	cht.cacheHandle.isSequential = true
	cht.cacheHandle.cacheFileForRangeRead = true
	// First let the cache populated (we are doing this so that we can externally
	// modify file info cache for this unit test without hampering download job).
	_, cacheHit, err := cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, 0, dst)
	AssertEq(nil, err)
	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	ExpectGe(jobStatus.Offset, ReadContentSize)
	fileInfoKey := data.FileInfoKey{
		BucketName: cht.bucket.Name(),
		ObjectName: cht.object.Name,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	AssertEq(nil, err)
	ExpectFalse(cacheHit)

	// Delete the file info entry and again perform read
	_ = cht.cache.Erase(fileInfoKeyName)
	_, cacheHit, err = cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, 0, dst)

	expectedErr := fmt.Errorf("%v: no entry found in file info cache for key %v", util.InvalidFileInfoCacheErrMsg, fileInfoKeyName)
	AssertTrue(strings.Contains(err.Error(), expectedErr.Error()))
	AssertFalse(cacheHit)
}

func (cht *cacheHandleTest) Test_Read_FileInfoGenerationChanged() {
	dst := make([]byte, ReadContentSize)
	cht.cacheHandle.isSequential = true
	cht.cacheHandle.cacheFileForRangeRead = true
	// First let the cache populated (we are doing this so that we can externally
	// modify file info cache for this unit test without hampering download job).
	_, cacheHit, err := cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, 0, dst)
	AssertEq(nil, err)
	AssertFalse(cacheHit)
	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	ExpectGe(jobStatus.Offset, ReadContentSize)
	fileInfoKey := data.FileInfoKey{
		BucketName: cht.bucket.Name(),
		ObjectName: cht.object.Name,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	AssertEq(nil, err)
	fileInfo := cht.cache.LookUp(fileInfoKeyName)
	fileInfoData := fileInfo.(data.FileInfo)

	// Update the file info entry generation and again perform read
	fileInfoData.ObjectGeneration = 1
	err = cht.cache.UpdateWithoutChangingOrder(fileInfoKeyName, fileInfoData)
	AssertEq(nil, err)
	_, cacheHit, err = cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, 0, dst)

	expectedErr := fmt.Errorf("%v: generation of cached object: %v is different from required generation: ", util.InvalidFileInfoCacheErrMsg, fileInfoData.ObjectGeneration)
	AssertTrue(strings.Contains(err.Error(), expectedErr.Error()))
	AssertFalse(cacheHit)
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
	ExpectGe(jobStatus.Offset, offset)
	ExpectEq(n, ReadContentSize)
	cht.verifyContentRead(offset, dst[:n])
	ExpectFalse(cacheHit)
	ExpectEq(nil, err)

	// Second read should be cache hit.
	n, cacheHit, err = cht.cacheHandle.Read(context.Background(), cht.bucket, cht.object, offset, dst)

	ExpectEq(n, ReadContentSize)
	ExpectTrue(cacheHit)
	ExpectEq(nil, err)
}
