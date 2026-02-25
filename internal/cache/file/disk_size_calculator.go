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

package file

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	cacheutil "github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	baseutil "github.com/googlecloudplatform/gcsfuse/v3/internal/util"
)

const (
	DefaultFileCacheSizeScanFrequencySeconds = 10
)

type FileCacheDiskUtilizationCalculator struct {
	// filesSize tracks the size of files in the cache. It is a non-atomic uint64
	// because all its reads and writes (via InsertEntry, EvictEntry, AddDelta,
	// GetCurrentSize) are exclusively invoked by the lru.Cache, which already
	// holds its own global mutex during these operations. Avoiding atomics/mutexes
	// here prevents severe cache-line bouncing and lock contention under high
	// concurrency (e.g., 50+ threads).
	filesSize uint64

	// fileCacheFileOpsMu is a pointer to the LRU cache's mutex. It is provided from the outside
	// (via SetLRUMutex) so that the background periodic scanner can safely read
	// filesSize for debug logging without introducing a separate, redundant mutex
	// into the hot path of the calculator.
	fileCacheFileOpsMu *locker.RWLocker

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

	// sharedDirLocker is used to acquire locks on directories during scanning and deletion.
	sharedDirLocker *cacheutil.SharedDirLocker

	// stopCh is used to signal the background goroutine to stop.
	stopCh chan struct{}
	// wg is used to wait for the background goroutine to exit.
	wg sync.WaitGroup
}

// NewFileCacheDiskUtilizationCalculator creates a new calculator and starts the
// background directory size calculation.
func NewFileCacheDiskUtilizationCalculator(cacheDir string, frequency time.Duration, deleteEmptyDirs bool, volumeBlockSize uint64) *FileCacheDiskUtilizationCalculator {
	if frequency <= 0 {
		frequency = time.Duration(DefaultFileCacheSizeScanFrequencySeconds) * time.Second
	}
	c := &FileCacheDiskUtilizationCalculator{
		cacheDir:        cacheDir,
		frequency:       frequency,
		includeFiles:    false,
		deleteEmptyDirs: deleteEmptyDirs,
		volumeBlockSize: volumeBlockSize,
		stopCh:          make(chan struct{}),
	}
	c.wg.Add(1)
	go c.periodicSizeScan()
	return c
}

func (c *FileCacheDiskUtilizationCalculator) SetSharedDirLocker(sharedDirLocker *cacheutil.SharedDirLocker) error {
	if c.sharedDirLocker != nil {
		return fmt.Errorf("sharedDirLocker is already set")
	}
	c.sharedDirLocker = sharedDirLocker
	return nil
}

// SetLRUMutex sets the mutex from the LRU cache to allow safe reads of filesSize by the background scanner.
func (c *FileCacheDiskUtilizationCalculator) SetLRUMutex(mu *locker.RWLocker) error {
	if c.fileCacheFileOpsMu != nil {
		return fmt.Errorf("fileCacheFileOpsMu is already set")
	}
	c.fileCacheFileOpsMu = mu
	return nil
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
	// Use sharedDirLocker if available, otherwise nil (no locking).
	// Note: (*SharedDirLocker)(nil) satisfies the interface but we want the interface value to be nil if the pointer is nil.
	var locker baseutil.DirLocker
	if c.sharedDirLocker != nil {
		locker = c.sharedDirLocker
	}

	start := time.Now()

	// 1. Remove empty directories if enabled
	if c.deleteEmptyDirs {
		baseutil.RemoveEmptyDirsWithLocker(c.cacheDir, locker)
	}

	// 2. Calculate size on disk (using parallel traversal)
	// GetSizeOnDisk(dirPath, onlyDirs, ignoreErrors)
	// includeFiles in Calculator means we want file sizes.
	// onlyDirs in GetSizeOnDisk means "count ONLY directories".
	// So if c.includeFiles is true, onlyDirs should be false.
	// We ignore errors to match best-effort behavior.
	s, err := baseutil.GetSizeOnDiskWithLocker(c.cacheDir, !c.includeFiles, true, locker)
	if err != nil {
		logger.Warnf("Failed to calculate disk usage for %q: %v", c.cacheDir, err)
	}

	duration := time.Since(start)

	c.scannedSize.Store(s)

	// Debugging data
	var filesSize uint64
	if c.fileCacheFileOpsMu != nil {
		// Acquire the file-cache's read lock to safely read the non-atomic filesSize.
		// This prevents data races with concurrent Insert/Evict operations happening
		// in the foreground.
		(*c.fileCacheFileOpsMu).RLock()
		filesSize = c.filesSize
		(*c.fileCacheFileOpsMu).RUnlock()
	}
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
	filesDiskUtilization := c.filesSize
	dirsDiskUtilization := c.scannedSize.Load()
	total := filesDiskUtilization + dirsDiskUtilization
	logger.Tracef("GetCurrentSize for file-cache: files = %s, dirs = %s, total = %s", baseutil.PrettyPrintOf(filesDiskUtilization), baseutil.PrettyPrintOf(dirsDiskUtilization), baseutil.PrettyPrintOf(total))
	return total
}

// SizeOf returns the actual disk utilization of the underlying cached file.
func (c *FileCacheDiskUtilizationCalculator) SizeOf(entry lru.ValueType) uint64 {
	if fi, ok := entry.(data.FileInfo); ok {
		if fi.SparseMode {
			return baseutil.GetSpeculativeFileSizeOnDisk(fi.Size(), c.volumeBlockSize)
		}
		return baseutil.GetSpeculativeFileSizeOnDisk(fi.FileSize, c.volumeBlockSize)
	}
	return entry.Size()
}

// EvictEntry subtracts the size for the given entry.
func (c *FileCacheDiskUtilizationCalculator) EvictEntry(evictedEntry lru.ValueType) {
	size := c.SizeOf(evictedEntry)
	if c.filesSize > size {
		c.filesSize -= size
	} else {
		c.filesSize = 0
	}
}

// EvictEntry adds the size for the given entry.
func (c *FileCacheDiskUtilizationCalculator) InsertEntry(insertedEntry lru.ValueType) {
	c.filesSize += c.SizeOf(insertedEntry)
}

// AddDelta directly add the given delta to the stored files' size.
// It can be negative as well, in which case it reduces the existing stored files' size.
// If it's negative, the value of stored files' size is not allowed to go below 0.
func (c *FileCacheDiskUtilizationCalculator) AddDelta(delta int64) {
	if delta < 0 {
		negDelta := uint64(-delta)
		if negDelta > c.filesSize {
			c.filesSize = 0
		} else {
			c.filesSize -= negDelta
		}
	} else {
		c.filesSize += uint64(delta)
	}
}
