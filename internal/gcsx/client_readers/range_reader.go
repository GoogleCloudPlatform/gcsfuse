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
	"context"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/gcsfuse_errors"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/googlecloudplatform/gcsfuse/v3/tracing"
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
	traceHandle  tracing.TraceHandle
}

func NewRangeReader(object *gcs.MinObject, bucket gcs.Bucket, config *cfg.Config, metricHandle metrics.MetricHandle, traceHandle tracing.TraceHandle) *RangeReader {
	if traceHandle == nil {
		traceHandle = tracing.NewNoopTracer()
	}

	return &RangeReader{
		object:       object,
		bucket:       bucket,
		metricHandle: metricHandle,
		traceHandle:  traceHandle,
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

	if req.ForceCreateReader && rr.reader != nil {
		rr.closeReader()
		rr.reader = nil
		rr.cancel = nil
		rr.start = -1
		rr.limit = -1
	}

	readResponse, err = rr.readFromExistingReader(ctx, req)
	if errors.Is(err, gcsx.FallbackToAnotherReader) {
		readResponse, err = rr.readFromRangeReaderResp(ctx, req.Buffer, req.Offset, req.EndOffset, req.ReadType)
	}
	return readResponse, err
}

func (rr *RangeReader) readFromRangeReader(ctx context.Context, p []byte, offset int64, end int64, readType int64) (int, error) {
	resp, err := rr.readFromRangeReaderResp(ctx, p, offset, end, readType)
	if len(resp.Data) > 0 && !(len(resp.Data) == 1 && len(resp.Data[0]) > 0 && len(p) > 0 && &resp.Data[0][0] == &p[0]) {
		pos := 0
		for _, s := range resp.Data {
			pos += copy(p[pos:], s)
		}
	}
	if resp.Callback != nil {
		resp.Callback()
	}
	return resp.Size, err
}

// readFromRangeReaderResp reads using the NewReader interface of go-sdk. It uses
// the existing reader if available, otherwise makes a call to GCS.
// Before calling this method we have to use invalidateReaderIfMisalignedOrTooSmall to get the reader start at the correct position.
func (rr *RangeReader) readFromRangeReaderResp(ctx context.Context, p []byte, offset int64, end int64, readType int64) (gcsx.ReadResponse, error) {
	var err error
	// If we do not have a reader, start a read operation.
	if rr.reader == nil {
		err = rr.startRead(ctx, offset, end, readType)
		if err != nil {
			err = fmt.Errorf("startRead: %w", err)
			return gcsx.ReadResponse{}, err
		}
	}

	var resp gcsx.ReadResponse
	resp, err = rr.readFull(ctx, p)
	rr.start += int64(resp.Size)

	// Sanity check.
	if rr.start > rr.limit {
		if resp.Callback != nil {
			resp.Callback()
		}
		err = fmt.Errorf("reader returned extra bytes: %d", rr.start-rr.limit)

		// Do not attempt to reuse the reader when it is malfunctioning.
		rr.closeReader()
		rr.reader = nil
		rr.cancel = nil
		rr.start = -1
		rr.limit = -1

		return gcsx.ReadResponse{}, err
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
		// if the reader peters out early. That is fine, but it means we should
		// have hit the limit above.
		if rr.reader != nil {
			if resp.Callback != nil {
				resp.Callback()
			}
			err = fmt.Errorf("range reader returned early by skipping %d bytes: %w", rr.limit-rr.start, util.ErrShortRead)
			return gcsx.ReadResponse{}, err
		}

		err = nil

	case err != nil:
		if resp.Callback != nil {
			resp.Callback()
		}
		// Propagate other errors.
		err = fmt.Errorf("readFull: %w", err)
		return gcsx.ReadResponse{}, err
	}

	return resp, err
}

// Like io.ReadFull, but deals with the cancellation issues.
//
// REQUIRES: rr.reader != nil
func (rr *RangeReader) readFull(ctx context.Context, p []byte) (gcsx.ReadResponse, error) {
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

	if vr, ok := rr.reader.(gcs.VectoredStorageReader); ok {
		data, n, cb, err := vr.ReadVectored(p)
		return gcsx.ReadResponse{Data: data, Size: n, Callback: cb}, err
	}
	n, err := io.ReadFull(rr.reader, p)
	return gcsx.ReadResponse{Size: n}, err
}

// Ensure that rr.reader is set up for a range for which [start, start+size) is
// a prefix. Irrespective of the size requested, we try to fetch more data
// from GCS defined by SequentialReadSizeMb flag to serve future read requests.
func (rr *RangeReader) startRead(ctx context.Context, start int64, end int64, readType int64) error {
	ctx, cancel := context.WithCancel(rr.traceHandle.PropagateTraceContext(context.Background(), ctx))
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

	var notFoundError *gcs.NotFoundError
	if errors.As(err, &notFoundError) {
		rr.reader = nil
		err = &gcsfuse_errors.FileClobberedError{
			Err:        fmt.Errorf("NewReader: %w", err),
			ObjectName: rr.object.Name,
		}
		cancel()
		return err
	}

	if err != nil {
		rr.reader = nil
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
	if rr.reader != nil && rr.start < offset && offset-rr.start < maxReadSize {
		bytesToSkip := offset - rr.start
		discardedBytes, copyError := io.CopyN(io.Discard, rr.reader, bytesToSkip)
		if copyError != nil && !errors.Is(copyError, io.EOF) {
			logger.Warnf("Error while skipping reader bytes: %v", copyError)
		}
		rr.start += discardedBytes
	}
}

func (rr *RangeReader) invalidateReaderIfMisalignedOrTooSmall(startOffset, endOffset int64) {
	dataToRead := math.Min(float64(endOffset), float64(rr.object.Size))
	if rr.reader != nil && (rr.start != startOffset || int64(dataToRead) > rr.limit) {
		rr.closeReader()
		rr.reader = nil
		rr.cancel = nil
	}
}

func (rr *RangeReader) readFromExistingReader(ctx context.Context, req *gcsx.GCSReaderRequest) (gcsx.ReadResponse, error) {
	rr.skipBytes(req.Offset)
	endOffset := min(req.Offset + int64(len(req.Buffer)))

	rr.invalidateReaderIfMisalignedOrTooSmall(req.Offset, endOffset)
	if rr.reader != nil {
		return rr.readFromRangeReaderResp(ctx, req.Buffer, req.Offset, endOffset, req.ReadType)
	}

	return gcsx.ReadResponse{}, gcsx.FallbackToAnotherReader
}
