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

package gcsx

import (
	"bytes"
	"context"
	"io"
	"os"
	"sync"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/file"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/googlecloudplatform/gcsfuse/v3/tracing"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	testBucketName  = "test-bucket"
	testObjectName  = "test-object.txt"
	testObjectSize  = 1024 * 1024 // 1 MB
	testChunkSizeMB = 1           // 1 MB chunks (not 256KB - config is in MB)
)

type sharedChunkCacheReaderTest struct {
	suite.Suite
	ctx        context.Context
	cacheDir   string
	manager    *file.SharedChunkCacheManager
	bucket     gcs.Bucket
	object     *gcs.MinObject
	reader     *SharedChunkCacheReader
	objectData []byte
}

func TestSharedChunkCacheReaderTestSuite(t *testing.T) {
	suite.Run(t, new(sharedChunkCacheReaderTest))
}

func (t *sharedChunkCacheReaderTest) SetupTest() {
	// Arrange - Create test fixtures
	t.ctx = context.Background()
	t.cacheDir = t.T().TempDir()

	// Create manager with 1 MB chunk size for testing
	config := &cfg.FileCacheConfig{
		SharedCacheChunkSizeMb: testChunkSizeMB,
	}
	var err error
	t.manager, err = file.NewSharedChunkCacheManager(t.cacheDir, 0644, 0755, config)
	require.NoError(t.T(), err)

	// Create fake bucket with test data
	t.bucket = fake.NewFakeBucket(timeutil.RealClock(), testBucketName, gcs.BucketType{})

	// Create test object data
	t.objectData = make([]byte, testObjectSize)
	for i := range t.objectData {
		t.objectData[i] = byte(i % 256)
	}

	// Create object in fake bucket
	createReq := &gcs.CreateObjectRequest{
		Name:     testObjectName,
		Contents: io.NopCloser(bytes.NewReader(t.objectData)),
	}
	createdObj, err := t.bucket.CreateObject(t.ctx, createReq)
	require.NoError(t.T(), err)

	t.object = &gcs.MinObject{
		Name:       createdObj.Name,
		Size:       createdObj.Size,
		Generation: createdObj.Generation,
	}

	// Create reader
	t.reader = NewSharedChunkCacheReader(
		t.manager,
		t.bucket,
		t.object,
		metrics.NewNoopMetrics(),
		tracing.NewNoopTracer(),
		0,
	)
}

func (t *sharedChunkCacheReaderTest) TearDownTest() {
	if t.cacheDir != "" {
		os.RemoveAll(t.cacheDir)
	}
}

func (t *sharedChunkCacheReaderTest) TestNewSharedChunkCacheReader() {
	// Arrange
	expectedBucketName := testBucketName
	expectedObjectName := t.object.Name
	expectedGeneration := t.object.Generation
	expectedSize := t.object.Size

	// Act - Reader is already created in SetupTest
	actualReader := t.reader

	// Assert
	assert.NotNil(t.T(), actualReader)
	assert.Equal(t.T(), expectedBucketName, actualReader.bucket.Name())
	assert.Equal(t.T(), expectedObjectName, actualReader.object.Name)
	assert.Equal(t.T(), expectedGeneration, actualReader.object.Generation)
	assert.Equal(t.T(), expectedSize, actualReader.object.Size)
}

func (t *sharedChunkCacheReaderTest) TestReadAt_SingleChunk() {
	// Arrange
	buffer := make([]byte, 100)
	req := &ReadRequest{
		Offset: 0,
		Buffer: buffer,
	}

	// Act
	resp, err := t.reader.ReadAt(t.ctx, req)

	// Assert
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 100, resp.Size)
	assert.Equal(t.T(), t.objectData[:100], buffer[:100])
}

func (t *sharedChunkCacheReaderTest) TestReadAt_CacheHit() {
	// Arrange - First read to populate cache
	buffer1 := make([]byte, 100)
	req1 := &ReadRequest{Offset: 0, Buffer: buffer1}
	_, err := t.reader.ReadAt(t.ctx, req1)
	require.NoError(t.T(), err)
	chunkPath := t.manager.GetChunkPath(testBucketName, t.object.Name, t.object.Generation, 0)
	require.FileExists(t.T(), chunkPath, "Chunk should be cached after first read")
	// Capture chunk file info after first read (cache miss)
	fileInfoBeforeCacheHit, err := os.Stat(chunkPath)
	require.NoError(t.T(), err)
	modTimeBeforeCacheHit := fileInfoBeforeCacheHit.ModTime()
	// Arrange - Prepare second read from cached chunk
	buffer2 := make([]byte, 100)
	req2 := &ReadRequest{Offset: 50, Buffer: buffer2}
	expectedData := t.objectData[50:150]

	// Act - Second read should be a cache hit
	resp, err := t.reader.ReadAt(t.ctx, req2)

	// Assert
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 100, resp.Size)
	assert.Equal(t.T(), expectedData, buffer2[:100])
	// Verify cache hit - chunk file should not have been modified
	fileInfoAfterCacheHit, err := os.Stat(chunkPath)
	require.NoError(t.T(), err)
	modTimeAfterCacheHit := fileInfoAfterCacheHit.ModTime()
	assert.Equal(t.T(), modTimeBeforeCacheHit, modTimeAfterCacheHit, "Cache hit: chunk file should not be re-downloaded")
}

func (t *sharedChunkCacheReaderTest) TestReadAt_AcrossChunkBoundary() {
	// Arrange
	largeObjectData := make([]byte, 3*1024*1024)
	for i := range largeObjectData {
		largeObjectData[i] = byte(i % 256)
	}
	createReq := &gcs.CreateObjectRequest{
		Name:     "large-object.txt",
		Contents: io.NopCloser(bytes.NewReader(largeObjectData)),
	}
	createdObj, err := t.bucket.CreateObject(t.ctx, createReq)
	require.NoError(t.T(), err)
	largeObject := &gcs.MinObject{
		Name:       createdObj.Name,
		Size:       createdObj.Size,
		Generation: createdObj.Generation,
	}
	largeReader := NewSharedChunkCacheReader(
		t.manager,
		t.bucket,
		largeObject,
		metrics.NewNoopMetrics(),
		tracing.NewNoopTracer(),
		0,
	)
	chunkSize := int(t.manager.GetChunkSize())
	buffer := make([]byte, 2000)
	offset := int64(chunkSize - 1000)
	req := &ReadRequest{
		Offset: offset,
		Buffer: buffer,
	}
	expectedData := largeObjectData[offset : offset+2000]

	// Act
	resp, err := largeReader.ReadAt(t.ctx, req)

	// Assert
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 2000, resp.Size)
	assert.Equal(t.T(), expectedData, buffer[:2000])
	chunk0Path := t.manager.GetChunkPath(testBucketName, largeObject.Name, largeObject.Generation, 0)
	chunk1Path := t.manager.GetChunkPath(testBucketName, largeObject.Name, largeObject.Generation, 1)
	assert.FileExists(t.T(), chunk0Path, "First chunk should be cached")
	assert.FileExists(t.T(), chunk1Path, "Second chunk should be cached")
}

func (t *sharedChunkCacheReaderTest) TestReadAt_EOF() {
	// Arrange
	buffer := make([]byte, 100)
	offsetAtEOF := int64(t.object.Size)
	req := &ReadRequest{
		Offset: offsetAtEOF,
		Buffer: buffer,
	}

	// Act
	_, err := t.reader.ReadAt(t.ctx, req)

	// Assert
	assert.Equal(t.T(), io.EOF, err, "Reading at EOF should return io.EOF")
}

func (t *sharedChunkCacheReaderTest) TestReadAt_NegativeOffset() {
	// Arrange
	buffer := make([]byte, 100)
	req := &ReadRequest{
		Offset: -10,
		Buffer: buffer,
	}

	// Act
	_, err := t.reader.ReadAt(t.ctx, req)

	// Assert
	assert.Error(t.T(), err)
	assert.Contains(t.T(), err.Error(), "negative offset")
}

func (t *sharedChunkCacheReaderTest) TestReadAt_PartialRead() {
	// Arrange
	buffer := make([]byte, 1000)
	offset := int64(t.object.Size - 100)
	req := &ReadRequest{
		Offset: offset,
		Buffer: buffer,
	}
	expectedSize := 100
	expectedData := t.objectData[offset:]

	// Act
	resp, err := t.reader.ReadAt(t.ctx, req)

	// Assert
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), expectedSize, resp.Size, "Should read only remaining 100 bytes")
	assert.Equal(t.T(), expectedData, buffer[:100])
}

func (t *sharedChunkCacheReaderTest) TestReadAt_ExcludedByRegex() {
	// Arrange
	excludeConfig := &cfg.FileCacheConfig{
		ExcludeRegex:           ".*\\.txt$",
		SharedCacheChunkSizeMb: testChunkSizeMB,
	}
	excludemanager, err := file.NewSharedChunkCacheManager(t.cacheDir, 0644, 0755, excludeConfig)
	require.NoError(t.T(), err)
	excludeReader := NewSharedChunkCacheReader(
		excludemanager,
		t.bucket,
		t.object,
		metrics.NewNoopMetrics(),
		tracing.NewNoopTracer(),
		0,
	)
	buffer := make([]byte, 100)
	req := &ReadRequest{Offset: 0, Buffer: buffer}

	// Act
	_, err = excludeReader.ReadAt(t.ctx, req)

	// Assert
	assert.ErrorIs(t.T(), err, FallbackToAnotherReader, "Files matching exclude regex should fallback to another reader")
}

func (t *sharedChunkCacheReaderTest) TestReadAt_MultipleChunks() {
	// Arrange
	largeObjectData := make([]byte, 5*1024*1024)
	for i := range largeObjectData {
		largeObjectData[i] = byte(i % 256)
	}
	createReq := &gcs.CreateObjectRequest{
		Name:     "multi-chunk-object.txt",
		Contents: io.NopCloser(bytes.NewReader(largeObjectData)),
	}
	createdObj, err := t.bucket.CreateObject(t.ctx, createReq)
	require.NoError(t.T(), err)
	largeObject := &gcs.MinObject{
		Name:       createdObj.Name,
		Size:       createdObj.Size,
		Generation: createdObj.Generation,
	}
	largeReader := NewSharedChunkCacheReader(
		t.manager,
		t.bucket,
		largeObject,
		metrics.NewNoopMetrics(),
		tracing.NewNoopTracer(),
		0,
	)
	buffer := make([]byte, largeObject.Size)
	req := &ReadRequest{
		Offset: 0,
		Buffer: buffer,
	}
	expectedSize := int(largeObject.Size)
	expectedData := largeObjectData
	chunkSize := int(t.manager.GetChunkSize())
	expectedNumChunks := (int(largeObject.Size) + chunkSize - 1) / chunkSize

	// Act
	resp, err := largeReader.ReadAt(t.ctx, req)

	// Assert
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), expectedSize, resp.Size)
	assert.Equal(t.T(), expectedData, buffer)
	for i := range expectedNumChunks {
		chunkPath := t.manager.GetChunkPath(testBucketName, largeObject.Name, largeObject.Generation, int64(i))
		assert.FileExists(t.T(), chunkPath, "Chunk %d should be cached", i)
	}
}

func (t *sharedChunkCacheReaderTest) TestReadAt_ZeroLengthRead() {
	// Arrange
	buffer := make([]byte, 0)
	req := &ReadRequest{
		Offset: 0,
		Buffer: buffer,
	}

	// Act
	resp, err := t.reader.ReadAt(t.ctx, req)

	// Assert
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 0, resp.Size)
}

func TestSharedChunkCacheReader_ConcurrentReads(t *testing.T) {
	// Arrange
	cacheDir := t.TempDir()
	config := &cfg.FileCacheConfig{
		SharedCacheChunkSizeMb: 1,
	}
	manager, err := file.NewSharedChunkCacheManager(cacheDir, 0644, 0755, config)
	require.NoError(t, err)
	bucket := fake.NewFakeBucket(timeutil.RealClock(), testBucketName, gcs.BucketType{})
	objectData := make([]byte, 5*1024*1024)
	for i := range objectData {
		objectData[i] = byte(i % 256)
	}
	ctx := context.Background()
	createReq := &gcs.CreateObjectRequest{
		Name:     testObjectName,
		Contents: io.NopCloser(bytes.NewReader(objectData)),
	}
	createdObj, err := bucket.CreateObject(ctx, createReq)
	require.NoError(t, err)
	object := &gcs.MinObject{
		Name:       createdObj.Name,
		Size:       createdObj.Size,
		Generation: createdObj.Generation,
	}
	reader := NewSharedChunkCacheReader(
		manager,
		bucket,
		object,
		metrics.NewNoopMetrics(),
		tracing.NewNoopTracer(),
		0,
	)
	wg := sync.WaitGroup{}
	const numGoroutines = 5
	wg.Add(numGoroutines)

	// Act - Launch concurrent reads
	for i := range numGoroutines {
		go func(offset int64) {
			defer wg.Done()
			buffer := make([]byte, 1000)
			req := &ReadRequest{Offset: offset, Buffer: buffer}
			resp, readErr := reader.ReadAt(ctx, req)
			assert.NoError(t, readErr)
			assert.Equal(t, 1000, resp.Size)
			assert.Equal(t, objectData[offset:offset+1000], buffer)
		}(int64(i * 100000))
	}

	// Assert - Wait for all goroutines to complete successfully
	wg.Wait()
	assert.True(t, true, "All concurrent reads completed successfully")
}

func TestSharedChunkCacheReader_ChunkRaceCondition(t *testing.T) {
	// Arrange
	cacheDir := t.TempDir()
	config := &cfg.FileCacheConfig{
		SharedCacheChunkSizeMb: 1,
	}
	manager, err := file.NewSharedChunkCacheManager(cacheDir, 0644, 0755, config)
	require.NoError(t, err)
	bucket := fake.NewFakeBucket(timeutil.RealClock(), testBucketName, gcs.BucketType{})
	objectData := make([]byte, 2*1024*1024)
	for i := range objectData {
		objectData[i] = byte(i % 256)
	}
	ctx := context.Background()
	createReq := &gcs.CreateObjectRequest{
		Name:     testObjectName,
		Contents: io.NopCloser(bytes.NewReader(objectData)),
	}
	createdObj, err := bucket.CreateObject(ctx, createReq)
	require.NoError(t, err)
	object := &gcs.MinObject{
		Name:       createdObj.Name,
		Size:       createdObj.Size,
		Generation: createdObj.Generation,
	}
	reader := NewSharedChunkCacheReader(
		manager,
		bucket,
		object,
		metrics.NewNoopMetrics(),
		tracing.NewNoopTracer(),
		0,
	)
	const numGoroutines = 5
	wg := sync.WaitGroup{}
	wg.Add(numGoroutines)
	expectedChunkPath := manager.GetChunkPath(testBucketName, object.Name, object.Generation, 0)

	// Act - Launch concurrent reads to same chunk
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			buffer := make([]byte, 100)
			req := &ReadRequest{Offset: 50, Buffer: buffer}
			resp, readErr := reader.ReadAt(ctx, req)
			assert.NoError(t, readErr)
			assert.Equal(t, 100, resp.Size)
			assert.Equal(t, objectData[50:150], buffer)
		}()
	}

	// Assert - All goroutines should complete and chunk should be cached once
	wg.Wait()
	assert.FileExists(t, expectedChunkPath, "Chunk should be cached exactly once despite race condition")
}

// TestSharedChunkCacheReader_DownloadChunkWithDeletedDirectory tests that the system
// handles directory deletion gracefully, recovering and completing the download successfully.
// This verifies resilience against concurrent LRU cache eviction scenarios.
func TestSharedChunkCacheReader_DownloadChunkWithDeletedDirectory(t *testing.T) {
	// Arrange
	cacheDir := t.TempDir()
	config := &cfg.FileCacheConfig{
		SharedCacheChunkSizeMb: 1,
	}
	manager, err := file.NewSharedChunkCacheManager(cacheDir, 0644, 0755, config)
	require.NoError(t, err)
	bucket := fake.NewFakeBucket(timeutil.RealClock(), testBucketName, gcs.BucketType{})
	objectData := make([]byte, 2*1024*1024) // 2 MB
	for i := range objectData {
		objectData[i] = byte(i % 256)
	}
	ctx := context.Background()
	createReq := &gcs.CreateObjectRequest{
		Name:     testObjectName,
		Contents: io.NopCloser(bytes.NewReader(objectData)),
	}
	createdObj, err := bucket.CreateObject(ctx, createReq)
	require.NoError(t, err)
	object := &gcs.MinObject{
		Name:       createdObj.Name,
		Size:       createdObj.Size,
		Generation: createdObj.Generation,
	}
	reader := NewSharedChunkCacheReader(
		manager,
		bucket,
		object,
		metrics.NewNoopMetrics(),
		tracing.NewNoopTracer(),
		0,
	)
	// Pre-delete the cache directory structure to simulate LRU eviction
	// The download should handle this and recreate necessary directories
	objDir := manager.GetObjectDir(testBucketName, object.Name, object.Generation)
	os.RemoveAll(objDir)

	// Act - Trigger download with missing directory structure
	buffer := make([]byte, 1024)
	req := &ReadRequest{
		Offset: 0,
		Buffer: buffer,
	}
	resp, err := reader.ReadAt(ctx, req)

	// Assert - Should succeed by recreating necessary directories
	assert.NoError(t, err, "Should handle missing directories gracefully")
	assert.Equal(t, 1024, resp.Size)
	assert.Equal(t, objectData[:1024], buffer)
	// Verify chunk was successfully cached after recreating directories
	chunkPath := manager.GetChunkPath(testBucketName, object.Name, object.Generation, 0)
	assert.FileExists(t, chunkPath)
}

// TestSharedChunkCacheReader_ReadAtFallbackOnDownloadError tests that ReadAt falls back
// to GCS reader when downloadChunk fails after retry.
func TestSharedChunkCacheReader_ReadAtFallbackOnDownloadError(t *testing.T) {
	// Arrange
	cacheDir := t.TempDir()
	// Make cache directory read-only to cause download failures
	err := os.Chmod(cacheDir, 0444)
	require.NoError(t, err)
	defer func() {
		_ = os.Chmod(cacheDir, 0755) // Restore for cleanup
	}()
	config := &cfg.FileCacheConfig{
		SharedCacheChunkSizeMb: 1,
	}
	manager, err := file.NewSharedChunkCacheManager(cacheDir, 0644, 0755, config)
	require.NoError(t, err)
	bucket := fake.NewFakeBucket(timeutil.RealClock(), testBucketName, gcs.BucketType{})
	objectData := make([]byte, 1024*1024) // 1 MB
	for i := range objectData {
		objectData[i] = byte(i % 256)
	}
	ctx := context.Background()
	createReq := &gcs.CreateObjectRequest{
		Name:     testObjectName,
		Contents: io.NopCloser(bytes.NewReader(objectData)),
	}
	createdObj, err := bucket.CreateObject(ctx, createReq)
	require.NoError(t, err)
	object := &gcs.MinObject{
		Name:       createdObj.Name,
		Size:       createdObj.Size,
		Generation: createdObj.Generation,
	}
	reader := NewSharedChunkCacheReader(
		manager,
		bucket,
		object,
		metrics.NewNoopMetrics(),
		tracing.NewNoopTracer(),
		0,
	)

	// Act - ReadAt should fail to cache and return FallbackToAnotherReader error
	buffer := make([]byte, 1024)
	req := &ReadRequest{
		Offset: 0,
		Buffer: buffer,
	}
	resp, err := reader.ReadAt(ctx, req)

	// Assert - Should return FallbackToAnotherReader due to permission denied on directory
	assert.Error(t, err)
	assert.ErrorIs(t, err, FallbackToAnotherReader)
	assert.Equal(t, 0, resp.Size)
}

// TestSharedChunkCacheReader_ReadAtWithCorruptedCache tests ReadAt behavior when
// cached chunk is corrupted (partial/incomplete).
func TestSharedChunkCacheReader_ReadAtWithCorruptedCache(t *testing.T) {
	// Arrange
	cacheDir := t.TempDir()
	config := &cfg.FileCacheConfig{
		SharedCacheChunkSizeMb: 1,
	}
	manager, err := file.NewSharedChunkCacheManager(cacheDir, 0644, 0755, config)
	require.NoError(t, err)

	bucket := fake.NewFakeBucket(timeutil.RealClock(), testBucketName, gcs.BucketType{})
	objectData := make([]byte, 2*1024*1024) // 2 MB
	for i := range objectData {
		objectData[i] = byte(i % 256)
	}
	ctx := context.Background()
	createReq := &gcs.CreateObjectRequest{
		Name:     testObjectName,
		Contents: io.NopCloser(bytes.NewReader(objectData)),
	}
	createdObj, err := bucket.CreateObject(ctx, createReq)
	require.NoError(t, err)
	object := &gcs.MinObject{
		Name:       createdObj.Name,
		Size:       createdObj.Size,
		Generation: createdObj.Generation,
	}
	reader := NewSharedChunkCacheReader(
		manager,
		bucket,
		object,
		metrics.NewNoopMetrics(),
		tracing.NewNoopTracer(),
		0,
	)
	// First, populate the cache normally
	buffer := make([]byte, 1024)
	req := &ReadRequest{
		Offset: 0,
		Buffer: buffer,
	}
	resp, err := reader.ReadAt(ctx, req)
	require.NoError(t, err)
	require.Equal(t, 1024, resp.Size)
	// Corrupt the cached chunk by truncating it
	chunkPath := manager.GetChunkPath(testBucketName, object.Name, object.Generation, 0)
	chunkFile, err := os.OpenFile(chunkPath, os.O_WRONLY|os.O_TRUNC, 0644)
	require.NoError(t, err)
	// Write only 512 bytes instead of full chunk
	_, err = chunkFile.Write(objectData[:512])
	require.NoError(t, err)
	chunkFile.Close()

	// Act - Try to read from corrupted cache (should fallback to GCS)
	buffer2 := make([]byte, 1024)
	req2 := &ReadRequest{
		Offset: 512, // Read beyond corrupted data
		Buffer: buffer2,
	}
	resp2, err2 := reader.ReadAt(ctx, req2)

	// Assert - Should detect corruption and fallback
	assert.Error(t, err2)
	assert.ErrorIs(t, err2, FallbackToAnotherReader)
	assert.Equal(t, 0, resp2.Size)
}

// TestSharedChunkCacheReader_DownloadChunkGCSError tests downloadChunk behavior
// when GCS reader creation fails.
func TestSharedChunkCacheReader_DownloadChunkGCSError(t *testing.T) {
	// Arrange
	cacheDir := t.TempDir()
	config := &cfg.FileCacheConfig{
		SharedCacheChunkSizeMb: 1,
	}
	manager, err := file.NewSharedChunkCacheManager(cacheDir, 0644, 0755, config)
	require.NoError(t, err)
	bucket := fake.NewFakeBucket(timeutil.RealClock(), testBucketName, gcs.BucketType{})
	ctx := context.Background()
	// Create object that doesn't exist in bucket
	object := &gcs.MinObject{
		Name:       "nonexistent-object.txt",
		Size:       1024 * 1024,
		Generation: 12345,
	}
	reader := NewSharedChunkCacheReader(
		manager,
		bucket,
		object,
		metrics.NewNoopMetrics(),
		tracing.NewNoopTracer(),
		0,
	)

	// Act - Try to read non-existent object
	buffer := make([]byte, 1024)
	req := &ReadRequest{
		Offset: 0,
		Buffer: buffer,
	}
	resp, err := reader.ReadAt(ctx, req)

	// Assert - Should fail and return FallbackToAnotherReader
	assert.Error(t, err)
	assert.ErrorIs(t, err, FallbackToAnotherReader)
	assert.Equal(t, 0, resp.Size)
}
