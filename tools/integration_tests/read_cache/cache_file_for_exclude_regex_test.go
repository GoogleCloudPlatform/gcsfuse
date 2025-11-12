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
	"fmt"
	"log"
	"os"
	"testing"

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

type cacheFileForExcludeRegexTest struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
	baseTestName  string
	suite.Suite
}

func (s *cacheFileForExcludeRegexTest) SetupSuite() {
	setupLogFileAndCacheDir(s.baseTestName)
	mountGCSFuseAndSetupTestDir(s.flags, s.ctx, s.storageClient)
}

func (s *cacheFileForExcludeRegexTest) SetupTest() {
	//Truncate log file created.
	err := os.Truncate(testEnv.cfg.LogFile, 0)
	require.NoError(s.T(), err)
	// Clean up the cache directory path as gcsfuse don't clean up on mounting.
	operations.RemoveDir(testEnv.cacheDirPath)
	testEnv.testDirPath = client.SetupTestDirectory(s.ctx, s.storageClient, testDirName)
}

func (s *cacheFileForExcludeRegexTest) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
}

func (s *cacheFileForExcludeRegexTest) TearDownSuite() {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *cacheFileForExcludeRegexTest) TestReadsForExcludedFile() {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, fileSizeForRangeRead, s.T())

	expectedOutcome1 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, zeroOffset, s.T())
	expectedOutcome2 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offset1000, s.T())

	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(testEnv.cfg.LogFile, s.T())
	validate(expectedOutcome1, structuredReadLogs[0], true, false, 1, s.T())
	validate(expectedOutcome2, structuredReadLogs[1], false, false, 1, s.T())
	validateFileIsNotCached(testFileName, s.T())
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestCacheFileForExcludeRegexTest(t *testing.T) {
	ts := &cacheFileForExcludeRegexTest{
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
	if setup.OnlyDirMounted() != "" {
		flagsSet = append(flagsSet,
			[]string{fmt.Sprintf("--file-cache-exclude-regex=^%s/%s/", setup.TestBucket(), onlyDirMounted), "--file-cache-max-size-mb=50", "--file-cache-cache-file-for-range-read=true", "--file-cache-enable-parallel-downloads=false", "--file-cache-enable-o-direct=false", fmt.Sprintf("--cache-dir=%s/gcsfuse-tmp/TestCacheFileForExcludeRegexTest", setup.TestDir()), fmt.Sprintf("--log-file=%s/gcsfuse-tmp/TestCacheFileForExcludeRegexTest.log", setup.TestDir()), "--log-severity=TRACE"},
			[]string{fmt.Sprintf("--file-cache-exclude-regex=^%s/%s/", setup.TestBucket(), onlyDirMounted), "--file-cache-max-size-mb=50", "--file-cache-cache-file-for-range-read=true", "--file-cache-enable-parallel-downloads=false", "--file-cache-enable-o-direct=false", fmt.Sprintf("--cache-dir=%s/gcsfuse-tmp/TestCacheFileForExcludeRegexTest", setup.TestDir()), fmt.Sprintf("--log-file=%s/gcsfuse-tmp/TestCacheFileForExcludeRegexTest.log", setup.TestDir()), "--log-severity=TRACE", "--client-protocol=grpc"},
		)
	}
	for _, ts.flags = range flagsSet {
		log.Printf("Running tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}
