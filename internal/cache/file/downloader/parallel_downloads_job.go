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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/data"
	cacheutil "github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"golang.org/x/sync/errgroup"
)

// downloadRange is a helper function to download a given range of object from
// GCS into given destination writer.
//
// This function doesn't take locks and can be executed parallely.
func (job *Job) downloadRange(ctx context.Context, dstWriter io.Writer, start, end int64, dummyBuffer []byte, rangeMap map[int64]int64) error {
	dummyBufferSize := int64(len(dummyBuffer))
	if dummyBufferSize == 0 {
		return fmt.Errorf("downloadRange: dummyBuffer is empty")
	}

	totalBytesToRead := end - start
	bytesReadSoFar := int64(0)

	// Use standard copy function if O_DIRECT is disabled and memory aligned
	// buffer otherwise.
	if !job.fileCacheConfig.EnableODirect {
		if job.IsExperimentalParallelDownloadsDefaultOn() {
			currentOffset := start
			for bytesReadSoFar < totalBytesToRead {
				bufferReader := bytes.NewReader(dummyBuffer)

				chunkSize := min(totalBytesToRead-bytesReadSoFar, ReadChunkSize)
				readSize := min(chunkSize, dummyBufferSize)

				_, err := io.CopyN(dstWriter, bufferReader, readSize)
				if err != nil && err != io.EOF {
					return fmt.Errorf("downloadRange: error copying dummy buffer content to cache file: %w", err)
				}

				err = job.updateRangeMap(rangeMap, currentOffset, currentOffset+readSize)
				if err != nil {
					return err
				}
				bytesReadSoFar += readSize
				currentOffset += readSize
			}
		} else {
			bytesCopied := int64(0)
			var err error
			for bytesCopied < totalBytesToRead {
				bufferReader := bytes.NewReader(dummyBuffer)
				toCopy := min(totalBytesToRead-bytesCopied, dummyBufferSize)
				var n int64
				n, err = io.CopyN(dstWriter, bufferReader, toCopy)
				bytesCopied += n
				if err != nil && err != io.EOF {
					return fmt.Errorf("downloadRange: error copying dummy buffer content (non-experimental): %w", err)
				}
				if err == io.EOF && toCopy > 0 && n < toCopy {
					return fmt.Errorf("downloadRange: unexpected EOF from bytes.Reader")
				}

				if err == io.EOF {
					err = nil
				}
				if n == 0 && toCopy > 0 {
					// Break if we cannot copy anything more, e.g. totalBytesToRead met.
					break
				}
			}

			mapErr := job.updateRangeMap(rangeMap, start, end)
			if mapErr != nil {
				return mapErr
			}

			if err != nil {
				return fmt.Errorf("downloadRange: error copying dummy buffer content (non-experimental): %w", err)
			}
		}
	} else {
		if job.IsExperimentalParallelDownloadsDefaultOn() {
			currentOffset := start
			for bytesReadSoFar < totalBytesToRead {
				bufferReader := bytes.NewReader(dummyBuffer)

				chunkSize := min(totalBytesToRead-bytesReadSoFar, ReadChunkSize)
				readSize := min(chunkSize, dummyBufferSize)
				chunkReader := io.LimitReader(bufferReader, readSize)

				_, err := cacheutil.CopyUsingMemoryAlignedBuffer(ctx, chunkReader, dstWriter, readSize,
					job.fileCacheConfig.WriteBufferSize)

				// If context is canceled while reading/writing in CopyUsingMemoryAlignedBuffer
				// then it returns error different from context cancelled (invalid argument),
				// and we need to report that error as context cancelled.
				if !errors.Is(err, context.Canceled) && errors.Is(ctx.Err(), context.Canceled) {
					err = errors.Join(err, ctx.Err())
					return err
				}
				if err != nil && err != io.EOF {
					return fmt.Errorf("downloadRange: error copying dummy buffer content (O_DIRECT): %w", err)
				}

				err = job.updateRangeMap(rangeMap, currentOffset, currentOffset+readSize)
				if err != nil {
					return err
				}

				bytesReadSoFar += readSize
				currentOffset += readSize
			}
		} else {
			bytesCopied := int64(0)
			var lastErr error
			for bytesCopied < totalBytesToRead {
				bufferReader := bytes.NewReader(dummyBuffer)
				toCopy := min(totalBytesToRead-bytesCopied, dummyBufferSize)
				chunkReader := io.LimitReader(bufferReader, toCopy)
				var nCopied int64

				nCopied, lastErr = cacheutil.CopyUsingMemoryAlignedBuffer(ctx, chunkReader, dstWriter, toCopy,
					job.fileCacheConfig.WriteBufferSize)

				bytesCopied += nCopied

				// If context is canceled while reading/writing in CopyUsingMemoryAlignedBuffer
				// then it returns error different from context cancelled (invalid argument),
				// and we need to report that error as context cancelled.
				if !errors.Is(lastErr, context.Canceled) && errors.Is(ctx.Err(), context.Canceled) {
					lastErr = errors.Join(lastErr, ctx.Err())
					break
				}

				if lastErr != nil && lastErr != io.EOF {
					break
				}
				if nCopied == 0 && toCopy > 0 {
					break
				}

				if lastErr == io.EOF {
					lastErr = nil
				}
			}

			mapErr := job.updateRangeMap(rangeMap, start, end)
			if mapErr != nil {
				return mapErr
			}
			if lastErr != nil {
				return fmt.Errorf("downloadRange: error copying dummy buffer content (O_DIRECT non-experimental): %w", lastErr)
			}
		}
	}

	return nil
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
func (job *Job) downloadOffsets(ctx context.Context, goroutineIndex int64, cacheFile *os.File, dummyBuffer []byte, rangeMap map[int64]int64) func() error {
	return func() error {
		// Since we keep a goroutine for each job irrespective of the maxParallelism,
		// not releasing the default goroutine to the pool.
		if goroutineIndex > 0 {
			defer job.maxParallelismSem.Release(1)
		}
		var err error

		for {
			// Read the offset to be downloaded from the channel.
			objectRange, ok := <-job.rangeChan
			if !ok {
				// In case channel is closed return.
				return nil
			}

			offsetWriter := io.NewOffsetWriter(cacheFile, objectRange.Start)
			err = job.downloadRange(ctx, offsetWriter, objectRange.Start, objectRange.End, dummyBuffer, rangeMap)
			if err != nil {
				return err
			}

			if !job.IsExperimentalParallelDownloadsDefaultOn() {
				err = job.updateRangeMap(rangeMap, objectRange.Start, objectRange.End)
				if err != nil {
					return err
				}
			}
		}
	}
}

// parallelDownloadObjectToFile does parallel download of the backing GCS object
// into given file handle using multiple NewReader method of gcs.Bucket running
// in parallel. This function is canceled if job.cancelCtx is canceled.
func (job *Job) parallelDownloadObjectToFile(cacheFile *os.File) (err error) {
	downloadChunkSize := job.fileCacheConfig.DownloadChunkSizeMb * cacheutil.MiB
	if downloadChunkSize <= 0 {
		return fmt.Errorf("parallelDownloadObjectToFile: invalid DownloadChunkSizeMb %d", job.fileCacheConfig.DownloadChunkSizeMb)
	}
	dummyBuffer := make([]byte, downloadChunkSize)
	for i := range dummyBuffer {
		dummyBuffer[i] = 0xDA
	}
	logger.Debugf("Created dummy buffer of size %d bytes", downloadChunkSize)

	startTime := time.Now()
	logger.Errorf("start downloading file %s", job.object.Name)
	rangeMap := make(map[int64]int64)
	job.rangeChan = make(chan data.ObjectRange, 2*job.fileCacheConfig.ParallelDownloadsPerFile)
	var numGoRoutines int64
	var start int64
	downloadErrGroup, downloadErrGroupCtx := errgroup.WithContext(job.cancelCtx)

	for numGoRoutines = 0; (numGoRoutines < job.fileCacheConfig.ParallelDownloadsPerFile) && (start < int64(job.object.Size)); numGoRoutines++ {
		if numGoRoutines > 0 && !job.maxParallelismSem.TryAcquire(1) {
			break
		}

		downloadErrGroup.Go(job.downloadOffsets(downloadErrGroupCtx, numGoRoutines, cacheFile, dummyBuffer, rangeMap))
		start = start + downloadChunkSize
	}

	start = 0
	for start < int64(job.object.Size) {
		nextRange := data.ObjectRange{
			Start: start,
			End:   min(int64(job.object.Size), start+downloadChunkSize),
		}

		select {
		case job.rangeChan <- nextRange:
			start = nextRange.End
			for numGoRoutines < job.fileCacheConfig.ParallelDownloadsPerFile && job.maxParallelismSem.TryAcquire(1) {
				downloadErrGroup.Go(job.downloadOffsets(downloadErrGroupCtx, numGoRoutines, cacheFile, dummyBuffer, rangeMap))
				numGoRoutines++
			}
		case <-downloadErrGroupCtx.Done():
			return job.handleJobCompletion(downloadErrGroupCtx, downloadErrGroup)
		}
	}

	err = job.handleJobCompletion(downloadErrGroupCtx, downloadErrGroup)
	logger.Errorf("Download time for file %s, time: %s", job.object.Name, time.Since(startTime))
	elapsedTime := time.Since(startTime).Seconds()
	throughputMiB := float64(job.object.Size) / float64(cacheutil.MiB) / elapsedTime
	logger.Errorf("Throughput for file %s: %.2f MiB/s", job.object.Name, throughputMiB)
	return err
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
