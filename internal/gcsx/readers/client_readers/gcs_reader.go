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
	"fmt"

	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx/readers"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
)

// ReaderType represents different types of go-sdk gcs readers.
// For eg: NewReader and MRD both point to bidi read api. This enum specifies
// the go-sdk type.
type ReaderType int

// ReaderType enum values.
const (
	// MB is 1 Megabyte. (Silly comment to make the lint warning go away)
	MB = 1 << 20

	// Min read size in bytes for random reads.
	// We will not send a request to GCS for less than this many bytes (unless the
	// end of the object comes first).
	minReadSize = MB

	// Max read size in bytes for random reads.
	// If the average read size (between seeks) is below this number, reads will
	// optimised for random access.
	// We will skip forwards in a GCS response at most this many bytes.
	// About 6 MB of data is buffered anyway, so 8 MB seems like a good round number.
	maxReadSize = 8 * MB

	// Minimum number of seeks before evaluating if the read pattern is random.
	minSeeksForRandom = 2
)

// ReaderType enum values.
const (
	// RangeReader corresponds to NewReader method in bucket_handle.go
	RangeReaderType ReaderType = iota
	// MultiRangeReader corresponds to NewMultiRangeDownloader method in bucket_handle.go
	MultiRangeReaderType
)

type GCSReader struct {
	obj    *gcs.MinObject
	bucket gcs.Bucket

	rangeReader RangeReader
	mrr         MultiRangeReader
	// Stores the handle associated with the previously closed newReader instance.
	// This will be used while making the new connection to bypass auth and metadata
	// checks.
	readHandle []byte
	readType   string

	sequentialReadSizeMb int32

	seeks          uint64
	totalReadBytes uint64
}

func NewGCSReader(obj *gcs.MinObject, bucket gcs.Bucket, metricHandle common.MetricHandle, mrdWrapper *MultiRangeDownloaderWrapper, sequentialReadSizeMb int32) GCSReader {
	return GCSReader{
		obj:                  obj,
		bucket:               bucket,
		sequentialReadSizeMb: sequentialReadSizeMb,
		rangeReader:          NewRangeReader(obj, bucket, metricHandle),
		mrr:                  NewMultiRangeReader(metricHandle, mrdWrapper),
	}
}

func (gr *GCSReader) Object() *gcs.MinObject {
	return nil
}

func (gr *GCSReader) CheckInvariants() {
}

func (gr *GCSReader) ReadAt(ctx context.Context, p []byte, offset, end int64) (readers.ObjectData, error) {
	objectData := readers.ObjectData{
		DataBuf: p,
		Size:    0,
	}
	var err error
	gr.rangeReader.SkipBytes(offset)

	if gr.rangeReader.DiscardReader(offset, p) {
		gr.seeks++
	}

	if gr.rangeReader.reader != nil {
		objectData, err = gr.rangeReader.ReadAt(ctx, p, offset, -1)

		return objectData, err
	}

	// If we don't have a reader, determine whether to read from NewReader or Mgr.
	end, err = gr.getReadInfo(offset, int64(len(p)))
	if err != nil {
		err = fmt.Errorf("ReadAt: getReaderInfo: %w", err)
		return objectData, err
	}

	readerType := gr.getReaderType(offset, end, gr.bucket.BucketType())
	if readerType == RangeReaderType {
		objectData, err = gr.rangeReader.ReadAt(ctx, p, offset, end)
		gr.totalReadBytes += uint64(objectData.Size)
		return objectData, err
	}

	objectData, err = gr.mrr.ReadAt(ctx, p, offset, end)
	gr.totalReadBytes += uint64(objectData.Size)
	if err != nil {
		return objectData, err
	}

	return objectData, nil
}

// readerType specifies the go-sdk interface to use for reads.
func (gr *GCSReader) getReaderType(start int64, end int64, bucketType gcs.BucketType) ReaderType {
	bytesToBeRead := end - start
	if gr.readType == util.Random && bytesToBeRead < maxReadSize && bucketType.Zonal {
		return MultiRangeReaderType
	}
	return RangeReaderType
}

// getReadInfo determines the readType and provides the range to query GCS.
// Range here is [start, end]. end is computed using the readType, start offset
// and size of the data the callers needs.
func (gr *GCSReader) getReadInfo(start int64, size int64) (int64, error) {
	// Make sure start and size are legal.
	if start < 0 || uint64(start) > gr.obj.Size || size < 0 {
		return 0, fmt.Errorf(
			"range [%d, %d) is illegal for %d-byte object",
			start,
			start+size,
			gr.obj.Size)
	}

	// Determine end based on read pattern.
	end := gr.determineEnd(start)

	// Limit end to sequentialReadSizeMb.
	end = gr.limitToEnd(start, end)

	return end, nil
}

// determineEnd determines the end of the read based on the read pattern.
func (gr *GCSReader) determineEnd(start int64) int64 {
	end := int64(gr.obj.Size)
	if gr.seeks >= minSeeksForRandom {
		gr.readType = util.Random
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
	if end > int64(gr.obj.Size) {
		end = int64(gr.obj.Size)
	}
	return end
}

// limitToEnd limits the end of the read to sequentialReadSizeMb.
func (gr *GCSReader) limitToEnd(start int64, end int64) int64 {
	maxSizeToReadFromGCS := int64(gr.sequentialReadSizeMb * MB)
	if end-start > maxSizeToReadFromGCS {
		end = start + maxSizeToReadFromGCS
	}
	return end
}

func (gr *GCSReader) Destroy() {

}
