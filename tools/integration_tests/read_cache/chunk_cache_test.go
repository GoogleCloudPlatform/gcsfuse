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
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/data"
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

func (s *chunkCacheTest) TestRandomRead() {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, 20*util.MiB, s.T())

	expectedOutcome1 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, 15*util.MiB, s.T())
	expectedOutcome2 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, 5*util.MiB, s.T())
	expectedOutcome3 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, 10*util.MiB, s.T())

	structuredLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	require.Len(s.T(), structuredLogs, 3)
	validate(expectedOutcome1, structuredLogs[0], false, true, 1, s.T())
	validate(expectedOutcome2, structuredLogs[1], false, true, 1, s.T())
	validate(expectedOutcome3, structuredLogs[2], false, true, 1, s.T())
	jobLogs := read_logs.GetJobLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	require.NotEmpty(s.T(), jobLogs)
	expectedDownloads := []data.ObjectRange{
		{Start: 10 * util.MiB, End: 20 * util.MiB},
		{Start: 0, End: 10 * util.MiB},
	}
	validateDownloads(s.T(), jobLogs[0].ChunkCacheDownloads, expectedDownloads, testFileName)
}

func (s *chunkCacheTest) TestFullSequentialRead() {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, 30*util.MiB, s.T())

	expectedOutcome := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, 30*util.MiB, false, s.T())

	structuredLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	require.Len(s.T(), structuredLogs, 1)
	validate(expectedOutcome, structuredLogs[0], true, true, int(30*util.MiB/chunkSizeToRead), s.T())
	jobLogs := read_logs.GetJobLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	require.NotEmpty(s.T(), jobLogs)
	expectedDownloads := []data.ObjectRange{
		{Start: 0, End: 10 * util.MiB},
		{Start: 10 * util.MiB, End: 20 * util.MiB},
		{Start: 20 * util.MiB, End: 30 * util.MiB},
	}
	validateDownloads(s.T(), jobLogs[0].ChunkCacheDownloads, expectedDownloads, testFileName)
}

func (s *chunkCacheTest) TestSequentialReadWithCachedChunk() {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, 30*util.MiB, s.T())

	expectedOutcome1 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, 15*util.MiB, s.T())
	expectedOutcome2 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, 30*util.MiB, false, s.T())

	structuredLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	require.Len(s.T(), structuredLogs, 2)
	validate(expectedOutcome1, structuredLogs[0], false, true, 1, s.T())
	validate(expectedOutcome2, structuredLogs[1], true, true, int(30*util.MiB/chunkSizeToRead), s.T())
	jobLogs := read_logs.GetJobLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	require.NotEmpty(s.T(), jobLogs)
	// Since [10, 20) is already downloaded in the first read, we expect that we don't download it again.
	expectedDownloads := []data.ObjectRange{
		{Start: 10 * util.MiB, End: 20 * util.MiB},
		{Start: 0, End: 10 * util.MiB},
		{Start: 20 * util.MiB, End: 30 * util.MiB},
	}
	validateDownloads(s.T(), jobLogs[0].ChunkCacheDownloads, expectedDownloads, testFileName)
}

func (s *chunkCacheTest) TestReadSpanningTwoBlocks() {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, 30*util.MiB, s.T())

	content, err := operations.ReadChunkFromFile(path.Join(testEnv.testDirPath, testFileName), 1*util.MiB, 19.5*util.MiB, os.O_RDONLY|syscall.O_DIRECT)

	require.NoError(s.T(), err)
	client.ValidateObjectChunkFromGCS(s.ctx, s.storageClient, testDirName, testFileName, 19.5*util.MiB, 1*util.MiB, string(content), s.T())
	jobLogs := read_logs.GetJobLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	require.NotEmpty(s.T(), jobLogs)
	expectedDownloads := []data.ObjectRange{
		{Start: 10 * util.MiB, End: 20 * util.MiB},
		{Start: 20 * util.MiB, End: 30 * util.MiB},
	}
	validateDownloads(s.T(), jobLogs[0].ChunkCacheDownloads, expectedDownloads, testFileName)
}

func (s *chunkCacheTest) TestConcurrentDeduplication() {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, 10*util.MiB, s.T())
	outcomes := make([]*Expected, 8)
	var wg sync.WaitGroup
	wg.Add(8)

	for i := 0; i < 8; i++ {
		go func(i int) {
			defer wg.Done()
			outcomes[i] = readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, 5*util.MiB, s.T())
		}(i)
	}
	wg.Wait()

	structuredLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	require.Len(s.T(), structuredLogs, 8)
	for i := 0; i < 8; i++ {
		validate(outcomes[i], structuredLogs[i], false, true, 1, s.T())
	}
	jobLogs := read_logs.GetJobLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	require.NotEmpty(s.T(), jobLogs)
	// Despite 8 concurrent reads from the same chunk, it should be downloaded only once.
	expectedDownloads := []data.ObjectRange{
		{Start: 0, End: 10 * util.MiB},
	}
	validateDownloads(s.T(), jobLogs[0].ChunkCacheDownloads, expectedDownloads, testFileName)
}

func (s *chunkCacheTest) TestReadOfDeletedfile() {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, 20*util.MiB, s.T())
	// Read first chunk to ensure cache file is created and populated.
	readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, 0, s.T())
	// Delete the file from GCS to trigger a failure during the next download attempt.
	_, objectName := setup.GetBucketAndObjectBasedOnTypeOfMount(path.Join(testDirName, testFileName))
	err := client.DeleteObjectOnGCS(s.ctx, s.storageClient, objectName)
	require.NoError(s.T(), err)

	// Read second chunk. This should fail because the source object is gone.
	_, err = operations.ReadChunkFromFile(path.Join(testEnv.testDirPath, testFileName), chunkSizeToRead, 10*util.MiB, os.O_RDONLY|syscall.O_DIRECT)

	require.Error(s.T(), err, "Read should fail after GCS object is deleted")
	operations.ValidateESTALEError(s.T(), err)
	logContent, err := os.ReadFile(testEnv.cfg.LogFile)
	require.NoError(s.T(), err)
	assert.Contains(s.T(), string(logContent), "Sparse file read failed")
	assert.Contains(s.T(), string(logContent), "Falling back to GCS")
}

func (s *chunkCacheTest) TestCacheFileAllocatedSize() {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, 21*util.MiB, s.T())

	tests := []struct {
		name                  string
		offset                int64
		expectedAllocatedSize int64
		isSeq                 bool
		expectedDownloads     []data.ObjectRange
	}{
		{
			name:                  "ReadLastChunk",
			offset:                20 * util.MiB,
			expectedAllocatedSize: 1 * util.MiB,
			isSeq:                 false,
			expectedDownloads: []data.ObjectRange{
				{Start: 20 * util.MiB, End: 21 * util.MiB},
			},
		},
		{
			name:                  "ReadFirstChunk",
			offset:                0,
			expectedAllocatedSize: 11 * util.MiB, // 1MB (from previous) + 10MB (current)
			isSeq:                 true,
			expectedDownloads: []data.ObjectRange{
				{Start: 20 * util.MiB, End: 21 * util.MiB},
				{Start: 0, End: 10 * util.MiB},
			},
		},
	}
	for i, tc := range tests {
		s.T().Run(tc.name, func(t *testing.T) {
			expectedOutcome := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, tc.offset, t)

			validateAllocatedFileSize(t, testFileName, tc.expectedAllocatedSize)
			structuredLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, t)
			require.Len(t, structuredLogs, i+1)
			validate(expectedOutcome, structuredLogs[i], tc.isSeq, true, 1, t)
			jobLogs := read_logs.GetJobLogsSortedByTimestamp(testEnv.cfg.LogFile, t)
			require.NotEmpty(t, jobLogs)
			validateDownloads(t, jobLogs[0].ChunkCacheDownloads, tc.expectedDownloads, testFileName)
		})
	}
}

func TestChunkCacheTest(t *testing.T) {
	ts := &chunkCacheTest{
		ctx:           context.Background(),
		storageClient: testEnv.storageClient,
		baseTestName:  t.Name(),
	}
	// Run tests for mounted directory if the flag is set. This assumes that run flag is properly passed by GKE team as per the config.
	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		suite.Run(t, ts)
		return
	}

	// Run tests for GCE environment otherwise.
	flagsSet := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	for _, flags := range flagsSet {
		ts.flags = flags
		log.Printf("running tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}
