// Copyright 2023 Google Inc. All Rights Reserved.
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
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/file/downloader"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/gcs"
)

const (
	InvalidFileHandle      = "destroy this cache_handle: invalid file handle"
	InvalidFileDownloadJob = "destroy this cache_handle: invalid download job"
	InvalidCacheHandle     = "destroy this cache_handle: invalid cache handle"
	InvalidFileInfoCache   = "file info cache is nil"
	ErrInSeekingFileHandle = "destroy this cache_handle: error while seeking file handle"
	ErrInReadingFileHandle = "destroy this cache_handle: error while reading file handle"
	FallbackToGCS          = "read via gcs without destroying this cache_handle"
	MB                     = 1 << 20
	MaxAllowedMB           = 8 * MB
)

type CacheHandle struct {
	// fileHandle to a local file which contains locally downloaded data.
	fileHandle *os.File

	//	fileDownloadJob is a reference to async download Job.
	fileDownloadJob *downloader.Job

	// fileInfoCache contains the reference of fileInfo cache.
	fileInfoCache *lru.Cache

	// isSequential saves if the current read performed via cache handle is sequential or
	// random.
	isSequential bool

	// prevOffset stores the offset of previous cache_handle read call. This is used
	// to decide the type of read.
	prevOffset int64

	// contributesToJobRefCount indicates that the cache handle contains the reference
	// count of the Job. We maintain this reference count manually, since we keep the
	// Job reference always, regardless of whether the read is sequential or random.
	// However, by design, we want to cancel the Job if there is no sequential read
	// active that depends on the Job.
	contributesToJobRefCount bool
}

func NewCacheHandle(localFileHandle *os.File, fileDownloadJob *downloader.Job, fileInfoCache *lru.Cache, initialOffset int64) *CacheHandle {
	return &CacheHandle{
		fileHandle:               localFileHandle,
		fileDownloadJob:          fileDownloadJob,
		fileInfoCache:            fileInfoCache,
		isSequential:             initialOffset == 0,
		prevOffset:               initialOffset,
		contributesToJobRefCount: true,
	}
}

func (fch *CacheHandle) validateCacheHandle() error {

	if fch.fileHandle == nil {
		return errors.New(InvalidFileHandle)
	}

	if fch.fileDownloadJob == nil {
		return errors.New(InvalidFileDownloadJob)
	}

	if fch.fileInfoCache == nil {
		return errors.New(InvalidFileInfoCache)
	}

	return nil
}

func (fch *CacheHandle) checkIfEntryExistWithCorrectGeneration(object *gcs.MinObject, bucket gcs.Bucket) (bool, error) {
	// Create fileInfoKey to get the existing fileInfoEntry in the cache.
	// we don't need this.
	fileInfoKeyName, err := data.GetFileInfoKeyName(object.Name, time.Time{}, bucket.Name())
	if err != nil {
		return false, fmt.Errorf("error while fetching key: %v", err)
	}

	// Get the existing fileInfo entry in the cache.
	fileInfo := fch.fileInfoCache.LookUp(fileInfoKeyName)
	if fileInfo == nil {
		return false, nil
	}

	return fileInfo.(data.FileInfo).ObjectGeneration == object.Generation, nil
}

// Read attempts to read the data from the cached location. This expects a
// fileInfoCache entry for the current read request, and will wait to download
// the requested chunk if it is not already present for sequential read.
// It doesn't wait, in case of random reads.
func (fch *CacheHandle) Read(ctx context.Context, object *gcs.MinObject, bucket gcs.Bucket, offset int64, dst []byte) (n int, err error) {
	err = fch.validateCacheHandle()
	if err != nil {
		return
	}

	fch.prevOffset = offset

	// We need to download the data till offset + len(dst), if not already.
	bufferLen := int64(len(dst))
	requiredOffset := offset + bufferLen

	// Also, need to make sure, it should not exceed the total object-size.
	objSize := int64(object.Size)
	if requiredOffset > objSize {
		requiredOffset = objSize
	}

	waitForDownload := true
	if !fch.isSequential {
		waitForDownload = false
	}

	// Todo (raj-prince): to remove this check after implementation of download method.
	if false {
		var jobStatus downloader.JobStatus
		jobStatus, err = fch.fileDownloadJob.Download(ctx, int64(requiredOffset), waitForDownload)
		if err != nil {
			n = 0
			err = fmt.Errorf("while downloading job: %v", err)
			return
		}

		// Todo - (raj-prince): to incorporate the newly introduced Invalid flag.
		if jobStatus.Name == "Invalid" ||
				jobStatus.Name == downloader.NOT_STARTED ||
				jobStatus.Name == downloader.FAILED {
			return 0, errors.New(InvalidFileDownloadJob)
		} else if jobStatus.Offset < requiredOffset {
			return 0, errors.New(FallbackToGCS)
		}
	}

	// We are here means, we have the data downloaded which kernel has asked for.
	_, err = fch.fileHandle.Seek(int64(offset), 0)
	if err != nil {
		logger.Warnf("error while seeking for %d offset in local file: %v", offset, err)
		return 0, errors.New(ErrInSeekingFileHandle)
	}

	n, err = io.ReadFull(fch.fileHandle, dst)
	if err == io.EOF {
		err = nil
	}
	if err != nil {
		logger.Warnf("error while reading from %d offset of the local file: %v", offset, err)
		return 0, errors.New(ErrInReadingFileHandle)
	}

	// Handling special cases:
	// 1. If there is no cache entry for the file, we will return no data.
	// 2. If there is a race condition between read and check, e.g., eviction
	//		following up with the addition of the same fileInfo.
	//    (a) If the generations are the same, this will not cause any issue.
	//    (b) If the generations are different, we will discard the old data and
	//        serve new data from gcs for this call.
	ok, err := fch.checkIfEntryExistWithCorrectGeneration(object, bucket)
	if err != nil || !ok {
		n = 0
		err = errors.New(InvalidCacheHandle)
		return
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

	if currentOffset-fch.prevOffset > MaxAllowedMB {
		return false
	}

	return true
}

// DoesContributeForJobRef returns the value of contributesToJobRefCount.
// Please check the documentation of CacheHandle.contributesToJobRefCount for more details.
func (fch *CacheHandle) DoesContributeForJobRef() bool {
	return fch.contributesToJobRefCount
}

// RemoveContributionForJobRef changes the state of contributesToJobRefCount, which
// signifies it doesn't depend on the job. We may cancel the job if no other reference
// to the download job.
func (fch *CacheHandle) RemoveContributionForJobRef() {
	fch.contributesToJobRefCount = false
}

// Close closes the underlined fileHandle points to locally downloaded data.
func (fch *CacheHandle) Close() (err error) {
	if fch.fileHandle != nil {
		err = fch.fileHandle.Close()
		fch.fileHandle = nil
	}

	return
}
