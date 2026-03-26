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

package diskutil

import (
	"syscall"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
)

const (
	// defaultVolumeBlockSize is the block-size used if statfs fails.
	// 4 KiB
	defaultVolumeBlockSize uint64 = 4096

	// maxVolumeBlockSize is the max block-size supported for sanity. Beyond this, block-size returned by statfs.Bsize will be considered suspiciously large and will be truncated.
	// 1 MiB
	maxVolumeBlockSize uint64 = 1024 * 1024
)

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
		logger.Errorf("statsfs failed for %q: %v. Defaulting to block-size %d for this directory.", path, err, defaultVolumeBlockSize)
		return defaultVolumeBlockSize
	}
	// Prefer Frsize (fragment size) over Bsize for actual disk allocation if available.
	blockSize := uint64(stat.Bsize)
	if stat.Frsize > 0 {
		blockSize = uint64(stat.Frsize)
	}
	// Sanity check: If the value is 0 or suspiciously large, fallback to default or max
	if blockSize == 0 {
		logger.Errorf("statfs for %q returned Bsize = 0, so defaulting to %d", path, defaultVolumeBlockSize)
		return defaultVolumeBlockSize
	}
	if blockSize > maxVolumeBlockSize {
		logger.Errorf("statfs for %q returned Bsize (%d), which is too high, so truncating it to %d", path, blockSize, maxVolumeBlockSize)
		return maxVolumeBlockSize
	}
	return blockSize
}
