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

package gcs_readers

import (
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/fs/gcsfuse_errors"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"golang.org/x/net/context"
)

type RangeReader struct {
	Obj    *gcs.MinObject
	Bucket gcs.Bucket

	Start int64
	Limit int64
	End   int64
	Seeks uint64

	// If non-nil, an in-flight read request and a function for cancelling it.
	//
	// INVARIANT: (reader == nil) == (cancel == nil)
	Reader         gcs.StorageReader
	TotalReadBytes uint64

	ReaderType string

	// Stores the handle associated with the previously closed newReader instance.
	// This will be used while making the new connection to bypass auth and metadata
	// checks.
	ReadHandle []byte
	cancel     func()

	MetricHandle common.MetricHandle
}

func (rr *RangeReader) CheckInvariants() {
}

func (rr *RangeReader) ReadAt(ctx context.Context, p []byte, offset int64) (ObjectData, error) {
	objectData := ObjectData{
		DataBuf:  p,
		CacheHit: false,
		Size:     0,
	}
	var err error
	objectData.Size, err = rr.readFromRangeReader(ctx, p, offset, rr.End, rr.ReaderType)
	return objectData, err
}

func (rr *RangeReader) Destroy() {
	// Close out the reader, if we have one.
	if rr.Reader != nil {
		rr.closeReader()
		rr.Reader = nil
		rr.cancel = nil
	}
}

// closeReader fetches the readHandle before closing the reader instance.
func (rr *RangeReader) closeReader() {
	rr.ReadHandle = rr.Reader.ReadHandle()
	err := rr.Reader.Close()
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
	if rr.Reader == nil {
		err = rr.startRead(offset, end)
		if err != nil {
			err = fmt.Errorf("startRead: %w", err)
			return 0, err
		}
	}

	// Now we have a reader positioned at the correct place. Consume as much from
	// it as possible.
	n, err = rr.readFull(ctx, p)
	rr.Start += int64(n)
	rr.TotalReadBytes += uint64(n)

	// Sanity check.
	if rr.Start > rr.Limit {
		err = fmt.Errorf("Reader returned extra bytes: %d", rr.Start-rr.Limit)

		// Don't attempt to reuse the reader when it's behaving wackily.
		rr.closeReader()
		rr.Reader = nil
		rr.cancel = nil
		rr.Start = -1
		rr.Limit = -1

		return n, err
	}

	// Are we finished with this reader now?
	if rr.Start == rr.Limit {
		rr.closeReader()
		rr.Reader = nil
		rr.cancel = nil
	}

	// Handle errors.
	switch {
	case err == io.EOF || err == io.ErrUnexpectedEOF:
		// For a non-empty buffer, ReadFull returns EOF or ErrUnexpectedEOF only
		// if the reader peters out early. That's fine, but it means we should
		// have hit the limit above.
		if rr.Reader != nil {
			err = fmt.Errorf("Reader returned early by skipping %d bytes", rr.Limit-rr.Start)
			return 0, err
		}

		err = nil

	case err != nil:
		// Propagate other errors.
		err = fmt.Errorf("readFull: %w", err)
		return 0, err
	}

	requestedDataSize := end - offset
	common.CaptureGCSReadMetrics(ctx, rr.MetricHandle, readType, requestedDataSize)

	return n, err
}

// Like io.ReadFull, but deals with the cancellation issues.
//
// REQUIRES: rr.Reader != nil
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
	n, err = io.ReadFull(rr.Reader, p)

	return
}

// Ensure that rr.Reader is set up for a range for which [start, start+size) is
// a prefix. Irrespective of the size requested, we try to fetch more data
// from GCS defined by sequentialReadSizeMb flag to serve future read requests.
func (rr *RangeReader) startRead(start int64, end int64) (err error) {
	// Begin the read.
	ctx, cancel := context.WithCancel(context.Background())

	log.Println("Start And End", start, end, len(rr.ReadHandle))

	rc, err := rr.Bucket.NewReaderWithReadHandle(
		ctx,
		&gcs.ReadObjectRequest{
			Name:       rr.Obj.Name,
			Generation: rr.Obj.Generation,
			Range: &gcs.ByteRange{
				Start: uint64(start),
				Limit: uint64(end),
			},
			ReadCompressed: rr.Obj.HasContentEncodingGzip(),
			ReadHandle:     rr.ReadHandle,
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

	rr.Reader = rc
	rr.cancel = cancel
	rr.Start = start
	rr.Limit = end

	requestedDataSize := end - start
	common.CaptureGCSReadMetrics(ctx, rr.MetricHandle, util.Sequential, requestedDataSize)

	return
}
