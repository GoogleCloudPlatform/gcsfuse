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
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
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
	// Clean up the cache directory path as gcsfuse don't clean up on mounting.
	operations.RemoveDir(testEnv.cacheDirPath)
	testEnv.testDirPath = client.SetupTestDirectory(s.ctx, s.storageClient, testDirName)
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

	// Do a random read on file and validate from gcs.
	expectedOutcome1 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offset5000, s.T())
	// Wait for the cache to propagate the updates before proceeding to get cache hit.
	time.Sleep(4 * time.Second)
	// Read file again from zeroOffset 1000 and validate from gcs.
	expectedOutcome2 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offset1000, s.T())

	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	validate(expectedOutcome1, structuredReadLogs[0], false, false, 1, s.T())
	validate(expectedOutcome2, structuredReadLogs[1], false, true, 1, s.T())
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
