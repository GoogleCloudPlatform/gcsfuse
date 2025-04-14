// Copyright 2025 Google LLC
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

package gcsx

import (
	"errors"
	"fmt"

	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/file"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/lru"
	cacheutil "github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
)

type FileCacheReader struct {
	Reader
	obj    *gcs.MinObject
	bucket gcs.Bucket

	// cacheFileForRangeRead is also valid for cache workflow, if true, object content
	// will be downloaded for random reads as well too.
	cacheFileForRangeRead bool

	// fileCacheHandle is used to read from the cached location. It is created on the fly
	// using fileCacheHandler for the given object and bucket.
	fileCacheHandle file.CacheHandleInterface

	metricHandle common.MetricHandle
}

func NewFileCacheReader(o *gcs.MinObject, bucket gcs.Bucket, fileCacheHandler file.CacheHandlerInterface, cacheFileForRangeRead bool, metricHandle common.MetricHandle, offset int64) (*FileCacheReader, error) {
	fileCacheHandle, err := getCacheHandle(o, bucket, fileCacheHandler, cacheFileForRangeRead, offset)
	if err != nil {
		return nil, err
	}

	return &FileCacheReader{
		obj:                   o,
		bucket:                bucket,
		fileCacheHandle:       fileCacheHandle,
		cacheFileForRangeRead: cacheFileForRangeRead,
		metricHandle:          metricHandle,
	}, nil
}

func getCacheHandle(o *gcs.MinObject, bucket gcs.Bucket, fileCacheHandler file.CacheHandlerInterface, cacheFileForRangeRead bool, offset int64) (file.CacheHandleInterface, error) {
	fileCacheHandle, err := fileCacheHandler.GetCacheHandle(o, bucket, cacheFileForRangeRead, offset)
	if err != nil {
		// We fall back to GCS if file size is greater than the cache size
		if errors.Is(err, lru.ErrInvalidEntrySize) {
			logger.Warnf("tryReadingFromFileCache: while creating CacheHandle: %v", err)
			return nil, nil
		} else if errors.Is(err, cacheutil.ErrCacheHandleNotRequiredForRandomRead) {
			return nil, nil
		}
		return nil, fmt.Errorf("tryReadingFromFileCache: while creating CacheHandle instance: %w", err)
	}
	return fileCacheHandle, nil
}
