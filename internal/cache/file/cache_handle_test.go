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
	"strconv"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/file/downloader"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/gcs"
	. "github.com/jacobsa/ogletest"
)

func TestCacheHandle(t *testing.T) { RunTests(t) }

type cacheHandleTest struct {
	ch *CacheHandle
}

const CacheMaxSize = 50
const DstLen = 5
const TestFilePath = "test_file.txt"
const TestFileContent = "abcdefghijklmnop"
const TestBucketName = "test-bucket"

/*********** Fake Bucket **********/
type testBucket struct {
	gcs.Bucket
	BucketName string
}

func (tb testBucket) Name() string {
	return tb.BucketName
}

/************ File Utility ************/
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

	key1 := "test1" + strconv.FormatInt(data.TestTimeInEpoch, 10) + "test1.txt"
	_, err := cache.Insert(key1, data.FileInfo{
		Key:              data.FileInfoKey{BucketName: "test", BucketCreationTime: time.Unix(data.TestTimeInEpoch, 0), ObjectName: "test1.txt"},
		Offset:           uint64(len(TestFileContent)),
		ObjectGeneration: "test1",
		FileSize:         5,
	})
	AssertEq(nil, err)

	key2 := "test2" + strconv.FormatInt(data.TestTimeInEpoch, 10) + "test2.txt"
	_, err = cache.Insert(key2, data.FileInfo{
		Key:              data.FileInfoKey{BucketName: "test2", BucketCreationTime: time.Unix(data.TestTimeInEpoch, 0), ObjectName: "test2.txt"},
		Offset:           20,
		ObjectGeneration: "test2",
		FileSize:         10,
	})
	AssertEq(nil, err)

	return &cache
}

func getTestMinGCSObject() *gcs.MinObject {
	return &gcs.MinObject{
		Name: "test",
		Size: 10,
	}
}

func getDefaultCacheHandle() CacheHandle {
	cache := getPrepopulatedLRUCacheWithFileInfo()
	return CacheHandle{
		fileHandle:      &os.File{},
		fileDownloadJob: &downloader.Job{},
		fileInfoCache:   cache,
	}
}

func init() {
	RegisterTestSuite(&cacheHandleTest{})
}

func (t *cacheHandleTest) Setup(*TestInfo) {

}

func (t *cacheHandleTest) TestValidateCacheHandleWithNilFileHandle() {
	cfh := getDefaultCacheHandle()
	cfh.fileHandle = nil

	err := cfh.validateCacheHandle()

	ExpectEq(InvalidFileHandle, err.Error())
}

func (t *cacheHandleTest) TestValidateCacheHandleWithNilFileDownloadJob() {
	cfh := getDefaultCacheHandle()
	cfh.fileDownloadJob = nil

	err := cfh.validateCacheHandle()

	ExpectEq(InvalidFileDownloadJob, err.Error())
}

func (t *cacheHandleTest) TestValidateCacheHandleWithNilFileInfoCache() {
	cfh := getDefaultCacheHandle()
	cfh.fileInfoCache = nil

	err := cfh.validateCacheHandle()

	ExpectEq(InvalidFileInfoCache, err.Error())
}

func (t *cacheHandleTest) TestValidateCacheHandleWithNonNilMemberAttributes() {
	cfh := getDefaultCacheHandle()

	err := cfh.validateCacheHandle()

	ExpectEq(nil, err)
}

func (t *cacheHandleTest) TestReadWithFileInfoKeyNotPresentInTheCache() {
	cfh := getDefaultCacheHandle()
	dst := make([]byte, DstLen)

	_, err := cfh.Read(getTestMinGCSObject(), testBucket{BucketName: "test1"}, 10, dst)

	ExpectEq(InvalidFileInfo, err.Error())
}

func (t *cacheHandleTest) TestReadWithReadFromLocalCachedFilePath() {
	file, err := createFileWithContent(TestFilePath, TestFileContent)
	defer deleteFile(TestFilePath)

	AssertEq(nil, err)

	cfh := getDefaultCacheHandle()
	cfh.fileHandle = file
	dst := make([]byte, DstLen)

	tb := getTestMinGCSObject()
	tb.Name = "test1.txt"
	n, err := cfh.Read(tb, testBucket{BucketName: "test1"}, 5, dst)

	AssertEq(nil, err)
	AssertEq(n, DstLen)
	AssertEq("fghij", string(dst))
}

// TODO (princer): write test which validates download flow in the cache_handle.read()
