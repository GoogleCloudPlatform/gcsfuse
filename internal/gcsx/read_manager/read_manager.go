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
	clientReaders "github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx/client_readers"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
)

type ReadManager struct {
	gcsx.ReadManager
	object *gcs.MinObject

	// readers holds a list of data readers, prioritized for reading.
	// e.g., File cache reader, GCS reader.
	readers []gcsx.Reader
}

// NewReadManager creates a new ReadManager for the given GCS object.
// It initializes the manager with a file cache reader and a GCS reader.
func NewReadManager(o *gcs.MinObject, bucket gcs.Bucket, sequentialReadSizeMB int32, fileCacheHandler *file.CacheHandler, cacheFileForRangeRead bool, metricHandle common.MetricHandle, mrdWrapper *gcsx.MultiRangeDownloaderWrapper) *ReadManager {
	gcsReader := clientReaders.NewGCSReader(o, bucket, metricHandle, mrdWrapper, sequentialReadSizeMB)
	fileCacheReader := gcsx.NewFileCacheReader(o, bucket, fileCacheHandler, cacheFileForRangeRead, metricHandle)

	return &ReadManager{
		object: o,
		// Readers are prioritized in this order: file cache first, then GCS.
		readers: []gcsx.Reader{
			fileCacheReader,
			gcsReader,
		},
	}
}

func (rr *ReadManager) Object() *gcs.MinObject {
	return rr.object
}

func (rr *ReadManager) CheckInvariants() {
	for _, r := range rr.readers {
		r.CheckInvariants()
	}
}

// ReadAt attempts to read data from the provided offset, using the configured readers.
// It prioritizes readers in the order they are defined (file cache first, then GCS).
// If a reader returns a FallbackToAnotherReader error, it tries the next reader.
func (rr *ReadManager) ReadAt(ctx context.Context, p []byte, offset int64) (gcsx.ReaderResponse, error) {
	if offset >= int64(rr.object.Size) {
		return gcsx.ReaderResponse{}, io.EOF
	}

	var readerResponse gcsx.ReaderResponse
	var err error
	for _, r := range rr.readers {
		readerResponse, err = r.ReadAt(ctx, p, offset)
		if err == nil {
			return readerResponse, nil
		}
		if !errors.Is(err, gcsx.FallbackToAnotherReader) {
			// Non-fallback error, return it.
			return readerResponse, err
		}
		// Fallback to the next reader.
	}

	// If all readers failed with FallbackToAnotherReader, return the last response and error.
	// This case should not happen as the last reader should always succeed.
	return readerResponse, err
}

func (rr *ReadManager) Destroy() {
	for _, r := range rr.readers {
		r.Destroy()
	}
}
