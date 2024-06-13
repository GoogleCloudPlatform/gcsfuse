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
		closeErr := newReader.Close()
		if closeErr != nil {
			logger.Errorf("Job:%p (%s:/%s) error while closing reader: %v", job, job.bucket.Name(), job.object.Name, closeErr)
		}
	}()

	monitor.CaptureGCSReadMetrics(ctx, util.Parallel, end-start)

	_, err = io.CopyN(dstWriter, newReader, end-start)
	if err != nil {
		err = fmt.Errorf("downloadRange: error at the time of copying content to cache file %w", err)
	}
	return err
}

// parallelDownloadObjectToFile does parallel download of the backing GCS object
// into given file handle using multiple NewReader method of gcs.Bucket running
// in parallel. This function is canceled if job.cancelCtx is canceled.
func (job *Job) parallelDownloadObjectToFile(cacheFile *os.File) (err error) {
	var start, end int64
	end = int64(job.object.Size)
	var parallelReadRequestSize = int64(job.fileCacheConfig.ReadRequestSizeMB) * cacheutil.MiB

	// Each iteration of this for loop downloads job.fileCacheConfig.ReadRequestSizeMB * job.fileCacheConfig.DownloadParallelismPerFile
	// size of range of object from GCS into given file handle and updates the
	// file info cache.
	for start < end {
		downloadErrGroup, downloadErrGroupCtx := errgroup.WithContext(job.cancelCtx)

		for goRoutineIdx := 0; (goRoutineIdx < job.fileCacheConfig.DownloadParallelismPerFile) && (start < end); goRoutineIdx++ {
			rangeStart := start
			rangeEnd := min(rangeStart+parallelReadRequestSize, end)

			downloadErrGroup.Go(func() error {
				// Copy the contents from NewReader to cache file at appropriate offset.
				offsetWriter := io.NewOffsetWriter(cacheFile, rangeStart)
				return job.downloadRange(downloadErrGroupCtx, offsetWriter, rangeStart, rangeEnd)
			})

			start = rangeEnd
		}

		// If any of the go routines failed, consider the async job failed.
		err = downloadErrGroup.Wait()
		if err != nil {
			return
		}

		err = job.updateStatusOffset(start)
		if err != nil {
			return err
		}
	}
	return
}
