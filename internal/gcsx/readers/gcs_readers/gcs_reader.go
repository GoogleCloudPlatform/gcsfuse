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

package gcs_readers

import (
	"context"
	"fmt"
	"log"

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
	readerType string

	sequentialReadSizeMb int32
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

func (gr *GCSReader) ReadAt(ctx context.Context, p []byte, offset int64) (readers.ObjectData, error) {
	objectData := readers.ObjectData{
		DataBuf:                 p,
		CacheHit:                false,
		Size:                    0,
		FallBackToAnotherReader: false,
	}
	var err error
	gr.rangeReader.SkipBytes(offset)

	gr.rangeReader.DiscardReader(offset, p)

	if gr.rangeReader.reader != nil {
		objectData, err = gr.rangeReader.ReadAt(ctx, p, offset)

		return objectData, err
	}

	// If we don't have a reader, determine whether to read from NewReader or Mgr.
	end, err := gr.getReadInfo(offset, int64(len(p)))
	if err != nil {
		err = fmt.Errorf("ReadAt: getReaderInfo: %w", err)
		return objectData, err
	}
	gr.rangeReader.SetEnd(end)
	gr.mrr.SetEnd(end)

	readerType := gr.getReaderType(gr.readerType, offset, end, gr.bucket.BucketType())
	if readerType == RangeReaderType {
		objectData, err = gr.rangeReader.ReadAt(ctx, p, offset)
		return objectData, err
	}

	objectData, err = gr.mrr.ReadAt(ctx, p, offset)
	if err != nil {
		return objectData, err
	}

	return objectData, nil
}

// readerType specifies the go-sdk interface to use for reads.
func (gr *GCSReader) getReaderType(readType string, start int64, end int64, bucketType gcs.BucketType) ReaderType {
	bytesToBeRead := end - start
	if readType == util.Random && bytesToBeRead < maxReadSize && bucketType.Zonal {
		return MultiRangeReaderType
	}
	return RangeReaderType
}

// getReaderInfo determines the readType and provides the range to query GCS.
// Range here is [start, end]. end is computed using the readType, start offset
// and size of the data the callers needs.
func (gr *GCSReader) getReadInfo(start int64, size int64) (int64, error) {
	// Make sure start and size are legal.
	if start < 0 || uint64(start) > gr.obj.Size || size < 0 {
		err := fmt.Errorf(
			"range [%d, %d) is illegal for %d-byte object",
			start,
			start+size,
			gr.obj.Size)
		return 0, err
	}

	// GCS requests are expensive. Prefer to issue read requests defined by
	// sequentialReadSizeMb flag. Sequential reads will simply sip from the fire house
	// with each call to ReadAt. In practice, GCS will fill the TCP buffers
	// with about 6 MB of data. Requests from outside GCP will be charged
	// about 6MB of egress data, even if less data is read. Inside GCP
	// regions, GCS egress is free. This logic should limit the number of
	// GCS read requests, which are not free.

	// But if we notice random read patterns after a minimum number of seeks,
	// optimise for random reads. Random reads will read data in chunks of
	// (average read size in bytes rounded up to the next MB).
	end := int64(gr.obj.Size)
	if gr.rangeReader.GetSeeks() >= minSeeksForRandom {
		gr.readerType = util.Random
		averageReadBytes := gr.rangeReader.GetTotalBytes() / gr.rangeReader.GetSeeks()
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

	// To avoid overloading GCS and to have reasonable latencies, we will only
	// fetch data of max size defined by sequentialReadSizeMb.
	maxSizeToReadFromGCS := int64(gr.sequentialReadSizeMb * MB)
	log.Println("end-start", end-start)
	log.Println("maxSizeToReadFromGCS", maxSizeToReadFromGCS)
	if end-start > maxSizeToReadFromGCS {
		end = start + maxSizeToReadFromGCS
	}

	return end, nil
}

func (gr *GCSReader) Destroy() {

}
