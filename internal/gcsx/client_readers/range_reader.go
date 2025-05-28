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
	"math"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/common"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/gcsfuse_errors"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
)

const (
	MiB = 1 << 20

	// Max read size in bytes for random reads.
	// If the average read size (between seeks) is below this number, reads will optimise for random access.
	// We will skip forwards in a GCS response at most this many bytes.
	maxReadSize = 8 * MiB
)

type RangeReader struct {
	gcsx.GCSReader
	object *gcs.MinObject
	bucket gcs.Bucket

	// start is the current read offset of the reader.
	start int64

	// limit is the exclusive upper bound up to which the reader can read.
	limit int64

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
	readConfig   *cfg.ReadConfig
	metricHandle common.MetricHandle
}

func NewRangeReader(object *gcs.MinObject, bucket gcs.Bucket, readConfig *cfg.ReadConfig, metricHandle common.MetricHandle) *RangeReader {
	return &RangeReader{
		object:       object,
		bucket:       bucket,
		metricHandle: metricHandle,
		readConfig:   readConfig,
		start:        -1,
		limit:        -1,
	}
}

func (rr *RangeReader) checkInvariants() {
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

func (rr *RangeReader) destroy() {
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

	if req.Offset >= int64(rr.object.Size) {
		err = io.EOF
		return readerResponse, err
	}

	readerResponse.Size, err = rr.readFromRangeReader(ctx, req.Buffer, req.Offset, req.EndOffset, rr.readType)

	return readerResponse, err
}

// readFromRangeReader reads using the NewReader interface of go-sdk. It uses
// the existing reader if available, otherwise makes a call to GCS.
// Before calling this method we have to use invalidateReaderIfMisalignedOrTooSmall to get the reader start at the correct position.
func (rr *RangeReader) readFromRangeReader(ctx context.Context, p []byte, offset int64, end int64, readType string) (int, error) {
	var err error
	// If we don't have a reader, start a read operation.
	if rr.reader == nil {
		err = rr.startRead(offset, end)
		if err != nil {
			err = fmt.Errorf("startRead: %w", err)
			return 0, err
		}
	}

	var n int
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
// from GCS defined by SequentialReadSizeMb flag to serve future read requests.
func (rr *RangeReader) startRead(start int64, end int64) error {
	ctx, cancel := context.WithCancel(context.Background())
	var err error

	if rr.readConfig != nil && rr.readConfig.InactiveStreamTimeout > 0 {
		rr.reader, err = gcsx.NewInactiveTimeoutReader(
			ctx,
			rr.bucket,
			rr.object,
			rr.readHandle,
			gcs.ByteRange{
				Start: uint64(start),
				Limit: uint64(end),
			},
			rr.readConfig.InactiveStreamTimeout)
	} else {
		rr.reader, err = rr.bucket.NewReaderWithReadHandle(
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
	}

	// If a file handle is open locally, but the corresponding object doesn't exist
	// in GCS, it indicates a file clobbering scenario. This likely occurred because:
	//  - The file was deleted in GCS while a local handle was still open.
	//  - The file content was modified leading to different generation number.
	var notFoundError *gcs.NotFoundError
	if errors.As(err, &notFoundError) {
		err = &gcsfuse_errors.FileClobberedError{
			Err: fmt.Errorf("NewReader: %w", err),
		}
		cancel()
		return err
	}

	if err != nil {
		err = fmt.Errorf("NewReaderWithReadHandle: %w", err)
		cancel()
		return err
	}

	rr.cancel = cancel
	rr.start = start
	rr.limit = end

	requestedDataSize := end - start
	common.CaptureGCSReadMetrics(ctx, rr.metricHandle, util.Sequential, requestedDataSize)

	return nil
}

// skipBytes attempts to advance the reader position to the given offset without
// discarding the existing reader. If possible, it reads and discards data to
// maintain an active GCS connection, improving throughput for sequential reads.
func (rr *RangeReader) skipBytes(offset int64) {
	// When the offset is AFTER the reader position, try to seek forward, within reason.
	// This happens when the kernel page cache serves some data. It's very common for
	// concurrent reads, often by only a few 128kB fuse read requests. The aim is to
	// re-use GCS connection and avoid throwing away already read data.
	// For parallel sequential reads to a single file, not throwing away the connections
	// is a 15-20x improvement in throughput: 150-200 MiB/s instead of 10 MiB/s.
	if rr.reader != nil && rr.start < offset && offset-rr.start < maxReadSize {
		bytesToSkip := offset - rr.start
		discardedBytes, copyError := io.CopyN(io.Discard, rr.reader, bytesToSkip)
		// io.EOF is expected if the reader is shorter than the requested offset to read.
		if copyError != nil && !errors.Is(copyError, io.EOF) {
			logger.Warnf("Error while skipping reader bytes: %v", copyError)
		}
		rr.start += discardedBytes
	}
}

// invalidateReaderIfMisalignedOrTooSmall ensures that the existing reader is valid
// for the requested offset and length. If the reader is misaligned (not at the requested
// offset) or cannot serve the full request within its limit, it is closed and discarded.
//
// It attempts to skip forward to the requested offset if possible to avoid creating
// a new reader unnecessarily. If the reader is discarded due to misalignment, the method
// returns true to signal that a seek should be recorded.
//
// Parameters:
//   - offset: the starting byte position of the requested read.
//   - p: the buffer representing the size of the requested read.
//
// Returns:
//   - true if the reader was discarded due to being misaligned (seek should be counted).
//   - false otherwise.
func (rr *RangeReader) invalidateReaderIfMisalignedOrTooSmall(offset int64, p []byte) bool {
	rr.skipBytes(offset)

	// If we have an existing reader, but it's positioned at the wrong place,
	// clean it up and throw it away.
	// We will also clean up the existing reader if it can't serve the entire request.
	dataToRead := math.Min(float64(offset+int64(len(p))), float64(rr.object.Size))
	if rr.reader != nil && (rr.start != offset || int64(dataToRead) > rr.limit) {
		rr.closeReader()
		rr.reader = nil
		rr.cancel = nil
		if rr.start != offset {
			// Return true to increment the seek count when discarding a reader due to incorrect positioning.
			// Discarding readers that can't fulfill the entire request without this check would prevent
			// the reader size from growing appropriately in random read scenarios.
			return true
		}
	}
	return false
}

// readFromExistingReader attempts to read data from an existing reader if one is available.
// If a reader exists and the read is successful, the data is returned.
// Otherwise, it returns an error indicating that a fallback to another reader is needed.
// Make sure to call invalidateReaderIfMisalignedOrTooSmall before using this method.
func (rr *RangeReader) readFromExistingReader(ctx context.Context, req *gcsx.GCSReaderRequest) (gcsx.ReaderResponse, error) {
	if rr.reader != nil {
		return rr.ReadAt(ctx, req)
	}

	return gcsx.ReaderResponse{
		DataBuf: req.Buffer,
		Size:    0,
	}, gcsx.FallbackToAnotherReader
}
