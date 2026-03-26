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
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type rangeReadTest struct {
	flags                      []string
	storageClient              *storage.Client
	ctx                        context.Context
	isParallelDownloadsEnabled bool
	baseTestName               string
	suite.Suite
}

func (s *rangeReadTest) SetupSuite() {
	setupLogFileAndCacheDir(s.baseTestName)
	mountGCSFuseAndSetupTestDir(s.flags, s.ctx, s.storageClient)
}

func (s *rangeReadTest) SetupTest() {
	//Truncate log file created.
	err := os.Truncate(testEnv.cfg.LogFile, 0)
	require.NoError(s.T(), err)
	// Clean up the cache directory path as gcsfuse don't clean up on mounting.
	operations.RemoveDir(testEnv.cacheDirPath)
	testEnv.testDirPath = client.SetupUniqueTestDirectory(s.ctx, s.storageClient, testDirPrefix)
}

func (s *rangeReadTest) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
}

func (s *rangeReadTest) TearDownSuite() {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *rangeReadTest) TestRangeReadsWithinReadChunkSize() {
	if s.isParallelDownloadsEnabled {
		// This test verifies that the reads are all cache hit within a downloaded chunk.
		// However, with parallel downloads, we cannot guarantee this behavior, so
		// we skip this test when parallel downloads are enabled.
		s.T().SkipNow()
	}
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, largeFileSize, s.T())

	expectedOutcome1 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, zeroOffset, s.T())
	expectedOutcome2 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offsetForRangeReadWithin8MB, s.T())

	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	validate(expectedOutcome1, structuredReadLogs[0], true, false, 1, s.T())
	validate(expectedOutcome2, structuredReadLogs[1], false, true, 1, s.T())
}

// TestRangeReadsBeyondReadChunkSizeWithFileCached verifies that read operations beyond the first 8MiB chunk
// result in a cache hit if the file was previously cached by a background download job.
func (s *rangeReadTest) TestRangeReadsBeyondReadChunkSizeWithFileCached() {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, largeFileSize, s.T())

	// Read first chunk (0-128KB) to trigger the background file cache download job.
	expectedOutcome1 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, zeroOffset, s.T())

	// Wait until the background job downloads both the first 8MiB chunk and the second 8MiB-15MiB chunk.
	// This ensures the read at 10MiB is always a cache hit, making the test deterministic.
	s.T().Logf("Waiting for file cache Job with data reaching %d bytes", largeFileSize)
	JobLog := operations.RetryUntil(s.ctx, s.T(), retryFrequency, retryDuration, func() ([]*read_logs.Job, error) {
		logs := read_logs.GetJobLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
		if len(logs) == 1 {
			for _, entry := range logs[0].JobEntries {
				if entry.Offset >= largeFileSize {
					s.T().Logf("Found file cache Job with sufficient data (offset %d): %v", entry.Offset, logs[0])
					return logs, nil
				}
			}
		}
		return nil, fmt.Errorf("expected 1 Job with an entry >= %d bytes, found %d jobs", largeFileSize, len(logs))
	})
	require.Equal(s.T(), expectedOutcome1.ObjectName, JobLog[0].ObjectName)

	// Read the second chunk at offset 10MiB. This should be a cache hit since the background job
	// was verified to have downloaded the second chunk (reaching up to 15MiB).
	expectedOutcome2 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offset10MiB, s.T())

	// Validate results for both reads, verifying the second one is a cache hit.
	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	require.Len(s.T(), structuredReadLogs, 2, "Should have exactly 2 read records in logs")
	validate(expectedOutcome1, structuredReadLogs[0], true, false, 1, s.T())
	validate(expectedOutcome2, structuredReadLogs[1], false, true, 1, s.T())

	validateFileInCacheDirectory(testFileName, largeFileSize, testEnv.ctx, s.storageClient, s.T())
	validateCacheSizeWithinLimit(largeFileCacheCapacity, s.T())
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func runTests(t *testing.T, ts *rangeReadTest) {
	// Run tests for mounted directory if the flag is set. This assumes that run flag is properly passed by GKE team as per the config.
	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		suite.Run(t, ts)
		return
	}

	// Run tests for GCE environment otherwise.
	flagsSet := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	for _, ts.flags = range flagsSet {
		log.Printf("Running tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}

func TestRangeReadTest(t *testing.T) {
	ts := &rangeReadTest{ctx: context.Background(), storageClient: testEnv.storageClient, baseTestName: t.Name(), isParallelDownloadsEnabled: false}
	runTests(t, ts)
}

func TestRangeReadWithParallelDownloadsTest(t *testing.T) {
	ts := &rangeReadTest{ctx: context.Background(), storageClient: testEnv.storageClient, baseTestName: t.Name(), isParallelDownloadsEnabled: true}
	runTests(t, ts)
}
