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
//
// File that contains tests specific to sparse downloads job.

package downloader

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
	testutil "github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	. "github.com/jacobsa/ogletest"
)

// TestSparseDownloader runs all tests for sparse file downloads
func TestSparseDownloader(t *testing.T) { RunTests(t) }

type sparseDownloaderTest struct {
	downloaderTest
}

func init() { RegisterTestSuite(&sparseDownloaderTest{}) }

func (dt *sparseDownloaderTest) SetUp(*TestInfo) {
	dt.defaultFileCacheConfig = &cfg.FileCacheConfig{
		ExperimentalEnablechunkCache:           true,
		DownloadChunkSizeMb:                    20, // 20MB chunks for sparse files
		EnableCrc:                              true,
		ExperimentalParallelDownloadsDefaultOn: true,
	}
	dt.setupHelper()
}

func (dt *sparseDownloaderTest) Test_calculateSparseChunkBoundaries() {
	tests := []struct {
		name           string
		offset         int64
		requiredOffset int64
		chunkSizeMb    int64
		objectSize     uint64
		expectedStart  uint64
		expectedEnd    uint64
		expectError    bool
	}{
		{
			name:           "single chunk - aligned",
			offset:         0,
			requiredOffset: 10 * util.MiB,
			chunkSizeMb:    20,
			objectSize:     100 * util.MiB,
			expectedStart:  0,
			expectedEnd:    20 * util.MiB,
			expectError:    false,
		},
		{
			name:           "single chunk - unaligned start",
			offset:         5 * util.MiB,
			requiredOffset: 10 * util.MiB,
			chunkSizeMb:    20,
			objectSize:     100 * util.MiB,
			expectedStart:  0, // rounds down to chunk boundary
			expectedEnd:    20 * util.MiB,
			expectError:    false,
		},
		{
			name:           "single chunk - unaligned end",
			offset:         15 * util.MiB,
			requiredOffset: 25 * util.MiB,
			chunkSizeMb:    20,
			objectSize:     100 * util.MiB,
			expectedStart:  0,
			expectedEnd:    40 * util.MiB, // rounds up to chunk boundary
			expectError:    false,
		},
		{
			name:           "chunk end capped at object size",
			offset:         90 * util.MiB,
			requiredOffset: 95 * util.MiB,
			chunkSizeMb:    20,
			objectSize:     100 * util.MiB,
			expectedStart:  80 * util.MiB,
			expectedEnd:    100 * util.MiB, // capped at object size, not 120MB
			expectError:    false,
		},
		{
			name:           "invalid range - offset >= requiredOffset",
			offset:         10,
			requiredOffset: 10,
			chunkSizeMb:    20,
			objectSize:     100 * util.MiB,
			expectError:    true,
		},
		{
			name:           "invalid range - negative offset",
			offset:         -1,
			requiredOffset: 10,
			chunkSizeMb:    20,
			objectSize:     100 * util.MiB,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		objectName := "test/sparse_chunk_boundaries.txt"
		objectContent := testutil.GenerateRandomBytes(int(tt.objectSize))
		dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, tt.objectSize, func() {})
		dt.job.fileCacheConfig.DownloadChunkSizeMb = tt.chunkSizeMb

		chunkStart, chunkEnd, err := dt.job.calculateSparseChunkBoundaries(tt.offset, tt.requiredOffset)

		if tt.expectError {
			AssertNe(nil, err, fmt.Sprintf("Test case %q: expected error but got none", tt.name))
		} else {
			AssertEq(nil, err, fmt.Sprintf("Test case %q: unexpected error: %v", tt.name, err))
			AssertEq(tt.expectedStart, chunkStart, fmt.Sprintf("Test case %q: chunkStart mismatch", tt.name))
			AssertEq(tt.expectedEnd, chunkEnd, fmt.Sprintf("Test case %q: chunkEnd mismatch", tt.name))
		}
	}
}

func (dt *sparseDownloaderTest) Test_DownloadRange() {
	objectName := "test/sparse_download_range.txt"
	objectSize := 50 * util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	var callbackExecuted atomic.Bool
	removeCallback := func() { callbackExecuted.Store(true) }
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize), removeCallback)

	// Set up sparse file mode
	dt.job.fileCacheConfig.ExperimentalEnableChunkCache = true
	dt.job.fileCacheConfig.DownloadChunkSizeMb = 20

	// Create the cache file
	file, err := util.CreateFile(data.FileSpec{
		Path:     dt.job.fileSpec.Path,
		FilePerm: os.FileMode(0600),
		DirPerm:  os.FileMode(0700),
	}, os.O_TRUNC|os.O_RDWR)
	AssertEq(nil, err)
	defer file.Close()

	// Set up sparse file info in cache
	fileInfoKey := data.FileInfoKey{
		BucketName: dt.bucket.Name(),
		ObjectName: objectName,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	AssertEq(nil, err)

	chunkSizeBytes := uint64(20) * 1024 * 1024
	fileInfo := data.FileInfo{
		Key:              fileInfoKey,
		ObjectGeneration: dt.object.Generation,
		Offset:           ^uint64(0), // MaxUint64 for sparse mode
		FileSize:         uint64(objectSize),
		SparseMode:       true,
		DownloadedRanges: data.NewByteRangeMap(chunkSizeBytes),
	}
	_, err = dt.cache.Insert(fileInfoKeyName, fileInfo)
	AssertEq(nil, err)

	// Download a range [10MB, 30MB)
	start := uint64(10 * util.MiB)
	end := uint64(30 * util.MiB)
	err = dt.job.downloadSparseRange(context.Background(), start, end)
	AssertEq(nil, err)

	// Verify the content was written correctly
	_, err = file.Seek(int64(start), 0)
	AssertEq(nil, err)
	buf := make([]byte, end-start)
	_, err = file.Read(buf)
	AssertEq(nil, err)
	AssertTrue(reflect.DeepEqual(objectContent[start:end], buf), "Downloaded content doesn't match")

	// Verify the downloaded range was tracked in ByteRangeMap
	updatedFileInfoVal := dt.cache.LookUpWithoutChangingOrder(fileInfoKeyName)
	AssertTrue(updatedFileInfoVal != nil, "FileInfo should exist in cache")
	updatedFileInfo := updatedFileInfoVal.(data.FileInfo)
	AssertTrue(updatedFileInfo.DownloadedRanges.ContainsRange(start, end), "Downloaded range not tracked in ByteRangeMap")
}

func (dt *sparseDownloaderTest) Test_HandleSparseRead_AlreadyDownloaded() {
	objectName := "test/sparse_already_downloaded.txt"
	objectSize := 50 * util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize), func() {})

	dt.job.fileCacheConfig.ExperimentalEnableChunkCache = true
	dt.job.fileCacheConfig.DownloadChunkSizeMb = 20

	// Set up sparse file info with pre-downloaded range
	fileInfoKey := data.FileInfoKey{
		BucketName: dt.bucket.Name(),
		ObjectName: objectName,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	AssertEq(nil, err)

	chunkSizeBytes := uint64(20) * 1024 * 1024
	downloadedRanges := data.NewByteRangeMap(chunkSizeBytes)
	downloadedRanges.AddRange(0, 40*util.MiB) // Mark first 40MB as downloaded

	fileInfo := data.FileInfo{
		Key:              fileInfoKey,
		ObjectGeneration: dt.object.Generation,
		Offset:           ^uint64(0),
		FileSize:         uint64(objectSize),
		SparseMode:       true,
		DownloadedRanges: downloadedRanges,
	}
	_, err = dt.cache.Insert(fileInfoKeyName, fileInfo)
	AssertEq(nil, err)

	// Request a range that's already downloaded [5MB, 10MB)
	offset := int64(5 * util.MiB)
	requiredOffset := int64(10 * util.MiB)
	cacheHit, err := dt.job.HandleSparseRead(context.Background(), offset, requiredOffset)

	AssertEq(nil, err)
	AssertTrue(cacheHit, "Should be a cache hit for already downloaded range")
}

func (dt *sparseDownloaderTest) Test_HandleSparseRead_NeedsDownload() {
	objectName := "test/sparse_needs_download.txt"
	objectSize := 100 * util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize), func() {})

	dt.job.fileCacheConfig.ExperimentalEnableChunkCache = true
	dt.job.fileCacheConfig.DownloadChunkSizeMb = 20

	// Create the cache file
	file, err := util.CreateFile(data.FileSpec{
		Path:     dt.job.fileSpec.Path,
		FilePerm: os.FileMode(0600),
		DirPerm:  os.FileMode(0700),
	}, os.O_TRUNC|os.O_RDWR)
	AssertEq(nil, err)
	defer file.Close()

	// Set up sparse file info with empty downloaded ranges
	fileInfoKey := data.FileInfoKey{
		BucketName: dt.bucket.Name(),
		ObjectName: objectName,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	AssertEq(nil, err)

	chunkSizeBytes := uint64(20) * 1024 * 1024
	fileInfo := data.FileInfo{
		Key:              fileInfoKey,
		ObjectGeneration: dt.object.Generation,
		Offset:           ^uint64(0),
		FileSize:         uint64(objectSize),
		SparseMode:       true,
		DownloadedRanges: data.NewByteRangeMap(chunkSizeBytes),
	}
	_, err = dt.cache.Insert(fileInfoKeyName, fileInfo)
	AssertEq(nil, err)

	// Request a range that needs to be downloaded [15MB, 25MB)
	offset := int64(15 * util.MiB)
	requiredOffset := int64(25 * util.MiB)
	cacheHit, err := dt.job.HandleSparseRead(context.Background(), offset, requiredOffset)

	AssertEq(nil, err)
	AssertTrue(cacheHit, "Should be a cache hit after successful download")

	// Verify the chunk was downloaded [0, 40MB) due to alignment
	// offset 15MB rounds down to 0, requiredOffset 25MB rounds up to 40MB
	updatedFileInfoVal := dt.cache.LookUpWithoutChangingOrder(fileInfoKeyName)
	AssertTrue(updatedFileInfoVal != nil, "FileInfo should exist in cache")
	updatedFileInfo := updatedFileInfoVal.(data.FileInfo)
	AssertTrue(updatedFileInfo.DownloadedRanges.ContainsRange(0, 40*util.MiB),
		"Expected range [0, 40MB) to be downloaded")

	// Verify the content
	_, err = file.Seek(int64(offset), 0)
	AssertEq(nil, err)
	buf := make([]byte, requiredOffset-offset)
	_, err = file.Read(buf)
	AssertEq(nil, err)
	AssertTrue(reflect.DeepEqual(objectContent[offset:requiredOffset], buf),
		"Downloaded content doesn't match")
}
