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

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
)

// RangeKernelReader implements the Reader interface for regional buckets.
// It serves read requests by creating a new GCS range reader for each request.
type RangeKernelReader struct {
	bucket  gcs.Bucket
	object  *gcs.MinObject
	metrics metrics.MetricHandle
}

// NewRangeKernelReader creates a new RangeKernelReader.
func NewRangeKernelReader(bucket gcs.Bucket, object *gcs.MinObject, metricHandle metrics.MetricHandle) *RangeKernelReader {
	return &RangeKernelReader{
		bucket:  bucket,
		object:  object,
		metrics: metricHandle,
	}
}

// CheckInvariants performs internal consistency checks on the reader state.
func (rkr *RangeKernelReader) CheckInvariants() {
	if rkr.object == nil {
		panic("RangeKernelReader: object is nil")
	}
	if rkr.bucket == nil {
		panic("RangeKernelReader: bucket is nil")
	}
}

// ReadAt reads data from the object by creating a new range reader.
func (rkr *RangeKernelReader) ReadAt(ctx context.Context, req *ReadRequest) (ReadResponse, error) {
	var resp ReadResponse

	if req.Offset >= int64(rkr.object.Size) {
		return resp, io.EOF
	}

	endOffset := req.Offset + int64(len(req.Buffer))
	if endOffset > int64(rkr.object.Size) {
		endOffset = int64(rkr.object.Size)
	}

	reader, err := rkr.bucket.NewReaderWithReadHandle(
		ctx,
		&gcs.ReadObjectRequest{
			Name:       rkr.object.Name,
			Generation: rkr.object.Generation,
			Range: &gcs.ByteRange{
				Start: uint64(req.Offset),
				Limit: uint64(endOffset),
			},
			ReadCompressed: rkr.object.HasContentEncodingGzip(),
		})
	if err != nil {
		return resp, fmt.Errorf("failed to create range reader: %w", err)
	}
	defer reader.Close()

	n, err := io.ReadFull(reader, req.Buffer[:endOffset-req.Offset])
	if err == io.ErrUnexpectedEOF || err == io.EOF {
		err = nil
	}
	resp.Size = n

	if rkr.metrics != nil {
		metrics.CaptureGCSReadMetrics(rkr.metrics, metrics.ReadTypeParallelAttr, int64(n))
		rkr.metrics.GcsReadBytesCount(int64(n))
	}

	return resp, err
}

// Destroy releases resources.
func (rkr *RangeKernelReader) Destroy() {
}

// ReaderName returns the reader name.
func (rkr *RangeKernelReader) ReaderName() string {
	return "RangeKernelReader"
}
