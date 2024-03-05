// Copyright 2024 Google Inc. All Rights Reserved.
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
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/test_setup"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type cacheFileForRangeReadTrueTest struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
}

func (s *cacheFileForRangeReadTrueTest) Setup(t *testing.T) {
	setupForMountedDirectoryTests()
	// Clean up the cache directory path as gcsfuse don't clean up on mounting.
	operations.RemoveDir(cacheDirPath)
	mountGCSFuseAndSetupTestDir(s.flags, s.ctx, s.storageClient, testDirName)
}

func (s *cacheFileForRangeReadTrueTest) Teardown(t *testing.T) {
	setup.UnmountGCSFuseAndDeleteLogFile(rootDir)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *cacheFileForRangeReadTrueTest) TestRangeReadsWithCacheHit(t *testing.T) {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, testDirName, fileSizeForRangeRead, t)

	// Do a random read on file and validate from gcs.
	expectedOutcome1 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offsetForFirstRangeRead, t)
	// Wait for the cache to propagate the updates before proceeding to get cache hit.
	time.Sleep(2 * time.Second)
	// Read file again from zeroOffset 1000 and validate from gcs.
	expectedOutcome2 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offsetForSecondRangeRead, t)

	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), t)
	validate(expectedOutcome1, structuredReadLogs[0], false, false, 1, t)
	validate(expectedOutcome2, structuredReadLogs[1], false, true, 1, t)
	// Validate cached content with gcs.
	validateFileInCacheDirectory(testFileName, fileSizeForRangeRead, s.ctx, s.storageClient, t)
	// Validate cache size within limit.
	validateCacheSizeWithinLimit(cacheCapacityForRangeReadTestInMiB, t)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestCacheFileForRangeReadTrueTest(t *testing.T) {
	ts := &cacheFileForRangeReadTrueTest{ctx: context.Background()}
	// Create storage client before running tests.
	closeStorageClient := createStorageClient(t, &ts.ctx, &ts.storageClient)
	defer closeStorageClient()

	// Run tests for mounted directory if the flag is set.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		test_setup.RunTests(t, ts)
		return
	}

	// Define flag set to run the tests.
	flagSet := [][]string{
		{"--implicit-dirs=true"},
		{"--implicit-dirs=false"},
	}
	appendFlags(&flagSet,
		"--config-file="+createConfigFile(cacheCapacityForRangeReadTestInMiB, true, configFileName))
	appendFlags(&flagSet, "--o=ro", "")

	// Run tests.
	for _, flags := range flagSet {
		ts.flags = flags
		log.Printf("Running tests with flags: %s", ts.flags)
		test_setup.RunTests(t, ts)
	}
}
