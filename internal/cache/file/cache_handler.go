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
	"path"

	"github.com/googlecloudplatform/gcsfuse/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/file/downloader"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/gcs"
)

const DefaultFileMode = os.FileMode(0644)

// CacheHandler Responsible for managing fileInfoCache as well as fileDownloadManager.
type CacheHandler struct {
	// fileInfoCache contains the reference of fileInfo cache.
	fileInfoCache *lru.Cache

	// jobManager contains reference to a singleton jobManager.
	jobManager *downloader.JobManager

	// Directory location which would contain local file to keep the downloaded data.
	cacheLocation string

	// Guards the handling of cache entry insertion while creating the CacheHandle.
	// Also guards post eviction logic E.g. cancelling and deletion of async download job,
	// deletion of local file containing downloaded data.
	mu locker.Locker
}

func (chr *CacheHandler) getLocalFilePath(objectName string, bucketName string) string {
	return path.Join(chr.cacheLocation, bucketName, objectName)
}

func (chr *CacheHandler) createLocalFileReadHandle(objectName string, bucketName string) (*os.File, error) {
	fileSpec := data.FileSpec{
		Path: chr.getLocalFilePath(objectName, bucketName),
		Perm: DefaultFileMode,
	}

	return util.CreateFile(fileSpec, os.O_RDONLY)
}

func NewCacheHandler(fileInfoCache *lru.Cache, jobManager *downloader.JobManager, cacheLocation string) *CacheHandler {
	return &CacheHandler{
		fileInfoCache: fileInfoCache,
		jobManager:    jobManager,
		cacheLocation: cacheLocation,
	}
}

func (chr *CacheHandler) performPostEvictionWork(evictedValues []lru.ValueType) error {
	for _, val := range evictedValues {
		fileInfo := val.(data.FileInfo)
		key := fileInfo.Key

		_, err := key.Key()
		if err != nil {
			return fmt.Errorf("error while performing post eviction: %v", err)
		}

		chr.jobManager.RemoveJob(key.ObjectName, key.BucketName)

		localFilePath := chr.getLocalFilePath(key.ObjectName, key.BucketName)
		err = os.Remove(localFilePath)
		if err != nil {
			return fmt.Errorf("while deleting file: %s, error: %v", localFilePath, err)
		}
	}

	return nil
}

func (chr *CacheHandler) addFileInfoEntryInTheCacheIfNotAlready(object *gcs.MinObject, bucket gcs.Bucket) error {
	fileInfoKey := data.FileInfoKey{
		BucketName: bucket.Name(),
		ObjectName: object.Name,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	if err != nil {
		return fmt.Errorf("while creating key: %v", fileInfoKeyName)
	}

	chr.mu.Lock()
	defer chr.mu.Unlock()

	fileInfo := chr.fileInfoCache.LookUp(fileInfoKeyName)
	if fileInfo == nil {
		fileInfo = data.FileInfo{
			Key:              fileInfoKey,
			ObjectGeneration: object.Generation,
			Offset:           0,
			FileSize:         object.Size,
		}

		evictedValues, err := chr.fileInfoCache.Insert(fileInfoKeyName, fileInfo)
		if err != nil {
			return fmt.Errorf("while inserting into the cache: %v", err)
		}

		err = chr.performPostEvictionWork(evictedValues)
		if err != nil {
			return fmt.Errorf("while performing post eviction step: %v", err)
		}
	}

	return nil
}

// GetCacheHandle creates an entry in fileInfoCache if it does not already exist.
// It creates FileDownloadJob if not already exist. Also, creates localFilePath
// which contains the downloaded content. Finally, it returns a CacheHandle that
// contains the async DownloadJob and the local file handle.
func (chr *CacheHandler) GetCacheHandle(object *gcs.MinObject, bucket gcs.Bucket, initialOffset int64) (*CacheHandle, error) {
	localFileReadHandle, err := chr.createLocalFileReadHandle(object.Name, bucket.Name())
	if err != nil {
		return nil, fmt.Errorf("error while create local-file read handle: %v", err)
	}

	err = chr.addFileInfoEntryInTheCacheIfNotAlready(object, bucket)
	if err != nil {
		return nil, fmt.Errorf("while addingen try in the cache: %v", err)
	}

	return NewCacheHandle(localFileReadHandle, chr.jobManager.GetJob(object, bucket), chr.fileInfoCache, initialOffset), nil
}

// DecrementJobRefCount decrement the reference count of clients which is dependent of
// async job. This will cancel the async job once, the count reaches to zero.
// TODO (raj-prince) to implement.
func (chr *CacheHandler) DecrementJobRefCount(object *gcs.MinObject, bucket gcs.Bucket) error {
	return nil
}

// RemoveFileFromCache removes the entry from the fileInfoCache, cancel the async running job incase,
// and delete the locally downloaded cached-file.
// TODO (raj-prince) to implement.
func (chr *CacheHandler) RemoveFileFromCache(object *gcs.MinObject, bucket gcs.Bucket) {

}

// Destroy destroys the internal state of CacheHandler correctly specifically closing any fileHandles.
// TODO (raj-prince) to implement.
func (chr *CacheHandler) Destroy() {

}
