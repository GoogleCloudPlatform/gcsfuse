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

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/test_setup"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type disabledCacheTTLTest struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
}

func (s *disabledCacheTTLTest) Setup(t *testing.T) {
	setupForMountedDirectoryTests()
	// Clean up the cache directory path as gcsfuse don't clean up on mounting.
	operations.RemoveDir(cacheDirPath)
	mountGCSFuseAndSetupTestDir(s.flags, s.ctx, s.storageClient, testDirName)
}

func (s *disabledCacheTTLTest) Teardown(t *testing.T) {
	setup.UnmountGCSFuseAndDeleteLogFile(rootDir)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *disabledCacheTTLTest) TestReadAfterObjectUpdateIsCacheMiss(t *testing.T) {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, testDirName, fileSize, t)

	// Read file 1st time.
	expectedOutcome1 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, fileSize, true, t)
	// Modify the file.
	modifyFile(s.ctx, s.storageClient, testFileName, t)
	// Read same file again immediately. New content should be served as cache ttl is 0.
	expectedOutcome2 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, smallContentSize, true, t)
	// Read the same file again. The data should be served from cache.
	expectedOutcome3 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, smallContentSize, true, t)

	// Parse the log file and validate cache hit or miss from the structured logs.
	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), t)
	validate(expectedOutcome1, structuredReadLogs[0], true, false, chunksRead, t)
	validate(expectedOutcome2, structuredReadLogs[1], true, false, chunksReadAfterUpdate, t)
	validate(expectedOutcome3, structuredReadLogs[2], true, true, chunksReadAfterUpdate, t)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestDisabledCacheTTLTest(t *testing.T) {
	ts := &disabledCacheTTLTest{ctx: context.Background()}
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
	appendFlags(&flagSet, "--config-file="+createConfigFile(cacheCapacityInMB, false, configFileName))
	appendFlags(&flagSet, "--stat-cache-ttl=0s")
	appendFlags(&flagSet, "--o=ro", "")

	// Run tests.
	for _, flags := range flagSet {
		ts.flags = flags
		log.Printf("Running tests with flags: %s", ts.flags)
		test_setup.RunTests(t, ts)
	}
}
