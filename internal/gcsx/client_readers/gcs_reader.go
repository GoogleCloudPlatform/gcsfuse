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
	"sync"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
)

// ReaderType represents different types of go-sdk gcs readers.
type ReaderType int

// ReaderType enum values.
const (
	// RangeReaderType corresponds to NewReader method in bucket_handle.go
	RangeReaderType ReaderType = iota

	// MultiRangeReaderType corresponds to NewMultiRangeDownloader method in bucket_handle.go
	MultiRangeReaderType
)

type GCSReader struct {
	gcsx.Reader
	object *gcs.MinObject
	bucket gcs.Bucket

	rangeReader *RangeReader
	mrr         *MultiRangeReader

	// mu synchronizes reads through range reader.
	mu sync.Mutex

	// readTypeClassifier tracks the read access pattern (e.g., sequential, random)
	// to optimize read strategies. It is shared across different reader layers.
	readTypeClassifier *gcsx.ReadTypeClassifier
}

type GCSReaderConfig struct {
	MetricHandle       metrics.MetricHandle
	MrdWrapper         *gcsx.MultiRangeDownloaderWrapper
	Config             *cfg.Config
	ReadTypeClassifier *gcsx.ReadTypeClassifier
}

func NewGCSReader(obj *gcs.MinObject, bucket gcs.Bucket, config *GCSReaderConfig) *GCSReader {
	return &GCSReader{
		object:             obj,
		bucket:             bucket,
		rangeReader:        NewRangeReader(obj, bucket, config.Config, config.MetricHandle),
		mrr:                NewMultiRangeReader(obj, config.MetricHandle, config.MrdWrapper),
		readTypeClassifier: config.ReadTypeClassifier,
	}
}

// Detects whether the read was short or not and returns whether it should be retried or not.
// Reads would only be retried in case of zonal buckets and when the read data was less than requested (& object size)
// and there was no error apart from EOF or short reads.
func shouldRetryForShortRead(err error, bytesRead int, p []byte, offset int64, objectSize uint64, bucketType gcs.BucketType) bool {
	if !bucketType.Zonal {
		return false
	}

	if bytesRead >= len(p) {
		return false
	}

	if offset+int64(bytesRead) >= int64(objectSize) {
		return false
	}

	if !(err == nil || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, util.ErrShortRead)) {
		return false
	}

	return true
}

func (gr *GCSReader) ReadAt(ctx context.Context, req *gcsx.ReadRequest) (readResponse gcsx.ReadResponse, err error) {

	if req.Offset >= int64(gr.object.Size) {
		return readResponse, io.EOF
	} else if req.Offset < 0 {
		err := fmt.Errorf(
			"illegal offset %d for %d byte object",
			req.Offset,
			gr.object.Size)
		return readResponse, err
	}

	readReq := &gcsx.GCSReaderRequest{
		ReadRequest:       *req,
		EndOffset:         req.Offset + int64(len(req.Buffer)),
		ForceCreateReader: false,
	}

	bytesRead, err := gr.read(ctx, readReq)
	readResponse.Size = bytesRead

	// Retry reading in case of short read.
	if shouldRetryForShortRead(err, bytesRead, req.Buffer, req.Offset, gr.object.Size, gr.bucket.BucketType()) {
		readReq.Offset += int64(bytesRead)
		readReq.Buffer = req.Buffer[bytesRead:]
		readReq.ForceCreateReader = true
		var bytesReadOnRetry int
		bytesReadOnRetry, err = gr.read(ctx, readReq)
		readResponse.Size += bytesReadOnRetry
	}

	return readResponse, err
}

func (gr *GCSReader) read(ctx context.Context, readReq *gcsx.GCSReaderRequest) (bytesRead int, err error) {
	// We don't take a lock here to allow random reads to proceed without waiting.
	// The read type is re-evaluated for zonal buckets inside the lock if necessary.
	reqReaderType := gr.readerType(readReq.ReadType, gr.bucket.BucketType())
	var readResp gcsx.ReadResponse

	if reqReaderType == RangeReaderType {
		gr.mu.Lock()

		// In case of multiple threads reading parallely, it is possible that many of them might be waiting
		// at this lock and hence the earlier calculated value of readerType might not be valid once they
		// acquire the lock. Hence, needs to be calculated again.
		// Recalculating only for ZB and only when another read had been performed between now and
		// the time when readerType was calculated for this request.
		newReadInfo := readReq.ReadInfo
		if gr.bucket.BucketType().Zonal && readReq.ExpectedOffset != gr.readTypeClassifier.NextExpectedOffset() {
			newReadInfo = gr.readTypeClassifier.GetReadInfo(readReq.Offset, readReq.SeekRecorded)
			reqReaderType = gr.readerType(newReadInfo.ReadType, gr.bucket.BucketType())
		}
		// If the readerType is range reader after re calculation, then use range reader.
		// Otherwise fall back to MultiRange Downloder
		if reqReaderType == RangeReaderType {
			defer gr.mu.Unlock()
			// Calculate the end offset based on previous read requests.
			// It will be used if a new range reader needs to be created.
			readReq.EndOffset = gr.getEndOffset(readReq.Offset)
			readReq.ReadType = newReadInfo.ReadType
			readResp, err = gr.rangeReader.ReadAt(ctx, readReq)
			return readResp.Size, err
		}
		gr.mu.Unlock()
	}

	readResp, err = gr.mrr.ReadAt(ctx, readReq)
	return readResp.Size, err
}

// readerType specifies the go-sdk interface to use for reads.
func (gr *GCSReader) readerType(readType int64, bucketType gcs.BucketType) ReaderType {
	if readType == metrics.ReadTypeRandom && bucketType.Zonal {
		return MultiRangeReaderType
	}
	return RangeReaderType
}

func (gr *GCSReader) getEndOffset(start int64) int64 {
	end := start + gr.readTypeClassifier.ComputeSeqPrefetchWindowAndAdjustType()
	if end > int64(gr.object.Size) {
		end = int64(gr.object.Size)
	}
	return end
}

func (gr *GCSReader) Destroy() {
	gr.mu.Lock()
	defer gr.mu.Unlock()
	gr.rangeReader.destroy()
	gr.mrr.destroy()
}

func (gr *GCSReader) CheckInvariants() {
	gr.rangeReader.checkInvariants()
}
