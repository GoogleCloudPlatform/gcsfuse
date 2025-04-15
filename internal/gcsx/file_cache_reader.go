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
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/file"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/lru"
	cacheutil "github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"github.com/jacobsa/fuse/fuseops"
)

type FileCacheReader struct {
	Reader
	obj    *gcs.MinObject
	bucket gcs.Bucket

	// fileCacheHandler is used to get file cache handle and read happens using that.
	// This will be nil if the file cache is disabled.
	fileCacheHandler *file.CacheHandler

	// cacheFileForRangeRead is also valid for cache workflow, if true, object content
	// will be downloaded for random reads as well too.
	cacheFileForRangeRead bool

	// fileCacheHandle is used to read from the cached location. It is created on the fly
	// using fileCacheHandler for the given object and bucket.
	fileCacheHandle *file.CacheHandle

	metricHandle common.MetricHandle
}

func NewFileCacheReader(o *gcs.MinObject, bucket gcs.Bucket, fileCacheHandler *file.CacheHandler, cacheFileForRangeRead bool, metricHandle common.MetricHandle) FileCacheReader {
	return FileCacheReader{
		obj:                   o,
		bucket:                bucket,
		fileCacheHandler:      fileCacheHandler,
		cacheFileForRangeRead: cacheFileForRangeRead,
		metricHandle:          metricHandle,
	}
}

func (fc *FileCacheReader) tryReadingFromFileCache(ctx context.Context, p []byte, offset int64) (int, bool, error) {
	if fc.fileCacheHandler == nil {
		return 0, false, nil
	}

	isSequential := offset == 0
	requestID := uuid.New()

	var handleID uint64
	if readOp, ok := ctx.Value(ReadOp).(*fuseops.ReadFileOp); ok {
		handleID = uint64(readOp.Handle)
	}

	logger.Tracef("%.13v <- FileCache(%s:/%s, offset: %d, size: %d, handle: %d)", requestID, fc.bucket.Name(), fc.obj.Name, offset, len(p), handleID)

	start := time.Now()
	var bytesRead int
	var hit bool
	var err error

	defer func() {
		duration := time.Since(start)

		if fc.fileCacheHandle != nil {
			isSequential = fc.fileCacheHandle.IsSequential(offset)
		}

		readType := util.Random
		if isSequential {
			readType = util.Sequential
		}

		result := fmt.Sprintf("OK (isSeq: %t, hit: %t)", isSequential, hit)
		if err != nil {
			result = fmt.Sprintf("err: %v", err)
		}

		logger.Tracef("%.13v -> %s (%v)", requestID, result, duration)
		fc.captureFileCacheMetrics(ctx, fc.metricHandle, readType, bytesRead, hit, duration)
	}()

	// Lazy handle creation
	if fc.fileCacheHandle == nil {
		fc.fileCacheHandle, err = fc.fileCacheHandler.GetCacheHandle(fc.obj, fc.bucket, fc.cacheFileForRangeRead, offset)
		if err != nil {
			switch {
			case errors.Is(err, lru.ErrInvalidEntrySize):
				logger.Warnf("cache handle creation failed due to size constraints: %v", err)
				return 0, false, nil
			case errors.Is(err, cacheutil.ErrCacheHandleNotRequiredForRandomRead):
				return 0, false, nil
			default:
				return 0, false, fmt.Errorf("tryReadingFromFileCache: getCacheHandle failed: %w", err)
			}
		}
	}

	bytesRead, hit, err = fc.fileCacheHandle.Read(ctx, fc.bucket, fc.obj, offset, p)
	if err == nil {
		return bytesRead, hit, nil
	}

	bytesRead = 0
	hit = false
	// On read error
	if cacheutil.IsCacheHandleInvalid(err) {
		logger.Tracef("Closing cacheHandle:%p for object: %s:/%s", fc.fileCacheHandle, fc.bucket.Name(), fc.obj.Name)
		closeErr := fc.fileCacheHandle.Close()
		if closeErr != nil {
			logger.Warnf("tryReadingFromFileCache: close cacheHandle error: %v", closeErr)
		}
		fc.fileCacheHandle = nil
	} else if !errors.Is(err, cacheutil.ErrFallbackToGCS) {
		return 0, false, fmt.Errorf("tryReadingFromFileCache: read failed: %w", err)
	}
	err = nil

	return 0, false, nil
}

func (fc *FileCacheReader) captureFileCacheMetrics(ctx context.Context, metricHandle common.MetricHandle, readType string, readDataSize int, cacheHit bool, readLatency time.Duration) {
	metricHandle.FileCacheReadCount(ctx, 1, []common.MetricAttr{
		{Key: common.ReadType, Value: readType},
		{Key: common.CacheHit, Value: strconv.FormatBool(cacheHit)},
	})

	metricHandle.FileCacheReadBytesCount(ctx, int64(readDataSize), []common.MetricAttr{{Key: common.ReadType, Value: readType}})
	metricHandle.FileCacheReadLatency(ctx, float64(readLatency.Microseconds()), []common.MetricAttr{{Key: common.CacheHit, Value: strconv.FormatBool(cacheHit)}})
}
