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

package client_readers

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math"

	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/fs/gcsfuse_errors"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx/readers"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"golang.org/x/net/context"
)

type RangeReader struct {
	obj    *gcs.MinObject
	bucket gcs.Bucket

	start int64
	limit int64

	// If non-nil, an in-flight read request and a function for cancelling it.
	//
	// INVARIANT: (reader == nil) == (cancel == nil)
	reader gcs.StorageReader

	readType string

	// Stores the handle associated with the previously closed newReader instance.
	// This will be used while making the new connection to bypass auth and metadata
	// checks.
	readHandle []byte
	cancel     func()

	metricHandle common.MetricHandle
}

func NewRangeReader(obj *gcs.MinObject, bucket gcs.Bucket, metricHandle common.MetricHandle) RangeReader {
	return RangeReader{
		obj:          obj,
		bucket:       bucket,
		metricHandle: metricHandle,
		start:        -1,
		limit:        -1,
	}
}
func (rr *RangeReader) CheckInvariants() {
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

func (rr *RangeReader) ReadAt(ctx context.Context, p []byte, offset, end int64) (readers.ObjectData, error) {
	objectData := readers.ObjectData{
		DataBuf: p,
		Size:    0,
	}
	var err error
	objectData.Size, err = rr.readFromRangeReader(ctx, p, offset, end, rr.readType)
	return objectData, err
}

func (rr *RangeReader) Destroy() {
	// Close out the reader, if we have one.
	if rr.reader != nil {
		rr.closeReader()
		rr.reader = nil
		rr.cancel = nil
	}
}

// closeReader fetches the readHandle before closing the reader instance.
func (rr *RangeReader) closeReader() {
	rr.readHandle = rr.reader.ReadHandle()
	err := rr.reader.Close()
	if err != nil {
		logger.Warnf("error while closing reader: %v", err)
	}
}

// readFromRangeReader reads using the NewReader interface of go-sdk. Its uses
// the existing reader if available, otherwise makes a call to GCS.
func (rr *RangeReader) readFromRangeReader(ctx context.Context, p []byte, offset int64, end int64, readType string) (int, error) {
	var n int
	var err error
	// If we don't have a reader, start a read operation.
	if rr.reader == nil {
		err = rr.startRead(offset, end)
		if err != nil {
			err = fmt.Errorf("startRead: %w", err)
			return 0, err
		}
	}

	// Now we have a reader positioned at the correct place. Consume as much from
	// it as possible.
	n, err = rr.readFull(ctx, p)
	rr.start += int64(n)

	// Sanity check.
	if rr.start > rr.limit {
		err = fmt.Errorf("reader returned extra bytes: %d", rr.start-rr.limit)

		// Don't attempt to reuse the reader when it's behaving wackily.
		rr.closeReader()
		rr.reader = nil
		rr.cancel = nil
		rr.start = -1
		rr.limit = -1

		return n, err
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
			err = fmt.Errorf("reader returned early by skipping %d bytes", rr.limit-rr.start)
			return 0, err
		}

		err = nil

	case err != nil:
		// Propagate other errors.
		err = fmt.Errorf("readFull: %w", err)
		return 0, err
	}

	requestedDataSize := end - offset
	common.CaptureGCSReadMetrics(ctx, rr.metricHandle, readType, requestedDataSize)

	return n, err
}

// Like io.ReadFull, but deals with the cancellation issues.
//
// REQUIRES: rr.reader != nil
func (rr *RangeReader) readFull(ctx context.Context, p []byte) (n int, err error) {
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
func (rr *RangeReader) startRead(start int64, end int64) (err error) {
	// Begin the read.
	ctx, cancel := context.WithCancel(context.Background())

	log.Println("start And end", start, end, len(rr.readHandle))

	rc, err := rr.bucket.NewReaderWithReadHandle(
		ctx,
		&gcs.ReadObjectRequest{
			Name:       rr.obj.Name,
			Generation: rr.obj.Generation,
			Range: &gcs.ByteRange{
				Start: uint64(start),
				Limit: uint64(end),
			},
			ReadCompressed: rr.obj.HasContentEncodingGzip(),
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

func (rr *RangeReader) skipBytes(offset int64) {
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
}

func (rr *RangeReader) discardReader(offset int64, p []byte) bool {
	// If we have an existing reader, but it's positioned at the wrong place,
	// clean it up and throw it away.
	// We will also clean up the existing reader if it can't serve the entire request.
	dataToRead := math.Min(float64(offset+int64(len(p))), float64(rr.obj.Size))
	if rr.reader != nil && (rr.start != offset || int64(dataToRead) > rr.limit) {
		rr.closeReader()
		rr.reader = nil
		if rr.start != offset {
			// We should only increase the seek count if we have to discard the reader when it's
			// positioned at wrong place. Discarding it if can't serve the entire request would
			// result in reader size not growing for random reads scenario.
			return true
		}
	}
	return false
}

func (rr *RangeReader) readFromExistingReader(ctx context.Context, p []byte, offset, end int64) (readers.ObjectData, error) {
	objectData := readers.ObjectData{
		DataBuf:                 p,
		Size:                    0,
		FallBackToAnotherReader: true,
	}
	var err error

	if rr.reader != nil {
		objectData, err = rr.ReadAt(ctx, p, offset, end)
		objectData.FallBackToAnotherReader = false
	}

	return objectData, err
}
