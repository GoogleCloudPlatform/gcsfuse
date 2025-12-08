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

type FileCacheDiskUtilizationCalculator struct {
	// filesSize tracks the size of files in the cache.
	filesSize atomic.Uint64
	// dirsSize tracks the size of directories in the cache.
	dirsSize atomic.Uint64
	// cacheDir is the directory path of the cache.
	cacheDir string
	// frequency is the duration after which dirsSize is recalculated.
	frequency time.Duration

	// stopCh is used to signal the background goroutine to stop.
	stopCh chan struct{}
	// wg is used to wait for the background goroutine to exit.
	wg sync.WaitGroup
}

// NewFileCacheDiskUtilizationCalculator creates a new calculator and starts the
// background directory size calculation.
func NewFileCacheDiskUtilizationCalculator(cacheDir string, frequency time.Duration) *FileCacheDiskUtilizationCalculator {
	c := &FileCacheDiskUtilizationCalculator{
		cacheDir:  cacheDir,
		frequency: frequency,
		stopCh:    make(chan struct{}),
	}
	c.wg.Add(1)
	go c.monitorDirSize()
	return c
}

func (c *FileCacheDiskUtilizationCalculator) monitorDirSize() {
	defer c.wg.Done()
	ticker := time.NewTicker(c.frequency)
	defer ticker.Stop()

	// Initial calculation
	c.updateDirSize()

	for {
		select {
		case <-ticker.C:
			c.updateDirSize()
		case <-c.stopCh:
			return
		}
	}
}

func (c *FileCacheDiskUtilizationCalculator) updateDirSize() {
	// Recalculate directories size: onlyDirs=true, ignoreErrors=true
	s, err := baseutil.GetSizeOnDisk(c.cacheDir, true, true)
	if err != nil {
		logger.Warnf("Failed to calculate directory size for %q: %v", c.cacheDir, err)
		return
	}
	c.dirsSize.Store(s)
}

func (c *FileCacheDiskUtilizationCalculator) Stop() {
	close(c.stopCh)
	c.wg.Wait()
}

func (c *FileCacheDiskUtilizationCalculator) GetCurrentSize() uint64 {
	return c.filesSize.Load() + c.dirsSize.Load()
}

func (c *FileCacheDiskUtilizationCalculator) AccountForEvictedEntry(evictedEntry lru.ValueType) {
	c.filesSize.Add(-evictedEntry.Size())
}

func (c *FileCacheDiskUtilizationCalculator) AccountForInsertedEntry(insertedEntry lru.ValueType) {
	c.filesSize.Add(insertedEntry.Size())
}

func (c *FileCacheDiskUtilizationCalculator) AccountForReplacedEntry(replacedEntry, newEntry lru.ValueType) {
	c.filesSize.Add(-replacedEntry.Size())
	c.filesSize.Add(newEntry.Size())
}

func (c *FileCacheDiskUtilizationCalculator) AddDeltaUsage(delta int64) {
	// Casting int64 to uint64 correctly handles negative values as 2's complement
	c.filesSize.Add(uint64(delta))
}
