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
	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx/readers/cache_readers"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx/readers/gcs_readers"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
)

type readManager struct {
	object *gcs.MinObject
	reader gcs.StorageReader
	cancel func()

	readers []Reader
}

// NewRandomReader create a random reader for the supplied object record that
// reads using the given bucket.
func NewReadManager(o *gcs.MinObject, bucket gcs.Bucket, sequentialReadSizeMb int32, fileCacheHandler *file.CacheHandler, cacheFileForRangeRead bool, metricHandle common.MetricHandle, mrdWrapper *gcs_readers.MultiRangeDownloaderWrapper) Reader {
	var gcsReader gcs_readers.GCSReader
	var fileCacheReader cache_readers.FileCacheReader
	gcsReader = gcs_readers.GCSReader{
		Obj:            o,
		Bucket:         bucket,
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
	}
	fileCacheReader = cache_readers.FileCacheReader{
		Obj:                   o,
		Bucket:                bucket,
		FileCacheHandler:      fileCacheHandler,
		CacheFileForRangeRead: cacheFileForRangeRead,
		MetricHandle:          metricHandle,
	}

	return &readManager{
		object: o,
		readers: []Reader{
			&fileCacheReader,
			&gcsReader,
		},
	}
}

func (rr *readManager) Object() (o *gcs.MinObject) {
	return rr.object
}

func (rr *readManager) CheckInvariants() {
	// INVARIANT: (reader == nil) == (cancel == nil)
	if (rr.reader == nil) != (rr.cancel == nil) {
		panic(fmt.Sprintf("Mismatch: %v vs. %v", rr.reader == nil, rr.cancel == nil))
	}
}

func (rr *readManager) ReadAt(ctx context.Context, p []byte, offset int64) (readers.ObjectData, error) {
	var err error
	objectData := readers.ObjectData{
		DataBuf:  p,
		CacheHit: false,
		Size:     0,
	}

	if offset >= int64(rr.object.Size) {
		err = io.EOF
		return objectData, err
	}

	for _, r := range rr.readers {
		objectData, err = r.ReadAt(ctx, p, offset)
		if err == nil {
			return objectData, err
		}
	}

	return objectData, err
}

func (rr *readManager) Destroy() {
}
