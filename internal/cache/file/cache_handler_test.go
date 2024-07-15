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
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
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

func TestCacheHandler(testSuite *testing.T) {
	testSuite.Parallel()
	suite.Run(testSuite, new(CacheHandlerTest))
}

type cacheHandlerTestArgs struct {
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

type CacheHandlerTest struct {
	suite.Suite
	cacheDir   string
	chTestArgs *cacheHandlerTestArgs
}

func (chrT *CacheHandlerTest) SetupTest() {
	chrT.cacheDir = path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir")
}

func setupHelper(t *testing.T, fileCacheConfig *config.FileCacheConfig, cacheDir string) (chTestArgs *cacheHandlerTestArgs) {
	t.Helper()
	chTestArgs = &cacheHandlerTestArgs{}
	locker.EnableInvariantsCheck()
	chTestArgs.cacheDir = cacheDir

	// Create bucket in fake storage.
	chTestArgs.fakeStorage = storage.NewFakeStorage()
	storageHandle := chTestArgs.fakeStorage.CreateStorageHandle()
	chTestArgs.bucket = storageHandle.BucketHandle(storage.TestBucketName, "")

	// Create test object in the bucket.
	testObjectContent := make([]byte, TestObjectSize)
	_, err := rand.Read(testObjectContent)
	require.NoError(t, err)
	chTestArgs.object = createObject(t, chTestArgs.bucket, TestObjectName, testObjectContent)

	// fileInfoCache with testFileInfoEntry
	chTestArgs.cache = lru.NewCache(HandlerCacheMaxSize)

	// Job manager
	chTestArgs.jobManager = downloader.NewJobManager(chTestArgs.cache, util.DefaultFilePerm,
		util.DefaultDirPerm, cacheDir, DefaultSequentialReadSizeMb, fileCacheConfig)

	// Mocked cached handler object.
	chTestArgs.cacheHandler = NewCacheHandler(chTestArgs.cache, chTestArgs.jobManager, cacheDir, util.DefaultFilePerm, util.DefaultDirPerm)

	// Follow consistency, local-cache file, entry in fileInfo cache and job should exist initially.
	chTestArgs.fileInfoKeyName = addTestFileInfoEntryInCache(t, chTestArgs, storage.TestBucketName, TestObjectName)
	chTestArgs.downloadPath = util.GetDownloadPath(chTestArgs.cacheHandler.cacheDir, util.GetObjectPath(chTestArgs.bucket.Name(), chTestArgs.object.Name))
	_, err = util.CreateFile(data.FileSpec{Path: chTestArgs.downloadPath, FilePerm: util.DefaultFilePerm, DirPerm: util.DefaultDirPerm}, os.O_RDONLY)
	require.NoError(t, err)
	_ = getDownloadJobForTestObject(t, chTestArgs)

	t.Cleanup(func() {
		chTestArgs.fakeStorage.ShutDown()
		operations.RemoveDir(cacheDir)
	})
	return chTestArgs
}

func createObject(t *testing.T, bucket gcs.Bucket, objName string, objContent []byte) *gcs.MinObject {
	t.Helper()
	ctx := context.Background()
	objects := map[string][]byte{objName: objContent}
	err := storageutil.CreateObjects(ctx, bucket, objects)
	require.NoError(t, err)

	minObject, _, err := bucket.StatObject(ctx, &gcs.StatObjectRequest{Name: objName,
		ForceFetchFromGcs: true})
	require.NoError(t, err)
	require.NotNil(t, minObject)
	return minObject
}

func addTestFileInfoEntryInCache(t *testing.T, chTestArgs *cacheHandlerTestArgs, bucketName string, objectName string) string {
	t.Helper()
	// Add an entry into
	fileInfoKey := data.FileInfoKey{
		BucketName: bucketName,
		ObjectName: objectName,
	}
	fileInfo := data.FileInfo{
		Key:              fileInfoKey,
		ObjectGeneration: chTestArgs.object.Generation,
		FileSize:         chTestArgs.object.Size,
		Offset:           0,
	}

	fileInfoKeyName, err := fileInfoKey.Key()
	require.NoError(t, nil, err)

	_, err = chTestArgs.cache.Insert(fileInfoKeyName, fileInfo)
	require.NoError(t, nil, err)

	return fileInfoKeyName
}

func getDownloadJobForTestObject(t *testing.T, chTestArgs *cacheHandlerTestArgs) *downloader.Job {
	t.Helper()
	job := chTestArgs.jobManager.CreateJobIfNotExists(chTestArgs.object, chTestArgs.bucket)
	require.NotNil(t, job)
	return job
}

func isEntryInFileInfoCache(t *testing.T, cache *lru.Cache, objectName string, bucketName string) bool {
	t.Helper()
	fileInfoKey := data.FileInfoKey{
		BucketName: bucketName,
		ObjectName: objectName,
	}

	fileInfoKeyName, err := fileInfoKey.Key()
	require.NoError(t, nil, err)

	fileInfo := cache.LookUp(fileInfoKeyName)
	return fileInfo != nil
}

// doesFileExist returns true if the file exists and false otherwise.
func doesFileExist(t *testing.T, filePath string) bool {
	t.Helper()
	_, err := os.Stat(filePath)

	if err == nil {
		return true
	}
	require.ErrorIs(t, err, os.ErrNotExist)
	return false
}

func (chrT *CacheHandlerTest) Test_createLocalFileReadHandle_OnlyForRead() {
	chrT.chTestArgs = setupHelper(chrT.T(), &config.FileCacheConfig{EnableCRC: true}, chrT.cacheDir)

	readFileHandle, err := chrT.chTestArgs.cacheHandler.createLocalFileReadHandle(chrT.chTestArgs.object.Name, chrT.chTestArgs.bucket.Name())

	assert.NoError(chrT.T(), err)
	_, err = readFileHandle.Write([]byte("test"))
	assert.ErrorContains(chrT.T(), err, "bad file descriptor")
}

func (chrT *CacheHandlerTest) Test_cleanUpEvictedFile() {
	tbl := []struct {
		name            string
		fileCacheConfig config.FileCacheConfig
		cacheDir        string
	}{
		{
			name:            "Non parallel downloads",
			fileCacheConfig: config.FileCacheConfig{EnableCRC: true},
			cacheDir:        path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
		},
		{
			name: "Parallel downloads",
			fileCacheConfig: config.FileCacheConfig{EnableCRC: true, EnableParallelDownloads: true,
				ParallelDownloadsPerFile: 4, MaxParallelDownloads: 20, DownloadChunkSizeMB: 3},
			cacheDir: path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
		},
	}
	for _, tc := range tbl {
		chrT.T().Run(tc.name, func(t *testing.T) {
			chrT.chTestArgs = setupHelper(chrT.T(), &tc.fileCacheConfig, tc.cacheDir)
			fileDownloadJob := getDownloadJobForTestObject(chrT.T(), chrT.chTestArgs)
			fileInfo := chrT.chTestArgs.cache.LookUp(chrT.chTestArgs.fileInfoKeyName)
			fileInfoData := fileInfo.(data.FileInfo)
			jobStatusBefore := fileDownloadJob.GetStatus()
			require.Equal(chrT.T(), downloader.NotStarted, jobStatusBefore.Name)
			jobStatusBefore, err := fileDownloadJob.Download(context.Background(), int64(util.MiB), false)
			require.NoError(chrT.T(), err)
			require.Equal(chrT.T(), downloader.Downloading, jobStatusBefore.Name)

			err = chrT.chTestArgs.cacheHandler.cleanUpEvictedFile(&fileInfoData)

			assert.NoError(chrT.T(), err)
			jobStatusAfter := fileDownloadJob.GetStatus()
			assert.Equal(chrT.T(), downloader.Invalid, jobStatusAfter.Name)
			assert.False(chrT.T(), doesFileExist(chrT.T(), chrT.chTestArgs.downloadPath))
			// Job should be removed from job manager
			assert.Nil(chrT.T(), chrT.chTestArgs.jobManager.GetJob(chrT.chTestArgs.object.Name, chrT.chTestArgs.bucket.Name()))
		})
	}
}

func (chrT *CacheHandlerTest) Test_cleanUpEvictedFile_WhenLocalFileNotExist() {
	chrT.chTestArgs = setupHelper(chrT.T(), &config.FileCacheConfig{EnableCRC: true}, chrT.cacheDir)
	fileDownloadJob := getDownloadJobForTestObject(chrT.T(), chrT.chTestArgs)
	fileInfo := chrT.chTestArgs.cache.LookUp(chrT.chTestArgs.fileInfoKeyName)
	fileInfoData := fileInfo.(data.FileInfo)
	jobStatusBefore := fileDownloadJob.GetStatus()
	require.Equal(chrT.T(), downloader.NotStarted, jobStatusBefore.Name)
	jobStatusBefore, err := fileDownloadJob.Download(context.Background(), int64(util.MiB), false)
	require.NoError(chrT.T(), err)
	require.Equal(chrT.T(), downloader.Downloading, jobStatusBefore.Name)
	err = os.Remove(chrT.chTestArgs.downloadPath)
	require.NoError(chrT.T(), err)

	err = chrT.chTestArgs.cacheHandler.cleanUpEvictedFile(&fileInfoData)

	assert.NoError(chrT.T(), err)
	jobStatusAfter := fileDownloadJob.GetStatus()
	assert.Equal(chrT.T(), downloader.Invalid, jobStatusAfter.Name)
	assert.False(chrT.T(), doesFileExist(chrT.T(), chrT.chTestArgs.downloadPath))
	// Job should be removed from job manager
	assert.Nil(chrT.T(), chrT.chTestArgs.jobManager.GetJob(chrT.chTestArgs.object.Name, chrT.chTestArgs.bucket.Name()))
}

func (chrT *CacheHandlerTest) Test_addFileInfoEntryAndCreateDownloadJob_IfAlready() {
	chrT.chTestArgs = setupHelper(chrT.T(), &config.FileCacheConfig{EnableCRC: true}, chrT.cacheDir)
	existingJob := getDownloadJobForTestObject(chrT.T(), chrT.chTestArgs)

	err := chrT.chTestArgs.cacheHandler.addFileInfoEntryAndCreateDownloadJob(chrT.chTestArgs.object, chrT.chTestArgs.bucket)

	assert.NoError(chrT.T(), err)
	assert.True(chrT.T(), isEntryInFileInfoCache(chrT.T(), chrT.chTestArgs.cache, chrT.chTestArgs.object.Name, chrT.chTestArgs.bucket.Name()))
	// File download job should also be same
	actualJob := chrT.chTestArgs.jobManager.GetJob(chrT.chTestArgs.object.Name, chrT.chTestArgs.bucket.Name())
	assert.Equal(chrT.T(), existingJob, actualJob)
}

func (chrT *CacheHandlerTest) Test_addFileInfoEntryAndCreateDownloadJob_GenerationChanged() {
	chrT.chTestArgs = setupHelper(chrT.T(), &config.FileCacheConfig{EnableCRC: true}, chrT.cacheDir)
	existingJob := getDownloadJobForTestObject(chrT.T(), chrT.chTestArgs)
	chrT.chTestArgs.object.Generation = chrT.chTestArgs.object.Generation + 1

	err := chrT.chTestArgs.cacheHandler.addFileInfoEntryAndCreateDownloadJob(chrT.chTestArgs.object, chrT.chTestArgs.bucket)

	assert.NoError(chrT.T(), err)
	assert.True(chrT.T(), isEntryInFileInfoCache(chrT.T(), chrT.chTestArgs.cache, chrT.chTestArgs.object.Name, chrT.chTestArgs.bucket.Name()))
	// File download job should be new as the file info and job should be cleaned
	// up.
	actualJob := chrT.chTestArgs.jobManager.GetJob(chrT.chTestArgs.object.Name, chrT.chTestArgs.bucket.Name())
	assert.NotEqual(chrT.T(), existingJob, actualJob)
}

func (chrT *CacheHandlerTest) Test_addFileInfoEntryAndCreateDownloadJob_IfNotAlready() {
	chrT.chTestArgs = setupHelper(chrT.T(), &config.FileCacheConfig{EnableCRC: true}, chrT.cacheDir)
	oldJob := getDownloadJobForTestObject(chrT.T(), chrT.chTestArgs)
	// Content of size more than 20 leads to eviction of initial TestObjectName.
	// Here, content size is 21.
	minObject := createObject(chrT.T(), chrT.chTestArgs.bucket, "object_1", []byte("content of object_1 ..."))
	// There should be no file download job corresponding to minObject
	existingJob := chrT.chTestArgs.jobManager.GetJob(minObject.Name, chrT.chTestArgs.bucket.Name())
	require.Nil(chrT.T(), existingJob)

	// Insertion will happen and that leads to eviction.
	err := chrT.chTestArgs.cacheHandler.addFileInfoEntryAndCreateDownloadJob(minObject, chrT.chTestArgs.bucket)

	assert.NoError(chrT.T(), err)
	assert.True(chrT.T(), isEntryInFileInfoCache(chrT.T(), chrT.chTestArgs.cache, minObject.Name, chrT.chTestArgs.bucket.Name()))
	jobStatus := oldJob.GetStatus()
	assert.Equal(chrT.T(), downloader.Invalid, jobStatus.Name)
	assert.False(chrT.T(), doesFileExist(chrT.T(), chrT.chTestArgs.downloadPath))
	// Job should be added for minObject
	minObjectJob := chrT.chTestArgs.jobManager.GetJob(minObject.Name, chrT.chTestArgs.bucket.Name())
	assert.NotNil(chrT.T(), minObjectJob)
	assert.Equal(chrT.T(), downloader.NotStarted, minObjectJob.GetStatus().Name)
}

func (chrT *CacheHandlerTest) Test_addFileInfoEntryAndCreateDownloadJob_IfLocalFileGetsDeleted() {
	chrT.chTestArgs = setupHelper(chrT.T(), &config.FileCacheConfig{EnableCRC: true}, chrT.cacheDir)
	// Delete the local cache file.
	err := os.Remove(chrT.chTestArgs.downloadPath)
	require.NoError(chrT.T(), err)

	// There is a fileInfoEntry in the fileInfoCache but the corresponding local file doesn't exist.
	// Hence, this will return error containing util.FileNotPresentInCacheErrMsg.
	err = chrT.chTestArgs.cacheHandler.addFileInfoEntryAndCreateDownloadJob(chrT.chTestArgs.object, chrT.chTestArgs.bucket)

	assert.ErrorContains(chrT.T(), err, util.FileNotPresentInCacheErrMsg)
}

func (chrT *CacheHandlerTest) Test_addFileInfoEntryAndCreateDownloadJob_WhenJobHasCompleted() {
	chrT.chTestArgs = setupHelper(chrT.T(), &config.FileCacheConfig{EnableCRC: true}, chrT.cacheDir)
	existingJob := getDownloadJobForTestObject(chrT.T(), chrT.chTestArgs)
	// Make the job completed, so it's removed from job manager.
	jobStatus, err := existingJob.Download(context.Background(), int64(chrT.chTestArgs.object.Size), true)
	require.NoError(chrT.T(), err)
	require.Equal(chrT.T(), int64(chrT.chTestArgs.object.Size), jobStatus.Offset)
	// Give time for execution of callback to remove from job manager
	time.Sleep(time.Second)
	actualJob := chrT.chTestArgs.jobManager.GetJob(chrT.chTestArgs.object.Name, chrT.chTestArgs.bucket.Name())
	require.Nil(chrT.T(), actualJob)

	err = chrT.chTestArgs.cacheHandler.addFileInfoEntryAndCreateDownloadJob(chrT.chTestArgs.object, chrT.chTestArgs.bucket)

	assert.NoError(chrT.T(), err)
	assert.True(chrT.T(), isEntryInFileInfoCache(chrT.T(), chrT.chTestArgs.cache, chrT.chTestArgs.object.Name, chrT.chTestArgs.bucket.Name()))
	// No new job should be added to job manager
	actualJob = chrT.chTestArgs.jobManager.GetJob(chrT.chTestArgs.object.Name, chrT.chTestArgs.bucket.Name())
	assert.Nil(chrT.T(), actualJob)
}

func (chrT *CacheHandlerTest) Test_addFileInfoEntryAndCreateDownloadJob_WhenJobIsInvalidatedAndRemoved() {
	chrT.chTestArgs = setupHelper(chrT.T(), &config.FileCacheConfig{EnableCRC: true}, chrT.cacheDir)
	chrT.chTestArgs.jobManager.InvalidateAndRemoveJob(chrT.chTestArgs.object.Name, chrT.chTestArgs.bucket.Name())
	existingJob := chrT.chTestArgs.jobManager.GetJob(chrT.chTestArgs.object.Name, chrT.chTestArgs.bucket.Name())
	require.Nil(chrT.T(), existingJob)

	// Because the job has been removed and file info entry is still present, new
	// file info entry and job should be created.
	err := chrT.chTestArgs.cacheHandler.addFileInfoEntryAndCreateDownloadJob(chrT.chTestArgs.object, chrT.chTestArgs.bucket)

	assert.NoError(chrT.T(), err)
	assert.True(chrT.T(), isEntryInFileInfoCache(chrT.T(), chrT.chTestArgs.cache, chrT.chTestArgs.object.Name, chrT.chTestArgs.bucket.Name()))
	// New job should be added to job manager
	actualJob := chrT.chTestArgs.jobManager.GetJob(chrT.chTestArgs.object.Name, chrT.chTestArgs.bucket.Name())
	assert.NotNil(chrT.T(), actualJob)
	assert.Equal(chrT.T(), downloader.NotStarted, actualJob.GetStatus().Name)
}

func (chrT *CacheHandlerTest) Test_addFileInfoEntryAndCreateDownloadJob_WhenJobHasFailed() {
	chrT.chTestArgs = setupHelper(chrT.T(), &config.FileCacheConfig{EnableCRC: true}, chrT.cacheDir)
	existingJob := getDownloadJobForTestObject(chrT.T(), chrT.chTestArgs)
	// Hack to fail the async job
	correctSize := chrT.chTestArgs.object.Size
	chrT.chTestArgs.object.Size = 2
	jobStatus, err := existingJob.Download(context.Background(), 1, true)
	require.NoError(chrT.T(), err)
	require.Equal(chrT.T(), downloader.Failed, jobStatus.Name)
	chrT.chTestArgs.object.Size = correctSize

	// Because the job has been failed and file info entry is still present with
	// size less than the object's size (because the async job failed), new job
	// should be created
	err = chrT.chTestArgs.cacheHandler.addFileInfoEntryAndCreateDownloadJob(chrT.chTestArgs.object, chrT.chTestArgs.bucket)

	assert.NoError(chrT.T(), err)
	assert.True(chrT.T(), isEntryInFileInfoCache(chrT.T(), chrT.chTestArgs.cache, chrT.chTestArgs.object.Name, chrT.chTestArgs.bucket.Name()))
	// New job should be added to job manager
	actualJob := chrT.chTestArgs.jobManager.GetJob(chrT.chTestArgs.object.Name, chrT.chTestArgs.bucket.Name())
	assert.NotNil(chrT.T(), actualJob)
	assert.Equal(chrT.T(), downloader.NotStarted, actualJob.GetStatus().Name)
}

func (chrT *CacheHandlerTest) Test_GetCacheHandle_WhenCacheHasDifferentGeneration() {
	chrT.chTestArgs = setupHelper(chrT.T(), &config.FileCacheConfig{EnableCRC: true}, chrT.cacheDir)
	existingJob := getDownloadJobForTestObject(chrT.T(), chrT.chTestArgs)
	require.NotNil(chrT.T(), existingJob)
	require.Equal(chrT.T(), downloader.NotStarted, existingJob.GetStatus().Name)
	// Change the version of the object, but cache still keeps old generation
	chrT.chTestArgs.object.Generation = chrT.chTestArgs.object.Generation + 1

	newCacheHandle, err := chrT.chTestArgs.cacheHandler.GetCacheHandle(chrT.chTestArgs.object, chrT.chTestArgs.bucket, false, 0)

	assert.NoError(chrT.T(), err)
	assert.Nil(chrT.T(), newCacheHandle.validateCacheHandle())
	jobStatusOfOldJob := existingJob.GetStatus()
	assert.Equal(chrT.T(), downloader.Invalid, jobStatusOfOldJob.Name)
	jobStatusOfNewHandle := newCacheHandle.fileDownloadJob.GetStatus()
	assert.Equal(chrT.T(), downloader.NotStarted, jobStatusOfNewHandle.Name)
}

func (chrT *CacheHandlerTest) Test_GetCacheHandle_WhenAsyncDownloadJobHasFailed() {
	chrT.chTestArgs = setupHelper(chrT.T(), &config.FileCacheConfig{EnableCRC: true}, chrT.cacheDir)
	existingJob := getDownloadJobForTestObject(chrT.T(), chrT.chTestArgs)
	// Hack to fail the async job
	correctSize := chrT.chTestArgs.object.Size
	chrT.chTestArgs.object.Size = 2
	jobStatus, err := existingJob.Download(context.Background(), 1, true)
	require.NoError(chrT.T(), err)
	require.Equal(chrT.T(), downloader.Failed, jobStatus.Name)
	chrT.chTestArgs.object.Size = correctSize

	newCacheHandle, err := chrT.chTestArgs.cacheHandler.GetCacheHandle(chrT.chTestArgs.object, chrT.chTestArgs.bucket, false, 0)

	// New job should be created because the earlier job has failed.
	assert.NoError(chrT.T(), err)
	assert.Nil(chrT.T(), newCacheHandle.validateCacheHandle())
	assert.True(chrT.T(), isEntryInFileInfoCache(chrT.T(), chrT.chTestArgs.cache, chrT.chTestArgs.object.Name, chrT.chTestArgs.bucket.Name()))
	jobStatusOfNewHandle := newCacheHandle.fileDownloadJob.GetStatus()
	assert.Equal(chrT.T(), downloader.NotStarted, jobStatusOfNewHandle.Name)
}

func (chrT *CacheHandlerTest) Test_GetCacheHandle_WhenFileInfoAndJobAreAlreadyPresent() {
	chrT.chTestArgs = setupHelper(chrT.T(), &config.FileCacheConfig{EnableCRC: true}, chrT.cacheDir)
	// File info and download job are already present for test object.
	existingJob := getDownloadJobForTestObject(chrT.T(), chrT.chTestArgs)

	cacheHandle, err := chrT.chTestArgs.cacheHandler.GetCacheHandle(chrT.chTestArgs.object, chrT.chTestArgs.bucket, false, 0)

	assert.NoError(chrT.T(), err)
	assert.Nil(chrT.T(), cacheHandle.validateCacheHandle())
	// Job and file info are still present
	assert.True(chrT.T(), isEntryInFileInfoCache(chrT.T(), chrT.chTestArgs.cache, chrT.chTestArgs.object.Name, chrT.chTestArgs.bucket.Name()))
	assert.Equal(chrT.T(), existingJob, cacheHandle.fileDownloadJob)
	jobStatusOfNewHandle := cacheHandle.fileDownloadJob.GetStatus()
	assert.Equal(chrT.T(), downloader.NotStarted, jobStatusOfNewHandle.Name)
}

func (chrT *CacheHandlerTest) Test_GetCacheHandle_WhenFileInfoAndJobAreNotPresent() {
	chrT.chTestArgs = setupHelper(chrT.T(), &config.FileCacheConfig{EnableCRC: true}, chrT.cacheDir)
	minObject := createObject(chrT.T(), chrT.chTestArgs.bucket, "object_1", []byte("content of object_1"))

	cacheHandle, err := chrT.chTestArgs.cacheHandler.GetCacheHandle(minObject, chrT.chTestArgs.bucket, false, 0)

	assert.NoError(chrT.T(), err)
	assert.Nil(chrT.T(), cacheHandle.validateCacheHandle())
	// New Job and file info are created.
	assert.True(chrT.T(), isEntryInFileInfoCache(chrT.T(), chrT.chTestArgs.cache, minObject.Name, chrT.chTestArgs.bucket.Name()))
	jobStatusOfNewHandle := cacheHandle.fileDownloadJob.GetStatus()
	assert.Equal(chrT.T(), downloader.NotStarted, jobStatusOfNewHandle.Name)
}

func (chrT *CacheHandlerTest) Test_GetCacheHandle_WithEviction() {
	tbl := []struct {
		name            string
		fileCacheConfig config.FileCacheConfig
		cacheDir        string
	}{
		{
			name:            "Non parallel downloads",
			fileCacheConfig: config.FileCacheConfig{EnableCRC: true},
			cacheDir:        path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
		},
		{
			name: "Parallel downloads",
			fileCacheConfig: config.FileCacheConfig{EnableCRC: true, EnableParallelDownloads: true,
				ParallelDownloadsPerFile: 4, MaxParallelDownloads: 20, DownloadChunkSizeMB: 3},
			cacheDir: path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
		},
	}
	for _, tc := range tbl {
		chrT.T().Run(tc.name, func(t *testing.T) {
			chrT.chTestArgs = setupHelper(chrT.T(), &tc.fileCacheConfig, tc.cacheDir)
			// Start the existing job
			existingJob := getDownloadJobForTestObject(chrT.T(), chrT.chTestArgs)
			_, err := existingJob.Download(context.Background(), 1, false)
			require.NoError(chrT.T(), err)
			// Content of size more than 20 leads to eviction of initial TestObjectName.
			// Here, content size is 21.
			minObject := createObject(chrT.T(), chrT.chTestArgs.bucket, "object_1", []byte("content of object_1 ..."))

			cacheHandle2, err := chrT.chTestArgs.cacheHandler.GetCacheHandle(minObject, chrT.chTestArgs.bucket, false, 0)

			assert.NoError(chrT.T(), err)
			assert.Nil(chrT.T(), cacheHandle2.validateCacheHandle())
			jobStatus := existingJob.GetStatus()
			assert.Equal(chrT.T(), downloader.Invalid, jobStatus.Name)
			assert.False(chrT.T(), doesFileExist(chrT.T(), chrT.chTestArgs.downloadPath))
		})
	}
}

func (chrT *CacheHandlerTest) Test_GetCacheHandle_IfLocalFileGetsDeleted() {
	chrT.chTestArgs = setupHelper(chrT.T(), &config.FileCacheConfig{EnableCRC: true}, chrT.cacheDir)
	// Delete the local cache file.
	err := os.Remove(chrT.chTestArgs.downloadPath)
	require.NoError(chrT.T(), err)
	existingJob := getDownloadJobForTestObject(chrT.T(), chrT.chTestArgs)

	cacheHandle, err := chrT.chTestArgs.cacheHandler.GetCacheHandle(chrT.chTestArgs.object, chrT.chTestArgs.bucket, false, 0)

	assert.ErrorContains(chrT.T(), err, util.FileNotPresentInCacheErrMsg)
	assert.Nil(chrT.T(), cacheHandle)
	// Check file info and download job are not removed
	assert.True(chrT.T(), isEntryInFileInfoCache(chrT.T(), chrT.chTestArgs.cache, chrT.chTestArgs.object.Name, chrT.chTestArgs.bucket.Name()))
	actualJob := chrT.chTestArgs.jobManager.GetJob(chrT.chTestArgs.object.Name, chrT.chTestArgs.bucket.Name())
	assert.Equal(chrT.T(), existingJob, actualJob)
	assert.Equal(chrT.T(), downloader.NotStarted, existingJob.GetStatus().Name)
}

func (chrT *CacheHandlerTest) Test_GetCacheHandle_CacheForRangeRead() {
	tbl := []struct {
		name            string
		fileCacheConfig config.FileCacheConfig
		cacheDir        string
	}{
		{
			name:            "Non parallel downloads",
			fileCacheConfig: config.FileCacheConfig{EnableCRC: true},
			cacheDir:        path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
		},
		{
			name: "Parallel downloads",
			fileCacheConfig: config.FileCacheConfig{EnableCRC: true, EnableParallelDownloads: true,
				ParallelDownloadsPerFile: 4, MaxParallelDownloads: 20, DownloadChunkSizeMB: 3},
			cacheDir: path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
		},
	}
	for _, tc := range tbl {
		chrT.T().Run(tc.name, func(t *testing.T) {
			chrT.chTestArgs = setupHelper(chrT.T(), &tc.fileCacheConfig, tc.cacheDir)
			minObject1 := createObject(chrT.T(), chrT.chTestArgs.bucket, "object_1", []byte("content of object_1 ..."))
			cacheHandle1, err1 := chrT.chTestArgs.cacheHandler.GetCacheHandle(minObject1, chrT.chTestArgs.bucket, false, 0)
			minObject2 := createObject(chrT.T(), chrT.chTestArgs.bucket, "object_2", []byte("content of object_2 ..."))
			cacheHandle2, err2 := chrT.chTestArgs.cacheHandler.GetCacheHandle(minObject2, chrT.chTestArgs.bucket, false, 5)
			minObject3 := createObject(chrT.T(), chrT.chTestArgs.bucket, "object_3", []byte("content of object_3 ..."))
			cacheHandle3, err3 := chrT.chTestArgs.cacheHandler.GetCacheHandle(minObject3, chrT.chTestArgs.bucket, true, 0)
			minObject4 := createObject(chrT.T(), chrT.chTestArgs.bucket, "object_4", []byte("content of object_4 ..."))
			cacheHandle4, err4 := chrT.chTestArgs.cacheHandler.GetCacheHandle(minObject4, chrT.chTestArgs.bucket, true, 5)

			assert.NoError(chrT.T(), err1)
			assert.Nil(chrT.T(), cacheHandle1.validateCacheHandle())
			assert.ErrorContains(chrT.T(), err2, util.CacheHandleNotRequiredForRandomReadErrMsg)
			assert.Nil(chrT.T(), cacheHandle2)
			assert.NoError(chrT.T(), err3)
			assert.Nil(chrT.T(), cacheHandle3.validateCacheHandle())
			assert.NoError(chrT.T(), err4)
			assert.Nil(chrT.T(), cacheHandle4.validateCacheHandle())
		})
	}
}

func (chrT *CacheHandlerTest) Test_GetCacheHandle_ConcurrentSameFile() {
	tbl := []struct {
		name                      string
		fileCacheConfig           config.FileCacheConfig
		cacheDir                  string
		expectedGetCacheHandleErr error
	}{
		{
			name:                      "Non parallel downloads",
			fileCacheConfig:           config.FileCacheConfig{EnableCRC: true},
			cacheDir:                  path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
			expectedGetCacheHandleErr: nil,
		},
		{
			name: "Parallel downloads",
			fileCacheConfig: config.FileCacheConfig{EnableCRC: true, EnableParallelDownloads: true,
				ParallelDownloadsPerFile: 1, MaxParallelDownloads: 20, DownloadChunkSizeMB: 3},
			cacheDir:                  path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
			expectedGetCacheHandleErr: nil,
		},
	}
	for _, tc := range tbl {
		chrT.T().Run(tc.name, func(t *testing.T) {
			chrT.chTestArgs = setupHelper(chrT.T(), &tc.fileCacheConfig, tc.cacheDir)
			// Check async job and file info cache not preset for object_1
			testObjectName := "object_1"
			existingJob := chrT.chTestArgs.jobManager.GetJob(testObjectName, chrT.chTestArgs.bucket.Name())
			require.Nil(chrT.T(), existingJob)
			wg := sync.WaitGroup{}
			getCacheHandleTestFun := func() {
				defer wg.Done()
				minObj := createObject(chrT.T(), chrT.chTestArgs.bucket, testObjectName, []byte("content of object_1 ..."))

				var err error
				cacheHandle, err := chrT.chTestArgs.cacheHandler.GetCacheHandle(minObj, chrT.chTestArgs.bucket, false, 0)

				assert.ErrorIs(chrT.T(), err, tc.expectedGetCacheHandleErr)
				assert.Nil(chrT.T(), cacheHandle.validateCacheHandle())
			}

			// Start concurrent GetCacheHandle()
			for i := 0; i < 5; i++ {
				wg.Add(1)
				go getCacheHandleTestFun()
			}
			wg.Wait()

			// Job should be added now
			actualJob := chrT.chTestArgs.jobManager.GetJob(testObjectName, chrT.chTestArgs.bucket.Name())
			jobStatus := actualJob.GetStatus()
			assert.Equal(chrT.T(), downloader.NotStarted, jobStatus.Name)
			assert.True(chrT.T(), doesFileExist(chrT.T(), util.GetDownloadPath(chrT.cacheDir,
				util.GetObjectPath(chrT.chTestArgs.bucket.Name(), testObjectName))))
		})
	}
}

func (chrT *CacheHandlerTest) Test_GetCacheHandle_ConcurrentDifferentFiles() {
	chrT.chTestArgs = setupHelper(chrT.T(), &config.FileCacheConfig{EnableCRC: true}, chrT.cacheDir)
	existingJob := getDownloadJobForTestObject(chrT.T(), chrT.chTestArgs)
	require.Equal(chrT.T(), downloader.NotStarted, existingJob.GetStatus().Name)
	wg := sync.WaitGroup{}

	getCacheHandleTestFun := func(index int) {
		defer wg.Done()
		objName := "object" + strconv.Itoa(index)
		objContent := "object content: content#" + strconv.Itoa(index)
		minObj := createObject(chrT.T(), chrT.chTestArgs.bucket, objName, []byte(objContent))

		cacheHandle, err := chrT.chTestArgs.cacheHandler.GetCacheHandle(minObj, chrT.chTestArgs.bucket, false, 0)

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
	assert.Equal(chrT.T(), false, doesFileExist(chrT.T(), chrT.chTestArgs.downloadPath))
	// File info should also be removed.
	assert.False(chrT.T(), isEntryInFileInfoCache(chrT.T(), chrT.chTestArgs.cache, chrT.chTestArgs.object.Name, chrT.chTestArgs.bucket.Name()))
}

func (chrT *CacheHandlerTest) Test_InvalidateCache_WhenAlreadyInCache() {
	chrT.chTestArgs = setupHelper(chrT.T(), &config.FileCacheConfig{EnableCRC: true}, chrT.cacheDir)
	existingJob := getDownloadJobForTestObject(chrT.T(), chrT.chTestArgs)
	require.Equal(chrT.T(), downloader.NotStarted, existingJob.GetStatus().Name)
	require.True(chrT.T(), isEntryInFileInfoCache(chrT.T(), chrT.chTestArgs.cache, chrT.chTestArgs.object.Name, chrT.chTestArgs.bucket.Name()))

	err := chrT.chTestArgs.cacheHandler.InvalidateCache(chrT.chTestArgs.object.Name, chrT.chTestArgs.bucket.Name())

	assert.NoError(chrT.T(), err)
	// Existing job for default chrT object should be invalidated.
	assert.NotNil(chrT.T(), existingJob)
	assert.Equal(chrT.T(), downloader.Invalid, existingJob.GetStatus().Name)
	assert.Equal(chrT.T(), false, doesFileExist(chrT.T(), chrT.chTestArgs.downloadPath))
	// File info should also be removed.
	assert.False(chrT.T(), isEntryInFileInfoCache(chrT.T(), chrT.chTestArgs.cache, chrT.chTestArgs.object.Name, chrT.chTestArgs.bucket.Name()))
}

func (chrT *CacheHandlerTest) Test_InvalidateCache_WhenEntryNotInCache() {
	chrT.chTestArgs = setupHelper(chrT.T(), &config.FileCacheConfig{EnableCRC: true}, chrT.cacheDir)
	minObject := createObject(chrT.T(), chrT.chTestArgs.bucket, "object_1", []byte("content of object_1"))
	require.False(chrT.T(), isEntryInFileInfoCache(chrT.T(), chrT.chTestArgs.cache, minObject.Name, chrT.chTestArgs.bucket.Name()))
	require.Nil(chrT.T(), chrT.chTestArgs.jobManager.GetJob(minObject.Name, chrT.chTestArgs.bucket.Name()))

	err := chrT.chTestArgs.cacheHandler.InvalidateCache(minObject.Name, chrT.chTestArgs.bucket.Name())

	assert.NoError(chrT.T(), err)
	assert.False(chrT.T(), isEntryInFileInfoCache(chrT.T(), chrT.chTestArgs.cache, minObject.Name, chrT.chTestArgs.bucket.Name()))
	assert.Nil(chrT.T(), chrT.chTestArgs.jobManager.GetJob(minObject.Name, chrT.chTestArgs.bucket.Name()))
}

func (chrT *CacheHandlerTest) Test_InvalidateCache_Truncates() {
	tbl := []struct {
		name                        string
		fileCacheConfig             config.FileCacheConfig
		cacheDir                    string
		expectedCInvalidateCacheErr error
		expectedCacheHandleReadErr  error
		expectedCacheFileReadErr    error
	}{
		{
			name:                        "Non parallel downloads",
			fileCacheConfig:             config.FileCacheConfig{EnableCRC: true},
			cacheDir:                    path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
			expectedCInvalidateCacheErr: nil,
			expectedCacheHandleReadErr:  nil,
			expectedCacheFileReadErr:    io.EOF,
		},
		{
			name: "Parallel downloads",
			fileCacheConfig: config.FileCacheConfig{EnableCRC: true, EnableParallelDownloads: true,
				ParallelDownloadsPerFile: 4, MaxParallelDownloads: 20, DownloadChunkSizeMB: 3},
			cacheDir:                    path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
			expectedCInvalidateCacheErr: nil,
			expectedCacheHandleReadErr:  fmt.Errorf("read via gcs"),
			expectedCacheFileReadErr:    io.EOF,
		},
	}
	for _, tc := range tbl {
		chrT.T().Run(tc.name, func(t *testing.T) {
			chrT.chTestArgs = setupHelper(chrT.T(), &tc.fileCacheConfig, tc.cacheDir)
			objectContent := []byte("content of object_1")
			minObject := createObject(chrT.T(), chrT.chTestArgs.bucket, "object_1", objectContent)
			cacheHandle, err := chrT.chTestArgs.cacheHandler.GetCacheHandle(minObject, chrT.chTestArgs.bucket, false, 0)
			require.NoError(chrT.T(), err)
			buf := make([]byte, 3)
			ctx := context.Background()
			// Read to populate cache
			_, cacheHit, err := cacheHandle.Read(ctx, chrT.chTestArgs.bucket, minObject, 0, buf)
			if tc.expectedCacheHandleReadErr == nil {
				require.NoError(chrT.T(), err)
				require.Equal(chrT.T(), string(objectContent[:3]), string(buf))
			} else {
				require.ErrorContains(chrT.T(), err, tc.expectedCacheHandleReadErr.Error())
			}
			require.Equal(chrT.T(), false, cacheHit)
			require.Equal(chrT.T(), nil, cacheHandle.Close())
			// Open cache file before invalidation
			objectPath := util.GetObjectPath(chrT.chTestArgs.bucket.Name(), minObject.Name)
			downloadPath := util.GetDownloadPath(chrT.cacheDir, objectPath)
			file, err := os.OpenFile(downloadPath, os.O_RDONLY, 0600)
			require.NoError(chrT.T(), err)

			err = chrT.chTestArgs.cacheHandler.InvalidateCache(minObject.Name, chrT.chTestArgs.bucket.Name())

			assert.ErrorIs(t, err, tc.expectedCInvalidateCacheErr)
			// Reading from the open file handle should fail as the file is truncated.
			_, err = file.Read(buf)
			assert.ErrorIs(chrT.T(), err, tc.expectedCacheFileReadErr)
		})
	}
}

func (chrT *CacheHandlerTest) Test_InvalidateCache_ConcurrentSameFile() {
	tbl := []struct {
		name                       string
		fileCacheConfig            config.FileCacheConfig
		cacheDir                   string
		expectedInvalidateCacheErr error
	}{
		{
			name:                       "Non parallel downloads",
			fileCacheConfig:            config.FileCacheConfig{EnableCRC: true},
			cacheDir:                   path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
			expectedInvalidateCacheErr: nil,
		},
		{
			name: "Parallel downloads",
			fileCacheConfig: config.FileCacheConfig{EnableCRC: true, EnableParallelDownloads: true,
				ParallelDownloadsPerFile: 1, MaxParallelDownloads: 20, DownloadChunkSizeMB: 3},
			cacheDir:                   path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
			expectedInvalidateCacheErr: nil,
		},
	}
	for _, tc := range tbl {
		chrT.T().Run(tc.name, func(t *testing.T) {
			chrT.chTestArgs = setupHelper(chrT.T(), &tc.fileCacheConfig, tc.cacheDir)
			existingJob := getDownloadJobForTestObject(chrT.T(), chrT.chTestArgs)
			require.Equal(chrT.T(), downloader.NotStarted, existingJob.GetStatus().Name)
			require.True(chrT.T(), isEntryInFileInfoCache(chrT.T(), chrT.chTestArgs.cache, chrT.chTestArgs.object.Name, chrT.chTestArgs.bucket.Name()))
			wg := sync.WaitGroup{}
			InvalidateCacheTestFun := func() {
				defer wg.Done()

				err := chrT.chTestArgs.cacheHandler.InvalidateCache(chrT.chTestArgs.object.Name, chrT.chTestArgs.bucket.Name())

				assert.ErrorIs(chrT.T(), err, tc.expectedInvalidateCacheErr)
				assert.NotNil(chrT.T(), existingJob)
				assert.Equal(chrT.T(), downloader.Invalid, existingJob.GetStatus().Name)
				assert.Equal(chrT.T(), false, doesFileExist(chrT.T(), chrT.chTestArgs.downloadPath))
				// File info should also be removed.
				assert.False(chrT.T(), isEntryInFileInfoCache(chrT.T(), chrT.chTestArgs.cache, chrT.chTestArgs.object.Name, chrT.chTestArgs.bucket.Name()))
			}

			// Start concurrent GetCacheHandle()
			for i := 0; i < 5; i++ {
				wg.Add(1)
				go InvalidateCacheTestFun()
			}
			wg.Wait()
		})
	}
}

func (chrT *CacheHandlerTest) Test_InvalidateCache_ConcurrentDifferentFiles() {
	tbl := []struct {
		name                       string
		fileCacheConfig            config.FileCacheConfig
		cacheDir                   string
		expectedInvalidateCacheErr error
	}{
		{
			name:                       "Non parallel downloads",
			fileCacheConfig:            config.FileCacheConfig{EnableCRC: true},
			cacheDir:                   path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
			expectedInvalidateCacheErr: nil,
		},
		{
			name: "Parallel downloads",
			fileCacheConfig: config.FileCacheConfig{EnableCRC: true, EnableParallelDownloads: true,
				ParallelDownloadsPerFile: 1, MaxParallelDownloads: 20, DownloadChunkSizeMB: 3},
			cacheDir:                   path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
			expectedInvalidateCacheErr: nil,
		},
	}
	for _, tc := range tbl {
		chrT.T().Run(tc.name, func(t *testing.T) {
			chrT.chTestArgs = setupHelper(chrT.T(), &tc.fileCacheConfig, tc.cacheDir)
			wg := sync.WaitGroup{}
			InvalidateCacheTestFun := func(index int) {
				defer wg.Done()
				objName := "object" + strconv.Itoa(index)
				objContent := "object content: content#" + strconv.Itoa(index)
				minObj := createObject(chrT.T(), chrT.chTestArgs.bucket, objName, []byte(objContent))

				err := chrT.chTestArgs.cacheHandler.InvalidateCache(minObj.Name, chrT.chTestArgs.bucket.Name())

				assert.ErrorIs(chrT.T(), err, tc.expectedInvalidateCacheErr)
				assert.Nil(chrT.T(), chrT.chTestArgs.jobManager.GetJob(objName, chrT.chTestArgs.bucket.Name()))
				assert.False(chrT.T(), isEntryInFileInfoCache(chrT.T(), chrT.chTestArgs.cache, objName, chrT.chTestArgs.bucket.Name()))
			}

			// Start concurrent GetCacheHandle()
			for i := 0; i < 5; i++ {
				wg.Add(1)
				go InvalidateCacheTestFun(i)
			}
			wg.Wait()
		})
	}
}

func (chrT *CacheHandlerTest) Test_InvalidateCache_GetCacheHandle_Concurrent() {
	tbl := []struct {
		name                   string
		fileCacheConfig        config.FileCacheConfig
		cacheDir               string
		expectedCacheHandleErr error
	}{
		{
			name:                   "Non parallel downloads",
			fileCacheConfig:        config.FileCacheConfig{EnableCRC: true},
			cacheDir:               path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
			expectedCacheHandleErr: nil,
		},
		{
			name: "Parallel downloads",
			fileCacheConfig: config.FileCacheConfig{EnableCRC: true, EnableParallelDownloads: true,
				ParallelDownloadsPerFile: 1, MaxParallelDownloads: 20, DownloadChunkSizeMB: 3},
			cacheDir:               path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
			expectedCacheHandleErr: nil,
		},
	}
	for _, tc := range tbl {
		chrT.T().Run(tc.name, func(t *testing.T) {
			chrT.chTestArgs = setupHelper(chrT.T(), &tc.fileCacheConfig, tc.cacheDir)
			wg := sync.WaitGroup{}
			invalidateCacheTestFun := func(index int) {
				defer wg.Done()
				objName := "object" + strconv.Itoa(index)
				objContent := "object content: content#" + strconv.Itoa(index)
				minObj := createObject(chrT.T(), chrT.chTestArgs.bucket, objName, []byte(objContent))

				err := chrT.chTestArgs.cacheHandler.InvalidateCache(minObj.Name, chrT.chTestArgs.bucket.Name())

				assert.NoError(chrT.T(), err)
			}

			getCacheHandleTestFun := func(index int) {
				defer wg.Done()
				objName := "object" + strconv.Itoa(index)
				objContent := "object content: content#" + strconv.Itoa(index)
				minObj := createObject(chrT.T(), chrT.chTestArgs.bucket, objName, []byte(objContent))

				cacheHandle, err := chrT.chTestArgs.cacheHandler.GetCacheHandle(minObj, chrT.chTestArgs.bucket, false, 0)

				assert.ErrorIs(chrT.T(), err, tc.expectedCacheHandleErr)
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
		})
	}
}

func (chrT *CacheHandlerTest) Test_Destroy() {
	tbl := []struct {
		name                   string
		fileCacheConfig        config.FileCacheConfig
		cacheDir               string
		expectedCacheHandleErr error
		expectedJobStatus      []string
	}{
		{
			name:                   "Non parallel downloads",
			fileCacheConfig:        config.FileCacheConfig{EnableCRC: true},
			cacheDir:               path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
			expectedCacheHandleErr: nil,
			expectedJobStatus:      []string{string(downloader.Completed)},
		},
		{
			name: "Parallel downloads",
			fileCacheConfig: config.FileCacheConfig{EnableCRC: true, EnableParallelDownloads: true,
				ParallelDownloadsPerFile: 4, MaxParallelDownloads: 20, DownloadChunkSizeMB: 3},
			cacheDir:               path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
			expectedCacheHandleErr: fmt.Errorf("read via gcs"),
			expectedJobStatus:      []string{string(downloader.Completed), string(downloader.Invalid)},
		},
	}
	for _, tc := range tbl {
		chrT.T().Run(tc.name, func(t *testing.T) {
			chrT.chTestArgs = setupHelper(chrT.T(), &tc.fileCacheConfig, tc.cacheDir)
			minObject1 := createObject(chrT.T(), chrT.chTestArgs.bucket, "object_1", []byte("content of object_1"))
			minObject2 := createObject(chrT.T(), chrT.chTestArgs.bucket, "object_2", []byte("content of object_2"))
			cacheHandle1, err := chrT.chTestArgs.cacheHandler.GetCacheHandle(minObject1, chrT.chTestArgs.bucket, true, 0)
			require.NoError(chrT.T(), err)
			cacheHandle2, err := chrT.chTestArgs.cacheHandler.GetCacheHandle(minObject2, chrT.chTestArgs.bucket, true, 0)
			require.NoError(chrT.T(), err)
			ctx := context.Background()
			// Read to create and populate file in cache.
			buf := make([]byte, 3)
			_, cacheHit, err := cacheHandle1.Read(ctx, chrT.chTestArgs.bucket, minObject1, 4, buf)
			if tc.expectedCacheHandleErr == nil {
				require.NoError(chrT.T(), err)
			} else {
				require.ErrorContains(chrT.T(), err, tc.expectedCacheHandleErr.Error())
			}
			require.Equal(chrT.T(), false, cacheHit)
			_, cacheHit, err = cacheHandle2.Read(ctx, chrT.chTestArgs.bucket, minObject2, 4, buf)
			if tc.expectedCacheHandleErr == nil {
				require.NoError(chrT.T(), err)
			} else {
				require.ErrorContains(chrT.T(), err, tc.expectedCacheHandleErr.Error())
			}
			require.Equal(chrT.T(), false, cacheHit)
			err = cacheHandle1.Close()
			require.NoError(chrT.T(), err)
			err = cacheHandle2.Close()
			require.NoError(chrT.T(), err)

			err = chrT.chTestArgs.cacheHandler.Destroy()

			assert.NoError(chrT.T(), err)
			// Verify the cacheDir is deleted.
			_, err = os.Stat(path.Join(chrT.cacheDir, util.FileCache))
			assert.ErrorIs(chrT.T(), err, os.ErrNotExist)
			// Verify jobs statuses.
			job1 := chrT.chTestArgs.jobManager.GetJob(minObject1.Name, chrT.chTestArgs.bucket.Name())
			job2 := chrT.chTestArgs.jobManager.GetJob(minObject1.Name, chrT.chTestArgs.bucket.Name())
			if job1 != nil {
				assert.Contains(chrT.T(), tc.expectedJobStatus, job1.GetStatus().Name)
			}
			if job2 != nil {
				assert.Contains(chrT.T(), tc.expectedJobStatus, job2.GetStatus().Name)
			}
			// Job manager should no longer contain the jobs
			assert.Nil(chrT.T(), chrT.chTestArgs.jobManager.GetJob(minObject1.Name, chrT.chTestArgs.bucket.Name()))
			assert.Nil(chrT.T(), chrT.chTestArgs.jobManager.GetJob(minObject2.Name, chrT.chTestArgs.bucket.Name()))
		})
	}
}
