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
	readType string

	sequentialReadSizeMb int32

	// Specifies the next expected offset for the reads. Used to distinguish between
	// sequential and random reads.
	expectedOffset int64

	// seeks represents the number of random reads performed by the reader.
	seeks uint64

	// totalReadBytes is the total number of bytes read by the reader.
	totalReadBytes uint64
}

type GCSReaderConfig struct {
	MetricHandle         common.MetricHandle
	MrdWrapper           *gcsx.MultiRangeDownloaderWrapper
	SequentialReadSizeMb int32
	ReadConfig           *cfg.ReadConfig
}

func NewGCSReader(obj *gcs.MinObject, bucket gcs.Bucket, config *GCSReaderConfig) *GCSReader {
	return &GCSReader{
		object:               obj,
		bucket:               bucket,
		sequentialReadSizeMb: config.SequentialReadSizeMb,
		rangeReader:          NewRangeReader(obj, bucket, config.ReadConfig, config.MetricHandle),
		mrr:                  NewMultiRangeReader(obj, config.MetricHandle, config.MrdWrapper),
		readType:             common.ReadTypeSequential,
	}
}

func (gr *GCSReader) ReadAt(ctx context.Context, p []byte, offset int64) (gcsx.ReaderResponse, error) {
	var readerResponse gcsx.ReaderResponse

	if offset >= int64(gr.object.Size) {
		return readerResponse, io.EOF
	}

	gr.rangeReader.invalidateReaderIfMisalignedOrTooSmall(offset, p)

	readReq := &gcsx.GCSReaderRequest{
		Buffer:    p,
		Offset:    offset,
		EndOffset: -1,
	}
	defer func() {
		gr.updateExpectedOffset(offset + int64(readerResponse.Size))
		gr.totalReadBytes += uint64(readerResponse.Size)
	}()

	var err error
	readerResponse, err = gr.rangeReader.readFromExistingReader(ctx, readReq)
	if err == nil {
		return readerResponse, nil
	}
	if !errors.Is(err, gcsx.FallbackToAnotherReader) {
		return readerResponse, err
	}

	// If the data can't be served from the existing reader, then we need to update the seeks.
	// If current offset is not same as expected offset, it's a random read.
	if gr.expectedOffset != 0 && gr.expectedOffset != offset {
		gr.seeks++
	}

	// If we don't have a reader, determine whether to read from RangeReader or MultiRangeReader.
	end, err := gr.getReadInfo(offset, int64(len(p)))
	if err != nil {
		err = fmt.Errorf("ReadAt: getReaderInfo: %w", err)
		return readerResponse, err
	}

	readReq.EndOffset = end
	readerType := gr.readerType(offset, end, gr.bucket.BucketType())
	if readerType == RangeReaderType {
		readerResponse, err = gr.rangeReader.ReadAt(ctx, readReq)
		return readerResponse, err
	}

	readerResponse, err = gr.mrr.ReadAt(ctx, readReq)

	return readerResponse, err
}

// readerType specifies the go-sdk interface to use for reads.
func (gr *GCSReader) readerType(start int64, end int64, bucketType gcs.BucketType) ReaderType {
	bytesToBeRead := end - start
	if gr.readType == common.ReadTypeRandom && bytesToBeRead < maxReadSize && bucketType.Zonal {
		return MultiRangeReaderType
	}
	return RangeReaderType
}

// getReadInfo determines the readType and provides the range to query GCS.
// Range here is [start, end]. end is computed using the readType, start offset
// and size of the data the caller needs.
func (gr *GCSReader) getReadInfo(start int64, size int64) (int64, error) {
	// Make sure start and size are legal.
	if start < 0 || uint64(start) > gr.object.Size || size < 0 {
		return 0, fmt.Errorf("range [%d, %d) is illegal for %d-byte object", start, start+size, gr.object.Size)
	}

	// Determine the end position based on the read pattern.
	end := gr.determineEnd(start)

	// Limit the end position to sequentialReadSizeMb.
	end = gr.limitEnd(start, end)

	return end, nil
}

// determineEnd calculates the end position for a read operation based on the current read pattern.
func (gr *GCSReader) determineEnd(start int64) int64 {
	end := int64(gr.object.Size)
	if gr.seeks >= minSeeksForRandom {
		gr.readType = common.ReadTypeRandom
		averageReadBytes := gr.totalReadBytes / gr.seeks
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
	gr.expectedOffset = offset
}

func (gr *GCSReader) Destroy() {
	gr.rangeReader.destroy()
	gr.mrr.destroy()
}

func (gr *GCSReader) CheckInvariants() {
	gr.rangeReader.checkInvariants()
}
