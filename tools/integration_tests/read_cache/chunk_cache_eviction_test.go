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
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type chunkCacheEvictionTest struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
	baseTestName  string
	suite.Suite
}

func (s *chunkCacheEvictionTest) SetupSuite() {
	setupLogFileAndCacheDir(s.baseTestName)
	mountGCSFuseAndSetupTestDir(s.flags, s.ctx, s.storageClient)
}

func (s *chunkCacheEvictionTest) SetupTest() {
	//Truncate log file created.
	err := os.Truncate(testEnv.cfg.LogFile, 0)
	require.NoError(s.T(), err)
	// Clean up the cache directory path as gcsfuse don't clean up on mounting.
	operations.RemoveDir(testEnv.cacheDirPath)
	testEnv.testDirPath = client.SetupTestDirectory(s.ctx, s.storageClient, testDirName)
}

func (s *chunkCacheEvictionTest) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
}

func (s *chunkCacheEvictionTest) TearDownSuite() {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
}

func (s *chunkCacheEvictionTest) TestEviction() {
	testFileName1 := setupFileInTestDir(s.ctx, s.storageClient, 20*util.MiB, s.T())
	testFileName2 := setupFileInTestDir(s.ctx, s.storageClient, 20*util.MiB, s.T())

	expectedOutcome1 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName1, 20*util.MiB, false, s.T())
	expectedOutcome2 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName2, 20*util.MiB, false, s.T())
	expectedOutcome3 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName1, 0, s.T())

	structuredLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	require.Len(s.T(), structuredLogs, 3)
	validate(expectedOutcome1, structuredLogs[0], true, true, int(20*util.MiB/chunkSizeToRead), s.T())
	validate(expectedOutcome2, structuredLogs[1], true, true, int(20*util.MiB/chunkSizeToRead), s.T())
	validate(expectedOutcome3, structuredLogs[2], true, true, 1, s.T())
	jobLogs := read_logs.GetJobLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	require.NotEmpty(s.T(), jobLogs)
	var allDownloads []read_logs.ChunkDownloadLogEntry
	for _, jobLog := range jobLogs {
		allDownloads = append(allDownloads, jobLog.ChunkCacheDownloads...)
	}
	// Expected download sequence:
	// 1. File 1 is read (20MB). Cache grows to 20MB (exceeding 15MB limit, but eviction is deferred).
	// 2. File 2 is read. Insertion of File 2 triggers eviction of File 1 (LRU) to enforce limit.
	// 3. File 2 is downloaded (20MB). Cache grows to 20MB.
	// 4. File 1 is read again. Insertion of File 1 triggers eviction of File 2. File 1 [0, 10) is re-downloaded.
	expectedDownloads := []data.ObjectRange{
		{Start: 0, End: 10 * util.MiB},             // File 1 chunk 1
		{Start: 10 * util.MiB, End: 20 * util.MiB}, // File 1 chunk 2
		{Start: 0, End: 10 * util.MiB},             // File 2 chunk 1
		{Start: 10 * util.MiB, End: 20 * util.MiB}, // File 2 chunk 2
		{Start: 0, End: 10 * util.MiB},             // File 1 chunk 1 (again)
	}
	// Skip content validation as downloads span multiple files.
	validateDownloads(s.T(), allDownloads, expectedDownloads, "")
}

func TestChunkCacheEviction(t *testing.T) {
	ts := &chunkCacheEvictionTest{
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
