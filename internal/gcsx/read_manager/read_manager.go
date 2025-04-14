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

package read_manager

import (
	"context"
	"errors"
	"io"

	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/file"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx"
	client_readers2 "github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx/client_readers"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
)

type readManager struct {
	object  *gcs.MinObject
	readers []gcsx.Reader
	gcsx.ReadManager
}

// NewRandomReader create a random reader for the supplied object record that
// reads using the given bucket.
func NewReadManager(o *gcs.MinObject, bucket gcs.Bucket, sequentialReadSizeMb int32, fileCacheHandler *file.CacheHandler, cacheFileForRangeRead bool, metricHandle common.MetricHandle, mrdWrapper *gcsx.MultiRangeDownloaderWrapper, offset int64) (gcsx.ReadManager, error) {
	gcsReader := client_readers2.NewGCSReader(o, bucket, metricHandle, mrdWrapper, sequentialReadSizeMb)
	fileCacheReader, err := gcsx.NewFileCacheReader(o, bucket, fileCacheHandler, cacheFileForRangeRead, metricHandle, offset)
	if err != nil {
		return nil, err
	}

	return &readManager{
		object: o,
		readers: []gcsx.Reader{
			fileCacheReader,
			&gcsReader,
		},
	}, nil
}

func (rr *readManager) Object() (o *gcs.MinObject) {
	return rr.object
}

func (rr *readManager) CheckInvariants() {
	for _, r := range rr.readers {
		r.CheckInvariants()
	}
}

func (rr *readManager) ReadAt(ctx context.Context, p []byte, offset int64) (gcsx.ObjectData, error) {
	var err error
	var objectData gcsx.ObjectData

	if offset >= int64(rr.object.Size) {
		err = io.EOF
		return objectData, err
	}

	for _, r := range rr.readers {
		objectData, err = r.ReadAt(ctx, p, offset)
		if err == nil {
			return objectData, nil
		}
		if !errors.As(err, &gcsx.FallbackToAnotherReader) {
			return objectData, err
		}
	}

	return objectData, err
}

func (rr *readManager) Destroy() {
	for _, r := range rr.readers {
		r.Destroy()
	}
}
