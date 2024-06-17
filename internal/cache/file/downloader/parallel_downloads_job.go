// Copyright 2024 Google Inc. All Rights Reserved.
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
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/lru"
	cacheutil "github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/monitor"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

// downloadRangeToFile is a helper function to download a given range of object
// from GCS into given file handle.
//
// This function doesn't take locks and can be executed parallely.
func (job *Job) downloadRangeToFile(ctx context.Context, cacheFile *os.File, start, end int64) error {
	newReader, err := job.bucket.NewReader(
		ctx,
		&gcs.ReadObjectRequest{
			Name:       job.object.Name,
			Generation: job.object.Generation,
			Range: &gcs.ByteRange{
				Start: uint64(start),
				Limit: uint64(end),
			},
			ReadCompressed: job.object.HasContentEncodingGzip(),
		})
	if err != nil {
		err = fmt.Errorf("downloadRangeToFile: error in creating NewReader with start %d and limit %d: %w", start, end, err)
		return err
	}
	defer func() {
		closeErr := newReader.Close()
		if closeErr != nil {
			logger.Errorf("Job:%p (%s:/%s) error while closing reader: %v", job, job.bucket.Name(), job.object.Name, closeErr)
		}
	}()

	monitor.CaptureGCSReadMetrics(ctx, util.Parallel, end-start)

	// Copy the contents from NewReader to cache file at appropriate offset.
	offsetWriter := io.NewOffsetWriter(cacheFile, start)
	_, err = io.CopyN(offsetWriter, newReader, end-start)
	if err != nil {
		err = fmt.Errorf("downloadRangeToFile: error at the time of copying content to cache file %w", err)
	}
	return err
}

// parallelDownloadObjectAsync does parallel download of the backing GCS object
// into a file as part of file cache using multiple NewReader method of
// gcs.Bucket running in parallel
//
// Note: There can only be one async parallel or non-parallel download running
// for a job at a time.
// Acquires and releases LOCK(job.mu)
func (job *Job) parallelDownloadObjectAsync(maxTotalConcurrency *semaphore.Weighted) {
	// Cleanup the async job in all cases - completion/failure/invalidation.
	defer job.cleanUpDownloadAsyncJob()

	// Create, and open cache file for writing object into it.
	cacheFile, err := cacheutil.CreateFile(job.fileSpec, os.O_TRUNC|os.O_RDWR)
	if err != nil {
		err = fmt.Errorf("parallelDownloadObjectAsync: error in creating cache file: %w", err)
		job.failWhileDownloading(err)
		return
	}
	defer func() {
		err = cacheFile.Close()
		if err != nil {
			err = fmt.Errorf("parallelDownloadObjectAsync: error while closing cache file: %w", err)
			job.failWhileDownloading(err)
		}
	}()

	notifyInvalid := func() {
		job.mu.Lock()
		job.status.Name = Invalid
		job.notifySubscribers()
		job.mu.Unlock()
	}

	handleError := func(err error) {
		if errors.Is(err, context.Canceled) {
			notifyInvalid()
			return
		}
		job.failWhileDownloading(err)
	}

	var start, end int64
	end = int64(job.object.Size)
	var parallelReadRequestSize = int64(job.fileCacheConfig.ReadRequestSizeMB) * cacheutil.MiB
	var maxSingleStepReadSize = parallelReadRequestSize * int64(job.fileCacheConfig.DownloadParallelismPerFile)

	for {
		select {
		case <-job.cancelCtx.Done():
			return
		default:
			if start < end {
				// Parallely download different ranges not more than maxSingleStepReadSize
				// and not using go routines more than job.downloadParallelism
				downloadErrGroup, downloadErrGroupCtx := errgroup.WithContext(job.cancelCtx)
				var singleStepReadSize int64 = 0
				for goRoutineIdx := 0; (goRoutineIdx < job.fileCacheConfig.DownloadParallelismPerFile) &&
					(singleStepReadSize < maxSingleStepReadSize) && (start < end); goRoutineIdx++ {
					rangeStart := start
					rangeEnd := min(rangeStart+parallelReadRequestSize, end)
					if goRoutineIdx == 0 {
						if err = maxTotalConcurrency.Acquire(downloadErrGroupCtx, 1); err != nil {
							logger.Tracef("Error while acquiring semaphore resource: %v", err)
							handleError(err)
							return
						}
					} else if s := maxTotalConcurrency.TryAcquire(1); !s {
						break
					}
					downloadErrGroup.Go(func() error {
						defer maxTotalConcurrency.Release(1)
						return job.downloadRangeToFile(downloadErrGroupCtx, cacheFile, rangeStart, rangeEnd)
					})

					singleStepReadSize = singleStepReadSize + rangeEnd - rangeStart
					start = rangeEnd
				}
				// If any of the go routine failed, consider the async job failed.
				err = downloadErrGroup.Wait()
				if err != nil {
					handleError(err)
					return
				}

				job.mu.Lock()
				job.status.Offset = start
				err = job.updateFileInfoCache()
				// Notify subscribers if file cache is updated.
				if err == nil {
					job.notifySubscribers()
				} else if strings.Contains(err.Error(), lru.EntryNotExistErrMsg) {
					// Download job expects entry in file info cache for the file it is
					// downloading. If the entry is deleted in between which is expected
					// to happen at the time of eviction, then the job should be
					// marked Invalid instead of Failed.
					job.status.Name = Invalid
					job.notifySubscribers()
					logger.Tracef("Job:%p (%s:/%s) is no longer valid due to absense of entry in file info cache.", job, job.bucket.Name(), job.object.Name)
					job.mu.Unlock()
					return
				}
				job.mu.Unlock()
				// Change status of job in case of error while updating file cache.
				if err != nil {
					job.failWhileDownloading(err)
					return
				}
			} else {
				err = job.validateCRC()
				if err != nil {
					job.failWhileDownloading(err)
					return
				}

				job.mu.Lock()
				job.status.Name = Completed
				job.notifySubscribers()
				job.mu.Unlock()
				return
			}
		}
	}
}
