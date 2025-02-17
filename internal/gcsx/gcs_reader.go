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

	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
)

type gcsReader struct {
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

	// Stores the handle associated with the previously closed newReader instance.
	// This will be used while making the new connection to bypass auth and metadata
	// checks.
	readHandle   []byte
	metricHandle common.MetricHandle
}

func (gr *gcsReader) NewGcsReader(o *gcs.MinObject, bucket gcs.Bucket, sequentialReadSizeMb int32) *gcsReader {
	return &gcsReader{
		object:               o,
		bucket:               bucket,
		start:                -1,
		limit:                -1,
		seeks:                0,
		totalReadBytes:       0,
		readType:             util.Sequential,
		sequentialReadSizeMb: sequentialReadSizeMb,
	}
}

func (gr *gcsReader) CheckInvariants() {
	// INVARIANT: (reader == nil) == (cancel == nil)
	if (gr.reader == nil) != (gr.cancel == nil) {
		panic(fmt.Sprintf("Mismatch: %v vs. %v", gr.reader == nil, gr.cancel == nil))
	}

	// INVARIANT: start <= limit
	if !(gr.start <= gr.limit) {
		panic(fmt.Sprintf("Unexpected range: [%d, %d)", gr.start, gr.limit))
	}

	// INVARIANT: limit < 0 implies reader != nil
	if gr.limit < 0 && gr.reader != nil {
		panic(fmt.Sprintf("Unexpected non-nil reader with limit == %d", gr.limit))
	}
}

func (gr *gcsReader) Read(ctx context.Context, p []byte, offset int64) (int, error) {
	// When the offset is AFTER the reader position, try to seek forward, within reason.
	// This happens when the kernel page cache serves some data. It's very common for
	// concurrent reads, often by only a few 128kB fuse read requests. The aim is to
	// re-use GCS connection and avoid throwing away already read data.
	// For parallel sequential reads to a single file, not throwing away the connections
	// is a 15-20x improvement in throughput: 150-200 MB/s instead of 10 MB/s.
	if gr.reader != nil && gr.start < offset && offset-gr.start < maxReadSize {
		bytesToSkip := offset - gr.start
		p := make([]byte, bytesToSkip)
		n, _ := io.ReadFull(gr.reader, p)
		gr.start += int64(n)
	}

	// If we have an existing reader, but it's positioned at the wrong place,
	// clean it up and throw it away.
	// We will also clean up the existing reader if it can't serve the entire request.
	dataToRead := math.Min(float64(offset+int64(len(p))), float64(gr.object.Size))
	if gr.reader != nil && (gr.start != offset || int64(dataToRead) > gr.limit) {
		gr.closeReader()
		gr.reader = nil
		gr.cancel = nil
		if gr.start != offset {
			// We should only increase the seek count if we have to discard the reader when it's
			// positioned at wrong place. Discarding it if can't serve the entire request would
			// result in reader size not growing for random reads scenario.
			gr.seeks++
		}
	}
	if gr.reader != nil {
		size, err := gr.readFromRangeReader(ctx, p, offset, -1, gr.readType)
		return size, err
	}

	// If we don't have a reader, determine whether to read from NewReader or Mgr.
	end, err := gr.getReadInfo(offset, int64(len(p)))
	if err != nil {
		err = fmt.Errorf("ReadAt: getReaderInfo: %w", err)
		return 0, err
	}

	readerType := readerType(gr.readType, offset, end, gr.bucket.BucketType())
	if readerType == RangeReader {
		size, err := gr.readFromRangeReader(ctx, p, offset, end, gr.readType)
		return size, err
	}
   return 0, err
}

func (gr *gcsReader) Destroy() {
	// Close out the reader, if we have one.
	if gr.reader != nil {
		gr.closeReader()
		gr.reader = nil
		gr.cancel = nil
	}
}

// closeReader fetches the readHandle before closing the reader instance.
func (gr *gcsReader) closeReader() {
	gr.readHandle = gr.reader.ReadHandle()
	err := gr.reader.Close()
	if err != nil {
		logger.Warnf("error while closing reader: %v", err)
	}
}

// readFromRangeReader reads using the NewReader interface of go-sdk. Its uses
// the existing reader if available, otherwise makes a call to GCS.
func (gr *gcsReader) readFromRangeReader(ctx context.Context, p []byte, offset int64, end int64, readType string) (n int, err error) {
	// If we don't have a reader, start a read operation.
	if gr.reader == nil {
		err = gr.startRead(offset, end)
		if err != nil {
			err = fmt.Errorf("startRead: %w", err)
			return
		}
	}

	// Now we have a reader positioned at the correct place. Consume as much from
	// it as possible.
	n, err = gr.readFull(ctx, p)
	gr.start += int64(n)
	gr.totalReadBytes += uint64(n)

	// Sanity check.
	if gr.start > gr.limit {
		err = fmt.Errorf("Reader returned extra bytes: %d", gr.start-gr.limit)

		// Don't attempt to reuse the reader when it's behaving wackily.
		gr.closeReader()
		gr.reader = nil
		gr.cancel = nil
		gr.start = -1
		gr.limit = -1

		return
	}

	// Are we finished with this reader now?
	if gr.start == gr.limit {
		gr.closeReader()
		gr.reader = nil
		gr.cancel = nil
	}

	// Handle errors.
	switch {
	case err == io.EOF || err == io.ErrUnexpectedEOF:
		// For a non-empty buffer, ReadFull returns EOF or ErrUnexpectedEOF only
		// if the reader peters out early. That's fine, but it means we should
		// have hit the limit above.
		if gr.reader != nil {
			err = fmt.Errorf("Reader returned early by skipping %d bytes", gr.limit-gr.start)
			return
		}

		err = nil

	case err != nil:
		// Propagate other errors.
		err = fmt.Errorf("readFull: %w", err)
		return
	}

	requestedDataSize := end - offset
	common.CaptureGCSReadMetrics(ctx, gr.metricHandle, readType, requestedDataSize)

	return
}

// getReaderInfo determines the readType and provides the range to query GCS.
// Range here is [start, end]. End is computed using the readType, start offset
// and size of the data the callers needs.
func (gr *gcsReader) getReadInfo(
	start int64,
	size int64) (end int64, err error) {
	// Make sure start and size are legal.
	if start < 0 || uint64(start) > gr.object.Size || size < 0 {
		err = fmt.Errorf(
			"range [%d, %d) is illegal for %d-byte object",
			start,
			start+size,
			gr.object.Size)
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
	end = int64(gr.object.Size)
	if gr.seeks >= minSeeksForRandom {
		gr.readType = util.Random
		averageReadBytes := gr.totalReadBytes / gr.seeks
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
	if end > int64(gr.object.Size) {
		end = int64(gr.object.Size)
	}

	// To avoid overloading GCS and to have reasonable latencies, we will only
	// fetch data of max size defined by sequentialReadSizeMb.
	maxSizeToReadFromGCS := int64(gr.sequentialReadSizeMb * MB)
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
