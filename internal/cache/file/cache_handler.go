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
	"github.com/googlecloudplatform/gcsfuse/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/gcs"
)

/****** Dummy struct - will be removed ********/

// DownloadJobManager this will be removed once.
type DownloadJobManager struct {
}

// CacheHandle - this will be removed once the CacheHandle changes will be merged.
type CacheHandle struct {
}

/*************/

// CacheHandler Responsible for managing fileInfoCache as well as fileDownloadManager.
type CacheHandler struct {
	fileInfoCache *lru.Cache

	fileDownloadManager *DownloadJobManager
}

func NewCacheHandler(fileInfoCache *lru.Cache, fdm *DownloadJobManager) *CacheHandler {
	return &CacheHandler{
		fileInfoCache:       fileInfoCache,
		fileDownloadManager: fdm,
	}
}

// InitiateRead creates an entry in fileInfoCache if it does not already exist.
// It creates FileDownloadJob if not already exist. Also, creates localFilePath
// which contains the downloaded content. Finally, it returns a CacheHandle that
// contains the async DownloadJob and the local file handle.
// TODO (raj-prince) to implement.
func (ch *CacheHandler) InitiateRead(object *gcs.MinObject, bucket gcs.Bucket) (*CacheHandle, error) {
	return nil, nil
}

// DecrementJobRefCount decrement the reference count of clients which is dependent of
// async job. This will cancel the async job once, the count reaches to zero.
// TODO (raj-prince) to implement.
func (ch *CacheHandler) DecrementJobRefCount(object *gcs.MinObject, bucket gcs.Bucket) error {
	return nil
}

// RemoveFileFromCache removes the entry from the fileInfoCache, cancel the async running job incase,
// and delete the locally downloaded cached-file.
// TODO (raj-prince) to implement.
func (ch *CacheHandler) RemoveFileFromCache(object *gcs.MinObject, bucket gcs.Bucket) {

}

// Destroy destroys the internal state of CacheHandler correctly specifically closing any fileHandles.
// TODO (raj-prince) to implement.
func (ch *CacheHandler) Destroy() {

}
