// Copyright 2023 Google LLC
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
	"os"
	"path"
	"strconv"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/storage/control/apiv2/controlpb"
	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/file/downloader"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const HandlerCacheMaxSize = TestObjectSize + ObjectSizeToCauseEviction
const ObjectSizeToCauseEviction = 20

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

func initializeCacheHandlerTestArgs(t *testing.T, fileCacheConfig *cfg.FileCacheConfig, cacheDir string) *cacheHandlerTestArgs {
	t.Helper()
	locker.EnableInvariantsCheck()

	// Create bucket in fake storage.
	mockClient := new(storage.MockStorageControlClient)
	fakeStorage := storage.NewFakeStorageWithMockClient(mockClient, cfg.HTTP2)
	t.Cleanup(func() {
		fakeStorage.ShutDown()
	})
	storageHandle := fakeStorage.CreateStorageHandle()
	mockClient.On("GetStorageLayout", mock.Anything, mock.Anything, mock.Anything).
		Return(&controlpb.StorageLayout{}, nil)
	ctx := context.Background()
	bucket, err := storageHandle.BucketHandle(ctx, storage.TestBucketName, "", false)
	require.NoError(t, err)

	// Create test object in the bucket.
	testObjectContent := make([]byte, TestObjectSize)
	_, err = rand.Read(testObjectContent)
	require.NoError(t, err)
	object := createObject(t, bucket, TestObjectName, testObjectContent)

	// fileInfoCache with testFileInfoEntry
	cache := lru.NewCache(HandlerCacheMaxSize)

	// Job manager
	jobManager := downloader.NewJobManager(cache, util.DefaultFilePerm,
		util.DefaultDirPerm, cacheDir, DefaultSequentialReadSizeMb, fileCacheConfig, metrics.NewNoopMetrics(), tracing.NewNoopTracer())

	// Mocked cached handler object.
	isSparse := fileCacheConfig != nil && fileCacheConfig.ExperimentalEnableChunkCache
	cacheHandler := NewCacheHandler(cache, jobManager, cacheDir, util.DefaultFilePerm, util.DefaultDirPerm, fileCacheConfig.ExcludeRegex, fileCacheConfig.IncludeRegex, isSparse)

	// Follow consistency, local-cache file, entry in fileInfo cache and job should exist initially.
	fileInfoKeyName := addTestFileInfoEntryInCache(t, cache, object, storage.TestBucketName)
	downloadPath := util.GetDownloadPath(cacheHandler.cacheDir, util.GetObjectPath(bucket.Name(), object.Name))
	_, err = util.CreateFile(data.FileSpec{Path: downloadPath, FilePerm: util.DefaultFilePerm, DirPerm: util.DefaultDirPerm}, os.O_RDONLY)
	t.Cleanup(func() {
		operations.RemoveDir(cacheDir)
	})
	require.NoError(t, err)

	job := jobManager.CreateJobIfNotExists(object, bucket)
	require.NotNil(t, job)

	return &cacheHandlerTestArgs{
		jobManager:      jobManager,
		bucket:          bucket,
		fakeStorage:     fakeStorage,
		object:          object,
		cache:           cache,
		cacheHandler:    cacheHandler,
		downloadPath:    downloadPath,
		fileInfoKeyName: fileInfoKeyName,
		cacheDir:        cacheDir,
	}
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

func addTestFileInfoEntryInCache(t *testing.T, cache *lru.Cache, object *gcs.MinObject, bucketName string) string {
	t.Helper()
	// Add an entry into
	fileInfoKey := data.FileInfoKey{
		BucketName: bucketName,
		ObjectName: object.Name,
	}
	fileInfo := data.FileInfo{
		Key:              fileInfoKey,
		ObjectGeneration: object.Generation,
		FileSize:         object.Size,
		Offset:           0,
	}

	fileInfoKeyName, err := fileInfoKey.Key()
	require.NoError(t, err)

	_, err = cache.Insert(fileInfoKeyName, fileInfo)
	require.NoError(t, err)

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
	require.NoError(t, err)

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
func Test_createLocalFileReadHandle_OnlyForRead(t *testing.T) {
	cacheDir := path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir")
	chTestArgs := initializeCacheHandlerTestArgs(t, &cfg.FileCacheConfig{EnableCrc: true}, cacheDir)

	readFileHandle, err := chTestArgs.cacheHandler.createLocalFileReadHandle(chTestArgs.object.Name, chTestArgs.bucket.Name())

	assert.NoError(t, err)
	_, err = readFileHandle.Write([]byte("test"))
	assert.ErrorContains(t, err, "bad file descriptor")
}

func Test_cleanUpEvictedFile(t *testing.T) {
	tbl := []struct {
		name            string
		fileCacheConfig cfg.FileCacheConfig
		cacheDir        string
	}{
		{
			name:            "Non parallel downloads",
			fileCacheConfig: cfg.FileCacheConfig{EnableCrc: true},
			cacheDir:        path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
		},
		{
			name: "Parallel downloads",
			fileCacheConfig: cfg.FileCacheConfig{
				EnableCrc:                true,
				EnableParallelDownloads:  true,
				ParallelDownloadsPerFile: 4,
				MaxParallelDownloads:     20,
				DownloadChunkSizeMb:      3,
				WriteBufferSize:          4 * 1024 * 1024,
			},
			cacheDir: path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
		},
	}
	for _, tc := range tbl {
		t.Run(tc.name, func(t *testing.T) {
			chTestArgs := initializeCacheHandlerTestArgs(t, &tc.fileCacheConfig, tc.cacheDir)
			fileDownloadJob := getDownloadJobForTestObject(t, chTestArgs)
			fileInfo := chTestArgs.cache.LookUp(chTestArgs.fileInfoKeyName)
			fileInfoData := fileInfo.(data.FileInfo)
			jobStatusBefore := fileDownloadJob.GetStatus()
			require.Equal(t, downloader.NotStarted, jobStatusBefore.Name)
			jobStatusBefore, err := fileDownloadJob.Download(context.Background(), int64(util.MiB), false)
			require.NoError(t, err)
			require.Equal(t, downloader.Downloading, jobStatusBefore.Name)

			err = chTestArgs.cacheHandler.cleanUpEvictedFile(&fileInfoData)

			assert.NoError(t, err)
			jobStatusAfter := fileDownloadJob.GetStatus()
			assert.Equal(t, downloader.Invalid, jobStatusAfter.Name)
			assert.False(t, doesFileExist(t, chTestArgs.downloadPath))
			// Job should be removed from job manager
			assert.Nil(t, chTestArgs.jobManager.GetJob(chTestArgs.object.Name, chTestArgs.bucket.Name()))
		})
	}
}

func Test_cleanUpEvictedFile_WhenLocalFileNotExist(t *testing.T) {
	cacheDir := path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir")
	chTestArgs := initializeCacheHandlerTestArgs(t, &cfg.FileCacheConfig{EnableCrc: true}, cacheDir)
	fileDownloadJob := getDownloadJobForTestObject(t, chTestArgs)
	fileInfo := chTestArgs.cache.LookUp(chTestArgs.fileInfoKeyName)
	fileInfoData := fileInfo.(data.FileInfo)
	jobStatusBefore := fileDownloadJob.GetStatus()
	require.Equal(t, downloader.NotStarted, jobStatusBefore.Name)
	jobStatusBefore, err := fileDownloadJob.Download(context.Background(), int64(util.MiB), false)
	require.NoError(t, err)
	require.Equal(t, downloader.Downloading, jobStatusBefore.Name)
	err = os.Remove(chTestArgs.downloadPath)
	require.NoError(t, err)

	err = chTestArgs.cacheHandler.cleanUpEvictedFile(&fileInfoData)

	assert.NoError(t, err)
	jobStatusAfter := fileDownloadJob.GetStatus()
	assert.Equal(t, downloader.Invalid, jobStatusAfter.Name)
	assert.False(t, doesFileExist(t, chTestArgs.downloadPath))
	// Job should be removed from job manager
	assert.Nil(t, chTestArgs.jobManager.GetJob(chTestArgs.object.Name, chTestArgs.bucket.Name()))
}

func Test_addFileInfoEntryAndCreateDownloadJob_IfAlready(t *testing.T) {
	cacheDir := path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir")
	chTestArgs := initializeCacheHandlerTestArgs(t, &cfg.FileCacheConfig{EnableCrc: true}, cacheDir)
	existingJob := getDownloadJobForTestObject(t, chTestArgs)

	err := chTestArgs.cacheHandler.addFileInfoEntryAndCreateDownloadJob(chTestArgs.object, chTestArgs.bucket)

	assert.NoError(t, err)
	assert.True(t, isEntryInFileInfoCache(t, chTestArgs.cache, chTestArgs.object.Name, chTestArgs.bucket.Name()))
	// File download job should also be same
	actualJob := chTestArgs.jobManager.GetJob(chTestArgs.object.Name, chTestArgs.bucket.Name())
	assert.Equal(t, existingJob, actualJob)
}

func Test_addFileInfoEntryAndCreateDownloadJob_GenerationChanged(t *testing.T) {
	cacheDir := path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir")
	chTestArgs := initializeCacheHandlerTestArgs(t, &cfg.FileCacheConfig{EnableCrc: true}, cacheDir)
	existingJob := getDownloadJobForTestObject(t, chTestArgs)
	chTestArgs.object.Generation = chTestArgs.object.Generation + 1

	err := chTestArgs.cacheHandler.addFileInfoEntryAndCreateDownloadJob(chTestArgs.object, chTestArgs.bucket)

	assert.NoError(t, err)
	assert.True(t, isEntryInFileInfoCache(t, chTestArgs.cache, chTestArgs.object.Name, chTestArgs.bucket.Name()))
	// File download job should be new as the file info and job should be cleaned
	// up.
	actualJob := chTestArgs.jobManager.GetJob(chTestArgs.object.Name, chTestArgs.bucket.Name())
	assert.NotEqual(t, existingJob, actualJob)
}

func Test_addFileInfoEntryAndCreateDownloadJob_IfNotAlready(t *testing.T) {
	cacheDir := path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir")
	chTestArgs := initializeCacheHandlerTestArgs(t, &cfg.FileCacheConfig{EnableCrc: true}, cacheDir)
	oldJob := getDownloadJobForTestObject(t, chTestArgs)
	// Content of size more than 20 leads to eviction of initial TestObjectName.
	// Here, content size is 21.
	minObject := createObject(t, chTestArgs.bucket, "object_1", []byte("content of object_1 ..."))
	// There should be no file download job corresponding to minObject
	existingJob := chTestArgs.jobManager.GetJob(minObject.Name, chTestArgs.bucket.Name())
	require.Nil(t, existingJob)

	// Insertion will happen and that leads to eviction.
	err := chTestArgs.cacheHandler.addFileInfoEntryAndCreateDownloadJob(minObject, chTestArgs.bucket)

	assert.NoError(t, err)
	assert.True(t, isEntryInFileInfoCache(t, chTestArgs.cache, minObject.Name, chTestArgs.bucket.Name()))
	jobStatus := oldJob.GetStatus()
	assert.Equal(t, downloader.Invalid, jobStatus.Name)
	assert.False(t, doesFileExist(t, chTestArgs.downloadPath))
	// Job should be added for minObject
	minObjectJob := chTestArgs.jobManager.GetJob(minObject.Name, chTestArgs.bucket.Name())
	assert.NotNil(t, minObjectJob)
	assert.Equal(t, downloader.NotStarted, minObjectJob.GetStatus().Name)
}

func Test_addFileInfoEntryAndCreateDownloadJob_IfLocalFileGetsDeleted(t *testing.T) {
	cacheDir := path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir")
	chTestArgs := initializeCacheHandlerTestArgs(t, &cfg.FileCacheConfig{EnableCrc: true}, cacheDir)
	// Delete the local cache file.
	err := os.Remove(chTestArgs.downloadPath)
	require.NoError(t, err)

	// There is a fileInfoEntry in the fileInfoCache but the corresponding local file doesn't exist.
	// Hence, this will return error containing util.FileNotPresentInCacheErrMsg.
	err = chTestArgs.cacheHandler.addFileInfoEntryAndCreateDownloadJob(chTestArgs.object, chTestArgs.bucket)

	assert.True(t, errors.Is(err, util.ErrFileNotPresentInCache))
}

func Test_addFileInfoEntryAndCreateDownloadJob_WhenJobHasCompleted(t *testing.T) {
	cacheDir := path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir")
	chTestArgs := initializeCacheHandlerTestArgs(t, &cfg.FileCacheConfig{EnableCrc: true}, cacheDir)
	existingJob := getDownloadJobForTestObject(t, chTestArgs)
	// Make the job completed, so it's removed from job manager.
	jobStatus, err := existingJob.Download(context.Background(), int64(chTestArgs.object.Size), true)
	require.NoError(t, err)
	require.Equal(t, int64(chTestArgs.object.Size), jobStatus.Offset)
	// Give time for execution of callback to remove from job manager
	time.Sleep(time.Second)
	actualJob := chTestArgs.jobManager.GetJob(chTestArgs.object.Name, chTestArgs.bucket.Name())
	require.Nil(t, actualJob)

	err = chTestArgs.cacheHandler.addFileInfoEntryAndCreateDownloadJob(chTestArgs.object, chTestArgs.bucket)

	assert.NoError(t, err)
	assert.True(t, isEntryInFileInfoCache(t, chTestArgs.cache, chTestArgs.object.Name, chTestArgs.bucket.Name()))
	// No new job should be added to job manager
	actualJob = chTestArgs.jobManager.GetJob(chTestArgs.object.Name, chTestArgs.bucket.Name())
	assert.Nil(t, actualJob)
}

func Test_addFileInfoEntryAndCreateDownloadJob_WhenJobIsInvalidatedAndRemoved(t *testing.T) {
	cacheDir := path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir")
	chTestArgs := initializeCacheHandlerTestArgs(t, &cfg.FileCacheConfig{EnableCrc: true}, cacheDir)
	chTestArgs.jobManager.InvalidateAndRemoveJob(chTestArgs.object.Name, chTestArgs.bucket.Name())
	existingJob := chTestArgs.jobManager.GetJob(chTestArgs.object.Name, chTestArgs.bucket.Name())
	require.Nil(t, existingJob)

	// Because the job has been removed and file info entry is still present, new
	// file info entry and job should be created.
	err := chTestArgs.cacheHandler.addFileInfoEntryAndCreateDownloadJob(chTestArgs.object, chTestArgs.bucket)

	assert.NoError(t, err)
	assert.True(t, isEntryInFileInfoCache(t, chTestArgs.cache, chTestArgs.object.Name, chTestArgs.bucket.Name()))
	// New job should be added to job manager
	actualJob := chTestArgs.jobManager.GetJob(chTestArgs.object.Name, chTestArgs.bucket.Name())
	assert.NotNil(t, actualJob)
	assert.Equal(t, downloader.NotStarted, actualJob.GetStatus().Name)
}

func Test_addFileInfoEntryAndCreateDownloadJob_WhenJobHasFailed(t *testing.T) {
	cacheDir := path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir")
	chTestArgs := initializeCacheHandlerTestArgs(t, &cfg.FileCacheConfig{EnableCrc: true}, cacheDir)
	existingJob := getDownloadJobForTestObject(t, chTestArgs)
	// Hack to fail the async job
	correctSize := chTestArgs.object.Size
	chTestArgs.object.Size = 2
	jobStatus, err := existingJob.Download(context.Background(), 1, true)
	require.NoError(t, err)
	require.Equal(t, downloader.Failed, jobStatus.Name)
	chTestArgs.object.Size = correctSize

	// Because the job has been failed and file info entry is still present with
	// size less than the object's size (because the async job failed), new job
	// should be created
	err = chTestArgs.cacheHandler.addFileInfoEntryAndCreateDownloadJob(chTestArgs.object, chTestArgs.bucket)

	assert.NoError(t, err)
	assert.True(t, isEntryInFileInfoCache(t, chTestArgs.cache, chTestArgs.object.Name, chTestArgs.bucket.Name()))
	// New job should be added to job manager
	actualJob := chTestArgs.jobManager.GetJob(chTestArgs.object.Name, chTestArgs.bucket.Name())
	assert.NotNil(t, actualJob)
	assert.Equal(t, downloader.NotStarted, actualJob.GetStatus().Name)
}

func Test_GetCacheHandle_WhenCacheHasDifferentGeneration(t *testing.T) {
	cacheDir := path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir")
	chTestArgs := initializeCacheHandlerTestArgs(t, &cfg.FileCacheConfig{EnableCrc: true}, cacheDir)
	existingJob := getDownloadJobForTestObject(t, chTestArgs)
	require.NotNil(t, existingJob)
	require.Equal(t, downloader.NotStarted, existingJob.GetStatus().Name)
	// Change the version of the object, but cache still keeps old generation
	chTestArgs.object.Generation = chTestArgs.object.Generation + 1

	newCacheHandle, err := chTestArgs.cacheHandler.GetCacheHandle(chTestArgs.object, chTestArgs.bucket, false, 0)

	assert.NoError(t, err)
	assert.Nil(t, newCacheHandle.validateCacheHandle())
	jobStatusOfOldJob := existingJob.GetStatus()
	assert.Equal(t, downloader.Invalid, jobStatusOfOldJob.Name)
	jobStatusOfNewHandle := newCacheHandle.fileDownloadJob.GetStatus()
	assert.Equal(t, downloader.NotStarted, jobStatusOfNewHandle.Name)
}

func Test_GetCacheHandle_WhenAsyncDownloadJobHasFailed(t *testing.T) {
	cacheDir := path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir")
	chTestArgs := initializeCacheHandlerTestArgs(t, &cfg.FileCacheConfig{EnableCrc: true}, cacheDir)
	existingJob := getDownloadJobForTestObject(t, chTestArgs)
	// Hack to fail the async job
	correctSize := chTestArgs.object.Size
	chTestArgs.object.Size = 2
	jobStatus, err := existingJob.Download(context.Background(), 1, true)
	require.NoError(t, err)
	require.Equal(t, downloader.Failed, jobStatus.Name)
	chTestArgs.object.Size = correctSize

	newCacheHandle, err := chTestArgs.cacheHandler.GetCacheHandle(chTestArgs.object, chTestArgs.bucket, false, 0)

	// New job should be created because the earlier job has failed.
	assert.NoError(t, err)
	assert.Nil(t, newCacheHandle.validateCacheHandle())
	assert.True(t, isEntryInFileInfoCache(t, chTestArgs.cache, chTestArgs.object.Name, chTestArgs.bucket.Name()))
	jobStatusOfNewHandle := newCacheHandle.fileDownloadJob.GetStatus()
	assert.Equal(t, downloader.NotStarted, jobStatusOfNewHandle.Name)
}

func Test_GetCacheHandle_WhenFileInfoAndJobAreAlreadyPresent(t *testing.T) {
	cacheDir := path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir")
	chTestArgs := initializeCacheHandlerTestArgs(t, &cfg.FileCacheConfig{EnableCrc: true}, cacheDir)
	// File info and download job are already present for test object.
	existingJob := getDownloadJobForTestObject(t, chTestArgs)

	cacheHandle, err := chTestArgs.cacheHandler.GetCacheHandle(chTestArgs.object, chTestArgs.bucket, false, 0)

	assert.NoError(t, err)
	assert.Nil(t, cacheHandle.validateCacheHandle())
	// Job and file info are still present
	assert.True(t, isEntryInFileInfoCache(t, chTestArgs.cache, chTestArgs.object.Name, chTestArgs.bucket.Name()))
	assert.Equal(t, existingJob, cacheHandle.fileDownloadJob)
	jobStatusOfNewHandle := cacheHandle.fileDownloadJob.GetStatus()
	assert.Equal(t, downloader.NotStarted, jobStatusOfNewHandle.Name)
}

func Test_GetCacheHandle_WhenFileInfoAndJobAreNotPresent(t *testing.T) {
	cacheDir := path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir")
	chTestArgs := initializeCacheHandlerTestArgs(t, &cfg.FileCacheConfig{EnableCrc: true}, cacheDir)
	minObject := createObject(t, chTestArgs.bucket, "object_1", []byte("content of object_1"))

	cacheHandle, err := chTestArgs.cacheHandler.GetCacheHandle(minObject, chTestArgs.bucket, false, 0)

	assert.NoError(t, err)
	assert.Nil(t, cacheHandle.validateCacheHandle())
	// New Job and file info are created.
	assert.True(t, isEntryInFileInfoCache(t, chTestArgs.cache, minObject.Name, chTestArgs.bucket.Name()))
	jobStatusOfNewHandle := cacheHandle.fileDownloadJob.GetStatus()
	assert.Equal(t, downloader.NotStarted, jobStatusOfNewHandle.Name)
}

func Test_GetCacheHandle_WithEviction(t *testing.T) {
	tbl := []struct {
		name            string
		fileCacheConfig cfg.FileCacheConfig
		cacheDir        string
	}{
		{
			name:            "Non parallel downloads",
			fileCacheConfig: cfg.FileCacheConfig{EnableCrc: true},
			cacheDir:        path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
		},
		{
			name: "Parallel downloads",
			fileCacheConfig: cfg.FileCacheConfig{
				EnableCrc:                true,
				EnableParallelDownloads:  true,
				ParallelDownloadsPerFile: 4,
				MaxParallelDownloads:     20,
				DownloadChunkSizeMb:      3,
				WriteBufferSize:          4 * 1024 * 1024,
			},
			cacheDir: path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
		},
	}
	for _, tc := range tbl {
		t.Run(tc.name, func(t *testing.T) {
			chTestArgs := initializeCacheHandlerTestArgs(t, &tc.fileCacheConfig, tc.cacheDir)
			// Start the existing job
			existingJob := getDownloadJobForTestObject(t, chTestArgs)
			_, err := existingJob.Download(context.Background(), 1, false)
			require.NoError(t, err)
			// Content of size more than 20 leads to eviction of initial TestObjectName.
			// Here, content size is 21.
			minObject := createObject(t, chTestArgs.bucket, "object_1", []byte("content of object_1 ..."))

			cacheHandle2, err := chTestArgs.cacheHandler.GetCacheHandle(minObject, chTestArgs.bucket, false, 0)

			assert.NoError(t, err)
			assert.Nil(t, cacheHandle2.validateCacheHandle())
			jobStatus := existingJob.GetStatus()
			assert.Equal(t, downloader.Invalid, jobStatus.Name)
			assert.False(t, doesFileExist(t, chTestArgs.downloadPath))
		})
	}
}

func Test_GetCacheHandle_IfLocalFileGetsDeleted(t *testing.T) {
	cacheDir := path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir")
	chTestArgs := initializeCacheHandlerTestArgs(t, &cfg.FileCacheConfig{EnableCrc: true}, cacheDir)
	// Delete the local cache file.
	err := os.Remove(chTestArgs.downloadPath)
	require.NoError(t, err)
	existingJob := getDownloadJobForTestObject(t, chTestArgs)

	cacheHandle, err := chTestArgs.cacheHandler.GetCacheHandle(chTestArgs.object, chTestArgs.bucket, false, 0)

	assert.True(t, errors.Is(err, util.ErrFileNotPresentInCache))
	assert.Nil(t, cacheHandle)
	// Check file info and download job are not removed
	assert.True(t, isEntryInFileInfoCache(t, chTestArgs.cache, chTestArgs.object.Name, chTestArgs.bucket.Name()))
	actualJob := chTestArgs.jobManager.GetJob(chTestArgs.object.Name, chTestArgs.bucket.Name())
	assert.Equal(t, existingJob, actualJob)
	assert.Equal(t, downloader.NotStarted, existingJob.GetStatus().Name)
}

func Test_GetCacheHandle_ExcludeFromCache(t *testing.T) {
	regex := ".*object_1"
	cacheDir := path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir")
	chTestArgs := initializeCacheHandlerTestArgs(t, &cfg.FileCacheConfig{EnableCrc: true, ExcludeRegex: regex}, cacheDir)

	// Check cache handle is not created for excluded file
	chTestArgs.object.Name = "object_1"
	cacheHandle, err := chTestArgs.cacheHandler.GetCacheHandle(chTestArgs.object, chTestArgs.bucket, false, 0)
	assert.True(t, errors.Is(err, util.ErrFileExcludedFromCacheByRegex))
	assert.Nil(t, cacheHandle)

	// Check cache handle is created for file not excluded.
	chTestArgs.object.Name = "object_2"
	cacheHandle, err = chTestArgs.cacheHandler.GetCacheHandle(chTestArgs.object, chTestArgs.bucket, false, 0)
	assert.NoError(t, err)
	assert.Nil(t, cacheHandle.validateCacheHandle())

}

func Test_GetCacheHandle_IncludeInCache(t *testing.T) {
	regex := ".*object_1"
	cacheDir := path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir")
	chTestArgs := initializeCacheHandlerTestArgs(t, &cfg.FileCacheConfig{EnableCrc: true, IncludeRegex: regex}, cacheDir)

	// Check cache handle is created for included file.
	chTestArgs.object.Name = "object_1"
	cacheHandle, err := chTestArgs.cacheHandler.GetCacheHandle(chTestArgs.object, chTestArgs.bucket, false, 0)
	assert.NoError(t, err)
	assert.Nil(t, cacheHandle.validateCacheHandle())

	// Check cache handle is not created for file not included.
	chTestArgs.object.Name = "object_2"
	cacheHandle, err = chTestArgs.cacheHandler.GetCacheHandle(chTestArgs.object, chTestArgs.bucket, false, 0)
	assert.True(t, errors.Is(err, util.ErrFileExcludedFromCacheByRegex))
	assert.Nil(t, cacheHandle)
}

func Test_GetCacheHandle_IncludeAndExclude(t *testing.T) {
	includeRegex := ".*\\.txt"
	excludeRegex := ".*_internal\\.txt"
	cacheDir := path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir")
	chTestArgs := initializeCacheHandlerTestArgs(t, &cfg.FileCacheConfig{EnableCrc: true, IncludeRegex: includeRegex, ExcludeRegex: excludeRegex}, cacheDir)

	// Check cache handle is created for included file.
	chTestArgs.object.Name = "some_file.txt"
	cacheHandle, err := chTestArgs.cacheHandler.GetCacheHandle(chTestArgs.object, chTestArgs.bucket, false, 0)
	assert.NoError(t, err)
	assert.Nil(t, cacheHandle.validateCacheHandle())

	// Check cache handle is not created for excluded file even if it matches include pattern.
	chTestArgs.object.Name = "some_file_internal.txt"
	cacheHandle, err = chTestArgs.cacheHandler.GetCacheHandle(chTestArgs.object, chTestArgs.bucket, false, 0)
	assert.True(t, errors.Is(err, util.ErrFileExcludedFromCacheByRegex))
	assert.Nil(t, cacheHandle)
}

func Test_GetCacheHandle_SameIncludeAndExcludeRegex(t *testing.T) {
	regex := ".*\\.txt"
	cacheDir := path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir")
	chTestArgs := initializeCacheHandlerTestArgs(t, &cfg.FileCacheConfig{EnableCrc: true, IncludeRegex: regex, ExcludeRegex: regex}, cacheDir)

	// Check cache handle is not created for a file that matches both include and
	// exclude regex, as exclude takes precedence.
	chTestArgs.object.Name = "some_file.txt"
	cacheHandle, err := chTestArgs.cacheHandler.GetCacheHandle(chTestArgs.object, chTestArgs.bucket, false, 0)

	assert.True(t, errors.Is(err, util.ErrFileExcludedFromCacheByRegex))
	assert.Nil(t, cacheHandle)
}

func Test_GetCacheHandle_CacheForRangeRead(t *testing.T) {
	tbl := []struct {
		name            string
		fileCacheConfig cfg.FileCacheConfig
		cacheDir        string
	}{
		{
			name:            "Non parallel downloads",
			fileCacheConfig: cfg.FileCacheConfig{EnableCrc: true},
			cacheDir:        path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
		},
		{
			name: "Parallel downloads",
			fileCacheConfig: cfg.FileCacheConfig{
				EnableCrc:                true,
				EnableParallelDownloads:  true,
				ParallelDownloadsPerFile: 4,
				MaxParallelDownloads:     20,
				DownloadChunkSizeMb:      3,
				WriteBufferSize:          4 * 1024 * 1024,
			},
			cacheDir: path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
		},
	}
	for _, tc := range tbl {
		t.Run(tc.name, func(t *testing.T) {
			chTestArgs := initializeCacheHandlerTestArgs(t, &tc.fileCacheConfig, tc.cacheDir)
			minObject1 := createObject(t, chTestArgs.bucket, "object_1", []byte("content of object_1 ..."))
			cacheHandle1, err1 := chTestArgs.cacheHandler.GetCacheHandle(minObject1, chTestArgs.bucket, false, 0)
			minObject2 := createObject(t, chTestArgs.bucket, "object_2", []byte("content of object_2 ..."))
			cacheHandle2, err2 := chTestArgs.cacheHandler.GetCacheHandle(minObject2, chTestArgs.bucket, false, 5)
			minObject3 := createObject(t, chTestArgs.bucket, "object_3", []byte("content of object_3 ..."))
			cacheHandle3, err3 := chTestArgs.cacheHandler.GetCacheHandle(minObject3, chTestArgs.bucket, true, 0)
			minObject4 := createObject(t, chTestArgs.bucket, "object_4", []byte("content of object_4 ..."))
			cacheHandle4, err4 := chTestArgs.cacheHandler.GetCacheHandle(minObject4, chTestArgs.bucket, true, 5)

			assert.NoError(t, err1)
			assert.Nil(t, cacheHandle1.validateCacheHandle())
			assert.True(t, errors.Is(err2, util.ErrCacheHandleNotRequiredForRandomRead))
			assert.Nil(t, cacheHandle2)
			assert.NoError(t, err3)
			assert.Nil(t, cacheHandle3.validateCacheHandle())
			assert.NoError(t, err4)
			assert.Nil(t, cacheHandle4.validateCacheHandle())
		})
	}
}

func Test_GetCacheHandle_ConcurrentSameFile(t *testing.T) {
	tbl := []struct {
		name            string
		fileCacheConfig cfg.FileCacheConfig
		cacheDir        string
	}{
		{
			name:            "Non parallel downloads",
			fileCacheConfig: cfg.FileCacheConfig{EnableCrc: true},
			cacheDir:        path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
		},
		{
			name: "Parallel downloads",
			fileCacheConfig: cfg.FileCacheConfig{
				EnableCrc:                true,
				EnableParallelDownloads:  true,
				ParallelDownloadsPerFile: 1,
				MaxParallelDownloads:     20,
				DownloadChunkSizeMb:      3,
				WriteBufferSize:          4 * 1024 * 1024,
			},
			cacheDir: path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
		},
	}
	for _, tc := range tbl {
		t.Run(tc.name, func(t *testing.T) {
			chTestArgs := initializeCacheHandlerTestArgs(t, &tc.fileCacheConfig, tc.cacheDir)
			// Check async job and file info cache not preset for object_1
			testObjectName := "object_1"
			existingJob := chTestArgs.jobManager.GetJob(testObjectName, chTestArgs.bucket.Name())
			require.Nil(t, existingJob)
			wg := sync.WaitGroup{}
			getCacheHandleTestFun := func(t *testing.T) {
				defer wg.Done()
				minObj := createObject(t, chTestArgs.bucket, testObjectName, []byte("content of object_1 ..."))

				var err error
				cacheHandle, err := chTestArgs.cacheHandler.GetCacheHandle(minObj, chTestArgs.bucket, false, 0)

				assert.NoError(t, err)
				assert.Nil(t, cacheHandle.validateCacheHandle())
			}

			// Start concurrent GetCacheHandle()
			for range 5 {
				wg.Add(1)
				go getCacheHandleTestFun(t)
			}
			wg.Wait()

			// Job should be added now
			actualJob := chTestArgs.jobManager.GetJob(testObjectName, chTestArgs.bucket.Name())
			jobStatus := actualJob.GetStatus()
			assert.Equal(t, downloader.NotStarted, jobStatus.Name)
			assert.True(t, doesFileExist(t, util.GetDownloadPath(chTestArgs.cacheDir,
				util.GetObjectPath(chTestArgs.bucket.Name(), testObjectName))))
		})
	}
}

func Test_GetCacheHandle_ConcurrentDifferentFiles(t *testing.T) {
	cacheDir := path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir")
	chTestArgs := initializeCacheHandlerTestArgs(t, &cfg.FileCacheConfig{EnableCrc: true}, cacheDir)
	existingJob := getDownloadJobForTestObject(t, chTestArgs)
	require.Equal(t, downloader.NotStarted, existingJob.GetStatus().Name)
	wg := sync.WaitGroup{}

	getCacheHandleTestFun := func(index int) {
		defer wg.Done()
		objName := "object" + strconv.Itoa(index)
		objContent := "object content: content#" + strconv.Itoa(index)
		minObj := createObject(t, chTestArgs.bucket, objName, []byte(objContent))

		cacheHandle, err := chTestArgs.cacheHandler.GetCacheHandle(minObj, chTestArgs.bucket, false, 0)

		assert.NoError(t, err)
		assert.Nil(t, cacheHandle.validateCacheHandle())
	}

	// Start concurrent GetCacheHandle()
	for i := range 5 {
		wg.Add(1)
		go getCacheHandleTestFun(i)
	}
	wg.Wait()

	assert.NotNil(t, existingJob)
	assert.Equal(t, downloader.Invalid, existingJob.GetStatus().Name)
	assert.False(t, doesFileExist(t, chTestArgs.downloadPath))
	// File info should also be removed.
	assert.False(t, isEntryInFileInfoCache(t, chTestArgs.cache, chTestArgs.object.Name, chTestArgs.bucket.Name()))
}

func Test_InvalidateCache_WhenAlreadyInCache(t *testing.T) {
	cacheDir := path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir")
	chTestArgs := initializeCacheHandlerTestArgs(t, &cfg.FileCacheConfig{EnableCrc: true}, cacheDir)
	existingJob := getDownloadJobForTestObject(t, chTestArgs)
	require.Equal(t, downloader.NotStarted, existingJob.GetStatus().Name)
	require.True(t, isEntryInFileInfoCache(t, chTestArgs.cache, chTestArgs.object.Name, chTestArgs.bucket.Name()))

	err := chTestArgs.cacheHandler.InvalidateCache(chTestArgs.object.Name, chTestArgs.bucket.Name())

	assert.NoError(t, err)
	// Existing job for default chrT object should be invalidated.
	assert.NotNil(t, existingJob)
	assert.Equal(t, downloader.Invalid, existingJob.GetStatus().Name)
	assert.False(t, doesFileExist(t, chTestArgs.downloadPath))
	// File info should also be removed.
	assert.False(t, isEntryInFileInfoCache(t, chTestArgs.cache, chTestArgs.object.Name, chTestArgs.bucket.Name()))
}

func Test_InvalidateCache_WhenEntryNotInCache(t *testing.T) {
	cacheDir := path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir")
	chTestArgs := initializeCacheHandlerTestArgs(t, &cfg.FileCacheConfig{EnableCrc: true}, cacheDir)
	minObject := createObject(t, chTestArgs.bucket, "object_1", []byte("content of object_1"))
	require.False(t, isEntryInFileInfoCache(t, chTestArgs.cache, minObject.Name, chTestArgs.bucket.Name()))
	require.Nil(t, chTestArgs.jobManager.GetJob(minObject.Name, chTestArgs.bucket.Name()))

	err := chTestArgs.cacheHandler.InvalidateCache(minObject.Name, chTestArgs.bucket.Name())

	assert.NoError(t, err)
	assert.False(t, isEntryInFileInfoCache(t, chTestArgs.cache, minObject.Name, chTestArgs.bucket.Name()))
	assert.Nil(t, chTestArgs.jobManager.GetJob(minObject.Name, chTestArgs.bucket.Name()))
}

func Test_InvalidateCache_Truncates(t *testing.T) {
	tbl := []struct {
		name                         string
		fileCacheConfig              cfg.FileCacheConfig
		cacheDir                     string
		isCacheHandleReadErrExpected bool
		isInvalidateCacheErrExpected bool
		isCacheFileReadErrExpected   bool
	}{
		{
			name:                         "Non parallel downloads",
			fileCacheConfig:              cfg.FileCacheConfig{EnableCrc: true},
			cacheDir:                     path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
			isCacheHandleReadErrExpected: false,
			isInvalidateCacheErrExpected: false,
			isCacheFileReadErrExpected:   true,
		},
		{
			name: "Parallel downloads",
			fileCacheConfig: cfg.FileCacheConfig{
				EnableCrc:                true,
				EnableParallelDownloads:  true,
				ParallelDownloadsPerFile: 4,
				MaxParallelDownloads:     20,
				DownloadChunkSizeMb:      3,
				WriteBufferSize:          4 * 1024 * 1024,
			},
			cacheDir: path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
			// Error is expected in parallel downloads because the foreground reads
			// doesn't wait for async job to download till the requested offset unlike
			// in case of non-parallel downloads for sequential reads.
			isCacheHandleReadErrExpected: true,
			isInvalidateCacheErrExpected: false,
			isCacheFileReadErrExpected:   true,
		},
	}
	for _, tc := range tbl {
		t.Run(tc.name, func(t *testing.T) {
			chTestArgs := initializeCacheHandlerTestArgs(t, &tc.fileCacheConfig, tc.cacheDir)
			objectContent := []byte("content of object_1")
			minObject := createObject(t, chTestArgs.bucket, "object_1", objectContent)
			cacheHandle, err := chTestArgs.cacheHandler.GetCacheHandle(minObject, chTestArgs.bucket, false, 0)
			require.NoError(t, err)
			buf := make([]byte, 3)
			ctx := context.Background()
			// Read to populate cache
			_, cacheHit, err := cacheHandle.Read(ctx, chTestArgs.bucket, minObject, 0, buf)
			if !tc.isCacheHandleReadErrExpected {
				require.NoError(t, err)
				require.Equal(t, string(objectContent[:3]), string(buf))
			} else {
				require.NotNil(t, err)
			}
			require.False(t, cacheHit)
			require.Nil(t, cacheHandle.Close())
			// Open cache file before invalidation
			objectPath := util.GetObjectPath(chTestArgs.bucket.Name(), minObject.Name)
			downloadPath := util.GetDownloadPath(chTestArgs.cacheDir, objectPath)
			file, err := os.OpenFile(downloadPath, os.O_RDONLY, 0600)
			require.NoError(t, err)
			defer func() {
				_ = file.Close()
			}()

			err = chTestArgs.cacheHandler.InvalidateCache(minObject.Name, chTestArgs.bucket.Name())

			if tc.isInvalidateCacheErrExpected {
				assert.NotNil(t, err)
			} else {
				assert.NoError(t, err)
			}
			// Reading from the open file handle should fail as the file is truncated.
			_, err = file.Read(buf)
			if tc.isCacheFileReadErrExpected {
				assert.NotNil(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_InvalidateCache_ConcurrentSameFile(t *testing.T) {
	tbl := []struct {
		name            string
		fileCacheConfig cfg.FileCacheConfig
		cacheDir        string
	}{
		{
			name:            "Non parallel downloads",
			fileCacheConfig: cfg.FileCacheConfig{EnableCrc: true},
			cacheDir:        path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
		},
		{
			name: "Parallel downloads",
			fileCacheConfig: cfg.FileCacheConfig{
				EnableCrc:                true,
				EnableParallelDownloads:  true,
				ParallelDownloadsPerFile: 1,
				MaxParallelDownloads:     20,
				DownloadChunkSizeMb:      3,
				WriteBufferSize:          4 * 1024 * 1024,
			},
			cacheDir: path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
		},
	}
	for _, tc := range tbl {
		t.Run(tc.name, func(t *testing.T) {
			chTestArgs := initializeCacheHandlerTestArgs(t, &tc.fileCacheConfig, tc.cacheDir)
			existingJob := getDownloadJobForTestObject(t, chTestArgs)
			require.Equal(t, downloader.NotStarted, existingJob.GetStatus().Name)
			require.True(t, isEntryInFileInfoCache(t, chTestArgs.cache, chTestArgs.object.Name, chTestArgs.bucket.Name()))
			wg := sync.WaitGroup{}
			invalidateCacheTestFun := func(t *testing.T) {
				defer wg.Done()

				err := chTestArgs.cacheHandler.InvalidateCache(chTestArgs.object.Name, chTestArgs.bucket.Name())

				assert.NoError(t, err)
				assert.NotNil(t, existingJob)
				assert.Equal(t, downloader.Invalid, existingJob.GetStatus().Name)
				assert.False(t, doesFileExist(t, chTestArgs.downloadPath))
				// File info should also be removed.
				assert.False(t, isEntryInFileInfoCache(t, chTestArgs.cache, chTestArgs.object.Name, chTestArgs.bucket.Name()))
			}

			// Start concurrent GetCacheHandle()
			for range 5 {
				wg.Add(1)
				go invalidateCacheTestFun(t)
			}
			wg.Wait()
		})
	}
}

func Test_InvalidateCache_ConcurrentDifferentFiles(t *testing.T) {
	tbl := []struct {
		name            string
		fileCacheConfig cfg.FileCacheConfig
		cacheDir        string
	}{
		{
			name:            "Non parallel downloads",
			fileCacheConfig: cfg.FileCacheConfig{EnableCrc: true},
			cacheDir:        path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
		},
		{
			name: "Parallel downloads",
			fileCacheConfig: cfg.FileCacheConfig{
				EnableCrc:                true,
				EnableParallelDownloads:  true,
				ParallelDownloadsPerFile: 1,
				MaxParallelDownloads:     20,
				DownloadChunkSizeMb:      3,
				WriteBufferSize:          4 * 1024 * 1024,
			},
			cacheDir: path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
		},
	}
	for _, tc := range tbl {
		t.Run(tc.name, func(t *testing.T) {
			chTestArgs := initializeCacheHandlerTestArgs(t, &tc.fileCacheConfig, tc.cacheDir)
			wg := sync.WaitGroup{}
			invalidateCacheTestFun := func(index int) {
				defer wg.Done()
				objName := "object" + strconv.Itoa(index)
				objContent := "object content: content#" + strconv.Itoa(index)
				minObj := createObject(t, chTestArgs.bucket, objName, []byte(objContent))

				err := chTestArgs.cacheHandler.InvalidateCache(minObj.Name, chTestArgs.bucket.Name())

				assert.NoError(t, err)
				assert.Nil(t, chTestArgs.jobManager.GetJob(objName, chTestArgs.bucket.Name()))
				assert.False(t, isEntryInFileInfoCache(t, chTestArgs.cache, objName, chTestArgs.bucket.Name()))
			}

			// Start concurrent GetCacheHandle()
			for i := range 5 {
				wg.Add(1)
				go invalidateCacheTestFun(i)
			}
			wg.Wait()
		})
	}
}

func Test_InvalidateCache_GetCacheHandle_Concurrent(t *testing.T) {
	tbl := []struct {
		name            string
		fileCacheConfig cfg.FileCacheConfig
		cacheDir        string
	}{
		{
			name:            "Non parallel downloads",
			fileCacheConfig: cfg.FileCacheConfig{EnableCrc: true},
			cacheDir:        path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
		},
		{
			name: "Parallel downloads",
			fileCacheConfig: cfg.FileCacheConfig{
				EnableCrc:                true,
				EnableParallelDownloads:  true,
				ParallelDownloadsPerFile: 1,
				MaxParallelDownloads:     20,
				DownloadChunkSizeMb:      3,
				WriteBufferSize:          4 * 1024 * 1024,
			},
			cacheDir: path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
		},
	}
	for _, tc := range tbl {
		t.Run(tc.name, func(t *testing.T) {
			chTestArgs := initializeCacheHandlerTestArgs(t, &tc.fileCacheConfig, tc.cacheDir)
			wg := sync.WaitGroup{}
			invalidateCacheTestFun := func(index int) {
				defer wg.Done()
				objName := "object" + strconv.Itoa(index)
				objContent := "object content: content#" + strconv.Itoa(index)
				minObj := createObject(t, chTestArgs.bucket, objName, []byte(objContent))

				err := chTestArgs.cacheHandler.InvalidateCache(minObj.Name, chTestArgs.bucket.Name())

				assert.NoError(t, err)
			}

			getCacheHandleTestFun := func(index int) {
				defer wg.Done()
				objName := "object" + strconv.Itoa(index)
				objContent := "object content: content#" + strconv.Itoa(index)
				minObj := createObject(t, chTestArgs.bucket, objName, []byte(objContent))

				cacheHandle, err := chTestArgs.cacheHandler.GetCacheHandle(minObj, chTestArgs.bucket, false, 0)

				assert.NoError(t, err)
				assert.Nil(t, cacheHandle.validateCacheHandle())
			}

			// Start concurrent GetCacheHandle()
			for i := range 5 {
				wg.Add(1)
				go invalidateCacheTestFun(i)
				wg.Add(1)
				go getCacheHandleTestFun(i)
			}
			wg.Wait()
		})
	}
}

func Test_Destroy(t *testing.T) {
	tbl := []struct {
		name                     string
		fileCacheConfig          cfg.FileCacheConfig
		cacheDir                 string
		isCacheHandleErrExpected bool
		expectedJobStatus        []string
	}{
		{
			name:                     "Non parallel downloads",
			fileCacheConfig:          cfg.FileCacheConfig{EnableCrc: true},
			cacheDir:                 path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
			isCacheHandleErrExpected: false,
			expectedJobStatus:        []string{string(downloader.Completed)},
		},
		{
			name: "Parallel downloads",
			fileCacheConfig: cfg.FileCacheConfig{
				EnableCrc:                true,
				EnableParallelDownloads:  true,
				ParallelDownloadsPerFile: 4,
				MaxParallelDownloads:     20,
				DownloadChunkSizeMb:      3,
				WriteBufferSize:          4 * 1024 * 1024,
			},
			cacheDir: path.Join(os.Getenv("HOME"), "CacheHandlerTest/dir"),
			// Error is expected in parallel downloads because the foreground reads
			// doesn't wait for async job to download till the requested offset unlike
			// in case of non-parallel downloads for sequential reads.
			isCacheHandleErrExpected: true,
			expectedJobStatus:        []string{string(downloader.Completed), string(downloader.Invalid)},
		},
	}
	for _, tc := range tbl {
		t.Run(tc.name, func(t *testing.T) {
			chTestArgs := initializeCacheHandlerTestArgs(t, &tc.fileCacheConfig, tc.cacheDir)
			minObject1 := createObject(t, chTestArgs.bucket, "object_1", []byte("content of object_1"))
			minObject2 := createObject(t, chTestArgs.bucket, "object_2", []byte("content of object_2"))
			cacheHandle1, err := chTestArgs.cacheHandler.GetCacheHandle(minObject1, chTestArgs.bucket, true, 0)
			require.NoError(t, err)
			cacheHandle2, err := chTestArgs.cacheHandler.GetCacheHandle(minObject2, chTestArgs.bucket, true, 0)
			require.NoError(t, err)
			ctx := context.Background()
			// Read to create and populate file in cache.
			buf := make([]byte, 3)
			_, cacheHit, err := cacheHandle1.Read(ctx, chTestArgs.bucket, minObject1, 4, buf)
			if tc.isCacheHandleErrExpected {
				require.NotNil(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, false, cacheHit)
			_, cacheHit, err = cacheHandle2.Read(ctx, chTestArgs.bucket, minObject2, 4, buf)
			if tc.isCacheHandleErrExpected {
				require.NotNil(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, false, cacheHit)
			err = cacheHandle1.Close()
			require.NoError(t, err)
			err = cacheHandle2.Close()
			require.NoError(t, err)

			err = chTestArgs.cacheHandler.Destroy()

			assert.NoError(t, err)
			// Verify the cacheDir is deleted.
			_, err = os.Stat(path.Join(chTestArgs.cacheDir, util.FileCache))
			assert.ErrorIs(t, err, os.ErrNotExist)
			// Verify jobs statuses.
			job1 := chTestArgs.jobManager.GetJob(minObject1.Name, chTestArgs.bucket.Name())
			job2 := chTestArgs.jobManager.GetJob(minObject1.Name, chTestArgs.bucket.Name())
			if job1 != nil {
				assert.Contains(t, tc.expectedJobStatus, job1.GetStatus().Name)
			}
			if job2 != nil {
				assert.Contains(t, tc.expectedJobStatus, job2.GetStatus().Name)
			}
			// Job manager should no longer contain the jobs
			assert.Nil(t, chTestArgs.jobManager.GetJob(minObject1.Name, chTestArgs.bucket.Name()))
			assert.Nil(t, chTestArgs.jobManager.GetJob(minObject2.Name, chTestArgs.bucket.Name()))
		})
	}
}
