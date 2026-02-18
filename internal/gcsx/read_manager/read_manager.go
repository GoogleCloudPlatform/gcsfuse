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

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/jacobsa/fuse/fuseops"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/bufferedread"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/file"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	clientReaders "github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx/client_readers"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/workerpool"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/googlecloudplatform/gcsfuse/v3/tracing"
	"golang.org/x/sync/semaphore"
)

type ReadManager struct {
	gcsx.ReadManager
	object *gcs.MinObject

	// readers holds a list of data readers, prioritized for reading.
	// e.g., File cache reader, GCS reader.
	readers []gcsx.Reader

	// readTypeClassifier tracks the read access pattern (e.g., sequential, random)
	// across all readers for a file handle to optimize read strategies.
	readTypeClassifier *gcsx.ReadTypeClassifier

	traceHandle tracing.TraceHandle
}

// ReadManagerConfig holds the configuration parameters for creating a new ReadManager.
type ReadManagerConfig struct {
	SequentialReadSizeMB int32
	// Exactly one of these should be set:
	FileCacheHandler        *file.CacheHandler
	SharedChunkCacheManager *file.SharedChunkCacheManager
	CacheFileForRangeRead   bool
	MetricHandle            metrics.MetricHandle
	TraceHandle             tracing.TraceHandle
	MrdWrapper              *gcsx.MultiRangeDownloaderWrapper
	Config                  *cfg.Config
	GlobalMaxBlocksSem      *semaphore.Weighted
	WorkerPool              workerpool.WorkerPool
	HandleID                fuseops.HandleID
	InitialOffset           int64
}

// NewReadManager creates a new ReadManager for the given GCS object,
// using the provided configuration. It initializes the manager with a
// file cache reader and a GCS reader, prioritizing the file cache reader if available.
func NewReadManager(object *gcs.MinObject, bucket gcs.Bucket, config *ReadManagerConfig) *ReadManager {
	// Create a slice to hold all readers. The file cache reader will be added first if it exists.
	var readers []gcsx.Reader

	// If a shared chunk cache handler is provided, use it
	if config.SharedChunkCacheManager != nil {
		// For SharedChunkCacheManager, create ShareChunkCacheReader directly
		reader := gcsx.NewSharedChunkCacheReader(
			config.SharedChunkCacheManager,
			bucket,
			object,
			config.MetricHandle,
			config.TraceHandle,
			config.HandleID,
		)
		readers = append(readers, reader)
	} else if config.FileCacheHandler != nil {
		// For traditional cache handler, use FileCacheReader
		fileCacheReader := gcsx.NewFileCacheReader(
			object,
			bucket,
			config.FileCacheHandler,
			config.CacheFileForRangeRead,
			config.MetricHandle,
			config.TraceHandle,
			config.HandleID,
		)
		readers = append(readers, fileCacheReader)
	}

	readClassifier := gcsx.NewReadTypeClassifier(int64(config.SequentialReadSizeMB), config.InitialOffset)

	// If buffered read is enabled, initialize the buffered reader and add it to the readers.
	if config.Config.Read.EnableBufferedRead {
		readConfig := config.Config.Read
		bufferedReadConfig := &bufferedread.BufferedReadConfig{
			MaxPrefetchBlockCnt:     readConfig.MaxBlocksPerHandle,
			PrefetchBlockSizeBytes:  readConfig.BlockSizeMb * util.MiB,
			InitialPrefetchBlockCnt: readConfig.StartBlocksPerHandle,
			MinBlocksPerHandle:      readConfig.MinBlocksPerHandle,
			RandomSeekThreshold:     readConfig.RandomSeekThreshold,
		}
		opts := &bufferedread.BufferedReaderOptions{
			Object:             object,
			Bucket:             bucket,
			Config:             bufferedReadConfig,
			GlobalMaxBlocksSem: config.GlobalMaxBlocksSem,
			WorkerPool:         config.WorkerPool,
			MetricHandle:       config.MetricHandle,
			TraceHandle:        config.TraceHandle,
			ReadTypeClassifier: readClassifier,
			HandleID:           config.HandleID,
		}
		bufferedReader, err := bufferedread.NewBufferedReader(opts)
		if err != nil {
			logger.Tracef("Failed to create bufferedReader: %v. Buffered reading will be disabled for this file handle.", err)
		} else {
			readers = append(readers, bufferedReader)
		}
	}

	// Initialize the GCS reader, which is always present.
	gcsReader := clientReaders.NewGCSReader(
		object,
		bucket,
		&clientReaders.GCSReaderConfig{
			MetricHandle:       config.MetricHandle,
			MrdWrapper:         config.MrdWrapper,
			Config:             config.Config,
			ReadTypeClassifier: readClassifier,
		},
	)
	// Add the GCS reader as a fallback.
	readers = append(readers, gcsReader)

	return &ReadManager{
		object:             object,
		readers:            readers, // Readers are prioritized: file cache first, then GCS.
		readTypeClassifier: readClassifier,
		traceHandle:        config.TraceHandle,
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
func (rr *ReadManager) ReadAt(ctx context.Context, req *gcsx.ReadRequest) (gcsx.ReadResponse, error) {
	var readResponse gcsx.ReadResponse
	if req.Offset >= int64(rr.object.Size) && !req.SkipSizeChecks {
		return readResponse, io.EOF
	}

	// empty read
	if len(req.Buffer) == 0 {
		return readResponse, nil
	}

	// Get read-related information (e.g., read type) and add it to the read request.
	// This information is used by underlying readers to optimize read strategies
	// based on the access pattern.
	req.ReadInfo = rr.readTypeClassifier.GetReadInfo(req.Offset, false)

	var err error
	for _, r := range rr.readers {
		readResponse, err = r.ReadAt(ctx, req)
		if err == nil {
			rr.readTypeClassifier.RecordRead(req.Offset, int64(readResponse.Size))
			return readResponse, nil
		}
		if !errors.Is(err, gcsx.FallbackToAnotherReader) {
			// Non-fallback error, return it.
			return readResponse, err
		}
		// Fallback to the next reader.
	}

	// If all readers failed with FallbackToAnotherReader, return the last response and error.
	// This case should not happen as the last reader should always succeed.
	return readResponse, err
}

func (rr *ReadManager) Destroy() {
	for _, r := range rr.readers {
		r.Destroy()
	}
}
