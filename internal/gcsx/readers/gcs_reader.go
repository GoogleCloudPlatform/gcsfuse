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
	"io"
	"log"
	"math"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx/readers/gcs_readers"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
)

// ReaderType represents different types of go-sdk gcs readers.
// For eg: NewReader and MRD both point to bidi read api. This enum specifies
// the go-sdk type.
type ReaderType int

// ReaderType enum values.
const (
	// MB is 1 Megabyte. (Silly comment to make the lint warning go away)
	MB = 1 << 20

	// Min read size in bytes for random reads.
	// We will not send a request to GCS for less than this many bytes (unless the
	// end of the object comes first).
	minReadSize = MB

	// Max read size in bytes for random reads.
	// If the average read size (between seeks) is below this number, reads will
	// optimised for random access.
	// We will skip forwards in a GCS response at most this many bytes.
	// About 6 MB of data is buffered anyway, so 8 MB seems like a good round number.
	maxReadSize = 8 * MB

	// Minimum number of seeks before evaluating if the read pattern is random.
	minSeeksForRandom = 2

	// TODO(b/385826024): Revert timeout to an appropriate value
	TimeoutForMultiRangeRead = time.Hour
)

// ReaderType enum values.
const (
	// RangeReader corresponds to NewReader method in bucket_handle.go
	RangeReader ReaderType = iota
	// MultiRangeReader corresponds to NewMultiRangeDownloader method in bucket_handle.go
	MultiRangeReader
)

type GCSReader struct {
	Obj    *gcs.MinObject
	Bucket gcs.Bucket

	RangeReader gcs_readers.RangeReader
	Mrr         gcs_readers.MultiRangeReader
	// Stores the handle associated with the previously closed newReader instance.
	// This will be used while making the new connection to bypass auth and metadata
	// checks.
	ReadHandle []byte
	ReaderType string

	Start int64
	Limit int64
	Seeks uint64

	// If non-nil, an in-flight read request and a function for cancelling it.
	//
	// INVARIANT: (reader == nil) == (cancel == nil)
	Reader         gcs.StorageReader
	TotalReadBytes uint64
	// If non-nil, an in-flight read request and a function for cancelling it.
	//
	// INVARIANT: (reader == nil) == (cancel == nil)
	reader gcs.StorageReader
	cancel func()

	SequentialReadSizeMb int32
}

func (gr *GCSReader) Object() *gcs.MinObject {
	return nil
}

func (gr *GCSReader) CheckInvariants() {
}

func (gr *GCSReader) ReadAt(ctx context.Context, p []byte, offset int64) (gcs_readers.ObjectData, error) {
	var objectData gcs_readers.ObjectData
	var err error

	// Check first if we can read using existing reader. if not, determine which
	// api to use and call gcs accordingly.

	// When the offset is AFTER the reader position, try to seek forward, within reason.
	// This happens when the kernel page cache serves some data. It's very common for
	// concurrent reads, often by only a few 128kB fuse read requests. The aim is to
	// re-use GCS connection and avoid throwing away already read data.
	// For parallel sequential reads to a single file, not throwing away the connections
	// is a 15-20x improvement in throughput: 150-200 MB/s instead of 10 MB/s.
	if gr.Reader != nil && gr.Start < offset && offset-gr.Start < maxReadSize {
		bytesToSkip := offset - gr.Start
		p := make([]byte, bytesToSkip)
		n, _ := io.ReadFull(gr.reader, p)
		gr.Start += int64(n)
	}

	// If we have an existing reader, but it's positioned at the wrong place,
	// clean it up and throw it away.
	// We will also clean up the existing reader if it can't serve the entire request.
	dataToRead := math.Min(float64(offset+int64(len(p))), float64(gr.Obj.Size))
	if gr.reader != nil && (gr.Start != offset || int64(dataToRead) > gr.Limit) {
		gr.closeReader()
		gr.reader = nil
		gr.cancel = nil
		if gr.Start != offset {
			// We should only increase the seek count if we have to discard the reader when it's
			// positioned at wrong place. Discarding it if can't serve the entire request would
			// result in reader size not growing for random reads scenario.
			gr.Seeks++
		}
	}

	gr.RangeReader.ReadHandle = gr.ReadHandle

	if gr.reader != nil {
		objectData, err = gr.RangeReader.ReadAt(ctx, p, offset)
		return objectData, err
	}

	// If we don't have a reader, determine whether to read from NewReader or Mgr.
	end, err := gr.getReadInfo(offset, int64(len(p)))
	if err != nil {
		err = fmt.Errorf("ReadAt: getReaderInfo: %w", err)
		return objectData, err
	}
	gr.RangeReader.End = end
	gr.Mrr.End = end

	readerType := gr.readerType(gr.ReaderType, offset, end, gr.Bucket.BucketType())
	if readerType == RangeReader {
		objectData, err = gr.RangeReader.ReadAt(ctx, p, offset)
		return objectData, err
	}

	objectData, err = gr.Mrr.ReadAt(ctx, p, offset)
	if err != nil {
		return objectData, err
	}

	return objectData, nil
}

// closeReader fetches the readHandle before closing the reader instance.
func (gr *GCSReader) closeReader() {
	gr.ReadHandle = gr.reader.ReadHandle()
	err := gr.reader.Close()
	if err != nil {
		logger.Warnf("error while closing reader: %v", err)
	}
}

// readerType specifies the go-sdk interface to use for reads.
func (gr *GCSReader) readerType(readType string, start int64, end int64, bucketType gcs.BucketType) ReaderType {
	bytesToBeRead := end - start
	if readType == util.Random && bytesToBeRead < maxReadSize && bucketType.Zonal {
		return MultiRangeReader
	}
	return RangeReader
}

// getReaderInfo determines the readType and provides the range to query GCS.
// Range here is [start, end]. End is computed using the readType, start offset
// and size of the data the callers needs.
func (gr *GCSReader) getReadInfo(start int64, size int64) (int64, error) {
	// Make sure start and size are legal.
	if start < 0 || uint64(start) > gr.Obj.Size || size < 0 {
		err := fmt.Errorf(
			"range [%d, %d) is illegal for %d-byte object",
			start,
			start+size,
			gr.Obj.Size)
		return 0, err
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
	end := int64(gr.Obj.Size)
	if gr.Seeks >= minSeeksForRandom {
		gr.ReaderType = util.Random
		averageReadBytes := gr.TotalReadBytes / gr.Seeks
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
	if end > int64(gr.Obj.Size) {
		end = int64(gr.Obj.Size)
	}

	// To avoid overloading GCS and to have reasonable latencies, we will only
	// fetch data of max size defined by sequentialReadSizeMb.
	maxSizeToReadFromGCS := int64(gr.SequentialReadSizeMb * MB)
	log.Println("End-Start", end-start)
	log.Println("maxSizeToReadFromGCS", maxSizeToReadFromGCS)
	if end-start > maxSizeToReadFromGCS {
		end = start + maxSizeToReadFromGCS
	}

	return end, nil
}
