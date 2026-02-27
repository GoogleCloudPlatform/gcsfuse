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
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type chunkCacheDisabledTest struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
	baseTestName  string
	suite.Suite
}

func (s *chunkCacheDisabledTest) SetupSuite() {
	setupLogFileAndCacheDir(s.baseTestName)
	mountGCSFuseAndSetupTestDir(s.flags, s.ctx, s.storageClient)
}

func (s *chunkCacheDisabledTest) SetupTest() {
	//Truncate log file created.
	err := os.Truncate(testEnv.cfg.LogFile, 0)
	require.NoError(s.T(), err)
	// Clean up the cache directory path as gcsfuse don't clean up on mounting.
	operations.RemoveDir(testEnv.cacheDirPath)
	testEnv.testDirPath = client.SetupTestDirectory(s.ctx, s.storageClient, testDirName)
}

func (s *chunkCacheDisabledTest) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
}

func (s *chunkCacheDisabledTest) TearDownSuite() {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
}

func (s *chunkCacheDisabledTest) TestNormalFileCacheWithChunkCacheDisabled() {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, 10*util.MiB, s.T())

	expectedOutcome := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, 0, s.T())

	// Verify cache miss for normal file cache
	structuredLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	require.Len(s.T(), structuredLogs, 1)
	validate(expectedOutcome, structuredLogs[0], true, false, 1, s.T())

	jobLogs := read_logs.GetJobLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	require.Len(s.T(), jobLogs, 1, "Job logs should have exactly 1 entry")
	assert.Empty(s.T(), jobLogs[0].ChunkCacheDownloads, "Should not have chunk downloads")
	assert.NotEmpty(s.T(), jobLogs[0].JobEntries, "Should have normal file cache downloads")
}

func TestChunkCacheDisabledTest(t *testing.T) {
	ts := &chunkCacheDisabledTest{
		ctx:           context.Background(),
		storageClient: testEnv.storageClient,
		baseTestName:  t.Name(),
	}
	// Run tests for mounted directory if the flag is set. This assumes that run flag is properly passed by GKE team as per the config.
	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		suite.Run(t, ts)
		return
	}

	flagsSet := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	for _, flags := range flagsSet {
		ts.flags = flags
		log.Printf("running tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}
