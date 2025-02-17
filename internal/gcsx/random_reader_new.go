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
	"io"
	"strconv"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/common"
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

	// 2. Try reading from the GCS reader (if it exists and is positioned correctly).
	if rr.gr.reader != nil {
		objectData, err = rr.gr.ReadAt(ctx, p, offset) // Call gcsReader.ReadAt
		if err == nil {
			rr.gr.totalReadBytes += uint64(objectData.Size) // Update totalReadBytes in gcsReader if successful
		}
		if err == nil || err == io.EOF { // Treat EOF as success (partial read).
			if objectData.Size > 0 { // Only update seeks if data was actually read from GCS.
				rr.gr.seeks++
			}
			return objectData, err
		}
		if err != nil { // Cleanup the gcsReader if ReadAt fails (other than EOF).
			rr.gr.Destroy()
			rr.gr = rr.gr.NewGcsReader(rr.object, rr.bucket, rr.gr.sequentialReadSizeMb) // Initialize a new reader
		}
	} else {
		// Initialize a new reader and attempt reading.
		rr.gr = rr.gr.NewGcsReader(rr.object, rr.bucket, rr.gr.sequentialReadSizeMb) // Initialize a new reader
		objectData, err = rr.gr.ReadAt(ctx, p, offset)                               // Call gcsReader.ReadAt
		if err == nil {
			rr.gr.totalReadBytes += uint64(objectData.Size) // Update totalReadBytes in gcsReader if successful
		}
		if err == nil || err == io.EOF { // Treat EOF as success (partial read).
			if objectData.Size > 0 { // Only update seeks if data was actually read from GCS.
				rr.gr.seeks++
			}
			return objectData, err
		} else { // If it fails, clean up.
			rr.gr.Destroy()
			rr.gr = nil // Make sure it's cleaned up correctly.
		}
	}
	// 3. Fall back to Multi-Range Downloader if appropriate and not already using it.
	if rr.mrd != nil { // Check if mrd is available and not in use.
		objectData.Size, err = rr.mrd.Read(ctx, p, offset, offset+int64(len(p)), multiRangeDownloaderTimeout, rr.fc.metricHandle)
		if err == nil || err == io.EOF {
			rr.gr.totalReadBytes += uint64(objectData.Size)
			return objectData, err
		}
	}
	// Handle errors, potentially retrying or falling back to a different strategy.
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
