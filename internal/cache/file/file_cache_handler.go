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
type FileCacheHandler struct {
	lruCache            LRUCache
	fileDownloadManager FileDownloadManager
}

func NewFileCacheHandler() FileCacheHandler {
	// Init lru-cache
	// Init file-download manager

	// and return the handle of FileCacheHandler
	return FileCacheHandler{}
}

func (fch *FileCacheHandler) ReadFile(ctx context.Context, fileName string, fileSize int64, triggerDownload bool) *CacheFileHandle {
	return nil
}
