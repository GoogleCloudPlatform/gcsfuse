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
	"log"
	"path"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type cacheFileForRangeReadTrueTest struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
	suite.Suite
}

func (s *cacheFileForRangeReadTrueTest) SetupTest() {
	setupForMountedDirectoryTests()
	// Clean up the cache directory path as gcsfuse don't clean up on mounting.
	operations.RemoveDir(cacheDirPath)
	mountGCSFuseAndSetupTestDir(s.flags, s.ctx, s.storageClient)
}

func (s *cacheFileForRangeReadTrueTest) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
	setup.UnmountGCSFuseAndDeleteLogFile(rootDir)
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

	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), s.T())
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

func TestCacheFileForRangeReadTrueTest(t *testing.T) {
	ts := &cacheFileForRangeReadTrueTest{ctx: context.Background()}
	// Create storage client before running tests.
	closeStorageClient := client.CreateStorageClientWithCancel(&ts.ctx, &ts.storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			t.Errorf("closeStorageClient failed: %v", err)
		}
	}()

	// Run tests for mounted directory if the flag is set.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		suite.Run(t, ts)
		return
	}

	// Run with cache directory pointing to RAM based dir
	ramCacheDir := path.Join("/dev/shm", cacheDirName)

	// Define flag set to run the tests.
	flagsSet := []gcsfuseTestFlags{
		{
			cliFlags:                []string{"--implicit-dirs"},
			cacheSize:               cacheCapacityForRangeReadTestInMiB,
			cacheFileForRangeRead:   true,
			fileName:                configFileName,
			enableParallelDownloads: false,
			enableODirect:           false,
			cacheDirPath:            getDefaultCacheDirPathForTests(),
		},
		{
			cliFlags:                nil,
			cacheSize:               cacheCapacityForRangeReadTestInMiB,
			cacheFileForRangeRead:   true,
			fileName:                configFileNameForParallelDownloadTests,
			enableParallelDownloads: true,
			enableODirect:           false,
			cacheDirPath:            getDefaultCacheDirPathForTests(),
		},
		{
			cliFlags:                nil,
			cacheSize:               cacheCapacityForRangeReadTestInMiB,
			cacheFileForRangeRead:   true,
			fileName:                configFileNameForParallelDownloadTests,
			enableParallelDownloads: true,
			enableODirect:           false,
			cacheDirPath:            ramCacheDir,
		},
		{
			cliFlags:                nil,
			cacheSize:               cacheCapacityForRangeReadTestInMiB,
			cacheFileForRangeRead:   true,
			fileName:                configFileNameForParallelDownloadTests,
			enableParallelDownloads: true,
			enableODirect:           true,
			cacheDirPath:            getDefaultCacheDirPathForTests(),
		},
		{
			cliFlags:                nil,
			cacheSize:               cacheCapacityForRangeReadTestInMiB,
			cacheFileForRangeRead:   true,
			fileName:                configFileNameForParallelDownloadTests,
			enableParallelDownloads: true,
			enableODirect:           true,
			cacheDirPath:            ramCacheDir,
		},
		{
			cliFlags:                nil,
			cacheSize:               cacheCapacityForRangeReadTestInMiB,
			cacheFileForRangeRead:   true,
			fileName:                configFileName,
			enableParallelDownloads: false,
			enableODirect:           false,
			cacheDirPath:            ramCacheDir,
		},
	}
	flagsSet = appendClientProtocolConfigToFlagSet(flagsSet)
	// Run tests.
	for _, flags := range flagsSet {
		configFilePath := createConfigFile(&flags)
		ts.flags = []string{"--config-file=" + configFilePath}
		if flags.cliFlags != nil {
			ts.flags = append(ts.flags, flags.cliFlags...)
		}
		log.Printf("Running tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}
