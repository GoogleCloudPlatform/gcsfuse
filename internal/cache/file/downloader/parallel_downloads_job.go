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

	cacheutil "github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/monitor"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"golang.org/x/sync/errgroup"
)

// downloadRange is a helper function to download a given range of object from
// GCS into given destination writer.
//
// This function doesn't take locks and can be executed parallely.
func (job *Job) downloadRange(ctx context.Context, dstWriter io.Writer, start, end uint64) error {
	newReader, err := job.bucket.NewReader(
		ctx,
		&gcs.ReadObjectRequest{
			Name:       job.object.Name,
			Generation: job.object.Generation,
			Range: &gcs.ByteRange{
				Start: start,
				Limit: end,
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

	monitor.CaptureGCSReadMetrics(ctx, util.Parallel, int64(end-start))

	_, err = io.CopyN(dstWriter, newReader, int64(end-start))
	if err != nil {
		err = fmt.Errorf("downloadRange: error at the time of copying content to cache file %w", err)
	}
	return err
}

func (job *Job) updateRangeMap(rangeMap map[uint64]uint64, offsetStart uint64, offsetEnd uint64) error {
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
		updateErr := job.updateStatusOffset(finalEnd)
		if updateErr != nil {
			return updateErr
		}
	}
	return nil
}

func (job *Job) downloadOffsets(ctx context.Context, goroutineIndex int, cacheFile *os.File, rangeMap map[uint64]uint64) func() error {
	return func() error {
		// Since we keep a goroutine for each job irrespective of the maxParallelism,
		// not releasing the default goroutine to the pool.
		if goroutineIndex > 0 {
			defer job.maxParallelismSem.Release(1)
		}

		for {
			// Read the offset to be downloaded from the channel.
			offsetToDownload, ok := <-job.offsetChan
			if !ok {
				// In case channel is closed return.
				return nil
			}

			offsetWriter := io.NewOffsetWriter(cacheFile, int64(offsetToDownload.start))
			err := job.downloadRange(ctx, offsetWriter, offsetToDownload.start, offsetToDownload.end)
			if err != nil {
				return err
			}

			err = job.updateRangeMap(rangeMap, offsetToDownload.start, offsetToDownload.end)
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
	rangeMap := make(map[uint64]uint64)
	job.offsetChan = make(chan offset, 2*job.fileCacheConfig.ParallelDownloadsPerFile)
	var numGoRoutines int64
	var start uint64
	downloadChunkSize := uint64(job.fileCacheConfig.DownloadChunkSizeMb) * uint64(cacheutil.MiB)
	downloadErrGroup, downloadErrGroupCtx := errgroup.WithContext(job.cancelCtx)

	// Start the goroutines as per the config and the availability.
	for numGoRoutines = 0; (numGoRoutines < job.fileCacheConfig.ParallelDownloadsPerFile) && (start < job.object.Size); numGoRoutines++ {
		// Respect max download parallelism only beyond first go routine.
		if numGoRoutines > 0 && !job.maxParallelismSem.TryAcquire(1) {
			break
		}

		downloadErrGroup.Go(job.downloadOffsets(downloadErrGroupCtx, numGoRoutines, cacheFile, rangeMap))
		start = start + downloadChunkSize
	}

	for start = 0; start < job.object.Size; {
		nextOffset := offset{
			start: start,
			end:   min(job.object.Size, start+downloadChunkSize),
		}

		select {
		case job.offsetChan <- nextOffset:
			start = nextOffset.end
			// In case we haven't started the goroutines as per the config, checking
			// if any goroutines are available now.
			// This may not be the ideal way, but since we don't have any way of
			// listening if goroutines from other jobs have freed up, checking it here.
			for numGoRoutines < job.fileCacheConfig.ParallelDownloadsPerFile && job.maxParallelismSem.TryAcquire(1) {
				downloadErrGroup.Go(job.downloadOffsets(downloadErrGroupCtx, numGoRoutines, cacheFile, rangeMap))
				numGoRoutines++
			}
		case <-downloadErrGroupCtx.Done():
			return job.handleJobCompletion(downloadErrGroupCtx, start, downloadErrGroup)
		}
	}

	return job.handleJobCompletion(downloadErrGroupCtx, start, downloadErrGroup)
}

// Job can be success or failure. This method will handle all the scenarios and
// return the appropriate error.
func (job *Job) handleJobCompletion(ctx context.Context, offset uint64, group *errgroup.Group) error {
	// Close the channel since we are ending the job.
	close(job.offsetChan)

	// First check if the context has reported any error.
	err := ctx.Err()
	if err != nil {
		// Also wait for all the goroutines to finish to ensure job is stopped.
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
