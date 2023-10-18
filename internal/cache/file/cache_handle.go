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
	"github.com/googlecloudplatform/gcsfuse/internal/storage/gcs"
)

const InvalidFileHandle = "fileHandle is nil"
const InvalidFileDownloadJob = "download job is nil"
const InvalidFileInfoCache = "fileInfo cache is nil"

const InvalidFileInfo = "fileInfo is nil"

type CacheHandle struct {
	fileHandle      *os.File
	fileDownloadJob *downloader.Job
	fileInfoCache   *lru.Cache
}

// validateCacheHandle will validate the  cache-handle and return appropriate error.
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

// Expects the fileInfoCache entry for the read file.
// This will serve one kernel request.
func (fch *CacheHandle) Read(object *gcs.MinObject, bucket gcs.Bucket, offset uint64, dst []byte) (n int, err error) {

	err = fch.validateCacheHandle()
	if err != nil {
		return
	}

	// TODO (princer) - Get the actual bucket creation time. Ideally we will fetch using the bucket object.
	bucketCreationTime := time.Unix(data.TestTimeInEpoch, 0)
	fileInfoKey := data.FileInfoKey{ObjectName: object.Name, BucketName: bucket.Name(), BucketCreationTime: bucketCreationTime}
	fileInfoKeyName, err := fileInfoKey.Key()
	if err != nil {
		return
	}

	fileInfo := fch.fileInfoCache.LookUp(fileInfoKeyName)
	if fileInfo == nil {
		err = errors.New(InvalidFileInfo)
		return
	}

	// We need to download the data till offset + len(dst), if not already.
	bufferLen := len(dst)
	requiredDataOffset := fileInfo.(data.FileInfo).Offset + uint64(bufferLen)
	if requiredDataOffset < offset {
		ctx := context.Background()
		jobStatus := fch.fileDownloadJob.Download(ctx, int64(requiredDataOffset), true)

		if jobStatus.Err != nil {
			err = fmt.Errorf("error while downloading the data: %v", jobStatus.Err)
			return
		}
	}

	// We are here means, we have the data downloaded which kernel asked for.
	_, err = fch.fileHandle.Seek(int64(offset), 0)
	n, err = io.ReadFull(fch.fileHandle, dst)

	if err == io.EOF {
		err = nil
	}
	return
}
