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
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const HandlerCacheMaxSize = TestObjectSize + ObjectSizeToCauseEviction
const ObjectSizeToCauseEviction = 20

func TestCacheHandler(t *testing.T) {
	suite.Run(t, new(cacheHandlerTest))
}

type cacheHandlerTest struct {
	suite.Suite
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

func (chrT *cacheHandlerTest) SetupTest() {
	locker.EnableInvariantsCheck()
	chrT.cacheDir = path.Join(os.Getenv("HOME"), "cache/dir")

	// Create bucket in fake storage.
	chrT.fakeStorage = storage.NewFakeStorage()
	storageHandle := chrT.fakeStorage.CreateStorageHandle()
	chrT.bucket = storageHandle.BucketHandle(storage.TestBucketName, "")

	// Create test object in the bucket.
	testObjectContent := make([]byte, TestObjectSize)
	_, err := rand.Read(testObjectContent)
	require.NoError(chrT.T(), err)
	chrT.object = chrT.getMinObject(TestObjectName, testObjectContent)

	// fileInfoCache with testFileInfoEntry
	chrT.cache = lru.NewCache(HandlerCacheMaxSize)

	// Job manager
	chrT.jobManager = downloader.NewJobManager(chrT.cache, util.DefaultFilePerm,
		util.DefaultDirPerm, chrT.cacheDir, DefaultSequentialReadSizeMb, &config.FileCacheConfig{
			EnableCRC: true,
		})

	// Mocked cached handler object.
	chrT.cacheHandler = NewCacheHandler(chrT.cache, chrT.jobManager, chrT.cacheDir, util.DefaultFilePerm, util.DefaultDirPerm)

	// Follow consistency, local-cache file, entry in fileInfo cache and job should exist initially.
	chrT.fileInfoKeyName = chrT.addTestFileInfoEntryInCache(storage.TestBucketName, TestObjectName)
	chrT.downloadPath = util.GetDownloadPath(chrT.cacheHandler.cacheDir, util.GetObjectPath(chrT.bucket.Name(), chrT.object.Name))
	_, err = util.CreateFile(data.FileSpec{Path: chrT.downloadPath, FilePerm: util.DefaultFilePerm, DirPerm: util.DefaultDirPerm}, os.O_RDONLY)
	require.NoError(chrT.T(), err)
	_ = chrT.getDownloadJobForTestObject()
}

func (chrT *cacheHandlerTest) TearDownTest() {
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
	require.NoError(chrT.T(), err)

	_, err = chrT.cache.Insert(fileInfoKeyName, fileInfo)
	require.NoError(chrT.T(), err)

	return fileInfoKeyName
}

func (chrT *cacheHandlerTest) isEntryInFileInfoCache(objectName string, bucketName string) bool {
	fileInfoKey := data.FileInfoKey{
		BucketName: bucketName,
		ObjectName: objectName,
	}

	fileInfoKeyName, err := fileInfoKey.Key()
	require.NoError(chrT.T(), err)

	fileInfo := chrT.cache.LookUp(fileInfoKeyName)
	return fileInfo != nil
}

func (chrT *cacheHandlerTest) getDownloadJobForTestObject() *downloader.Job {
	job := chrT.jobManager.CreateJobIfNotExists(chrT.object, chrT.bucket)
	require.NotNil(chrT.T(), job)
	return job
}

func (chrT *cacheHandlerTest) getMinObject(objName string, objContent []byte) *gcs.MinObject {
	ctx := context.Background()
	objects := map[string][]byte{objName: objContent}
	err := storageutil.CreateObjects(ctx, chrT.bucket, objects)
	require.NoError(chrT.T(), err)

	minObject, _, err := chrT.bucket.StatObject(ctx, &gcs.StatObjectRequest{Name: objName,
		ForceFetchFromGcs: true})
	require.NoError(chrT.T(), err)
	require.NotNil(chrT.T(), minObject)
	return minObject
}

// doesFileExist returns true if the file exists and false otherwise.
// If an error occurs, the function panics.
func (chrT *cacheHandlerTest) doesFileExist(filePath string) bool {
	_, err := os.Stat(filePath)

	if err == nil {
		return true
	}

	if os.IsNotExist(err) {
		return false
	}

	require.NoError(chrT.T(), err)
	return false
}

func (chrT *cacheHandlerTest) Test_createLocalFileReadHandle_OnlyForRead() {
	readFileHandle, err := chrT.cacheHandler.createLocalFileReadHandle(chrT.object.Name, chrT.bucket.Name())

	assert.NoError(chrT.T(), err)
	_, err = readFileHandle.Write([]byte("test"))
	assert.ErrorContains(chrT.T(), err, "bad file descriptor")
}

func (chrT *cacheHandlerTest) Test_cleanUpEvictedFile() {
	fileDownloadJob := chrT.getDownloadJobForTestObject()
	fileInfo := chrT.cache.LookUp(chrT.fileInfoKeyName)
	fileInfoData := fileInfo.(data.FileInfo)
	jobStatusBefore := fileDownloadJob.GetStatus()
	require.Equal(chrT.T(), jobStatusBefore.Name, downloader.NotStarted)
	jobStatusBefore, err := fileDownloadJob.Download(context.Background(), int64(util.MiB), false)
	require.NoError(chrT.T(), err)
	require.Equal(chrT.T(), jobStatusBefore.Name, downloader.Downloading)

	err = chrT.cacheHandler.cleanUpEvictedFile(&fileInfoData)

	assert.NoError(chrT.T(), err)
	jobStatusAfter := fileDownloadJob.GetStatus()
	assert.Equal(chrT.T(), jobStatusAfter.Name, downloader.Invalid)
	assert.False(chrT.T(), chrT.doesFileExist(chrT.downloadPath))
	// Job should be removed from job manager
	assert.Nil(chrT.T(), chrT.jobManager.GetJob(chrT.object.Name, chrT.bucket.Name()))
}

func (chrT *cacheHandlerTest) Test_cleanUpEvictedFile_WhenLocalFileNotExist() {
	fileDownloadJob := chrT.getDownloadJobForTestObject()
	fileInfo := chrT.cache.LookUp(chrT.fileInfoKeyName)
	fileInfoData := fileInfo.(data.FileInfo)
	jobStatusBefore := fileDownloadJob.GetStatus()
	require.Equal(chrT.T(), jobStatusBefore.Name, downloader.NotStarted)
	jobStatusBefore, err := fileDownloadJob.Download(context.Background(), int64(util.MiB), false)
	require.NoError(chrT.T(), err)
	require.Equal(chrT.T(), jobStatusBefore.Name, downloader.Downloading)
	err = os.Remove(chrT.downloadPath)
	require.NoError(chrT.T(), err)

	err = chrT.cacheHandler.cleanUpEvictedFile(&fileInfoData)

	assert.NoError(chrT.T(), err)
	jobStatusAfter := fileDownloadJob.GetStatus()
	assert.Equal(chrT.T(), jobStatusAfter.Name, downloader.Invalid)
	assert.False(chrT.T(), chrT.doesFileExist(chrT.downloadPath))
	// Job should be removed from job manager
	assert.Nil(chrT.T(), chrT.jobManager.GetJob(chrT.object.Name, chrT.bucket.Name()))
}

func (chrT *cacheHandlerTest) Test_addFileInfoEntryAndCreateDownloadJob_IfAlready() {
	existingJob := chrT.getDownloadJobForTestObject()

	err := chrT.cacheHandler.addFileInfoEntryAndCreateDownloadJob(chrT.object, chrT.bucket)

	assert.NoError(chrT.T(), err)
	assert.True(chrT.T(), chrT.isEntryInFileInfoCache(chrT.object.Name, chrT.bucket.Name()))
	// File download job should also be same
	actualJob := chrT.jobManager.GetJob(chrT.object.Name, chrT.bucket.Name())
	assert.Equal(chrT.T(), existingJob, actualJob)
}

func (chrT *cacheHandlerTest) Test_addFileInfoEntryAndCreateDownloadJob_GenerationChanged() {
	existingJob := chrT.getDownloadJobForTestObject()
	chrT.object.Generation = chrT.object.Generation + 1

	err := chrT.cacheHandler.addFileInfoEntryAndCreateDownloadJob(chrT.object, chrT.bucket)

	assert.NoError(chrT.T(), err)
	assert.True(chrT.T(), chrT.isEntryInFileInfoCache(chrT.object.Name, chrT.bucket.Name()))
	// File download job should be new as the file info and job should be cleaned
	// up.
	actualJob := chrT.jobManager.GetJob(chrT.object.Name, chrT.bucket.Name())
	assert.NotEqual(chrT.T(), existingJob, actualJob)
}

func (chrT *cacheHandlerTest) Test_addFileInfoEntryAndCreateDownloadJob_IfNotAlready() {
	oldJob := chrT.getDownloadJobForTestObject()
	// Content of size more than 20 leads to eviction of initial TestObjectName.
	// Here, content size is 21.
	minObject := chrT.getMinObject("object_1", []byte("content of object_1 ..."))
	// There should be no file download job corresponding to minObject
	existingJob := chrT.jobManager.GetJob(minObject.Name, chrT.bucket.Name())
	require.Nil(chrT.T(), existingJob)

	// Insertion will happen and that leads to eviction.
	err := chrT.cacheHandler.addFileInfoEntryAndCreateDownloadJob(minObject, chrT.bucket)

	assert.NoError(chrT.T(), err)
	assert.True(chrT.T(), chrT.isEntryInFileInfoCache(minObject.Name, chrT.bucket.Name()))
	jobStatus := oldJob.GetStatus()
	assert.Equal(chrT.T(), downloader.Invalid, jobStatus.Name)
	assert.Equal(chrT.T(), false, chrT.doesFileExist(chrT.downloadPath))
	// Job should be added for minObject
	minObjectJob := chrT.jobManager.GetJob(minObject.Name, chrT.bucket.Name())
	assert.NotNil(chrT.T(), minObjectJob)
	assert.Equal(chrT.T(), downloader.NotStarted, minObjectJob.GetStatus().Name)
}

func (chrT *cacheHandlerTest) Test_addFileInfoEntryAndCreateDownloadJob_IfLocalFileGetsDeleted() {
	// Delete the local cache file.
	err := os.Remove(chrT.downloadPath)
	require.NoError(chrT.T(), err)

	// There is a fileInfoEntry in the fileInfoCache but the corresponding local file doesn't exist.
	// Hence, this will return error containing util.FileNotPresentInCacheErrMsg.
	err = chrT.cacheHandler.addFileInfoEntryAndCreateDownloadJob(chrT.object, chrT.bucket)

	assert.NotNil(chrT.T(), err)
	assert.True(chrT.T(), strings.Contains(err.Error(), util.FileNotPresentInCacheErrMsg))
}

func (chrT *cacheHandlerTest) Test_addFileInfoEntryAndCreateDownloadJob_WhenJobHasCompleted() {
	existingJob := chrT.getDownloadJobForTestObject()
	// Make the job completed, so it's removed from job manager.
	jobStatus, err := existingJob.Download(context.Background(), int64(chrT.object.Size), true)
	require.NoError(chrT.T(), err)
	require.Equal(chrT.T(), uint64(jobStatus.Offset), chrT.object.Size)
	// Give time for execution of callback to remove from job manager
	time.Sleep(time.Second)
	actualJob := chrT.jobManager.GetJob(chrT.object.Name, chrT.bucket.Name())
	require.Nil(chrT.T(), actualJob)

	err = chrT.cacheHandler.addFileInfoEntryAndCreateDownloadJob(chrT.object, chrT.bucket)

	assert.NoError(chrT.T(), err)
	assert.True(chrT.T(), chrT.isEntryInFileInfoCache(chrT.object.Name, chrT.bucket.Name()))
	// No new job should be added to job manager
	actualJob = chrT.jobManager.GetJob(chrT.object.Name, chrT.bucket.Name())
	assert.Nil(chrT.T(), actualJob)
}

func (chrT *cacheHandlerTest) Test_addFileInfoEntryAndCreateDownloadJob_WhenJobIsInvalidatedAndRemoved() {
	chrT.jobManager.InvalidateAndRemoveJob(chrT.object.Name, chrT.bucket.Name())
	existingJob := chrT.jobManager.GetJob(chrT.object.Name, chrT.bucket.Name())
	require.Nil(chrT.T(), existingJob)

	// Because the job has been removed and file info entry is still present, new
	// file info entry and job should be created.
	err := chrT.cacheHandler.addFileInfoEntryAndCreateDownloadJob(chrT.object, chrT.bucket)

	assert.NoError(chrT.T(), err)
	assert.True(chrT.T(), chrT.isEntryInFileInfoCache(chrT.object.Name, chrT.bucket.Name()))
	// New job should be added to job manager
	actualJob := chrT.jobManager.GetJob(chrT.object.Name, chrT.bucket.Name())
	assert.NotNil(chrT.T(), actualJob)
	assert.Equal(chrT.T(), downloader.NotStarted, actualJob.GetStatus().Name)
}

func (chrT *cacheHandlerTest) Test_addFileInfoEntryAndCreateDownloadJob_WhenJobHasFailed() {
	existingJob := chrT.getDownloadJobForTestObject()
	// Hack to fail the async job
	correctSize := chrT.object.Size
	chrT.object.Size = 2
	jobStatus, err := existingJob.Download(context.Background(), 1, true)
	require.NoError(chrT.T(), err)
	require.Equal(chrT.T(), downloader.Failed, jobStatus.Name)
	chrT.object.Size = correctSize

	// Because the job has been failed and file info entry is still present with
	// size less than the object's size (because the async job failed), new job
	// should be created
	err = chrT.cacheHandler.addFileInfoEntryAndCreateDownloadJob(chrT.object, chrT.bucket)

	assert.NoError(chrT.T(), err)
	assert.True(chrT.T(), chrT.isEntryInFileInfoCache(chrT.object.Name, chrT.bucket.Name()))
	// New job should be added to job manager
	actualJob := chrT.jobManager.GetJob(chrT.object.Name, chrT.bucket.Name())
	assert.NotNil(chrT.T(), actualJob)
	assert.Equal(chrT.T(), downloader.NotStarted, actualJob.GetStatus().Name)
}

func (chrT *cacheHandlerTest) Test_GetCacheHandle_WhenCacheHasDifferentGeneration() {
	existingJob := chrT.getDownloadJobForTestObject()
	require.NotNil(chrT.T(), existingJob)
	require.Equal(chrT.T(), downloader.NotStarted, existingJob.GetStatus().Name)
	// Change the version of the object, but cache still keeps old generation
	chrT.object.Generation = chrT.object.Generation + 1

	newCacheHandle, err := chrT.cacheHandler.GetCacheHandle(chrT.object, chrT.bucket, false, 0)

	assert.NoError(chrT.T(), err)
	assert.Nil(chrT.T(), newCacheHandle.validateCacheHandle())
	jobStatusOfOldJob := existingJob.GetStatus()
	assert.Equal(chrT.T(), jobStatusOfOldJob.Name, downloader.Invalid)
	jobStatusOfNewHandle := newCacheHandle.fileDownloadJob.GetStatus()
	assert.Equal(chrT.T(), jobStatusOfNewHandle.Name, downloader.NotStarted)
}

func (chrT *cacheHandlerTest) Test_GetCacheHandle_WhenAsyncDownloadJobHasFailed() {
	existingJob := chrT.getDownloadJobForTestObject()
	// Hack to fail the async job
	correctSize := chrT.object.Size
	chrT.object.Size = 2
	jobStatus, err := existingJob.Download(context.Background(), 1, true)
	require.NoError(chrT.T(), err)
	require.Equal(chrT.T(), downloader.Failed, jobStatus.Name)
	chrT.object.Size = correctSize

	newCacheHandle, err := chrT.cacheHandler.GetCacheHandle(chrT.object, chrT.bucket, false, 0)

	// New job should be created because the earlier job has failed.
	assert.NoError(chrT.T(), err)
	assert.Nil(chrT.T(), newCacheHandle.validateCacheHandle())
	assert.True(chrT.T(), chrT.isEntryInFileInfoCache(chrT.object.Name, chrT.bucket.Name()))
	jobStatusOfNewHandle := newCacheHandle.fileDownloadJob.GetStatus()
	assert.Equal(chrT.T(), downloader.NotStarted, jobStatusOfNewHandle.Name)
}

func (chrT *cacheHandlerTest) Test_GetCacheHandle_WhenFileInfoAndJobAreAlreadyPresent() {
	// File info and download job are already present for test object.
	existingJob := chrT.getDownloadJobForTestObject()

	cacheHandle, err := chrT.cacheHandler.GetCacheHandle(chrT.object, chrT.bucket, false, 0)

	assert.NoError(chrT.T(), err)
	assert.Nil(chrT.T(), cacheHandle.validateCacheHandle())
	// Job and file info are still present
	assert.True(chrT.T(), chrT.isEntryInFileInfoCache(chrT.object.Name, chrT.bucket.Name()))
	assert.Equal(chrT.T(), existingJob, cacheHandle.fileDownloadJob)
	jobStatusOfNewHandle := cacheHandle.fileDownloadJob.GetStatus()
	assert.Equal(chrT.T(), downloader.NotStarted, jobStatusOfNewHandle.Name)
}

func (chrT *cacheHandlerTest) Test_GetCacheHandle_WhenFileInfoAndJobAreNotPresent() {
	minObject := chrT.getMinObject("object_1", []byte("content of object_1"))

	cacheHandle, err := chrT.cacheHandler.GetCacheHandle(minObject, chrT.bucket, false, 0)

	assert.NoError(chrT.T(), err)
	assert.Nil(chrT.T(), cacheHandle.validateCacheHandle())
	// New Job and file info are created.
	assert.True(chrT.T(), chrT.isEntryInFileInfoCache(minObject.Name, chrT.bucket.Name()))
	jobStatusOfNewHandle := cacheHandle.fileDownloadJob.GetStatus()
	assert.Equal(chrT.T(), downloader.NotStarted, jobStatusOfNewHandle.Name)
}

func (chrT *cacheHandlerTest) Test_GetCacheHandle_WithEviction() {
	// Start the existing job
	existingJob := chrT.getDownloadJobForTestObject()
	_, err := existingJob.Download(context.Background(), 1, false)
	require.NoError(chrT.T(), err)
	// Content of size more than 20 leads to eviction of initial TestObjectName.
	// Here, content size is 21.
	minObject := chrT.getMinObject("object_1", []byte("content of object_1 ..."))

	cacheHandle2, err := chrT.cacheHandler.GetCacheHandle(minObject, chrT.bucket, false, 0)

	assert.NoError(chrT.T(), err)
	assert.Nil(chrT.T(), cacheHandle2.validateCacheHandle())
	jobStatus := existingJob.GetStatus()
	assert.Equal(chrT.T(), downloader.Invalid, jobStatus.Name)
	assert.False(chrT.T(), chrT.doesFileExist(chrT.downloadPath))
}

func (chrT *cacheHandlerTest) Test_GetCacheHandle_IfLocalFileGetsDeleted() {
	// Delete the local cache file.
	err := os.Remove(chrT.downloadPath)
	require.NoError(chrT.T(), err)
	existingJob := chrT.getDownloadJobForTestObject()

	cacheHandle, err := chrT.cacheHandler.GetCacheHandle(chrT.object, chrT.bucket, false, 0)

	assert.NotNil(chrT.T(), err)
	assert.True(chrT.T(), strings.Contains(err.Error(), util.FileNotPresentInCacheErrMsg))
	assert.Nil(chrT.T(), cacheHandle)
	// Check file info and download job are not removed
	assert.True(chrT.T(), chrT.isEntryInFileInfoCache(chrT.object.Name, chrT.bucket.Name()))
	actualJob := chrT.jobManager.GetJob(chrT.object.Name, chrT.bucket.Name())
	assert.Equal(chrT.T(), existingJob, actualJob)
	assert.Equal(chrT.T(), downloader.NotStarted, existingJob.GetStatus().Name)

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

	assert.NoError(chrT.T(), err1)
	assert.Nil(chrT.T(), cacheHandle1.validateCacheHandle())
	assert.ErrorContains(chrT.T(), err2, util.CacheHandleNotRequiredForRandomReadErrMsg)
	assert.Nil(chrT.T(), cacheHandle2)
	assert.NoError(chrT.T(), err3)
	assert.Nil(chrT.T(), cacheHandle3.validateCacheHandle())
	assert.NoError(chrT.T(), err4)
	assert.Nil(chrT.T(), cacheHandle4.validateCacheHandle())
}

func (chrT *cacheHandlerTest) Test_GetCacheHandle_ConcurrentSameFile() {
	// Check async job and file info cache not preset for object_1
	testObjectName := "object_1"
	existingJob := chrT.jobManager.GetJob(testObjectName, chrT.bucket.Name())
	require.Nil(chrT.T(), existingJob)
	wg := sync.WaitGroup{}
	getCacheHandleTestFun := func() {
		defer wg.Done()
		minObj := chrT.getMinObject(testObjectName, []byte("content of object_1 ..."))

		var err error
		cacheHandle, err := chrT.cacheHandler.GetCacheHandle(minObj, chrT.bucket, false, 0)

		assert.NoError(chrT.T(), err)
		assert.Nil(chrT.T(), cacheHandle.validateCacheHandle())
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
	assert.Equal(chrT.T(), downloader.NotStarted, jobStatus.Name)
	assert.True(chrT.T(), chrT.doesFileExist(util.GetDownloadPath(chrT.cacheDir, util.GetObjectPath(chrT.bucket.Name(), testObjectName))))
}

func (chrT *cacheHandlerTest) Test_GetCacheHandle_ConcurrentDifferentFiles() {
	existingJob := chrT.getDownloadJobForTestObject()
	require.Equal(chrT.T(), downloader.NotStarted, existingJob.GetStatus().Name)
	wg := sync.WaitGroup{}

	getCacheHandleTestFun := func(index int) {
		defer wg.Done()
		objName := "object" + strconv.Itoa(index)
		objContent := "object content: content#" + strconv.Itoa(index)
		minObj := chrT.getMinObject(objName, []byte(objContent))

		cacheHandle, err := chrT.cacheHandler.GetCacheHandle(minObj, chrT.bucket, false, 0)

		assert.NoError(chrT.T(), err)
		assert.Nil(chrT.T(), cacheHandle.validateCacheHandle())
	}

	// Start concurrent GetCacheHandle()
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go getCacheHandleTestFun(i)
	}
	wg.Wait()

	// Existing job for default chrT object should be invalidated.
	assert.NotNil(chrT.T(), existingJob)
	assert.Equal(chrT.T(), downloader.Invalid, existingJob.GetStatus().Name)
	assert.Equal(chrT.T(), false, chrT.doesFileExist(chrT.downloadPath))
	// File info should also be removed.
	assert.False(chrT.T(), chrT.isEntryInFileInfoCache(chrT.object.Name, chrT.bucket.Name()))
}

func (chrT *cacheHandlerTest) Test_InvalidateCache_WhenAlreadyInCache() {
	existingJob := chrT.getDownloadJobForTestObject()
	require.Equal(chrT.T(), downloader.NotStarted, existingJob.GetStatus().Name)
	require.True(chrT.T(), chrT.isEntryInFileInfoCache(chrT.object.Name, chrT.bucket.Name()))

	err := chrT.cacheHandler.InvalidateCache(chrT.object.Name, chrT.bucket.Name())

	assert.NoError(chrT.T(), err)
	// Existing job for default chrT object should be invalidated.
	assert.NotNil(chrT.T(), existingJob)
	assert.Equal(chrT.T(), downloader.Invalid, existingJob.GetStatus().Name)
	assert.Equal(chrT.T(), false, chrT.doesFileExist(chrT.downloadPath))
	// File info should also be removed.
	assert.False(chrT.T(), chrT.isEntryInFileInfoCache(chrT.object.Name, chrT.bucket.Name()))
}

func (chrT *cacheHandlerTest) Test_InvalidateCache_WhenEntryNotInCache() {
	minObject := chrT.getMinObject("object_1", []byte("content of object_1"))
	require.False(chrT.T(), chrT.isEntryInFileInfoCache(minObject.Name, chrT.bucket.Name()))
	require.Nil(chrT.T(), chrT.jobManager.GetJob(minObject.Name, chrT.bucket.Name()))

	err := chrT.cacheHandler.InvalidateCache(minObject.Name, chrT.bucket.Name())

	assert.NoError(chrT.T(), err)
	assert.False(chrT.T(), chrT.isEntryInFileInfoCache(minObject.Name, chrT.bucket.Name()))
	assert.Nil(chrT.T(), chrT.jobManager.GetJob(minObject.Name, chrT.bucket.Name()))
}

func (chrT *cacheHandlerTest) Test_InvalidateCache_Truncates() {
	objectContent := []byte("content of object_1")
	minObject := chrT.getMinObject("object_1", objectContent)
	cacheHandle, err := chrT.cacheHandler.GetCacheHandle(minObject, chrT.bucket, false, 0)
	require.NoError(chrT.T(), err)
	buf := make([]byte, 3)
	ctx := context.Background()
	// Read to populate cache
	_, cacheHit, err := cacheHandle.Read(ctx, chrT.bucket, minObject, 0, buf)
	require.NoError(chrT.T(), err)
	require.Equal(chrT.T(), false, cacheHit)
	require.Equal(chrT.T(), string(objectContent[:3]), string(buf))
	require.Nil(chrT.T(), cacheHandle.Close())
	// Open cache file before invalidation
	objectPath := util.GetObjectPath(chrT.bucket.Name(), minObject.Name)
	downloadPath := util.GetDownloadPath(chrT.cacheDir, objectPath)
	file, err := os.OpenFile(downloadPath, os.O_RDONLY, 0600)
	require.NoError(chrT.T(), err)
	_, err = file.Read(buf)
	require.NoError(chrT.T(), err)
	require.Equal(chrT.T(), string(objectContent[:3]), string(buf))

	err = chrT.cacheHandler.InvalidateCache(minObject.Name, chrT.bucket.Name())

	assert.NoError(chrT.T(), err)
	// Reading from the open file handle should fail as the file is truncated.
	_, err = file.Read(buf)
	assert.NotNil(chrT.T(), err)
	assert.Equal(chrT.T(), io.EOF, err)
}

func (chrT *cacheHandlerTest) Test_InvalidateCache_ConcurrentSameFile() {
	existingJob := chrT.getDownloadJobForTestObject()
	require.Equal(chrT.T(), downloader.NotStarted, existingJob.GetStatus().Name)
	require.True(chrT.T(), chrT.isEntryInFileInfoCache(chrT.object.Name, chrT.bucket.Name()))
	wg := sync.WaitGroup{}
	InvalidateCacheTestFun := func() {
		defer wg.Done()

		err := chrT.cacheHandler.InvalidateCache(chrT.object.Name, chrT.bucket.Name())

		assert.NoError(chrT.T(), err)
		assert.NotNil(chrT.T(), existingJob)
		assert.Equal(chrT.T(), downloader.Invalid, existingJob.GetStatus().Name)
		assert.Equal(chrT.T(), false, chrT.doesFileExist(chrT.downloadPath))
		// File info should also be removed.
		assert.False(chrT.T(), chrT.isEntryInFileInfoCache(chrT.object.Name, chrT.bucket.Name()))
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

		assert.NoError(chrT.T(), err)
		assert.Nil(chrT.T(), chrT.jobManager.GetJob(objName, chrT.bucket.Name()))
		assert.False(chrT.T(), chrT.isEntryInFileInfoCache(objName, chrT.bucket.Name()))
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

		assert.NoError(chrT.T(), err)
	}

	getCacheHandleTestFun := func(index int) {
		defer wg.Done()
		objName := "object" + strconv.Itoa(index)
		objContent := "object content: content#" + strconv.Itoa(index)
		minObj := chrT.getMinObject(objName, []byte(objContent))

		cacheHandle, err := chrT.cacheHandler.GetCacheHandle(minObj, chrT.bucket, false, 0)

		assert.NoError(chrT.T(), err)
		assert.Nil(chrT.T(), cacheHandle.validateCacheHandle())
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
	require.NoError(chrT.T(), err)
	cacheHandle2, err := chrT.cacheHandler.GetCacheHandle(minObject2, chrT.bucket, true, 0)
	require.NoError(chrT.T(), err)
	ctx := context.Background()
	// Read to create and populate file in cache.
	buf := make([]byte, 3)
	_, cacheHit, err := cacheHandle1.Read(ctx, chrT.bucket, minObject1, 4, buf)
	require.NoError(chrT.T(), err)
	require.Equal(chrT.T(), false, cacheHit)
	_, cacheHit, err = cacheHandle2.Read(ctx, chrT.bucket, minObject2, 4, buf)
	require.NoError(chrT.T(), err)
	require.Equal(chrT.T(), false, cacheHit)
	err = cacheHandle1.Close()
	require.NoError(chrT.T(), err)
	err = cacheHandle2.Close()
	require.NoError(chrT.T(), err)

	err = chrT.cacheHandler.Destroy()

	assert.NoError(chrT.T(), err)
	// Verify the cacheDir is deleted.
	_, err = os.Stat(path.Join(chrT.cacheDir, util.FileCache))
	assert.NotNil(chrT.T(), err)
	assert.True(chrT.T(), errors.Is(err, os.ErrNotExist))
	// Verify jobs are either removed or completed and removed themselves.
	job1 := chrT.jobManager.GetJob(minObject1.Name, chrT.bucket.Name())
	job2 := chrT.jobManager.GetJob(minObject1.Name, chrT.bucket.Name())
	assert.True(chrT.T(), (job1 == nil) || (job1.GetStatus().Name == downloader.Completed))
	assert.True(chrT.T(), (job2 == nil) || (job2.GetStatus().Name == downloader.Completed))
	// Job manager should no longer contain the jobs
	assert.Nil(chrT.T(), chrT.jobManager.GetJob(minObject1.Name, chrT.bucket.Name()))
	assert.Nil(chrT.T(), chrT.jobManager.GetJob(minObject2.Name, chrT.bucket.Name()))
}
