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
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
)

// The percentage of total CPUs to be used for the concurrent disk size calculation.
const cpuUtilizationPercentageForDiskSizeCalculation = 0.20

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

// DirLocker is an interface for locking directories during traversal or modification.
type DirLocker interface {
	// ReadLock should be invoked before listing/creating/deleting any files in the given directory.
	ReadLock(path string)
	// ReadUnlock should be invoked to undo ReadLock.
	ReadUnlock(path string)
	// WriteLock should be invoked before deleting/renaming the passed directory.
	WriteLock(path string)
	// WriteUnlock should be invoked to undo WriteLock.
	WriteUnlock(path string)
}

func GetSizeOnDisk(dirPath string, onlyDirs bool, ignoreErrors bool) (uint64, error) {
	return GetSizeOnDiskWithLocker(dirPath, onlyDirs, ignoreErrors, nil)
}

func GetSizeOnDiskWithLocker(dirPath string, onlyDirs bool, ignoreErrors bool, locker DirLocker) (uint64, error) {
	var sizeOnDisk int64
	semSize := int(math.Ceil(float64(runtime.NumCPU()) * cpuUtilizationPercentageForDiskSizeCalculation))
	if semSize < 1 {
		semSize = 1
	}
	var sem = make(chan struct{}, semSize)
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

		if locker != nil {
			locker.ReadLock(dir)
			defer locker.ReadUnlock(dir)
		}

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
		return 0, fmt.Errorf("disk utilization calculation resulted in a negative value for %q: %d", dirPath, sizeOnDisk)
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
//
// This function uses a post-order traversal to ensure that directories which become
// empty after their subdirectories are removed are also cleaned up.
func RemoveEmptyDirs(dir string) {
	removeEmptyDirs(dir, nil)
}

// RemoveEmptyDirsWithLocker recursively removes all empty subdirectories, executing
// the provided locker's WriteLock before attempting to remove a directory.
func RemoveEmptyDirsWithLocker(dir string, locker DirLocker) {
	removeEmptyDirs(dir, locker)
}

// removeEmptyDirs recursively attempts to remove empty directories.
// It returns true if the directory is effectively empty (contains no files and
// all subdirectories were successfully removed), and false otherwise.
func removeEmptyDirs(dir string, locker DirLocker) bool {
	// ReadDir can only compete with the cached file creation/deletion operations
	// which shouldn't have a destructive impact on the ReadDir call, so we need not lock it.
	entries, err := os.ReadDir(dir)
	if err != nil {
		// If we can't read the directory, we assume it's not safe to consider it empty.
		return false
	}

	isEmpty := true
	for _, entry := range entries {
		if entry.IsDir() {
			fullPath := filepath.Join(dir, entry.Name())
			// Recurse first (post-order traversal).
			childEmpty := removeEmptyDirs(fullPath, locker)

			if childEmpty {
				// If the child directory is empty (or became empty), attempt to remove it.
				if locker != nil {
					locker.WriteLock(fullPath)
					err = os.Remove(fullPath)
					locker.WriteUnlock(fullPath)
				} else {
					err = os.Remove(fullPath)
				}

				if err != nil {
					// Failed to remove (e.g. permissions), so this directory is not effectively empty.
					isEmpty = false
				}
			} else {
				// Child directory is not empty, so this directory cannot be empty.
				isEmpty = false
			}
		} else {
			// Found a file, so this directory is not empty.
			isEmpty = false
		}
	}
	return isEmpty
}
