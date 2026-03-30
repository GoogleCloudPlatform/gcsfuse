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

package data

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/util/diskutil"
)

const InvalidKeyAttributes = "key attributes not initialised"

type FileInfoKey struct {
	BucketName         string
	BucketCreationTime time.Time
	ObjectName         string
}

// Key will return a string, combining all the attributes of FileInfoKey.
// Returns error in case of uninitialized value.
func (fik FileInfoKey) Key() (string, error) {
	return GetFileInfoKeyName(fik.ObjectName, fik.BucketCreationTime, fik.BucketName)
}

func GetFileInfoKeyName(objectName string, bucketCreationTime time.Time, bucketName string) (string, error) {
	if bucketName == "" || objectName == "" {
		return "", errors.New(InvalidKeyAttributes)
	}
	unixTimeString := strconv.FormatInt(bucketCreationTime.Unix(), 10)
	return bucketName + unixTimeString + objectName, nil
}

type FileInfo struct {
	Key              FileInfoKey
	ObjectGeneration int64
	Offset           uint64 // For non-sparse files: bytes downloaded so far. For sparse files: set to MaxUint64 as sentinel
	FileSize         uint64
	SparseMode       bool          // Whether this file is using sparse file mode
	DownloadedChunks *ByteRangeMap // For sparse files: tracks which chunks have been downloaded
	// CacheDirVolumeBlockSize is used to round-up the FileSize to calculate the speculative
	// disk utilization of this file in cache-directory.
	// Speculative size = Round-up of FileSize to the next multiple of CacheDirVolumeBlockSize.
	// 0 or 1 mean size in cache-dir is same as FileSize.
	CacheDirVolumeBlockSize uint64
}

// ContentSize returns the logical size of the given file, or in other words, the size
// of the corresponding GCS object.
func (fi FileInfo) ContentSize() uint64 {
	// For sparse files, return actual downloaded bytes, not full file size
	if fi.SparseMode && fi.DownloadedChunks != nil {
		return fi.DownloadedChunks.TotalBytes()
	}
	return fi.FileSize
}

// Size returns the speculative physical size on disk, rounded up to the volume block size.
// If CacheDirVolumeBlockSize is 0 or 1, it returns the exact logical ContentSize.
// This satisfies the LRU ValueType interface for eviction accounting.
func (fi FileInfo) Size() uint64 {
	return diskutil.GetSpeculativeFileSizeOnDisk(fi.ContentSize(), fi.CacheDirVolumeBlockSize)
}

type FileSpec struct {
	Path     string
	FilePerm os.FileMode
	DirPerm  os.FileMode
}

// NewFileInfo creates and returns a new FileInfo struct, ensuring that
// the CacheDirVolumeBlockSize field is explicitly provided.
func NewFileInfo(key FileInfoKey, objectGeneration int64, fileSize uint64, offset uint64, sparseMode bool, downloadedChunks *ByteRangeMap, cacheDirVolumeBlockSize uint64) FileInfo {
	return FileInfo{
		Key:                     key,
		ObjectGeneration:        objectGeneration,
		FileSize:                fileSize,
		Offset:                  offset,
		SparseMode:              sparseMode,
		DownloadedChunks:        downloadedChunks,
		CacheDirVolumeBlockSize: cacheDirVolumeBlockSize,
	}
}
