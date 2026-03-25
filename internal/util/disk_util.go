// Copyright 2026 Google LLC
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

package util

import (
	"syscall"
)

// DefaultCacheDirVolumeBlockSize is the block-size used for cache-dir in case
// statfs call fails for it.
const DefaultVolumeBlockSize = 4096

// GetSpeculativeFileSizeOnDisk calculates the theoretical disk space a file will
// consume given its actual content size and the filesystem's block size. It rounds
// up the content size to the nearest block boundary to simulate block allocation.
func GetSpeculativeFileSizeOnDisk(fileContentSize, volumeBlockSize uint64) uint64 {
	if volumeBlockSize <= 1 {
		return fileContentSize
	}
	numBlocks := (fileContentSize + volumeBlockSize - 1) / volumeBlockSize
	return numBlocks * volumeBlockSize
}

// GetVolumeBlockSize retrieves the block size of the file system containing the given path
// using statfs sys-call. The block-size can be 0 or 2^n. The most common value is 4096.
func GetVolumeBlockSize(path string) uint64 {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return DefaultVolumeBlockSize
	}
	// Bsize is int64, casting it to uint64.
	return uint64(stat.Bsize)
}
