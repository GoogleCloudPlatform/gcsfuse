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
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockValue struct {
	size uint64
}

func (m mockValue) Size() uint64 {
	return m.size
}

func TestFileCacheDiskUtilizationCalculator_CurrentSize(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a subdirectory to ensure directory size is non-zero (if FS supports it)
	// or at least calculator runs without error.
	subDir := filepath.Join(tmpDir, "subdir")
	require.NoError(t, os.Mkdir(subDir, 0755))

	calc := NewFileCacheDiskUtilizationCalculator(tmpDir, 100*time.Millisecond, false, false, 4096)
	defer calc.Stop()

	// Initial size should include directory size (which might be 0 on tmpfs or 4096 on ext4)
	initialSize := calc.GetCurrentSize()
	assert.GreaterOrEqual(t, initialSize, uint64(0))

	// Add file usage
	calc.InsertEntry(mockValue{size: 100})
	assert.Equal(t, initialSize+100, calc.GetCurrentSize())

	// Evict file usage
	calc.EvictEntry(mockValue{size: 40})
	assert.Equal(t, initialSize+60, calc.GetCurrentSize())

	// Add delta
	calc.AddDelta(10)
	assert.Equal(t, initialSize+70, calc.GetCurrentSize())

	calc.AddDelta(-50)
	assert.Equal(t, initialSize+20, calc.GetCurrentSize())
}

func TestFileCacheDiskUtilizationCalculator_ClearEmptyDirsAndRescanSize(t *testing.T) {
	tmpDir := t.TempDir()
	// Create empty dirs
	emptyDir := filepath.Join(tmpDir, "empty")
	require.NoError(t, os.Mkdir(emptyDir, 0755))
	// Create non-empty dir
	nonEmptyDir := filepath.Join(tmpDir, "nonEmpty")
	require.NoError(t, os.Mkdir(nonEmptyDir, 0755))
	f, err := os.Create(filepath.Join(nonEmptyDir, "file"))
	require.NoError(t, err)
	f.Close()
	calc := NewFileCacheDiskUtilizationCalculator(tmpDir, 50*time.Millisecond, false, true, 4096)
	defer calc.Stop()

	// Initial check
	initialSize := calc.GetCurrentSize()
	// Wait for update (ticker is 50ms)
	time.Sleep(100 * time.Millisecond)

	// Verify empty dir is gone
	_, err = os.Stat(emptyDir)
	assert.True(t, os.IsNotExist(err), "Empty directory should be removed")
	// Verify non-empty dir exists
	_, err = os.Stat(nonEmptyDir)
	assert.NoError(t, err, "Non-empty directory should exist")
	newSize := calc.GetCurrentSize()
	// Depending on FS, newSize might be same or larger.
	// We at least verify it doesn't crash and returns valid value.
	assert.GreaterOrEqual(t, newSize, initialSize)
}

func TestFileCacheDiskUtilizationCalculator_FullScan(t *testing.T) {
	tmpDir := t.TempDir()
	f, err := os.CreateTemp(tmpDir, "testfile")
	require.NoError(t, err)
	_, err = f.Write([]byte("hello"))
	require.NoError(t, err)
	f.Close()

	// Use includeFiles=true
	calc := NewFileCacheDiskUtilizationCalculator(tmpDir, 50*time.Millisecond, true, false, 4096)
	defer calc.Stop()

	// Wait for update
	time.Sleep(100 * time.Millisecond)

	size := calc.GetCurrentSize()
	// Should include file size (4096 on most FS due to block size).
	// On tmpfs, directory might be 0, file 4096.
	assert.GreaterOrEqual(t, size, uint64(4096))
}

func TestFileCacheDiskUtilizationCalculator_AddDelta(t *testing.T) {
	tmpDir := t.TempDir()
	calc := NewFileCacheDiskUtilizationCalculator(tmpDir, time.Hour, false, false, 4096)
	defer calc.Stop()

	// Initial size (empty dir)
	initialSize := calc.GetCurrentSize()

	// Add positive delta
	calc.AddDelta(100)
	assert.Equal(t, initialSize+100, calc.GetCurrentSize())

	// Add another positive delta
	calc.AddDelta(50)
	assert.Equal(t, initialSize+150, calc.GetCurrentSize())

	// Add negative delta
	calc.AddDelta(-30)
	assert.Equal(t, initialSize+120, calc.GetCurrentSize())

	// Add negative delta to zero out added amount
	calc.AddDelta(-120)
	assert.Equal(t, initialSize, calc.GetCurrentSize())
}

func TestFileCacheDiskUtilizationCalculator_SizeOf_NonSparseFile(t *testing.T) {
	tmpDir := t.TempDir()
	calc := NewFileCacheDiskUtilizationCalculator(tmpDir, time.Hour, false, false, 4096)
	defer calc.Stop()
	// FileSize: 10000. Block size: 4096.
	// Blocks: ceil(10000/4096) = 3. Size: 3 * 4096 = 12288.
	fiNonSparse := data.FileInfo{
		FileSize:   10000,
		SparseMode: false,
	}
	expectedNonSparse := uint64(12288)

	size := calc.SizeOf(fiNonSparse)

	assert.Equal(t, expectedNonSparse, size)
}

func TestFileCacheDiskUtilizationCalculator_SizeOf_SparseFile(t *testing.T) {
	tmpDir := t.TempDir()
	calc := NewFileCacheDiskUtilizationCalculator(tmpDir, time.Hour, false, false, 4096)
	defer calc.Stop()
	// Downloaded: 2 chunks of 1024 bytes.
	// Chunk 0: [0, 1024)
	// Chunk 2: [2048, 3072)
	brm := data.NewByteRangeMap(1024)
	brm.AddRange(0, 1024)
	brm.AddRange(2048, 3072)
	fiSparse := data.FileInfo{
		FileSize:         10000,
		SparseMode:       true,
		DownloadedRanges: brm,
	}
	// Total downloaded bytes = 2048.
	// Block size: 4096.
	// ceil(2048/4096) = 1. Size: 4096.
	expectedSparse := uint64(4096)

	size := calc.SizeOf(fiSparse)

	assert.Equal(t, expectedSparse, size)
}
