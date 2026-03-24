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
	"fmt"
	"syscall"
)

// GetSpeculativeFileSizeOnDisk calculates the theoretical disk space a file will
// consume given its actual content size and the filesystem's block size. It rounds
// up the content size to the nearest block boundary to simulate block allocation.
func GetSpeculativeFileSizeOnDisk(fileContentSize, volumeBlockSize uint64) uint64 {
	if volumeBlockSize == 0 {
		return 0
	}
	numBlocks := (fileContentSize + volumeBlockSize - 1) / volumeBlockSize
	return numBlocks * volumeBlockSize
}

// GetVolumeBlockSize retrieves the block size of the file system containing the given path
// using statfs sys-call. The block-size can be 0 or 2^n. The most common value is 4096.
func GetVolumeBlockSize(path string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, fmt.Errorf("failed to get stats for path %q: %w", path, err)
	}
	// Bsize is int64 on some platforms (like Linux/amd64) and uint32 on others (like Linux/arm).
	// Casting to uint64 ensures consistency.
	return uint64(stat.Bsize), nil
}
