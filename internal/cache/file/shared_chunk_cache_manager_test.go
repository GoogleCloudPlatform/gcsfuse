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
	"os"
	"path/filepath"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSharedChunkCacheManager(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()

	// Act
	manager, err := NewSharedChunkCacheManager(
		tmpDir,
		0644,
		0755,
		&cfg.FileCacheConfig{},
	)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, manager)
	assert.NotEmpty(t, manager.cacheDir)
	assert.Equal(t, int64(manager.config.SharedCacheChunkSizeMb*1024*1024), manager.chunkSize)
}

func TestSharedChunkCacheManager_ShouldExcludeFromCache(t *testing.T) {
	tests := []struct {
		name         string
		includeRegex string
		excludeRegex string
		bucketName   string
		objectName   string
		wantExcluded bool
	}{
		{
			name:         "No regex - should not exclude",
			includeRegex: "",
			excludeRegex: "",
			bucketName:   "test-bucket",
			objectName:   "file.txt",
			wantExcluded: false,
		},
		{
			name:         "Include regex matches - should not exclude",
			includeRegex: ".*\\.txt$",
			excludeRegex: "",
			bucketName:   "test-bucket",
			objectName:   "file.txt",
			wantExcluded: false,
		},
		{
			name:         "Include regex does not match - should exclude",
			includeRegex: ".*\\.txt$",
			excludeRegex: "",
			bucketName:   "test-bucket",
			objectName:   "file.log",
			wantExcluded: true,
		},
		{
			name:         "Exclude regex matches - should exclude",
			includeRegex: "",
			excludeRegex: ".*\\.log$",
			bucketName:   "test-bucket",
			objectName:   "file.log",
			wantExcluded: true,
		},
		{
			name:         "Exclude regex does not match - should not exclude",
			includeRegex: "",
			excludeRegex: ".*\\.log$",
			bucketName:   "test-bucket",
			objectName:   "file.txt",
			wantExcluded: false,
		},
		{
			name:         "Both regexes - include matches, exclude does not - should not exclude",
			includeRegex: ".*\\.txt$",
			excludeRegex: ".*temp.*",
			bucketName:   "test-bucket",
			objectName:   "file.txt",
			wantExcluded: false,
		},
		{
			name:         "Both regexes - include matches, exclude also matches - should exclude",
			includeRegex: ".*\\.txt$",
			excludeRegex: ".*temp.*",
			bucketName:   "test-bucket",
			objectName:   "temp.txt",
			wantExcluded: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			tmpDir := t.TempDir()
			config := &cfg.FileCacheConfig{
				IncludeRegex: tt.includeRegex,
				ExcludeRegex: tt.excludeRegex,
			}
			manager, err := NewSharedChunkCacheManager(tmpDir, 0644, 0755, config)
			require.NoError(t, err)
			bucket := fake.NewFakeBucket(timeutil.RealClock(), tt.bucketName, gcs.BucketType{})
			object := &gcs.MinObject{Name: tt.objectName}

			// Act
			result := manager.ShouldExcludeFromCache(bucket, object)

			// Assert
			assert.Equal(t, tt.wantExcluded, result)
		})
	}
}

func TestSharedChunkCacheManager_GetChunkIndex(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	config := &cfg.FileCacheConfig{
		SharedCacheChunkSizeMb: 8, // 8 MB chunks
	}
	manager, err := NewSharedChunkCacheManager(tmpDir, 0644, 0755, config)
	require.NoError(t, err)

	tests := []struct {
		name      string
		offset    int64
		wantIndex int64
	}{
		{"First byte", 0, 0},
		{"Last byte of chunk 0", 8*1024*1024 - 1, 0},
		{"First byte of chunk 1", 8 * 1024 * 1024, 1},
		{"Middle of chunk 1", 10 * 1024 * 1024, 1},
		{"Chunk 2", 16 * 1024 * 1024, 2},
		{"Chunk 10", 80 * 1024 * 1024, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result := manager.GetChunkIndex(tt.offset)

			// Assert
			assert.Equal(t, tt.wantIndex, result)
		})
	}
}

func TestSharedChunkCacheManager_GetChunkSize(t *testing.T) {
	tests := []struct {
		name              string
		chunkSizeMb       int64
		expectedChunkSize int64
	}{
		{
			name:              "Default chunk size",
			chunkSizeMb:       0, // 0 means use default
			expectedChunkSize: 0,
		},
		{
			name:              "Custom 16MB chunk size",
			chunkSizeMb:       16,
			expectedChunkSize: 16 * 1024 * 1024,
		},
		{
			name:              "Custom 32MB chunk size",
			chunkSizeMb:       32,
			expectedChunkSize: 32 * 1024 * 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			tmpDir := t.TempDir()
			config := &cfg.FileCacheConfig{
				SharedCacheChunkSizeMb: tt.chunkSizeMb,
			}
			manager, err := NewSharedChunkCacheManager(tmpDir, 0644, 0755, config)
			require.NoError(t, err)

			// Act
			result := manager.GetChunkSize()

			// Assert
			assert.Equal(t, tt.expectedChunkSize, result)
		})
	}
}

func TestSharedChunkCacheManager_GetObjectDir(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	manager, err := NewSharedChunkCacheManager(tmpDir, 0644, 0755, &cfg.FileCacheConfig{})
	require.NoError(t, err)
	tests := []struct {
		name             string
		bucketName       string
		objectName       string
		generation       int64
		expectedCacheDir string
	}{
		{
			name:             "Simple object",
			bucketName:       "my-bucket",
			objectName:       "file.txt",
			generation:       12345,
			expectedCacheDir: "41/7d/417d6e4989a22cfa815f9e622a859475121dacee0793e846b03e089b9d837e6a",
		},
		{
			name:             "Object with path",
			bucketName:       "my-bucket",
			objectName:       "dir/subdir/file.txt",
			generation:       67890,
			expectedCacheDir: "73/8e/738ef21e631dc30612f56ccc87eba8f76bd714ccc22238a2549c7a3177f44bfa",
		},
		{
			name:             "Zero generation",
			bucketName:       "bucket",
			objectName:       "obj",
			generation:       0,
			expectedCacheDir: "51/e5/51e59a4c0ab165b7b8c93715fcba438c133f8e8a14d94452dc7b2ce7315ae321",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			expectedCacheDir := filepath.Join(tmpDir, tt.expectedCacheDir)

			// Act
			result := manager.GetObjectDir(tt.bucketName, tt.objectName, tt.generation)

			// Assert
			assert.Equal(t, expectedCacheDir, result)
		})
	}
}

func TestSharedChunkCacheManager_GetChunkPath(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	manager, err := NewSharedChunkCacheManager(tmpDir, 0644, 0755, &cfg.FileCacheConfig{
		SharedCacheChunkSizeMb: 8, // 8 MB chunks
	})
	require.NoError(t, err)

	tests := []struct {
		name         string
		bucketName   string
		objectName   string
		generation   int64
		chunkIndex   int64
		expectedPath string
	}{
		{
			name:         "Chunk 0",
			bucketName:   "my-bucket",
			objectName:   "file.txt",
			generation:   12345,
			chunkIndex:   0,
			expectedPath: "41/7d/417d6e4989a22cfa815f9e622a859475121dacee0793e846b03e089b9d837e6a/0_8388608.bin",
		},
		{
			name:         "Chunk 5",
			bucketName:   "my-bucket",
			objectName:   "dir/file.txt",
			generation:   67890,
			chunkIndex:   5,
			expectedPath: "8e/0b/8e0bf1cb92e4496f8107137549a19d9c429fdea785e54258e529751c8cc98093/41943040_50331648.bin",
		},
		{
			name:         "Large chunk index",
			bucketName:   "bucket",
			objectName:   "object",
			generation:   1,
			chunkIndex:   999,
			expectedPath: "1a/8b/1a8b371ee4267a3bbed920ce28dc0e8796a7bfe39b17e3b312f4469c6c25a7b1/8380219392_8388608000.bin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			expectedPath := filepath.Join(tmpDir, tt.expectedPath)

			// Act
			result := manager.GetChunkPath(tt.bucketName, tt.objectName, tt.generation, tt.chunkIndex)

			// Assert
			assert.Equal(t, expectedPath, result)
		})
	}
}

func TestSharedChunkCacheManager_GenerateTmpPath(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	manager, err := NewSharedChunkCacheManager(tmpDir, 0644, 0755, &cfg.FileCacheConfig{
		SharedCacheChunkSizeMb: 8, // 8 MB chunks
	})
	require.NoError(t, err)

	tests := []struct {
		name             string
		bucketName       string
		objectName       string
		generation       int64
		chunkIndex       int64
		expectedBasePath string // Path without random prefix
	}{
		{
			name:             "Chunk 0",
			bucketName:       "my-bucket",
			objectName:       "file.txt",
			generation:       12345,
			chunkIndex:       0,
			expectedBasePath: "41/7d/417d6e4989a22cfa815f9e622a859475121dacee0793e846b03e089b9d837e6a/0_8388608.bin",
		},
		{
			name:             "Chunk 5",
			bucketName:       "my-bucket",
			objectName:       "dir/file.txt",
			generation:       67890,
			chunkIndex:       5,
			expectedBasePath: "8e/0b/8e0bf1cb92e4496f8107137549a19d9c429fdea785e54258e529751c8cc98093/41943040_50331648.bin",
		},
		{
			name:             "Large chunk index",
			bucketName:       "bucket",
			objectName:       "object",
			generation:       1,
			chunkIndex:       999,
			expectedBasePath: "1a/8b/1a8b371ee4267a3bbed920ce28dc0e8796a7bfe39b17e3b312f4469c6c25a7b1/8380219392_8388608000.bin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			expectedBasePath := filepath.Join(tmpDir, tt.expectedBasePath)

			// Act
			result := manager.GenerateTmpPath(tt.bucketName, tt.objectName, tt.generation, tt.chunkIndex)

			// Assert - Verify pattern: <base>.<16-hex-chars>.tmp
			assert.Contains(t, result, expectedBasePath, "Tmp path should contain base chunk path")
			assert.True(t, filepath.Ext(result) == ".tmp", "Tmp path should end with .tmp extension")
			// Verify it has the format: <base>.<random>.tmp
			// The random part should be 16 hex characters (8 bytes encoded)
			assert.Regexp(t, `\.bin\.[0-9a-f]{16}\.tmp$`, result, "Tmp path should have random hex prefix before .tmp")
		})
	}
}

func TestSharedChunkCacheManager_GenerateTmpPath_NoCollisions(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	manager, err := NewSharedChunkCacheManager(tmpDir, 0644, 0755, &cfg.FileCacheConfig{
		SharedCacheChunkSizeMb: 8,
	})
	require.NoError(t, err)
	bucketName := "test-bucket"
	objectName := "test-object.txt"
	generation := int64(12345)
	chunkIndex := int64(0)
	numPaths := 100

	// Act - Generate multiple tmp paths for the same chunk
	generatedPaths := make(map[string]bool)
	for i := 0; i < numPaths; i++ {
		path := manager.GenerateTmpPath(bucketName, objectName, generation, chunkIndex)
		generatedPaths[path] = true
	}

	// Assert - All paths should be unique (no collisions)
	assert.Equal(t, numPaths, len(generatedPaths), "All generated tmp paths should be unique - no collisions")
	// Assert - Verify all paths follow the expected pattern
	for path := range generatedPaths {
		assert.Regexp(t, `\.bin\.[0-9a-f]{16}\.tmp$`, path, "Each tmp path should have random hex prefix before .tmp")
	}
}

func TestSharedChunkCacheManager_GetFilePerm(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	filePerm := os.FileMode(0600)
	manager, err := NewSharedChunkCacheManager(tmpDir, filePerm, 0755, &cfg.FileCacheConfig{})
	require.NoError(t, err)

	// Act
	result := manager.GetFilePerm()

	// Assert
	assert.Equal(t, filePerm, result)
}

func TestSharedChunkCacheManager_GetDirPerm(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	dirPerm := os.FileMode(0700)
	manager, err := NewSharedChunkCacheManager(tmpDir, 0644, dirPerm, &cfg.FileCacheConfig{})
	require.NoError(t, err)

	// Act
	result := manager.GetDirPerm()

	// Assert
	assert.Equal(t, dirPerm, result)
}
