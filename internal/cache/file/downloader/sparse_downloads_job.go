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

package downloader

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

// HandleSparseRead manages the download and validation of sparse file ranges.
// It checks if the requested range is already downloaded, calculates chunk
// boundaries, downloads missing chunks, and validates the cache hit status.
//
// Returns:
// - cacheHit: true if the requested range is available in cache, false if fallback to GCS is needed
// - error: non-nil if there was an error during the operation
//
// Note: The FileInfo in the cache is automatically updated by DownloadRange.
func (job *Job) HandleSparseRead(ctx context.Context, startOffset, endOffset int64) (cacheHit bool, err error) {
	// Get current file info from cache
	fileInfo, err := job.getFileInfo()
	if err != nil {
		return false, fmt.Errorf("HandleSparseRead: error getting file info: %w", err)
	}

	// Check if the requested range is already downloaded
	if fileInfo.DownloadedRanges.ContainsRange(uint64(startOffset), uint64(endOffset)) {
		// Range already downloaded, return cache hit
		return true, nil
	}

	// Calculate the chunk boundaries to download
	chunkStart, chunkEnd, err := job.calculateSparseChunkBoundaries(startOffset, endOffset)
	if err != nil {
		return false, fmt.Errorf("HandleSparseRead: error calculating chunk boundaries: %w", err)
	}

	// Download the chunk
	if err := job.DownloadRange(ctx, chunkStart, chunkEnd); err != nil {
		return false, fmt.Errorf("download failed for range [%d, %d): %w", chunkStart, chunkEnd, err)
	}

	// Verify the download was successful
	cacheHit, err = job.verifySparseRangeDownloaded(startOffset, endOffset)
	if err != nil {
		return false, fmt.Errorf("error verifying download: %w", err)
	}

	if !cacheHit {
		return false, fmt.Errorf("cache miss after download: range [%d, %d) not found in downloaded ranges", startOffset, endOffset)
	}

	return true, nil
}

// getFileInfo retrieves the FileInfo from cache for this job's object.
func (job *Job) getFileInfo() (data.FileInfo, error) {
	fileInfoKey := data.FileInfoKey{
		BucketName: job.bucket.Name(),
		ObjectName: job.object.Name,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	if err != nil {
		return data.FileInfo{}, fmt.Errorf("error creating fileInfoKeyName: %w", err)
	}

	fileInfoVal := job.fileInfoCache.LookUpWithoutChangingOrder(fileInfoKeyName)
	if fileInfoVal == nil {
		return data.FileInfo{}, fmt.Errorf("file info not found in cache")
	}

	return fileInfoVal.(data.FileInfo), nil
}

// calculateSparseChunkBoundaries calculates the aligned chunk boundaries
// for a sparse file download based on the configured chunk size.
//
// The chunk boundaries are aligned to ensure efficient downloads:
// - chunkStart is rounded down to the nearest chunk boundary
// - chunkEnd is rounded up to the nearest chunk boundary
// - chunkEnd is capped at the object size
//
// Uses DownloadChunkSizeMb from fileCacheConfig (default 200MB) for chunk size.
func (job *Job) calculateSparseChunkBoundaries(startOffset, endOffset int64) (chunkStart, chunkEnd uint64, err error) {
	if startOffset < 0 || endOffset < 0 || startOffset >= endOffset {
		return 0, 0, fmt.Errorf("invalid offset range: [%d, %d)", startOffset, endOffset)
	}

	// Get the configured chunk size from fileCacheConfig
	chunkSize := uint64(job.fileCacheConfig.DownloadChunkSizeMb) * 1024 * 1024

	// Align chunk start and end to chunk boundaries
	// Start: round down to chunk boundary
	chunkStart = (uint64(startOffset) / chunkSize) * chunkSize

	// End: round up endOffset to chunk boundary to ensure full coverage
	chunkEnd = ((uint64(endOffset) + chunkSize - 1) / chunkSize) * chunkSize

	// Cap at object size
	if chunkEnd > job.object.Size {
		chunkEnd = job.object.Size
	}

	return chunkStart, chunkEnd, nil
}

// verifySparseRangeDownloaded verifies that the requested range has been
// successfully downloaded by checking the updated FileInfo from the cache.
//
// Returns:
// - cacheHit: whether the range is present in the downloaded ranges
// - err: any error that occurred while fetching the FileInfo
func (job *Job) verifySparseRangeDownloaded(startOffset, endOffset int64) (cacheHit bool, err error) {
	// Create file info key
	fileInfoKey := data.FileInfoKey{
		BucketName: job.bucket.Name(),
		ObjectName: job.object.Name,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	if err != nil {
		return false, fmt.Errorf("error creating fileInfoKeyName: %w", err)
	}

	// Fetch updated file info from cache
	fileInfoVal := job.fileInfoCache.LookUpWithoutChangingOrder(fileInfoKeyName)
	if fileInfoVal == nil {
		return false, fmt.Errorf("file info not found in cache after download")
	}
	fileInfo := fileInfoVal.(data.FileInfo)

	// Check if the range is downloaded
	cacheHit = fileInfo.DownloadedRanges != nil && fileInfo.DownloadedRanges.ContainsRange(uint64(startOffset), uint64(endOffset))

	logger.Tracef("Sparse file cache hit check: startOffset=%d, endOffset=%d, DownloadedRanges=%v, cacheHit=%t",
		startOffset, endOffset, fileInfo.DownloadedRanges != nil, cacheHit)

	return cacheHit, nil
}

// DownloadRange downloads a specific byte range [start, end) from the GCS object
// for sparse file support. It writes the data to the cache file at the appropriate
// offset.
//
// Acquires and releases LOCK(job.mu)
func (job *Job) DownloadRange(ctx context.Context, start, end uint64) error {
	if start >= end {
		return fmt.Errorf("DownloadRange: invalid range [%d, %d)", start, end)
	}

	if end > job.object.Size {
		end = job.object.Size
	}

	// Check if this is a sparse file
	fileInfoKey := data.FileInfoKey{
		BucketName: job.bucket.Name(),
		ObjectName: job.object.Name,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	if err != nil {
		return fmt.Errorf("DownloadRange: error creating fileInfoKeyName: %w", err)
	}

	// Create GCS reader for the specific range
	newReader, err := job.bucket.NewReaderWithReadHandle(
		ctx,
		&gcs.ReadObjectRequest{
			Name:       job.object.Name,
			Generation: job.object.Generation,
			Range: &gcs.ByteRange{
				Start: start,
				Limit: end,
			},
			ReadCompressed: job.object.HasContentEncodingGzip(),
			ReadHandle:     nil,
		})
	if err != nil {
		return fmt.Errorf("DownloadRange: error creating reader for range [%d, %d): %w", start, end, err)
	}
	defer newReader.Close()

	// Open cache file for writing
	cacheFile, err := os.OpenFile(job.fileSpec.Path, os.O_WRONLY, job.fileSpec.FilePerm)
	if err != nil {
		return fmt.Errorf("DownloadRange: error opening cache file: %w", err)
	}
	defer cacheFile.Close()

	// Download from GCS and write to cache file
	offsetWriter := io.NewOffsetWriter(cacheFile, int64(start))
	bytesWritten, err := io.CopyN(offsetWriter, newReader, int64(end-start))

	// Update FileInfo with downloaded range
	job.mu.Lock()
	defer job.mu.Unlock()

	// Re-fetch FileInfo in case it changed
	fileInfoVal := job.fileInfoCache.LookUpWithoutChangingOrder(fileInfoKeyName)
	if fileInfoVal == nil {
		return fmt.Errorf("DownloadRange: file info not found in cache after download")
	}
	fileInfo := fileInfoVal.(data.FileInfo)

	// Add the downloaded range
	bytesAdded := fileInfo.DownloadedRanges.AddRange(start, start+uint64(bytesWritten))

	// Update LRU cache size accounting
	err = job.fileInfoCache.UpdateSize(fileInfoKeyName, bytesAdded)
	if err != nil {
		return fmt.Errorf("DownloadRange: error updating cache size: %w", err)
	}

	logger.Tracef("Job:%p (%s:/%s) downloaded range [%d, %d), added %d bytes to sparse file",
		job, job.bucket.Name(), job.object.Name, start, end, bytesAdded)

	return nil
}
