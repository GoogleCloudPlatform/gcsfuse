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

	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/file"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx/readers"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx/readers/gcs_readers"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
)

type readManager struct {
	object *gcs.MinObject
	bucket gcs.Bucket

	// If non-nil, an in-flight read request and a function for cancelling it.
	//
	// INVARIANT: (reader == nil) == (cancel == nil)
	reader gcs.StorageReader
	cancel func()

	// The range of the object that we expect reader to yield, when reader is
	// non-nil. When reader is nil, limit is the limit of the previous read
	// operation, or -1 if there has never been one.
	//
	// INVARIANT: start <= limit
	// INVARIANT: limit < 0 implies reader != nil
	// All these properties will be used only in case of GCS reads and not for
	// reads from cache.
	start          int64
	limit          int64
	seeks          uint64
	totalReadBytes uint64

	// ReadType of the reader. Will be sequential by default.
	readType string

	sequentialReadSizeMb int32

	// Stores the handle associated with the previously closed newReader instance.
	// This will be used while making the new connection to bypass auth and metadata
	// checks.
	readHandle []byte

	gcsReader       readers.GCSReader
	fileCacheReader readers.FileCacheReader
}

// NewRandomReader create a random reader for the supplied object record that
// reads using the given bucket.
func NewReadManager(o *gcs.MinObject, bucket gcs.Bucket, sequentialReadSizeMb int32, fileCacheHandler *file.CacheHandler, cacheFileForRangeRead bool, metricHandle common.MetricHandle, mrdWrapper *gcs_readers.MultiRangeDownloaderWrapper) Reader {
	return &readManager{
		object:               o,
		bucket:               bucket,
		start:                -1,
		limit:                -1,
		seeks:                0,
		totalReadBytes:       0,
		readType:             util.Sequential,
		sequentialReadSizeMb: sequentialReadSizeMb,
		gcsReader: readers.GCSReader{
			Obj:            o,
			Bucket:         bucket,
			Start:          -1,
			Limit:          -1,
			Seeks:          0,
			TotalReadBytes: 0,
			RangeReader: gcs_readers.RangeReader{
				Obj:            o,
				Bucket:         bucket,
				Start:          -1,
				Limit:          -1,
				Seeks:          0,
				MetricHandle:   metricHandle,
				TotalReadBytes: 0,
			},
			Mrr: gcs_readers.MultiRangeReader{
				MrdWrapper:   mrdWrapper,
				MetricHandle: metricHandle,
			},
			SequentialReadSizeMb: sequentialReadSizeMb,
		},
		fileCacheReader: readers.FileCacheReader{
			Obj:                   o,
			Bucket:                bucket,
			FileCacheHandler:      fileCacheHandler,
			CacheFileForRangeRead: cacheFileForRangeRead,
			MetricHandle:          metricHandle,
		},
	}
}

func (rr *readManager) Object() (o *gcs.MinObject) {
	o = rr.object
	return
}

func (rr *readManager) CheckInvariants() {
	// INVARIANT: (reader == nil) == (cancel == nil)
	if (rr.reader == nil) != (rr.cancel == nil) {
		panic(fmt.Sprintf("Mismatch: %v vs. %v", rr.reader == nil, rr.cancel == nil))
	}

	// INVARIANT: start <= limit
	if !(rr.start <= rr.limit) {
		panic(fmt.Sprintf("Unexpected range: [%d, %d)", rr.start, rr.limit))
	}

	// INVARIANT: limit < 0 implies reader != nil
	if rr.limit < 0 && rr.reader != nil {
		panic(fmt.Sprintf("Unexpected non-nil reader with limit == %d", rr.limit))
	}
}

func (rr *readManager) ReadAt(ctx context.Context, p []byte, offset int64) (gcs_readers.ObjectData, error) {
	var err error
	objectData := gcs_readers.ObjectData{
		DataBuf:  p,
		CacheHit: false,
		Size:     0,
	}

	if offset >= int64(rr.object.Size) {
		err = io.EOF
		return objectData, err
	}

	objectData, err = rr.fileCacheReader.ReadAt(ctx, p, offset)
	if err != nil {
		err = fmt.Errorf("ReadAt: while reading from cache: %w", err)
		return objectData, err
	}
	if objectData.CacheHit || objectData.Size == len(p) {
		return objectData, nil
	}

	objectData, err = rr.gcsReader.ReadAt(ctx, p, offset)

	return objectData, err
}

func (rr *readManager) Destroy() {
	defer rr.gcsReader.Mrr.Destroy()

	// Close out the reader, if we have one.
	rr.gcsReader.RangeReader.Destroy()

	rr.fileCacheReader.Destroy()
}
