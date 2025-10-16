// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type cacheFileForRangeReadFalseTest struct {
	flags                      []string
	storageClient              *storage.Client
	ctx                        context.Context
	isParallelDownloadsEnabled bool
	isCacheOnRAM               bool
	baseTestName               string
	suite.Suite
}

func (s *cacheFileForRangeReadFalseTest) SetupSuite() {
	setupLogFileAndCacheDir(s.baseTestName)
	if s.isCacheOnRAM {
		testEnv.cacheDirPath = "/dev/shm/" + s.baseTestName
	}
	mountGCSFuseAndSetupTestDir(s.flags, s.ctx, s.storageClient)
}

func (s *cacheFileForRangeReadFalseTest) SetupTest() {
	//Truncate log file created.
	err := os.Truncate(testEnv.cfg.LogFile, 0)
	require.NoError(s.T(), err)
	// Clean up the cache directory path as gcsfuse don't clean up on mounting.
	operations.RemoveDir(testEnv.cacheDirPath)
	testEnv.testDirPath = client.SetupTestDirectory(s.ctx, s.storageClient, testDirName)
}

func (s *cacheFileForRangeReadFalseTest) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
}

func (s *cacheFileForRangeReadFalseTest) TearDownSuite() {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func readFileBetweenOffset(t *testing.T, file *os.File, startOffset, endOffSet int64) *Expected {
	t.Helper()
	expected := &Expected{
		StartTimeStampSeconds: time.Now().Unix(),
		BucketName:            setup.TestBucket(),
		ObjectName:            path.Join(testDirName, path.Base(file.Name())),
	}
	if setup.DynamicBucketMounted() != "" {
		expected.BucketName = setup.DynamicBucketMounted()
	}

	expected.content = operations.ReadFileBetweenOffset(t, file, startOffset, endOffSet, chunkSizeToRead)
	expected.EndTimeStampSeconds = time.Now().Unix()
	return expected
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *cacheFileForRangeReadFalseTest) TestRangeReadsWithCacheMiss() {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, fileSizeForRangeRead, s.T())

	// Do a random read on file and validate from gcs.
	expectedOutcome1 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offset5000, s.T())
	// Read file again from offset 1000 and validate from gcs.
	expectedOutcome2 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offset1000, s.T())

	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	validate(expectedOutcome1, structuredReadLogs[0], false, false, 1, s.T())
	validate(expectedOutcome2, structuredReadLogs[1], false, false, 1, s.T())
	validateFileIsNotCached(testFileName, s.T())
}

func (s *cacheFileForRangeReadFalseTest) TestReadIsTreatedNonSequentialAfterFileIsRemovedFromCache() {
	var testFileNames [2]string
	var expectedOutcome [4]*Expected
	testFileNames[0] = setupFileInTestDir(s.ctx, s.storageClient, fileSizeSameAsCacheCapacity, s.T())
	testFileNames[1] = setupFileInTestDir(s.ctx, s.storageClient, fileSizeSameAsCacheCapacity, s.T())
	randomReadChunkCount := fileSizeSameAsCacheCapacity / chunkSizeToRead
	readTillChunk := randomReadChunkCount / 2
	fh1 := operations.OpenFile(path.Join(testEnv.testDirPath, testFileNames[0]), s.T())
	defer operations.CloseFileShouldNotThrowError(s.T(), fh1)
	fh2 := operations.OpenFile(path.Join(testEnv.testDirPath, testFileNames[1]), s.T())
	defer operations.CloseFileShouldNotThrowError(s.T(), fh2)

	// Use file handle 1 to read file 1 partially.
	expectedOutcome[0] = readFileBetweenOffset(s.T(), fh1, 0, int64(readTillChunk*chunkSizeToRead))
	// Use file handle 2 to read file 2 partially. This will evict file 1 from
	// cache due to cache capacity constraints.
	expectedOutcome[1] = readFileBetweenOffset(s.T(), fh2, 0, int64(readTillChunk*chunkSizeToRead))
	// Read remaining file 1. File 2 remains cached. Cache eviction happens on
	// cache handler creation, which is tied to the file handle. Since the handle
	// isn't recreated, eviction doesn't occur.
	expectedOutcome[2] = readFileBetweenOffset(s.T(), fh1, int64(readTillChunk*chunkSizeToRead)+1, fileSizeSameAsCacheCapacity)
	// Read remaining file 2.
	expectedOutcome[3] = readFileBetweenOffset(s.T(), fh2, int64(readTillChunk*chunkSizeToRead)+1, fileSizeSameAsCacheCapacity)

	// Merge the expected outcomes.
	expectedOutcome[0].EndTimeStampSeconds = expectedOutcome[2].EndTimeStampSeconds
	expectedOutcome[0].content = expectedOutcome[0].content + expectedOutcome[2].content
	expectedOutcome[1].EndTimeStampSeconds = expectedOutcome[3].EndTimeStampSeconds
	expectedOutcome[1].content = expectedOutcome[1].content + expectedOutcome[3].content
	// Parse the logs and validate with expected outcome.
	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	require.Equal(s.T(), 2, len(structuredReadLogs))
	validate(expectedOutcome[0], structuredReadLogs[0], true, false, randomReadChunkCount, s.T())
	validate(expectedOutcome[1], structuredReadLogs[1], true, false, randomReadChunkCount, s.T())
	// Validate after cache eviction, read was considered non-sequential and cache
	// hit false for first file.
	// Checking for the last chunk, not readTillChunk+1, due to potential kernel
	// over-reads on some architectures.
	assert.False(s.T(), structuredReadLogs[0].Chunks[randomReadChunkCount-1].IsSequential)
	assert.False(s.T(), structuredReadLogs[0].Chunks[randomReadChunkCount-1].CacheHit)
	// Validate for 2nd file read was considered sequential because of no cache eviction.
	assert.True(s.T(), structuredReadLogs[1].Chunks[randomReadChunkCount-1].IsSequential)
	if !s.isParallelDownloadsEnabled {
		// When parallel downloads are enabled, we can't concretely say that the read will be cache Hit.
		assert.True(s.T(), structuredReadLogs[1].Chunks[randomReadChunkCount-1].CacheHit)
	}

	validateFileIsNotCached(testFileNames[0], s.T())
	validateFileInCacheDirectory(testFileNames[1], fileSizeSameAsCacheCapacity, s.ctx, s.storageClient, s.T())
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func (s *cacheFileForRangeReadFalseTest) runTests(t *testing.T) {
	t.Helper()
	// Run tests for mounted directory if the flag is set. This assumes that run flag is properly passed by GKE team as per the config.
	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		suite.Run(t, s)
		return
	}

	// Run tests for GCE environment otherwise.
	flagsSet := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	for _, s.flags = range flagsSet {
		log.Printf("Running tests with flags: %s", s.flags)
		suite.Run(t, s)
	}
}

func TestCacheFileForRangeReadFalseTest(t *testing.T) {
	ts := &cacheFileForRangeReadFalseTest{
		ctx:           context.Background(),
		storageClient: testEnv.storageClient,
		baseTestName:  t.Name(),
	}
	ts.runTests(t)
}

func TestCacheFileForRangeReadFalseWithRamCache(t *testing.T) {
	ts := &cacheFileForRangeReadFalseTest{
		ctx:           context.Background(),
		storageClient: testEnv.storageClient,
		baseTestName:  t.Name(),
		isCacheOnRAM:  true,
	}
	ts.runTests(t)
}

func TestCacheFileForRangeReadFalseWithParallelDownloads(t *testing.T) {
	ts := &cacheFileForRangeReadFalseTest{
		ctx:                        context.Background(),
		storageClient:              testEnv.storageClient,
		baseTestName:               t.Name(),
		isParallelDownloadsEnabled: true,
	}
	ts.runTests(t)
}

func TestCacheFileForRangeReadFalseWithParallelDownloadsAndRamCache(t *testing.T) {
	ts := &cacheFileForRangeReadFalseTest{
		ctx:                        context.Background(),
		storageClient:              testEnv.storageClient,
		baseTestName:               t.Name(),
		isParallelDownloadsEnabled: true,
		isCacheOnRAM:               true,
	}
	ts.runTests(t)
}
