// Copyright 2025 Google LLC
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
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type cacheFileForIncludeRegexTest struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
	baseTestName  string
	suite.Suite
}

func (s *cacheFileForIncludeRegexTest) SetupSuite() {
	setupLogFileAndCacheDir(s.baseTestName)
	mountGCSFuseAndSetupTestDir(s.flags, s.ctx, s.storageClient)
}

func (s *cacheFileForIncludeRegexTest) SetupTest() {
	//Truncate log file created.
	err := os.Truncate(testEnv.cfg.LogFile, 0)
	require.NoError(s.T(), err)
	// Clean up the cache directory path as gcsfuse don't clean up on mounting.
	operations.RemoveDir(testEnv.cacheDirPath)
	testEnv.testDirPath = client.SetupTestDirectory(s.ctx, s.storageClient, testDirName)
}

func (s *cacheFileForIncludeRegexTest) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
}

func (s *cacheFileForIncludeRegexTest) TearDownSuite() {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *cacheFileForIncludeRegexTest) TestCacheFileForIncludeRegexForIncludedFile() {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, fileSize, s.T())

	// Read the file and validate that it is cached.
	expectedOutcome1 := readFileAndGetExpectedOutcome(testEnv.testDirPath, testFileName, true, 0, s.T())
	expectedOutcome2 := readFileAndGetExpectedOutcome(testEnv.testDirPath, testFileName, true, 0, s.T())

	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	validate(expectedOutcome1, structuredReadLogs[0], true, false, chunksRead, s.T())
	validate(expectedOutcome2, structuredReadLogs[1], true, true, chunksRead, s.T())
	validateFileIsCached(testFileName, s.T())
}

func (s *cacheFileForIncludeRegexTest) TestCacheFileForIncludeRegexForNonIncludedFile() {
	testFileName := "non-matching-regex" + setup.GenerateRandomString(testFileNameSuffixLength)
	client.SetupFileInTestDirectory(s.ctx, s.storageClient, testDirName, testFileName, fileSize, s.T())

	// Read the file and validate that it is not cached.
	expectedOutcome1 := readFileAndGetExpectedOutcome(testEnv.testDirPath, testFileName, true, 0, s.T())
	expectedOutcome2 := readFileAndGetExpectedOutcome(testEnv.testDirPath, testFileName, true, 0, s.T())

	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	validate(expectedOutcome1, structuredReadLogs[0], true, false, chunksRead, s.T())
	validate(expectedOutcome2, structuredReadLogs[1], true, false, chunksRead, s.T())
	validateFileIsNotCached(testFileName, s.T())
}

func (s *cacheFileForIncludeRegexTest) TestCacheFileForIncludeRegexForIncludedAndExcludeNoOverlap() {
	includedFileName := setupFileInTestDir(s.ctx, s.storageClient, fileSize, s.T())
	excludedFileName := "non-matching-regex" + setup.GenerateRandomString(testFileNameSuffixLength)
	client.SetupFileInTestDirectory(s.ctx, s.storageClient, testDirName, excludedFileName, fileSize, s.T())

	// Read the included file and validate that it is cached.
	expectedOutcome1 := readFileAndGetExpectedOutcome(testEnv.testDirPath, includedFileName, true, 0, s.T())
	expectedOutcome2 := readFileAndGetExpectedOutcome(testEnv.testDirPath, includedFileName, true, 0, s.T())
	// Read the excluded file and validate that it is not cached.
	expectedOutcome3 := readFileAndGetExpectedOutcome(testEnv.testDirPath, excludedFileName, true, 0, s.T())
	expectedOutcome4 := readFileAndGetExpectedOutcome(testEnv.testDirPath, excludedFileName, true, 0, s.T())

	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	validate(expectedOutcome1, structuredReadLogs[0], true, false, chunksRead, s.T())
	validate(expectedOutcome2, structuredReadLogs[1], true, true, chunksRead, s.T())
	validateFileIsCached(includedFileName, s.T())
	validate(expectedOutcome3, structuredReadLogs[2], true, false, chunksRead, s.T())
	validate(expectedOutcome4, structuredReadLogs[3], true, false, chunksRead, s.T())
	validateFileIsNotCached(excludedFileName, s.T())
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestCacheFileForIncludeRegexTest(t *testing.T) {
	ts := &cacheFileForIncludeRegexTest{
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
	for _, ts.flags = range flagsSet {
		log.Printf("Running tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}
