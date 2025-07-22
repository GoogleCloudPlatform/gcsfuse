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
	"github.com/googlecloudplatform/gcsfuse/v3/common"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
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

	rrMu sync.Mutex
}

type GCSReaderConfig struct {
	MetricHandle         common.MetricHandle
	MrdWrapper           *gcsx.MultiRangeDownloaderWrapper
	SequentialReadSizeMb int32
	ReadConfig           *cfg.ReadConfig
}

type readInfo struct {
	readType       int64
	expectedOffset int64
}

func NewGCSReader(obj *gcs.MinObject, bucket gcs.Bucket, config *GCSReaderConfig) *GCSReader {
	return &GCSReader{
		object:               obj,
		bucket:               bucket,
		sequentialReadSizeMb: config.SequentialReadSizeMb,
		rangeReader:          NewRangeReader(obj, bucket, config.ReadConfig, config.MetricHandle, config.SequentialReadSizeMb),
		mrr:                  NewMultiRangeReader(obj, config.MetricHandle, config.MrdWrapper),
	}
}

func (gr *GCSReader) ReadAt(ctx context.Context, p []byte, offset int64) (gcsx.ReaderResponse, error) {
	var readerResponse gcsx.ReaderResponse
	var err error

	if offset >= int64(gr.object.Size) {
		return readerResponse, io.EOF
	} else if offset < 0 {
		return readerResponse, fmt.Errorf("Illegal offset %d for read", offset)
	}

	readInfo := gr.getReadInfo(offset)
	reader := gr.readerType(readInfo.readType, gr.bucket.BucketType())

	readReq := &gcsx.GCSReaderRequest{
		Buffer:    p,
		Offset:    offset,
		EndOffset: offset + int64(len(p)),
	}
	defer func() {
		gr.updateExpectedOffset(offset + int64(readerResponse.Size))
		gr.totalReadBytes.Add(uint64(readerResponse.Size))
	}()

	if reader == RangeReaderType {
		gr.rrMu.Lock()

		if readInfo.expectedOffset != gr.expectedOffset.Load() {
			readInfo = gr.getReadInfo(offset)
			reader = gr.readerType(readInfo.readType, gr.bucket.BucketType())
		}

		if reader == MultiRangeReaderType {
			gr.rrMu.Unlock()
		} else if reader == RangeReaderType {
			defer gr.rrMu.Unlock()
			readReq.EndOffset = gr.getEndOffsetForRangeReader(offset)
			readerResponse, err = gr.rangeReader.ReadAt(ctx, readReq)
		}
	}

	if reader == MultiRangeReaderType {
		readerResponse, err = gr.mrr.ReadAt(ctx, readReq)
	}

	return readerResponse, err
}

// readerType specifies the go-sdk interface to use for reads.
func (gr *GCSReader) readerType(readType int64, bucketType gcs.BucketType) ReaderType {
	if readType == common.ReadTypeRandom && bucketType.Zonal {
		return MultiRangeReaderType
	}
	return RangeReaderType
}

func (gr *GCSReader) getReadInfo(offset int64) readInfo {
	readType := gr.readType.Load()
	expOffset := gr.expectedOffset.Load()
	numSeeks := gr.seeks.Load()

	// Here, we will be different from existing algorithm in only one scenario
	// When the read type is sequential but the reader has been closed (due to any reason)
	// fmt.Printf("Expected Offset is %d; Offset is %d\n", expOffset, offset)
	seqReadSeekNeeded := (readType == common.ReadTypeSequential) && (offset < expOffset || offset > expOffset+maxReadSize)

	randomReadSeekNeeded := (readType == common.ReadTypeRandom) && (expOffset != offset)
	if expOffset != 0 && (seqReadSeekNeeded || randomReadSeekNeeded) {
		numSeeks = gr.seeks.Add(1)
	}

	if numSeeks >= minSeeksForRandom {
		readType = common.ReadTypeRandom
	}

	if numSeeks == 0 {
		numSeeks = 1
	}
	averageReadBytes := int64(gr.totalReadBytes.Load() / numSeeks)
	if averageReadBytes >= maxReadSize {
		readType = common.ReadTypeSequential
	}
	gr.readType.Store(readType)
	return readInfo{
		readType:       readType,
		expectedOffset: expOffset,
	}
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

func (gr *GCSReader) getEndOffsetForRangeReader(start int64) (end int64) {
	totalReadBytes := gr.totalReadBytes.Load()
	numSeeks := gr.seeks.Load()
	end = int64(gr.object.Size)
	if numSeeks > minSeeksForRandom {
		averageReadBytes := totalReadBytes / numSeeks
		if averageReadBytes < maxReadSize {
			randomReadSize := int64(((averageReadBytes / MiB) + 1) * MiB)
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

	// To avoid overloading GCS and to have reasonable latencies, we will only
	// fetch data of max size defined by sequentialReadSizeMb.
	maxSizeToReadFromGCS := int64(gr.sequentialReadSizeMb * MiB)
	if end-start > maxSizeToReadFromGCS {
		end = start + maxSizeToReadFromGCS
	}
	return
}
