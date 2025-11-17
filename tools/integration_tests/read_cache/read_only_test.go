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

type readOnlyTest struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
	baseTestName  string
	suite.Suite
}

func (s *readOnlyTest) SetupSuite() {
	setupLogFileAndCacheDir(s.baseTestName)
	mountGCSFuseAndSetupTestDir(s.flags, s.ctx, s.storageClient)
}

func (s *readOnlyTest) SetupTest() {
	//Truncate log file created.
	err := os.Truncate(testEnv.cfg.LogFile, 0)
	require.NoError(s.T(), err)
	// Clean up the cache directory path as gcsfuse don't clean up on mounting.
	operations.RemoveDir(testEnv.cacheDirPath)
	testEnv.testDirPath = client.SetupTestDirectory(s.ctx, s.storageClient, testDirName)
}

func (s *readOnlyTest) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
}

func (s *readOnlyTest) TearDownSuite() {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
}

////////////////////////////////////////////////////////////////////////
// Helper functions
////////////////////////////////////////////////////////////////////////

func readMultipleFiles(numFiles int, ctx context.Context, storageClient *storage.Client, fileNames []string, t *testing.T) (expectedOutcome []*Expected) {
	for i := range numFiles {
		expectedOutcome = append(expectedOutcome, readFileAndValidateCacheWithGCS(ctx, storageClient, fileNames[i], fileSize, true, t))
	}
	return expectedOutcome
}

func validateCacheOfMultipleObjectsUsingStructuredLogs(startIndex int, numFiles int, expectedOutcome []*Expected, structuredReadLogs []*read_logs.StructuredReadLogEntry, cacheHit bool, t *testing.T) {
	endIndex := startIndex + numFiles

	for i := startIndex; i < endIndex; i++ {
		validate(expectedOutcome[i], structuredReadLogs[i], true, cacheHit, chunksRead, t)
	}
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *readOnlyTest) TestSecondSequentialReadIsCacheHit() {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, fileSize, s.T())

	// Read file 1st time.
	expectedOutcome1 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, fileSize, true, s.T())
	// Read file 2nd time.
	expectedOutcome2 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, fileSize, true, s.T())

	// Parse the log file and validate cache hit or miss from the structured logs.
	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	validate(expectedOutcome1, structuredReadLogs[0], true, false, chunksRead, s.T())
	validate(expectedOutcome2, structuredReadLogs[1], true, true, chunksRead, s.T())
}

func (s *readOnlyTest) TestReadFileSequentiallyLargerThanCacheCapacity() {
	// Set up a file in test directory of size more than cache capacity.
	fileName := setup.GenerateRandomString(7)
	client.SetupFileInTestDirectory(s.ctx, s.storageClient, testDirName, fileName, largeFileSize, s.T())

	// Read file 1st time.
	expectedOutcome1 := readFileAndValidateFileIsNotCached(s.ctx, s.storageClient, fileName, true, zeroOffset, s.T())
	// Read file 2nd time.
	expectedOutcome2 := readFileAndValidateFileIsNotCached(s.ctx, s.storageClient, fileName, true, zeroOffset, s.T())

	// Parse the log file and validate cache hit or miss from the structured logs.
	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	validate(expectedOutcome1, structuredReadLogs[0], true, false, largeFileChunksRead, s.T())
	validate(expectedOutcome2, structuredReadLogs[1], true, false, largeFileChunksRead, s.T())
}

func (s *readOnlyTest) TestReadFileRandomlyLargerThanCacheCapacity() {
	// Set up a file in test directory of size more than cache capacity.
	fileName := setup.GenerateRandomString(7)
	client.SetupFileInTestDirectory(s.ctx, s.storageClient, testDirName, fileName, largeFileSize, s.T())

	// Do a random read on file.
	expectedOutcome1 := readFileAndValidateFileIsNotCached(s.ctx, s.storageClient, fileName, false, randomReadOffset, s.T())
	// Read file sequentially again.
	expectedOutcome2 := readFileAndValidateFileIsNotCached(s.ctx, s.storageClient, fileName, true, zeroOffset, s.T())

	// Parse the log file and validate cache hit or miss from the structured logs.
	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	validate(expectedOutcome1, structuredReadLogs[0], false, false, 1, s.T())
	validate(expectedOutcome2, structuredReadLogs[1], true, false, largeFileChunksRead, s.T())
}

func (s *readOnlyTest) TestReadMultipleFilesMoreThanCacheLimit() {
	fileNames := client.CreateNFilesInDir(s.ctx, s.storageClient, NumberOfFilesMoreThanCacheLimit, testFileName, fileSize, testDirName, s.T())

	expectedOutcome := readMultipleFiles(NumberOfFilesMoreThanCacheLimit, s.ctx, s.storageClient, fileNames, s.T())
	expectedOutcome = append(expectedOutcome, readMultipleFiles(NumberOfFilesMoreThanCacheLimit, s.ctx, s.storageClient, fileNames, s.T())...)

	// Parse the log file and validate cache hit or miss from the structured logs.
	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	validateCacheOfMultipleObjectsUsingStructuredLogs(0, NumberOfFilesMoreThanCacheLimit, expectedOutcome, structuredReadLogs, false, s.T())
	validateCacheOfMultipleObjectsUsingStructuredLogs(NumberOfFilesMoreThanCacheLimit, NumberOfFilesMoreThanCacheLimit, expectedOutcome, structuredReadLogs, false, s.T())
}

func (s *readOnlyTest) TestReadMultipleFilesWithinCacheLimit() {
	fileNames := client.CreateNFilesInDir(s.ctx, s.storageClient, NumberOfFilesWithinCacheLimit, testFileName, fileSize, testDirName, s.T())

	expectedOutcome := readMultipleFiles(NumberOfFilesWithinCacheLimit, s.ctx, s.storageClient, fileNames, s.T())
	expectedOutcome = append(expectedOutcome, readMultipleFiles(NumberOfFilesWithinCacheLimit, s.ctx, s.storageClient, fileNames, s.T())...)

	// Parse the log file and validate cache hit or miss from the structured logs.
	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	validateCacheOfMultipleObjectsUsingStructuredLogs(0, NumberOfFilesWithinCacheLimit, expectedOutcome, structuredReadLogs, false, s.T())
	validateCacheOfMultipleObjectsUsingStructuredLogs(NumberOfFilesWithinCacheLimit, NumberOfFilesWithinCacheLimit, expectedOutcome, structuredReadLogs, true, s.T())
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestReadOnlyTest(t *testing.T) {
	ts := &readOnlyTest{ctx: context.Background(), storageClient: testEnv.storageClient, baseTestName: t.Name()}

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
