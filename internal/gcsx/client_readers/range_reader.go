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
	"sync"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/gcsfuse_errors"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
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

	config       *cfg.Config
	metricHandle metrics.MetricHandle

	// mu synchronizes reads through range reader.
	mu sync.Mutex
}

func NewRangeReader(object *gcs.MinObject, bucket gcs.Bucket, config *cfg.Config, metricHandle metrics.MetricHandle) *RangeReader {
	return &RangeReader{
		object:       object,
		bucket:       bucket,
		metricHandle: metricHandle,
		config:       config,
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
	rr.mu.Lock()
	defer rr.mu.Unlock()
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

func (rr *RangeReader) ReadAt(ctx context.Context, req *gcsx.GCSReaderRequest) (gcsx.ReadResponse, error) {
	var (
		readResponse gcsx.ReadResponse
		err          error
	)
	rr.mu.Lock()
	defer rr.mu.Unlock()

	// Re-evaluate if RangeReader should be used, as the read pattern might have
	// changed while waiting for the lock.
	if !req.ShouldUseRangeReader(req.Offset) {
		return readResponse, gcsx.FallbackToAnotherReader
	}

	if req.ForceCreateReader && rr.reader != nil {
		rr.closeReader()
		rr.reader = nil
		rr.cancel = nil
		rr.start = -1
		rr.limit = -1
	}

	// Ensure we have a valid reader for the request, creating one if necessary.
	err = rr.ensureReader(req.Offset, req.EndOffset, req.ReadType)
	if err != nil {
		return readResponse, fmt.Errorf("ensureReader: %w", err)
	}

	// Now that we have a valid reader, perform the read.
	readResponse.Size, err = rr.readFromReader(ctx, req.Buffer)

	return readResponse, err
}

// ensureReader makes sure that rr.reader is valid for a read at the given
// offset. If the existing reader is misaligned or nil, it creates a new one.
func (rr *RangeReader) ensureReader(offset int64, end int64, readType int64) (err error) {
	// Try to reuse the existing reader by skipping forward if it's a small gap.
	rr.skipBytes(offset)

	// If the reader is still misaligned (or nil), create a new one.
	rr.invalidateReaderIfMisaligned(offset)
	// If we don't have a reader, start a read operation.
	if rr.reader == nil {
		err = rr.startRead(offset, end, readType)
		if err != nil {
			err = fmt.Errorf("startRead: %w", err)
			return err
		}
	}

	return
}

// readFromReader reads from the existing rr.reader into the provided buffer.
// It assumes ensureReader has already been called to set up a valid reader.
func (rr *RangeReader) readFromReader(ctx context.Context, p []byte) (n int, err error) {
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
			err = fmt.Errorf("range reader returned early by skipping %d bytes: %w", rr.limit-rr.start, util.ErrShortRead)
			return 0, err
		}

		err = nil

	case err != nil:
		// Propagate other errors.
		err = fmt.Errorf("readFull: %w", err)
		return 0, err
	}

	return n, err
}

// Like io.ReadFull, but deals with the cancellation issues.
//
// REQUIRES: rr.reader != nil
func (rr *RangeReader) readFull(ctx context.Context, p []byte) (int, error) {
	if rr.config != nil && !rr.config.FileSystem.IgnoreInterrupts {
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
	}

	return io.ReadFull(rr.reader, p)
}

// Ensure that rr.reader is set up for a range for which [start, start+size) is
// a prefix. Irrespective of the size requested, we try to fetch more data
// from GCS defined by SequentialReadSizeMb flag to serve future read requests.
func (rr *RangeReader) startRead(start int64, end int64, readType int64) error {
	ctx, cancel := context.WithCancel(context.Background())
	var err error

	if rr.config != nil && rr.config.Read.InactiveStreamTimeout > 0 {
		rr.reader, err = gcsx.NewInactiveTimeoutReader(
			ctx,
			rr.bucket,
			rr.object,
			rr.readHandle,
			gcs.ByteRange{
				Start: uint64(start),
				Limit: uint64(end),
			},
			rr.config.Read.InactiveStreamTimeout)
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
			Err:        fmt.Errorf("NewReader: %w", err),
			ObjectName: rr.object.Name,
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
	metrics.CaptureGCSReadMetrics(rr.metricHandle, metrics.ReadTypeNames[readType], requestedDataSize)

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

// invalidateReaderIfMisaligned ensures that the existing reader is valid for
// the requested offset. If the reader is misaligned (not at the requested
// offset), it is closed and discarded.
func (rr *RangeReader) invalidateReaderIfMisaligned(startOffset int64) {
	// If we have an existing reader, but it's positioned at the wrong place,
	// clean it up and throw it away.
	if rr.reader != nil && rr.start != startOffset {
		rr.closeReader()
		rr.reader = nil
		rr.cancel = nil
	}
}
