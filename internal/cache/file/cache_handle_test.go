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
	AssertEq(cht.fileSpec.Perm, fileStat.Mode())

	// Create a byte buffer of same len as expectedContent.
	buf := make([]byte, len(expectedContent))

	// Read from file and compare with expectedContent.
	_, err = cht.cacheHandle.fileHandle.Seek(readStartOffset, 0)
	AssertEq(nil, err)
	_, err = io.ReadFull(cht.cacheHandle.fileHandle, buf)
	AssertEq(nil, err)
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
	cht.fileSpec = data.FileSpec{Path: localDownloadedPath, Perm: util.DefaultFilePerm}

	readLocalFileHandle, err := util.CreateFile(cht.fileSpec, os.O_RDONLY)
	AssertEq(nil, err)

	fileDownloadJob := downloader.NewJob(cht.object, cht.bucket, cht.cache, DefaultSequentialReadSizeMb, cht.fileSpec)

	cht.cacheHandle = NewCacheHandle(readLocalFileHandle, fileDownloadJob, cht.cache, 0)
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
	ExpectTrue(strings.Contains(err.Error(), util.InvalidFileDownloadJobErrMsg))
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

func (cht *cacheHandleTest) Test_Read_RequestingMoreOffsetThanSize() {
	dst := make([]byte, ReadContentSize)
	offset := int64(cht.object.Size + 1)

	n, err := cht.cacheHandle.Read(context.Background(), cht.object, true, offset, dst)

	ExpectNe(nil, err)
	ExpectEq(0, n)
	ExpectTrue(strings.Contains(err.Error(), "wrong offset requested"))
}

func (cht *cacheHandleTest) Test_Read_WithNilFileHandle() {
	dst := make([]byte, ReadContentSize)
	offset := int64(5)
	cht.cacheHandle.fileHandle = nil

	n, err := cht.cacheHandle.Read(context.Background(), cht.object, true, offset, dst)
	ExpectNe(nil, err)
	ExpectEq(0, n)
	ExpectEq(util.InvalidFileHandleErrMsg, err.Error())
}

func (cht *cacheHandleTest) Test_Read_Random() {
	dst := make([]byte, ReadContentSize)
	offset := int64(cht.object.Size - ReadContentSize)
	cht.cacheHandle.isSequential = false

	// Since, it's a random read hence will not wait to download till requested offset.
	n, err := cht.cacheHandle.Read(context.Background(), cht.object, true, offset, dst)

	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	ExpectLt(jobStatus.Offset, offset)
	ExpectEq(n, 0)
	ExpectNe(nil, err)
	ExpectTrue(strings.Contains(err.Error(), util.FallbackToGCSErrMsg))
}

func (cht *cacheHandleTest) Test_Read_RandomWithNoRandomDownload() {
	dst := make([]byte, ReadContentSize)
	offset := int64(cht.object.Size - ReadContentSize)
	cht.cacheHandle.isSequential = false

	// Since, it's a random read hence will not wait to download till requested offset.
	n, err := cht.cacheHandle.Read(context.Background(), cht.object, false, offset, dst)

	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	ExpectEq(jobStatus.Name, downloader.NOT_STARTED)
	ExpectLt(jobStatus.Offset, offset)
	ExpectEq(n, 0)
	ExpectNe(nil, err)
	ExpectTrue(strings.Contains(err.Error(), util.InvalidFileDownloadJobErrMsg))
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
	_, err = cht.cacheHandle.Read(context.Background(), cht.object, false, offset, dst)

	jobStatus = cht.cacheHandle.fileDownloadJob.GetStatus()
	ExpectTrue(jobStatus.Name == downloader.DOWNLOADING || jobStatus.Name == downloader.COMPLETED)
	ExpectGe(jobStatus.Offset, offset)
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
	_, err = cht.cacheHandle.Read(context.Background(), cht.object, false, offset, dst)

	jobStatus = cht.cacheHandle.fileDownloadJob.GetStatus()
	ExpectTrue(jobStatus.Name == downloader.CANCELLED)
	ExpectGe(jobStatus.Offset, offset)
	ExpectEq(nil, err)
	cht.verifyContentRead(offset, dst)
}

func (cht *cacheHandleTest) Test_Read_Sequential() {
	dst := make([]byte, ReadContentSize)
	offset := int64(cht.object.Size - ReadContentSize)
	cht.cacheHandle.isSequential = true
	cht.cacheHandle.prevOffset = offset - util.MiB

	// Since, it's a sequential read, hence will wait to download till requested offset.
	_, err := cht.cacheHandle.Read(context.Background(), cht.object, true, offset, dst)

	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	ExpectGe(jobStatus.Offset, offset)
	cht.verifyContentRead(offset, dst)
	ExpectEq(nil, err)
}

func (cht *cacheHandleTest) Test_Read_SequentialToRandom() {
	dst := make([]byte, ReadContentSize)
	firstReqOffset := int64(0)
	cht.cacheHandle.isSequential = true
	// Since, it's a sequential read, hence will wait to download till requested offset.
	_, err := cht.cacheHandle.Read(context.Background(), cht.object, true, firstReqOffset, dst)
	jobStatus := cht.cacheHandle.fileDownloadJob.GetStatus()
	AssertGe(jobStatus.Offset, firstReqOffset)
	AssertEq(nil, err)
	AssertEq(cht.cacheHandle.isSequential, true)

	secondReqOffset := int64(cht.object.Size - ReadContentSize) // type will change to random.
	_, err = cht.cacheHandle.Read(context.Background(), cht.object, true, secondReqOffset, dst)

	jobStatus = cht.cacheHandle.fileDownloadJob.GetStatus()
	ExpectLe(jobStatus.Offset, secondReqOffset)
	ExpectNe(nil, err)
	ExpectTrue(strings.Contains(err.Error(), util.FallbackToGCSErrMsg))
	ExpectEq(cht.cacheHandle.isSequential, false)
}
