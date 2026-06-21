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
	"fmt"
	"io"

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
	}

	var totalCapacity int64
	if len(req.Buffers) == 0 {
		totalCapacity = int64(len(req.Buffer))
	} else {
		for _, b := range req.Buffers {
			totalCapacity += int64(len(b))
		}
	}

	sizeToRead := req.Size
	if sizeToRead <= 0 || sizeToRead > totalCapacity {
		sizeToRead = totalCapacity
	}

	endOffset := req.Offset + sizeToRead
	if endOffset > int64(obj.Size) {
		endOffset = int64(obj.Size)
	}

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
	if err != nil {
		return resp, fmt.Errorf("KernelRangeReader::ReadAt Failed to create range reader: %w", err)
	}
	defer func() {
		if err := reader.Close(); err != nil {
			logger.Warnf("KernelRangeReader::ReadAt Error while closing reader: %v", err)
		}
	}()

	var n int
	if len(req.Buffers) == 0 {
		n, err = io.ReadFull(reader, req.Buffer[:endOffset-req.Offset])
	} else {
		bytesToRead := endOffset - req.Offset
		for _, b := range req.Buffers {
			remaining := bytesToRead - int64(n)
			if remaining <= 0 {
				break
			}
			bufToRead := b
			if int64(len(bufToRead)) > remaining {
				bufToRead = bufToRead[:remaining]
			}
			var bytesRead int
			bytesRead, err = io.ReadFull(reader, bufToRead)
			n += bytesRead
			if err != nil {
				break
			}
		}
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
