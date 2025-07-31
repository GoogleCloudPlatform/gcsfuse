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
	"sync"
	"sync/atomic"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
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

type readInfo struct {
	readType       int64
	expectedOffset int64
	seekRecorded   bool
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
}

type GCSReaderConfig struct {
	MetricHandle         metrics.MetricHandle
	MrdWrapper           *gcsx.MultiRangeDownloaderWrapper
	SequentialReadSizeMb int32
	Config               *cfg.Config
}

func NewGCSReader(obj *gcs.MinObject, bucket gcs.Bucket, config *GCSReaderConfig) *GCSReader {
	return &GCSReader{
		object:               obj,
		bucket:               bucket,
		sequentialReadSizeMb: config.SequentialReadSizeMb,
		rangeReader:          NewRangeReader(obj, bucket, config.Config, config.MetricHandle),
		mrr:                  NewMultiRangeReader(obj, config.MetricHandle, config.MrdWrapper),
	}
}

func (gr *GCSReader) ReadAt(ctx context.Context, p []byte, offset int64) (readerResponse gcsx.ReaderResponse, err error) {

	if offset >= int64(gr.object.Size) {
		return readerResponse, io.EOF
	} else if offset < 0 {
		err := fmt.Errorf(
			"illegal offset %d for %d-byte object",
			offset,
			gr.object.Size)
		return readerResponse, err
	}

	readReq := &gcsx.GCSReaderRequest{
		Buffer:    p,
		Offset:    offset,
		EndOffset: offset + int64(len(p)),
	}
	defer func() {
		gr.updateExpectedOffset(offset + int64(readerResponse.Size))
		gr.totalReadBytes.Add(uint64(readerResponse.Size))
	}()

	// Not taking any lock for getting reader type to ensure random read requests do not wait.
	readInfo := gr.getReadInfo(offset, false)
	reqReaderType := gr.readerType(readInfo.readType, gr.bucket.BucketType())

	if reqReaderType == RangeReaderType {
		gr.mu.Lock()
		if gr.bucket.BucketType().Zonal && readInfo.expectedOffset != gr.expectedOffset.Load() {
			readInfo = gr.getReadInfo(offset, readInfo.seekRecorded)
			reqReaderType = gr.readerType(readInfo.readType, gr.bucket.BucketType())
		}
		if reqReaderType == RangeReaderType {
			defer gr.mu.Unlock()
			readReq.EndOffset = gr.getEndOffset(readReq.Offset)
			readerResponse, err = gr.rangeReader.ReadAt(ctx, readReq)
			return
		}
		gr.mu.Unlock()
	}

	if reqReaderType == MultiRangeReaderType {
		readerResponse, err = gr.mrr.ReadAt(ctx, readReq)
	}

	return
}

// readerType specifies the go-sdk interface to use for reads.
func (gr *GCSReader) readerType(start int64, bucketType gcs.BucketType) ReaderType {
	if gr.readType.Load() == metrics.ReadTypeRandom && bucketType.Zonal {
		return MultiRangeReaderType
	}
	return RangeReaderType
}

// isSeekNeeded determines if the current read at `offset` should be considered a
// seek, given the previous read pattern & the expected offset.
func isSeekNeeded(readType, offset, expectedOffset int64) bool {
	if expectedOffset == 0 {
		return false
	}

	if readType == metrics.ReadTypeRandom {
		return offset != expectedOffset
	}

	if readType == metrics.ReadTypeSequential {
		return offset < expectedOffset || offset > expectedOffset+maxReadSize
	}

	return false
}

func (gr *GCSReader) getEndOffset(
	start int64) (end int64) {

	end = gr.determineEnd(start)
	end = gr.limitEnd(start, end)
	return
}

// getReadInfo determines the read strategy (sequential or random) for a read
// request at a given offset and returns read metadata. It also updates the
// reader's internal state based on the read pattern.
func (gr *GCSReader) getReadInfo(offset int64, seekRecorded bool) readInfo {
	readType := gr.readType.Load()
	expOffset := gr.expectedOffset.Load()
	numSeeks := gr.seeks.Load()

	if !seekRecorded && isSeekNeeded(readType, offset, expOffset) {
		numSeeks = gr.seeks.Add(1)
		seekRecorded = true
	}

	if numSeeks >= minSeeksForRandom {
		readType = metrics.ReadTypeRandom
	}

	averageReadBytes := gr.totalReadBytes.Load()
	if numSeeks > 0 {
		averageReadBytes /= numSeeks
	}

	if averageReadBytes >= maxReadSize {
		readType = metrics.ReadTypeSequential
	}

	gr.readType.Store(readType)
	return readInfo{
		readType:       readType,
		expectedOffset: expOffset,
		seekRecorded:   seekRecorded,
	}
}

// determineEnd calculates the end position for a read operation based on the current read pattern.
func (gr *GCSReader) determineEnd(start int64) int64 {
	end := int64(gr.object.Size)
	if seeks := gr.seeks.Load(); seeks >= minSeeksForRandom {
		gr.readType.Store(metrics.ReadTypeRandom)
		averageReadBytes := gr.totalReadBytes.Load() / seeks
		if averageReadBytes < maxReadSize {
			randomReadSize := int64(((averageReadBytes / MB) + 1) * MB)
			if randomReadSize < minReadSize {
				randomReadSize = minReadSize
			}
			if randomReadSize > maxReadSize {
				randomReadSize = maxReadSize
			}
			end = start + randomReadSize
		}
	}
	if end > int64(gr.object.Size) {
		end = int64(gr.object.Size)
	}
	return end
}

// Limit the read end to ensure it doesn't exceed the maximum sequential read size.
func (gr *GCSReader) limitEnd(start, currentEnd int64) int64 {
	maxSize := int64(gr.sequentialReadSizeMb) * MB
	if currentEnd-start > maxSize {
		return start + maxSize
	}
	return currentEnd
}

func (gr *GCSReader) updateExpectedOffset(offset int64) {
	gr.expectedOffset.Store(offset)
}

func (gr *GCSReader) Destroy() {
	gr.rangeReader.destroy()
	gr.mrr.destroy()
}

func (gr *GCSReader) CheckInvariants() {
	gr.rangeReader.checkInvariants()
}
