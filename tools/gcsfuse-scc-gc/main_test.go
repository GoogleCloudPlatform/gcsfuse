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
	manifest, err := createManifest(cacheDir)
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
	manifest, err := createManifest(cacheDir)
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
	removeAllBakFiles(cacheDir)

	// Assert
	_, err := os.Stat(bakPath)
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
	removeOldTmpFiles(cacheDir)

	// Assert
	_, err := os.Stat(oldTmpPath)
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
	manifest, err := createManifest(cacheDir)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, 2, len(manifest.Files), "Expected 2 .bin files")
	for _, f := range manifest.Files {
		assert.Equal(t, ".bin", filepath.Ext(f.Path), "All files should be .bin")
	}
}
