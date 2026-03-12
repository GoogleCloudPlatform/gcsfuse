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
	"testing"

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

type cacheFileForRangeReadTrueTest struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
	baseTestName  string
	isCacheOnRAM  bool
	suite.Suite
}

func (s *cacheFileForRangeReadTrueTest) SetupSuite() {
	setupLogFileAndCacheDir(s.baseTestName)
	if s.isCacheOnRAM {
		testEnv.cacheDirPath = "/dev/shm/" + s.baseTestName
	}
	mountGCSFuseAndSetupTestDir(s.flags, s.ctx, s.storageClient)
}

func (s *cacheFileForRangeReadTrueTest) SetupTest() {
	//Truncate log file created.
	err := os.Truncate(testEnv.cfg.LogFile, 0)
	require.NoError(s.T(), err)
	s.T().Logf("GCSFuse Log File: %s", testEnv.cfg.LogFile)
	// Clean up the cache directory path as gcsfuse don't clean up on mounting.
	operations.RemoveDir(testEnv.cacheDirPath)
	testEnv.testDirPath = client.SetupUniqueTestDirectory(s.ctx, s.storageClient, testDirPrefix)
}

func (s *cacheFileForRangeReadTrueTest) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
}

func (s *cacheFileForRangeReadTrueTest) TearDownSuite() {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *cacheFileForRangeReadTrueTest) TestRangeReadsWithCacheHit() {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, fileSizeForRangeRead, s.T())

	// Do a first random read on file and validate from gcs.
	firstReadOutcome := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offset5000, s.T())
	// Validate a single read log has cache hit 'false' recorded.
	structuredReadFalseCacheHit := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	assert.Len(s.T(), structuredReadFalseCacheHit, 1, "Should have exactly 1 false cache hit read record in logs")
	validate(firstReadOutcome, structuredReadFalseCacheHit[0], false /* isSeq */, false /* cacheHit */, 1 /* chunkCount */, s.T())

	// RetryUntil we have exactly 1 Download Job logs (downloaded till <offset>)
	s.T().Logf("Waiting for file cache Job completion log in GCSFuse Logs")
	JobLog := operations.RetryUntil(s.ctx, s.T(), retryFrequency, retryDuration, func() ([]*read_logs.Job, bool) {
		logs := read_logs.GetJobLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
		if len(logs) == 1 {
			return logs, true
		}
		return nil, false
	})
	assert.Equal(s.T(), structuredReadFalseCacheHit[0].ObjectName, JobLog[0].ObjectName)
	// Read file second time from Offset 1000 and validate from gcs.
	secondReadOutcome := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offset1000, s.T())

	// Validate two read Logs and the second log must have cache hit 'true' recorded.
	structuredReadLogsCacheHitTrueOnSecondLog := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	assert.Len(s.T(), structuredReadLogsCacheHitTrueOnSecondLog, 2, "Should have exactly 2 read records in logs")
	validate(secondReadOutcome, structuredReadLogsCacheHitTrueOnSecondLog[1], false /* isSeq */, true /* cacheHit */, 1 /* chunkCount */, s.T())
	// Validate cached content with gcs.
	validateFileInCacheDirectory(testFileName, fileSizeForRangeRead, s.ctx, s.storageClient, s.T())
	// Validate cache size within limit.
	validateCacheSizeWithinLimit(cacheCapacityForRangeReadTestInMiB, s.T())
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func (s *cacheFileForRangeReadTrueTest) runTests(t *testing.T) {
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

func TestCacheFileForRangeReadTrueTest(t *testing.T) {
	ts := &cacheFileForRangeReadTrueTest{ctx: context.Background(), storageClient: testEnv.storageClient, baseTestName: t.Name()}
	ts.runTests(t)
}

func TestCacheFileForRangeReadTrueWithRamCache(t *testing.T) {
	ts := &cacheFileForRangeReadTrueTest{ctx: context.Background(), storageClient: testEnv.storageClient, baseTestName: t.Name(), isCacheOnRAM: true}
	ts.runTests(t)
}
