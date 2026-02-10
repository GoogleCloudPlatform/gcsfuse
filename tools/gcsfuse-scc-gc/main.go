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
* 1. Cleans up `.bak` files expired during the previous run.
* 2. Scans cache directory for `.bin` files with atime and size
* 3. If total size < target, then exit without expiration or eviction.
* 4. Sorts by atime and selects oldest files to expire.
* 5. Renames selected files to `.bak` (kept until next run for ongoing reads).
* 6. Removes old `.tmp` files (older than 1 hour)
* 7. Cleans up empty directories.
 */

package main

import (
	"errors"
	"flag"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	cacheDir   = flag.String("cache-dir", "", "Path to the cache directory")
	targetSize = flag.Int64("target-size-mb", 10240, "Target cache size in MB (default: 10GB)")
	dryRun     = flag.Bool("dry-run", false, "Dry run mode - don't delete/expire files")
	debug      = flag.Bool("debug", false, "Enable debug logging")
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
	Files        []FileInfo
	TotalSize    int64
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

	// Step 1: Clean up any previous .bak expired as part of previous run.
	if !*dryRun {
		removeAllBakFiles(*cacheDir)
	}

	// Step 2: Create manifest by enumerating all files
	manifest, err := createManifest(*cacheDir)
	if err != nil {
		slog.Error("Failed to create manifest", "error", err)
		os.Exit(1)
	}
	slog.Debug("Manifest created",
		"files", len(manifest.Files),
		"total_size_mb", float64(manifest.TotalSize)/MiB,
		"scan_duration", manifest.ScanDuration)

	// Step 3: Check if we need to expire files
	targetBytes := *targetSize * MiB
	if manifest.TotalSize <= targetBytes {
		slog.Info("Cache below target, nothing to do",
			"cache_size_mb", float64(manifest.TotalSize)/MiB,
			"target_size_mb", float64(targetBytes)/MiB)
		return
	}

	// Step 4: Find LRU files to expire
	filesToExpire := findLRUFiles(manifest, targetBytes)
	slog.Info("Expiring files",
		"expired_size_mb", float64(manifest.TotalSize-targetBytes)/MiB,
		"file_count", len(filesToExpire))

	// Step 5: Expire files in parallel (rename to .bak)
	if !*dryRun {
		expireFiles(filesToExpire)

		// Step 6: Remove old .tmp files (older than 1 hour)
		removeOldTmpFiles(*cacheDir)

		// Step 7: Cleanup empty directories
		cleanupEmptyDirs(*cacheDir)
	} else {
		slog.Info("DRY RUN: Would expire files", "file_count", len(filesToExpire))
		for i, f := range filesToExpire {
			if i < 10 { // Show first 10
				slog.Debug("Would evict",
					"path", f.Path,
					"size_mb", float64(f.Size)/MiB,
					"atime", f.Atime)
			}
		}
		if len(filesToExpire) > 10 {
			slog.Debug("Additional files to evict", "count", len(filesToExpire)-10)
		}
	}

	slog.Info("LRU cache eviction completed")
}

// createManifest enumerates all cache files
func createManifest(cacheDir string) (*Manifest, error) {
	start := time.Now()
	manifest := &Manifest{
		Files: make([]FileInfo, 0),
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

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process .bin files (cache chunks)
		if filepath.Ext(path) != ".bin" {
			return nil
		}

		// Get atime and mtime
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return nil
		}

		atime := time.Unix(stat.Atim.Sec, stat.Atim.Nsec)
		mtime := time.Unix(stat.Mtim.Sec, stat.Mtim.Nsec)

		// Use mtime as fallback if atime looks stale (not updated due to noatime/relatime)
		// If atime equals mtime, it likely means atime isn't being updated
		effectiveTime := atime
		if atime.Equal(mtime) || atime.Before(mtime) {
			effectiveTime = mtime
		}

		manifest.Files = append(manifest.Files, FileInfo{
			Path:  path,
			Atime: effectiveTime,
			Mtime: mtime,
			Size:  info.Size(),
		})
		manifest.TotalSize += info.Size()

		return nil
	})

	manifest.ScanDuration = time.Since(start)
	return manifest, err
}

// findLRUFiles finds least recently used files until we have enough to expire
func findLRUFiles(manifest *Manifest, targetSize int64) []FileInfo {
	if manifest.TotalSize <= targetSize {
		return []FileInfo{}
	}

	bytesToExpire := manifest.TotalSize - targetSize

	// Sort by atime (oldest first)
	sorted := make([]FileInfo, len(manifest.Files))
	copy(sorted, manifest.Files)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Atime.Before(sorted[j].Atime)
	})

	// Select files until we have enough bytes
	var selected []FileInfo
	var totalBytes int64

	for _, f := range sorted {
		selected = append(selected, f)
		totalBytes += f.Size
		if totalBytes >= bytesToExpire {
			break
		}
	}

	return selected
}

// expireFiles renames files to .bak in parallel
func expireFiles(files []FileInfo) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // Limit concurrency

	for _, f := range files {
		wg.Add(1)
		go func(file FileInfo) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			bakPath := file.Path + ".bak"
			if err := os.Rename(file.Path, bakPath); err != nil && !os.IsNotExist(err) {
				slog.Warn("Failed to rename file", "path", file.Path, "error", err)
			}
		}(f)
	}

	wg.Wait()
}

// removeOldTmpFiles removes .tmp files older than 1 hour
func removeOldTmpFiles(cacheDir string) {
	cutoff := time.Now().Add(-1 * time.Hour)
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10)

	if err := filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			slog.Warn("Skipping file due to error", "path", path, "error", err)
			return nil
		}

		if info.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".tmp" {
			return nil
		}

		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return nil
		}

		atime := time.Unix(stat.Atim.Sec, stat.Atim.Nsec)
		if atime.Before(cutoff) {
			wg.Add(1)
			go func(p string) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
					slog.Warn("Failed to remove old tmp file", "path", p, "error", err)
				}
			}(path)
		}

		return nil
	}); err != nil {
		slog.Error("Failed to walk cache directory for tmp cleanup", "error", err)
		return
	}

	wg.Wait()
}

// removeAllBakFiles removes all existing .bak files in the cache
func removeAllBakFiles(cacheDir string) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10)
	var totalSize int64
	var fileCount int

	if err := filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			slog.Warn("Skipping file due to error", "path", path, "error", err)
			return nil
		}

		if info.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".bak" {
			return nil
		}

		size := info.Size()
		fileCount++

		wg.Add(1)
		go func(p string, s int64) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
				slog.Warn("Failed to remove old bak file", "path", p, "error", err)
			} else {
				atomic.AddInt64(&totalSize, s)
			}
		}(path, size)

		return nil
	}); err != nil {
		slog.Error("Failed to walk cache directory for bak cleanup", "error", err)
		return
	}

	wg.Wait()

	if fileCount > 0 {
		slog.Info("Removed previously expired (.bak) files",
			"file_count", fileCount,
			"size_mb", float64(totalSize)/(1024*1024))
	}
}

// cleanupEmptyDirs removes empty object directories
func cleanupEmptyDirs(cacheDir string) {
	// Walk from deepest to shallowest
	var dirs []string
	if err := filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() || path == cacheDir {
			return nil
		}
		dirs = append(dirs, path)
		return nil
	}); err != nil {
		slog.Error("Failed to walk cache directory for empty dir cleanup", "error", err)
		return
	}

	// Sort by depth (deepest first)
	sort.Slice(dirs, func(i, j int) bool {
		iDepth := len(filepath.SplitList(dirs[i]))
		jDepth := len(filepath.SplitList(dirs[j]))
		return iDepth > jDepth
	})

	for _, dir := range dirs {
		// Try to remove if empty
		if err := os.Remove(dir); err != nil {
			// Ignore ENOTEMPTY - concurrently files were addeed back to this directory.
			if !errors.Is(err, syscall.ENOTEMPTY) {
				slog.Debug("Failed to remove directory", "path", dir, "error", err)
			}
		}
	}
}
