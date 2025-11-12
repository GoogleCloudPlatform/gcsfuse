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

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////
type localModificationTest struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
	baseTestName  string
	suite.Suite
}

func (s *localModificationTest) SetupSuite() {
	setupLogFileAndCacheDir(s.baseTestName)
	mountGCSFuseAndSetupTestDir(s.flags, s.ctx, s.storageClient)
}

func (s *localModificationTest) SetupTest() {
	//Truncate log file created.
	err := os.Truncate(testEnv.cfg.LogFile, 0)
	require.NoError(s.T(), err)
	// Clean up the cache directory path as gcsfuse don't clean up on mounting.
	operations.RemoveDir(testEnv.cacheDirPath)
	testEnv.testDirPath = client.SetupTestDirectory(s.ctx, s.storageClient, testDirName)
}

func (s *localModificationTest) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
}

func (s *localModificationTest) TearDownSuite() {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *localModificationTest) TestReadAfterLocalGCSFuseWriteIsCacheMiss() {
	testFileName := testDirName + setup.GenerateRandomString(testFileNameSuffixLength)
	operations.CreateFileOfSize(fileSize, path.Join(testEnv.testDirPath, testFileName), s.T())

	// Read file 1st time.
	expectedOutcome1 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, fileSize, true, s.T())
	// Append data in the same file to change object generation.
	smallContent, err := operations.GenerateRandomData(smallContentSize)
	if err != nil {
		s.T().Errorf("TestReadAfterLocalGCSFuseWriteIsCacheMiss: could not generate randomm data: %v", err)
	}
	err = operations.WriteFileInAppendMode(path.Join(testEnv.testDirPath, testFileName), string(smallContent))
	if err != nil {
		s.T().Errorf("Error in appending data in file: %v", err)
	}
	if !setup.IsZonalBucketRun() {
		// Read file 2nd time.
		expectedOutcome2 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, fileSize+smallContentSize, true, s.T())

		// Parse the log file and validate cache hit or miss from the structured logs.
		structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
		validate(expectedOutcome1, structuredReadLogs[0], true, false, chunksRead, s.T())
		validate(expectedOutcome2, structuredReadLogs[1], true, false, chunksRead+1, s.T())
	} else {
		// Read file 2nd time.
		expectedOutcome2 := readFileAndGetExpectedOutcome(testEnv.testDirPath, testFileName, true, zeroOffset, s.T())
		expectedPathOfCachedFile := getCachedFilePath(testFileName)
		fileInfo, err := operations.StatFile(expectedPathOfCachedFile)

		// Validate cache size is within limit and cache file size is same as the original file size.
		// This is because for unfinalized objects, we do not trigger a new download job due to appends,
		// we simply fall back to another reader to serve newer reads (which are not cached).
		validateCacheSizeWithinLimit(cacheCapacityInMB, s.T())
		assert.NoError(s.T(), err)
		assert.Equal(s.T(), int64(fileSize), (*fileInfo).Size())

		// Parse the log file and validate cache hit or miss from the structured logs.
		structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
		validate(expectedOutcome2, structuredReadLogs[1], true, true, chunksRead+1, s.T())
	}
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestLocalModificationTest(t *testing.T) {
	ts := &localModificationTest{ctx: context.Background(), storageClient: testEnv.storageClient, baseTestName: t.Name()}

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
