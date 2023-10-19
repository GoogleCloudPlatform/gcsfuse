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
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/file/downloader"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/gcs"
	. "github.com/jacobsa/ogletest"
)

const CacheMaxSize = 50
const TestFilePath = "test_file.txt"
const TestFileContent = "abcdefghijklmnop"
const DstBufferLen = 5
const TestOffset = 5

const FileContentFromTestOffset = "fghij"

const TestBucketName = "test-bucket"
const TestObjectName = "test.txt"
const TestObjectNameNotInFileInfoCache = "test_not_in_file_info.txt"
const TestObjectGeneration = "3434343"
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
	fullPath, err := filepath.Abs(filePath)
	AssertEq(nil, err)

	file, err := os.Create(fullPath)
	if err != nil {
		return nil, err
	}

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
		BucketName:         TestBucketName,
		BucketCreationTime: time.Unix(data.TestTimeInEpoch, 0),
		ObjectName:         TestObjectName,
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
		Name: TestObjectName,
		Size: TestFileInfoFileSize,
	}
}

func init() {
	RegisterTestSuite(&cacheHandleTest{})
}

func (t *cacheHandleTest) SetUp(*TestInfo) {
	t.ch = &CacheHandle{
		fileHandle:      &os.File{},
		fileDownloadJob: &downloader.Job{},
		fileInfoCache:   getPrepopulatedLRUCacheWithFileInfo(),
	}
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

	// This will return the minGCSObject whose entry is in fileInfoCache.
	minGCSObject := getTestMinGCSObject()
	minGCSObject.Name = TestObjectNameNotInFileInfoCache

	_, err := t.ch.Read(minGCSObject, testBucket{BucketName: TestBucketName}, TestOffset, dst)

	ExpectEq(InvalidFileInfo, err.Error())
}

func (t *cacheHandleTest) TestReadWithReadFromLocalCachedFilePath() {
	file, err := createFileWithContent(TestFilePath, TestFileContent)
	defer deleteFile(TestFilePath)

	AssertEq(nil, err)

	t.ch.fileHandle = file
	dst := make([]byte, DstBufferLen)

	tb := getTestMinGCSObject()
	tb.Name = TestObjectName
	n, err := t.ch.Read(tb, testBucket{BucketName: TestBucketName}, TestOffset, dst)

	AssertEq(nil, err)
	AssertEq(n, DstBufferLen)
	AssertEq(FileContentFromTestOffset, string(dst))
}

func (t *cacheHandleTest) TestClose() {
	file, err := createFileWithContent(TestFilePath, TestFileContent)
	defer deleteFile(TestFilePath)
	AssertEq(nil, err)
	t.ch.fileHandle = file

	err = t.ch.Close()
	AssertEq(nil, err)

	ExpectEq(nil, t.ch.fileHandle)
}

// TODO (princer): write test which validates download flow in the cache_handle.read()
