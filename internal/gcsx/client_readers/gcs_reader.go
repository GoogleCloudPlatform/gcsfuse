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
	"sync/atomic"

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
	MB = 1 << 20

	// Min read size in bytes for random reads.
	// We will not send a request to GCS for less than this many bytes (unless the
	// end of the object comes first).
	minReadSize = MB

	// Minimum number of seeks before evaluating if the read pattern is random.
	minSeeksForRandom = 2
)

// ReaderType enum values.
const (
	// RangeReaderType corresponds to NewReader method in bucket_handle.go
	RangeReaderType ReaderType = iota

	// MultiRangeReaderType corresponds to NewMultiRangeDownloader method in bucket_handle.go
	MultiRangeReaderType
)

// readInfo Stores information for this read request.
type readInfo struct {
	// readType stores the read type evaluated for this request.
	readType int64
	// expectedOffset stores the expected offset for this request. Will be
	// used to determine if re-evaluation of readType is required or not with range reader.
	expectedOffset int64
	// seekRecorded tells whether a seek has been performed for this read request.
	seekRecorded bool
}

type GCSReader struct {
	gcsx.Reader
	object *gcs.MinObject
	bucket gcs.Bucket

	rangeReader *RangeReader
	mrr         *MultiRangeReader

	// ReadType of the reader. Will be sequential by default.
	readType atomic.Int64

	sequentialReadSizeMb int32

	// Specifies the next expected offset for the reads. Used to distinguish between
	// sequential and random reads.
	expectedOffset atomic.Int64

	// seeks represents the number of random reads performed by the reader.
	seeks atomic.Uint64

	// totalReadBytes is the total number of bytes read by the reader.
	totalReadBytes atomic.Uint64

	// mu synchronizes reads through range reader.
	mu sync.Mutex

	readPatternTracker *gcsx.ReadPatternTracker
}

type GCSReaderConfig struct {
	MetricHandle         metrics.MetricHandle
	MrdWrapper           *gcsx.MultiRangeDownloaderWrapper
	SequentialReadSizeMb int32
	Config               *cfg.Config
	ReadPatternTracker    *gcsx.ReadPatternTracker
}

func NewGCSReader(obj *gcs.MinObject, bucket gcs.Bucket, config *GCSReaderConfig) *GCSReader {
	return &GCSReader{
		object:               obj,
		bucket:               bucket,
		sequentialReadSizeMb: config.SequentialReadSizeMb,
		rangeReader:          NewRangeReader(obj, bucket, config.Config, config.MetricHandle),
		mrr:                  NewMultiRangeReader(obj, config.MetricHandle, config.MrdWrapper),
		readPatternTracker:   config.ReadPatternTracker,
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

func (gr *GCSReader) ReadAt(ctx context.Context, p []byte, offset int64) (readerResponse gcsx.ReaderResponse, err error) {

	if offset >= int64(gr.object.Size) {
		return readerResponse, io.EOF
	} else if offset < 0 {
		err := fmt.Errorf(
			"illegal offset %d for %d byte object",
			offset,
			gr.object.Size)
		return readerResponse, err
	}

	readReq := &gcsx.GCSReaderRequest{
		Buffer:            p,
		Offset:            offset,
		EndOffset:         offset + int64(len(p)),
		ForceCreateReader: false,
	}
	defer func() {
		readerResponse.DataBuf = p
	}()

	bytesRead, err := gr.read(ctx, readReq)
	readerResponse.Size = bytesRead

	// Retry reading in case of short read.
	if shouldRetryForShortRead(err, bytesRead, p, offset, gr.object.Size, gr.bucket.BucketType()) {
		readReq.Offset += int64(bytesRead)
		readReq.Buffer = p[bytesRead:]
		readReq.ForceCreateReader = true
		var bytesReadOnRetry int
		bytesReadOnRetry, err = gr.read(ctx, readReq)
		readerResponse.Size += bytesReadOnRetry
	}

	return readerResponse, err
}

func (gr *GCSReader) read(ctx context.Context, readReq *gcsx.GCSReaderRequest) (bytesRead int, err error) {
	// Not taking any lock for getting reader type to ensure random read requests do not wait.
	readInfo := gr.readPatternTracker.GetReadInfo(readReq.Offset, true)
	reqReaderType := gr.readerType(readInfo.ReadType, gr.bucket.BucketType())
	var readerResp gcsx.ReaderResponse

	if reqReaderType == RangeReaderType {
		gr.mu.Lock()

		// In case of multiple threads reading parallely, it is possible that many of them might be waiting
		// at this lock and hence the earlier calculated value of readerType might not be valid once they
		// acquire the lock. Hence, needs to be calculated again.
		// Recalculating only for ZB and only when another read had been performed between now and
		// the time when readerType was calculated for this request.
		if gr.bucket.BucketType().Zonal && readInfo.ExpectedOffset != gr.expectedOffset.Load() {
			readInfo = gr.readPatternTracker.GetReadInfo(readReq.Offset, true)
			reqReaderType = gr.readerType(readInfo.ReadType, gr.bucket.BucketType())
		}
		// If the readerType is range reader after re calculation, then use range reader.
		// Otherwise fall back to MultiRange Downloder
		if reqReaderType == RangeReaderType {
			defer gr.mu.Unlock()
			// Calculate the end offset based on previous read requests.
			// It will be used if a new range reader needs to be created.
			readReq.EndOffset = gr.getEndOffset(readReq.Offset)
			readerResp, err = gr.rangeReader.ReadAt(ctx, readReq)
			return readerResp.Size, err
		}
		gr.mu.Unlock()
	}

	readerResp, err = gr.mrr.ReadAt(ctx, readReq)
	return readerResp.Size, err
}

// readerType specifies the go-sdk interface to use for reads.
func (gr *GCSReader) readerType(readType int64, bucketType gcs.BucketType) ReaderType {
	if readType == metrics.ReadTypeRandom && bucketType.Zonal {
		return MultiRangeReaderType
	}
	return RangeReaderType
}

func (gr *GCSReader) getEndOffset(
	start int64) (end int64) {
	end = start + gr.readPatternTracker.SeqReadIO()

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
