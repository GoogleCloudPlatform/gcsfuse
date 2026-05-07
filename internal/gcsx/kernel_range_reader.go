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

// KernelRangeReader implements the Reader interface for regional buckets.
// It serves read requests by creating a new GCS range reader for each request.
type KernelRangeReader struct {
	bucket  gcs.Bucket
	object  *gcs.MinObject
	metrics metrics.MetricHandle
}

// NewKernelRangeReader creates a new KernelRangeReader.
func NewKernelRangeReader(bucket gcs.Bucket, object *gcs.MinObject, metricHandle metrics.MetricHandle) *KernelRangeReader {
	return &KernelRangeReader{
		bucket:  bucket,
		object:  object,
		metrics: metricHandle,
	}
}

// CheckInvariants performs internal consistency checks on the reader state.
func (rkr *KernelRangeReader) CheckInvariants() {
	if rkr.object == nil {
		panic("KernelRangeReader: object is nil")
	}
	if rkr.bucket == nil {
		panic("KernelRangeReader: bucket is nil")
	}
}

// ReadAt reads data from the object by creating a new range reader.
func (rkr *KernelRangeReader) ReadAt(ctx context.Context, req *ReadRequest) (ReadResponse, error) {
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
func (rkr *KernelRangeReader) Destroy() {
}

// ReaderName returns the reader name.
func (rkr *KernelRangeReader) ReaderName() string {
	return "KernelRangeReader"
}
