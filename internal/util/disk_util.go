// Copyright 2025 Google LLC
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
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
)

// getDiskUsageFromInfo extracts the allocated size from the system specific info
func getDiskUsageFromInfo(info fs.FileInfo) int64 {
	// Retrieve the underlying system-specific stats
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		// Fallback for non-Unix systems (e.g., Windows)
		// On Windows, Size() is usually the best approximation available
		// without complex API calls.
		return info.Size()
	}

	// ST_BLOCKS is always in units of 512 bytes on Linux/Unix
	return int64(stat.Blocks) * 512
}

// getDiskUsage extracts the allocated size from the system specific info
func getDiskUsage(entry fs.DirEntry) int64 {
	info, err := entry.Info()
	if err != nil {
		return 0
	}

	return getDiskUsageFromInfo(info)
}

func GetSizeOnDisk(dirPath string, onlyDirs bool, ignoreErrors bool) (uint64, error) {
	var sizeOnDisk int64
	var sem = make(chan struct{}, 32)
	var wg sync.WaitGroup
	var errMu sync.Mutex

	// firstErr captures the first error encountered during the concurrent walk.
	// It is only used/populated when ignoreErrors is false.
	var firstErr error

	// Add the size of the root directory itself
	info, err := os.Stat(dirPath)
	if err != nil {
		if !ignoreErrors {
			return 0, fmt.Errorf("failed to stat root dir %q: %w", dirPath, err)
		}
	} else {
		sizeOnDisk = getDiskUsageFromInfo(info)
	}

	var walkDir func(dir string)
	walkDir = func(dir string) {
		defer wg.Done()
		sem <- struct{}{}
		defer func() { <-sem }()

		if !ignoreErrors {
			errMu.Lock()
			if firstErr != nil {
				errMu.Unlock()
				return
			}
			errMu.Unlock()
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			if !ignoreErrors {
				errMu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("failed to read dir %q: %w", dir, err)
				}
				errMu.Unlock()
			}
			return
		}

		for _, entry := range entries {
			if entry.IsDir() {
				wg.Add(1)
				go walkDir(filepath.Join(dir, entry.Name()))

				// Directories themselves also take up space on disk (metadata)
				// We should count the directory's own block usage too.
				atomic.AddInt64(&sizeOnDisk, getDiskUsage(entry))
			} else if !onlyDirs {
				atomic.AddInt64(&sizeOnDisk, getDiskUsage(entry))
			}
		}
	}

	wg.Add(1)
	walkDir(dirPath)
	wg.Wait()

	if firstErr != nil {
		return 0, firstErr
	}

	if sizeOnDisk < 0 {
		return 0, fmt.Errorf("Something failed while calculating disk utilization of %q", dirPath)
	}
	return uint64(sizeOnDisk), nil
}

func GetSpeculativeFileSizeOnDisk(fileContentSize uint64, volumeBlockSize uint64) uint64 {
	if volumeBlockSize == 0 {
		return 0
	}
	numBlocks := (fileContentSize + volumeBlockSize - 1) / volumeBlockSize
	return numBlocks * volumeBlockSize
}

// GetVolumeBlockSize retrieves the block size of the file system containing the given path.
// It returns the block size in bytes (e.g., 4096).
func GetVolumeBlockSize(path string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, fmt.Errorf("failed to get stats for path %q: %w", path, err)
	}
	// Bsize is int64 on some platforms (like Linux/amd64) and uint32 on others (like Linux/arm).
	// Casting to uint64 ensures consistency.
	return uint64(stat.Bsize), nil
}

// RemoveEmptyDirs recursively removes all empty subdirectories within the given directory.
// It does not remove the given directory itself.
func RemoveEmptyDirs(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			fullPath := filepath.Join(dir, entry.Name())
			RemoveEmptyDirs(fullPath)
			_ = os.Remove(fullPath)
		}
	}
}
