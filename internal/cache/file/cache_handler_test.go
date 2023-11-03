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
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
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

const HandlerCacheMaxSize = TestObjectSize + 20

func TestCacheHandler(t *testing.T) { RunTests(t) }

type cacheHandlerTest struct {
	jobManager      *downloader.JobManager
	bucket          gcs.Bucket
	fakeStorage     storage.FakeStorage
	object          *gcs.MinObject
	cache           *lru.Cache
	cacheHandler    *CacheHandler
	downloadPath    string
	fileInfoKeyName string
	cacheLocation   string
}

func init() { RegisterTestSuite(&cacheHandlerTest{}) }

func (chrT *cacheHandlerTest) addTestFileInfoEntryInCache(bucketName string, objectName string) string {
	// Add an entry into
	fileInfoKey := data.FileInfoKey{
		BucketName: bucketName,
		ObjectName: objectName,
	}
	fileInfo := data.FileInfo{
		Key:              fileInfoKey,
		ObjectGeneration: chrT.object.Generation,
		FileSize:         chrT.object.Size,
		Offset:           0,
	}

	fileInfoKeyName, err := fileInfoKey.Key()
	AssertEq(nil, err)

	_, err = chrT.cache.Insert(fileInfoKeyName, fileInfo)
	AssertEq(nil, err)

	return fileInfoKeyName
}

func (chrT *cacheHandlerTest) SetUp(*TestInfo) {
	locker.EnableInvariantsCheck()
	chrT.cacheLocation = path.Join(os.Getenv("HOME"), "cache/location")

	// Create bucket in fake storage.
	chrT.fakeStorage = storage.NewFakeStorage()
	storageHandle := chrT.fakeStorage.CreateStorageHandle()
	chrT.bucket = storageHandle.BucketHandle(storage.TestBucketName, "")

	// Create test object in the bucket.
	ctx := context.Background()
	testObjectContent := make([]byte, TestObjectSize)
	n, err := rand.Read(testObjectContent)
	AssertEq(TestObjectSize, n)
	AssertEq(nil, err)
	objects := map[string][]byte{TestObjectName: testObjectContent}
	err = storageutil.CreateObjects(ctx, chrT.bucket, objects)
	AssertEq(nil, err)

	gcsObj, err := chrT.bucket.StatObject(ctx, &gcs.StatObjectRequest{Name: TestObjectName,
		ForceFetchFromGcs: true})
	AssertEq(nil, err)
	minObject := storageutil.ConvertObjToMinObject(gcsObj)
	chrT.object = &minObject

	// fileInfoCache with testFileInfoEntry
	chrT.cache = lru.NewCache(HandlerCacheMaxSize)
	chrT.fileInfoKeyName = chrT.addTestFileInfoEntryInCache(storage.TestBucketName, TestObjectName)

	// Job manager
	chrT.jobManager = downloader.NewJobManager(chrT.cache, util.DefaultFilePerm, chrT.cacheLocation, DefaultSequentialReadSizeMb)

	// Mocked cached handler object.
	chrT.cacheHandler = NewCacheHandler(chrT.cache, chrT.jobManager, chrT.cacheLocation)

	chrT.downloadPath = util.GetDownloadPath(chrT.cacheHandler.cacheLocation, util.GetObjectPath(chrT.bucket.Name(), chrT.object.Name))
}

func (chrT *cacheHandlerTest) getMinObject(objName string, objContent string) *gcs.MinObject {
	ctx := context.Background()
	objects := map[string][]byte{objName: []byte(objContent)}
	err := storageutil.CreateObjects(ctx, chrT.bucket, objects)
	AssertEq(nil, err)

	gcsObj, err := chrT.bucket.StatObject(ctx, &gcs.StatObjectRequest{Name: objName,
		ForceFetchFromGcs: true})
	AssertEq(nil, err)
	minObject := storageutil.ConvertObjToMinObject(gcsObj)
	return &minObject
}

func (chrT *cacheHandlerTest) TearDown() {
	chrT.fakeStorage.ShutDown()
	operations.RemoveDir(chrT.cacheLocation)
}

// This will return true, if file exist, but might return false also.
func IsFileExist(filePath string) bool {
	_, err := os.Stat(filePath)

	if err == nil {
		return true
	}

	if os.IsNotExist(err) {
		return false
	}

	return false
}

func (chrT *cacheHandlerTest) Test_performPostEvictionWork() {
	cacheHandle, err := chrT.cacheHandler.GetCacheHandle(chrT.object, chrT.bucket, 0)
	AssertEq(nil, err)
	AssertNe(nil, cacheHandle)
	AssertEq(true, IsFileExist(chrT.downloadPath))
	fileInfo := chrT.cache.LookUp(chrT.fileInfoKeyName)
	fileInfoData := fileInfo.(data.FileInfo)
	job := chrT.jobManager.GetJob(chrT.object, chrT.bucket)
	jobStatusBefore := job.GetStatus()
	AssertEq(jobStatusBefore.Name, downloader.NOT_STARTED)
	jobStatusBefore, err = cacheHandle.fileDownloadJob.Download(context.Background(), int64(util.MiB), false)
	AssertEq(nil, err)
	AssertEq(jobStatusBefore.Name, downloader.DOWNLOADING)

	err = chrT.cacheHandler.performPostEvictionWork(&fileInfoData)

	ExpectEq(nil, err)
	jobStatusAfter := job.GetStatus()
	ExpectEq(jobStatusAfter.Name, downloader.INVALID)
	ExpectEq(false, IsFileExist(chrT.downloadPath))
}

func (chrT *cacheHandlerTest) Test_performPostEvictionWork_WhenLocalFileNotExist() {
	cacheHandle, err := chrT.cacheHandler.GetCacheHandle(chrT.object, chrT.bucket, 0)
	AssertEq(nil, err)
	AssertNe(nil, cacheHandle)
	AssertEq(true, IsFileExist(chrT.downloadPath))
	err = os.Remove(chrT.downloadPath)
	AssertEq(nil, err)
	fileInfo := chrT.cache.LookUp(chrT.fileInfoKeyName)
	fileInfoData := fileInfo.(data.FileInfo)
	job := chrT.jobManager.GetJob(chrT.object, chrT.bucket)
	jobStatusBefore := job.GetStatus()
	AssertEq(jobStatusBefore.Name, downloader.NOT_STARTED)
	jobStatusBefore, err = cacheHandle.fileDownloadJob.Download(context.Background(), int64(util.MiB), false)
	AssertEq(nil, err)
	AssertEq(jobStatusBefore.Name, downloader.DOWNLOADING)

	err = chrT.cacheHandler.performPostEvictionWork(&fileInfoData)

	ExpectNe(nil, err)
	ExpectTrue(strings.Contains(err.Error(), "while deleting file"))
}

func (chrT *cacheHandlerTest) Test_GetCacheHandle_WhenEntryAlreadyInCache() {
	cacheHandle, err := chrT.cacheHandler.GetCacheHandle(chrT.object, chrT.bucket, 0)

	ExpectEq(nil, err)
	ExpectEq(nil, cacheHandle.validateCacheHandle())
}

func (chrT *cacheHandlerTest) Test_GetCacheHandle_WhenEntryNotInCache() {
	minObject := chrT.getMinObject("object_1", "content of object_1")

	cacheHandle, err := chrT.cacheHandler.GetCacheHandle(minObject, chrT.bucket, 0)

	ExpectEq(nil, err)
	ExpectEq(nil, cacheHandle.validateCacheHandle())
}

func (chrT *cacheHandlerTest) Test_GetCacheHandle_WithEviction() {
	// Existing cacheHandle.
	cacheHandle1, err := chrT.cacheHandler.GetCacheHandle(chrT.object, chrT.bucket, 0)
	AssertEq(nil, err)
	AssertNe(nil, cacheHandle1)
	AssertEq(true, IsFileExist(chrT.downloadPath))
	// Content of size more than 20 leads to eviction of initial TestObjectName.
	// Here, content size is 21.
	minObject := chrT.getMinObject("object_1", "content of object_1 ...")

	cacheHandle2, err := chrT.cacheHandler.GetCacheHandle(minObject, chrT.bucket, 0)

	ExpectEq(nil, err)
	ExpectEq(nil, cacheHandle2.validateCacheHandle())
	jobStatus := cacheHandle1.fileDownloadJob.GetStatus()
	ExpectEq(downloader.INVALID, jobStatus.Name)
	ExpectEq(false, IsFileExist(chrT.downloadPath))
}

func (chrT *cacheHandlerTest) Test_GetCacheHandle_ConcurrentSameFile() {
	// Existing cacheHandle.
	cacheHandle1, err := chrT.cacheHandler.GetCacheHandle(chrT.object, chrT.bucket, 0)
	AssertEq(nil, err)
	AssertNe(nil, cacheHandle1)
	AssertEq(true, IsFileExist(chrT.downloadPath))

	wg := sync.WaitGroup{}

	getCacheHandleTestFun := func() {
		defer wg.Done()
		minObj := chrT.getMinObject("object_1", "content of object_1 ...")

		cacheHandle, err := chrT.cacheHandler.GetCacheHandle(minObj, chrT.bucket, 0)

		AssertEq(nil, err)
		AssertEq(nil, cacheHandle.validateCacheHandle())
	}

	// Start concurrent GetCacheHandle()
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go getCacheHandleTestFun()
	}
	wg.Wait()

	jobStatus := cacheHandle1.fileDownloadJob.GetStatus()
	ExpectEq(downloader.INVALID, jobStatus.Name)
	ExpectEq(false, IsFileExist(chrT.downloadPath))
}

func (chrT *cacheHandlerTest) Test_GetCacheHandle_ConcurrentDifferentFiles() {
	// Existing cacheHandle.
	cacheHandle1, err := chrT.cacheHandler.GetCacheHandle(chrT.object, chrT.bucket, 0)
	AssertEq(nil, err)
	AssertNe(nil, cacheHandle1)
	AssertEq(true, IsFileExist(chrT.downloadPath))
	wg := sync.WaitGroup{}

	getCacheHandleTestFun := func(index int) {
		defer wg.Done()
		objName := "object" + strconv.Itoa(index)
		objContent := "object content: " + strconv.Itoa(index)
		minObj := chrT.getMinObject(objName, objContent)

		cacheHandle, err := chrT.cacheHandler.GetCacheHandle(minObj, chrT.bucket, 0)

		AssertEq(nil, err)
		AssertEq(nil, cacheHandle.validateCacheHandle())
	}

	// Start concurrent GetCacheHandle()
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go getCacheHandleTestFun(i)
	}
	wg.Wait()

	jobStatus := cacheHandle1.fileDownloadJob.GetStatus()
	ExpectEq(downloader.INVALID, jobStatus.Name)
	ExpectEq(false, IsFileExist(chrT.downloadPath))
}
