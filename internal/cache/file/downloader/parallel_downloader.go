package downloader

import (
	"bufio"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"strings"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/lru"
	cacheutil "github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"golang.org/x/net/context"
)

// downloadObjectAsync downloads the backing GCS object into a file as part of
// file cache using NewReader method of gcs.Bucket.
//
// Note: There can only be one async download running for a job at a time.
// Acquires and releases LOCK(job.mu)
func (job *Job) downloadObjectInParallelAsync() {
	// Close the job.doneCh, clear the cancelFunc & cancelCtx and call the
	// remove job callback function in any case - completion/failure.
	defer func() {
		job.cancelFunc()
		close(job.doneCh)

		job.mu.Lock()
		if job.removeJobCallback != nil {
			job.removeJobCallback()
			job.removeJobCallback = nil
		}
		job.cancelCtx, job.cancelFunc = nil, nil
		job.mu.Unlock()
	}()

	// Create, open and truncate cache file for writing object into it.
	cacheFile, err := cacheutil.CreateFile(job.fileSpec, os.O_TRUNC|os.O_WRONLY)
	defer cacheFile.Close()
	if err != nil {
		err = fmt.Errorf("downloadObjectAsync: error in creating cache file: %w", err)
		job.failWhileDownloading(err)
		return
	}
	// Assumption: If we don't have the file already created then it may happen
	// that parallel go routines writing to the same file even at disjoint parts
	// can make the data corrupt.
	err = cacheFile.Truncate(int64(job.object.Size))
	if err != nil {
		err = fmt.Errorf("downloadObjectAsync: error while truncating the cache file: %w", err)
		job.failWhileDownloading(err)
		return
	}

	notifyInvalid := func() {
		job.mu.Lock()
		job.status.Name = Invalid
		job.notifySubscribers()
		job.mu.Unlock()
	}

	var start, end int64
	end = int64(job.object.Size)
	var readRequestSize int64 = int64(job.readRequestSizeMb * cacheutil.MiB)

	for {
		select {
		case <-job.cancelCtx.Done():
			return
		default:
			if start < end {
				logger.Infof("Start: %d, End: %d", start, end)
				limit := uint64(start) + uint64(readRequestSize)*uint64(job.downloadParallelism)
				if limit > job.object.Size {
					limit = job.object.Size
				}
				err = job.bucket.ParallelDownloadToFile(job.cancelCtx, &gcs.ParallelDownloadToFileRequest{
					Name:       job.object.Name,
					Generation: job.object.Generation,
					Range:      &gcs.ByteRange{Start: uint64(start), Limit: limit},
					FileHandle: cacheFile,
					PartSize:   uint64(job.readRequestSizeMb * cacheutil.MiB),
				})
				if err != nil {
					if errors.Is(err, context.Canceled) {
						notifyInvalid()
						return
					}
					job.failWhileDownloading(err)
					return
				} else {
					// Update the start offset on success.
					start = int64(limit)
					logger.Infof("changing the start: %d", start)
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
				job.mu.Lock()

				// Validate CRC32
				downloadedCrc, err := CRC32(job.fileSpec.Path)
				logger.Infof("Downloaded CRC: %d", downloadedCrc)
				logger.Infof("GCS crc: %d", *job.object.CRC32C)
				if err == nil && *job.object.CRC32C == downloadedCrc {
					logger.Infof("CRC32 check passed.")
				} else {
					job.mu.Unlock()
					job.failWhileDownloading(err)
					return
				}

				job.status.Name = Completed

				job.notifySubscribers()
				job.mu.Unlock()
				return
			}
		}
	}
}

const bufferSize = 65536

// CRCReader returns CRC-32-Castagnoli checksum of content in reader
func CRCReader(reader io.Reader) (uint32, error) {
	table := crc32.MakeTable(crc32.Castagnoli)
	checksum := crc32.Checksum([]byte(""), table)
	buf := make([]byte, bufferSize)
	for {
		switch n, err := reader.Read(buf); err {
		case nil:
			checksum = crc32.Update(checksum, table, buf[:n])
		case io.EOF:
			return checksum, nil
		default:
			return 0, err
		}
	}
}

func CRC32(filename string) (uint32, error) {
	if info, err := os.Stat(filename); err != nil || info.IsDir() {
		return 0, err
	}

	file, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer func() { _ = file.Close() }()

	return CRCReader(bufio.NewReader(file))
}
