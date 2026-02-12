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

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLRUEviction(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	objDir := filepath.Join(cacheDir, "a1", "b2", "hash123")
	require.NoError(t, os.MkdirAll(objDir, 0755))
	now := time.Now()
	files := []struct {
		name  string
		size  int64
		atime time.Time
	}{
		{"0_1048576.bin", 1024 * 1024, now.Add(-3 * time.Hour)},
		{"1048576_2097152.bin", 1024 * 1024, now.Add(-2 * time.Hour)},
		{"2097152_3145728.bin", 1024 * 1024, now.Add(-1 * time.Hour)},
	}
	for _, f := range files {
		path := filepath.Join(objDir, f.name)
		require.NoError(t, os.WriteFile(path, make([]byte, f.size), 0644))
		require.NoError(t, os.Chtimes(path, f.atime, f.atime))
	}

	// Act
	manifest, err := scanCache(cacheDir)
	require.NoError(t, err)
	targetSize := int64(2 * 1024 * 1024)
	filesToExpire := findLRUFiles(manifest, targetSize)
	expireFiles(filesToExpire)

	// Assert
	assert.Equal(t, 3, len(manifest.Files))
	assert.Equal(t, 1, len(filesToExpire))
	if len(filesToExpire) > 0 {
		assert.Equal(t, "0_1048576.bin", filepath.Base(filesToExpire[0].Path))
	}
	bakPath := filesToExpire[0].Path + ".bak"
	_, err = os.Stat(bakPath)
	assert.NoError(t, err, "Expected .bak file to exist")
	_, err = os.Stat(filesToExpire[0].Path)
	assert.True(t, os.IsNotExist(err), "Expected original file to be gone")
}

func TestNoEvictionWhenBelowTarget(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	objDir := filepath.Join(cacheDir, "a1", "b2", "hash123")
	require.NoError(t, os.MkdirAll(objDir, 0755))
	path := filepath.Join(objDir, "0_1048576.bin")
	require.NoError(t, os.WriteFile(path, make([]byte, 1024), 0644))

	// Act
	manifest, err := scanCache(cacheDir)
	require.NoError(t, err)
	targetSize := int64(2 * 1024 * 1024)
	filesToExpire := findLRUFiles(manifest, targetSize)

	// Assert
	assert.Equal(t, 1, len(manifest.Files))
	assert.Equal(t, 0, len(filesToExpire), "Expected no files to expire")
}

func TestBakFileCleanup(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	objDir := filepath.Join(cacheDir, "a1", "b2", "hash123")
	require.NoError(t, os.MkdirAll(objDir, 0755))
	bakPath := filepath.Join(objDir, "old_file.bin.bak")
	require.NoError(t, os.WriteFile(bakPath, []byte("test"), 0644))

	// Act
	manifest, err := scanCache(cacheDir)
	require.NoError(t, err)
	removeBakFiles(manifest.BakFiles)

	// Assert
	_, err = os.Stat(bakPath)
	assert.True(t, os.IsNotExist(err), "Expected .bak file to be removed")
}

func TestTmpFileCleanup(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	objDir := filepath.Join(cacheDir, "a1", "b2", "hash123")
	require.NoError(t, os.MkdirAll(objDir, 0755))
	now := time.Now()
	oldTmpPath := filepath.Join(objDir, "old.tmp")
	require.NoError(t, os.WriteFile(oldTmpPath, []byte("test"), 0644))
	oldTime := now.Add(-2 * time.Hour)
	require.NoError(t, os.Chtimes(oldTmpPath, oldTime, oldTime))
	recentTmpPath := filepath.Join(objDir, "recent.tmp")
	require.NoError(t, os.WriteFile(recentTmpPath, []byte("test"), 0644))
	recentTime := now.Add(-30 * time.Minute)
	require.NoError(t, os.Chtimes(recentTmpPath, recentTime, recentTime))

	// Act
	manifest, err := scanCache(cacheDir)
	require.NoError(t, err)
	removeOldTmpFiles(manifest.TmpFiles)

	// Assert
	_, err = os.Stat(oldTmpPath)
	assert.True(t, os.IsNotExist(err), "Expected old .tmp file to be removed")
	_, err = os.Stat(recentTmpPath)
	assert.NoError(t, err, "Expected recent .tmp file to still exist")
}

func TestOnlyBinFilesProcessed(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	objDir := filepath.Join(cacheDir, "a1", "b2", "hash123")
	require.NoError(t, os.MkdirAll(objDir, 0755))
	files := []string{
		"0_1048576.bin",
		"test.txt",
		"data.json",
		"1048576_2097152.bin",
	}
	for _, f := range files {
		path := filepath.Join(objDir, f)
		require.NoError(t, os.WriteFile(path, []byte("test"), 0644))
	}

	// Act
	manifest, err := scanCache(cacheDir)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, 2, len(manifest.Files), "Expected 2 .bin files")
	for _, f := range manifest.Files {
		assert.Equal(t, ".bin", filepath.Ext(f.Path), "All files should be .bin")
	}
}

func TestMultipleFilesExpiredToReachTarget(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	objDir := filepath.Join(cacheDir, "a1", "b2", "hash123")
	require.NoError(t, os.MkdirAll(objDir, 0755))
	now := time.Now()
	files := []struct {
		name  string
		size  int64
		atime time.Time
	}{
		{"0_1048576.bin", 1024 * 1024, now.Add(-5 * time.Hour)},       // Oldest
		{"1048576_2097152.bin", 1024 * 1024, now.Add(-4 * time.Hour)}, // 2nd oldest
		{"2097152_3145728.bin", 1024 * 1024, now.Add(-3 * time.Hour)}, // 3rd oldest
		{"3145728_4194304.bin", 1024 * 1024, now.Add(-2 * time.Hour)}, // Newest (kept)
	}
	for _, f := range files {
		path := filepath.Join(objDir, f.name)
		require.NoError(t, os.WriteFile(path, make([]byte, f.size), 0644))
		require.NoError(t, os.Chtimes(path, f.atime, f.atime))
	}

	// Act
	manifest, err := scanCache(cacheDir)
	require.NoError(t, err)
	targetSize := int64(1536 * 1024) // 1.5 MB - need to expire ~2.5MB
	filesToExpire := findLRUFiles(manifest, targetSize)

	// Assert
	assert.Equal(t, 4, len(manifest.Files))
	assert.Equal(t, 3, len(filesToExpire), "Expected 3 oldest files to be expired")
	// Verify correct LRU order
	assert.Equal(t, "0_1048576.bin", filepath.Base(filesToExpire[0].Path))
	assert.Equal(t, "1048576_2097152.bin", filepath.Base(filesToExpire[1].Path))
	assert.Equal(t, "2097152_3145728.bin", filepath.Base(filesToExpire[2].Path))
}

func TestLRUWithIdenticalAtimes(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	objDir := filepath.Join(cacheDir, "a1", "b2", "hash123")
	require.NoError(t, os.MkdirAll(objDir, 0755))
	now := time.Now()
	sameTime := now.Add(-2 * time.Hour)
	files := []struct {
		name  string
		size  int64
		atime time.Time
	}{
		{"0_1048576.bin", 1024 * 1024, sameTime},
		{"1048576_2097152.bin", 1024 * 1024, sameTime},
		{"2097152_3145728.bin", 1024 * 1024, now.Add(-1 * time.Hour)}, // Newer
	}
	for _, f := range files {
		path := filepath.Join(objDir, f.name)
		require.NoError(t, os.WriteFile(path, make([]byte, f.size), 0644))
		require.NoError(t, os.Chtimes(path, f.atime, f.atime))
	}

	// Act
	manifest, err := scanCache(cacheDir)
	require.NoError(t, err)
	targetSize := int64(1024 * 1024) // 1 MB - need to expire 2MB
	filesToExpire := findLRUFiles(manifest, targetSize)

	// Assert
	assert.Equal(t, 3, len(manifest.Files))
	assert.GreaterOrEqual(t, len(filesToExpire), 2, "Expected at least 2 files to be expired")
	// The newest file should not be in the expired list
	for _, f := range filesToExpire {
		assert.NotEqual(t, "2097152_3145728.bin", filepath.Base(f.Path),
			"Newest file should not be expired")
	}
}

func TestAtimeFallbackToMtime(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	objDir := filepath.Join(cacheDir, "a1", "b2", "hash123")
	require.NoError(t, os.MkdirAll(objDir, 0755))
	now := time.Now()
	oldTime := now.Add(-5 * time.Hour)
	recentTime := now.Add(-1 * time.Hour)
	path := filepath.Join(objDir, "0_1048576.bin")
	require.NoError(t, os.WriteFile(path, make([]byte, 1024), 0644))
	require.NoError(t, os.Chtimes(path, recentTime, recentTime))
	// Create another file with older times
	path2 := filepath.Join(objDir, "1048576_2097152.bin")
	require.NoError(t, os.WriteFile(path2, make([]byte, 1024), 0644))
	require.NoError(t, os.Chtimes(path2, oldTime, oldTime))

	// Act
	manifest, err := scanCache(cacheDir)
	require.NoError(t, err)
	targetSize := int64(1024) // Keep only 1KB
	filesToExpire := findLRUFiles(manifest, targetSize)

	// Assert
	assert.Equal(t, 2, len(manifest.Files))
	require.Equal(t, 1, len(filesToExpire), "Expected 1 file to be expired")
	// The older file should be expired first
	assert.Equal(t, "1048576_2097152.bin", filepath.Base(filesToExpire[0].Path),
		"Oldest file (by mtime fallback) should be expired")
}

func TestTmpFileAtOneHourBoundary(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	objDir := filepath.Join(cacheDir, "a1", "b2", "hash123")
	require.NoError(t, os.MkdirAll(objDir, 0755))
	now := time.Now()
	// File well over 1 hour old
	oldFile := filepath.Join(objDir, "old.tmp")
	require.NoError(t, os.WriteFile(oldFile, []byte("test"), 0644))
	oldTime := now.Add(-2 * time.Hour)
	require.NoError(t, os.Chtimes(oldFile, oldTime, oldTime))
	// File just under 1 hour old
	recentFile := filepath.Join(objDir, "recent.tmp")
	require.NoError(t, os.WriteFile(recentFile, []byte("test"), 0644))
	recentTime := now.Add(-59 * time.Minute)
	require.NoError(t, os.Chtimes(recentFile, recentTime, recentTime))

	// Act
	manifest, err := scanCache(cacheDir)
	require.NoError(t, err)
	removeOldTmpFiles(manifest.TmpFiles)

	// Assert
	assert.Equal(t, 2, len(manifest.TmpFiles))
	// File over 1 hour should be removed
	_, err = os.Stat(oldFile)
	assert.True(t, os.IsNotExist(err), "File over 1 hour should be removed")
	// File under 1 hour should still exist
	_, err = os.Stat(recentFile)
	assert.NoError(t, err, "File under 1 hour should still exist")
}

func TestEmptyTmpFileList(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	objDir := filepath.Join(cacheDir, "a1", "b2", "hash123")
	require.NoError(t, os.MkdirAll(objDir, 0755))
	// Create only .bin files, no .tmp files
	binPath := filepath.Join(objDir, "0_1048576.bin")
	require.NoError(t, os.WriteFile(binPath, []byte("test"), 0644))

	// Act
	manifest, err := scanCache(cacheDir)
	require.NoError(t, err)
	// Should not panic with empty list
	removeOldTmpFiles(manifest.TmpFiles)

	// Assert
	assert.Equal(t, 0, len(manifest.TmpFiles), "Expected no tmp files")
	assert.Equal(t, 1, len(manifest.Files), "Expected 1 bin file")
}

func TestLRUSortingOrderVerification(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	objDir := filepath.Join(cacheDir, "a1", "b2", "hash123")
	require.NoError(t, os.MkdirAll(objDir, 0755))
	now := time.Now()
	// Create files in non-chronological order
	files := []struct {
		name  string
		size  int64
		atime time.Time
	}{
		{"file3.bin", 512 * 1024, now.Add(-2 * time.Hour)},    // Middle
		{"file1.bin", 512 * 1024, now.Add(-5 * time.Hour)},    // Oldest
		{"file5.bin", 512 * 1024, now.Add(-30 * time.Minute)}, // Newest
		{"file2.bin", 512 * 1024, now.Add(-4 * time.Hour)},    // 2nd oldest
		{"file4.bin", 512 * 1024, now.Add(-1 * time.Hour)},    // 2nd newest
	}
	for _, f := range files {
		path := filepath.Join(objDir, f.name)
		require.NoError(t, os.WriteFile(path, make([]byte, f.size), 0644))
		require.NoError(t, os.Chtimes(path, f.atime, f.atime))
	}

	// Act
	manifest, err := scanCache(cacheDir)
	require.NoError(t, err)
	targetSize := int64(1024 * 1024) // 1 MB - need to expire ~1.5MB
	filesToExpire := findLRUFiles(manifest, targetSize)

	// Assert
	assert.Equal(t, 5, len(manifest.Files))
	require.GreaterOrEqual(t, len(filesToExpire), 3, "Expected at least 3 files to be expired")
	// Verify files are in strict chronological order (oldest first)
	assert.Equal(t, "file1.bin", filepath.Base(filesToExpire[0].Path), "1st expired should be oldest")
	assert.Equal(t, "file2.bin", filepath.Base(filesToExpire[1].Path), "2nd expired should be 2nd oldest")
	assert.Equal(t, "file3.bin", filepath.Base(filesToExpire[2].Path), "3rd expired should be 3rd oldest")
	// Verify chronological ordering
	for i := 1; i < len(filesToExpire); i++ {
		assert.True(t, filesToExpire[i-1].Atime.Before(filesToExpire[i].Atime) ||
			filesToExpire[i-1].Atime.Equal(filesToExpire[i].Atime),
			"Files should be sorted by atime (oldest first)")
	}
}
