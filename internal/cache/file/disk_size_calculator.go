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

package file

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	baseutil "github.com/googlecloudplatform/gcsfuse/v3/internal/util"
)

const (
	DefaultFileCacheSizeScanFrequencySeconds = 10
)

type FileCacheDiskUtilizationCalculator struct {
	// filesSize tracks the size of files in the cache.
	filesSize atomic.Uint64
	// scannedSize tracks the size calculated from disk scan (directories or full).
	scannedSize atomic.Uint64
	// cacheDir is the directory path of the cache.
	cacheDir string
	// frequency is the duration after which scannedSize is recalculated.
	frequency time.Duration
	// includeFiles determines if the disk scan includes files (true) or just directories (false).
	includeFiles bool
	// deleteEmptyDirs determines if empty directories are deleted during scan.
	deleteEmptyDirs bool
	// volumeBlockSize stores the block size of the volume.
	volumeBlockSize uint64

	// stopCh is used to signal the background goroutine to stop.
	stopCh chan struct{}
	// wg is used to wait for the background goroutine to exit.
	wg sync.WaitGroup
}

// NewFileCacheDiskUtilizationCalculator creates a new calculator and starts the
// background directory size calculation.
func NewFileCacheDiskUtilizationCalculator(cacheDir string, frequency time.Duration, includeFiles bool, deleteEmptyDirs bool, volumeBlockSize uint64) *FileCacheDiskUtilizationCalculator {
	if frequency <= 0 {
		frequency = time.Duration(DefaultFileCacheSizeScanFrequencySeconds) * time.Second
	}
	c := &FileCacheDiskUtilizationCalculator{
		cacheDir:        cacheDir,
		frequency:       frequency,
		includeFiles:    includeFiles,
		deleteEmptyDirs: deleteEmptyDirs,
		volumeBlockSize: volumeBlockSize,
		stopCh:          make(chan struct{}),
	}
	c.wg.Add(1)
	go c.periodicSizeScan()
	return c
}

func (c *FileCacheDiskUtilizationCalculator) periodicSizeScan() {
	defer c.wg.Done()
	ticker := time.NewTicker(c.frequency)
	defer ticker.Stop()

	// Initial calculation
	c.clearEmptyDirsAndRescanSize()

	for {
		select {
		case <-ticker.C:
			c.clearEmptyDirsAndRescanSize()
		case <-c.stopCh:
			return
		}
	}
}

func (c *FileCacheDiskUtilizationCalculator) clearEmptyDirsAndRescanSize() {
	start := time.Now()

	// First, remove empty directories. We ignore errors here.
	if c.deleteEmptyDirs {
		baseutil.RemoveEmptyDirs(c.cacheDir)
	}

	// Recalculate size: if includeFiles is true, we scan everything (onlyDirs=false).
	// If includeFiles is false, we scan only directories (onlyDirs=true).
	s, err := baseutil.GetSizeOnDisk(c.cacheDir, !c.includeFiles, true)
	duration := time.Since(start)

	if err != nil {
		logger.Warnf("Failed to calculate disk usage for %q: %v", c.cacheDir, err)
		return
	}
	c.scannedSize.Store(s)

	// Debugging data
	filesSize := c.filesSize.Load()
	total := s
	if !c.includeFiles {
		total += filesSize
	}
	logger.Debugf("Calculated disk usage for %q: %s bytes. Took %v. (includeFiles=%v). filesSize=%s, total=%s", c.cacheDir, baseutil.PrettyPrintOf(s), duration, c.includeFiles, baseutil.PrettyPrintOf(filesSize), baseutil.PrettyPrintOf(total))
}

// Stop stops the periodic size scanner.
func (c *FileCacheDiskUtilizationCalculator) Stop() {
	close(c.stopCh)
	c.wg.Wait()
}

// GetCurrentSize returns the latest size of the file-cache cache-dir
// returned on the last scan.
// If the scanned size included files, then this is the latest scanned size directly,
// otherwise it adds the incremently updated size for all files and returns that.
func (c *FileCacheDiskUtilizationCalculator) GetCurrentSize() uint64 {
	if c.includeFiles {
		total := c.scannedSize.Load()
		logger.Tracef("GetCurrentSize for file-cache: %s", baseutil.PrettyPrintOf(total))
		return total
	}
	filesDiskUtilization := c.filesSize.Load()
	dirsDiskUtilization := c.scannedSize.Load()
	total := filesDiskUtilization + dirsDiskUtilization
	logger.Tracef("GetCurrentSize for file-cache: files = %s, dirs = %s, total = %s", baseutil.PrettyPrintOf(filesDiskUtilization), baseutil.PrettyPrintOf(dirsDiskUtilization), baseutil.PrettyPrintOf(total))
	return total
}

// SizeOf returns the actual disk utilization of the underlying cached file.
func (c *FileCacheDiskUtilizationCalculator) SizeOf(entry lru.ValueType) uint64 {
	if fi, ok := entry.(data.FileInfo); ok {
		return baseutil.GetSpeculativeFileSizeOnDisk(fi.Size(), c.volumeBlockSize)
	}
	return entry.Size()
}

// EvictEntry subtracts the size for the given entry.
func (c *FileCacheDiskUtilizationCalculator) EvictEntry(evictedEntry lru.ValueType) {
	c.filesSize.Add(-c.SizeOf(evictedEntry))
}

// EvictEntry adds the size for the given entry.
func (c *FileCacheDiskUtilizationCalculator) InsertEntry(insertedEntry lru.ValueType) {
	c.filesSize.Add(c.SizeOf(insertedEntry))
}

// AddDelta directly add the given delta to the stored files' size.
// It can be negative as well, in which case it reduces the existing stored files' size.
// If it's negative, the value of stored files' size is not allowed to go below 0.
func (c *FileCacheDiskUtilizationCalculator) AddDelta(delta int64) {
	if delta < 0 {
		negDelta := uint64(-delta)
		if negDelta > c.filesSize.Load() {
			c.filesSize.Store(0)
		} else {
			c.filesSize.Add(-negDelta)
		}
	} else {
		c.filesSize.Add(uint64(delta))
	}
}
