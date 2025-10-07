// Copyright 2024 Google LLC
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
	"fmt"
	"log"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type smallCacheTTLTest struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
	baseTestName  string
	suite.Suite
}

func (s *smallCacheTTLTest) SetupTest() {
	setupForMountedDirectoryTestsWithConfig(s.baseTestName)
	// Clean up the cache directory path as gcsfuse don't clean up on mounting.
	fmt.Println("cacheDirPath is ", cacheDirPath)
	operations.RemoveDir(cacheDirPath)
	mountGCSFuseAndSetupTestDir(s.flags, s.ctx, s.storageClient)
}

func (s *smallCacheTTLTest) TearDownTest() {
	fmt.Println("running teardown")

	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
	setup.UnmountGCSFuseAndDeleteLogFileWithCfg(testEnv.cfg, rootDir)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *smallCacheTTLTest) TestReadAfterUpdateAndCacheExpiryIsCacheMiss() {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, fileSize, s.T())

	// Read file 1st time.
	expectedOutcome1 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, fileSize, true, s.T())
	// Modify the file.
	modifyFile(s.ctx, s.storageClient, testFileName, s.T())
	// Read same file again immediately.
	expectedOutcome2 := readFileAndGetExpectedOutcome(testEnv.testDirPath, testFileName, true, zeroOffset, s.T())
	validateFileSizeInCacheDirectory(testFileName, fileSize, s.T())
	// Validate that stale data is served from cache in this case.
	if strings.Compare(expectedOutcome1.content, expectedOutcome2.content) != 0 {
		s.T().Errorf("content mismatch. Expected old data to be served again.")
	}
	// Wait for metadata cache expiry and read the file again.
	time.Sleep(metadataCacheTTlInSec * time.Second)
	expectedOutcome3 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, smallContentSize, true, s.T())

	// Parse the log file and validate cache hit or miss from the structured logs.
	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), s.T())
	require.Equal(s.T(), 3, len(structuredReadLogs))
	validate(expectedOutcome1, structuredReadLogs[0], true, false, chunksRead, s.T())
	validate(expectedOutcome2, structuredReadLogs[1], true, true, chunksRead, s.T())
	validate(expectedOutcome3, structuredReadLogs[2], true, false, chunksReadAfterUpdate, s.T())
}

func (s *smallCacheTTLTest) TestReadForLowMetaDataCacheTTLIsCacheHit() {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, fileSize, s.T())

	// Read file 1st time.
	expectedOutcome1 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, fileSize, true, s.T())
	// Wait for metadata cache expiry and read the file again.
	time.Sleep(metadataCacheTTlInSec * time.Second)
	expectedOutcome2 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, fileSize, true, s.T())
	// Read same file again immediately.
	expectedOutcome3 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, fileSize, true, s.T())

	// Parse the log file and validate cache hit or miss from the structured logs.
	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), s.T())
	require.Equal(s.T(), 3, len(structuredReadLogs))
	validate(expectedOutcome1, structuredReadLogs[0], true, false, chunksRead, s.T())
	validate(expectedOutcome2, structuredReadLogs[1], true, true, chunksRead, s.T())
	validate(expectedOutcome3, structuredReadLogs[2], true, true, chunksRead, s.T())
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestSmallCacheTTLTest(t *testing.T) {
	ts := &smallCacheTTLTest{ctx: context.Background(), storageClient: testEnv.storageClient, baseTestName: t.Name()}

	// Run tests for mounted directory if the flag is set. This assumes that run flag is properly passed by GKE team as per the config.
	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		suite.Run(t, ts)
		return
	}

	// Run tests for GCE environment otherwise.
	for _, configs := range testEnv.cfg.Configs {
		if configs.Run == t.Name() {
			flagsSet := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType)
			for _, ts.flags = range flagsSet {
				log.Printf("Running tests with flags: %s", ts.flags)
				suite.Run(t, ts)
			}
		}
	}
}
