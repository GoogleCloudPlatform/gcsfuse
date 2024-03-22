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
	"fmt"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/test_setup"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type smallCacheTTLTest struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
}

func (s *smallCacheTTLTest) Setup(t *testing.T) {
	setupForMountedDirectoryTests()
	// Clean up the cache directory path as gcsfuse don't clean up on mounting.
	operations.RemoveDir(cacheDirPath)
	mountGCSFuseAndSetupTestDir(s.flags, s.ctx, s.storageClient, testDirName)
}

func (s *smallCacheTTLTest) Teardown(t *testing.T) {
	setup.UnmountGCSFuseAndDeleteLogFile(rootDir)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *smallCacheTTLTest) TestReadAfterUpdateAndCacheExpiryIsCacheMiss(t *testing.T) {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, testDirName, fileSize, t)

	// Read file 1st time.
	expectedOutcome1 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, fileSize, true, t)
	// Modify the file.
	modifyFile(s.ctx, s.storageClient, testFileName, t)
	// Read same file again immediately.
	expectedOutcome2 := readFileAndGetExpectedOutcome(testDirPath, testFileName, true, zeroOffset, t)
	validateFileSizeInCacheDirectory(testFileName, fileSize, t)
	// Validate that stale data is served from cache in this case.
	if strings.Compare(expectedOutcome1.content, expectedOutcome2.content) != 0 {
		t.Errorf("content mismatch. Expected old data to be served again.")
	}
	// Wait for metadata cache expiry and read the file again.
	time.Sleep(metadataCacheTTlInSec * time.Second)
	expectedOutcome3 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, smallContentSize, true, t)

	// Parse the log file and validate cache hit or miss from the structured logs.
	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), t)
	validate(expectedOutcome1, structuredReadLogs[0], true, false, chunksRead, t)
	validate(expectedOutcome2, structuredReadLogs[1], true, true, chunksRead, t)
	validate(expectedOutcome3, structuredReadLogs[2], true, false, chunksReadAfterUpdate, t)
}

func (s *smallCacheTTLTest) TestReadForLowMetaDataCacheTTLIsCacheHit(t *testing.T) {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, testDirName, fileSize, t)

	// Read file 1st time.
	expectedOutcome1 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, fileSize, true, t)
	// Wait for metadata cache expiry and read the file again.
	time.Sleep(metadataCacheTTlInSec * time.Second)
	expectedOutcome2 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, fileSize, true, t)
	// Read same file again immediately.
	expectedOutcome3 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, fileSize, true, t)

	// Parse the log file and validate cache hit or miss from the structured logs.
	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), t)
	validate(expectedOutcome1, structuredReadLogs[0], true, false, chunksRead, t)
	validate(expectedOutcome2, structuredReadLogs[1], true, true, chunksRead, t)
	validate(expectedOutcome3, structuredReadLogs[2], true, true, chunksRead, t)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestSmallCacheTTLTest(t *testing.T) {
	ts := &smallCacheTTLTest{ctx: context.Background()}
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
	appendFlags(&flagSet, fmt.Sprintf("--stat-cache-ttl=%ds", metadataCacheTTlInSec))
	appendFlags(&flagSet, "--o=ro", "")

	// Run tests.
	for _, flags := range flagSet {
		ts.flags = flags
		log.Printf("Running tests with flags: %s", ts.flags)
		test_setup.RunTests(t, ts)
	}
}
