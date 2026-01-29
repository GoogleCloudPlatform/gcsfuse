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
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"golang.org/x/sync/errgroup"
)

// HandleSparseRead manages the download and validation of sparse file ranges.
// It checks if the requested range is already downloaded, calculates chunk
// boundaries, downloads missing chunks, and validates the cache hit status.
//
// Returns:
// - cacheHit: true if the requested range is available in cache, false if fallback to GCS is needed
// - error: non-nil if there was an error during the operation
//
// Note: The FileInfo in the cache is automatically updated by downloadSparseRange.
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

	// Calculate the missing ranges to download and filter out in-flight chunks
	chunksToDownload, waitChans, err := job.getChunksToDownload(fileInfo, startOffset, endOffset)
	if err != nil {
		return false, fmt.Errorf("HandleSparseRead: error calculating ranges: %w", err)
	}

	downloadErrGroup, downloadErrGroupCtx := errgroup.WithContext(ctx)
	chunkSize := uint64(job.fileCacheConfig.DownloadChunkSizeMb) * 1024 * 1024

	// Download missing chunks in parallel
	for _, chunkID := range chunksToDownload {
		downloadErrGroup.Go(func() error {
			// Acquire semaphore to limit concurrency across all jobs
			if err := job.maxParallelismSem.Acquire(downloadErrGroupCtx, 1); err != nil {
				return err
			}
			defer job.maxParallelismSem.Release(1)
			start := chunkID * chunkSize
			end := start + chunkSize
			return job.downloadSparseRange(downloadErrGroupCtx, start, end)
		})
	}

	downloadErr := downloadErrGroup.Wait()

	// Cleanup inflight chunks for all download attempts.
	job.mu.Lock()
	for _, chunkID := range chunksToDownload {
		if ch, ok := job.inflightChunks[chunkID]; ok {
			close(ch)
			delete(job.inflightChunks, chunkID)
		}
	}
	job.mu.Unlock()

	// Wait for other inflight chunks
	for _, ch := range waitChans {
		select {
		case <-ch:
		case <-ctx.Done():
			return false, ctx.Err()
		}
	}

	if downloadErr != nil {
		return false, fmt.Errorf("sparse download failed: %w", downloadErr)
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

// getChunksToDownload calculates the missing chunks that need to be downloaded
// and filters out chunks that are already in-flight. It returns a list of
// individual chunks to download and marks them as in-flight.
func (job *Job) getChunksToDownload(fileInfo data.FileInfo, startOffset, endOffset int64) ([]uint64, []chan struct{}, error) {
	if startOffset < 0 || endOffset < 0 || startOffset >= endOffset {
		return nil, nil, fmt.Errorf("invalid offset range: [%d, %d)", startOffset, endOffset)
	}

	// Get missing ranges from ByteRangeMap
	missingChunks := fileInfo.DownloadedRanges.GetMissingChunks(uint64(startOffset), uint64(endOffset))

	var chunksToDownload []uint64
	var waitChans []chan struct{}

	job.mu.Lock()
	defer job.mu.Unlock()

	for _, chunkID := range missingChunks {
		if ch, ok := job.inflightChunks[chunkID]; ok {
			// Chunk is inflight, wait for it
			waitChans = append(waitChans, ch)
		} else {
			// Chunk is not inflight, mark it and add to download list
			ch := make(chan struct{})
			job.inflightChunks[chunkID] = ch

			chunksToDownload = append(chunksToDownload, chunkID)
		}
	}
	return chunksToDownload, waitChans, nil
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

	fileInfo, ok := fileInfoVal.(data.FileInfo)
	if !ok {
		return data.FileInfo{}, fmt.Errorf("getFileInfo: cached value has wrong type")
	}

	return fileInfo, nil
}

// verifySparseRangeDownloaded verifies that the requested range has been
// successfully downloaded by checking the updated FileInfo from the cache.
//
// Returns:
// - cacheHit: whether the range is present in the downloaded ranges
// - err: any error that occurred while fetching the FileInfo
func (job *Job) verifySparseRangeDownloaded(startOffset, endOffset int64) (cacheHit bool, err error) {
	// Fetch updated file info from cache
	fileInfo, err := job.getFileInfo()
	if err != nil {
		return false, fmt.Errorf("verifySparseRangeDownloaded: %w", err)
	}

	// Check if the range is downloaded
	cacheHit = fileInfo.DownloadedRanges != nil && fileInfo.DownloadedRanges.ContainsRange(uint64(startOffset), uint64(endOffset))

	logger.Tracef("Sparse file cache hit check: startOffset=%d, endOffset=%d, DownloadedRanges=%v, cacheHit=%t",
		startOffset, endOffset, fileInfo.DownloadedRanges != nil, cacheHit)

	return cacheHit, nil
}

// downloadSparseRange downloads a specific byte range [start, end) from the GCS object
// for sparse file support. It writes the data to the cache file at the appropriate
// offset.
//
// Acquires and releases LOCK(job.mu)
func (job *Job) downloadSparseRange(ctx context.Context, start, end uint64) error {
	if start >= end {
		return fmt.Errorf("downloadSparseRange: invalid range [%d, %d)", start, end)
	}

	if end > job.object.Size {
		end = job.object.Size
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
		return fmt.Errorf("downloadSparseRange: error creating reader for range [%d, %d): %w", start, end, err)
	}
	defer newReader.Close()

	metrics.CaptureGCSReadMetrics(job.metricsHandle, metrics.ReadTypeNames[metrics.ReadTypeRandom], int64(end-start))

	// Open cache file for writing
	cacheFile, err := os.OpenFile(job.fileSpec.Path, os.O_WRONLY, job.fileSpec.FilePerm)
	if err != nil {
		return fmt.Errorf("downloadSparseRange: error opening cache file: %w", err)
	}
	defer cacheFile.Close()

	// Download from GCS and write to cache file
	offsetWriter := io.NewOffsetWriter(cacheFile, int64(start))
	bytesWritten, err := io.CopyN(offsetWriter, newReader, int64(end-start))
	if err != nil {
		return fmt.Errorf("downloadSparseRange: error copying data: %w", err)
	}

	// Update FileInfo with downloaded range
	job.mu.Lock()
	defer job.mu.Unlock()

	// Re-fetch FileInfo in case it changed
	fileInfo, err := job.getFileInfo()
	if err != nil {
		return fmt.Errorf("downloadSparseRange: %w", err)
	}

	// Add the downloaded range
	bytesAdded := fileInfo.DownloadedRanges.AddRange(start, start+uint64(bytesWritten))

	// Update LRU cache size accounting
	fileInfoKey := data.FileInfoKey{
		BucketName: job.bucket.Name(),
		ObjectName: job.object.Name,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	if err != nil {
		return fmt.Errorf("downloadSparseRange: error creating fileInfoKeyName: %w", err)
	}
	err = job.fileInfoCache.UpdateSize(fileInfoKeyName, bytesAdded)
	if err != nil {
		return fmt.Errorf("downloadSparseRange: error updating cache size: %w", err)
	}

	logger.Tracef("Job:%p (%s:/%s) downloaded range [%d, %d), added %d bytes to sparse file",
		job, job.bucket.Name(), job.object.Name, start, end, bytesAdded)

	return nil
}
