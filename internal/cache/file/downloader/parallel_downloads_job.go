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
func (job *Job) downloadRange(ctx context.Context, dstWriter io.Writer, start, end int64, dummyFilePath string, rangeMap map[int64]int64) error {
	// Open the dummy file for reading.
	dummyFile, err := os.Open(dummyFilePath)
	if err != nil {
		return fmt.Errorf("downloadRange: error opening dummy file %s: %w", dummyFilePath, err)
	}
	defer dummyFile.Close()

	// Get dummy file info to know its size.
	dummyFileInfo, err := dummyFile.Stat()
	if err != nil {
		return fmt.Errorf("downloadRange: error stating dummy file %s: %w", dummyFilePath, err)
	}
	dummyFileSize := dummyFileInfo.Size()

	// Create a reader for the dummy file. We will read repeatedly from the start.
	// Use io.SectionReader to simulate reading a specific part if the dummy file
	// represented the whole object content, but the request implies *repeatedly* reading
	// the chunk-sized dummy file. So, we'll seek to start and limit the read.

	// Simulate reading the range [start, end) from the potentially repeating dummy data.
	totalBytesToRead := end - start
	bytesReadSoFar := int64(0)

	// Common metrics capture - kept for structure, but might need adjustment for simulation context.
	// common.CaptureGCSReadMetrics(ctx, job.metricsHandle, util.Parallel, totalBytesToRead)

	// Use standard copy function if O_DIRECT is disabled and memory aligned
	// buffer otherwise.
	if !job.fileCacheConfig.EnableODirect {
		if job.IsExperimentalParallelDownloadsDefaultOn() {
			currentOffset := start
			for bytesReadSoFar < totalBytesToRead {
				// Seek to the beginning of the dummy file for each chunk read to simulate repetition.
				_, err = dummyFile.Seek(0, io.SeekStart)
				if err != nil {
					return fmt.Errorf("downloadRange: error seeking dummy file: %w", err)
				}

				chunkSize := min(totalBytesToRead-bytesReadSoFar, ReadChunkSize)
				// Ensure we don't read past the dummy file's actual size in one go.
				readSize := min(chunkSize, dummyFileSize)
				// Wrap the dummy file reader with a LimitedReader for this chunk.
				chunkReader := io.LimitReader(dummyFile, readSize)

				_, err = io.CopyN(dstWriter, chunkReader, readSize) // Read only 'readSize' from dummy file
				if err != nil && err != io.EOF {                    // EOF is expected if readSize < chunkSize, but CopyN handles this
					return fmt.Errorf("downloadRange: error copying dummy content to cache file: %w", err)
				}

				err = job.updateRangeMap(rangeMap, currentOffset, currentOffset+readSize)
				if err != nil {
					return err
				}
				bytesReadSoFar += readSize
				currentOffset += readSize
			}
		} else {
			// Original logic for non-experimental or O_DIRECT disabled - needs adaptation
			// Seek to the beginning of the dummy file.
			_, err = dummyFile.Seek(0, io.SeekStart)
			if err != nil {
				return fmt.Errorf("downloadRange: error seeking dummy file: %w", err)
			}
			// Read repeatedly up to totalBytesToRead
			bytesCopied := int64(0)
			for bytesCopied < totalBytesToRead {
				_, err = dummyFile.Seek(0, io.SeekStart) // Seek back to start for repetition
				if err != nil {
					return fmt.Errorf("downloadRange: error seeking dummy file: %w", err)
				}
				toCopy := min(totalBytesToRead-bytesCopied, dummyFileSize)
				n, err := io.CopyN(dstWriter, dummyFile, toCopy)
				bytesCopied += n
				if err != nil && err != io.EOF { // EOF is ok if toCopy == dummyFileSize
					return fmt.Errorf("downloadRange: error copying dummy content (non-experimental): %w", err)
				}
				if err == io.EOF {
					break // Should not happen if totalBytesToRead > 0 and dummyFileSize > 0 unless totalBytesToRead is multiple of dummyFileSize
				}
			}
		}
	} else { // O_DIRECT enabled
		if job.IsExperimentalParallelDownloadsDefaultOn() {
			currentOffset := start
			for bytesReadSoFar < totalBytesToRead {
				// Seek to the beginning of the dummy file for each chunk read.
				_, err = dummyFile.Seek(0, io.SeekStart)
				if err != nil {
					return fmt.Errorf("downloadRange: error seeking dummy file (O_DIRECT): %w", err)
				}

				chunkSize := min(totalBytesToRead-bytesReadSoFar, ReadChunkSize)
				readSize := min(chunkSize, dummyFileSize)
				// Wrap the dummy file reader with a LimitedReader for this chunk.
				chunkReader := io.LimitReader(dummyFile, readSize)

				_, err = cacheutil.CopyUsingMemoryAlignedBuffer(ctx, chunkReader, dstWriter, readSize,
					job.fileCacheConfig.WriteBufferSize)

				// Context cancellation check
				if !errors.Is(err, context.Canceled) && errors.Is(ctx.Err(), context.Canceled) {
					err = errors.Join(err, ctx.Err())
					return err
				}
				if err != nil && err != io.EOF { // EOF is expected if readSize < chunkSize
					return fmt.Errorf("downloadRange: error copying dummy content (O_DIRECT): %w", err)
				}

				err = job.updateRangeMap(rangeMap, currentOffset, currentOffset+readSize)
				if err != nil {
					return err
				}

				bytesReadSoFar += readSize
				currentOffset += readSize
			}
		} else {
			// Original logic for non-experimental or O_DIRECT enabled - needs adaptation
			// Seek to the beginning of the dummy file.
			_, err = dummyFile.Seek(0, io.SeekStart)
			if err != nil {
				return fmt.Errorf("downloadRange: error seeking dummy file (O_DIRECT non-experimental): %w", err)
			}
			// Read repeatedly up to totalBytesToRead
			bytesCopied := int64(0)
			for bytesCopied < totalBytesToRead {
				_, err = dummyFile.Seek(0, io.SeekStart) // Seek back to start for repetition
				if err != nil {
					return fmt.Errorf("downloadRange: error seeking dummy file (O_DIRECT non-experimental): %w", err)
				}
				toCopy := min(totalBytesToRead-bytesCopied, dummyFileSize)

				// Use CopyUsingMemoryAlignedBuffer repeatedly
				nCopied, err := cacheutil.CopyUsingMemoryAlignedBuffer(ctx, dummyFile, dstWriter, toCopy,
					job.fileCacheConfig.WriteBufferSize)

				bytesCopied += nCopied
				// Context cancellation check
				if !errors.Is(err, context.Canceled) && errors.Is(ctx.Err(), context.Canceled) {
					err = errors.Join(err, ctx.Err())
				}
				if err != nil && err != io.EOF { // EOF is ok if toCopy == dummyFileSize
					return fmt.Errorf("downloadRange: error copying dummy content (O_DIRECT non-experimental): %w", err)
				}
				if err == io.EOF {
					break // Should not happen if totalBytesToRead > 0 and dummyFileSize > 0 unless totalBytesToRead is multiple of dummyFileSize
				}
			}
			// Check context error after loop if original logic did
			if !errors.Is(err, context.Canceled) && errors.Is(ctx.Err(), context.Canceled) {
				err = errors.Join(err, ctx.Err())
			}
		}
	}

	// Final error check based on original logic (which checked 'err' after copy)
	if err != nil && err != io.EOF { // Allow EOF as it signifies end of dummy chunk read potentially
		return fmt.Errorf("downloadRange: error copying dummy content to cache file: %w", err)
	}

	// Return success (nil error). Removed ReadHandle return.
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
func (job *Job) downloadOffsets(ctx context.Context, goroutineIndex int64, cacheFile *os.File, dummyFilePath string, rangeMap map[int64]int64) func() error {
	return func() error {
		// Since we keep a goroutine for each job irrespective of the maxParallelism,
		// not releasing the default goroutine to the pool.
		if goroutineIndex > 0 {
			defer job.maxParallelismSem.Release(1)
		}
		// Removed GCS specific 'readHandle'
		var err error

		for {
			// Read the offset to be downloaded from the channel.
			objectRange, ok := <-job.rangeChan
			if !ok {
				// In case channel is closed return.
				return nil
			}

			offsetWriter := io.NewOffsetWriter(cacheFile, objectRange.Start)
			// Call modified downloadRange, passing dummyFilePath instead of readHandle
			err = job.downloadRange(ctx, offsetWriter, objectRange.Start, objectRange.End, dummyFilePath, rangeMap)
			if err != nil {
				return err // Return error from downloadRange
			}

			// Update range map logic (original placement if not experimental)
			if !job.IsExperimentalParallelDownloadsDefaultOn() {
				err = job.updateRangeMap(rangeMap, objectRange.Start, objectRange.End)
				if err != nil {
					return err
				}
			}
		}
	}
}

// createDummyFile creates a dummy file in /tmp with the specified size.
func createDummyFile(size int64) (string, error) {
	// Create a temporary file in /tmp
	dummyFile, err := os.CreateTemp("/tmp", "gcsfuse-dummy-download-*.bin")
	if err != nil {
		return "", fmt.Errorf("createDummyFile: failed to create temp file: %w", err)
	}
	defer dummyFile.Close()

	// Write dummy data (zero bytes or a pattern) to the file until it reaches the desired size.
	// Using a simple loop with Write is often clearer than CopyN from /dev/zero for this purpose.
	// Alternatively, use Truncate and potentially fallocate if performance is critical for setup.
	// For simplicity, let's write in chunks.
	const bufferSize = 4 * 1024 * 1024 // 4 MiB buffer
	buffer := make([]byte, bufferSize)
	// Fill buffer with dummy data if needed, default is zero bytes.
	// for i := range buffer {
	// 	buffer[i] = dummyDataByte
	// }

	written := int64(0)
	for written < size {
		bytesToWrite := min(size-written, int64(len(buffer)))
		n, err := dummyFile.Write(buffer[:bytesToWrite])
		if err != nil {
			os.Remove(dummyFile.Name()) // Clean up on error
			return "", fmt.Errorf("createDummyFile: failed to write to temp file: %w", err)
		}
		written += int64(n)
	}

	err = dummyFile.Sync()
	if err != nil {
		os.Remove(dummyFile.Name())
		return "", fmt.Errorf("createDummyFile: failed to sync temp file: %w", err)
	}

	logger.Debugf("Created dummy file %s of size %d bytes", dummyFile.Name(), size)
	return dummyFile.Name(), nil
}

// parallelDownloadObjectToFile does parallel download of the backing GCS object
// into given file handle using multiple NewReader method of gcs.Bucket running
// in parallel. This function is canceled if job.cancelCtx is canceled.
func (job *Job) parallelDownloadObjectToFile(cacheFile *os.File) (err error) {
	// Create the dummy file
	downloadChunkSize := job.fileCacheConfig.DownloadChunkSizeMb * cacheutil.MiB
	dummyFilePath, err := createDummyFile(downloadChunkSize)
	if err != nil {
		return fmt.Errorf("parallelDownloadObjectToFile: failed to create dummy file: %w", err)
	}
	// Ensure dummy file is removed when the function returns
	defer func() {
		removeErr := os.Remove(dummyFilePath)
		if removeErr != nil {
			logger.Warnf("parallelDownloadObjectToFile: failed to remove dummy file %s: %v", dummyFilePath, removeErr)
		}
	}()

	startTime := time.Now()
	logger.Errorf("start downloading file %s", job.object.Name)
	rangeMap := make(map[int64]int64)
	// Trying to keep the channel size greater than ParallelDownloadsPerFile to ensure
	// that there is no goroutine waiting for data(nextRange) to be published to channel.
	job.rangeChan = make(chan data.ObjectRange, 2*job.fileCacheConfig.ParallelDownloadsPerFile)
	var numGoRoutines int64
	var start int64
	downloadErrGroup, downloadErrGroupCtx := errgroup.WithContext(job.cancelCtx)

	for numGoRoutines = 0; (numGoRoutines < job.fileCacheConfig.ParallelDownloadsPerFile) && (start < int64(job.object.Size)); numGoRoutines++ {
		// Respect max download parallelism only beyond first go routine.
		if numGoRoutines > 0 && !job.maxParallelismSem.TryAcquire(1) {
			break
		}

		downloadErrGroup.Go(job.downloadOffsets(downloadErrGroupCtx, numGoRoutines, cacheFile, dummyFilePath, rangeMap))
		start = start + downloadChunkSize // Increment start based on chunk size for goroutine distribution logic
	}

	for start = 0; start < int64(job.object.Size); {
		nextRange := data.ObjectRange{
			Start: start,
			End:   min(int64(job.object.Size), start+downloadChunkSize), // Ranges still based on object size and chunk size
		}

		select {
		case job.rangeChan <- nextRange:
			start = nextRange.End
			// Start more goroutines if available
			for numGoRoutines < job.fileCacheConfig.ParallelDownloadsPerFile && job.maxParallelismSem.TryAcquire(1) {
				downloadErrGroup.Go(job.downloadOffsets(downloadErrGroupCtx, numGoRoutines, cacheFile, dummyFilePath, rangeMap))
				numGoRoutines++
			}
		case <-downloadErrGroupCtx.Done():
			return job.handleJobCompletion(downloadErrGroupCtx, downloadErrGroup)
		}
	}

	err = job.handleJobCompletion(downloadErrGroupCtx, downloadErrGroup)
	logger.Errorf("Download time for file %s, time: %s", job.object.Name, time.Since(startTime))
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
