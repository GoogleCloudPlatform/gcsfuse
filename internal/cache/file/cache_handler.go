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

	// cacheLocation is the local path which contains the cache data i.e. objects stored as file.
	cacheLocation string

	// filePerm parameter specifies the permission of file in cache.
	filePerm os.FileMode

	// mu guards the handling of insertion into and eviction from file cache.
	mu locker.Locker
}

func NewCacheHandler(fileInfoCache *lru.Cache, jobManager *downloader.JobManager, cacheLocation string, filePerm os.FileMode) *CacheHandler {
	return &CacheHandler{
		fileInfoCache: fileInfoCache,
		jobManager:    jobManager,
		cacheLocation: cacheLocation,
		filePerm:      filePerm,
		mu:            locker.New("FileCacheHandler", func() {}),
	}
}

func (chr *CacheHandler) createLocalFileReadHandle(objectName string, bucketName string) (*os.File, error) {
	fileSpec := data.FileSpec{
		Path: util.GetDownloadPath(chr.cacheLocation, util.GetObjectPath(bucketName, objectName)),
		Perm: chr.filePerm,
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
		return fmt.Errorf("cleanUpEvictedFile: while performing post eviction: %v", err)
	}

	// Removing Job doesn't delete the job object itself but invalidates the job and
	// hence it's possible that some existing cache handle will have reference to older
	// job object, but that job object will have INVALID status.
	chr.jobManager.RemoveJob(key.ObjectName, key.BucketName)

	localFilePath := util.GetDownloadPath(chr.cacheLocation, util.GetObjectPath(key.BucketName, key.ObjectName))
	// Truncate the file to 0 size, so that even if there are open file handles
	// and linux doesn't delete the file, the file will not take space.
	err = os.Truncate(localFilePath, 0)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warnf("cleanUpEvictedFile: file was not present at the time of truncating: %v", err)
			return nil
		} else {
			return fmt.Errorf("cleanUpEvictedFile: while truncating file: %s, error: %v", localFilePath, err)
		}
	}
	err = os.Remove(localFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warnf("cleanUpEvictedFile: file was not present at the time of deleting: %v", err)
		} else {
			return fmt.Errorf("cleanUpEvictedFile: while deleting file: %s, error: %v", localFilePath, err)
		}
	}

	return nil
}

// addFileInfoEntryToCache adds a data.FileInfo entry for the given object and bucket
// in the file info cache if it does not already exist. It also cleans up for entries
// that are evicted at the time of adding new entry.
// In case if the cache contains the stale data.FileInfo entry (generation < object.generation)
// it cleans up (job and local cache file) for the old entry and adds the new entry with the
// latest generation to the cache.
//
// Requires Lock(chr.mu)
func (chr *CacheHandler) addFileInfoEntryToCache(object *gcs.MinObject, bucket gcs.Bucket) error {
	fileInfoKey := data.FileInfoKey{
		BucketName: bucket.Name(),
		ObjectName: object.Name,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	if err != nil {
		return fmt.Errorf("addFileInfoEntryToCache: while creating key: %v", fileInfoKeyName)
	}

	addEntryToCache := false
	fileInfo := chr.fileInfoCache.LookUpWithoutChangingOrder(fileInfoKeyName)
	if fileInfo == nil {
		addEntryToCache = true
	} else {
		// Throw an error, if there is an entry in the file-info cache and cache file doesn't
		// exist locally.
		filePath := util.GetDownloadPath(chr.cacheLocation, util.GetObjectPath(bucket.Name(), object.Name))
		_, err := os.Stat(filePath)
		if err != nil && os.IsNotExist(err) {
			return fmt.Errorf("addFileInfoEntryToCache: %s: %s", util.FileNotPresentInCacheErrMsg, filePath)
		}

		fileInfoData := fileInfo.(data.FileInfo)
		if fileInfoData.ObjectGeneration < object.Generation {
			erasedVal := chr.fileInfoCache.Erase(fileInfoKeyName)
			if erasedVal != nil {
				erasedFileInfo := erasedVal.(data.FileInfo)
				err := chr.cleanUpEvictedFile(&erasedFileInfo)
				if err != nil {
					return fmt.Errorf("addFileInfoEntryToCache: while performing post eviction of %s object error: %v", erasedFileInfo.Key.ObjectName, err)
				}
			}
			addEntryToCache = true
		} else if fileInfoData.ObjectGeneration > object.Generation {
			return fmt.Errorf("addFileInfoEntryToCache: cache generation %d is more than object generation: %d", fileInfoData.ObjectGeneration, object.Generation)
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
			return fmt.Errorf("addFileInfoEntryToCache: while inserting into the cache: %v", err)
		}

		for _, val := range evictedValues {
			fileInfo := val.(data.FileInfo)
			err := chr.cleanUpEvictedFile(&fileInfo)
			if err != nil {
				return fmt.Errorf("addFileInfoEntryToCache: while performing post eviction of %s object error: %v", fileInfo.Key.ObjectName, err)
			}
		}
	} else {
		job := chr.jobManager.GetJob(object, bucket)
		jobStatus := job.GetStatus()
		if jobStatus.Name != downloader.FAILED {
			// Move this entry on top of LRU.
			_ = chr.fileInfoCache.LookUp(fileInfoKeyName)
		}
	}

	return nil
}

// GetCacheHandle creates an entry in fileInfoCache if it does not already exist. It
// creates downloader.Job if not already exist. Also, creates local file into which
// the download job downloads the object content. Finally, it returns a CacheHandle
// that contains the reference to downloader.Job and the local file handle. This method
// is atomic, that means all the above-mentioned tasks are completed in one uninterrupted
// sequence guarded by (CacheHandler.mu). Note: It returns nil if downloadForRandom is
// set to False, initialOffset is non-zero (i.e. random read) and entry for file
// doesn't already exist in fileInfoCache then no need to create file in cache.
//
// Acquires and releases LOCK(CacheHandler.mu)
func (chr *CacheHandler) GetCacheHandle(object *gcs.MinObject, bucket gcs.Bucket, downloadForRandom bool, initialOffset int64) (*CacheHandle, error) {
	chr.mu.Lock()
	defer chr.mu.Unlock()

	// If downloadForRandom is set to False, initialOffset is non-zero (i.e. random read)
	// and entry for file doesn't already exist in fileInfoCache then no need to
	// create file in cache.
	if !downloadForRandom && initialOffset != 0 {
		fileInfoKey := data.FileInfoKey{
			BucketName: bucket.Name(),
			ObjectName: object.Name,
		}
		fileInfoKeyName, err := fileInfoKey.Key()
		if err != nil {
			return nil, fmt.Errorf("addFileInfoEntryToCache: while creating key: %v", fileInfoKeyName)
		}

		fileInfo := chr.fileInfoCache.LookUpWithoutChangingOrder(fileInfoKeyName)
		if fileInfo == nil {
			return nil, fmt.Errorf("addFileInfoEntryToCache: %s", util.CacheHandleNotRequiredForRandomReadErrMsg)
		}
	}

	err := chr.addFileInfoEntryToCache(object, bucket)
	if err != nil {
		return nil, fmt.Errorf("GetCacheHandle: while adding the entry in the cache: %v", err)
	}

	localFileReadHandle, err := chr.createLocalFileReadHandle(object.Name, bucket.Name())
	if err != nil {
		return nil, fmt.Errorf("GetCacheHandle: while create local-file read handle: %v", err)
	}

	return NewCacheHandle(localFileReadHandle, chr.jobManager.GetJob(object, bucket), chr.fileInfoCache, downloadForRandom, initialOffset), nil
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
			return fmt.Errorf("InvalidateCache: while performing clean-up for evicted  %s object, error: %v", fileInfo.Key.ObjectName, err)
		}
	}
	return nil
}

// Destroy destroys the internal state of CacheHandler correctly specifically evict all the
// entries in the file info cache and clean up for the evicted entries.
// TODO (raj-prince) to implement.
func (chr *CacheHandler) Destroy() {

}
