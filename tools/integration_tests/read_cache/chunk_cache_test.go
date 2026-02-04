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

package read_cache

import (
	"context"
	"log"
	"os"
	"path"
	"sync"
	"syscall"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func validateAllocatedFileSize(t *testing.T, fileName string, expectedSize int64) {
	cachedFilePath := getCachedFilePath(fileName)
	fi, err := os.Stat(cachedFilePath)
	require.NoError(t, err)
	stat := fi.Sys().(*syscall.Stat_t)
	allocatedSize := stat.Blocks * 512

	// Allow small overhead (1KB) for metadata stored for this file.
	overhead := int64(1024)
	assert.LessOrEqual(t, allocatedSize, expectedSize+overhead, "Allocated size should be close to expected size")
	assert.GreaterOrEqual(t, allocatedSize, expectedSize, "Allocated size should be at least expected size")
}

func validateSparse(expected *Expected, logEntry *read_logs.StructuredReadLogEntry, cacheHit bool, chunkCount int, t *testing.T) {
	// We ignore IsSequential check for sparse cache tests by passing the actual value from logs.
	isSeq := false
	if len(logEntry.Chunks) > 0 {
		isSeq = logEntry.Chunks[0].IsSequential
	}
	validate(expected, logEntry, isSeq, cacheHit, chunkCount, t)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

type chunkCacheTest struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
	baseTestName  string
	suite.Suite
}

func (s *chunkCacheTest) SetupSuite() {
	setupLogFileAndCacheDir(s.baseTestName)
	mountGCSFuseAndSetupTestDir(s.flags, s.ctx, s.storageClient)
}

func (s *chunkCacheTest) SetupTest() {
	//Truncate log file created.
	err := os.Truncate(testEnv.cfg.LogFile, 0)
	require.NoError(s.T(), err)
	// Clean up the cache directory path as gcsfuse don't clean up on mounting.
	operations.RemoveDir(testEnv.cacheDirPath)
	testEnv.testDirPath = client.SetupTestDirectory(s.ctx, s.storageClient, testDirName)
}

func (s *chunkCacheTest) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
}

func (s *chunkCacheTest) TearDownSuite() {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
}

func (s *chunkCacheTest) TestSparseRandomRead() {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, 50*util.MiB, s.T())

	// Read at 15MB offset (Chunk [10, 20))
	offset15 := int64(15 * util.MiB)
	expectedOutcome1 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offset15, s.T())

	// Read at 5MB offset (Chunk [0, 10))
	offset5 := int64(5 * util.MiB)
	expectedOutcome2 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offset5, s.T())

	// Read at 10MB offset (Chunk [10, 20)) - Should be cache hit
	offset10 := int64(10 * util.MiB)
	expectedOutcome3 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offset10, s.T())

	structuredLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	require.Len(s.T(), structuredLogs, 3)
	validateSparse(expectedOutcome1, structuredLogs[0], true, 1, s.T()) // First read is a cold read, now a cache hit.
	validateSparse(expectedOutcome2, structuredLogs[1], true, 1, s.T()) // Second read is a cold read, now a cache hit.
	validateSparse(expectedOutcome3, structuredLogs[2], true, 1, s.T()) // Third read is a warm read, a cache hit.

	jobLogs := read_logs.GetJobLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	require.NotEmpty(s.T(), jobLogs)
	downloads := jobLogs[0].SparseDownloads
	// We expect downloads for [10, 20) and [0, 10)
	// The order depends on execution, but we did 15MB first, so [10, 20) should be first.
	require.Len(s.T(), downloads, 2, "Expected exactly two downloads")
	count10_20 := 0
	count0_10 := 0

	for _, d := range downloads {
		if d.StartOffset == 10*util.MiB && d.EndOffset == 20*util.MiB {
			count10_20++
		}
		if d.StartOffset == 0 && d.EndOffset == 10*util.MiB {
			count0_10++
		}
	}
	assert.Equal(s.T(), 1, count10_20, "Expected exactly one download for range [10, 20)")
	assert.Equal(s.T(), 1, count0_10, "Expected exactly one download for range [0, 10)")
}

func (s *chunkCacheTest) TestFullSequentialSparseRead() {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, 30*util.MiB, s.T())

	// Read entire file
	expectedOutcome := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, 30*util.MiB, false, s.T())

	structuredLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	require.Len(s.T(), structuredLogs, 1)
	validateSparse(expectedOutcome, structuredLogs[0], true, int(30*util.MiB/chunkSizeToRead), s.T()) // Cold read is a cache hit.

	// Verify logs show sequential downloads
	jobLogs := read_logs.GetJobLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	require.NotEmpty(s.T(), jobLogs)
	downloads := jobLogs[0].SparseDownloads
	// Expect [0, 10), [10, 20), [20, 30)
	require.Len(s.T(), downloads, 3, "Expected exactly three downloads")
	count0_10 := 0
	count10_20 := 0
	count20_30 := 0
	for _, d := range downloads {
		if d.StartOffset == 0 && d.EndOffset == 10*util.MiB {
			count0_10++
		}
		if d.StartOffset == 10*util.MiB && d.EndOffset == 20*util.MiB {
			count10_20++
		}
		if d.StartOffset == 20*util.MiB && d.EndOffset == 30*util.MiB {
			count20_30++
		}
	}
	assert.Equal(s.T(), 1, count0_10, "Expected exactly one download for range [0, 10)")
	assert.Equal(s.T(), 1, count10_20, "Expected exactly one download for range [10, 20)")
	assert.Equal(s.T(), 1, count20_30, "Expected exactly one download for range [20, 30)")
}

func (s *chunkCacheTest) TestMergedRangeEfficiency() {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, 30*util.MiB, s.T())

	// Read at 15MB (fetches [10, 20))
	expectedOutcome1 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, 15*util.MiB, s.T())

	// Read entire file
	expectedOutcome2 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, 30*util.MiB, false, s.T())

	structuredLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	require.Len(s.T(), structuredLogs, 2)
	validateSparse(expectedOutcome1, structuredLogs[0], true, 1, s.T())                                // Cold read is a cache hit.
	validateSparse(expectedOutcome2, structuredLogs[1], true, int(30*util.MiB/chunkSizeToRead), s.T()) // Subsequent read is also a cache hit.

	// Verify logs: [10, 20) should be skipped in the second read's downloads
	// Total downloads should be [10, 20) (from first read), then [0, 10) and [20, 30) (from second read)
	jobLogs := read_logs.GetJobLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	require.NotEmpty(s.T(), jobLogs)
	downloads := jobLogs[0].SparseDownloads
	require.Len(s.T(), downloads, 3, "Expected exactly three downloads")
	count0_10 := 0
	count10_20 := 0
	count20_30 := 0
	for _, d := range downloads {
		if d.StartOffset == 0 && d.EndOffset == 10*util.MiB {
			count0_10++
		}
		if d.StartOffset == 10*util.MiB && d.EndOffset == 20*util.MiB {
			count10_20++
		}
		if d.StartOffset == 20*util.MiB && d.EndOffset == 30*util.MiB {
			count20_30++
		}
	}
	assert.Equal(s.T(), 1, count0_10, "Expected exactly one download for range [0, 10)")
	assert.Equal(s.T(), 1, count10_20, "Expected exactly one download for range [10, 20)")
	assert.Equal(s.T(), 1, count20_30, "Expected exactly one download for range [20, 30)")
}

func (s *chunkCacheTest) TestReadSpanningTwoBlocks() {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, 30*util.MiB, s.T())

	// Read 1MB spanning 9.5MB to 10.5MB
	offset := int64(9.5 * util.MiB)
	expectedOutcome := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offset, s.T())

	structuredLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	require.Len(s.T(), structuredLogs, 1)
	validateSparse(expectedOutcome, structuredLogs[0], true, 1, s.T()) // Cold read is a cache hit.

	// Verify downloads for [0, 10) and [10, 20)
	jobLogs := read_logs.GetJobLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	require.NotEmpty(s.T(), jobLogs)
	downloads := jobLogs[0].SparseDownloads
	require.Len(s.T(), downloads, 2, "Expected exactly two downloads")
	count0_10 := 0
	count10_20 := 0
	for _, d := range downloads {
		if d.StartOffset == 0 && d.EndOffset == 10*util.MiB {
			count0_10++
		}
		if d.StartOffset == 10*util.MiB && d.EndOffset == 20*util.MiB {
			count10_20++
		}
	}
	assert.Equal(s.T(), 1, count0_10, "Expected exactly one download for range [0, 10)")
	assert.Equal(s.T(), 1, count10_20, "Expected exactly one download for range [10, 20)")
}

func (s *chunkCacheTest) TestConcurrentDeduplication() {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, 30*util.MiB, s.T())
	offset := int64(5 * util.MiB)
	concurrency := 8
	outcomes := make([]*Expected, concurrency)
	var wg sync.WaitGroup
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func(i int) {
			defer wg.Done()
			outcomes[i] = readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offset, s.T())
		}(i)
	}
	wg.Wait()

	structuredLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	require.Len(s.T(), structuredLogs, concurrency)
	for i := 0; i < concurrency; i++ {
		validateSparse(outcomes[i], structuredLogs[i], true, 1, s.T()) // All concurrent reads are cache hits.
	}

	// Verify only one download for [0, 10)
	jobLogs := read_logs.GetJobLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	require.NotEmpty(s.T(), jobLogs)
	downloads := jobLogs[0].SparseDownloads
	require.Len(s.T(), downloads, 1, "Expected exactly one download")
	count := 0
	for _, d := range downloads {
		if d.StartOffset == 0 && d.EndOffset == 10*util.MiB {
			count++
		}
	}
	assert.Equal(s.T(), 1, count, "Expected exactly one download for range [0, 10)")
}

func (s *chunkCacheTest) TestAccuratePartialLastChunk() {
	// Adapted to 11MB file with 10MB chunk size (suite default)
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, 11*util.MiB, s.T())

	// Read last 1MB
	offset := int64(10 * util.MiB)
	expectedOutcome := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offset, s.T())

	structuredLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	require.Len(s.T(), structuredLogs, 1)
	validateSparse(expectedOutcome, structuredLogs[0], true, 1, s.T()) // Cold read is a cache hit.

	// Verify download range [10, 11)
	jobLogs := read_logs.GetJobLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	require.NotEmpty(s.T(), jobLogs)
	downloads := jobLogs[0].SparseDownloads
	require.Len(s.T(), downloads, 1, "Expected exactly one download")
	count := 0
	for _, d := range downloads {
		if d.StartOffset == 10*util.MiB && d.EndOffset == 11*util.MiB {
			count++
		}
	}
	assert.Equal(s.T(), 1, count, "Expected exactly one download for range [10MB, 11MB)")
}

func (s *chunkCacheTest) TestFallback() {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, 50*util.MiB, s.T())

	// 1. Read first chunk to ensure cache file is created and populated.
	expectedOutcome1 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, 0, s.T())

	// 2. Delete the file from GCS to trigger a failure during the next download attempt.
	err := client.DeleteObjectOnGCS(s.ctx, s.storageClient, path.Join(testDirName, testFileName))
	require.NoError(s.T(), err)

	// 3. Read second chunk. This should fail because the source object is gone.
	// The cache download will fail, triggering a fallback to GCS, which also fails.
	offset10 := int64(10 * util.MiB)
	_, err = operations.ReadChunkFromFile(path.Join(testEnv.testDirPath, testFileName), chunkSizeToRead, offset10, os.O_RDONLY|syscall.O_DIRECT)
	require.Error(s.T(), err, "Read should fail after GCS object is deleted")
	assert.Contains(s.T(), err.Error(), "input/output error")

	// Verify logs: first read is a cache hit (cold read), second read fails.
	structuredLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	require.Len(s.T(), structuredLogs, 1, "Only the first successful read should produce a structured log entry")
	validateSparse(expectedOutcome1, structuredLogs[0], true, 1, s.T())

	// 4. Verify logs
	logContent, err := os.ReadFile(testEnv.cfg.LogFile)
	require.NoError(s.T(), err)
	assert.Contains(s.T(), string(logContent), "Sparse file read failed")
	assert.Contains(s.T(), string(logContent), "Falling back to GCS")
	assert.Contains(s.T(), string(logContent), "storage: object doesn't exist")
}

func (s *chunkCacheTest) TestSparseCacheFileAllocatedSize() {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, 21*util.MiB, s.T())

	// Read at 20MB offset (Chunk [20, 30))
	// File size is 21MB, so this chunk is 1MB (20-21MB).
	offset20 := int64(20 * util.MiB)
	expectedOutcome1 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offset20, s.T())

	// Verify allocated size is ~1MB
	validateAllocatedFileSize(s.T(), testFileName, 1*util.MiB)

	// Read at 0MB offset (Chunk [0, 10))
	expectedOutcome2 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, 0, s.T())

	structuredLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	require.Len(s.T(), structuredLogs, 2)
	validateSparse(expectedOutcome1, structuredLogs[0], true, 1, s.T()) // Cold read is a cache hit.
	validateSparse(expectedOutcome2, structuredLogs[1], true, 1, s.T()) // Cold read is a cache hit.

	// Verify allocated size is ~11MB (1MB + 10MB)
	validateAllocatedFileSize(s.T(), testFileName, 11*util.MiB)
}

func TestChunkCacheTest(t *testing.T) {
	setupLogFileAndCacheDir(t.Name())
	// Run tests with sparse cache enabled and chunk size 10MB
	flagsSet := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	sparseFlags := []string{"--file-cache-experimental-enable-chunk-cache=true", "--file-cache-download-chunk-size-mb=10", "--cache-dir=" + testEnv.cacheDirPath, "--log-severity=TRACE"}

	for _, flags := range flagsSet {
		ts := &chunkCacheTest{
			ctx:           context.Background(),
			storageClient: testEnv.storageClient,
			baseTestName:  t.Name(),
			flags:         append(flags, sparseFlags...),
		}
		log.Printf("Running tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}

type chunkCacheDisabledTest struct {
	chunkCacheTest
}

func (s *chunkCacheDisabledTest) TestNormalFileCache_SparseDisabled() {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, 50*util.MiB, s.T())

	expectedOutcome := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, 0, s.T())

	// Verify cache miss for normal file cache cold read
	structuredLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	require.Len(s.T(), structuredLogs, 1)
	validateSparse(expectedOutcome, structuredLogs[0], false, 1, s.T()) // Cold read is a cache hit.

	jobLogs := read_logs.GetJobLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	require.Len(s.T(), jobLogs, 1, "Job logs should have exactly 1 entry")
	assert.Empty(s.T(), jobLogs[0].SparseDownloads, "Should not have sparse downloads")
	assert.NotEmpty(s.T(), jobLogs[0].JobEntries, "Should have normal file cache downloads")
}

func TestChunkCacheTestDisabled(t *testing.T) {
	setupLogFileAndCacheDir(t.Name())
	flagsSet := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	disabledFlags := []string{"--file-cache-experimental-enable-chunk-cache=false", "--cache-dir=" + testEnv.cacheDirPath, "--log-severity=TRACE"}

	for _, flags := range flagsSet {
		ts := &chunkCacheDisabledTest{
			chunkCacheTest: chunkCacheTest{
				ctx:           context.Background(),
				storageClient: testEnv.storageClient,
				baseTestName:  t.Name(),
				flags:         append(flags, disabledFlags...),
			},
		}
		log.Printf("Running tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}

type chunkCacheSmallCapacityTest struct {
	chunkCacheTest
}

func (s *chunkCacheSmallCapacityTest) TestSmallCapacityEviction() {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, 50*util.MiB, s.T())

	// Read first chunk [0, 10)
	expectedOutcome1 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, 0, s.T())

	// Read second chunk [10, 20) - should evict first chunk
	expectedOutcome2 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, 10*util.MiB, s.T())

	// Read first chunk again - should trigger download again
	expectedOutcome3 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, 0, s.T())

	structuredLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	require.Len(s.T(), structuredLogs, 3)
	validateSparse(expectedOutcome1, structuredLogs[0], true, 1, s.T()) // Cold read is a cache hit.
	validateSparse(expectedOutcome2, structuredLogs[1], true, 1, s.T()) // Cold read is a cache hit.
	validateSparse(expectedOutcome3, structuredLogs[2], true, 1, s.T()) // Cold read (after eviction) is a cache hit.

	// Verify logs show 3 downloads
	jobLogs := read_logs.GetJobLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	require.NotEmpty(s.T(), jobLogs)
	downloads := jobLogs[0].SparseDownloads
	require.Len(s.T(), downloads, 3, "Expected exactly three downloads")
	count0_10 := 0
	count10_20 := 0
	for _, d := range downloads {
		if d.StartOffset == 0 {
			count0_10++
		}
		if d.StartOffset == 10*util.MiB {
			count10_20++
		}
	}
	assert.Equal(s.T(), 2, count0_10, "Expected [0, 10) to be downloaded twice due to eviction")
	assert.Equal(s.T(), 1, count10_20, "Expected [10, 20) to be downloaded once")
}

func TestChunkCacheSmallCapacity(t *testing.T) {
	setupLogFileAndCacheDir(t.Name())
	flagsSet := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	// Cache size 15MB. Chunk size 10MB.
	smallCapacityFlags := []string{"--file-cache-experimental-enable-chunk-cache=true", "--file-cache-download-chunk-size-mb=10", "--file-cache-max-size-mb=15", "--cache-dir=" + testEnv.cacheDirPath, "--log-severity=TRACE"}

	for _, flags := range flagsSet {
		ts := &chunkCacheSmallCapacityTest{
			chunkCacheTest: chunkCacheTest{
				ctx:           context.Background(),
				storageClient: testEnv.storageClient,
				baseTestName:  t.Name(),
				flags:         append(flags, smallCapacityFlags...),
			},
		}
		log.Printf("Running tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}
