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

package readers

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/file"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/lru"
	cacheutil "github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx/readers/gcs_readers"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"github.com/jacobsa/fuse/fuseops"
)

// "readOp" is the value used in read context to store pointer to the read operation.
const ReadOp = "readOp"

type FileCacheReader struct {
	Obj    *gcs.MinObject
	Bucket gcs.Bucket

	// fileCacheHandler is used to get file cache handle and read happens using that.
	// This will be nil if the file cache is disabled.
	FileCacheHandler *file.CacheHandler

	// cacheFileForRangeRead is also valid for cache workflow, if true, object content
	// will be downloaded for random reads as well too.
	CacheFileForRangeRead bool

	// fileCacheHandle is used to read from the cached location. It is created on the fly
	// using fileCacheHandler for the given object and bucket.
	FileCacheHandle *file.CacheHandle

	MetricHandle common.MetricHandle
}

func (fc *FileCacheReader) Object() *gcs.MinObject {
	return fc.Obj
}

func (fc *FileCacheReader) CheckInvariants() {
}

func (fc *FileCacheReader) ReadAt(ctx context.Context, p []byte, offset int64) (gcs_readers.ObjectData, error) {
	var err error
	o := gcs_readers.ObjectData{
		DataBuf:  p,
		CacheHit: false,
		Size:     0,
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
	if cacheHit || n == len(p) || (n < len(p) && uint64(offset)+uint64(n) == fc.Obj.Size) {
		o.CacheHit = cacheHit
		o.Size = n
	}

	return o, nil
}

func (fc *FileCacheReader) tryReadingFromFileCache(ctx context.Context, p []byte, offset int64) (int, bool, error) {
	var err error
	var n int
	var cacheHit bool

	if fc.FileCacheHandler == nil {
		return 0, false, nil
	}

	// By default, consider read type random if the offset is non-zero.
	isSeq := offset == 0
	log.Println("Is Seq in start: ", isSeq)

	// Request log and start the execution timer.
	requestId := uuid.New()
	readOp := ctx.Value(ReadOp).(*fuseops.ReadFileOp)
	logger.Tracef("%.13v <- FileCache(%s:/%s, offset: %d, size: %d handle: %d)", requestId, fc.Bucket.Name(), fc.Obj.Name, offset, len(p), readOp.Handle)
	startTime := time.Now()

	// Response log
	defer func() {
		executionTime := time.Since(startTime)
		var requestOutput string
		if err != nil {
			requestOutput = fmt.Sprintf("err: %v (%v)", err, executionTime)
		} else {
			if fc.FileCacheHandle != nil {
				isSeq = fc.FileCacheHandle.IsSequential(offset)
				log.Println("Is Sequentaillll: ", isSeq)
			}
			requestOutput = fmt.Sprintf("OK (isSeq: %t, hit: %t) (%v)", isSeq, cacheHit, executionTime)
		}

		// Here rr.fileCacheHandle will not be nil since we return from the above in those cases.
		logger.Tracef("%.13v -> %s", requestId, requestOutput)

		readType := util.Random
		if isSeq {
			readType = util.Sequential
		}
		log.Println("Is Seq2222: ", isSeq, readType)
		captureFileCacheMetrics(ctx, fc.MetricHandle, readType, n, cacheHit, executionTime)
	}()

	log.Println("Is Seq: ", isSeq)
	// Create fileCacheHandle if not already.
	if fc.FileCacheHandle == nil {
		fc.FileCacheHandle, err = fc.FileCacheHandler.GetCacheHandle(fc.Obj, fc.Bucket, fc.CacheFileForRangeRead, offset)
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

	n, cacheHit, err = fc.FileCacheHandle.Read(ctx, fc.Bucket, fc.Obj, offset, p)
	if err == nil {
		return n, cacheHit, nil
	}

	cacheHit = false
	n = 0

	if cacheutil.IsCacheHandleInvalid(err) {
		logger.Tracef("Closing cacheHandle:%p for object: %s:/%s", fc.FileCacheHandle, fc.Bucket.Name(), fc.Obj.Name)
		err = fc.FileCacheHandle.Close()
		if err != nil {
			logger.Warnf("tryReadingFromFileCache: while closing fileCacheHandle: %v", err)
		}
		fc.FileCacheHandle = nil
	} else if !strings.Contains(err.Error(), cacheutil.FallbackToGCSErrMsg) {
		err = fmt.Errorf("tryReadingFromFileCache: while reading via cache: %w", err)
		return 0, false, err
	}
	err = nil

	return n, cacheHit, nil
}

func (fc *FileCacheReader) Destroy() {
	if fc.FileCacheHandle != nil {
		logger.Tracef("Closing cacheHandle:%p for object: %s:/%s", fc.FileCacheHandle, fc.Bucket.Name(), fc.Obj.Name)
		err := fc.FileCacheHandle.Close()
		if err != nil {
			logger.Warnf("rr.Destroy(): while closing cacheFileHandle: %v", err)
		}
		fc.FileCacheHandle = nil
	}
}

func captureFileCacheMetrics(ctx context.Context, metricHandle common.MetricHandle, readType string, readDataSize int, cacheHit bool, readLatency time.Duration) {
	metricHandle.FileCacheReadCount(ctx, 1, []common.MetricAttr{
		{Key: common.ReadType, Value: readType},
		{Key: common.CacheHit, Value: strconv.FormatBool(cacheHit)},
	})

	metricHandle.FileCacheReadBytesCount(ctx, int64(readDataSize), []common.MetricAttr{{Key: common.ReadType, Value: readType}})
	metricHandle.FileCacheReadLatency(ctx, float64(readLatency.Microseconds()), []common.MetricAttr{{Key: common.CacheHit, Value: strconv.FormatBool(cacheHit)}})
}
