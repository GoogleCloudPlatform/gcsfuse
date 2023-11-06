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

func (chrT *cacheHandlerTest) SetUp(*TestInfo) {
	locker.EnableInvariantsCheck()
	chrT.cacheLocation = path.Join(os.Getenv("HOME"), "cache/location")

	// Create bucket in fake storage.
	chrT.fakeStorage = storage.NewFakeStorage()
	storageHandle := chrT.fakeStorage.CreateStorageHandle()
	chrT.bucket = storageHandle.BucketHandle(storage.TestBucketName, "")

	// Create test object in the bucket.
	testObjectContent := make([]byte, TestObjectSize)
	_, err := rand.Read(testObjectContent)
	AssertEq(nil, err)
	chrT.object = chrT.getMinObject(TestObjectName, testObjectContent)

	// fileInfoCache with testFileInfoEntry
	chrT.cache = lru.NewCache(HandlerCacheMaxSize)
	chrT.fileInfoKeyName = chrT.addTestFileInfoEntryInCache(storage.TestBucketName, TestObjectName)

	// Job manager
	chrT.jobManager = downloader.NewJobManager(chrT.cache, util.DefaultFilePerm, chrT.cacheLocation, DefaultSequentialReadSizeMb)

	// Mocked cached handler object.
	chrT.cacheHandler = NewCacheHandler(chrT.cache, chrT.jobManager, chrT.cacheLocation, util.DefaultFilePerm)

	chrT.downloadPath = util.GetDownloadPath(chrT.cacheHandler.cacheLocation, util.GetObjectPath(chrT.bucket.Name(), chrT.object.Name))
}

func (chrT *cacheHandlerTest) TearDown() {
	chrT.fakeStorage.ShutDown()
	operations.RemoveDir(chrT.cacheLocation)
}

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

func (chrT *cacheHandlerTest) isEntryInFileInfoCache(objectName string, bucketName string) bool {
	fileInfoKey := data.FileInfoKey{
		BucketName: bucketName,
		ObjectName: objectName,
	}

	fileInfoKeyName, err := fileInfoKey.Key()
	AssertEq(nil, err)

	fileInfo := chrT.cache.LookUp(fileInfoKeyName)
	return fileInfo != nil
}

func (chrT *cacheHandlerTest) getCacheHandleForSetupEntryInCache() *CacheHandle {
	cacheHandle, err := chrT.cacheHandler.GetCacheHandle(chrT.object, chrT.bucket, 0)
	AssertEq(nil, err)
	AssertNe(nil, cacheHandle)
	AssertEq(true, doesFileExist(chrT.downloadPath))

	return cacheHandle
}

func (chrT *cacheHandlerTest) getMinObject(objName string, objContent []byte) *gcs.MinObject {
	ctx := context.Background()
	objects := map[string][]byte{objName: objContent}
	err := storageutil.CreateObjects(ctx, chrT.bucket, objects)
	AssertEq(nil, err)

	gcsObj, err := chrT.bucket.StatObject(ctx, &gcs.StatObjectRequest{Name: objName,
		ForceFetchFromGcs: true})
	AssertEq(nil, err)
	minObject := storageutil.ConvertObjToMinObject(gcsObj)
	return &minObject
}

// doesFileExist returns true if the file exists and false otherwise.
// If an error occurs, the function panics.
func doesFileExist(filePath string) bool {
	_, err := os.Stat(filePath)

	if err == nil {
		return true
	}

	if os.IsNotExist(err) {
		return false
	}

	AssertEq(nil, err)
	return false
}

func (chrT *cacheHandlerTest) Test_createLocalFileReadHandle_OnlyForRead() {
	readFileHandle, err := chrT.cacheHandler.createLocalFileReadHandle(chrT.object.Name, chrT.bucket.Name())

	ExpectEq(nil, err)
	_, err = readFileHandle.Write([]byte("test"))
	ExpectNe(nil, err)
	ExpectTrue(strings.Contains(err.Error(), "bad file descriptor"))
}

func (chrT *cacheHandlerTest) Test_cleanUpEvictedFile() {
	// Existing cacheHandle.
	cacheHandle := chrT.getCacheHandleForSetupEntryInCache()
	AssertEq(nil, cacheHandle.validateCacheHandle())
	fileInfo := chrT.cache.LookUp(chrT.fileInfoKeyName)
	fileInfoData := fileInfo.(data.FileInfo)
	jobStatusBefore := cacheHandle.fileDownloadJob.GetStatus()
	AssertEq(jobStatusBefore.Name, downloader.NOT_STARTED)
	jobStatusBefore, err := cacheHandle.fileDownloadJob.Download(context.Background(), int64(util.MiB), false)
	AssertEq(nil, err)
	AssertEq(jobStatusBefore.Name, downloader.DOWNLOADING)

	err = chrT.cacheHandler.cleanUpEvictedFile(&fileInfoData)

	ExpectEq(nil, err)
	jobStatusAfter := cacheHandle.fileDownloadJob.GetStatus()
	ExpectEq(jobStatusAfter.Name, downloader.INVALID)
	ExpectEq(false, doesFileExist(chrT.downloadPath))
}

func (chrT *cacheHandlerTest) Test_cleanUpEvictedFile_WhenLocalFileNotExist() {
	cacheHandle := chrT.getCacheHandleForSetupEntryInCache()
	AssertEq(nil, cacheHandle.validateCacheHandle())
	err := os.Remove(chrT.downloadPath)
	AssertEq(nil, err)
	fileInfo := chrT.cache.LookUp(chrT.fileInfoKeyName)
	fileInfoData := fileInfo.(data.FileInfo)
	jobStatusBefore := cacheHandle.fileDownloadJob.GetStatus()
	AssertEq(jobStatusBefore.Name, downloader.NOT_STARTED)
	jobStatusBefore, err = cacheHandle.fileDownloadJob.Download(context.Background(), int64(util.MiB), false)
	AssertEq(nil, err)
	AssertEq(jobStatusBefore.Name, downloader.DOWNLOADING)

	err = chrT.cacheHandler.cleanUpEvictedFile(&fileInfoData)

	ExpectEq(nil, err)
}

func (chrT *cacheHandlerTest) Test_addFileInfoEntryInCache_IfAlready() {
	// Existing cacheHandle.
	cacheHandle1 := chrT.getCacheHandleForSetupEntryInCache()
	AssertEq(nil, cacheHandle1.validateCacheHandle())

	err := chrT.cacheHandler.addFileInfoEntryInCache(chrT.object, chrT.bucket)

	ExpectEq(nil, err)
	ExpectTrue(chrT.isEntryInFileInfoCache(chrT.object.Name, chrT.bucket.Name()))
}

func (chrT *cacheHandlerTest) Test_addFileInfoEntryInCache_IfNotAlready() {
	// Existing cacheHandle.
	cacheHandle1 := chrT.getCacheHandleForSetupEntryInCache()
	AssertEq(nil, cacheHandle1.validateCacheHandle())
	// Content of size more than 20 leads to eviction of initial TestObjectName.
	// Here, content size is 21.
	minObject := chrT.getMinObject("object_1", []byte("content of object_1 ..."))

	// Insertion will happen and that leads to eviction.
	err := chrT.cacheHandler.addFileInfoEntryInCache(minObject, chrT.bucket)

	ExpectEq(nil, err)
	ExpectTrue(chrT.isEntryInFileInfoCache(minObject.Name, chrT.bucket.Name()))
	jobStatus := cacheHandle1.fileDownloadJob.GetStatus()
	ExpectEq(downloader.INVALID, jobStatus.Name)
	ExpectEq(false, doesFileExist(chrT.downloadPath))
}

func (chrT *cacheHandlerTest) Test_GetCacheHandle_WhenEntryAlreadyInCache() {
	cacheHandle, err := chrT.cacheHandler.GetCacheHandle(chrT.object, chrT.bucket, 0)

	ExpectEq(nil, err)
	ExpectEq(nil, cacheHandle.validateCacheHandle())
}

func (chrT *cacheHandlerTest) Test_GetCacheHandle_WhenEntryNotInCache() {
	minObject := chrT.getMinObject("object_1", []byte("content of object_1"))

	cacheHandle, err := chrT.cacheHandler.GetCacheHandle(minObject, chrT.bucket, 0)

	ExpectEq(nil, err)
	ExpectEq(nil, cacheHandle.validateCacheHandle())
}

func (chrT *cacheHandlerTest) Test_GetCacheHandle_WithEviction() {
	// Existing cacheHandle.
	cacheHandle1 := chrT.getCacheHandleForSetupEntryInCache()
	AssertEq(nil, cacheHandle1.validateCacheHandle())
	// Content of size more than 20 leads to eviction of initial TestObjectName.
	// Here, content size is 21.
	minObject := chrT.getMinObject("object_1", []byte("content of object_1 ..."))

	cacheHandle2, err := chrT.cacheHandler.GetCacheHandle(minObject, chrT.bucket, 0)

	ExpectEq(nil, err)
	ExpectEq(nil, cacheHandle2.validateCacheHandle())
	jobStatus := cacheHandle1.fileDownloadJob.GetStatus()
	ExpectEq(downloader.INVALID, jobStatus.Name)
	ExpectEq(false, doesFileExist(chrT.downloadPath))
}

func (chrT *cacheHandlerTest) Test_GetCacheHandle_ConcurrentSameFile() {
	// Existing cacheHandle.
	cacheHandle1 := chrT.getCacheHandleForSetupEntryInCache()
	AssertEq(nil, cacheHandle1.validateCacheHandle())

	wg := sync.WaitGroup{}

	getCacheHandleTestFun := func() {
		defer wg.Done()
		minObj := chrT.getMinObject("object_1", []byte("content of object_1 ..."))

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
	ExpectEq(false, doesFileExist(chrT.downloadPath))
}

func (chrT *cacheHandlerTest) Test_GetCacheHandle_ConcurrentDifferentFiles() {
	// Existing cacheHandle.
	cacheHandle1 := chrT.getCacheHandleForSetupEntryInCache()
	AssertEq(nil, cacheHandle1.validateCacheHandle())
	wg := sync.WaitGroup{}

	getCacheHandleTestFun := func(index int) {
		defer wg.Done()
		objName := "object" + strconv.Itoa(index)
		objContent := "object content: " + strconv.Itoa(index)
		minObj := chrT.getMinObject(objName, []byte(objContent))

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
	ExpectEq(false, doesFileExist(chrT.downloadPath))
}

func (chrT *cacheHandlerTest) Test_InvalidateCache_WhenEntryAlreadyInCache() {
	cacheHandle, err := chrT.cacheHandler.GetCacheHandle(chrT.object, chrT.bucket, 0)
	ExpectEq(nil, err)

	err = chrT.cacheHandler.InvalidateCache(chrT.object, chrT.bucket)

	ExpectEq(nil, err)
	jobStatus := cacheHandle.fileDownloadJob.GetStatus()
	ExpectEq(downloader.INVALID, jobStatus.Name)
	ExpectEq(false, doesFileExist(chrT.downloadPath))
}

func (chrT *cacheHandlerTest) Test_InvalidateCache_WhenEntryNotInCache() {
	minObject := chrT.getMinObject("object_1", []byte("content of object_1"))

	err := chrT.cacheHandler.InvalidateCache(minObject, chrT.bucket)

	ExpectEq(nil, err)
}

func (chrT *cacheHandlerTest) Test_InvalidateCache_ConcurrentSameFile() {
	// Existing cacheHandle.
	cacheHandle1 := chrT.getCacheHandleForSetupEntryInCache()
	AssertEq(nil, cacheHandle1.validateCacheHandle())

	wg := sync.WaitGroup{}

	InvalidateCacheTestFun := func() {
		defer wg.Done()
		minObj := chrT.getMinObject("object_1", []byte("content of object_1 ..."))

		err := chrT.cacheHandler.InvalidateCache(minObj, chrT.bucket)

		AssertEq(nil, err)
	}

	// Start concurrent GetCacheHandle()
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go InvalidateCacheTestFun()
	}
	wg.Wait()
}

func (chrT *cacheHandlerTest) Test_InvalidateCache_ConcurrentDifferentFiles() {
	// Existing cacheHandle.
	cacheHandle1 := chrT.getCacheHandleForSetupEntryInCache()
	AssertEq(nil, cacheHandle1.validateCacheHandle())
	wg := sync.WaitGroup{}

	InvalidateCacheTestFun := func(index int) {
		defer wg.Done()
		objName := "object" + strconv.Itoa(index)
		objContent := "object content: " + strconv.Itoa(index)
		minObj := chrT.getMinObject(objName, []byte(objContent))

		err := chrT.cacheHandler.InvalidateCache(minObj, chrT.bucket)

		AssertEq(nil, err)
	}

	// Start concurrent GetCacheHandle()
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go InvalidateCacheTestFun(i)
	}
	wg.Wait()
}

func (chrT *cacheHandlerTest) Test_InvalidateCacheAndGetHandle_Concurrent() {
	// Existing cacheHandle.
	cacheHandle1 := chrT.getCacheHandleForSetupEntryInCache()
	AssertEq(nil, cacheHandle1.validateCacheHandle())
	wg := sync.WaitGroup{}

	invalidateCacheTestFun := func(index int) {
		defer wg.Done()
		objName := "object" + strconv.Itoa(index)
		objContent := "object content: " + strconv.Itoa(index)
		minObj := chrT.getMinObject(objName, []byte(objContent))

		err := chrT.cacheHandler.InvalidateCache(minObj, chrT.bucket)

		AssertEq(nil, err)
	}

	getCacheHandleTestFun := func(index int) {
		defer wg.Done()
		objName := "object" + strconv.Itoa(index)
		objContent := "object content: " + strconv.Itoa(index)
		minObj := chrT.getMinObject(objName, []byte(objContent))

		cacheHandle, err := chrT.cacheHandler.GetCacheHandle(minObj, chrT.bucket, 0)

		AssertEq(nil, err)
		AssertEq(nil, cacheHandle.validateCacheHandle())
	}

	// Start concurrent GetCacheHandle()
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go invalidateCacheTestFun(i)
		wg.Add(1)
		go getCacheHandleTestFun(i)
	}
	wg.Wait()

	jobStatus := cacheHandle1.fileDownloadJob.GetStatus()
	ExpectEq(downloader.INVALID, jobStatus.Name)
	ExpectEq(false, doesFileExist(chrT.downloadPath))
}
