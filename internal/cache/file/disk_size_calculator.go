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

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	baseutil "github.com/googlecloudplatform/gcsfuse/v3/internal/util"
)

const (
	defaultFileCacheDiskSizeScanFrequency time.Duration = time.Minute
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

	// stopCh is used to signal the background goroutine to stop.
	stopCh chan struct{}
	// wg is used to wait for the background goroutine to exit.
	wg sync.WaitGroup
}

// NewFileCacheDiskUtilizationCalculator creates a new calculator and starts the
// background directory size calculation.
func NewFileCacheDiskUtilizationCalculator(cacheDir string, frequency time.Duration, includeFiles bool) *FileCacheDiskUtilizationCalculator {
	if frequency <= 0 {
		frequency = defaultFileCacheDiskSizeScanFrequency
	}
	c := &FileCacheDiskUtilizationCalculator{
		cacheDir:     cacheDir,
		frequency:    frequency,
		includeFiles: includeFiles,
		stopCh:       make(chan struct{}),
	}
	c.wg.Add(1)
	go c.monitorScannedSize()
	return c
}

func (c *FileCacheDiskUtilizationCalculator) monitorScannedSize() {
	defer c.wg.Done()
	ticker := time.NewTicker(c.frequency)
	defer ticker.Stop()

	// Initial calculation
	c.updateScannedSize()

	for {
		select {
		case <-ticker.C:
			c.updateScannedSize()
		case <-c.stopCh:
			return
		}
	}
}

func (c *FileCacheDiskUtilizationCalculator) updateScannedSize() {
	start := time.Now()
	// Recalculate size: if includeFiles is true, we scan everything (onlyDirs=false).
	// If includeFiles is false, we scan only directories (onlyDirs=true).
	s, err := baseutil.GetSizeOnDisk(c.cacheDir, !c.includeFiles, true)
	duration := time.Since(start)

	if err != nil {
		logger.Warnf("Failed to calculate disk usage for %q: %v", c.cacheDir, err)
		return
	}
	c.scannedSize.Store(s)
	logger.Debugf("Calculated disk usage for %q: %d bytes. Took %v. (includeFiles=%v)", c.cacheDir, s, duration, c.includeFiles)
}

func (c *FileCacheDiskUtilizationCalculator) Stop() {
	close(c.stopCh)
	c.wg.Wait()
}

func (c *FileCacheDiskUtilizationCalculator) GetCurrentSize() uint64 {
	if c.includeFiles {
		total := c.scannedSize.Load()
		logger.Debugf("Returning GetCurrentSize for file-cache: %v", total)
		return total
	}
	filesDiskUtilization := c.filesSize.Load()
	dirsDiskUtilization := c.scannedSize.Load()
	total := filesDiskUtilization + dirsDiskUtilization
	logger.Debugf("Returning GetCurrentSize for file-cache: files = %v, dirs = %v, total = %v", filesDiskUtilization, dirsDiskUtilization, total)
	return total
}

func (c *FileCacheDiskUtilizationCalculator) AccountForEvictedEntry(evictedEntry lru.ValueType) {
	c.filesSize.Add(-evictedEntry.Size())
	//logger.Debugf("file-cache's filesSize reduced to %v", c.filesSize.Load())
}

func (c *FileCacheDiskUtilizationCalculator) AccountForInsertedEntry(insertedEntry lru.ValueType) {
	c.filesSize.Add(insertedEntry.Size())
	//logger.Debugf("file-cache's filesSize increased to %v", c.filesSize.Load())
}

func (c *FileCacheDiskUtilizationCalculator) AccountForReplacedEntry(replacedEntry, newEntry lru.ValueType) {
	c.filesSize.Add(-replacedEntry.Size())
	c.filesSize.Add(newEntry.Size())
	//logger.Debugf("file-cache's filesSize changed to %v", c.filesSize.Load())
}

func (c *FileCacheDiskUtilizationCalculator) AddDeltaUsage(delta int64) {
	// Casting int64 to uint64 correctly handles negative values as 2's complement
	c.filesSize.Add(uint64(delta))
	//logger.Debugf("file-cache's filesSize changed to %v", c.filesSize.Load())
}
