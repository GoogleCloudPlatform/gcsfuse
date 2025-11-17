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
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/file"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	cacheUtil "github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/jacobsa/fuse/fuseops"
)

const (
	// ReadOp ("readOp") is the value used in read context to store pointer to the read operation.
	ReadOp = "readOp"
	MiB    = 1 << 20
)

// FileCacheReader is a reader that attempts to satisfy read requests for a GCS
// object from a local file cache. It is designed to be part of a layered
// reading strategy, where it acts as the first-level cache.
//
// FileCacheReader supports parallel reads.
type FileCacheReader struct {
	Reader
	object *gcs.MinObject
	bucket gcs.Bucket

	// fileCacheHandler is used to get file cache handle and read happens using that.
	// This will be nil if the file cache is disabled.
	fileCacheHandler *file.CacheHandler

	// cacheFileForRangeRead is also valid for cache workflow, if true, object content
	// will be downloaded for random reads as well too.
	cacheFileForRangeRead bool

	// To synchronize access to fileCacheHandle.
	mu sync.RWMutex

	// fileCacheHandle is used to read from the cached location. It is created on the fly
	// using fileCacheHandler for the given object and bucket.
	// GUARDED_BY(mu)
	fileCacheHandle *file.CacheHandle

	metricHandle metrics.MetricHandle
}

func NewFileCacheReader(o *gcs.MinObject, bucket gcs.Bucket, fileCacheHandler *file.CacheHandler, cacheFileForRangeRead bool, metricHandle metrics.MetricHandle) *FileCacheReader {
	return &FileCacheReader{
		object:                o,
		bucket:                bucket,
		fileCacheHandler:      fileCacheHandler,
		cacheFileForRangeRead: cacheFileForRangeRead,
		metricHandle:          metricHandle,
	}
}

// tryReadingFromFileCache creates the cache handle first if it doesn't exist already
// and then use that handle to read object's content which is cached in local file.
// For the successful read, it returns number of bytes read, and a boolean representing
// cacheHit as true.
// For unsuccessful read, returns cacheHit as false, in this case content
// should be read from GCS.
// And it returns non-nil error in case something unexpected happens during the execution.
// In this case, we must abort the Read operation.
//
// Important: What happens if the file in cache deleted externally?
// That means, we have fileInfo entry in the fileInfoCache for that deleted file.
// (a) If a new fileCacheHandle is created in that case it will return FileNotPresentInCache
// error, given by fileCacheHandler.GetCacheHandle().
// (b) If there is already an open fileCacheHandle then it means there is an open
// fileHandle to file in cache. So, we will get the correct data from fileHandle
// because Linux does not delete a file until open fileHandle count for a file is zero.
func (fc *FileCacheReader) tryReadingFromFileCache(ctx context.Context, p []byte, offset int64) (int, bool, error) {
	if fc.fileCacheHandler == nil {
		return 0, false, nil
	}

	// By default, consider read type random if the offset is non-zero.
	isSequential := offset == 0

	var handleID uint64
	if readOp, ok := ctx.Value(ReadOp).(*fuseops.ReadFileOp); ok {
		handleID = uint64(readOp.Handle)
	}
	requestID := uuid.New()
	logger.Tracef("%.13v <- FileCache(%s:/%s, offset: %d, size: %d handle: %d)", requestID, fc.bucket.Name(), fc.object.Name, offset, len(p), handleID)

	startTime := time.Now()
	var bytesRead int
	var cacheHit bool
	var err error

	defer func() {
		executionTime := time.Since(startTime)
		var requestOutput string
		if err != nil {
			requestOutput = fmt.Sprintf("err: %v (%v)", err, executionTime)
		} else {
			fc.mu.RLock()
			if fc.fileCacheHandle != nil {
				isSequential = fc.fileCacheHandle.IsSequential(offset)
			}
			fc.mu.RUnlock()
			requestOutput = fmt.Sprintf("OK (isSeq: %t, cacheHit: %t) (%v)", isSequential, cacheHit, executionTime)
		}

		logger.Tracef("%.13v -> %s", requestID, requestOutput)

		readType := metrics.ReadTypeRandom
		if isSequential {
			readType = metrics.ReadTypeSequential
		}
		captureFileCacheMetrics(ctx, fc.metricHandle, metrics.ReadTypeNames[readType], bytesRead, cacheHit, executionTime)
	}()

	// Create fileCacheHandle if not already.
	fc.mu.Lock()
	if fc.fileCacheHandle == nil {
		fc.fileCacheHandle, err = fc.fileCacheHandler.GetCacheHandle(fc.object, fc.bucket, fc.cacheFileForRangeRead, offset)
		if err != nil {
			fc.mu.Unlock()
			cacheHit = false
			bytesRead = 0
			switch {
			case errors.Is(err, lru.ErrInvalidEntrySize):
				logger.Warnf("tryReadingFromFileCache: while creating CacheHandle: %v", err)
				err = nil
				return 0, false, nil
			case errors.Is(err, cacheUtil.ErrCacheHandleNotRequiredForRandomRead):
				// Fall back to GCS if it is a random read, cacheFileForRangeRead is
				// false and there doesn't already exist file in cache.
				isSequential = false
				err = nil
				return 0, false, nil
			case errors.Is(err, cacheUtil.ErrFileExcludedFromCacheByRegex):
				err = nil
				return 0, false, nil
			default:
				err = fmt.Errorf("tryReadingFromFileCache: GetCacheHandle failed: %w", err)
				return 0, false, err
			}
		}
	}
	fc.mu.Unlock()

	fc.mu.RLock()
	// Because we're releasing write lock & then taking a read lock, we need to perform a nil check before accessing
	// fileCacheHandle as some other thread could have set it to nil in between.
	if fc.fileCacheHandle == nil {
		fc.mu.RUnlock()
		return 0, false, nil
	}
	bytesRead, cacheHit, err = fc.fileCacheHandle.Read(ctx, fc.bucket, fc.object, offset, p)
	fc.mu.RUnlock()
	if err == nil {
		return bytesRead, cacheHit, nil
	}

	bytesRead = 0
	cacheHit = false

	if cacheUtil.IsCacheHandleInvalid(err) {
		fc.mu.Lock()
		if fc.fileCacheHandle != nil {
			logger.Tracef("Closing cacheHandle:%p for object: %s:/%s", fc.fileCacheHandle, fc.bucket.Name(), fc.object.Name)
			closeErr := fc.fileCacheHandle.Close()
			if closeErr != nil {
				logger.Warnf("tryReadingFromFileCache: close cacheHandle error: %v", closeErr)
			}
			fc.fileCacheHandle = nil
		}
		fc.mu.Unlock()
	} else if !errors.Is(err, cacheUtil.ErrFallbackToGCS) {
		err = fmt.Errorf("tryReadingFromFileCache: while reading via cache: %w", err)
		return 0, false, err
	}
	err = nil

	return 0, false, nil
}

func (fc *FileCacheReader) ReadAt(ctx context.Context, p []byte, offset int64) (ReadResponse, error) {
	var readResponse ReadResponse

	if offset >= int64(fc.object.Size) {
		return readResponse, io.EOF
	}

	// Note: If we are reading the file for the first time and read type is sequential
	// then the file cache behavior is write-through i.e. data is first read from
	// GCS, cached in file and then served from that file. But the cacheHit is
	// false in that case.
	bytesRead, cacheHit, err := fc.tryReadingFromFileCache(ctx, p, offset)
	if err != nil {
		return readResponse, fmt.Errorf("ReadAt: while reading from cache: %w", err)
	}
	// Data was served from cache.
	if cacheHit || bytesRead == len(p) || (bytesRead < len(p) && uint64(offset)+uint64(bytesRead) == fc.object.Size) {
		readResponse.Size = bytesRead
		return readResponse, nil
	}

	// The cache is unable to serve data and requires a fallback to another reader.
	return readResponse, FallbackToAnotherReader
}

func captureFileCacheMetrics(ctx context.Context, metricHandle metrics.MetricHandle, readType metrics.ReadType, readDataSize int, cacheHit bool, readLatency time.Duration) {
	metricHandle.FileCacheReadCount(1, cacheHit, readType)
	metricHandle.FileCacheReadBytesCount(int64(readDataSize), readType)
	metricHandle.FileCacheReadLatencies(ctx, readLatency, cacheHit)
}

func (fc *FileCacheReader) Destroy() {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	if fc.fileCacheHandle != nil {
		logger.Tracef("Closing cacheHandle:%p for object: %s:/%s", fc.fileCacheHandle, fc.bucket.Name(), fc.object.Name)
		err := fc.fileCacheHandle.Close()
		if err != nil {
			logger.Warnf("fc.Destroy(): while closing cacheFileHandle: %v", err)
		}
		fc.fileCacheHandle = nil
	}
}

func (fc *FileCacheReader) CheckInvariants() {
}
