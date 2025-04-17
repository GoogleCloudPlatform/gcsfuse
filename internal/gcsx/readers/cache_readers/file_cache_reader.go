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

package cache_readers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/file"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/lru"
	cacheutil "github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx/readers"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"github.com/jacobsa/fuse/fuseops"
)

// "readOp" is the value used in read context to store pointer to the read operation.
const ReadOp = "readOp"

type FileCacheReader struct {
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

func (fc *FileCacheReader) ReadAt(ctx context.Context, p []byte, offset int64) (readers.ObjectData, error) {
	var err error
	o := readers.ObjectData{
		DataBuf: p,
		Size:    0,
	}

	// Note: If we are reading the file for the first time and read type is sequential
	// then the file cache behavior is write-through i.e. data is first read from
	// GCS, cached in file and then served from that file. But the cacheHit is
	// false in that case.
	n, cacheHit, err := fc.tryReadingFromFileCache(ctx, p, offset)
	if err != nil {
		err = fmt.Errorf("ReadAt: while reading from cache: %w", err)
		return o, err
	}
	// Data was served from cache.
	if cacheHit || n == len(p) || (n < len(p) && uint64(offset)+uint64(n) == fc.obj.Size) {
		o.Size = n
		err = readers.DontErrFallbackToAnotherReader
		return o, err
	}

	return o, err
}

func (fc *FileCacheReader) tryReadingFromFileCache(ctx context.Context, p []byte, offset int64) (n int, cacheHit bool, err error) {
	if fc.fileCacheHandler == nil {
		return 0, false, nil
	}

	// By default, consider read type random if the offset is non-zero.
	isSeq := offset == 0

	// Request log and start the execution timer.
	requestId := uuid.New()
	readOp := ctx.Value(ReadOp).(*fuseops.ReadFileOp)
	logger.Tracef("%.13v <- FileCache(%s:/%s, offset: %d, size: %d handle: %d)", requestId, fc.bucket.Name(), fc.obj.Name, offset, len(p), readOp.Handle)
	startTime := time.Now()

	// Response log
	defer func() {
		executionTime := time.Since(startTime)
		var requestOutput string
		if err != nil {
			requestOutput = fmt.Sprintf("err: %v (%v)", err, executionTime)
		} else {
			if fc.fileCacheHandle != nil {
				isSeq = fc.fileCacheHandle.IsSequential(offset)
			}
			requestOutput = fmt.Sprintf("OK (isSeq: %t, hit: %t) (%v)", isSeq, cacheHit, executionTime)
		}

		// Here rr.fileCacheHandle will not be nil since we return from the above in those cases.
		logger.Tracef("%.13v -> %s", requestId, requestOutput)

		readType := util.Random
		if isSeq {
			readType = util.Sequential
		}
		fc.captureFileCacheMetrics(ctx, fc.metricHandle, readType, n, cacheHit, executionTime)
	}()

	// Create fileCacheHandle if not already.
	if fc.fileCacheHandle == nil {
		fc.fileCacheHandle, err = fc.fileCacheHandler.GetCacheHandle(fc.obj, fc.bucket, fc.cacheFileForRangeRead, offset)
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

	n, cacheHit, err = fc.fileCacheHandle.Read(ctx, fc.bucket, fc.obj, offset, p)
	if err == nil {
		return n, cacheHit, nil
	}

	cacheHit = false
	n = 0

	if cacheutil.IsCacheHandleInvalid(err) {
		logger.Tracef("Closing cacheHandle:%p for object: %s:/%s", fc.fileCacheHandle, fc.bucket.Name(), fc.obj.Name)
		err = fc.fileCacheHandle.Close()
		if err != nil {
			logger.Warnf("tryReadingFromFileCache: while closing fileCacheHandle: %v", err)
		}
		fc.fileCacheHandle = nil
	} else if !strings.Contains(err.Error(), cacheutil.FallbackToGCSErrMsg) {
		err = fmt.Errorf("tryReadingFromFileCache: while reading via cache: %w", err)
		return n, cacheHit, err
	}
	err = nil

	return n, cacheHit, err
}

func (fc *FileCacheReader) Destroy() {
	if fc.fileCacheHandle != nil {
		logger.Tracef("Closing cacheHandle:%p for object: %s:/%s", fc.fileCacheHandle, fc.bucket.Name(), fc.obj.Name)
		err := fc.fileCacheHandle.Close()
		if err != nil {
			logger.Warnf("rr.Destroy(): while closing cacheFileHandle: %v", err)
		}
		fc.fileCacheHandle = nil
	}
}

func (fc *FileCacheReader) captureFileCacheMetrics(ctx context.Context, metricHandle common.MetricHandle, readType string, readDataSize int, cacheHit bool, readLatency time.Duration) {
	metricHandle.FileCacheReadCount(ctx, 1, []common.MetricAttr{
		{Key: common.ReadType, Value: readType},
		{Key: common.CacheHit, Value: strconv.FormatBool(cacheHit)},
	})

	metricHandle.FileCacheReadBytesCount(ctx, int64(readDataSize), []common.MetricAttr{{Key: common.ReadType, Value: readType}})
	metricHandle.FileCacheReadLatency(ctx, float64(readLatency.Microseconds()), []common.MetricAttr{{Key: common.CacheHit, Value: strconv.FormatBool(cacheHit)}})
}

func (fc *FileCacheReader) CheckInvariants() {
}
