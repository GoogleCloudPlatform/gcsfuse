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
	"fmt"
	"os"

	"github.com/googlecloudplatform/gcsfuse/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/file/downloader"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/gcs"
)

// CacheHandler is responsible for creating CacheHandle and invalidating file cache
// for a given object in the bucket. CacheHandle contains reference to download job and
// file handle to file in cache.
// Additionally, while creating the CacheHandle, it ensures that the file info entry
// is present in the fileInfoCache and a file is present in cache inside the appropriate
// directory.
type CacheHandler struct {
	// fileInfoCache contains the reference of fileInfo cache.
	fileInfoCache *lru.Cache

	// jobManager contains reference to a singleton jobManager.
	jobManager *downloader.JobManager

	// cacheDir is the local path which contains the cache data i.e. objects stored as file.
	cacheDir string

	// filePerm parameter specifies the permission of file in cache.
	filePerm os.FileMode

	// dirPerm parameter specifies the permission of cache directory.
	dirPerm os.FileMode

	// mu guards the handling of insertion into and eviction from file cache.
	mu locker.Locker
}

func NewCacheHandler(fileInfoCache *lru.Cache, jobManager *downloader.JobManager, cacheDir string, filePerm os.FileMode, dirPerm os.FileMode) *CacheHandler {
	return &CacheHandler{
		fileInfoCache: fileInfoCache,
		jobManager:    jobManager,
		cacheDir:      cacheDir,
		filePerm:      filePerm,
		dirPerm:       dirPerm,
		mu:            locker.New("FileCacheHandler", func() {}),
	}
}

func (chr *CacheHandler) createLocalFileReadHandle(objectName string, bucketName string) (*os.File, error) {
	fileSpec := data.FileSpec{
		Path:     util.GetDownloadPath(chr.cacheDir, util.GetObjectPath(bucketName, objectName)),
		FilePerm: chr.filePerm,
		DirPerm:  chr.dirPerm,
	}

	return util.CreateFile(fileSpec, os.O_RDONLY)
}

// cleanUpEvictedFile is a utility method called for the evicted/deleted fileInfo.
// As part of execution, it (a) stops and removes the download job (b) truncates
// and deletes the file in cache.
func (chr *CacheHandler) cleanUpEvictedFile(fileInfo *data.FileInfo) error {
	key := fileInfo.Key
	_, err := key.Key()
	if err != nil {
		return fmt.Errorf("cleanUpEvictedFile: while creating key: %w", err)
	}

	chr.jobManager.InvalidateAndRemoveJob(key.ObjectName, key.BucketName)

	localFilePath := util.GetDownloadPath(chr.cacheDir, util.GetObjectPath(key.BucketName, key.ObjectName))
	// Truncate the file to 0 size, so that even if there are open file handles
	// and linux doesn't delete the file, the file will not take space.
	err = os.Truncate(localFilePath, 0)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warnf("cleanUpEvictedFile: file was not present at the time of truncating: %v", err)
			return nil
		} else {
			return fmt.Errorf("cleanUpEvictedFile: while truncating file: %s, error: %w", localFilePath, err)
		}
	}
	err = os.Remove(localFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warnf("cleanUpEvictedFile: file was not present at the time of deleting: %v", err)
		} else {
			return fmt.Errorf("cleanUpEvictedFile: while deleting file: %s, error: %w", localFilePath, err)
		}
	}

	return nil
}

// addFileInfoEntryAndCreateDownloadJob adds data.FileInfo entry for the given
// object and bucket in the file info cache and creates download job if they do
// not already exist. It also cleans up for entries that are evicted at the time
// of adding new entry. In case the cache contains the data.FileInfo entry with
// different generation or if the job is failed/invalidated, it cleans up
// (job and local cache file) the old entry and adds the new entry and download
// job with the given generation to the cache.
//
// Requires Lock(chr.mu)
func (chr *CacheHandler) addFileInfoEntryAndCreateDownloadJob(object *gcs.MinObject, bucket gcs.Bucket) error {
	fileInfoKey := data.FileInfoKey{
		BucketName: bucket.Name(),
		ObjectName: object.Name,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	if err != nil {
		return fmt.Errorf("addFileInfoEntryAndCreateDownloadJob: while creating key: %v", fileInfoKeyName)
	}

	addEntryToCache := false
	fileInfo := chr.fileInfoCache.LookUpWithoutChangingOrder(fileInfoKeyName)
	if fileInfo == nil {
		addEntryToCache = true
	} else {
		// Throw an error, if there is an entry in the file-info cache and cache file doesn't
		// exist locally.
		filePath := util.GetDownloadPath(chr.cacheDir, util.GetObjectPath(bucket.Name(), object.Name))
		_, err := os.Stat(filePath)
		if err != nil && os.IsNotExist(err) {
			return fmt.Errorf("addFileInfoEntryAndCreateDownloadJob: %s: %s", util.FileNotPresentInCacheErrMsg, filePath)
		}

		// Evict object in cache if the generation of object in cache is different
		// from the generation of object in inode (we can't compare generations and
		// decide to evict or not because generations are not always increasing:
		// https://cloud.google.com/storage/docs/metadata#generation-number)
		// Also, invalidate the cache if download job has failed or not invalid.
		fileInfoData := fileInfo.(data.FileInfo)
		// If offset in file info cache is less than object size and there is no
		// reference to download job then it means the job has failed.
		existingJob := chr.jobManager.GetJob(object.Name, bucket.Name())
		shouldInvalidate := (existingJob == nil) && (fileInfoData.Offset < object.Size)
		if (!shouldInvalidate) && (existingJob != nil) {
			existingJobStatus := existingJob.GetStatus().Name
			shouldInvalidate = (existingJobStatus == downloader.Failed) || (existingJobStatus == downloader.Invalid)
		}
		if (fileInfoData.ObjectGeneration != object.Generation) || shouldInvalidate {
			erasedVal := chr.fileInfoCache.Erase(fileInfoKeyName)
			if erasedVal != nil {
				erasedFileInfo := erasedVal.(data.FileInfo)
				err := chr.cleanUpEvictedFile(&erasedFileInfo)
				if err != nil {
					return fmt.Errorf("addFileInfoEntryAndCreateDownloadJob: while performing post eviction of %s object error: %w", erasedFileInfo.Key.ObjectName, err)
				}
			}
			addEntryToCache = true
		}
	}

	if addEntryToCache {
		fileInfo = data.FileInfo{
			Key:              fileInfoKey,
			ObjectGeneration: object.Generation,
			Offset:           0,
			FileSize:         object.Size,
		}

		evictedValues, err := chr.fileInfoCache.Insert(fileInfoKeyName, fileInfo)
		if err != nil {
			return fmt.Errorf("addFileInfoEntryAndCreateDownloadJob: while inserting into the cache: %w", err)
		}
		// Create download job for new entry added to cache.
		_ = chr.jobManager.CreateJobIfNotExists(object, bucket)
		for _, val := range evictedValues {
			fileInfo := val.(data.FileInfo)
			err := chr.cleanUpEvictedFile(&fileInfo)
			if err != nil {
				return fmt.Errorf("addFileInfoEntryAndCreateDownloadJob: while performing post eviction of %s object error: %w", fileInfo.Key.ObjectName, err)
			}
		}
	} else {
		// Move this entry on top of LRU.
		_ = chr.fileInfoCache.LookUp(fileInfoKeyName)
	}

	return nil
}

// GetCacheHandle creates an entry in fileInfoCache if it does not already exist. It
// creates downloader.Job if not already exis and requiredt. Also, creates local
// file into which the download job downloads the object content. Finally, it
// returns a CacheHandle that contains the reference to downloader.Job and the
// local file handle. This method is atomic, that means all the above-mentioned
// tasks are completed in one uninterrupted sequence guarded by (CacheHandler.mu).
// Note: It returns nil if cacheForRangeRead is set to False, initialOffset is
// non-zero (i.e. random read) and entry for file doesn't already exist in
// fileInfoCache then no need to create file in cache.
//
// Acquires and releases LOCK(CacheHandler.mu)
func (chr *CacheHandler) GetCacheHandle(object *gcs.MinObject, bucket gcs.Bucket, cacheForRangeRead bool, initialOffset int64) (*CacheHandle, error) {
	chr.mu.Lock()
	defer chr.mu.Unlock()

	// If cacheForRangeRead is set to False, initialOffset is non-zero (i.e. random read)
	// and entry for file doesn't already exist in fileInfoCache then no need to
	// create file in cache.
	if !cacheForRangeRead && initialOffset != 0 {
		fileInfoKey := data.FileInfoKey{
			BucketName: bucket.Name(),
			ObjectName: object.Name,
		}
		fileInfoKeyName, err := fileInfoKey.Key()
		if err != nil {
			return nil, fmt.Errorf("addFileInfoEntryAndCreateDownloadJob: while creating key: %v", fileInfoKeyName)
		}

		fileInfo := chr.fileInfoCache.LookUpWithoutChangingOrder(fileInfoKeyName)
		if fileInfo == nil {
			return nil, fmt.Errorf("addFileInfoEntryAndCreateDownloadJob: %s", util.CacheHandleNotRequiredForRandomReadErrMsg)
		}
	}

	err := chr.addFileInfoEntryAndCreateDownloadJob(object, bucket)
	if err != nil {
		return nil, fmt.Errorf("GetCacheHandle: while adding the entry in the cache: %w", err)
	}

	localFileReadHandle, err := chr.createLocalFileReadHandle(object.Name, bucket.Name())
	if err != nil {
		return nil, fmt.Errorf("GetCacheHandle: while creating local-file read handle: %w", err)
	}

	return NewCacheHandle(localFileReadHandle, chr.jobManager.GetJob(object.Name, bucket.Name()), chr.fileInfoCache, cacheForRangeRead, initialOffset), nil
}

// InvalidateCache removes the file entry from the fileInfoCache and performs clean
// up for the removed entry.
//
// Acquires and releases LOCK(CacheHandler.mu)
func (chr *CacheHandler) InvalidateCache(objectName string, bucketName string) error {
	fileInfoKey := data.FileInfoKey{
		BucketName: bucketName,
		ObjectName: objectName,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	if err != nil {
		return fmt.Errorf("InvalidateCache: while creating key: %v", fileInfoKeyName)
	}

	chr.mu.Lock()
	defer chr.mu.Unlock()

	erasedVal := chr.fileInfoCache.Erase(fileInfoKeyName)
	if erasedVal != nil {
		fileInfo := erasedVal.(data.FileInfo)
		err := chr.cleanUpEvictedFile(&fileInfo)
		if err != nil {
			return fmt.Errorf("InvalidateCache: while performing clean-up for evicted  %s object, error: %w", fileInfo.Key.ObjectName, err)
		}
	}
	return nil
}

// Destroy destroys the job manager (i.e. invalidate all the jobs).
// Note: This method is expected to be called at the time of unmounting and
// because file info cache is in-memory, it is not required to destroy it.
//
// Acquires and releases Lock(chr.mu)
func (chr *CacheHandler) Destroy() (err error) {
	chr.mu.Lock()
	defer chr.mu.Unlock()

	chr.jobManager.Destroy()
	return
}
