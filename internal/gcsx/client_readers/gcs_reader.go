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
	"github.com/googlecloudplatform/gcsfuse/v3/tracing"
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
	traceHandle        tracing.TraceHandle
}

type GCSReaderConfig struct {
	MetricHandle       metrics.MetricHandle
	TraceHandle        tracing.TraceHandle
	MrdWrapper         *gcsx.MultiRangeDownloaderWrapper
	Config             *cfg.Config
	ReadTypeClassifier *gcsx.ReadTypeClassifier
}

func NewGCSReader(obj *gcs.MinObject, bucket gcs.Bucket, config *GCSReaderConfig) *GCSReader {
	if config.TraceHandle == nil {
		config.TraceHandle = tracing.NewNoopTracer()
	}

	return &GCSReader{
		object:             obj,
		bucket:             bucket,
		rangeReader:        NewRangeReader(obj, bucket, config.Config, config.MetricHandle, config.TraceHandle),
		mrr:                NewMultiRangeReader(obj, config.MetricHandle, config.TraceHandle, config.MrdWrapper),
		readTypeClassifier: config.ReadTypeClassifier,
		traceHandle:        config.TraceHandle,
	}
}

func shouldRetryForShortRead(err error, bytesRead int, p []byte, offset int64, objectSize uint64, bucketType gcs.BucketType, skipSizeChecks bool) bool {
	if !bucketType.IsRapid() {
		return false
	}

	if bytesRead >= len(p) {
		return false
	}

	if offset+int64(bytesRead) >= int64(objectSize) && !skipSizeChecks {
		return false
	}

	if !(err == nil || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, util.ErrShortRead)) {
		return false
	}

	return true
}

func (gr *GCSReader) ReaderName() string {
	return "gcs_reader"
}

func (gr *GCSReader) ReadAt(ctx context.Context, readRequest *gcsx.ReadRequest) (readResponse gcsx.ReadResponse, err error) {
	if readRequest.Offset >= int64(gr.object.Size) && !readRequest.SkipSizeChecks {
		return readResponse, io.EOF
	} else if readRequest.Offset < 0 {
		err := fmt.Errorf(
			"illegal offset %d for %d byte object",
			readRequest.Offset,
			gr.object.Size)
		return readResponse, err
	}

	gcsReaderRequest := &gcsx.GCSReaderRequest{
		Buffer:            readRequest.Buffer,
		Offset:            readRequest.Offset,
		EndOffset:         readRequest.Offset + int64(len(readRequest.Buffer)),
		ReadInfo:          &readRequest.ReadInfo,
		ForceCreateReader: false,
		SkipSizeChecks:    readRequest.SkipSizeChecks,
	}

	readResponse, err = gr.read(ctx, gcsReaderRequest)

	// Retry reading in case of short read.
	if shouldRetryForShortRead(err, readResponse.Size, readRequest.Buffer, readRequest.Offset, gr.object.Size, gr.bucket.BucketType(), readRequest.SkipSizeChecks) {
		bytesRead := readResponse.Size
		gcsReaderRequest.Offset += int64(bytesRead)
		gcsReaderRequest.Buffer = readRequest.Buffer[bytesRead:]
		gcsReaderRequest.ForceCreateReader = true
		var retryResp gcsx.ReadResponse
		retryResp, err = gr.read(ctx, gcsReaderRequest)
		if len(readResponse.Data) > 0 || len(retryResp.Data) > 0 || readResponse.Callback != nil || retryResp.Callback != nil {
			if len(readResponse.Data) == 0 && readResponse.Size > 0 {
				readResponse.Data = [][]byte{readRequest.Buffer[:readResponse.Size]}
			}
			if len(retryResp.Data) == 0 && retryResp.Size > 0 {
				retryResp.Data = [][]byte{gcsReaderRequest.Buffer[:retryResp.Size]}
			}
			if totalSlices := len(readResponse.Data) + len(retryResp.Data); totalSlices > 0 {
				combined := make([][]byte, 0, totalSlices)
				combined = append(combined, readResponse.Data...)
				combined = append(combined, retryResp.Data...)
				readResponse.Data = combined
			}
			cb1, cb2 := readResponse.Callback, retryResp.Callback
			readResponse.Callback = func() {
				if cb1 != nil {
					cb1()
				}
				if cb2 != nil {
					cb2()
				}
			}
		}
		readResponse.Size += retryResp.Size
	}

	if errors.Is(err, io.EOF) && readResponse.Size > 0 {
		err = nil
	}
	if err != nil {
		if readResponse.Callback != nil {
			readResponse.Callback()
		}
		return gcsx.ReadResponse{}, err
	}

	return readResponse, err
}

func (gr *GCSReader) read(ctx context.Context, readReq *gcsx.GCSReaderRequest) (readResp gcsx.ReadResponse, err error) {
	reqReaderType := gr.readerType(readReq.ReadType, gr.bucket.BucketType())

	if reqReaderType == RangeReaderType && !readReq.SkipSizeChecks {
		gr.mu.Lock()

		if readReq.ExpectedOffset != gr.readTypeClassifier.NextExpectedOffset() {
			*readReq.ReadInfo = gr.readTypeClassifier.GetReadInfo(readReq.Offset, readReq.SeekRecorded)
			reqReaderType = gr.readerType(readReq.ReadType, gr.bucket.BucketType())
		}
		if reqReaderType == RangeReaderType {
			defer gr.mu.Unlock()
			readReq.EndOffset = gr.getEndOffset(readReq.Offset)
			return gr.rangeReader.ReadAt(ctx, readReq)
		}
		gr.mu.Unlock()
	}

	return gr.mrr.ReadAt(ctx, readReq)
}

func (gr *GCSReader) readerType(readType int64, bucketType gcs.BucketType) ReaderType {
	if readType == metrics.ReadTypeRandom && bucketType.IsRapid() {
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
	if !gr.mu.TryLock() {
		return
	}
	defer gr.mu.Unlock()
	gr.rangeReader.checkInvariants()
}
