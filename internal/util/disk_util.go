// Copyright 2024 Google LLC
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

// getDiskUsage extracts the allocated size from the system specific info
func getDiskUsage(entry fs.DirEntry) int64 {
	info, err := entry.Info()
	if err != nil {
		return 0
	}

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

func walkDir(dir string, wg *sync.WaitGroup, sem chan struct{}, totalSizeOnDisk *int64, onlyDir bool) {
	defer wg.Done()
	sem <- struct{}{}
	defer func() { <-sem }()

	entries, err := os.ReadDir(dir)
	if err != nil {
		// Remember: Permission errors are common in DU, handle or log as needed
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			wg.Add(1)
			go walkDir(filepath.Join(dir, entry.Name()), wg, sem, totalSizeOnDisk, onlyDir)

			// Directories themselves also take up space on disk (metadata)
			// We should count the directory's own block usage too.
			atomic.AddInt64(totalSizeOnDisk, getDiskUsage(entry))
		} else if !onlyDir {
			atomic.AddInt64(totalSizeOnDisk, getDiskUsage(entry))
		}
	}
}

func GetSizeOnDisk(dirPath string, onlyDirs bool) (uint64, error) {
	var sizeOnDisk int64
	var sem = make(chan struct{}, 32)
	var wg sync.WaitGroup
	wg.Add(1)
	walkDir(dirPath, &wg, sem, &sizeOnDisk, onlyDirs)
	wg.Wait()
	if sizeOnDisk < 0 {
		return 0, fmt.Errorf("Something failed while calculating disk utilization of %q", dirPath)
	}
	return (uint64)(sizeOnDisk), nil
}

func GetSpeculativeFileSizeOnDisk(fileSize uint64) uint64 {
	blockSize := uint64(4096)
	numBlocks := (fileSize + blockSize - 1) / blockSize
	return numBlocks * blockSize
}
