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
	"os"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/file/downloader"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/gcs"
	. "github.com/jacobsa/ogletest"
)

const CacheMaxSize = 50
const TestFilePath = "test_file.txt"
const DefaultFileMode = 0644
const TestFileContent = "abcdefghijklmnop"
const DstBufferLen = 5
const TestOffset = 5

const FileContentFromTestOffset = "fghij"

const TestBucketName = "test-bucket"
const TestObjectName = "test.txt"
const TestObjectNameNotInFileInfoCache = "test_not_in_file_info.txt"
const TestObjectGeneration = 3434343
const TestFileInfoFileSize = 32

func TestCacheHandle(t *testing.T) { RunTests(t) }

type cacheHandleTest struct {
	ch *CacheHandle
}

// Mocking test bucket, which contains Name() method.
type testBucket struct {
	gcs.Bucket
	BucketName string
}

func (tb testBucket) Name() string {
	return tb.BucketName
}

// Test helper.
func createFileWithContent(filePath string, content string) (*os.File, error) {
	fileSpec := data.FileSpec{
		Path: filePath,
		Perm: os.FileMode(DefaultFileMode),
	}

	file, err := util.CreateFile(fileSpec, os.O_RDWR)
	AssertEq(nil, err)

	_, err = file.WriteString(content)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func deleteFile(filePath string) error {
	err := os.Remove(filePath)

	if err != nil {
		return err
	}

	return nil
}

func getPrepopulatedLRUCacheWithFileInfo() *lru.Cache {
	cache := lru.NewCache(CacheMaxSize)

	fileInfoKey := data.FileInfoKey{
		BucketName: TestBucketName,
		ObjectName: TestObjectName,
	}

	fileInfo := data.FileInfo{
		Key:              fileInfoKey,
		Offset:           uint64(len(TestFileContent)),
		ObjectGeneration: TestObjectGeneration,
		FileSize:         TestFileInfoFileSize,
	}

	key, err := fileInfoKey.Key()
	AssertEq(nil, err)

	_, err = cache.Insert(key, fileInfo)
	AssertEq(nil, err)

	return cache
}

// getTestMinGCSObject returns the MinGCSObject whose entry as fileInfo
// present in the pre-populated fileInfoCache.
func getTestMinGCSObject() *gcs.MinObject {
	return &gcs.MinObject{
		Name:       TestObjectName,
		Size:       TestFileInfoFileSize,
		Generation: TestObjectGeneration,
	}
}

func init() {
	RegisterTestSuite(&cacheHandleTest{})
}

func (t *cacheHandleTest) SetUp(*TestInfo) {
	file, err := createFileWithContent(TestFilePath, TestFileContent)
	AssertEq(nil, err)

	t.ch = &CacheHandle{
		fileHandle:      file,
		fileDownloadJob: &downloader.Job{},
		fileInfoCache:   getPrepopulatedLRUCacheWithFileInfo(),
	}
}

func (t *cacheHandleTest) TearDown() {
	err := t.ch.Close()
	AssertEq(nil, err)

	err = deleteFile(TestFilePath)
	AssertEq(nil, err)
}

func (t *cacheHandleTest) TestValidateCacheHandleWithNilFileHandle() {
	t.ch.fileHandle = nil

	err := t.ch.validateCacheHandle()

	ExpectEq(InvalidFileHandle, err.Error())
}

func (t *cacheHandleTest) TestValidateCacheHandleWithNilFileDownloadJob() {
	t.ch.fileDownloadJob = nil

	err := t.ch.validateCacheHandle()

	ExpectEq(InvalidFileDownloadJob, err.Error())
}

func (t *cacheHandleTest) TestValidateCacheHandleWithNilFileInfoCache() {
	t.ch.fileInfoCache = nil

	err := t.ch.validateCacheHandle()

	ExpectEq(InvalidFileInfoCache, err.Error())
}

func (t *cacheHandleTest) TestValidateCacheHandleWithNonNilMemberAttributes() {
	err := t.ch.validateCacheHandle()

	ExpectEq(nil, err)
}

func (t *cacheHandleTest) TestReadWithFileInfoKeyNotPresentInTheCache() {
	dst := make([]byte, DstBufferLen)

	// Create the minGCS object whose entry is not in fileInfoCache.
	minGCSObject := getTestMinGCSObject()
	minGCSObject.Name = TestObjectNameNotInFileInfoCache

	_, err := t.ch.Read(context.Background(), minGCSObject, testBucket{BucketName: TestBucketName}, TestOffset, dst)

	ExpectEq(InvalidCacheHandle, err.Error())
}

func (t *cacheHandleTest) TestReadWithLocalCachedFilePathWhenGenerationInCacheAndObjectMatch() {
	dst := make([]byte, DstBufferLen)
	tb := getTestMinGCSObject()
	tb.Name = TestObjectName

	n, err := t.ch.Read(context.Background(), tb, testBucket{BucketName: TestBucketName}, TestOffset, dst)

	ExpectEq(nil, err)
	ExpectEq(n, DstBufferLen)
	ExpectEq(FileContentFromTestOffset, string(dst))
}

func (t *cacheHandleTest) TestReadWithLocalCachedFilePathWhenGenerationInCacheAndObjectNotMatch() {
	dst := make([]byte, DstBufferLen)
	tb := getTestMinGCSObject()
	tb.Generation = 0 // overriding to not match with TestObjectGeneration
	tb.Name = TestObjectName

	n, err := t.ch.Read(context.Background(), tb, testBucket{BucketName: TestBucketName}, TestOffset, dst)

	ExpectEq(InvalidCacheHandle, err.Error())
	ExpectEq(n, 0)
}

func (t *cacheHandleTest) Test_checkIfEntryExistWithCorrectGenerationIfNoSuchEntryInCache() {
	tb := getTestMinGCSObject()
	tb.Name = TestObjectNameNotInFileInfoCache

	ok, err := t.ch.checkIfEntryExistWithCorrectGenerationAndOffset(tb, testBucket{BucketName: TestBucketName}, 0)

	ExpectEq(nil, err)
	ExpectEq(false, ok)
}

func (t *cacheHandleTest) Test_checkIfEntryExistWithCorrectGenerationIfGenerationMatchWithGreaterRequiredOffset() {
	tb := getTestMinGCSObject()
	tb.Name = TestObjectName
	requiredOffset := len(TestFileContent) + 1

	ok, err := t.ch.checkIfEntryExistWithCorrectGenerationAndOffset(tb, testBucket{BucketName: TestBucketName}, int64(requiredOffset))

	ExpectEq(nil, err)
	ExpectEq(false, ok)
}

func (t *cacheHandleTest) Test_checkIfEntryExistWithCorrectGenerationIfGenerationMatchWithLesserRequiredOffset() {
	tb := getTestMinGCSObject()
	tb.Name = TestObjectName
	requiredOffset := len(TestFileContent) - 1

	ok, err := t.ch.checkIfEntryExistWithCorrectGenerationAndOffset(tb, testBucket{BucketName: TestBucketName}, int64(requiredOffset))

	ExpectEq(nil, err)
	ExpectEq(true, ok)
}

func (t *cacheHandleTest) Test_checkIfEntryExistWithCorrectGenerationIfGenerationMatchWithEqualRequiredOffset() {
	tb := getTestMinGCSObject()
	tb.Name = TestObjectName
	requiredOffset := len(TestFileContent)

	ok, err := t.ch.checkIfEntryExistWithCorrectGenerationAndOffset(tb, testBucket{BucketName: TestBucketName}, int64(requiredOffset))

	ExpectEq(nil, err)
	ExpectEq(true, ok)
}

func (t *cacheHandleTest) Test_checkIfEntryExistWithCorrectGenerationIfGenerationNotMatch() {
	tb := getTestMinGCSObject()
	tb.Generation = 0 // overriding to not match with TestObjectGeneration
	tb.Name = TestObjectName

	ok, err := t.ch.checkIfEntryExistWithCorrectGenerationAndOffset(tb, testBucket{BucketName: TestBucketName}, 0)

	ExpectEq(nil, err)
	ExpectEq(false, ok)
}

func (t *cacheHandleTest) Test_checkIfEntryExistWithCorrectGenerationIfGenerationMatch() {
	tb := getTestMinGCSObject()
	tb.Name = TestObjectName

	ok, err := t.ch.checkIfEntryExistWithCorrectGenerationAndOffset(tb, testBucket{BucketName: TestBucketName}, 0)

	ExpectEq(nil, err)
	ExpectEq(true, ok)
}

func (t *cacheHandleTest) TestClose() {
	err := t.ch.Close()
	AssertEq(nil, err)

	ExpectEq(nil, t.ch.fileHandle)
}

func (t *cacheHandleTest) TestIsSequentialWhenReadTypeIsSequential() {
	t.ch.isSequential = false
	currentOffset := 3

	ExpectEq(false, t.ch.IsSequential(int64(currentOffset)))
}

func (t *cacheHandleTest) TestIsSequentialWhenPrevOffsetGreateThanCurrent() {
	t.ch.isSequential = true
	t.ch.prevOffset = 5
	currentOffset := 3

	ExpectEq(false, t.ch.IsSequential(int64(currentOffset)))
}

func (t *cacheHandleTest) TestIsSequentialWhenOffsetDiffIsMoreThanMaxAllowed() {
	t.ch.isSequential = true
	t.ch.prevOffset = 5
	currentOffset := 8 + MaxAllowedMB

	ExpectEq(false, t.ch.IsSequential(int64(currentOffset)))
}

func (t *cacheHandleTest) TestIsSequentialWhenOffsetDiffIsLessThanMaxAllowed() {
	t.ch.isSequential = true
	t.ch.prevOffset = 5
	currentOffset := 10

	ExpectEq(true, t.ch.IsSequential(int64(currentOffset)))
}

func (t *cacheHandleTest) TestIsSequentialWhenOffsetDiffIsEqualToMaxAllowed() {
	t.ch.isSequential = true
	t.ch.prevOffset = 5
	currentOffset := 5 + MaxAllowedMB

	ExpectEq(true, t.ch.IsSequential(int64(currentOffset)))
}

func (t *cacheHandleTest) TestDoesContributeForJobRefIfTrue() {
	t.ch.contributesToJobRefCount = true

	ExpectEq(true, t.ch.DoesContributeForJobRef())
}

func (t *cacheHandleTest) TestDoesContributeForJobRefIfFalse() {
	t.ch.contributesToJobRefCount = false

	ExpectEq(false, t.ch.DoesContributeForJobRef())
}

func (t *cacheHandleTest) TestRemoveContributionForJobRefIfTrue() {
	t.ch.contributesToJobRefCount = true

	t.ch.RemoveContributionForJobRef()

	ExpectEq(false, t.ch.DoesContributeForJobRef())
}

func (t *cacheHandleTest) TestRemoveContributionForJobRefIfFalse() {
	t.ch.contributesToJobRefCount = false

	t.ch.RemoveContributionForJobRef()

	ExpectEq(false, t.ch.DoesContributeForJobRef())
}

// TODO (princer): write test which validates download flow in the cache_handle.read()
