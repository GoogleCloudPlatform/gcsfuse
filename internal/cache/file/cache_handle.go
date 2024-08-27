// Copyright 2023 Google LLC
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

package file

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/file/downloader"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
)

type CacheHandle struct {
	// fileHandle to a local file which contains locally downloaded data.
	fileHandle *os.File

	// fileDownloadJob is a reference to async download Job. It can be nil if
	// job is already completed.
	fileDownloadJob *downloader.Job

	// fileInfoCache contains the reference to fileInfo cache.
	fileInfoCache *lru.Cache

	// cacheFileForRangeRead if true, async download job will start even for range
	// reads.
	cacheFileForRangeRead bool

	// isSequential saves if the current read performed via cache handle is sequential or
	// random.
	isSequential bool

	// prevOffset stores the offset of previous cache handle read call. This is used
	// to decide the type of read.
	prevOffset int64
}

func NewCacheHandle(localFileHandle *os.File, fileDownloadJob *downloader.Job,
	fileInfoCache *lru.Cache, cacheFileForRangeRead bool, initialOffset int64) *CacheHandle {
	return &CacheHandle{
		fileHandle:            localFileHandle,
		fileDownloadJob:       fileDownloadJob,
		fileInfoCache:         fileInfoCache,
		cacheFileForRangeRead: cacheFileForRangeRead,
		isSequential:          initialOffset == 0,
		prevOffset:            initialOffset,
	}
}

func (fch *CacheHandle) validateCacheHandle() error {
	if fch.fileHandle == nil {
		return errors.New(util.InvalidFileHandleErrMsg)
	}

	if fch.fileInfoCache == nil {
		return errors.New(util.InvalidFileInfoCacheErrMsg)
	}

	return nil
}

// shouldReadFromCache returns nil if the data should be read from the locally
// downloaded cache file. Otherwise, it returns an appropriate error message.
func (fch *CacheHandle) shouldReadFromCache(jobStatus *downloader.JobStatus, requiredOffset int64) (err error) {
	if jobStatus.Err != nil ||
		jobStatus.Name == downloader.Invalid ||
		jobStatus.Name == downloader.Failed {
		err := fmt.Errorf("%s: jobStatus: %s jobError: %w", util.InvalidFileDownloadJobErrMsg, jobStatus.Name, jobStatus.Err)
		return err
	} else if jobStatus.Offset < requiredOffset {
		err := fmt.Errorf("%s: jobOffset: %d is less than required offset: %d", util.FallbackToGCSErrMsg, jobStatus.Offset, requiredOffset)
		return err
	}
	return err
}

// validateEntryInFileInfoCache checks if entry is present for a given object in
// file info cache with same generation and at least requiredOffset.
// It returns nil if entry is present, otherwise returns an appropriate error.
// Whether to change the order in cache while lookup is controlled via
// changeCacheOrder.
func (fch *CacheHandle) validateEntryInFileInfoCache(bucket gcs.Bucket, object *gcs.MinObject, requiredOffset uint64, changeCacheOrder bool) error {
	fileInfoKey := data.FileInfoKey{
		BucketName: bucket.Name(),
		ObjectName: object.Name,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	if err != nil {
		return fmt.Errorf("error while creating key for bucket %s and object %s: %w", bucket.Name(), object.Name, err)
	}

	var fileInfo lru.ValueType
	if changeCacheOrder {
		fileInfo = fch.fileInfoCache.LookUp(fileInfoKeyName)
	} else {
		fileInfo = fch.fileInfoCache.LookUpWithoutChangingOrder(fileInfoKeyName)
	}
	if fileInfo == nil {
		err = fmt.Errorf("%v: no entry found in file info cache for key %v", util.InvalidFileInfoCacheErrMsg, fileInfoKeyName)
		return err
	}

	// The generation check below is required because it may happen that file
	// being read is evicted from cache during or after reading the required offset
	// from local cached file to `dst` buffer.
	fileInfoData := fileInfo.(data.FileInfo)
	if fileInfoData.ObjectGeneration != object.Generation {
		err = fmt.Errorf("%v: generation of cached object: %v is different from required generation: %v", util.InvalidFileInfoCacheErrMsg, fileInfoData.ObjectGeneration, object.Generation)
		return err
	}
	if fileInfoData.Offset < requiredOffset {
		err = fmt.Errorf("%v offset of cached object: %v is less than required offset %v", util.InvalidFileInfoCacheErrMsg, fileInfoData.Offset, requiredOffset)
		return err
	}

	return nil
}

// Read attempts to read the data from the cached location.
// For sequential reads, it will wait to download the requested chunk
// if it is not already present. For random reads, it does not wait for
// download. Additionally, for random reads, the download will not be
// initiated if fch.cacheFileForRangeRead is false.
func (fch *CacheHandle) Read(ctx context.Context, bucket gcs.Bucket, object *gcs.MinObject, offset int64, dst []byte) (n int, cacheHit bool, err error) {
	err = fch.validateCacheHandle()
	if err != nil {
		return
	}

	if offset < 0 || offset >= int64(object.Size) {
		return 0, false, fmt.Errorf("wrong offset requested: %d, object size: %d", offset, object.Size)
	}

	// Checking before updating the previous offset.
	isSequentialRead := fch.IsSequential(offset)
	waitForDownload := true
	if !isSequentialRead {
		fch.isSequential = false
		waitForDownload = false
	}

	// We need to download the data till offset + len(dst), if not already.
	bufferLen := int64(len(dst))
	requiredOffset := offset + bufferLen

	// Also, assuming that dst buffer in read can be more than the remaining object length
	// left for reading. Hence, making sure requiredOffset should not more than object-length.
	objSize := int64(object.Size)
	if requiredOffset > objSize {
		requiredOffset = objSize
	}

	// If fileDownloadJob is not nil, it's better to get status of cache file
	// from the job itself than to use file info cache.
	if fch.fileDownloadJob != nil {
		jobStatus := fch.fileDownloadJob.GetStatus()
		// If cacheFileForRangeRead is false and readType is random, download will
		// not be initiated.
		if !fch.cacheFileForRangeRead && !isSequentialRead {
			if err = fch.shouldReadFromCache(&jobStatus, requiredOffset); err != nil {
				return 0, false, err
			}
		}

		if jobStatus.Offset >= requiredOffset {
			cacheHit = true
		}

		fch.prevOffset = offset

		//if fch.fileDownloadJob.IsParallelDownloadsEnabled() {
		//	waitForDownload = false
		//}

		jobStatus, err = fch.fileDownloadJob.Download(ctx, requiredOffset, waitForDownload)
		if err != nil {
			n = 0
			cacheHit = false
			err = fmt.Errorf("read: while downloading through job: %w", err)
			return
		}

		if err = fch.shouldReadFromCache(&jobStatus, requiredOffset); err != nil {
			return 0, false, err
		}
	} else {
		// If fileDownloadJob is nil then it means either the job is successfully
		// completed or failed. The offset must be equal to size of object for job
		// to be completed.
		err = fch.validateEntryInFileInfoCache(bucket, object, object.Size, false)
		if err != nil {
			return 0, false, err
		}
		cacheHit = true
	}

	// We are here means, we have the data downloaded which kernel has asked for.
	n, err = fch.fileHandle.ReadAt(dst, offset)
	requestedNumBytes := int(requiredOffset - offset)
	// dst buffer has fixed size of 1 MiB even when the offset is such that
	// offset + 1 MiB > object size. In that case, io.ErrUnexpectedEOF is thrown
	// which should be ignored.
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		if n != requestedNumBytes {
			// Ensure that the number of bytes read into dst buffer is equal to what is
			// requested. It will also help catch cases where file in cache is truncated
			// externally to size offset + x where x < requestedNumBytes.
			errMsg := fmt.Sprintf("%s, number of bytes read from file in cache: %v are not equal to requested: %v", util.ErrInReadingFileHandleMsg, n, requestedNumBytes)
			return 0, false, errors.New(errMsg)
		}
		err = nil
	}
	if err != nil {
		err = fmt.Errorf("%s: while reading from %d offset of the local file: %w", util.ErrInReadingFileHandleMsg, offset, err)
		return 0, false, err
	}

	// Look up of file being read in file info cache is required to update the LRU
	// order on every read request from kernel i.e. with every read request from
	// kernel, the file being read becomes most recently used.
	err = fch.validateEntryInFileInfoCache(bucket, object, uint64(requiredOffset), true)
	if err != nil {
		return 0, false, err
	}

	return
}

// IsSequential returns true if the sequential read is being performed, false for
// random read.
func (fch *CacheHandle) IsSequential(currentOffset int64) bool {
	if !fch.isSequential {
		return false
	}

	if currentOffset < fch.prevOffset {
		return false
	}

	if currentOffset-fch.prevOffset > downloader.ReadChunkSize {
		return false
	}

	return true
}

// Close closes the underlying fileHandle pointing to locally downloaded cache file.
func (fch *CacheHandle) Close() (err error) {
	if fch.fileHandle != nil {
		err = fch.fileHandle.Close()
		if err != nil {
			err = fmt.Errorf("cacheHandle.Close(): while closing read file handle: %w", err)
		}
		fch.fileHandle = nil
	}

	return
}
