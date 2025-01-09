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
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/data"
	cacheutil "github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"golang.org/x/sync/errgroup"
)

// downloadRange is a helper function to download a given range of object from
// GCS into given destination writer.
//
// This function doesn't take locks and can be executed parallely.
func (job *Job) downloadRange(ctx context.Context, dstWriter io.Writer, start, end int64) error {
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
		err = fmt.Errorf("downloadRange: error in creating NewReader with start %d and limit %d: %w", start, end, err)
		return err
	}
	defer func() {
		// Reader is closed after the data has been read and the error from closure
		// is not reported as failure of async job, similar to how it's done for
		// foreground reads: https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/internal/gcsx/random_reader.go#L298.
		closeErr := newReader.Close()
		if closeErr != nil {
			logger.Warnf("Job:%p (%s:/%s) error while closing reader: %v", job, job.bucket.Name(), job.object.Name, closeErr)
		}
	}()

	common.CaptureGCSReadMetrics(ctx, job.metricsHandle, util.Parallel, end-start)

	// Use standard copy function if O_DIRECT is disabled and memory aligned
	// buffer otherwise.
	if !job.fileCacheConfig.EnableODirect {
		_, err = io.CopyN(dstWriter, newReader, end-start)
	} else {
		_, err = cacheutil.CopyUsingMemoryAlignedBuffer(ctx, newReader, dstWriter, end-start,
			job.fileCacheConfig.WriteBufferSize)
		// If context is canceled while reading/writing in CopyUsingMemoryAlignedBuffer
		// then it returns error different from context cancelled (invalid argument),
		// and we need to report that error as context cancelled.
		if !errors.Is(err, context.Canceled) && errors.Is(ctx.Err(), context.Canceled) {
			err = errors.Join(err, ctx.Err())
		}
	}

	if err != nil {
		err = fmt.Errorf("downloadRange: error at the time of copying content to cache file %w", err)
	}
	return err
}

// RangeMap maintains the ranges downloaded by the different goroutines. This
// function takes a new range and merges with existing ranges if they are continuous.
// If the start offset is 0, updates the job's status offset.
//
// Eg:
// Input: rangeMap entries 0-3, 5-6. New input 7-8.
// Output: rangeMap entries 0-3, 5-8.
func (job *Job) updateRangeMap(rangeMap map[int64]int64, offsetStart int64, offsetEnd int64) error {
	// Check if the chunk downloaded completes a range [0, R) and find that
	// R.
	job.mu.Lock()
	defer job.mu.Unlock()

	finalStart := offsetStart
	finalEnd := offsetEnd

	if offsetStart != 0 {
		leftStart, exist := rangeMap[offsetStart]
		if exist {
			finalStart = leftStart
			delete(rangeMap, offsetStart)
			delete(rangeMap, leftStart)
		}
	}

	rightEnd, exist := rangeMap[offsetEnd]
	if exist {
		finalEnd = rightEnd
		delete(rangeMap, offsetEnd)
		delete(rangeMap, rightEnd)
	}

	rangeMap[finalStart] = finalEnd
	rangeMap[finalEnd] = finalStart

	if finalStart == 0 {
		if updateErr := job.updateStatusOffset(finalEnd); updateErr != nil {
			return updateErr
		}
	}

	return nil
}

// Reads the range input from the range channel continuously and downloads that
// range from the GCS. If the range channel is closed, it will exit.
func (job *Job) downloadOffsets(ctx context.Context, goroutineIndex int64, cacheFile *os.File, rangeMap map[int64]int64) func() error {
	return func() error {
		// Since we keep a goroutine for each job irrespective of the maxParallelism,
		// not releasing the default goroutine to the pool.
		if goroutineIndex > 0 {
			defer job.maxParallelismSem.Release(1)
		}

		for {
			// Read the offset to be downloaded from the channel.
			objectRange, ok := <-job.rangeChan
			if !ok {
				// In case channel is closed return.
				return nil
			}

			offsetWriter := io.NewOffsetWriter(cacheFile, int64(objectRange.Start))
			err := job.downloadRange(ctx, offsetWriter, objectRange.Start, objectRange.End)
			if err != nil {
				return err
			}

			err = job.updateRangeMap(rangeMap, objectRange.Start, objectRange.End)
			if err != nil {
				return err
			}
		}
	}
}

// parallelDownloadObjectToFile does parallel download of the backing GCS object
// into given file handle using multiple NewReader method of gcs.Bucket running
// in parallel. This function is canceled if job.cancelCtx is canceled.
func (job *Job) parallelDownloadObjectToFile(cacheFile *os.File) (err error) {
	rangeMap := make(map[int64]int64)
	// Trying to keep the channel size greater than ParallelDownloadsPerFile to ensure
	// that there is no goroutine waiting for data(nextRange) to be published to channel.
	job.rangeChan = make(chan data.ObjectRange, 2*job.fileCacheConfig.ParallelDownloadsPerFile)
	var numGoRoutines int64
	var start int64
	downloadChunkSize := job.fileCacheConfig.DownloadChunkSizeMb * cacheutil.MiB
	downloadErrGroup, downloadErrGroupCtx := errgroup.WithContext(job.cancelCtx)

	// Start the goroutines as per the config and the availability.
	for numGoRoutines = 0; (numGoRoutines < job.fileCacheConfig.ParallelDownloadsPerFile) && (start < int64(job.object.Size)); numGoRoutines++ {
		// Respect max download parallelism only beyond first go routine.
		if numGoRoutines > 0 && !job.maxParallelismSem.TryAcquire(1) {
			break
		}

		downloadErrGroup.Go(job.downloadOffsets(downloadErrGroupCtx, numGoRoutines, cacheFile, rangeMap))
		start = start + downloadChunkSize
	}

	for start = 0; start < int64(job.object.Size); {
		nextRange := data.ObjectRange{
			Start: start,
			End:   min(int64(job.object.Size), start+downloadChunkSize),
		}

		select {
		case job.rangeChan <- nextRange:
			start = nextRange.End
			// In case we haven't started the goroutines as per the config, checking
			// if any goroutines are available now.
			// This may not be the ideal way, but since we don't have any way of
			// listening if goroutines from other jobs have freed up, checking it here.
			for numGoRoutines < job.fileCacheConfig.ParallelDownloadsPerFile && job.maxParallelismSem.TryAcquire(1) {
				downloadErrGroup.Go(job.downloadOffsets(downloadErrGroupCtx, numGoRoutines, cacheFile, rangeMap))
				numGoRoutines++
			}
		case <-downloadErrGroupCtx.Done():
			return job.handleJobCompletion(downloadErrGroupCtx, downloadErrGroup)
		}
	}

	return job.handleJobCompletion(downloadErrGroupCtx, downloadErrGroup)
}

// Job can be success or failure. This method will handle all the scenarios and
// return the appropriate error.
func (job *Job) handleJobCompletion(ctx context.Context, group *errgroup.Group) error {
	// Close the channel since we are ending the job.
	close(job.rangeChan)

	// First check if the context has reported any error. This is to handle scenario
	// where context is cancelled and no goroutines are running.
	err := ctx.Err()
	if err != nil {
		// Ideally not required, but this is an additional check to ensure that
		// no goroutines are running.
		waitErr := group.Wait()
		return errors.Join(err, waitErr)
	}

	// If any of the go routines failed, consider the async job failed.
	err = group.Wait()
	if err != nil {
		return err
	}

	return nil
}
