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
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
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

		// Try to open the chunk file directly (no stat() to avoid stale NFS cache and reduce round trips)
		chunkPath := r.manager.GetChunkPath(r.bucket.Name(), r.object.Name, r.object.Generation, chunkIndex)
		chunkFile, openErr := os.Open(chunkPath)
		if openErr != nil {
			if errors.Is(openErr, syscall.ENOENT) {
				// Chunk not in cache - download it
				logger.Tracef("Chunk %d not cached, downloading for %s/%s (offset %d)",
					chunkIndex, r.bucket.Name(), r.object.Name, currentOffset)

				if downloadErr := r.downloadChunk(ctx, chunkIndex, chunkStart, chunkEnd); downloadErr != nil {
					bytesRead = totalRead
					cacheHit = false
					logger.Warnf("DownloadChunk (%d, %d, %d) failed with: %v, read from GCS reader.", chunkIndex, chunkStart, chunkEnd, downloadErr)
					return readResponse, FallbackToAnotherReader
				}
				cacheHit = false // Cache miss - we had to download the chunk

				// Open the newly downloaded chunk, and file should exist.
				chunkFile, openErr = os.Open(chunkPath)
				if openErr != nil {
					bytesRead = totalRead
					logger.Warnf("Failed to open chunk %d after download at path %s: %v, falling back to GCS reader", chunkIndex, chunkPath, openErr)
					return readResponse, FallbackToAnotherReader
				}
			} else {
				// Any other error - fallback to GCS
				bytesRead = totalRead
				cacheHit = false
				logger.Warnf("Failed to open chunk %d at path %s: %v, falling back to GCS reader", chunkIndex, chunkPath, openErr)
				return readResponse, FallbackToAnotherReader
			}
		} else {
			// Cache hit - chunk was already cached
			cacheHit = true
		}
		defer chunkFile.Close()

		// Calculate exact bytes to read for this request within the chunk
		bytesAvailableInChunk := chunkEnd - currentOffset
		bytesToRead := min(int64(bufferRemaining), bytesAvailableInChunk)

		// Read only the required bytes from the chunk file at the specific offset
		offsetInChunk := currentOffset - chunkStart
		n, readErr := chunkFile.ReadAt(p[totalRead:totalRead+int(bytesToRead)], offsetInChunk)
		if (readErr != nil && !errors.Is(readErr, io.EOF)) || n < int(bytesToRead) {
			bytesRead = totalRead
			if n < int(bytesToRead) {
				logger.Warnf("Read only %d bytes from chunk %d at path %s, expected %d bytes. falling back to GCS reader.", n, chunkIndex, chunkPath, bytesToRead)
			} else {
				logger.Warnf("Failed to read chunk %d at path %s: %v, falling back to GCS reader", chunkIndex, chunkPath, readErr)
			}
			return readResponse, FallbackToAnotherReader
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
// This method handles concurrent access and LRU cache eviction race conditions.
// If any cache operation fails, we fallback to reading directly from GCS without caching.
func (r *SharedChunkCacheReader) downloadChunk(ctx context.Context, chunkIndex, chunkStart, chunkEnd int64) error {
	objDir := r.manager.GetObjectDir(r.bucket.Name(), r.object.Name, r.object.Generation)
	tmpPath := r.manager.GenerateTmpPath(r.bucket.Name(), r.object.Name, r.object.Generation, chunkIndex)

	// Step 1: Create object directory
	// Protects against concurrent LRU cache eviction that may have deleted the directory.
	// - EEXIST: Ignore, directory already exists (expected)
	// - Any other error: Fallback to GCS reader
	if err := os.MkdirAll(objDir, r.manager.GetDirPerm()); err != nil {
		if !errors.Is(err, syscall.EEXIST) {
			return fmt.Errorf("MkDirAll failed: %w", err)
		}
	}

	// Step 2: Create temporary file with O_EXCL (purely for defensive purposes to handle conflicting
	// temporary downloads of the same chunk), given odds of collision are essentially zero with 64-bit
	// hash-prefix.
	// - ENOENT: Directory was deleted (LRU race), retry once by recreating directory
	// - Any other error, including EEXIST (chunk path collision): Fallback to GCS reader
	tmpFile, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, r.manager.GetFilePerm())
	if err != nil {
		if errors.Is(err, syscall.ENOENT) {
			// Directory was deleted by LRU, retry once
			if mkdirErr := os.MkdirAll(objDir, r.manager.GetDirPerm()); mkdirErr != nil && !errors.Is(mkdirErr, syscall.EEXIST) {
				return fmt.Errorf("MkDirAll retry failed: %w", mkdirErr)
			}
			// Retry creating temp file
			tmpFile, err = os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, r.manager.GetFilePerm())
			if err != nil {
				return fmt.Errorf("retry to create tmp file failed: %w", err)
			}
		} else {
			return fmt.Errorf("create temp file failed: %w", err)
		}
	}

	defer func() {
		if tmpFile != nil {
			tmpFile.Close()
		}
	}()

	// Step 3: Create GCS reader for the specific byte range
	// Here, generation is important given we store the file under a hash that includes generation.
	readReq := &gcs.ReadObjectRequest{
		Name:       r.object.Name,
		Generation: r.object.Generation,
		Range: &gcs.ByteRange{
			Start: uint64(chunkStart),
			Limit: uint64(chunkEnd),
		},
	}
	reader, err := r.bucket.NewReaderWithReadHandle(ctx, readReq)
	if err != nil {
		os.Remove(tmpPath) // Cleanup
		return fmt.Errorf("failed to create GCS reader: %w", err)
	}
	defer reader.Close()

	// Step 4: Copy data from GCS to temp file
	bytesWritten, err := io.Copy(tmpFile, reader)
	if err != nil || bytesWritten != (chunkEnd-chunkStart) {
		os.Remove(tmpPath) // Cleanup
		if err != nil {
			return fmt.Errorf("failed to copy data to temp file: %w", err)
		}
		return fmt.Errorf("incomplete copy, expected %d bytes but wrote %d bytes", (chunkEnd - chunkStart), bytesWritten)
	}

	// Close implies Sync() on NFS (intended use case).
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath) // Cleanup
		return fmt.Errorf("failed to close tmpFile: %v", err)
	}
	tmpFile = nil // Avoid deferred close since file is already closed

	// Step 5: Atomically rename temp file to final location
	// Protects against concurrent downloads of the same chunk.
	chunkPath := r.manager.GetChunkPath(r.bucket.Name(), r.object.Name, r.object.Generation, chunkIndex)
	if err := os.Rename(tmpPath, chunkPath); err != nil {
		os.Remove(tmpPath) // Cleanup
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	logger.Tracef("Downloaded and cached chunk %d (range %d-%d, %d bytes)",
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
