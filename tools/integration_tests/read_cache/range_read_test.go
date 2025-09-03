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
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/stretchr/testify/suite"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type rangeReadTest struct {
	flags                      []string
	storageClient              *storage.Client
	ctx                        context.Context
	isParallelDownloadsEnabled bool
	suite.Suite
}

func (s *rangeReadTest) SetupTest() {
	setupForMountedDirectoryTests()
	// Clean up the cache directory path as gcsfuse don't clean up on mounting.
	operations.RemoveDir(cacheDirPath)
	mountGCSFuseAndSetupTestDir(s.flags, s.ctx, s.storageClient)
}

func (s *rangeReadTest) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
	setup.UnmountGCSFuseAndDeleteLogFile(rootDir)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *rangeReadTest) TestRangeReadsWithinReadChunkSize() {
	if s.isParallelDownloadsEnabled {
		// This test verifies that the reads are all cache hit within a downloaded chunk.
		// However, with parallel downloads, we cannot guarantee this behavior, so
		// we skip this test when parallel downloads are enabled.
		s.T().SkipNow()
	}
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, largeFileSize, s.T())

	expectedOutcome1 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, zeroOffset, s.T())
	expectedOutcome2 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offsetForRangeReadWithin8MB, s.T())

	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), s.T())
	validate(expectedOutcome1, structuredReadLogs[0], true, false, 1, s.T())
	validate(expectedOutcome2, structuredReadLogs[1], false, true, 1, s.T())
}

func (s *rangeReadTest) TestRangeReadsBeyondReadChunkSizeWithFileCached() {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, largeFileSize, s.T())

	expectedOutcome1 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, zeroOffset, s.T())
	validateFileInCacheDirectory(testFileName, largeFileSize, ctx, s.storageClient, s.T())
	expectedOutcome2 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offset10MiB, s.T())

	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), s.T())
	validate(expectedOutcome1, structuredReadLogs[0], true, false, 1, s.T())
	validate(expectedOutcome2, structuredReadLogs[1], false, true, 1, s.T())
	validateCacheSizeWithinLimit(largeFileCacheCapacity, s.T())
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestRangeReadTest(t *testing.T) {
	ts := &rangeReadTest{ctx: context.Background()}
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

	// Run tests with parallel downloads disabled.
	flagsSet := []gcsfuseTestFlags{
		{
			cliFlags:                []string{"--implicit-dirs"},
			cacheSize:               largeFileCacheCapacity,
			cacheFileForRangeRead:   false,
			fileName:                configFileName + "1",
			enableParallelDownloads: false,
			enableODirect:           false,
			cacheDirPath:            getDefaultCacheDirPathForTests(),
		},
	}
	flagsSet = appendClientProtocolConfigToFlagSet(flagsSet)
	for _, flags := range flagsSet {
		configFilePath := createConfigFile(&flags)
		ts.flags = []string{"--config-file=" + configFilePath}
		if flags.cliFlags != nil {
			ts.flags = append(ts.flags, flags.cliFlags...)
		}
		log.Printf("Running tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}

	// Run tests with parallel downloads enabled.
	flagsSet = []gcsfuseTestFlags{
		{
			cliFlags:                nil,
			cacheSize:               largeFileCacheCapacity,
			cacheFileForRangeRead:   true,
			fileName:                configFileName + "2",
			enableParallelDownloads: false,
			enableODirect:           false,
			cacheDirPath:            getDefaultCacheDirPathForTests(),
		},
	}
	flagsSet = appendClientProtocolConfigToFlagSet(flagsSet)
	for _, flags := range flagsSet {
		configFilePath := createConfigFile(&flags)
		ts.flags = []string{"--config-file=" + configFilePath}
		if flags.cliFlags != nil {
			ts.flags = append(ts.flags, flags.cliFlags...)
		}
		ts.isParallelDownloadsEnabled = true
		log.Printf("Running tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}
