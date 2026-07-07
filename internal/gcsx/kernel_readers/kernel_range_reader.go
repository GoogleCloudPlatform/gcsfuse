// Copyright 2026 Google LLC
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

package kernel_readers

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/gcsfuse_errors"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
)

// KernelRangeReader implements the Reader interface for regional buckets.
// It serves read requests by creating a new GCS range reader for each request.
type KernelRangeReader struct {
	bucket   gcs.Bucket
	instance *KernelRangeReaderInstance
	metrics  metrics.MetricHandle
}

// NewKernelRangeReader creates a new KernelRangeReader.
func NewKernelRangeReader(bucket gcs.Bucket, instance *KernelRangeReaderInstance, metricHandle metrics.MetricHandle) *KernelRangeReader {
	return &KernelRangeReader{
		bucket:   bucket,
		instance: instance,
		metrics:  metricHandle,
	}
}

// CheckInvariants performs internal consistency checks on the reader state.
func (krr *KernelRangeReader) CheckInvariants() {
	if krr.instance == nil {
		panic("KernelRangeReader: instance is nil")
	}
	if krr.bucket == nil {
		panic("KernelRangeReader: bucket is nil")
	}
}

// ReadAt reads data from the object by creating a new range reader.
func (krr *KernelRangeReader) ReadAt(ctx context.Context, req *gcsx.ReadRequest) (gcsx.ReadResponse, error) {
	var resp gcsx.ReadResponse

	obj := krr.instance.GetMinObject()
	if obj == nil {
		return resp, fmt.Errorf("KernelRangeReader::ReadAt Nil MinObject")
	}

	if req.Offset >= int64(obj.Size) {
		return resp, io.EOF
	} else if req.Offset < 0 {
		return resp, fmt.Errorf("KernelRangeReader::ReadAt: illegal offset %d for %d byte object", req.Offset, obj.Size)
	}

	// If the destination buffer is empty, there's nothing to read.
	if len(req.Buffer) == 0 && req.BufferPool == nil {
		return resp, nil
	}

	limit := int64(obj.Size) - req.Offset
	bytesToRead := req.GetReadSize(limit)
	endOffset := req.Offset + bytesToRead

	reader, err := krr.bucket.NewReaderWithReadHandle(
		ctx,
		&gcs.ReadObjectRequest{
			Name:       obj.Name,
			Generation: obj.Generation,
			Range: &gcs.ByteRange{
				Start: uint64(req.Offset),
				Limit: uint64(endOffset),
			},
			ReadCompressed: obj.HasContentEncodingGzip(),
		})

	// If a file handle is open locally, but the corresponding object doesn't exist
	// in GCS, it indicates a file clobbering scenario. This likely occurred because:
	//  - The file was deleted in GCS while a local handle was still open.
	//  - The file content was modified leading to different generation number.
	var notFoundError *gcs.NotFoundError
	if errors.As(err, &notFoundError) {
		return resp, &gcsfuse_errors.FileClobberedError{
			Err:        fmt.Errorf("KernelRangeReader::ReadAt Failed to create range reader: %w", err),
			ObjectName: obj.Name,
		}
	}
	if err != nil {
		return resp, fmt.Errorf("KernelRangeReader::ReadAt Failed to create range reader: %w", err)
	}
	defer func() {
		if err := reader.Close(); err != nil {
			logger.Warnf("KernelRangeReader::ReadAt Error while closing reader: %v", err)
		}
	}()

	var n int
	if req.BufferPool != nil {
		writer := gcsx.NewVectoredWriter(req.BufferPool, bytesToRead)
		var written int64
		written, err = io.CopyN(writer, reader, bytesToRead)
		n = int(written)
		if err == io.EOF && written > 0 {
			err = io.ErrUnexpectedEOF
		}
		if n > 0 {
			resp.Data = writer.Buffers()
			resp.Callback = func() { writer.Release() }
		} else {
			writer.Release() // Release immediately on 0-byte read or early error
		}
	} else {
		n, err = io.ReadFull(reader, req.Buffer[:bytesToRead])
	}
	resp.Size = n

	if krr.metrics != nil {
		metrics.CaptureGCSReadMetrics(krr.metrics, metrics.ReadTypeParallelAttr, int64(n))
		krr.metrics.GcsReadBytesCount(int64(n))
	}

	return resp, err
}

// Destroy releases resources.
func (krr *KernelRangeReader) Destroy() {
}

// ReaderName returns the reader name.
func (krr *KernelRangeReader) ReaderName() string {
	return "KernelRangeReader"
}
