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
	"io"
	"math"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
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

type randomReader struct {
	object *gcs.MinObject
	bucket gcs.Bucket
	gr     *gcsReader
	fc     *FileCacheReader
	mrd    *MultiRangeDownloaderWrapper
}

func (rr *randomReader) ReadAt(ctx context.Context, p []byte, offset int64) (objectData ObjectData, err error) {
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
	cacheReader := NewFileCacheReader(rr.fc.handler, rr.fc.cacheFileForRangeRead)
	n, cacheHit, err := cacheReader.ReadFromCache(ctx, rr.fc.bucket, rr.fc.object, p, offset)
	// Data was served from cache.
	if cacheHit || n == len(p) || (n < len(p) && uint64(offset)+uint64(n) == rr.fc.object.Size) {
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
