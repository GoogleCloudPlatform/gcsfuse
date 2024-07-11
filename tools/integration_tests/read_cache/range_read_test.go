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

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/test_setup"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type rangeReadTest struct {
	flags                      []string
	storageClient              *storage.Client
	ctx                        context.Context
	isParallelDownloadsEnabled bool
}

func (s *rangeReadTest) Setup(t *testing.T) {
	setupForMountedDirectoryTests()
	// Clean up the cache directory path as gcsfuse don't clean up on mounting.
	operations.RemoveDir(cacheDirPath)
	mountGCSFuseAndSetupTestDir(s.flags, s.ctx, s.storageClient, testDirName)
}

func (s *rangeReadTest) Teardown(t *testing.T) {
	setup.UnmountGCSFuseAndDeleteLogFile(rootDir)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *rangeReadTest) TestRangeReadsWithinReadChunkSize(t *testing.T) {
	if s.isParallelDownloadsEnabled {
		// This test verifies that the reads are all cache hit within a downloaded chunk.
		// However, with parallel downloads, we cannot guarantee this behavior, so
		// we skip this test when parallel downloads are enabled.
		t.SkipNow()
	}
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, testDirName, largeFileSize, t)

	expectedOutcome1 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, zeroOffset, t)
	expectedOutcome2 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offsetForRangeReadWithin8MB, t)

	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), t)
	validate(expectedOutcome1, structuredReadLogs[0], true, false, 1, t)
	validate(expectedOutcome2, structuredReadLogs[1], false, true, 1, t)
}

func (s *rangeReadTest) TestRangeReadsBeyondReadChunkSizeWithFileCached(t *testing.T) {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, testDirName, largeFileSize, t)

	expectedOutcome1 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, zeroOffset, t)
	validateFileInCacheDirectory(testFileName, largeFileSize, ctx, s.storageClient, t)
	expectedOutcome2 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offset10MiB, t)

	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), t)
	validate(expectedOutcome1, structuredReadLogs[0], true, false, 1, t)
	validate(expectedOutcome2, structuredReadLogs[1], false, true, 1, t)
	validateCacheSizeWithinLimit(largeFileCacheCapacity, t)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestRangeReadTest(t *testing.T) {
	ts := &rangeReadTest{ctx: context.Background()}
	// Create storage client before running tests.
	closeStorageClient := client.CreateStorageClientWithTimeOut(&ts.ctx, &ts.storageClient, 15*time.Minute)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			t.Errorf("closeStorageClient failed: %v", err)
		}
	}()

	// Run tests for mounted directory if the flag is set.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		test_setup.RunTests(t, ts)
		return
	}

	// Run tests with parallel downloads disabled.
	flagsSet := [][]string{
		{"--implicit-dirs", "--config-file=" + createConfigFile(largeFileCacheCapacity, false, configFileName+"1", false)},
	}
	for _, flags := range flagsSet {
		ts.flags = flags
		log.Printf("Running tests with flags: %s", ts.flags)
		test_setup.RunTests(t, ts)
	}

	// Run tests with parallel downloads enabled.
	flagsSet = [][]string{
		{"--config-file=" + createConfigFile(largeFileCacheCapacity, true, configFileName+"2", false)},
	}
	for _, flags := range flagsSet {
		ts.flags = flags
		ts.isParallelDownloadsEnabled = true
		log.Printf("Running tests with flags: %s", ts.flags)
		test_setup.RunTests(t, ts)
	}
}
