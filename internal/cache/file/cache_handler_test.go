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
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

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

const HandlerCacheMaxSize = TestObjectSize + ObjectSizeToCauseEviction
const ObjectSizeToCauseEviction = 20

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
	cacheDir        string
}

func init() { RegisterTestSuite(&cacheHandlerTest{}) }

func (chrT *cacheHandlerTest) SetUp(*TestInfo) {
	locker.EnableInvariantsCheck()
	chrT.cacheDir = path.Join(os.Getenv("HOME"), "cache/dir")

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

	// Job manager
	chrT.jobManager = downloader.NewJobManager(chrT.cache, util.DefaultFilePerm, util.DefaultDirPerm, chrT.cacheDir, DefaultSequentialReadSizeMb)

	// Mocked cached handler object.
	chrT.cacheHandler = NewCacheHandler(chrT.cache, chrT.jobManager, chrT.cacheDir, util.DefaultFilePerm, util.DefaultDirPerm)

	// Follow consistency, local-cache file, entry in fileInfo cache and job should exist initially.
	chrT.fileInfoKeyName = chrT.addTestFileInfoEntryInCache(storage.TestBucketName, TestObjectName)
	chrT.downloadPath = util.GetDownloadPath(chrT.cacheHandler.cacheDir, util.GetObjectPath(chrT.bucket.Name(), chrT.object.Name))
	_, err = util.CreateFile(data.FileSpec{Path: chrT.downloadPath, FilePerm: util.DefaultFilePerm, DirPerm: util.DefaultDirPerm}, os.O_RDONLY)
	AssertEq(nil, err)
	_ = chrT.getDownloadJobForTestObject()
}

func (chrT *cacheHandlerTest) TearDown() {
	chrT.fakeStorage.ShutDown()
	operations.RemoveDir(chrT.cacheDir)
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

func (chrT *cacheHandlerTest) getDownloadJobForTestObject() *downloader.Job {
	job := chrT.jobManager.CreateJobIfNotExists(chrT.object, chrT.bucket)
	AssertNe(nil, job)
	return job
}

func (chrT *cacheHandlerTest) getMinObject(objName string, objContent []byte) *gcs.MinObject {
	ctx := context.Background()
	objects := map[string][]byte{objName: objContent}
	err := storageutil.CreateObjects(ctx, chrT.bucket, objects)
	AssertEq(nil, err)

	minObject, _, err := chrT.bucket.StatObject(ctx, &gcs.StatObjectRequest{Name: objName,
		ForceFetchFromGcs: true})
	AssertEq(nil, err)
	AssertNe(nil, minObject)
	return minObject
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
	fileDownloadJob := chrT.getDownloadJobForTestObject()
	fileInfo := chrT.cache.LookUp(chrT.fileInfoKeyName)
	fileInfoData := fileInfo.(data.FileInfo)
	jobStatusBefore := fileDownloadJob.GetStatus()
	AssertEq(jobStatusBefore.Name, downloader.NotStarted)
	jobStatusBefore, err := fileDownloadJob.Download(context.Background(), int64(util.MiB), false)
	AssertEq(nil, err)
	AssertEq(jobStatusBefore.Name, downloader.Downloading)

	err = chrT.cacheHandler.cleanUpEvictedFile(&fileInfoData)

	ExpectEq(nil, err)
	jobStatusAfter := fileDownloadJob.GetStatus()
	ExpectEq(jobStatusAfter.Name, downloader.Invalid)
	ExpectFalse(doesFileExist(chrT.downloadPath))
	// Job should be removed from job manager
	ExpectEq(nil, chrT.jobManager.GetJob(chrT.object.Name, chrT.bucket.Name()))
}

func (chrT *cacheHandlerTest) Test_cleanUpEvictedFile_WhenLocalFileNotExist() {
	fileDownloadJob := chrT.getDownloadJobForTestObject()
	fileInfo := chrT.cache.LookUp(chrT.fileInfoKeyName)
	fileInfoData := fileInfo.(data.FileInfo)
	jobStatusBefore := fileDownloadJob.GetStatus()
	AssertEq(jobStatusBefore.Name, downloader.NotStarted)
	jobStatusBefore, err := fileDownloadJob.Download(context.Background(), int64(util.MiB), false)
	AssertEq(nil, err)
	AssertEq(jobStatusBefore.Name, downloader.Downloading)
	err = os.Remove(chrT.downloadPath)
	AssertEq(nil, err)

	err = chrT.cacheHandler.cleanUpEvictedFile(&fileInfoData)

	ExpectEq(nil, err)
	jobStatusAfter := fileDownloadJob.GetStatus()
	ExpectEq(jobStatusAfter.Name, downloader.Invalid)
	ExpectFalse(doesFileExist(chrT.downloadPath))
	// Job should be removed from job manager
	ExpectEq(nil, chrT.jobManager.GetJob(chrT.object.Name, chrT.bucket.Name()))
}

func (chrT *cacheHandlerTest) Test_addFileInfoEntryAndCreateDownloadJob_IfAlready() {
	existingJob := chrT.getDownloadJobForTestObject()

	err := chrT.cacheHandler.addFileInfoEntryAndCreateDownloadJob(chrT.object, chrT.bucket)

	ExpectEq(nil, err)
	ExpectTrue(chrT.isEntryInFileInfoCache(chrT.object.Name, chrT.bucket.Name()))
	// File download job should also be same
	actualJob := chrT.jobManager.GetJob(chrT.object.Name, chrT.bucket.Name())
	ExpectEq(existingJob, actualJob)
}

func (chrT *cacheHandlerTest) Test_addFileInfoEntryAndCreateDownloadJob_GenerationChanged() {
	existingJob := chrT.getDownloadJobForTestObject()
	chrT.object.Generation = chrT.object.Generation + 1

	err := chrT.cacheHandler.addFileInfoEntryAndCreateDownloadJob(chrT.object, chrT.bucket)

	ExpectEq(nil, err)
	ExpectTrue(chrT.isEntryInFileInfoCache(chrT.object.Name, chrT.bucket.Name()))
	// File download job should be new as the file info and job should be cleaned
	// up.
	actualJob := chrT.jobManager.GetJob(chrT.object.Name, chrT.bucket.Name())
	ExpectNe(existingJob, actualJob)
}

func (chrT *cacheHandlerTest) Test_addFileInfoEntryAndCreateDownloadJob_IfNotAlready() {
	oldJob := chrT.getDownloadJobForTestObject()
	// Content of size more than 20 leads to eviction of initial TestObjectName.
	// Here, content size is 21.
	minObject := chrT.getMinObject("object_1", []byte("content of object_1 ..."))
	// There should be no file download job corresponding to minObject
	existingJob := chrT.jobManager.GetJob(minObject.Name, chrT.bucket.Name())
	AssertEq(nil, existingJob)

	// Insertion will happen and that leads to eviction.
	err := chrT.cacheHandler.addFileInfoEntryAndCreateDownloadJob(minObject, chrT.bucket)

	ExpectEq(nil, err)
	ExpectTrue(chrT.isEntryInFileInfoCache(minObject.Name, chrT.bucket.Name()))
	jobStatus := oldJob.GetStatus()
	ExpectEq(downloader.Invalid, jobStatus.Name)
	ExpectEq(false, doesFileExist(chrT.downloadPath))
	// Job should be added for minObject
	minObjectJob := chrT.jobManager.GetJob(minObject.Name, chrT.bucket.Name())
	ExpectNe(nil, minObjectJob)
	ExpectEq(downloader.NotStarted, minObjectJob.GetStatus().Name)
}

func (chrT *cacheHandlerTest) Test_addFileInfoEntryAndCreateDownloadJob_IfLocalFileGetsDeleted() {
	// Delete the local cache file.
	err := os.Remove(chrT.downloadPath)
	AssertEq(nil, err)

	// There is a fileInfoEntry in the fileInfoCache but the corresponding local file doesn't exist.
	// Hence, this will return error containing util.FileNotPresentInCacheErrMsg.
	err = chrT.cacheHandler.addFileInfoEntryAndCreateDownloadJob(chrT.object, chrT.bucket)

	AssertNe(nil, err)
	ExpectTrue(strings.Contains(err.Error(), util.FileNotPresentInCacheErrMsg))
}

func (chrT *cacheHandlerTest) Test_addFileInfoEntryAndCreateDownloadJob_WhenJobHasCompleted() {
	existingJob := chrT.getDownloadJobForTestObject()
	// Make the job completed, so it's removed from job manager.
	jobStatus, err := existingJob.Download(context.Background(), int64(chrT.object.Size), true)
	AssertEq(nil, err)
	AssertEq(jobStatus.Offset, chrT.object.Size)
	// Give time for execution of callback to remove from job manager
	time.Sleep(time.Second)
	actualJob := chrT.jobManager.GetJob(chrT.object.Name, chrT.bucket.Name())
	ExpectEq(nil, actualJob)

	err = chrT.cacheHandler.addFileInfoEntryAndCreateDownloadJob(chrT.object, chrT.bucket)

	ExpectEq(nil, err)
	ExpectTrue(chrT.isEntryInFileInfoCache(chrT.object.Name, chrT.bucket.Name()))
	// No new job should be added to job manager
	actualJob = chrT.jobManager.GetJob(chrT.object.Name, chrT.bucket.Name())
	ExpectEq(nil, actualJob)
}

func (chrT *cacheHandlerTest) Test_addFileInfoEntryAndCreateDownloadJob_WhenJobIsInvalidatedAndRemoved() {
	chrT.jobManager.InvalidateAndRemoveJob(chrT.object.Name, chrT.bucket.Name())
	existingJob := chrT.jobManager.GetJob(chrT.object.Name, chrT.bucket.Name())
	ExpectEq(nil, existingJob)

	// Because the job has been removed and file info entry is still present, new
	// file info entry and job should be created.
	err := chrT.cacheHandler.addFileInfoEntryAndCreateDownloadJob(chrT.object, chrT.bucket)

	ExpectEq(nil, err)
	ExpectTrue(chrT.isEntryInFileInfoCache(chrT.object.Name, chrT.bucket.Name()))
	// New job should be added to job manager
	actualJob := chrT.jobManager.GetJob(chrT.object.Name, chrT.bucket.Name())
	ExpectNe(nil, actualJob)
	ExpectEq(downloader.NotStarted, actualJob.GetStatus().Name)
}

func (chrT *cacheHandlerTest) Test_addFileInfoEntryAndCreateDownloadJob_WhenJobHasFailed() {
	existingJob := chrT.getDownloadJobForTestObject()
	// Hack to fail the async job
	correctSize := chrT.object.Size
	chrT.object.Size = 2
	jobStatus, err := existingJob.Download(context.Background(), 1, true)
	AssertEq(nil, err)
	AssertEq(downloader.Failed, jobStatus.Name)
	chrT.object.Size = correctSize

	// Because the job has been failed and file info entry is still present with
	// size less than the object's size (because the async job failed), new job
	// should be created
	err = chrT.cacheHandler.addFileInfoEntryAndCreateDownloadJob(chrT.object, chrT.bucket)

	ExpectEq(nil, err)
	ExpectTrue(chrT.isEntryInFileInfoCache(chrT.object.Name, chrT.bucket.Name()))
	// New job should be added to job manager
	actualJob := chrT.jobManager.GetJob(chrT.object.Name, chrT.bucket.Name())
	ExpectNe(nil, actualJob)
	ExpectEq(downloader.NotStarted, actualJob.GetStatus().Name)
}

func (chrT *cacheHandlerTest) Test_GetCacheHandle_WhenCacheHasDifferentGeneration() {
	existingJob := chrT.getDownloadJobForTestObject()
	AssertNe(nil, existingJob)
	AssertEq(downloader.NotStarted, existingJob.GetStatus().Name)
	// Change the version of the object, but cache still keeps old generation
	chrT.object.Generation = chrT.object.Generation + 1

	newCacheHandle, err := chrT.cacheHandler.GetCacheHandle(chrT.object, chrT.bucket, false, 0)

	ExpectEq(nil, err)
	ExpectEq(nil, newCacheHandle.validateCacheHandle())
	jobStatusOfOldJob := existingJob.GetStatus()
	ExpectEq(jobStatusOfOldJob.Name, downloader.Invalid)
	jobStatusOfNewHandle := newCacheHandle.fileDownloadJob.GetStatus()
	ExpectEq(jobStatusOfNewHandle.Name, downloader.NotStarted)
}

func (chrT *cacheHandlerTest) Test_GetCacheHandle_WhenAsyncDownloadJobHasFailed() {
	existingJob := chrT.getDownloadJobForTestObject()
	// Hack to fail the async job
	correctSize := chrT.object.Size
	chrT.object.Size = 2
	jobStatus, err := existingJob.Download(context.Background(), 1, true)
	AssertEq(nil, err)
	AssertEq(downloader.Failed, jobStatus.Name)
	chrT.object.Size = correctSize

	newCacheHandle, err := chrT.cacheHandler.GetCacheHandle(chrT.object, chrT.bucket, false, 0)

	// New job should be created because the earlier job has failed.
	ExpectEq(nil, err)
	ExpectEq(nil, newCacheHandle.validateCacheHandle())
	ExpectTrue(chrT.isEntryInFileInfoCache(chrT.object.Name, chrT.bucket.Name()))
	jobStatusOfNewHandle := newCacheHandle.fileDownloadJob.GetStatus()
	ExpectEq(downloader.NotStarted, jobStatusOfNewHandle.Name)
}

func (chrT *cacheHandlerTest) Test_GetCacheHandle_WhenFileInfoAndJobAreAlreadyPresent() {
	// File info and download job are already present for test object.
	existingJob := chrT.getDownloadJobForTestObject()

	cacheHandle, err := chrT.cacheHandler.GetCacheHandle(chrT.object, chrT.bucket, false, 0)

	ExpectEq(nil, err)
	ExpectEq(nil, cacheHandle.validateCacheHandle())
	// Job and file info are still present
	ExpectTrue(chrT.isEntryInFileInfoCache(chrT.object.Name, chrT.bucket.Name()))
	ExpectEq(existingJob, cacheHandle.fileDownloadJob)
	jobStatusOfNewHandle := cacheHandle.fileDownloadJob.GetStatus()
	ExpectEq(downloader.NotStarted, jobStatusOfNewHandle.Name)
}

func (chrT *cacheHandlerTest) Test_GetCacheHandle_WhenFileInfoAndJobAreNotPresent() {
	minObject := chrT.getMinObject("object_1", []byte("content of object_1"))

	cacheHandle, err := chrT.cacheHandler.GetCacheHandle(minObject, chrT.bucket, false, 0)

	ExpectEq(nil, err)
	ExpectEq(nil, cacheHandle.validateCacheHandle())
	// New Job and file info are created.
	ExpectTrue(chrT.isEntryInFileInfoCache(minObject.Name, chrT.bucket.Name()))
	jobStatusOfNewHandle := cacheHandle.fileDownloadJob.GetStatus()
	ExpectEq(downloader.NotStarted, jobStatusOfNewHandle.Name)
}

func (chrT *cacheHandlerTest) Test_GetCacheHandle_WithEviction() {
	// Start the existing job
	existingJob := chrT.getDownloadJobForTestObject()
	_, err := existingJob.Download(context.Background(), 1, false)
	AssertEq(nil, err)
	// Content of size more than 20 leads to eviction of initial TestObjectName.
	// Here, content size is 21.
	minObject := chrT.getMinObject("object_1", []byte("content of object_1 ..."))

	cacheHandle2, err := chrT.cacheHandler.GetCacheHandle(minObject, chrT.bucket, false, 0)

	ExpectEq(nil, err)
	ExpectEq(nil, cacheHandle2.validateCacheHandle())
	jobStatus := existingJob.GetStatus()
	ExpectEq(downloader.Invalid, jobStatus.Name)
	ExpectFalse(doesFileExist(chrT.downloadPath))
}

func (chrT *cacheHandlerTest) Test_GetCacheHandle_IfLocalFileGetsDeleted() {
	// Delete the local cache file.
	err := os.Remove(chrT.downloadPath)
	AssertEq(nil, err)
	existingJob := chrT.getDownloadJobForTestObject()

	cacheHandle, err := chrT.cacheHandler.GetCacheHandle(chrT.object, chrT.bucket, false, 0)

	AssertNe(nil, err)
	ExpectTrue(strings.Contains(err.Error(), util.FileNotPresentInCacheErrMsg))
	AssertEq(nil, cacheHandle)
	// Check file info and download job are not removed
	ExpectTrue(chrT.isEntryInFileInfoCache(chrT.object.Name, chrT.bucket.Name()))
	actualJob := chrT.jobManager.GetJob(chrT.object.Name, chrT.bucket.Name())
	ExpectEq(existingJob, actualJob)
	ExpectEq(downloader.NotStarted, existingJob.GetStatus().Name)

}

func (chrT *cacheHandlerTest) Test_GetCacheHandle_CacheForRangeRead() {
	minObject1 := chrT.getMinObject("object_1", []byte("content of object_1 ..."))
	cacheHandle1, err1 := chrT.cacheHandler.GetCacheHandle(minObject1, chrT.bucket, false, 0)
	minObject2 := chrT.getMinObject("object_2", []byte("content of object_2 ..."))
	cacheHandle2, err2 := chrT.cacheHandler.GetCacheHandle(minObject2, chrT.bucket, false, 5)
	minObject3 := chrT.getMinObject("object_3", []byte("content of object_3 ..."))
	cacheHandle3, err3 := chrT.cacheHandler.GetCacheHandle(minObject3, chrT.bucket, true, 0)
	minObject4 := chrT.getMinObject("object_4", []byte("content of object_4 ..."))
	cacheHandle4, err4 := chrT.cacheHandler.GetCacheHandle(minObject4, chrT.bucket, true, 5)

	ExpectEq(nil, err1)
	ExpectEq(nil, cacheHandle1.validateCacheHandle())
	ExpectNe(nil, err2)
	ExpectEq(nil, cacheHandle2)
	ExpectTrue(strings.Contains(err2.Error(), util.CacheHandleNotRequiredForRandomReadErrMsg))
	ExpectEq(nil, err3)
	ExpectEq(nil, cacheHandle3.validateCacheHandle())
	ExpectEq(nil, err4)
	ExpectEq(nil, cacheHandle4.validateCacheHandle())
}

func (chrT *cacheHandlerTest) Test_GetCacheHandle_ConcurrentSameFile() {
	// Check async job and file info cache not preset for object_1
	testObjectName := "object_1"
	existingJob := chrT.jobManager.GetJob(testObjectName, chrT.bucket.Name())
	AssertEq(nil, existingJob)
	wg := sync.WaitGroup{}
	getCacheHandleTestFun := func() {
		defer wg.Done()
		minObj := chrT.getMinObject(testObjectName, []byte("content of object_1 ..."))

		var err error
		cacheHandle, err := chrT.cacheHandler.GetCacheHandle(minObj, chrT.bucket, false, 0)

		AssertEq(nil, err)
		AssertEq(nil, cacheHandle.validateCacheHandle())
	}

	// Start concurrent GetCacheHandle()
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go getCacheHandleTestFun()
	}
	wg.Wait()

	// Job should be added now
	actualJob := chrT.jobManager.GetJob(testObjectName, chrT.bucket.Name())
	jobStatus := actualJob.GetStatus()
	ExpectEq(downloader.NotStarted, jobStatus.Name)
	ExpectTrue(doesFileExist(util.GetDownloadPath(chrT.cacheDir, util.GetObjectPath(chrT.bucket.Name(), testObjectName))))
}

func (chrT *cacheHandlerTest) Test_GetCacheHandle_ConcurrentDifferentFiles() {
	existingJob := chrT.getDownloadJobForTestObject()
	AssertEq(downloader.NotStarted, existingJob.GetStatus().Name)
	wg := sync.WaitGroup{}

	getCacheHandleTestFun := func(index int) {
		defer wg.Done()
		objName := "object" + strconv.Itoa(index)
		objContent := "object content: content#" + strconv.Itoa(index)
		minObj := chrT.getMinObject(objName, []byte(objContent))

		cacheHandle, err := chrT.cacheHandler.GetCacheHandle(minObj, chrT.bucket, false, 0)

		AssertEq(nil, err)
		AssertEq(nil, cacheHandle.validateCacheHandle())
	}

	// Start concurrent GetCacheHandle()
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go getCacheHandleTestFun(i)
	}
	wg.Wait()

	// Existing job for default chrT object should be invalidated.
	ExpectNe(nil, existingJob)
	ExpectEq(downloader.Invalid, existingJob.GetStatus().Name)
	ExpectEq(false, doesFileExist(chrT.downloadPath))
	// File info should also be removed.
	ExpectFalse(chrT.isEntryInFileInfoCache(chrT.object.Name, chrT.bucket.Name()))
}

func (chrT *cacheHandlerTest) Test_InvalidateCache_WhenAlreadyInCache() {
	existingJob := chrT.getDownloadJobForTestObject()
	AssertEq(downloader.NotStarted, existingJob.GetStatus().Name)
	ExpectTrue(chrT.isEntryInFileInfoCache(chrT.object.Name, chrT.bucket.Name()))

	err := chrT.cacheHandler.InvalidateCache(chrT.object.Name, chrT.bucket.Name())

	ExpectEq(nil, err)
	// Existing job for default chrT object should be invalidated.
	ExpectNe(nil, existingJob)
	ExpectEq(downloader.Invalid, existingJob.GetStatus().Name)
	ExpectEq(false, doesFileExist(chrT.downloadPath))
	// File info should also be removed.
	ExpectFalse(chrT.isEntryInFileInfoCache(chrT.object.Name, chrT.bucket.Name()))
}

func (chrT *cacheHandlerTest) Test_InvalidateCache_WhenEntryNotInCache() {
	minObject := chrT.getMinObject("object_1", []byte("content of object_1"))
	ExpectFalse(chrT.isEntryInFileInfoCache(minObject.Name, chrT.bucket.Name()))
	ExpectEq(nil, chrT.jobManager.GetJob(minObject.Name, chrT.bucket.Name()))

	err := chrT.cacheHandler.InvalidateCache(minObject.Name, chrT.bucket.Name())

	ExpectEq(nil, err)
	ExpectFalse(chrT.isEntryInFileInfoCache(minObject.Name, chrT.bucket.Name()))
	ExpectEq(nil, chrT.jobManager.GetJob(minObject.Name, chrT.bucket.Name()))
}

func (chrT *cacheHandlerTest) Test_InvalidateCache_Truncates() {
	objectContent := []byte("content of object_1")
	minObject := chrT.getMinObject("object_1", objectContent)
	cacheHandle, err := chrT.cacheHandler.GetCacheHandle(minObject, chrT.bucket, false, 0)
	AssertEq(nil, err)
	buf := make([]byte, 3)
	ctx := context.Background()
	// Read to populate cache
	_, cacheHit, err := cacheHandle.Read(ctx, chrT.bucket, minObject, 0, buf)
	AssertEq(nil, err)
	ExpectEq(false, cacheHit)
	AssertEq(string(objectContent[:3]), string(buf))
	AssertEq(nil, cacheHandle.Close())
	// Open cache file before invalidation
	objectPath := util.GetObjectPath(chrT.bucket.Name(), minObject.Name)
	downloadPath := util.GetDownloadPath(chrT.cacheDir, objectPath)
	file, err := os.OpenFile(downloadPath, os.O_RDONLY, 0600)
	AssertEq(nil, err)
	_, err = file.Read(buf)
	AssertEq(nil, err)
	AssertEq(string(objectContent[:3]), string(buf))

	err = chrT.cacheHandler.InvalidateCache(minObject.Name, chrT.bucket.Name())

	AssertEq(nil, err)
	// Reading from the open file handle should fail as the file is truncated.
	_, err = file.Read(buf)
	AssertNe(nil, err)
	AssertEq(io.EOF, err)
}

func (chrT *cacheHandlerTest) Test_InvalidateCache_ConcurrentSameFile() {
	existingJob := chrT.getDownloadJobForTestObject()
	AssertEq(downloader.NotStarted, existingJob.GetStatus().Name)
	ExpectTrue(chrT.isEntryInFileInfoCache(chrT.object.Name, chrT.bucket.Name()))
	wg := sync.WaitGroup{}
	InvalidateCacheTestFun := func() {
		defer wg.Done()

		err := chrT.cacheHandler.InvalidateCache(chrT.object.Name, chrT.bucket.Name())

		AssertEq(nil, err)
		ExpectNe(nil, existingJob)
		ExpectEq(downloader.Invalid, existingJob.GetStatus().Name)
		ExpectEq(false, doesFileExist(chrT.downloadPath))
		// File info should also be removed.
		ExpectFalse(chrT.isEntryInFileInfoCache(chrT.object.Name, chrT.bucket.Name()))
	}

	// Start concurrent GetCacheHandle()
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go InvalidateCacheTestFun()
	}
	wg.Wait()
}

func (chrT *cacheHandlerTest) Test_InvalidateCache_ConcurrentDifferentFiles() {
	wg := sync.WaitGroup{}

	InvalidateCacheTestFun := func(index int) {
		defer wg.Done()
		objName := "object" + strconv.Itoa(index)
		objContent := "object content: content#" + strconv.Itoa(index)
		minObj := chrT.getMinObject(objName, []byte(objContent))

		err := chrT.cacheHandler.InvalidateCache(minObj.Name, chrT.bucket.Name())

		AssertEq(nil, err)
		AssertEq(nil, chrT.jobManager.GetJob(objName, chrT.bucket.Name()))
		AssertFalse(chrT.isEntryInFileInfoCache(objName, chrT.bucket.Name()))
	}

	// Start concurrent GetCacheHandle()
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go InvalidateCacheTestFun(i)
	}
	wg.Wait()
}

func (chrT *cacheHandlerTest) Test_InvalidateCache_GetCacheHandle_Concurrent() {
	wg := sync.WaitGroup{}

	invalidateCacheTestFun := func(index int) {
		defer wg.Done()
		objName := "object" + strconv.Itoa(index)
		objContent := "object content: content#" + strconv.Itoa(index)
		minObj := chrT.getMinObject(objName, []byte(objContent))

		err := chrT.cacheHandler.InvalidateCache(minObj.Name, chrT.bucket.Name())

		AssertEq(nil, err)
	}

	getCacheHandleTestFun := func(index int) {
		defer wg.Done()
		objName := "object" + strconv.Itoa(index)
		objContent := "object content: content#" + strconv.Itoa(index)
		minObj := chrT.getMinObject(objName, []byte(objContent))

		cacheHandle, err := chrT.cacheHandler.GetCacheHandle(minObj, chrT.bucket, false, 0)

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
}

func (chrT *cacheHandlerTest) Test_Destroy() {
	minObject1 := chrT.getMinObject("object_1", []byte("content of object_1"))
	minObject2 := chrT.getMinObject("object_2", []byte("content of object_2"))
	cacheHandle1, err := chrT.cacheHandler.GetCacheHandle(minObject1, chrT.bucket, true, 0)
	AssertEq(nil, err)
	cacheHandle2, err := chrT.cacheHandler.GetCacheHandle(minObject2, chrT.bucket, true, 0)
	AssertEq(nil, err)
	ctx := context.Background()
	// Read to create and populate file in cache.
	buf := make([]byte, 3)
	_, cacheHit, err := cacheHandle1.Read(ctx, chrT.bucket, minObject1, 4, buf)
	AssertEq(nil, err)
	AssertEq(false, cacheHit)
	_, cacheHit, err = cacheHandle2.Read(ctx, chrT.bucket, minObject2, 4, buf)
	AssertEq(nil, err)
	AssertEq(false, cacheHit)
	err = cacheHandle1.Close()
	AssertEq(nil, err)
	err = cacheHandle2.Close()
	AssertEq(nil, err)

	err = chrT.cacheHandler.Destroy()

	AssertEq(nil, err)
	// Verify the cacheDir is deleted.
	_, err = os.Stat(path.Join(chrT.cacheDir, util.FileCache))
	AssertNe(nil, err)
	AssertTrue(errors.Is(err, os.ErrNotExist))
	// Verify jobs are either removed or completed and removed themselves.
	job1 := chrT.jobManager.GetJob(minObject1.Name, chrT.bucket.Name())
	job2 := chrT.jobManager.GetJob(minObject1.Name, chrT.bucket.Name())
	AssertTrue((job1 == nil) || (job1.GetStatus().Name == downloader.Completed))
	AssertTrue((job2 == nil) || (job2.GetStatus().Name == downloader.Completed))
	// Job manager should no longer contain the jobs
	AssertEq(nil, chrT.jobManager.GetJob(minObject1.Name, chrT.bucket.Name()))
	AssertEq(nil, chrT.jobManager.GetJob(minObject2.Name, chrT.bucket.Name()))
}
