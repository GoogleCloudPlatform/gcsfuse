// Copyright 2024 Google LLC
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

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

// SparseReadResult contains the result of a sparse file read operation.
type SparseReadResult struct {
	CacheHit         bool
	NeedsFallbackGCS bool
}

// HandleSparseRead manages the download and validation of sparse file ranges.
// It checks if the requested range is already downloaded, calculates chunk
// boundaries, downloads missing chunks, and validates the cache hit status.
//
// Returns a SparseReadResult containing:
// - CacheHit: whether the requested range is available in cache
// - NeedsFallbackGCS: whether the operation should fallback to GCS
//
// Note: The FileInfo in the cache is automatically updated by DownloadRange.
func (job *Job) HandleSparseRead(ctx context.Context, offset, requiredOffset int64) (SparseReadResult, error) {
	// Get current file info from cache
	fileInfo, err := job.getFileInfo()
	if err != nil {
		return SparseReadResult{
			CacheHit:         false,
			NeedsFallbackGCS: true,
		}, fmt.Errorf("HandleSparseRead: error getting file info: %w", err)
	}

	// Check if the requested range is already downloaded
	if fileInfo.DownloadedRanges.ContainsRange(uint64(offset), uint64(requiredOffset)) {
		// Range already downloaded, return cache hit
		return SparseReadResult{
			CacheHit:         true,
			NeedsFallbackGCS: false,
		}, nil
	}

	// Calculate the chunk boundaries to download
	chunkStart, chunkEnd, err := job.calculateSparseChunkBoundaries(offset, requiredOffset)
	if err != nil {
		return SparseReadResult{
			CacheHit:         false,
			NeedsFallbackGCS: true,
		}, fmt.Errorf("HandleSparseRead: error calculating chunk boundaries: %w", err)
	}

	// Download the chunk
	if err := job.DownloadRange(ctx, chunkStart, chunkEnd); err != nil {
		logger.Infof("Sparse file download failed for range [%d, %d): %v. Falling back to GCS for this read.", chunkStart, chunkEnd, err)
		return SparseReadResult{
			CacheHit:         false,
			NeedsFallbackGCS: true,
		}, nil
	}

	// Verify the download was successful
	cacheHit, err := job.verifySparseRangeDownloaded(offset, requiredOffset)
	if err != nil {
		logger.Infof("Error verifying sparse file download: %v. Falling back to GCS for this read.", err)
		return SparseReadResult{
			CacheHit:         false,
			NeedsFallbackGCS: true,
		}, nil
	}

	if !cacheHit {
		logger.Errorf("Sparse file cache misses even after a seemingly successful download: offset=%d, requiredOffset=%d",
			offset, requiredOffset)
		return SparseReadResult{
			CacheHit:         false,
			NeedsFallbackGCS: true,
		}, nil
	}

	return SparseReadResult{
		CacheHit:         true,
		NeedsFallbackGCS: false,
	}, nil
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
func (job *Job) calculateSparseChunkBoundaries(offset, requiredOffset int64) (chunkStart, chunkEnd uint64, err error) {
	if offset < 0 || requiredOffset < 0 || offset >= requiredOffset {
		return 0, 0, fmt.Errorf("invalid offset range: [%d, %d)", offset, requiredOffset)
	}

	// Get the configured chunk size from fileCacheConfig
	chunkSize := uint64(job.fileCacheConfig.DownloadChunkSizeMb) * 1024 * 1024

	// Align chunk start and end to chunk boundaries
	// Start: round down to chunk boundary
	chunkStart = (uint64(offset) / chunkSize) * chunkSize

	// End: round up requiredOffset to chunk boundary to ensure full coverage
	chunkEnd = ((uint64(requiredOffset) + chunkSize - 1) / chunkSize) * chunkSize

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
func (job *Job) verifySparseRangeDownloaded(offset, requiredOffset int64) (cacheHit bool, err error) {
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
	cacheHit = fileInfo.DownloadedRanges != nil && fileInfo.DownloadedRanges.ContainsRange(uint64(offset), uint64(requiredOffset))

	logger.Tracef("Sparse file cache hit check: offset=%d, requiredOffset=%d, DownloadedRanges=%v, cacheHit=%t",
		offset, requiredOffset, fileInfo.DownloadedRanges != nil, cacheHit)

	return cacheHit, nil
}

// IsSparseFile returns true if the object is configured for sparse file mode.
func (job *Job) IsSparseFile(bucket gcs.Bucket, object *gcs.MinObject) (bool, error) {
	fileInfoKey := data.FileInfoKey{
		BucketName: bucket.Name(),
		ObjectName: object.Name,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	if err != nil {
		return false, fmt.Errorf("error creating fileInfoKeyName: %w", err)
	}

	fileInfoVal := job.fileInfoCache.LookUpWithoutChangingOrder(fileInfoKeyName)
	if fileInfoVal == nil {
		return false, nil
	}

	fileInfo := fileInfoVal.(data.FileInfo)
	return fileInfo.SparseMode, nil
}
