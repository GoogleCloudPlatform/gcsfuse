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
	"sync"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/vipnydav/gcsfuse/v3/internal/cache/util"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const cacheSizeMB int64 = 48

type jobChunkTest struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
	baseTestName  string
	chunkSize     int64
	suite.Suite
}

func (s *jobChunkTest) SetupSuite() {
	setupLogFileAndCacheDir(s.baseTestName)
	mountGCSFuseAndSetupTestDir(s.flags, s.ctx, s.storageClient)
}

func (s *jobChunkTest) SetupTest() {
	//Truncate log file created.
	err := os.Truncate(testEnv.cfg.LogFile, 0)
	require.NoError(s.T(), err)
	// Clean up the cache directory path as gcsfuse don't clean up on mounting.
	operations.RemoveDir(testEnv.cacheDirPath)
	testEnv.testDirPath = client.SetupTestDirectory(s.ctx, s.storageClient, testDirName)
}

func (s *jobChunkTest) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
}

func (s *jobChunkTest) TearDownSuite() {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *jobChunkTest) TestJobChunkSizeForSingleFileReads() {
	var fileSize int64 = 16 * util.MiB
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, fileSize, s.T())

	expectedOutcome := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, fileSize, false, s.T())

	// Parse the log file and validate cache hit or miss from the structured logs.
	structuredJobLogs := read_logs.GetJobLogsSortedByTimestamp(setup.LogFile(), s.T())
	assert.Equal(s.T(), expectedOutcome.BucketName, structuredJobLogs[0].BucketName)
	assert.Equal(s.T(), expectedOutcome.ObjectName, structuredJobLogs[0].ObjectName)

	// We need to check that downloadedOffset is always greater than the previous downloadedOffset
	// and is in multiples of chunkSize.
	for i := 1; i < len(structuredJobLogs[0].JobEntries); i++ {
		offsetDiff := structuredJobLogs[0].JobEntries[i].Offset - structuredJobLogs[0].JobEntries[i-1].Offset
		assert.Greater(s.T(), offsetDiff, int64(0))
		// This is true for all entries except last one.
		// Will be true for last entry only if the fileSize is multiple of chunkSize.
		assert.Equal(s.T(), int64(0), offsetDiff%s.chunkSize)
	}

	// Validate that last downloadedOffset is same as fileSize.
	assert.Equal(s.T(), fileSize, structuredJobLogs[0].JobEntries[len(structuredJobLogs[0].JobEntries)-1].Offset)
}

func (s *jobChunkTest) TestJobChunkSizeForMultipleFileReads() {
	var fileSize int64 = 16 * util.MiB
	var testFileNames [2]string
	var expectedOutcome [2]*Expected
	testFileNames[0] = setupFileInTestDir(s.ctx, s.storageClient, fileSize, s.T())
	testFileNames[1] = setupFileInTestDir(s.ctx, s.storageClient, fileSize, s.T())

	// Read 2 files in parallel.
	var wg sync.WaitGroup
	for i := range 2 {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			expectedOutcome[i] = readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileNames[i], fileSize, false, s.T())
		}()
	}
	wg.Wait()

	// Parse the log file and validate cache hit or miss from the structured logs.
	structuredJobLogs := read_logs.GetJobLogsSortedByTimestamp(setup.LogFile(), s.T())
	require.Equal(s.T(), 2, len(structuredJobLogs))
	// Goroutine execution order isn't guaranteed.
	// If the object name in expected outcome doesn't align with the logs, swap
	// the expected outcome objects and file names at positions 0 and 1.
	if expectedOutcome[0].ObjectName != structuredJobLogs[0].ObjectName {
		expectedOutcome[0], expectedOutcome[1] = expectedOutcome[1], expectedOutcome[0]
		testFileNames[0], testFileNames[1] = testFileNames[1], testFileNames[0]
	}

	for fileIndex := range 2 {
		assert.Equal(s.T(), expectedOutcome[fileIndex].BucketName, structuredJobLogs[fileIndex].BucketName)
		assert.Equal(s.T(), expectedOutcome[fileIndex].ObjectName, structuredJobLogs[fileIndex].ObjectName)

		// We need to check that downloadedOffset is always greater than the previous downloadedOffset
		// and is in multiples of chunkSize.
		entriesLen := len(structuredJobLogs[fileIndex].JobEntries)
		for entryIndex := 1; entryIndex < entriesLen; entryIndex++ {
			offsetDiff := structuredJobLogs[fileIndex].JobEntries[entryIndex].Offset - structuredJobLogs[fileIndex].JobEntries[entryIndex-1].Offset
			assert.Greater(s.T(), offsetDiff, int64(0))
			// This is true for all entries except last one.
			// Will be true for last entry only if the fileSize is multiple of chunkSize.
			assert.Equal(s.T(), int64(0), offsetDiff%s.chunkSize)
		}

		// Validate that last downloadedOffset is same as fileSize.
		assert.Equal(s.T(), fileSize, structuredJobLogs[fileIndex].JobEntries[entriesLen-1].Offset)
	}
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func (s *jobChunkTest) runTests(t *testing.T) {
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

func TestJobChunkTest(t *testing.T) {
	// Tests to validate chunk size when read cache parallel downloads are disabled.
	ts := &jobChunkTest{ctx: context.Background(), storageClient: testEnv.storageClient, chunkSize: 8 * util.MiB, baseTestName: t.Name()}
	ts.runTests(t)
}

func TestJobChunkTestWithParallelDownloads(t *testing.T) {
	// Tests to validate chunk size when read cache parallel downloads are enabled
	// The flag set combination is chosen in such a way that chunk size remains 4.
	ts := &jobChunkTest{ctx: context.Background(), storageClient: testEnv.storageClient, chunkSize: 4 * util.MiB, baseTestName: t.Name()}
	ts.runTests(t)
}
