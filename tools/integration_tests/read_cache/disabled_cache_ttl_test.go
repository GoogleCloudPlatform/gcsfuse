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

type disabledCacheTTLTest struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
	baseTestName  string
	suite.Suite
}

func (s *disabledCacheTTLTest) SetupSuite() {
	setupLogFileAndCacheDir(s.baseTestName)
	mountGCSFuseAndSetupTestDir(s.flags, s.ctx, s.storageClient)
}

func (s *disabledCacheTTLTest) SetupTest() {
	//Truncate log file created.
	err := os.Truncate(testEnv.cfg.LogFile, 0)
	require.NoError(s.T(), err)
	// Clean up the cache directory path as gcsfuse don't clean up on mounting.
	operations.RemoveDir(testEnv.cacheDirPath)
	testEnv.testDirPath = client.SetupTestDirectory(s.ctx, s.storageClient, testDirName)
}

func (s *disabledCacheTTLTest) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
}

func (s *disabledCacheTTLTest) TearDownSuite() {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *disabledCacheTTLTest) TestReadAfterObjectUpdateIsCacheMiss() {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, fileSize, s.T())

	// Read file 1st time.
	expectedOutcome1 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, fileSize, true, s.T())
	// Modify the file.
	modifyFile(s.ctx, s.storageClient, testFileName, s.T())
	// Read same file again immediately. New content should be served as cache ttl is 0.
	expectedOutcome2 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, smallContentSize, true, s.T())
	// Read the same file again. The data should be served from cache.
	expectedOutcome3 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, smallContentSize, true, s.T())

	// Parse the log file and validate cache hit or miss from the structured logs.
	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	validate(expectedOutcome1, structuredReadLogs[0], true, false, chunksRead, s.T())
	validate(expectedOutcome2, structuredReadLogs[1], true, false, chunksReadAfterUpdate, s.T())
	validate(expectedOutcome3, structuredReadLogs[2], true, true, chunksReadAfterUpdate, s.T())
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestDisabledCacheTTLTest(t *testing.T) {
	ts := &disabledCacheTTLTest{ctx: context.Background(), storageClient: testEnv.storageClient, baseTestName: t.Name()}

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
