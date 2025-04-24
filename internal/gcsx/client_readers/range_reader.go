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

	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/fs/gcsfuse_errors"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
)

type RangeReader struct {
	gcsx.Reader
	object *gcs.MinObject
	bucket gcs.Bucket
	start  int64
	limit  int64

	// If non-nil, an in-flight read request and a function for cancelling it.
	//
	// INVARIANT: (reader == nil) == (cancel == nil)
	reader gcs.StorageReader

	// Stores the handle associated with the previously closed newReader instance.
	// This will be used while making the new connection to bypass auth and metadata
	// checks.
	readHandle []byte
	cancel     func()

	readType     string
	metricHandle common.MetricHandle
}

func NewRangeReader(object *gcs.MinObject, bucket gcs.Bucket, metricHandle common.MetricHandle) *RangeReader {
	return &RangeReader{
		object:       object,
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

func (rr *RangeReader) ReadAt(ctx context.Context, req *gcsx.GCSReaderRequest) (gcsx.ReaderResponse, error) {
	readerResponse := gcsx.ReaderResponse{
		DataBuf: req.Buffer,
		Size:    0,
	}
	var err error

	readerResponse.Size, err = rr.readFromRangeReader(ctx, req.Buffer, req.Offset, req.EndOffset, rr.readType)

	return readerResponse, err
}

// readFromRangeReader reads using the NewReader interface of go-sdk. Its uses
// the existing reader if available, otherwise makes a call to GCS.
func (rr *RangeReader) readFromRangeReader(ctx context.Context, p []byte, offset int64, end int64, readType string) (int, error) {
	var err error
	// If we don't have a reader, start a read operation.
	if rr.reader == nil {
		err = rr.startRead(ctx, offset, end)
		if err != nil {
			err = fmt.Errorf("startRead: %w", err)
			return 0, err
		}
	}

	var n int
	// Now we have a reader positioned at the correct place. Consume as much from
	// it as possible.
	n, err = rr.readFull(ctx, p)
	rr.start += int64(n)

	// Sanity check.
	if rr.start > rr.limit {
		err = fmt.Errorf("reader returned extra bytes: %d", rr.start-rr.limit)

		// Don't attempt to reuse the reader when it's malfunctioning.
		rr.closeReader()
		rr.reader = nil
		rr.cancel = nil
		rr.start = -1
		rr.limit = -1

		return 0, err
	}

	// Are we finished with this reader now?
	if rr.start == rr.limit {
		rr.closeReader()
		rr.reader = nil
		rr.cancel = nil
	}

	// Handle errors.
	switch {
	case errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF):
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
func (rr *RangeReader) readFull(ctx context.Context, p []byte) (int, error) {
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

	return io.ReadFull(rr.reader, p)
}

// Ensure that rr.reader is set up for a range for which [start, start+size) is
// a prefix. Irrespective of the size requested, we try to fetch more data
// from GCS defined by sequentialReadSizeMb flag to serve future read requests.
func (rr *RangeReader) startRead(ctx context.Context, start int64, end int64) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

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
		return err
	}

	if err != nil {
		err = fmt.Errorf("NewReaderWithReadHandle: %w", err)
		return err
	}

	rr.reader = rc
	rr.cancel = cancel
	rr.start = start
	rr.limit = end

	requestedDataSize := end - start
	common.CaptureGCSReadMetrics(ctx, rr.metricHandle, util.Sequential, requestedDataSize)

	return nil
}
