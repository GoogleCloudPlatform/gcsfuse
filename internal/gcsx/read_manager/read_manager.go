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

	"github.com/googlecloudplatform/gcsfuse/v3/internal/bufferedread"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/file"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	clientReaders "github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx/client_readers"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/workerpool"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"golang.org/x/sync/semaphore"
)

type ReadManager struct {
	gcsx.ReadManager
	object *gcs.MinObject

	// readers holds a list of data readers, prioritized for reading.
	// e.g., File cache reader, GCS reader.
	readers []gcsx.Reader

	// sharedReadState holds shared state across all readers for this file handle
	sharedReadState *gcsx.SharedReadState

	// config stores the original configuration for potential buffered reader restart
	config *ReadManagerConfig

	// bucket stores the bucket reference for reader recreation
	bucket gcs.Bucket
}

// ReadManagerConfig holds the configuration parameters for creating a new ReadManager.
type ReadManagerConfig struct {
	SequentialReadSizeMB  int32
	FileCacheHandler      *file.CacheHandler
	CacheFileForRangeRead bool
	MetricHandle          metrics.MetricHandle
	MrdWrapper            *gcsx.MultiRangeDownloaderWrapper
	Config                *cfg.Config
	GlobalMaxBlocksSem    *semaphore.Weighted
	WorkerPool            workerpool.WorkerPool
	SharedReadState       *gcsx.SharedReadState
}

// NewReadManager creates a new ReadManager for the given GCS object,
// using the provided configuration. It initializes the manager with a
// file cache reader and a GCS reader, prioritizing the file cache reader if available.
func NewReadManager(object *gcs.MinObject, bucket gcs.Bucket, config *ReadManagerConfig) *ReadManager {
	// Create a slice to hold all readers. The file cache reader will be added first if it exists.
	var readers []gcsx.Reader

	// If no shared read state is provided, create a default one
	sharedReadState := config.SharedReadState
	if sharedReadState == nil {
		sharedReadState = gcsx.NewSharedReadState()
	}

	// If a file cache handler is provided, initialize the file cache reader and add it to the readers slice first.
	if config.FileCacheHandler != nil {
		fileCacheReader := gcsx.NewFileCacheReader(
			object,
			bucket,
			config.FileCacheHandler,
			config.CacheFileForRangeRead,
			config.MetricHandle,
		)
		readers = append(readers, fileCacheReader) // File cache reader is prioritized.
		sharedReadState.SetActiveReaderType("FileCacheReader")
	}

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
		bufferedReader, err := bufferedread.NewBufferedReader(
			object,
			bucket,
			bufferedReadConfig,
			config.GlobalMaxBlocksSem,
			config.WorkerPool,
			config.MetricHandle,
		)
		if err != nil {
			logger.Warnf("Failed to create bufferedReader: %v. Buffered reading will be disabled for this file handle.", err)
		} else {
			readers = append(readers, bufferedReader)
			if sharedReadState.GetActiveReaderType() == "" {
				sharedReadState.SetActiveReaderType("BufferedReader")
			}
		}
	}

	// Initialize the GCS reader, which is always present.
	gcsReader := clientReaders.NewGCSReader(
		object,
		bucket,
		&clientReaders.GCSReaderConfig{
			MetricHandle:         config.MetricHandle,
			MrdWrapper:           config.MrdWrapper,
			SequentialReadSizeMb: config.SequentialReadSizeMB,
			Config:               config.Config,
		},
	)
	// Add the GCS reader as a fallback.
	readers = append(readers, gcsReader)
	if sharedReadState.GetActiveReaderType() == "" {
		sharedReadState.SetActiveReaderType("GCSReader")
	}

	return &ReadManager{
		object:          object,
		readers:         readers, // Readers are prioritized: file cache first, then GCS.
		sharedReadState: sharedReadState,
		config:          config,
		bucket:          bucket,
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

	// empty read
	if len(p) == 0 {
		return gcsx.ReaderResponse{}, nil
	}

	// Check if we should restart buffered reader based on read pattern
	rr.checkAndRestartBufferedReader()

	var readerResponse gcsx.ReaderResponse
	var err error
	for i, r := range rr.readers {
		readerResponse, err = r.ReadAt(ctx, p, offset)
		if err == nil {
			// Update shared state to track which reader type was successful
			if rr.sharedReadState != nil {
				var readerType string
				switch i {
				case 0:
					if rr.hasFileCacheReader() {
						readerType = "FileCacheReader"
					} else if rr.hasBufferedReader() {
						readerType = "BufferedReader"
					} else {
						readerType = "GCSReader"
					}
				case 1:
					if rr.hasFileCacheReader() && rr.hasBufferedReader() {
						readerType = "BufferedReader"
					} else {
						readerType = "GCSReader"
					}
				case 2:
					readerType = "GCSReader"
				default:
					readerType = "GCSReader"
				}
				rr.sharedReadState.SetActiveReaderType(readerType)
			}
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

// hasFileCacheReader checks if the read manager has a file cache reader
func (rr *ReadManager) hasFileCacheReader() bool {
	if len(rr.readers) == 0 {
		return false
	}
	// File cache reader is always first if present
	_, ok := rr.readers[0].(*gcsx.FileCacheReader)
	return ok
}

// hasBufferedReader checks if the read manager has a buffered reader
func (rr *ReadManager) hasBufferedReader() bool {
	for _, r := range rr.readers {
		if _, ok := r.(*bufferedread.BufferedReader); ok {
			return true
		}
	}
	return false
}

// checkAndRestartBufferedReader checks if the buffered reader should be restarted
// based on read pattern changes and recreates it if necessary
func (rr *ReadManager) checkAndRestartBufferedReader() {
	// Only proceed if we have a shared read state and buffered reading is enabled
	if rr.sharedReadState == nil || !rr.config.Config.Read.EnableBufferedRead {
		return
	}

	// Check if we should restart buffered reader
	if !rr.sharedReadState.ShouldRestartBufferedReader() {
		return
	}

	// Find current buffered reader index
	bufferedReaderIndex := -1
	for i, r := range rr.readers {
		if _, ok := r.(*bufferedread.BufferedReader); ok {
			bufferedReaderIndex = i
			break
		}
	}

	// If no buffered reader exists, try to create one
	if bufferedReaderIndex == -1 {
		rr.recreateBufferedReader()
		return
	}

	// If buffered reader exists and should be restarted, recreate it
	logger.Infof("Restarting buffered reader due to read pattern change to sequential")

	// Destroy the existing buffered reader
	if destroyer, ok := rr.readers[bufferedReaderIndex].(interface{ Destroy() }); ok {
		destroyer.Destroy()
	}

	// Remove the buffered reader from the slice
	rr.readers = append(rr.readers[:bufferedReaderIndex], rr.readers[bufferedReaderIndex+1:]...)

	// Recreate the buffered reader
	rr.recreateBufferedReader()
}

// recreateBufferedReader creates a new buffered reader and inserts it in the appropriate position
func (rr *ReadManager) recreateBufferedReader() {
	readConfig := rr.config.Config.Read
	bufferedReadConfig := &bufferedread.BufferedReadConfig{
		MaxPrefetchBlockCnt:     readConfig.MaxBlocksPerHandle,
		PrefetchBlockSizeBytes:  readConfig.BlockSizeMb * util.MiB,
		InitialPrefetchBlockCnt: readConfig.StartBlocksPerHandle,
		MinBlocksPerHandle:      readConfig.MinBlocksPerHandle,
		RandomSeekThreshold:     readConfig.RandomSeekThreshold,
	}

	bufferedReader, err := bufferedread.NewBufferedReader(
		rr.object,
		rr.bucket,
		bufferedReadConfig,
		rr.config.GlobalMaxBlocksSem,
		rr.config.WorkerPool,
		rr.config.MetricHandle,
	)

	if err != nil {
		logger.Warnf("Failed to recreate bufferedReader: %v. Buffered reading will remain disabled for this file handle.", err)
		return
	}

	// Insert the buffered reader in the correct position
	// It should be after file cache reader (if present) but before GCS reader
	insertIndex := 0
	if rr.hasFileCacheReader() {
		insertIndex = 1
	}

	// Insert the new buffered reader at the correct position
	if insertIndex < len(rr.readers) {
		rr.readers = append(rr.readers[:insertIndex], append([]gcsx.Reader{bufferedReader}, rr.readers[insertIndex:]...)...)
	} else {
		rr.readers = append(rr.readers, bufferedReader)
	}

	logger.Infof("Successfully recreated buffered reader")
}
