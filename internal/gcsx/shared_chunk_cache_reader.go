// Copyright 2026 Google LLC
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
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/file"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/googlecloudplatform/gcsfuse/v3/tracing"
	"github.com/jacobsa/fuse/fuseops"
)

// SharedChunkCacheReader implements on-demand chunk-based reading without prefetching for
// shared cache. It downloads only the chunks needed for a read operation, no prefetching.
type SharedChunkCacheReader struct {
	manager      *file.SharedChunkCacheManager
	bucket       gcs.Bucket
	object       *gcs.MinObject
	metricHandle metrics.MetricHandle
	traceHandle  tracing.TraceHandle
	handleID     fuseops.HandleID
	retryConfig  file.RetryConfig
}

// NewSharedChunkCacheReader creates a new chunk-based reader for shared cache.
func NewSharedChunkCacheReader(
	manager *file.SharedChunkCacheManager,
	bucket gcs.Bucket,
	object *gcs.MinObject,
	metricHandle metrics.MetricHandle,
	traceHandle tracing.TraceHandle,
	handleID fuseops.HandleID,
) *SharedChunkCacheReader {
	return &SharedChunkCacheReader{
		manager:      manager,
		bucket:       bucket,
		object:       object,
		metricHandle: metricHandle,
		traceHandle:  traceHandle,
		handleID:     handleID,
		retryConfig:  file.DefaultRetryConfig(),
	}
}

// ReadAt reads data at the specified offset, downloading chunks on-demand.
// Implements the Reader interface.
func (r *SharedChunkCacheReader) ReadAt(ctx context.Context, req *ReadRequest) (ReadResponse, error) {
	var readResponse ReadResponse

	// Check if file should be excluded from cache based on regex
	if r.manager.ShouldExcludeFromCache(r.bucket, r.object) {
		return readResponse, FallbackToAnotherReader
	}

	offset := req.Offset
	p := req.Buffer

	if offset >= int64(r.object.Size) {
		return readResponse, io.EOF
	}

	if offset < 0 {
		return readResponse, fmt.Errorf("negative offset: %d", offset)
	}

	requestID := uuid.New()
	logger.Tracef("%.13v <- SharedChunkCache(%s:/%s, offset: %d, size: %d handle: %d)",
		requestID, r.bucket.Name(), r.object.Name, offset, len(p), r.handleID)

	startTime := time.Now()
	var bytesRead int
	var cacheHit bool
	var err error

	defer func() {
		executionTime := time.Since(startTime)
		var requestOutput string
		if err != nil {
			requestOutput = fmt.Sprintf("err: %v (%v)", err, executionTime)
		} else {
			requestOutput = fmt.Sprintf("OK (cacheHit: %t, bytes: %d) (%v)", cacheHit, bytesRead, executionTime)
		}

		logger.Tracef("%.13v -> %s", requestID, requestOutput)

		// Capture metrics
		readType := metrics.ReadTypeRandom
		if offset == 0 {
			readType = metrics.ReadTypeSequential
		}
		r.metricHandle.FileCacheReadCount(1, cacheHit, metrics.ReadTypeNames[readType])
		r.metricHandle.FileCacheReadBytesCount(int64(bytesRead), metrics.ReadTypeNames[readType])
		r.metricHandle.FileCacheReadLatencies(ctx, executionTime, cacheHit)
	}()

	totalRead := 0
	bufferRemaining := len(p)
	currentOffset := offset

	// Read across chunk boundaries if necessary
	for bufferRemaining > 0 && currentOffset < int64(r.object.Size) {
		// Calculate which chunk contains this offset
		chunkIndex := r.manager.GetChunkIndex(currentOffset)
		chunkStart := chunkIndex * r.manager.GetChunkSize()
		chunkEnd := min(chunkStart+r.manager.GetChunkSize(), int64(r.object.Size))

		offsetInChunk := currentOffset - chunkStart
		bytesAvailableInChunk := chunkEnd - currentOffset

		// Calculate exact bytes to read for this request
		bytesToRead := min(int64(bufferRemaining), bytesAvailableInChunk)

		// Check if chunk is cached
		chunkPath := r.manager.GetChunkPath(r.bucket.Name(), r.object.Name, r.object.Generation, chunkIndex)
		_, statErr := os.Stat(chunkPath)

		if statErr != nil {
			if os.IsNotExist(statErr) {
				// Chunk not in cache - download it
				logger.Infof("Chunk %d not cached, downloading for %s/%s (offset %d)",
					chunkIndex, r.bucket.Name(), r.object.Name, currentOffset)

				if downloadErr := r.downloadChunk(ctx, chunkIndex, chunkStart, chunkEnd); downloadErr != nil {
					bytesRead = totalRead
					cacheHit = false
					return readResponse, fmt.Errorf("failed to download chunk %d: %w", chunkIndex, downloadErr)
				}
				cacheHit = false // Cache miss - we just downloaded it
			} else {
				bytesRead = totalRead
				cacheHit = false
				return readResponse, fmt.Errorf("failed to stat chunk %d: %w", chunkIndex, statErr)
			}
		} else {
			// Cache hit - chunk was already cached
			cacheHit = true
		}

		// Open chunk file and read only the exact bytes needed
		chunkFile, openErr := os.Open(chunkPath)
		if openErr != nil {
			bytesRead = totalRead
			return readResponse, fmt.Errorf("failed to open chunk %d: %w", chunkIndex, openErr)
		}
		defer chunkFile.Close()

		// Read only the required bytes from the chunk file at the specific offset
		n, readErr := chunkFile.ReadAt(p[totalRead:totalRead+int(bytesToRead)], offsetInChunk)

		if readErr != nil && readErr != io.EOF {
			bytesRead = totalRead
			return readResponse, fmt.Errorf("failed to read from chunk %d at offset %d: %w", chunkIndex, offsetInChunk, readErr)
		}

		totalRead += n
		currentOffset += int64(n)
		bufferRemaining -= n
	}

	bytesRead = totalRead
	if totalRead == 0 && currentOffset >= int64(r.object.Size) {
		return readResponse, io.EOF
	}

	readResponse.Size = totalRead
	return readResponse, nil
}

// downloadChunk downloads a specific chunk from GCS and caches it atomically.
func (r *SharedChunkCacheReader) downloadChunk(ctx context.Context, chunkIndex, chunkStart, chunkEnd int64) error {
	chunkPath := r.manager.GetChunkPath(r.bucket.Name(), r.object.Name, r.object.Generation, chunkIndex)

	// Check again if chunk exists (another process might have downloaded it)
	if _, err := os.Stat(chunkPath); err == nil {
		logger.Tracef("Chunk %d already cached by another process", chunkIndex)
		return nil
	}

	// Create object directory and temp file with retry (handles LRU deletion race)
	objDir := r.manager.GetObjectDir(r.bucket.Name(), r.object.Name, r.object.Generation)
	tmpPath := r.manager.GenerateTmpPath(r.bucket.Name(), r.object.Name, r.object.Generation, chunkIndex)
	var tmpFile *os.File
	var bytesWritten int64

	err := file.RetryOnNFSError("download_and_write_chunk", func() error {
		// Ensure directory exists (handles LRU deletion race)
		if mkdirErr := os.MkdirAll(objDir, r.manager.GetDirPerm()); mkdirErr != nil {
			return mkdirErr
		}

		// Create/truncate temp file - O_TRUNC ensures clean slate on retry
		var openErr error
		tmpFile, openErr = os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, r.manager.GetFilePerm())
		if openErr != nil {
			return openErr
		}
		defer tmpFile.Close()

		// Read chunk from GCS
		readReq := &gcs.ReadObjectRequest{
			Name:       r.object.Name,
			Generation: r.object.Generation,
			Range: &gcs.ByteRange{
				Start: uint64(chunkStart),
				Limit: uint64(chunkEnd),
			},
		}

		reader, readErr := r.bucket.NewReaderWithReadHandle(ctx, readReq)
		if readErr != nil {
			return fmt.Errorf("failed to create GCS reader: %w", readErr)
		}
		defer reader.Close()

		// Download data from GCS to temp file
		var copyErr error
		bytesWritten, copyErr = io.Copy(tmpFile, reader)
		if copyErr != nil {
			return copyErr
		}

		// Sync to ensure data is written to disk before rename
		return tmpFile.Sync()
	}, r.retryConfig)

	if err != nil {
		os.Remove(tmpPath) // Best effort cleanup
		return fmt.Errorf("failed to download chunk data: %w", err)
	}

	// Atomically move chunk to final location with retry
	err = file.RetryOnNFSError("rename_to_final_path", func() error {
		return os.Rename(tmpPath, chunkPath)
	}, r.retryConfig)
	if err != nil {
		os.Remove(tmpPath) // Cleanup failed temp file
		if os.IsExist(err) {
			// Another process won the race - that's fine
			logger.Tracef("Chunk %d already cached by another process (race)", chunkIndex)
			return nil
		}
		return fmt.Errorf("failed to move chunk to cache: %w", err)
	}

	logger.Infof("Downloaded and cached chunk %d (range %d-%d, %d bytes)",
		chunkIndex, chunkStart, chunkEnd, bytesWritten)

	return nil
}

// CheckInvariants implements the Reader interface.
func (r *SharedChunkCacheReader) CheckInvariants() {
}

// Destroy implements the Reader interface cleanup.
func (r *SharedChunkCacheReader) Destroy() {
	// No resources to clean up for chunk-based reader
}
