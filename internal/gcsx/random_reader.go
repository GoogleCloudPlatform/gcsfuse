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
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/file"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/lru"
	cacheutil "github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/fs/gcsfuse_errors"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"github.com/jacobsa/fuse/fuseops"
	"golang.org/x/net/context"
)

// MB is 1 Megabyte. (Silly comment to make the lint warning go away)
const MB = 1 << 20

// Min read size in bytes for random reads.
// We will not send a request to GCS for less than this many bytes (unless the
// end of the object comes first).
const minReadSize = MB

// Max read size in bytes for random reads.
// If the average read size (between seeks) is below this number, reads will
// optimised for random access.
// We will skip forwards in a GCS response at most this many bytes.
// About 6 MB of data is buffered anyway, so 8 MB seems like a good round number.
const maxReadSize = 8 * MB

// Minimum number of seeks before evaluating if the read pattern is random.
const minSeeksForRandom = 2

// "readOp" is the value used in read context to store pointer to the read operation.
const ReadOp = "readOp"

// TODO(b/385826024): Revert timeout to an appropriate value
const TimeoutForMultiRangeRead = time.Hour

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
func NewRandomReader(o *gcs.MinObject, bucket gcs.Bucket, sequentialReadSizeMb int32, fileCacheHandler *file.CacheHandler, cacheFileForRangeRead bool, metricHandle common.MetricHandle, mrdWrapper *storage.MultiRangeDownloaderWrapper) RandomReader {
	return &randomReader{
		object:                o,
		bucket:                bucket,
		start:                 -1,
		limit:                 -1,
		seeks:                 0,
		totalReadBytes:        0,
		readType:              util.Sequential,
		sequentialReadSizeMb:  sequentialReadSizeMb,
		fileCacheHandler:      fileCacheHandler,
		cacheFileForRangeRead: cacheFileForRangeRead,
		mrdWrapper:            mrdWrapper,
		metricHandle:          metricHandle,
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
	seeks          uint64
	totalReadBytes uint64

	// ReadType of the reader. Will be sequential by default.
	readType string

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

	// mrdWrapper points to the wrapper object within inode.
	mrdWrapper *storage.MultiRangeDownloaderWrapper

	// boolean variable to determine if MRD is being used or not.
	isMRDInUse bool

	metricHandle common.MetricHandle
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
	readOp := ctx.Value(ReadOp).(*fuseops.ReadFileOp)
	logger.Tracef("%.13v <- FileCache(%s:/%s, offset: %d, size: %d handle: %d)", requestId, rr.bucket.Name(), rr.object.Name, offset, len(p), readOp.Handle)
	startTime := time.Now()

	// Response log
	defer func() {
		executionTime := time.Since(startTime)
		var requestOutput string
		if err != nil {
			requestOutput = fmt.Sprintf("err: %v (%v)", err, executionTime)
		} else {
			if rr.fileCacheHandle != nil {
				isSeq = rr.fileCacheHandle.IsSequential(offset)
			}
			requestOutput = fmt.Sprintf("OK (isSeq: %t, hit: %t) (%v)", isSeq, cacheHit, executionTime)
		}

		// Here rr.fileCacheHandle will not be nil since we return from the above in those cases.
		logger.Tracef("%.13v -> %s", requestId, requestOutput)

		readType := util.Random
		if isSeq {
			readType = util.Sequential
		}
		captureFileCacheMetrics(ctx, rr.metricHandle, readType, n, cacheHit, executionTime)
	}()

	// Create fileCacheHandle if not already.
	if rr.fileCacheHandle == nil {
		rr.fileCacheHandle, err = rr.fileCacheHandler.GetCacheHandle(rr.object, rr.bucket, rr.cacheFileForRangeRead, offset)
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

	n, cacheHit, err = rr.fileCacheHandle.Read(ctx, rr.bucket, rr.object, offset, p)
	if err == nil {
		return
	}

	cacheHit = false
	n = 0

	if cacheutil.IsCacheHandleInvalid(err) {
		logger.Tracef("Closing cacheHandle:%p for object: %s:/%s", rr.fileCacheHandle, rr.bucket.Name(), rr.object.Name)
		err = rr.fileCacheHandle.Close()
		if err != nil {
			logger.Warnf("tryReadingFromFileCache: while closing fileCacheHandle: %v", err)
		}
		rr.fileCacheHandle = nil
	} else if !strings.Contains(err.Error(), cacheutil.FallbackToGCSErrMsg) {
		err = fmt.Errorf("tryReadingFromFileCache: while reading via cache: %w", err)
		return
	}
	err = nil

	return
}

func captureFileCacheMetrics(ctx context.Context, metricHandle common.MetricHandle, readType string, readDataSize int, cacheHit bool, readLatency time.Duration) {
	metricHandle.FileCacheReadCount(ctx, 1, []common.MetricAttr{
		{Key: common.ReadType, Value: readType},
		{Key: common.CacheHit, Value: strconv.FormatBool(cacheHit)},
	})

	metricHandle.FileCacheReadBytesCount(ctx, int64(readDataSize), []common.MetricAttr{{Key: common.ReadType, Value: readType}})
	metricHandle.FileCacheReadLatency(ctx, float64(readLatency.Microseconds()), []common.MetricAttr{{Key: common.CacheHit, Value: strconv.FormatBool(cacheHit)}})
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

	// Check first if we can read using existing reader. if not, determine which
	// api to use and call gcs accordingly.

	// When the offset is AFTER the reader position, try to seek forward, within reason.
	// This happens when the kernel page cache serves some data. It's very common for
	// concurrent reads, often by only a few 128kB fuse read requests. The aim is to
	// re-use GCS connection and avoid throwing away already read data.
	// For parallel sequential reads to a single file, not throwing away the connections
	// is a 15-20x improvement in throughput: 150-200 MB/s instead of 10 MB/s.
	if rr.reader != nil && rr.start < offset && offset-rr.start < maxReadSize {
		bytesToSkip := offset - rr.start
		p := make([]byte, bytesToSkip)
		n, _ := io.ReadFull(rr.reader, p)
		rr.start += int64(n)
	}

	// If we have an existing reader, but it's positioned at the wrong place,
	// clean it up and throw it away.
	// We will also clean up the existing reader if it can't serve the entire request.
	dataToRead := math.Min(float64(offset+int64(len(p))), float64(rr.object.Size))
	if rr.reader != nil && (rr.start != offset || int64(dataToRead) > rr.limit) {
		rr.closeReader()
		rr.reader = nil
		rr.cancel = nil
		if rr.start != offset {
			// We should only increase the seek count if we have to discard the reader when it's
			// positioned at wrong place. Discarding it if can't serve the entire request would
			// result in reader size not growing for random reads scenario.
			rr.seeks++
		}
	}

	if rr.reader != nil {
		objectData.Size, err = rr.readFromRangeReader(ctx, p, offset, -1, rr.readType)
		return
	}

	// If we don't have a reader, determine whether to read from NewReader or MRR.
	end, err := rr.getReadInfo(offset, int64(len(p)))
	if err != nil {
		err = fmt.Errorf("ReadAt: getReaderInfo: %w", err)
		return
	}

	readerType := readerType(rr.readType, offset, end, rr.bucket.BucketType())
	if readerType == RangeReader {
		objectData.Size, err = rr.readFromRangeReader(ctx, p, offset, end, rr.readType)
		return
	}

	objectData.Size, err = rr.readFromMultiRangeReader(ctx, p, offset, end, TimeoutForMultiRangeRead)
	return
}

func (rr *randomReader) Object() (o *gcs.MinObject) {
	o = rr.object
	return
}

func (rr *randomReader) Destroy() {
	defer func() {
		if rr.isMRDInUse {
			err := rr.mrdWrapper.DecrementRefCount()
			if err != nil {
				logger.Errorf("randomReader::Destroy:%v", err)
			}
			rr.isMRDInUse = false
		}
	}()

	// Close out the reader, if we have one.
	if rr.reader != nil {
		rr.closeReader()
		rr.reader = nil
		rr.cancel = nil
	}

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

	// Call through.
	n, err = io.ReadFull(rr.reader, p)

	return
}

// Ensure that rr.reader is set up for a range for which [start, start+size) is
// a prefix. Irrespective of the size requested, we try to fetch more data
// from GCS defined by sequentialReadSizeMb flag to serve future read requests.
func (rr *randomReader) startRead(start int64, end int64) (err error) {
	// Begin the read.
	ctx, cancel := context.WithCancel(context.Background())

	rc, err := rr.bucket.NewReaderWithReadHandle(
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

	// If a file handle is open locally, but the corresponding object doesn't exist
	// in GCS, it indicates a file clobbering scenario. This likely occurred because:
	//  - The file was deleted in GCS while a local handle was still open.
	//  - The file content was modified leading to different generation number.
	var notFoundError *gcs.NotFoundError
	if errors.As(err, &notFoundError) {
		err = &gcsfuse_errors.FileClobberedError{
			Err: fmt.Errorf("NewReader: %w", err),
		}
		return
	}

	if err != nil {
		err = fmt.Errorf("NewReaderWithReadHandle: %w", err)
		return
	}

	rr.reader = rc
	rr.cancel = cancel
	rr.start = start
	rr.limit = end

	requestedDataSize := end - start
	common.CaptureGCSReadMetrics(ctx, rr.metricHandle, util.Sequential, requestedDataSize)

	return
}

// getReaderInfo determines the readType and provides the range to query GCS.
// Range here is [start, end]. End is computed using the readType, start offset
// and size of the data the callers needs.
func (rr *randomReader) getReadInfo(
	start int64,
	size int64) (end int64, err error) {
	// Make sure start and size are legal.
	if start < 0 || uint64(start) > rr.object.Size || size < 0 {
		err = fmt.Errorf(
			"range [%d, %d) is illegal for %d-byte object",
			start,
			start+size,
			rr.object.Size)
		return
	}

	if err != nil {
		return
	}

	// GCS requests are expensive. Prefer to issue read requests defined by
	// sequentialReadSizeMb flag. Sequential reads will simply sip from the fire house
	// with each call to ReadAt. In practice, GCS will fill the TCP buffers
	// with about 6 MB of data. Requests from outside GCP will be charged
	// about 6MB of egress data, even if less data is read. Inside GCP
	// regions, GCS egress is free. This logic should limit the number of
	// GCS read requests, which are not free.

	// But if we notice random read patterns after a minimum number of seeks,
	// optimise for random reads. Random reads will read data in chunks of
	// (average read size in bytes rounded up to the next MB).
	end = int64(rr.object.Size)
	if rr.seeks >= minSeeksForRandom {
		rr.readType = util.Random
		averageReadBytes := rr.totalReadBytes / rr.seeks
		if averageReadBytes < maxReadSize {
			randomReadSize := int64(((averageReadBytes / MB) + 1) * MB)
			if randomReadSize < minReadSize {
				randomReadSize = minReadSize
			}
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
	maxSizeToReadFromGCS := int64(rr.sequentialReadSizeMb * MB)
	if end-start > maxSizeToReadFromGCS {
		end = start + maxSizeToReadFromGCS
	}

	return
}

// readerType specifies the go-sdk interface to use for reads.
func readerType(readType string, start int64, end int64, bucketType gcs.BucketType) ReaderType {
	bytesToBeRead := end - start
	if readType == util.Random && bytesToBeRead < maxReadSize && bucketType.Zonal {
		return MultiRangeReader
	}
	return RangeReader
}

// readFromRangeReader reads using the NewReader interface of go-sdk. Its uses
// the existing reader if available, otherwise makes a call to GCS.
func (rr *randomReader) readFromRangeReader(ctx context.Context, p []byte, offset int64, end int64, readType string) (n int, err error) {
	// If we don't have a reader, start a read operation.
	if rr.reader == nil {
		err = rr.startRead(offset, end)
		if err != nil {
			err = fmt.Errorf("startRead: %w", err)
			return
		}
	}

	// Now we have a reader positioned at the correct place. Consume as much from
	// it as possible.
	n, err = rr.readFull(ctx, p)
	rr.start += int64(n)
	rr.totalReadBytes += uint64(n)

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
			err = fmt.Errorf("Reader returned early by skipping %d bytes", rr.limit-rr.start)
			return
		}

		err = nil

	case err != nil:
		// Propagate other errors.
		err = fmt.Errorf("readFull: %w", err)
		return
	}

	requestedDataSize := end - offset
	common.CaptureGCSReadMetrics(ctx, rr.metricHandle, readType, requestedDataSize)

	return
}

func (rr *randomReader) readFromMultiRangeReader(ctx context.Context, p []byte, offset, end int64, timeout time.Duration) (bytesRead int, err error) {
	if rr.mrdWrapper == nil {
		return 0, fmt.Errorf("readFromMultiRangeReader: Invalid MultiRangeDownloaderWrapper")
	}

	if !rr.isMRDInUse {
		rr.isMRDInUse = true
		rr.mrdWrapper.IncrementRefCount()
	}

	bytesRead, err = rr.mrdWrapper.Read(ctx, p, offset, end, timeout, rr.metricHandle)
	rr.totalReadBytes += uint64(bytesRead)
	return
}

// closeReader fetches the readHandle before closing the reader instance.
func (rr *randomReader) closeReader() {
	rr.readHandle = rr.reader.ReadHandle()
	err := rr.reader.Close()
	if err != nil {
		logger.Warnf("error while closing reader: %v", err)
	}
}
