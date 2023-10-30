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

	"github.com/googlecloudplatform/gcsfuse/internal/cache/file/downloader"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/gcs"
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
}

func NewCacheHandle(localFileHandle *os.File, fileDownloadJob *downloader.Job, fileInfoCache *lru.Cache, initialOffset int64) *CacheHandle {
	return &CacheHandle{
		fileHandle:      localFileHandle,
		fileDownloadJob: fileDownloadJob,
		fileInfoCache:   fileInfoCache,
		isSequential:    initialOffset == 0,
		prevOffset:      initialOffset,
	}
}

func (fch *CacheHandle) validateCacheHandle() error {

	if fch.fileHandle == nil {
		return errors.New(util.InvalidFileHandleErrMsg)
	}

	if fch.fileDownloadJob == nil {
		return errors.New(util.InvalidFileDownloadJobErrMsg)
	}

	if fch.fileInfoCache == nil {
		return errors.New(util.InvalidFileInfoCacheErrMsg)
	}

	return nil
}

// shouldReadFromLocalDownloadedFile returns nil if the data should be read from the local,
// downloaded file. Otherwise, it returns a non-nil error with an appropriate error message.
func (fch *CacheHandle) shouldReadFromLocalDownloadedFile(jobStatus *downloader.JobStatus, requiredOffset int64) (err error) {
	if jobStatus.Err != nil ||
		jobStatus.Name == downloader.INVALID ||
		jobStatus.Name == downloader.FAILED ||
		jobStatus.Name == downloader.NOT_STARTED {
		return errors.New(util.InvalidFileDownloadJobErrMsg)
	} else if jobStatus.Offset < requiredOffset {
		err := fmt.Errorf("%s: jobOffset: %d is less than required offset: %d", util.FallbackToGCSErrMsg, jobStatus.Offset, requiredOffset)
		return err
	}
	return err
}

// Read attempts to read the data from the cached location. This expects a
// fileInfoCache entry for the current read request, and will wait to download
// the requested chunk if it is not already present for sequential read.
// It doesn't wait, in case of random reads.
func (fch *CacheHandle) Read(ctx context.Context, object *gcs.MinObject, offset int64, dst []byte) (n int, err error) {
	err = fch.validateCacheHandle()
	if err != nil {
		return
	}

	if offset < 0 || offset > int64(object.Size) {
		return 0, fmt.Errorf("wrong offset requested: %d", offset)
	}

	// Checking before updating the previous offset.
	waitForDownload := true
	if !fch.IsSequential(offset) {
		fch.isSequential = false
		waitForDownload = false
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

	jobStatus, err := fch.fileDownloadJob.Download(ctx, requiredOffset, waitForDownload)
	if err != nil {
		n = 0
		err = fmt.Errorf("while downloading job: %v", err)
		return
	}

	if err = fch.shouldReadFromLocalDownloadedFile(&jobStatus, requiredOffset); err != nil {
		return 0, err
	}

	// We are here means, we have the data downloaded which kernel has asked for.
	_, err = fch.fileHandle.Seek(offset, 0)
	if err != nil {
		logger.Warnf("while seeking for %d offset in local file: %v", offset, err)
		return 0, errors.New(util.ErrInSeekingFileHandleMsg)
	}

	n, err = io.ReadFull(fch.fileHandle, dst)
	if err == io.EOF {
		err = nil
	}
	if err != nil {
		logger.Warnf("while reading from %d offset of the local file: %v", offset, err)
		return 0, errors.New(util.ErrInReadingFileHandleMsg)
	}

	// The job state may change while reading data from the local downloaded file. This may be
	// due to a failure in the download job, or cache eviction due to another file. In this case,
	// we will check the job state again after reading the data, and fall back to the normal flow
	// if it is not advisable to read the data from the local downloaded file.
	jobStatus = fch.fileDownloadJob.GetStatus()
	if err = fch.shouldReadFromLocalDownloadedFile(&jobStatus, requiredOffset); err != nil {
		return 0, err
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

// Close closes the underlined fileHandle points to locally downloaded data.
func (fch *CacheHandle) Close() (err error) {
	if fch.fileHandle != nil {
		err = fch.fileHandle.Close()
		fch.fileHandle = nil
	}

	return
}
