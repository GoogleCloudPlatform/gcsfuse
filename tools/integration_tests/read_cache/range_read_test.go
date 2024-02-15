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
	"os"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/test_setup"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type rangeReadTest struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
}

func (s *rangeReadTest) Setup(t *testing.T) {
	mountGCSFuseAndSetupTestDir(s.flags, s.ctx, s.storageClient, testDirName)
}

func (s *rangeReadTest) Teardown(t *testing.T) {
	unmountGCSFuseAndDeleteLogFile()
	// Clean up the cache directory path as gcsfuse don't clean up on mounting.
	os.RemoveAll(cacheLocationPath)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *rangeReadTest) TestRangeReadsWithinReadChunkSize(t *testing.T) {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, testDirName, fileSizeForRangeRead, t)

	expectedOutcome1 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, zeroOffset, t)
	expectedOutcome2 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offsetForRangeReadWithin8MB, t)

	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), t)
	validate(expectedOutcome1, structuredReadLogs[0], true, false, 1, t)
	validate(expectedOutcome2, structuredReadLogs[1], false, true, 1, t)
}

func (s *rangeReadTest) TestRangeReadsBeyondReadChunkSizeWithChunkDownloaded(t *testing.T) {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, testDirName, fileSizeForRangeRead, t)

	expectedOutcome1 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, zeroOffset, t)
	time.Sleep(1 * time.Second)
	expectedOutcome2 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offset10MiB, t)

	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), t)
	validate(expectedOutcome1, structuredReadLogs[0], true, false, 1, t)
	validate(expectedOutcome2, structuredReadLogs[1], false, true, 1, t)
	validateCacheSizeWithinLimit(cacheCapacityForRangeReadTestInMiB, t)
}

func (s *rangeReadTest) TestRangeReadsBeyondReadChunkSizeWithoutChunkDownloaded(t *testing.T) {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, testDirName, fileSizeForRangeRead, t)

	expectedOutcome1 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, zeroOffset, t)
	expectedOutcome2 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offsetEndOfFile, t)

	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), t)
	validate(expectedOutcome1, structuredReadLogs[0], true, false, 1, t)
	validate(expectedOutcome2, structuredReadLogs[1], false, false, 1, t)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestRangeReadTest(t *testing.T) {
	// Define flag set to run the tests.
	flagSet := [][]string{
		{"--implicit-dirs=true"},
		{"--implicit-dirs=false"},
	}
	appendFlags(&flagSet,
		"--config-file="+createConfigFile(cacheCapacityForRangeReadTestInMiB, false, configFileName+"1"),
		"--config-file="+createConfigFile(cacheCapacityForRangeReadTestInMiB, true, configFileName+"2"))
	appendFlags(&flagSet, "--o=ro", "")

	// Create storage client before running tests.
	ts := &rangeReadTest{ctx: context.Background()}
	closeStorageClient := createStorageClient(t, &ts.ctx, &ts.storageClient)
	defer closeStorageClient()

	// Run tests.
	for _, flags := range flagSet {
		ts.flags = flags
		log.Printf("Running tests with flags: %s", ts.flags)
		test_setup.RunTests(t, ts)
	}
}
