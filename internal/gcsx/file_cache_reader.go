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
	"fmt"
	"strings"
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
	object                *gcs.MinObject
	bucket                gcs.Bucket
	handler               *file.CacheHandler
	handle                *file.CacheHandle
	metricHandle          common.MetricHandle
	cacheEnabled          bool
	cacheFileForRangeRead bool
}

func NewFileCacheReader(handler *file.CacheHandler, cacheEnabled bool) *FileCacheReader {
	return &FileCacheReader{handler: handler, cacheEnabled: cacheEnabled}
}

func (fc *FileCacheReader) ReadFromCache(ctx context.Context, bucket gcs.Bucket, obj *gcs.MinObject, p []byte, offset int64) (int, bool, error) {
	var err error
	var n int
	var cacheHit bool
	if fc.handler == nil {
		return 0, false, nil
	}
	// By default, consider read type random if the offset is non-zero.
	isSeq := offset == 0

	// Request log and start the execution timer.
	requestId := uuid.New()
	readOp := ctx.Value(ReadOp).(*fuseops.ReadFileOp)
	logger.Tracef("%.13v <- FileCache(%s:/%s, offset: %d, size: %d handle: %d)", requestId, bucket.Name(), obj.Name, offset, len(p), readOp.Handle)
	startTime := time.Now()

	// Response log
	defer func() {
		executionTime := time.Since(startTime)
		var requestOutput string
		if err != nil {
			requestOutput = fmt.Sprintf("err: %v (%v)", err, executionTime)
		} else {
			if fc.handler != nil {
				isSeq = fc.handle.IsSequential(offset)
			}
			requestOutput = fmt.Sprintf("OK (isSeq: %t, hit: %t) (%v)", isSeq, cacheHit, executionTime)
		}

		// Here fc.fileCacheHandle will not be nil since we return from the above in those cases.
		logger.Tracef("%.13v -> %s", requestId, requestOutput)

		readType := util.Random
		if isSeq {
			readType = util.Sequential
		}
		captureFileCacheMetrics(ctx, fc.metricHandle, readType, n, cacheHit, executionTime)
	}()

	// Create fileCacheHandle if not already.
	if fc.handler == nil {
		fc.handle, err = fc.handler.GetCacheHandle(obj, bucket, fc.cacheFileForRangeRead, offset)
		if err != nil {
			// We fall back to GCS if file size is greater than the cache size
			if strings.Contains(err.Error(), lru.InvalidEntrySizeErrorMsg) {
				logger.Warnf("tryReadingFromFileCache: while creating CacheHandle: %v", err)
				return 0, false, nil
			} else if strings.Contains(err.Error(), cacheutil.CacheHandleNotRequiredForRandomReadErrMsg) {
				// Fall back to GCS if it is a random read, cacheFileForRangeRead is
				// False and there doesn't already exist file in cache.
				isSeq = false
				return 0, false, nil
			}

			return 0, false, fmt.Errorf("tryReadingFromFileCache: while creating CacheHandle instance: %w", err)
		}
	}

	n, cacheHit, err = fc.handle.Read(ctx, bucket, obj, offset, p)
	if err == nil {
		return n, cacheHit, err
	}

	cacheHit = false
	n = 0

	if cacheutil.IsCacheHandleInvalid(err) {
		logger.Tracef("Closing cacheHandle:%p for object: %s:/%s", fc.handler, bucket.Name(), obj.Name)
		err = fc.handle.Close()
		if err != nil {
			logger.Warnf("tryReadingFromFileCache: while closing fileCacheHandle: %v", err)
		}
		fc.handler = nil
	} else if !strings.Contains(err.Error(), cacheutil.FallbackToGCSErrMsg) {
		err = fmt.Errorf("tryReadingFromFileCache: while reading via cache: %w", err)
		return n, cacheHit, err
	}

	return n, cacheHit, nil
}

func (fc *FileCacheReader) Destroy() {
	if fc.handler != nil {
		logger.Tracef("Closing cacheHandle:%p for object: %s:/%s", fc.handler, fc.bucket.Name(), fc.object.Name)
		err := fc.handle.Close()
		if err != nil {
			logger.Warnf("rr.Destroy(): while closing cacheFileHandle: %v", err)
		}
		fc.handle = nil
	}
}

// closeReader fetches the readHandle before closing the reader instance.
func (fc *FileCacheReader) closeReader() {
}
