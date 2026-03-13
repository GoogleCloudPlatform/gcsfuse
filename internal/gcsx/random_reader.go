// Copyright 2015 Google LLC
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
	"io"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/file"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	cacheutil "github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/gcsfuse_errors"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/googlecloudplatform/gcsfuse/v3/tracing"
	"github.com/jacobsa/fuse/fuseops"
	"golang.org/x/net/context"
)

// Min read size in bytes for random reads.
// We will not send a request to GCS for less than this many bytes (unless the
// end of the object comes first).
const minReadSize = MiB

// Max read size in bytes for random reads.
// If the average read size (between seeks) is below this number, reads will
// optimised for random access.
// We will skip forwards in a GCS response at most this many bytes.
// About 6 MiB of data is buffered anyway, so 8 MiB seems like a good round number.
const maxReadSize = 8 * MiB

// Minimum number of seeks before evaluating if the read pattern is random.
const minSeeksForRandom = 2

// TODO(b/385826024): Revert timeout to an appropriate value
const TimeoutForMultiRangeRead = time.Hour

var FallbackToNewRangeReader = errors.New("fallback to new range reader is required")

// RandomReader is an object that knows how to read ranges within a particular
// generation of a particular GCS object. Optimised for (large) sequential reads.
//
// Not safe for concurrent access.
//
// TODO - (raj-prince) - Rename this with appropriate name as it also started
// fulfilling the responsibility of reading object's content from cache.
type RandomReader interface {
	// Panic if any internal invariants are violated.
	CheckInvariants()

	// ReadAt returns the data from the requested offset and upto the size of input
	// byte array. It either populates input array i.e., p or returns a different
	// byte array. In case input array is populated, the same array will be returned
	// as part of response. Hence the callers should use the byte array returned
	// as part of response always.
	ReadAt(ctx context.Context, p []byte, offset int64) (objectData ObjectData, err error)

	// Return the record for the object to which the reader is bound.
	Object() (o *gcs.MinObject)

	// Clean up any resources associated with the reader, which must not be used
	// again.
	Destroy()
}

// ObjectData specifies the response returned as part of ReadAt call.
type ObjectData struct {
	// Byte array populated with the requested data.
	DataBuf []byte
	// Size of the data returned.
	Size int
	// Specified whether data is served from cache or not.
	CacheHit bool
}

// ReaderType represents different types of go-sdk gcs readers.
// For eg: NewReader and MRD both point to bidi read api. This enum specifies
// the go-sdk type.
type ReaderType int

// ReaderType enum values.
const (
	// RangeReader corresponds to NewReader method in bucket_handle.go
	RangeReader ReaderType = iota
	// MultiRangeReader corresponds to NewMultiRangeDownloader method in bucket_handle.go
	MultiRangeReader
)

// NewRandomReader create a random reader for the supplied object record that
// reads using the given bucket.
func NewRandomReader(o *gcs.MinObject, bucket gcs.Bucket, sequentialReadSizeMb int32, fileCacheHandler *file.CacheHandler, cacheFileForRangeRead bool, metricHandle metrics.MetricHandle, traceHandle tracing.TraceHandle, mrdWrapper *MultiRangeDownloaderWrapper, config *cfg.Config, handleID fuseops.HandleID) RandomReader {
	return &randomReader{
		object:                o,
		bucket:                bucket,
		start:                 -1,
		limit:                 -1,
		sequentialReadSizeMb:  sequentialReadSizeMb,
		fileCacheHandler:      fileCacheHandler,
		cacheFileForRangeRead: cacheFileForRangeRead,
		mrdWrapper:            mrdWrapper,
		metricHandle:          metricHandle,
		traceHandle:           traceHandle,
		config:                config,
		handleID:              handleID,
	}
}

type randomReader struct {
	object *gcs.MinObject
	bucket gcs.Bucket

	// If non-nil, an in-flight read request and a function for cancelling it.
	//
	// INVARIANT: (reader == nil) == (cancel == nil)
	reader gcs.StorageReader
	cancel func()

	// The range of the object that we expect reader to yield, when reader is
	// non-nil. When reader is nil, limit is the limit of the previous read
	// operation, or -1 if there has never been one.
	//
	// INVARIANT: start <= limit
	// INVARIANT: limit < 0 implies reader != nil
	// All these properties will be used only in case of GCS reads and not for
	// reads from cache.
	start          int64
	limit          int64
	seeks          atomic.Uint64
	totalReadBytes atomic.Uint64

	// ReadType of the reader. Will be sequential by default.
	readType atomic.Int64

	sequentialReadSizeMb int32

	// fileCacheHandler is used to get file cache handle and read happens using that.
	// This will be nil if the file cache is disabled.
	fileCacheHandler *file.CacheHandler

	// cacheFileForRangeRead is also valid for cache workflow, if true, object content
	// will be downloaded for random reads as well too.
	cacheFileForRangeRead bool

	// fileCacheHandle is used to read from the cached location. It is created on the fly
	// using fileCacheHandler for the given object and bucket.
	fileCacheHandle *file.CacheHandle

	// Stores the handle associated with the previously closed newReader instance.
	// This will be used while making the new connection to bypass auth and metadata
	// checks.
	readHandle []byte

	handleID fuseops.HandleID

	// mrdWrapper points to the wrapper object within inode.
	mrdWrapper *MultiRangeDownloaderWrapper

	// boolean variable to determine if MRD is being used or not.
	isMRDInUse atomic.Bool

	metricHandle metrics.MetricHandle

	traceHandle tracing.TraceHandle

	config *cfg.Config

	// Specifies the next expected offset for the reads. Used to distinguish between
	// sequential and random reads.
	expectedOffset atomic.Int64

	// To synchronize reads served from range reader.
	mu sync.Mutex

	// To synchronize access to fileCacheHandle
	fileCacheMu sync.RWMutex
}

type readInfo struct {
	readType       int64
	expectedOffset int64
	seekRecorded   bool
}

func (rr *randomReader) CheckInvariants() {
	// INVARIANT: (reader == nil) == (cancel == nil)
	if (rr.reader == nil) != (rr.cancel == nil) {
		panic(fmt.Sprintf("Mismatch: %v vs. %v", rr.reader == nil, rr.cancel == nil))
	}

	// INVARIANT: start <= limit
	if !(rr.start <= rr.limit) {
		panic(fmt.Sprintf("Unexpected range: [%d, %d)", rr.start, rr.limit))
	}

	// INVARIANT: limit < 0 implies reader != nil
	if rr.limit < 0 && rr.reader != nil {
		panic(fmt.Sprintf("Unexpected non-nil reader with limit == %d", rr.limit))
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
func (rr *randomReader) tryReadingFromFileCache(ctx context.Context,
	p []byte,
	offset int64) (n int, cacheHit bool, err error) {

	if rr.fileCacheHandler == nil {
		return
	}

	// By default, consider read type random if the offset is non-zero.
	isSeq := offset == 0

	// Request log and start the execution timer.
	requestId := uuid.New()
	logger.Tracef("%.13v <- FileCache(%s:/%s, offset: %d, size: %d handle: %d)", requestId, rr.bucket.Name(), rr.object.Name, offset, len(p), rr.handleID)
	startTime := time.Now()

	// Response log
	defer func() {
		executionTime := time.Since(startTime)
		var requestOutput string
		if err != nil {
			requestOutput = fmt.Sprintf("err: %v (%v)", err, executionTime)
		} else {
			rr.fileCacheMu.RLock()
			if rr.fileCacheHandle != nil {
				isSeq = rr.fileCacheHandle.IsSequential(offset)
			}
			rr.fileCacheMu.RUnlock()
			requestOutput = fmt.Sprintf("OK (isSeq: %t, hit: %t) (%v)", isSeq, cacheHit, executionTime)
		}

		// Here rr.fileCacheHandle will not be nil since we return from the above in those cases.
		logger.Tracef("%.13v -> %s", requestId, requestOutput)

		readType := metrics.ReadTypeRandom
		if isSeq {
			readType = metrics.ReadTypeSequential
		}
		captureFileCacheMetrics(ctx, rr.metricHandle, metrics.ReadTypeNames[readType], n, cacheHit, executionTime)
	}()

	// Create fileCacheHandle if not already.
	rr.fileCacheMu.Lock()
	if rr.fileCacheHandle == nil {
		rr.fileCacheHandle, err = rr.fileCacheHandler.GetCacheHandle(rr.object, rr.bucket, rr.cacheFileForRangeRead, offset)
		if err != nil {
			rr.fileCacheMu.Unlock()
			// We fall back to GCS if file size is greater than the cache size
			if errors.Is(err, lru.ErrInvalidEntrySize) {
				logger.Warnf("tryReadingFromFileCache: while creating CacheHandle: %v", err)
				return 0, false, nil
			} else if errors.Is(err, cacheutil.ErrCacheHandleNotRequiredForRandomRead) {
				// Fall back to GCS if it is a random read, cacheFileForRangeRead is
				// False and there doesn't already exist file in cache.
				isSeq = false
				return 0, false, nil
			} else if errors.Is(err, cacheutil.ErrFileExcludedFromCacheByRegex) {
				// Fall back to GCS if the file is explicitly excluded from cache.
				return 0, false, nil
			}

			return 0, false, fmt.Errorf("tryReadingFromFileCache: while creating CacheHandle instance: %w", err)
		}
	}
	rr.fileCacheMu.Unlock()

	rr.fileCacheMu.RLock()
	if rr.fileCacheHandle == nil {
		rr.fileCacheMu.RUnlock()
		return
	}
	n, cacheHit, err = rr.fileCacheHandle.Read(ctx, rr.bucket, rr.object, offset, p)
	rr.fileCacheMu.RUnlock()
	if err == nil {
		return
	}

	cacheHit = false
	n = 0

	if cacheutil.IsCacheHandleInvalid(err) {
		rr.fileCacheMu.Lock()
		if rr.fileCacheHandle != nil {
			logger.Tracef("Closing cacheHandle:%p for object: %s:/%s", rr.fileCacheHandle, rr.bucket.Name(), rr.object.Name)
			err = rr.fileCacheHandle.Close()
			if err != nil {
				logger.Warnf("tryReadingFromFileCache: while closing fileCacheHandle: %v", err)
			}
			rr.fileCacheHandle = nil
		}
		rr.fileCacheMu.Unlock()
	} else if !errors.Is(err, cacheutil.ErrFallbackToGCS) {
		err = fmt.Errorf("tryReadingFromFileCache: while reading via cache: %w", err)
		return
	}
	err = nil

	return
}

func (rr *randomReader) ReadAt(
	ctx context.Context,
	p []byte,
	offset int64) (objectData ObjectData, err error) {
	objectData = ObjectData{
		DataBuf:  p,
		CacheHit: false,
		Size:     0,
	}

	if offset >= int64(rr.object.Size) {
		err = io.EOF
		return
	} else if offset < 0 {
		err = fmt.Errorf(
			"illegal offset %d for %d byte object",
			offset,
			rr.object.Size)
		return
	}

	// Note: If we are reading the file for the first time and read type is sequential
	// then the file cache behavior is write-through i.e. data is first read from
	// GCS, cached in file and then served from that file. But the cacheHit is
	// false in that case.
	n, cacheHit, err := rr.tryReadingFromFileCache(ctx, p, offset)
	if err != nil {
		err = fmt.Errorf("ReadAt: while reading from cache: %w", err)
		return
	}
	// Data was served from cache.
	if cacheHit || n == len(p) || (n < len(p) && uint64(offset)+uint64(n) == rr.object.Size) {
		objectData.CacheHit = cacheHit
		objectData.Size = n
		return
	}

	// Not taking any lock for getting reader type to ensure random read requests do not wait.
	readInfo := rr.getReadInfo(offset, false)
	reqReaderType := readerType(readInfo.readType, rr.bucket.BucketType())

	if reqReaderType == RangeReader {
		rr.mu.Lock()
		expectedOffset := rr.expectedOffset.Load()

		// Calculating reader type again for zonal buckets in case another read has been served
		// since last computation. This is to ensure that we don't use range reader incorrectly
		// when MRD should've been used.
		if rr.bucket.BucketType().Zonal && readInfo.expectedOffset != expectedOffset {
			readInfo = rr.getReadInfo(offset, readInfo.seekRecorded)
			reqReaderType = readerType(readInfo.readType, rr.bucket.BucketType())
		}

		if reqReaderType == MultiRangeReader {
			rr.mu.Unlock()
		} else {
			defer rr.mu.Unlock()

			// Check first if we can read using existing reader. if not, create a new range reader
			objectData.Size, err = rr.readFromExistingRangeReader(ctx, p, offset)
			if errors.Is(err, FallbackToNewRangeReader) {
				// reader does not exist and need to be created, get the end offset.
				end := rr.getEndOffset(offset)
				objectData.Size, err = rr.readFromRangeReader(ctx, p, offset, end, readInfo.readType)
			}
			return
		}
	}

	if reqReaderType == MultiRangeReader {
		objectData.Size, err = rr.readFromMultiRangeReader(ctx, p, offset, offset+int64(len(p)), TimeoutForMultiRangeRead)
	}

	return
}

func (rr *randomReader) Object() (o *gcs.MinObject) {
	o = rr.object
	return
}

func (rr *randomReader) Destroy() {
	defer func() {
		if rr.isMRDInUse.Load() {
			err := rr.mrdWrapper.DecrementRefCount()
			if err != nil {
				logger.Errorf("randomReader::Destroy:%v", err)
			}
			rr.isMRDInUse.Store(false)
		}
	}()

	// Close out the reader, if we have one.
	if rr.reader != nil {
		rr.mu.Lock()
		defer rr.mu.Unlock()
		if rr.reader != nil {
			rr.closeReader()
		}
		rr.reader = nil
		rr.cancel = nil
	}

	rr.fileCacheMu.Lock()
	defer rr.fileCacheMu.Unlock()
	if rr.fileCacheHandle != nil {
		logger.Tracef("Closing cacheHandle:%p for object: %s:/%s", rr.fileCacheHandle, rr.bucket.Name(), rr.object.Name)
		err := rr.fileCacheHandle.Close()
		if err != nil {
			logger.Warnf("rr.Destroy(): while closing cacheFileHandle: %v", err)
		}
		rr.fileCacheHandle = nil
	}
}

// Like io.ReadFull, but deals with the cancellation issues.
//
// REQUIRES: rr.reader != nil
func (rr *randomReader) readFull(
	ctx context.Context,
	p []byte) (n int, err error) {
	if rr.config != nil && !rr.config.FileSystem.IgnoreInterrupts {
		// Start a goroutine that will cancel the read operation we block on below if
		// the calling context is cancelled, but only if this method has not already
		// returned (to avoid souring the reader for the next read if this one is
		// successful, since the calling context will eventually be cancelled).
		readDone := make(chan struct{})
		defer close(readDone)

		go func() {
			select {
			case <-readDone:
				return

			case <-ctx.Done():
				select {
				case <-readDone:
					return

				default:
					rr.cancel()
				}
			}
		}()
	}

	// Call through.
	n, err = io.ReadFull(rr.reader, p)

	return
}

// Ensure that rr.reader is set up for a range for which [start, start+size) is
// a prefix. Irrespective of the size requested, we try to fetch more data
// from GCS defined by sequentialReadSizeMb flag to serve future read requests.
func (rr *randomReader) startRead(start int64, end int64, readType int64) (err error) {
	// Begin the read.
	ctx, cancel := context.WithCancel(context.Background())

	if rr.config != nil && rr.config.Read.InactiveStreamTimeout > 0 {
		rr.reader, err = NewInactiveTimeoutReader(
			ctx,
			rr.bucket,
			rr.object,
			rr.readHandle,
			gcs.ByteRange{
				Start: uint64(start),
				Limit: uint64(end),
			},
			rr.config.Read.InactiveStreamTimeout)
	} else {
		rr.reader, err = rr.bucket.NewReaderWithReadHandle(
			ctx,
			&gcs.ReadObjectRequest{
				Name:       rr.object.Name,
				Generation: rr.object.Generation,
				Range: &gcs.ByteRange{
					Start: uint64(start),
					Limit: uint64(end),
				},
				ReadCompressed: rr.object.HasContentEncodingGzip(),
				ReadHandle:     rr.readHandle,
			})
	}

	// If a file handle is open locally, but the corresponding object doesn't exist
	// in GCS, it indicates a file clobbering scenario. This likely occurred because:
	//  - The file was deleted in GCS while a local handle was still open.
	//  - The file content was modified leading to different generation number.
	var notFoundError *gcs.NotFoundError
	if errors.As(err, &notFoundError) {
		err = &gcsfuse_errors.FileClobberedError{
			Err:        fmt.Errorf("NewReader: %w", err),
			ObjectName: rr.object.Name,
		}
		return
	}

	if err != nil {
		err = fmt.Errorf("NewReaderWithReadHandle: %w", err)
		return
	}

	rr.cancel = cancel
	rr.start = start
	rr.limit = end

	requestedDataSize := end - start
	metrics.CaptureGCSReadMetrics(rr.metricHandle, metrics.ReadTypeNames[readType], requestedDataSize)

	return
}

// isSeekNeeded determines if the current read at `offset` should be considered a
// seek, given the previous read pattern & the expected offset.
func isSeekNeeded(readType, offset, expectedOffset int64) bool {
	if expectedOffset == 0 {
		return false
	}

	if readType == metrics.ReadTypeRandom {
		return offset != expectedOffset
	}

	if readType == metrics.ReadTypeSequential {
		return offset < expectedOffset || offset > expectedOffset+maxReadSize
	}

	return false
}

// getReadInfo determines the read strategy (sequential or random) for a read
// request at a given offset and returns read metadata. It also updates the
// reader's internal state based on the read pattern.
func (rr *randomReader) getReadInfo(offset int64, seekRecorded bool) readInfo {
	readType := rr.readType.Load()
	expOffset := rr.expectedOffset.Load()
	numSeeks := rr.seeks.Load()

	if !seekRecorded && isSeekNeeded(readType, offset, expOffset) {
		numSeeks = rr.seeks.Add(1)
		seekRecorded = true
	}

	if numSeeks >= minSeeksForRandom {
		readType = metrics.ReadTypeRandom
	}

	averageReadBytes := rr.totalReadBytes.Load()
	if numSeeks > 0 {
		averageReadBytes /= numSeeks
	}

	if averageReadBytes >= maxReadSize {
		readType = metrics.ReadTypeSequential
	}

	rr.readType.Store(readType)
	return readInfo{
		readType:       readType,
		expectedOffset: expOffset,
		seekRecorded:   seekRecorded,
	}
}

// getEndOffset returns the end offset for the range to query GCS.
// Range here is [start, end]. End is computed for sequential reads using
// start offset and size of the data the callers needs.
func (rr *randomReader) getEndOffset(
	start int64) (end int64) {
	// GCS requests are expensive. Prefer to issue read requests defined by
	// sequentialReadSizeMb flag. Sequential reads will simply sip from the fire house
	// with each call to ReadAt. In practice, GCS will fill the TCP buffers
	// with about 6 MiB of data. Requests from outside GCP will be charged
	// about 6MB of egress data, even if less data is read. Inside GCP
	// regions, GCS egress is free. This logic should limit the number of
	// GCS read requests, which are not free.

	// But if we notice random read patterns after a minimum number of seeks,
	// optimise for random reads. Random reads will read data in chunks of
	// (average read size in bytes rounded up to the next MiB).
	end = int64(rr.object.Size)
	if seeks := rr.seeks.Load(); seeks >= minSeeksForRandom {
		averageReadBytes := rr.totalReadBytes.Load() / seeks
		if averageReadBytes < maxReadSize {
			randomReadSize := max(int64(((averageReadBytes/MiB)+1)*MiB), minReadSize)
			if randomReadSize > maxReadSize {
				randomReadSize = maxReadSize
			}
			end = start + randomReadSize
		}
	}
	if end > int64(rr.object.Size) {
		end = int64(rr.object.Size)
	}

	// To avoid overloading GCS and to have reasonable latencies, we will only
	// fetch data of max size defined by sequentialReadSizeMb.
	maxSizeToReadFromGCS := int64(rr.sequentialReadSizeMb * MiB)
	if end-start > maxSizeToReadFromGCS {
		end = start + maxSizeToReadFromGCS
	}

	return
}

// readerType specifies the go-sdk interface to use for reads.
func readerType(readType int64, bucketType gcs.BucketType) ReaderType {
	if readType == metrics.ReadTypeRandom && bucketType.Zonal {
		return MultiRangeReader
	}
	return RangeReader
}

// skipBytes attempts to advance the reader position to the given offset without
// discarding the existing reader.
// LOCKS_REQUIRED (rr.mu)
func (rr *randomReader) skipBytes(offset int64) {
	// When the offset is AFTER the reader position, try to seek forward, within reason.
	// This happens when the kernel page cache serves some data. It's very common for
	// concurrent reads, often by only a few 128kB fuse read requests. The aim is to
	// re-use GCS connection and avoid throwing away already read data.
	// For parallel sequential reads to a single file, not throwing away the connections
	// is a 15-20x improvement in throughput: 150-200 MiB/s instead of 10 MiB/s.
	if rr.reader != nil && rr.start < offset && offset-rr.start < maxReadSize {
		bytesToSkip := offset - rr.start
		discardedBytes, copyError := io.CopyN(io.Discard, rr.reader, bytesToSkip)
		// io.EOF is expected if the reader is shorter than the requested offset to read.
		if copyError != nil && !errors.Is(copyError, io.EOF) {
			logger.Warnf("Error while skipping reader bytes: %v", copyError)
		}
		rr.start += discardedBytes
	}
}

// invalidateReaderIfMisalignedOrTooSmall ensures that the existing reader is valid
// for the requested offset and length. If the reader is misaligned (not at the requested
// offset) or cannot serve the full request within its limit, it is closed and discarded.
// LOCKS_REQUIRED (rr.mu)
func (rr *randomReader) invalidateReaderIfMisalignedOrTooSmall(startOffset, endOffset int64) {
	// If we have an existing reader, but it's positioned at the wrong place,
	// clean it up and throw it away.
	// We will also clean up the existing reader if it can't serve the entire request.
	dataToRead := math.Min(float64(endOffset), float64(rr.object.Size))
	if rr.reader != nil && (rr.start != startOffset || int64(dataToRead) > rr.limit) {
		rr.closeReader()
		rr.reader = nil
		rr.cancel = nil
	}
}

// readFromExistingRangeReader attempts to read data from an existing reader if one is available.
// If a reader exists and the read is successful, the data is returned.
// Otherwise, it returns an error indicating that a new reader is needed.
// LOCKS_REQUIRED (rr.mu)
func (rr *randomReader) readFromExistingRangeReader(ctx context.Context, p []byte, offset int64) (n int, err error) {
	rr.skipBytes(offset)
	rr.invalidateReaderIfMisalignedOrTooSmall(offset, offset+int64(len(p)))
	if rr.reader != nil {
		return rr.readFromRangeReader(ctx, p, offset, offset+int64(len(p)), rr.readType.Load())
	}
	return 0, FallbackToNewRangeReader
}

// readFromRangeReader reads using the NewReader interface of go-sdk. Its uses
// the existing reader if available, otherwise makes a call to GCS.
// LOCKS_REQUIRED (rr.mu)
func (rr *randomReader) readFromRangeReader(ctx context.Context, p []byte, offset int64, end int64, readType int64) (n int, err error) {
	// If we don't have a reader, start a read operation.
	if rr.reader == nil {
		err = rr.startRead(offset, end, readType)
		if err != nil {
			err = fmt.Errorf("startRead: %w", err)
			return
		}
	}

	// Now we have a reader positioned at the correct place. Consume as much from
	// it as possible.
	n, err = rr.readFull(ctx, p)
	rr.start += int64(n)
	rr.totalReadBytes.Add(uint64(n))

	// Sanity check.
	if rr.start > rr.limit {
		err = fmt.Errorf("Reader returned extra bytes: %d", rr.start-rr.limit)

		// Don't attempt to reuse the reader when it's behaving wackily.
		rr.closeReader()
		rr.reader = nil
		rr.cancel = nil
		rr.start = -1
		rr.limit = -1

		return
	}

	// Are we finished with this reader now?
	if rr.start == rr.limit {
		rr.closeReader()
		rr.reader = nil
		rr.cancel = nil
	}

	// Handle errors.
	switch {
	case err == io.EOF || err == io.ErrUnexpectedEOF:
		// For a non-empty buffer, ReadFull returns EOF or ErrUnexpectedEOF only
		// if the reader peters out early. That's fine, but it means we should
		// have hit the limit above.
		if rr.reader != nil {
			err = fmt.Errorf("random reader returned early by skipping %d bytes", rr.limit-rr.start)
			return
		}

		err = nil

	case err != nil:
		// Propagate other errors.
		err = fmt.Errorf("readFull: %w", err)
		return
	}

	rr.updateExpectedOffset(offset + int64(n))

	return
}

func (rr *randomReader) readFromMultiRangeReader(ctx context.Context, p []byte, offset, end int64, timeout time.Duration) (bytesRead int, err error) {
	if rr.mrdWrapper == nil {
		return 0, fmt.Errorf("readFromMultiRangeReader: Invalid MultiRangeDownloaderWrapper")
	}

	if rr.isMRDInUse.CompareAndSwap(false, true) {
		rr.mrdWrapper.IncrementRefCount()
	}

	bytesRead, err = rr.mrdWrapper.Read(ctx, p, offset, end, rr.metricHandle, false)
	rr.totalReadBytes.Add(uint64(bytesRead))
	rr.updateExpectedOffset(offset + int64(bytesRead))
	return
}

// closeReader fetches the readHandle before closing the reader instance.
// LOCKS_REQUIRED (rr.mu)
func (rr *randomReader) closeReader() {
	rr.readHandle = rr.reader.ReadHandle()

	// Drain in the background
	go func(r io.ReadCloser) {
		_, err := io.Copy(io.Discard, r)
		if err != nil {
			logger.Warnf("async drain error: %v", err)
		}
		err = r.Close()
		if err != nil {
			logger.Warnf("async close error: %v", err)
		}
	}(rr.reader)
}

func (rr *randomReader) updateExpectedOffset(offset int64) {
	rr.expectedOffset.Store(offset)
}
