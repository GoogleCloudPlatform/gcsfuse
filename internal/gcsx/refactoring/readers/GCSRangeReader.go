package readers

// This reader will be used for both single range reader and multi range reader
//i.e. random and sequential read for zonal as well as non zonal buckets

import (
	"bytes"
	"fmt"
	"io"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/fs/inode"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/monitor"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
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

// GCSRangeReader is an object that knows how to read ranges within a particular
// generation of a particular GCS object. Optimised for (large) sequential reads.
//
// Not safe for concurrent access.
//
// TODO - (raj-prince) - Rename this with appropriate name as it also started
// fulfilling the responsibility of reading object's content from cache.

// NewGCSRangeReader create a random reader for the supplied object record that
// reads using the given bucket.
func NewGCSRangeReader(o *gcs.MinObject, bucket gcs.Bucket, sequentialReadSizeMb int32, inode *inode.FileInode) *GCSRangeReader {
	return &GCSRangeReader{
		object:               o,
		bucket:               bucket,
		start:                -1,
		limit:                -1,
		seeks:                0,
		totalReadBytes:       0,
		sequentialReadSizeMb: sequentialReadSizeMb,
	}
}

type GCSRangeReader struct {
	object *gcs.MinObject
	bucket gcs.Bucket

	// If non-nil, an in-flight read request and a function for cancelling it.
	//
	// INVARIANT: (reader == nil) == (cancel == nil)
	reader io.ReadCloser
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

	sequentialReadSizeMb int32

	// MRD instance and read handle are maintained at inode to be used for ZB
	inode *inode.FileInode
	//this is a 8 MB buffer
	localCache []byte
	readHandle []byte
}

func (rr *GCSRangeReader) CheckInvariants() {
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

// dst needs to point this returned buffer
func (rr *GCSRangeReader) Read(
	ctx context.Context,
	p []byte,
	offset int64) (int, error, []byte) {

	var err error
	if offset >= int64(rr.object.Size) {
		err = io.EOF
		return 0, err, p
	}

	// This loop is not needed anymore as data requested is always less than data fetched from GCS
	for len(p) > 0 {
		// Have we blown past the end of the object?
		if offset >= int64(rr.object.Size) {
			err = io.EOF
			return 0, err, nil
		}

		// When the offset is AFTER the reader position, try to seek forward, within reason.
		// This happens when the kernel page cache serves some data. It's very common for
		// concurrent reads, often by only a few 128kB fuse read requests. The aim is to
		// re-use GCS connection and avoid throwing away already read data.
		// For parallel sequential reads to a single file, not throwing away the connections
		// is a 15-20x improvement in throughput: 150-200 MB/s instead of 10 MB/s.
		rr.seekReaderToPosition(offset)

		// If we don't have a reader, start a read operation.
		readType := util.Sequential
		var end int64 = -1
		if rr.reader == nil {
			readType, err, end = rr.getReadState(ctx, offset, int64(len(p)))
			if readType == util.Sequential || string(rr.bucket.BucketType()) != "Zonal" {
				rr.startRead(offset, end, readType)
			} else {
				//check if data can be served from local buffer, if no then call range in MRD
				bufWriter := bytes.NewBuffer(rr.localCache[0:])
				var tmp int
				tmp = rr.inode.MRD.Add(bufWriter, offset, end, func(start int64, end int64) {
					// Callback function
				})
				p = p[tmp:]
				mrrBuffer := rr.localCache[offset:rr.limit]
				rr.start += int64(tmp)
				offset += int64(tmp)
				rr.totalReadBytes += uint64(tmp)
				return tmp, nil, mrrBuffer
			}
			if err != nil {
				err = fmt.Errorf("startRead: %w", err)
				return 0, err, nil
			}
		}

		// Read from range reader in case bucket is non zonal and read from mrr if bucket is zonal
		var tmp int
		tmp, err = rr.readFull(ctx, p)
		p = p[tmp:]
		rr.start += int64(tmp)
		offset += int64(tmp)
		rr.totalReadBytes += uint64(tmp)

		// Sanity check.
		if rr.start > rr.limit {
			err = fmt.Errorf("reader returned %d too many bytes", rr.start-rr.limit)

			// Don't attempt to reuse the reader when it's behaving wackily.
			rr.reader.Close()
			rr.reader = nil
			rr.cancel = nil
			rr.start = -1
			rr.limit = -1

			return tmp, nil, p
		}

		// Are we finished with this reader now?
		if rr.start == rr.limit {
			rr.reader.Close()
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
				err = fmt.Errorf("reader returned %d too few bytes", rr.limit-rr.start)
				return tmp, err, p
			}

			err = nil

		case err != nil:
			// Propagate other errors.
			err = fmt.Errorf("readFull: %w", err)
			return tmp, err, p
		}
	}

	// return not needed as loop not needed
	return 0, nil, p
}

func (rr *GCSRangeReader) seekReaderToPosition(offset int64) {
	if rr.reader != nil && rr.start < offset && offset-rr.start < maxReadSize {
		bytesToSkip := int64(offset - rr.start)
		p := make([]byte, bytesToSkip)
		n, _ := io.ReadFull(rr.reader, p)
		rr.start += int64(n)
	}

	// If we have an existing reader but it's positioned at the wrong place,
	// clean it up and throw it away.
	// Add the check for limit here, check if current reader end <= limit provided, if not go inside if
	if rr.reader != nil && rr.start != offset {
		rr.reader.Close()
		rr.reader = nil
		rr.cancel = nil
		rr.seeks++
	}
	// if reader is not closed in above step then its implicit MRD is not needed and
	// new range reader is also not needed, data will served from existing reader
}

func (rr *GCSRangeReader) Object() (o *gcs.MinObject) {
	o = rr.object
	return
}

func (rr *GCSRangeReader) Destroy() {
	// Close out the reader, if we have one.
	if rr.reader != nil {
		err := rr.reader.Close()
		rr.reader = nil
		rr.cancel = nil
		if err != nil {
			logger.Warnf("rr.Destroy(): while closing reader: %v", err)
		}
	}
}

// Like io.ReadFull, but deals with the cancellation issues.
//
// REQUIRES: rr.reader != nil
func (rr *GCSRangeReader) readFull(
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
func (rr *GCSRangeReader) getReadState(
	ctx context.Context,
	start int64,
	size int64) (readType string, err error, end int64) {

	err = rr.sanityCheckForOffset(start, size)
	if err != nil {
		return
	}

	end = int64(rr.object.Size)
	readType = util.Sequential
	if rr.seeks >= minSeeksForRandom {
		readType = util.Random
		end = rr.endOffsetForRandomRead(end, start)
	}

	// If end - start > 8 MB and bucket is Zonal, change read type to Sequential

	end = rr.endOffsetWithinMaxLimit(end, start)

	//Check bucket type here if readtype is random and bucket type is zonal
	//create MRD and set in mrr of gcs range reader struct

	//if readType == util.Random && string(rr.bucket.BucketType()) == "Zonal" {
	//	//use inode MRD instance
	//	// use calculated start and end for this MRD add
	//	// use readType as random
	//	return
	//}

	return
}

// for non-zonal buckets only
func (rr *GCSRangeReader) startRead(start int64, end int64, readType string) {

	ctx, cancel := context.WithCancel(context.Background())
	//Pass ReadHandle to New Reader in case its not Nil
	rc, err := rr.bucket.NewReader(
		ctx,
		&gcs.ReadObjectRequest{
			Name:       rr.object.Name,
			Generation: rr.object.Generation,
			Range: &gcs.ByteRange{
				Start: uint64(start),
				Limit: uint64(end),
			},
			ReadCompressed: rr.object.HasContentEncodingGzip(),
		})

	if err != nil {
		err = fmt.Errorf("NewReader: %w", err)
		return
	}

	rr.reader = rc
	rr.cancel = cancel
	rr.start = start
	rr.limit = end

	requestedDataSize := end - start
	monitor.CaptureGCSReadMetrics(ctx, readType, requestedDataSize)

}

func (rr *GCSRangeReader) sanityCheckForOffset(start int64, size int64) error {
	if start < 0 || uint64(start) > rr.object.Size || size < 0 {
		err := fmt.Errorf(
			"range [%d, %d) is illegal for %d-byte object",
			start,
			start+size,
			rr.object.Size)
		return err
	}
	return nil
}

func (rr *GCSRangeReader) endOffsetForRandomRead(end int64, start int64) int64 {
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
	return end
}

func (rr *GCSRangeReader) endOffsetWithinMaxLimit(end int64, start int64) int64 {
	if end > int64(rr.object.Size) {
		end = int64(rr.object.Size)
	}

	// To avoid overloading GCS and to have reasonable latencies, we will only
	// fetch data of max size defined by sequentialReadSizeMb.
	maxSizeToReadFromGCS := int64(rr.sequentialReadSizeMb * MB)
	if end-start > maxSizeToReadFromGCS {
		end = start + maxSizeToReadFromGCS
	}
	return end
}
