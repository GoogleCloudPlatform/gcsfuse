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
	"runtime"
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
	semSize := (runtime.NumCPU() + 4) / 5
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
//
// This function uses a post-order traversal to ensure that directories which become
// empty after their subdirectories are removed are also cleaned up.
func RemoveEmptyDirs(dir string) {
	removeEmptyDirs(dir)
}

// removeEmptyDirs recursively attempts to remove empty directories.
// It returns true if the directory is effectively empty (contains no files and
// all subdirectories were successfully removed), and false otherwise.
func removeEmptyDirs(dir string) bool {
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
			childEmpty := removeEmptyDirs(fullPath)
			
			if childEmpty {
				// If the child directory is empty (or became empty), attempt to remove it.
				err := os.Remove(fullPath)
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

// PrettyPrintOf takes a uint64 number and returns a command-separated number string.
// e.g. input 12345678 and output "12,345,678"
func PrettyPrintOf(n uint64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

// GetSizeOnDiskAndClean calculates the size of the directory and optionally removes empty directories.
// It combines the functionality of GetSizeOnDisk (with ignoreErrors=true) and RemoveEmptyDirs.
func GetSizeOnDiskAndClean(dirPath string, includesFilesInSize bool, deleteEmptyDirs bool) uint64 {
	size, _ := getSizeOnDiskAndClean(dirPath, includesFilesInSize, deleteEmptyDirs)
	return size
}

func getSizeOnDiskAndClean(dirPath string, includesFilesInSize bool, deleteEmptyDirs bool) (uint64, bool) {
	var sizeOnDisk int64
	// Add the size of the root directory itself.
	info, err := os.Stat(dirPath)
	if err == nil {
		sizeOnDisk = getDiskUsageFromInfo(info)
	}

	// If we can't read the directory, we assume it's not empty (unsafe to delete)
	// and return its own size. This prevents deleting directories we don't have
	// permissions to inspecting, while still accounting for their metadata size.
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return uint64(sizeOnDisk), false
	}

	isEmpty := true

	for _, entry := range entries {
		if entry.IsDir() {
			subDir := filepath.Join(dirPath, entry.Name())
			subDirSize, subDirEmpty := getSizeOnDiskAndClean(subDir, includesFilesInSize, deleteEmptyDirs)

			// Attempt removal only if requested AND the subdirectory is effectively empty.
			if deleteEmptyDirs && subDirEmpty {
				err := os.Remove(subDir)
				if err != nil {
					// Failed to remove (e.g. permissions, race condition), so it remains on disk.
					// Count its size and mark the current parent as not empty.
					sizeOnDisk += int64(subDirSize)
					isEmpty = false
				}
				// If removal succeeded, we don't add its size to the total, effectively "cleaning" it.
			} else {
				// Subdirectory is not empty (or deletion disabled), so it persists.
				sizeOnDisk += int64(subDirSize)
				isEmpty = false
			}
		} else {
			// File exists, so this directory is not empty.
			isEmpty = false
			if includesFilesInSize {
				sizeOnDisk += getDiskUsage(entry)
			}
		}
	}

	// Defensive check: Ensure sizeOnDisk didn't wrap around to negative due to overflow
	// (unlikely for local cache sizes < 8 EiB).
	if sizeOnDisk < 0 {
		return 0, isEmpty
	}
	return uint64(sizeOnDisk), isEmpty
}
