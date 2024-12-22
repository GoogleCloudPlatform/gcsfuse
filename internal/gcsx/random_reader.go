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

	// ReadAt Matches the semantics of io.ReaderAt, with the addition of context
	// support and cache support. It returns a boolean which represent either
	// content is read from fileCache (cacheHit = true) or gcs (cacheHit = false)
	ReadAt(ctx context.Context, p []byte, offset int64) (n int, cacheHit bool, err error)

	// Return the record for the object to which the reader is bound.
	Object() (o *gcs.MinObject)

	// Clean up any resources associated with the reader, which must not be used
	// again.
	Destroy()
}

// NewRandomReader create a random reader for the supplied object record that
// reads using the given bucket.
func NewRandomReader(o *gcs.MinObject, bucket gcs.Bucket, sequentialReadSizeMb int32, fileCacheHandler *file.CacheHandler, cacheFileForRangeRead bool, metricHandle common.MetricHandle) RandomReader {
	return &randomReader{
		object:               o,
		bucket:               bucket,
		totalReadBytes:       0,
		sequentialReadSizeMb: sequentialReadSizeMb,
		// TODO(wlhe): make them flags
		perPIDReader:            true,
		minSequentialReadSizeMb: 200,
		fileCacheHandler:        fileCacheHandler,
		cacheFileForRangeRead:   cacheFileForRangeRead,
		metricHandle:            metricHandle,
	}
}

type randomReader struct {
	object *gcs.MinObject
	bucket gcs.Bucket

	// If perPIDReader is not set, then there is only one objectRangeReader.
	// Otherwise, there is could be multiple objectRangeReaders, one per PID.
	objReader      *objectRangeReader
	objReaders     []*objectRangeReader
	totalReadBytes uint64

	// perPIDReader is a flag to enable per PID GCSreader.
	perPIDReader         bool
	sequentialReadSizeMb int32

	// minSequentialReadSizeMb is the minimum size of sequential read size when creating a GCS reader.
	minSequentialReadSizeMb int32

	// fileCacheHandler is used to get file cache handle and read happens using that.
	// This will be nil if the file cache is disabled.
	fileCacheHandler *file.CacheHandler

	// cacheFileForRangeRead is also valid for cache workflow, if true, object content
	// will be downloaded for random reads as well too.
	cacheFileForRangeRead bool

	// fileCacheHandle is used to read from the cached location. It is created on the fly
	// using fileCacheHandler for the given object and bucket.
	fileCacheHandle *file.CacheHandle
	metricHandle    common.MetricHandle
}

func (rr *randomReader) CheckInvariants() {
	if rr.objReader != nil {
		rr.objReader.CheckInvariants()
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

func (rr *randomReader) getObjectRangeReader(ctx context.Context) *objectRangeReader {
	if rr.perPIDReader {
		readOp := ctx.Value(ReadOp).(*fuseops.ReadFileOp)
		pid := int64(readOp.OpContext.Pid)
		for _, objReader := range rr.objReaders {
			if objReader.pid == pid {
				return objReader
			}
		}
		or := newObjectRangeReader(rr.object, rr.bucket, rr.sequentialReadSizeMb, rr.minSequentialReadSizeMb, rr.metricHandle)
		or.pid = pid
		// TODO(wlhe): consider limit the max number of objReaders.
		rr.objReaders = append(rr.objReaders, or)
		logger.Tracef("Created new %s", or.name())
		return or
	}

	if rr.objReader == nil {
		rr.objReader = newObjectRangeReader(rr.object, rr.bucket, rr.sequentialReadSizeMb, rr.minSequentialReadSizeMb, rr.metricHandle)
		logger.Tracef("Created new %s", rr.objReader.name())
	}
	return rr.objReader
}

func (rr *randomReader) ReadAt(
	ctx context.Context,
	p []byte,
	offset int64) (n int, cacheHit bool, err error) {

	if offset >= int64(rr.object.Size) {
		err = io.EOF
		return
	}

	// Note: If we are reading the file for the first time and read type is sequential
	// then the file cache behavior is write-through i.e. data is first read from
	// GCS, cached in file and then served from that file. But the cacheHit is
	// false in that case.
	n, cacheHit, err = rr.tryReadingFromFileCache(ctx, p, offset)
	if err != nil {
		err = fmt.Errorf("ReadAt: while reading from cache: %w", err)
		return
	}
	// Data was served from cache.
	if cacheHit || n == len(p) || (n < len(p) && uint64(offset)+uint64(n) == rr.object.Size) {
		return
	}
	for len(p) > 0 {
		// Have we blown past the end of the object?
		if offset >= int64(rr.object.Size) {
			err = io.EOF
			return
		}

		// Get the proper objectRangeReader to read data from.
		reader := rr.getObjectRangeReader(ctx)

		var tmp int
		tmp, err = reader.readAt(ctx, p, offset)

		n += tmp
		p = p[tmp:]
		offset += int64(tmp)
		rr.totalReadBytes += uint64(tmp)
	}

	return
}

func (rr *randomReader) Object() (o *gcs.MinObject) {
	o = rr.object
	return
}

func (rr *randomReader) Destroy() {
	// Close out all the objectRangeReaders.
	for _, or := range rr.objReaders {
		or.destroy()
	}
	if rr.objReader != nil {
		rr.objReader.destroy()
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

type objectRangeReader struct {
	object *gcs.MinObject
	bucket gcs.Bucket

	// If non-nil, an in-flight read request and a function for cancelling it.
	//
	// INVARIANT: (reader == nil) == (cancel == nil)
	reader io.ReadCloser
	cancel func()

	// Optional. Only used when per PID reader is enabled.
	pid int64

	// The range of the object that we expect reader to yield, when reader is
	// non-nil. When reader is nil, limit is the limit of the previous read
	// operation, or -1 if there has never been one.
	//
	// INVARIANT: start <= limit
	// INVARIANT: limit < 0 implies reader != nil
	// All these properties will be used only in case of GCS reads and not for
	// reads from cache.
	start int64
	limit int64
	seeks uint64
	// Number of bytes read from the current GCS reader.
	readBytes uint64

	// Total number of bytes read from GCS of all the GCS readers ever created for this object range reader.
	totalReadBytes uint64

	sequentialReadSizeMb int32

	// minSequentialReadSizeMb is the minimum size of sequential read size when creating a GCS reader.
	minSequentialReadSizeMb int32

	metricHandle common.MetricHandle
}

func (or *objectRangeReader) name() string {
	if or.pid != -1 {
		return fmt.Sprintf("objectRangeReader (pid=%d)", or.pid)
	}
	return fmt.Sprintf("objectRangeReader")
}

// newObjectRangeReader create a object range reader for the supplied object record that
// reads using the given bucket.
func newObjectRangeReader(o *gcs.MinObject, bucket gcs.Bucket, sequentialReadSizeMb int32, minSequentialReadSizeMb int32, metricHandle common.MetricHandle) *objectRangeReader {
	return &objectRangeReader{
		object:                  o,
		bucket:                  bucket,
		pid:                     -1,
		start:                   -1,
		limit:                   -1,
		readBytes:               0,
		seeks:                   0,
		totalReadBytes:          0,
		sequentialReadSizeMb:    sequentialReadSizeMb,
		minSequentialReadSizeMb: minSequentialReadSizeMb,
		metricHandle:            metricHandle,
	}
}

func (or *objectRangeReader) CheckInvariants() {
	// INVARIANT: (reader == nil) == (cancel == nil)
	if (or.reader == nil) != (or.cancel == nil) {
		panic(fmt.Sprintf("Mismatch for %s: %v vs. %v", or.name(), or.reader == nil, or.cancel == nil))
	}

	// INVARIANT: start <= limit
	if !(or.start <= or.limit) {
		panic(fmt.Sprintf("Unexpected range for : [%d, %d)", or.name(), or.start, or.limit))
	}

	// INVARIANT: limit < 0 implies reader != nil
	if or.limit < 0 && or.reader != nil {
		panic(fmt.Sprintf("Unexpected non-nil reader with limit == %d", or.limit))
	}
}

func (or *objectRangeReader) readBytesMB() float32 {
	return float32(or.readBytes) / MB
}

func (or *objectRangeReader) totalReadBytesMB() float32 {
	return float32(or.totalReadBytes) / MB
}

func (or *objectRangeReader) gcsReaderStats() string {
	return fmt.Sprintf("readBytesMB: %.2f, totalReadBytesMB: %.2f, seeks: %d", or.readBytesMB(), or.totalReadBytesMB(), or.seeks)
}

func (or *objectRangeReader) destroy() {
	if or.reader != nil {
		logger.Tracef("%s closed the GCS reader at final destruction, %s", or.name(), or.gcsReaderStats())
		logger.Tracef("Destroying %s, totalReadBytesMB: %.2f, seeks: %d", or.name(), or.totalReadBytesMB(), or.seeks)
		err := or.reader.Close()
		or.reader = nil
		if err != nil {
			logger.Warnf("%d dstroy error: %v", or.name(), err)
		}
		or.cancel = nil
	}
}

// Like io.ReadFull, but deals with the cancellation issues.
//
// REQUIRES: rr.reader != nil
func (or *objectRangeReader) readFull(
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
				or.cancel()
			}
		}
	}()

	// Call through.
	n, err = io.ReadFull(or.reader, p)

	return
}

func (or *objectRangeReader) readAt(
	ctx context.Context,
	p []byte,
	offset int64) (n int, err error) {

	// When the offset is AFTER the reader position, try to seek forward, within reason.
	// This happens when the kernel page cache serves some data. It's very common for
	// concurrent reads, often by only a few 128kB fuse read requests. The aim is to
	// re-use GCS connection and avoid throwing away already read data.
	// For parallel sequential reads to a single file, not throwing away the connections
	// is a 15-20x improvement in throughput: 150-200 MB/s instead of 10 MB/s.
	//
	// It doesn't reuse the connection when seeking forward for more than maxReadSize before
	// seeking forward is actually done reading by discarding all the bytes between start and offset.
	// If the gap is too large, it's likely that it's slower than just creating a new connection.
	// at the new offset.

	logger.Tracef("%s readAt, start: %d, limit: %d, offset: %d", or.name(), or.start, or.limit, offset)

	if or.reader != nil && or.start < offset && offset-or.start < maxReadSize {
		bytesToSkip := int64(offset - or.start)
		skipBuffer := make([]byte, bytesToSkip)
		bytesRead, _ := io.ReadFull(or.reader, skipBuffer)
		or.start += int64(bytesRead)
		logger.Tracef("%s seek forward bytesMB: %.2f, start: %d", or.name(), float32(bytesRead)/MB, or.start)
	}

	// If we have an existing reader but it's positioned at the wrong place,
	// clean it up and throw it away.
	if or.reader != nil && or.start != offset {
		or.reader.Close()
		or.reader = nil
		or.cancel = nil
		or.seeks++
		if or.start > offset {
			logger.Tracef("%s closed the GCS reader due to seeking back, %s", or.name(), or.gcsReaderStats())
		} else if offset-or.start >= maxReadSize {
			logger.Tracef("%s closed the GCS reader due to seeking forward for more than %d MB, %s", or.name(), maxReadSize/MB, or.gcsReaderStats())
		} else if offset >= or.limit {
			logger.Tracef("%s closed the GCS reader due to seeking beyond the range, %s", or.name(), or.gcsReaderStats())
		} else {
			logger.Tracef("%s closed the GCS reader due to wrong position, %s", or.name(), or.gcsReaderStats())
		}
	}

	// If we don't have a reader, start a new GCS reader for the given range.
	if or.reader == nil {
		err = or.newGCSReader(ctx, offset, int64(len(p)))
		if err != nil {
			err = fmt.Errorf("%s newGCSReader error: %v", or.name(), err)
			return
		}
	}

	// Now we have a reader positioned at the correct place. Consume as much from
	// it as possible.
	n, err = or.readFull(ctx, p)

	or.start += int64(n)
	or.readBytes += uint64(n)
	or.totalReadBytes += uint64(n)

	// Sanity check.
	if or.start > or.limit {
		err = fmt.Errorf("%s reader returned %d too many bytes", or.name(), or.start-or.limit)
		logger.Tracef("%s closed the GCS reader due to too many bytes returned, %s", or.name(), or.gcsReaderStats())

		// Don't attempt to reuse the reader when it's behaving wackily.
		or.reader.Close()
		or.reader = nil
		or.cancel = nil
		or.start = -1
		or.limit = -1
		or.readBytes = 0

		return
	}

	// Are we finished with this reader now?
	if or.start == or.limit {
		logger.Tracef("%s closed the GCS reader due to byte limit reached: %d, %s", or.name(), or.limit, or.gcsReaderStats())

		or.reader.Close()
		or.reader = nil
		or.cancel = nil
	}

	// Handle errors.
	switch {
	case err == io.EOF || err == io.ErrUnexpectedEOF:
		// For a non-empty buffer, ReadFull returns EOF or ErrUnexpectedEOF only
		// if the reader peters out early. That's fine, but it means we should
		// have hit the limit above.
		if or.reader != nil {
			err = fmt.Errorf("%s reader returned %d too few bytes", or.name(), or.limit-or.start)
			return
		}

		err = nil

	case err != nil:
		// Propagate other errors.
		err = fmt.Errorf("%sreadFull: %v", or.name(), err)
		return
	}
	return
}

// Create a new GCS reader for the given range for the objectRangeReader.
//
// Ensure that or.reader is set up for a range for which [start, start+size) is
// a prefix. Irrespective of the size requested, we try to fetch more data
// from GCS defined by sequentialReadSizeMb flag to serve future read requests.
func (or *objectRangeReader) newGCSReader(
	ctx context.Context,
	start int64,
	size int64) (err error) {
	// Make sure start and size are legal.
	if start < 0 || uint64(start) > or.object.Size || size < 0 {
		err = fmt.Errorf(
			"range [%d, %d) is illegal for %d-byte object",
			start,
			start+size,
			or.object.Size)
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
	end := int64(or.object.Size)
	readType := util.Sequential
	logger.Tracef("%s calculating the read size for the new GCS reader, totalReadBytesMB: %.2f, seeks: %d", or.name(), or.totalReadBytesMB(), or.seeks)

	if or.seeks >= minSeeksForRandom {
		readType = util.Random
		averageReadBytes := or.totalReadBytes / or.seeks
		logger.Tracef("%s calculating the read size for the new GCS reader, averageReadBytesMB: %d", or.name(), averageReadBytes/MB)

		if averageReadBytes < maxReadSize {
			randomReadSize := int64(((averageReadBytes / MB) + 1) * MB)
			logger.Tracef("%s calculating the read size for the new GCS reader, randomReadSizeMB: %d, minReadSizeMB: %d, maxReadSizeMB: %d", or.name(), randomReadSize/MB, minReadSize/MB, maxReadSize/MB)

			if randomReadSize < minReadSize {
				randomReadSize = minReadSize
			}
			if randomReadSize > maxReadSize {
				randomReadSize = maxReadSize
			}

			if randomReadSize < int64(or.minSequentialReadSizeMb*MB) {
				logger.Tracef("%s overriding the read size for the new GCS reader, original sizeMB: %d, new sizeMB: %d", or.name(), randomReadSize/MB, or.minSequentialReadSizeMb)
				randomReadSize = int64(or.minSequentialReadSizeMb * MB)
			}
			end = start + randomReadSize
		}
	}
	if end > int64(or.object.Size) {
		end = int64(or.object.Size)
	}

	// To avoid overloading GCS and to have reasonable latencies, we will only
	// fetch data of max size defined by sequentialReadSizeMb.
	maxSizeToReadFromGCS := int64(or.sequentialReadSizeMb * MB)
	if end-start > maxSizeToReadFromGCS {
		end = start + maxSizeToReadFromGCS
	}
	logger.Tracef("%s creating a new GCS reader, offset: %d, sizeMB: %d", or.name(), start, (end-start)/MB)

	// Begin the read.
	// Use a Background context to keep the GCS stream open.
	ctx, cancel := context.WithCancel(context.Background())
	rc, err := or.bucket.NewReader(
		ctx,
		&gcs.ReadObjectRequest{
			Name:       or.object.Name,
			Generation: or.object.Generation,
			Range: &gcs.ByteRange{
				Start: uint64(start),
				Limit: uint64(end),
			},
			ReadCompressed: or.object.HasContentEncodingGzip(),
		})

	// If a file handle is open locally, but the corresponding object doesn't exist
	// in GCS, it indicates a file clobbering scenario. This likely occurred because:
	//  - The file was deleted in GCS while a local handle was still open.
	//  - The file content was modified leading to different generation number.
	var notFoundError *gcs.NotFoundError
	if errors.As(err, &notFoundError) {
		err = &gcsfuse_errors.FileClobberedError{
			Err: fmt.Errorf("%s newGCSReader: %w", or.name(), err),
		}
		return
	}

	if err != nil {
		err = fmt.Errorf("%s newGCSReader: %w", or.name(), err)
		return
	}

	or.reader = rc
	or.cancel = cancel
	or.start = start
	or.limit = end
	or.readBytes = 0

	requestedDataSize := end - start
	common.CaptureGCSReadMetrics(ctx, or.metricHandle, readType, requestedDataSize)

	return
}
