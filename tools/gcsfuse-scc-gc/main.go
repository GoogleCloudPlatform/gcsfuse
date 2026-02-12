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

/**
* How does gcsfuse-scc-gc (GCSFuse Shared Chunk Cache Garbage Collector) work?
*
* 1. Performs a single walk to scan the cache directory, collecting:
*    - `.bin` files (cache chunks) with atime/mtime and size
*    - `.bak` files (previously expired files from last run)
*    - `.tmp` files (incomplete downloads)
*    - Directories (for cleanup)
* 2. Cleans up `.bak` files expired during the previous run.
* 3. Sorts `.bin` files by atime and selects oldest files to expire, only if cache size exceeds target.
* 4. Renames selected files to `.bak` (kept until next run for ongoing reads).
* 5. Removes old `.tmp` files (older than 1 hour).
* 6. Cleans up empty directories (deepest first).
 */

package main

import (
	"errors"
	"flag"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	cacheDir    = flag.String("cache-dir", "", "Path to the cache directory")
	targetSize  = flag.Int64("target-size-mb", 10240, "Target cache size in MB (default: 10GB)")
	concurrency = flag.Int("concurrency", 16, "Maximum concurrent file operations (default: 16)")
	dryRun      = flag.Bool("dry-run", false, "Dry run mode - don't delete/expire files")
	debug       = flag.Bool("debug", false, "Enable debug logging")
)

const (
	MiB = 1024 * 1024
)

type FileInfo struct {
	Path  string
	Atime time.Time
	Mtime time.Time
	Size  int64
}

type Manifest struct {
	Files        []FileInfo // .bin files
	BakFiles     []FileInfo // .bak files
	TmpFiles     []FileInfo // .tmp files
	Dirs         []string
	TotalSize    int64 // Total size of .bin files
	ScanDuration time.Duration
}

func main() {
	flag.Parse()

	// Configure logging level
	level := slog.LevelInfo
	if *debug {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

	if *cacheDir == "" {
		slog.Error("cache-dir is required")
		os.Exit(1)
	}

	slog.Info("Starting LRU cache eviction", "cache_dir", *cacheDir, "target_size_mb", *targetSize)

	// Step 1: Single walk to scan cache and collect all file types
	manifest, err := scanCache(*cacheDir)
	if err != nil {
		slog.Error("Failed to scan cache", "error", err)
		os.Exit(1)
	}
	slog.Debug("Cache scanned",
		"bin_files", len(manifest.Files),
		"total_size_mb", float64(manifest.TotalSize)/MiB,
		"bak_files", len(manifest.BakFiles),
		"tmp_files", len(manifest.TmpFiles),
		"scan_duration", manifest.ScanDuration)

	// Step 2: Clean up .bak files from previous run
	if !*dryRun {
		removeBakFiles(manifest.BakFiles)
	} else {
		slog.Info("DRY RUN: Would remove previously expired files", "file_count", len(manifest.BakFiles))
		printFileInfo(manifest.BakFiles)
	}

	targetBytes := *targetSize * MiB
	if manifest.TotalSize > targetBytes {
		// Step 3: Find LRU files to expire if we are above target size.
		filesToExpire := findLRUFiles(manifest, targetBytes)
		slog.Info("Expiring files",
			"expired_size_mb", float64(manifest.TotalSize-targetBytes)/MiB,
			"file_count", len(filesToExpire))

		// Step 4: Expire files in parallel (rename to .bak)
		if !*dryRun {
			expireFiles(filesToExpire)
		} else {
			slog.Info("DRY RUN: Would expire files", "file_count", len(filesToExpire))
			printFileInfo(filesToExpire)
		}
	} else {
		slog.Info("Cache below target, nothing to do",
			"cache_size_mb", float64(manifest.TotalSize)/MiB,
			"target_size_mb", float64(targetBytes)/MiB)
	}

	// Step 5: Remove old .tmp files (older than 1 hour)
	if !*dryRun {
		removeOldTmpFiles(manifest.TmpFiles)
	} else {
		slog.Info("DRY RUN: Would remove old tmp files")
		printFileInfo(manifest.TmpFiles)
	}

	// Step 6: Cleanup empty directories
	if !*dryRun {
		cleanupEmptyDirs(manifest.Dirs)
	} else {
		slog.Info("DRY RUN: Would cleanup empty directories", "dir_count", len(manifest.Dirs))
		for _, dir := range manifest.Dirs {
			slog.Debug("Directory", "path", dir)
		}
	}

	slog.Info("LRU cache eviction completed")
}

// printFileInfo is a helper to log file info for debugging.
func printFileInfo(info []FileInfo) {
	for _, f := range info {
		slog.Debug("FileInfo",
			"path", f.Path,
			"size_mb", float64(f.Size)/MiB,
			"atime", f.Atime,
			"mtime", f.Mtime)
	}
}

// scanCache performs a single walk to collect all file types
func scanCache(cacheDir string) (*Manifest, error) {
	start := time.Now()
	manifest := &Manifest{
		Files:    make([]FileInfo, 0),
		BakFiles: make([]FileInfo, 0),
		TmpFiles: make([]FileInfo, 0),
		Dirs:     make([]string, 0),
	}

	// Look for gcsfuse-file-cache subdirectory
	actualCacheDir := filepath.Join(cacheDir, "gcsfuse-file-cache")
	if info, err := os.Stat(actualCacheDir); err == nil && info.IsDir() {
		cacheDir = actualCacheDir
	}

	err := filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			slog.Warn("Skipping file due to error", "path", path, "error", err)
			return nil // Skip errors
		}

		// Collect directories (excluding root)
		if info.IsDir() {
			if path != cacheDir {
				manifest.Dirs = append(manifest.Dirs, path)
			}
			return nil
		}

		ext := filepath.Ext(path)

		// Get file times (needed for .bin and .tmp files)
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return nil
		}
		atime := time.Unix(stat.Atim.Sec, stat.Atim.Nsec)
		mtime := time.Unix(stat.Mtim.Sec, stat.Mtim.Nsec)

		switch ext {
		case ".bin":
			// Cache chunks - add to manifest
			manifest.Files = append(manifest.Files, FileInfo{
				Path:  path,
				Atime: atime,
				Mtime: mtime,
				Size:  info.Size(),
			})
			manifest.TotalSize += info.Size()

		case ".bak":
			// Previously expired files - collect for cleanup
			manifest.BakFiles = append(manifest.BakFiles, FileInfo{
				Path:  path,
				Atime: atime,
				Mtime: mtime,
				Size:  info.Size(),
			})

		case ".tmp":
			// Temporary files - collect for cleanup
			manifest.TmpFiles = append(manifest.TmpFiles, FileInfo{
				Path:  path,
				Atime: atime,
				Mtime: mtime,
				Size:  info.Size(),
			})
		}

		return nil
	})

	manifest.ScanDuration = time.Since(start)
	return manifest, err
}

// removeBakFiles removes .bak files in parallel
func removeBakFiles(files []FileInfo) {
	if len(files) == 0 {
		return
	}

	var totalSize int64
	fileChan := make(chan FileInfo, *concurrency)
	var wg sync.WaitGroup

	// Start worker pool
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range fileChan {
				if err := os.Remove(file.Path); err != nil && !os.IsNotExist(err) {
					slog.Warn("Failed to remove old bak file", "path", file.Path, "error", err)
				} else {
					atomic.AddInt64(&totalSize, file.Size)
				}
			}
		}()
	}

	// Feed files to workers
	for _, f := range files {
		fileChan <- f
	}
	close(fileChan)

	wg.Wait()

	slog.Info("Removed previously expired (.bak) files",
		"file_count", len(files),
		"size_mb", float64(totalSize)/MiB)
}

// removeOldTmpFiles removes .tmp files older than 1 hour in parallel.
// Keeping 1 hour is a safety margin to avoid deleting in-progress chunk download
// over a tmpFile.
func removeOldTmpFiles(files []FileInfo) {
	cutoff := time.Now().Add(-1 * time.Hour)
	var removedCount int32
	fileChan := make(chan FileInfo, *concurrency)
	var wg sync.WaitGroup

	// Start worker pool
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range fileChan {
				if err := os.Remove(file.Path); err != nil && !os.IsNotExist(err) {
					slog.Warn("Failed to remove old tmp file", "path", file.Path, "error", err)
				} else {
					atomic.AddInt32(&removedCount, 1)
				}
			}
		}()
	}

	// Feed files to workers (only those older than cutoff)
	for _, f := range files {
		if f.Mtime.Before(cutoff) {
			fileChan <- f
		}
	}
	close(fileChan)

	wg.Wait()

	if removedCount > 0 {
		slog.Info("Removed old temporary files", "file_count", removedCount)
	}
}

// findLRUFiles finds least recently used files until we have enough to expire
func findLRUFiles(manifest *Manifest, targetSize int64) []FileInfo {
	if manifest.TotalSize <= targetSize {
		return []FileInfo{}
	}

	bytesToExpire := manifest.TotalSize - targetSize

	// Sort by atime (oldest first)
	slices.SortFunc(manifest.Files, func(a, b FileInfo) int {
		recentA := a.Atime
		if recentA.Before(a.Mtime) {
			recentA = a.Mtime
		}
		recentB := b.Atime
		if recentB.Before(b.Mtime) {
			recentB = b.Mtime
		}

		if recentA.Before(recentB) {
			return -1
		}
		if recentA.After(recentB) {
			return 1
		}
		return 0
	})

	// Select files until we have enough bytes
	var selected []FileInfo
	var totalBytes int64

	for _, f := range manifest.Files {
		selected = append(selected, f)
		totalBytes += f.Size
		if totalBytes >= bytesToExpire {
			break
		}
	}

	return selected
}

// expireFiles renames files from .bin to .bak in parallel.
// This allows ongoing reads to continue until the next run when .bak files
// are cleaned up. Existing request will continue since rename preserves the
// file handle.
func expireFiles(files []FileInfo) {
	fileChan := make(chan FileInfo, *concurrency)
	var wg sync.WaitGroup

	// Start worker pool
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range fileChan {
				bakPath := file.Path + ".bak"
				if err := os.Rename(file.Path, bakPath); err != nil && !os.IsNotExist(err) {
					slog.Warn("Failed to rename file", "path", file.Path, "error", err)
				}
			}
		}()
	}

	// Feed files to workers
	for _, f := range files {
		fileChan <- f
	}
	close(fileChan)

	wg.Wait()
}

// cleanupEmptyDirs removes empty object directories
func cleanupEmptyDirs(dirs []string) {
	// Sort by depth (deepest first)
	slices.SortFunc(dirs, func(a, b string) int {
		aDepth := len(filepath.SplitList(a))
		bDepth := len(filepath.SplitList(b))
		return bDepth - aDepth // Reverse order (deepest first)
	})

	for _, dir := range dirs {
		// Try to remove if empty
		if err := os.Remove(dir); err != nil {
			// Ignore ENOTEMPTY - concurrently files were added back to this directory.
			if !errors.Is(err, syscall.ENOTEMPTY) {
				slog.Debug("Failed to remove directory", "path", dir, "error", err)
			}
		}
	}
}
