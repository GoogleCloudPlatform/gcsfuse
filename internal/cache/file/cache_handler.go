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
	"os"

	"github.com/googlecloudplatform/gcsfuse/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/lru"
	"golang.org/x/net/context"
)

// This needs to remove
type LRUCache struct {
}

type FileDownloadJob struct {
}

type FileDownloadManager struct {
}

type CacheFileHandle struct {
	readFileHandle  *os.File
	fileDownloadJob FileDownloadJob
}

// Actual Implementation

// Responsible for managing fileInfoCache as well as fileDownloadManager.
type CacheHandler struct {
	fileInfoCache       *lru.Cache
	fileDownloadManager *FileDownloadManager
}

func NewFileCacheHandler(cacheSize uint64) CacheHandler {
	fileInfoCache := lru.NewCache(cacheSize)

	// Init file-download manager
	var fileDownloadManager FileDownloadManager

	// and return the handle of FileCacheHandler
	return CacheHandler{
		fileInfoCache:       &fileInfoCache,
		fileDownloadManager: &fileDownloadManager,
	}
}

func (fch *CacheHandler) HandleCache(key data.FileInfoKey) (*data.FileInfo, error) {
	cacheKey, err := key.Key()
	if err != nil {
		return nil, nil
	}

	cacheVal := fch.fileInfoCache.LookUp(cacheKey)
	if cacheVal != nil {
		fileInfo := cacheVal.(data.FileInfo)
		return &fileInfo, nil
	}
}

func (fch *CacheHandler) ReadFile(ctx context.Context, key data.FileInfoKey, offset int64, chunkSize int64, triggerDownload bool) *CacheFileHandle {

	fi, err := fch.HandleCache(key)

	if fi.Offset >= offset+chunkSize {

	}

	return nil
}
