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
	"sync"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/test_setup"
	"github.com/jacobsa/ogletest"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type cacheFileForRangeReadFalseTest struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
}

func (s *cacheFileForRangeReadFalseTest) Setup(t *testing.T) {
	setupForMountedDirectoryTests()
	// Clean up the cache directory path as gcsfuse don't clean up on mounting.
	operations.RemoveDir(cacheDirPath)
	mountGCSFuseAndSetupTestDir(s.flags, s.ctx, s.storageClient, testDirName)
}

func (s *cacheFileForRangeReadFalseTest) Teardown(t *testing.T) {
	setup.UnmountGCSFuseAndDeleteLogFile(rootDir)
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func readFileAsync(t *testing.T, wg *sync.WaitGroup, testFileName string, expectedOutcome **Expected) {
	go func() {
		defer wg.Done()
		*expectedOutcome = readFileAndGetExpectedOutcome(testDirPath, testFileName, true, zeroOffset, t)
	}()
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *cacheFileForRangeReadFalseTest) TestRangeReadsWithCacheMiss(t *testing.T) {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, testDirName, fileSizeForRangeRead, t)

	// Do a random read on file and validate from gcs.
	expectedOutcome1 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offsetForFirstRangeRead, t)
	// Read file again from offset 1000 and validate from gcs.
	expectedOutcome2 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offsetForSecondRangeRead, t)

	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), t)
	validate(expectedOutcome1, structuredReadLogs[0], false, false, 1, t)
	validate(expectedOutcome2, structuredReadLogs[1], false, false, 1, t)
	validateFileIsNotCached(testFileName, t)
}

func (s *cacheFileForRangeReadFalseTest) TestConcurrentReads_ReadIsTreatedNonSequentialAfterFileIsRemovedFromCache(t *testing.T) {
	var testFileNames [2]string
	var expectedOutcome [2]*Expected
	testFileNames[0] = setupFileInTestDir(s.ctx, s.storageClient, testDirName, fileSizeForRangeRead, t)
	testFileNames[1] = setupFileInTestDir(s.ctx, s.storageClient, testDirName, fileSizeForRangeRead, t)

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		readFileAsync(t, &wg, testFileNames[i], &expectedOutcome[i])
	}
	wg.Wait()

	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), t)
	// Goroutine execution order isn't guaranteed.
	// If the object name in expected outcome doesn't align with the logs, swap
	// the expected outcome objects and file names at positions 0 and 1.
	if expectedOutcome[0].ObjectName != structuredReadLogs[0].ObjectName {
		expectedOutcome[0], expectedOutcome[1] = expectedOutcome[1], expectedOutcome[0]
		testFileNames[0], testFileNames[1] = testFileNames[1], testFileNames[0]
	}
	validate(expectedOutcome[0], structuredReadLogs[0], true, false, randomReadChunkCount, t)
	validate(expectedOutcome[1], structuredReadLogs[1], true, false, randomReadChunkCount, t)
	// Validate last chunk was considered non-sequential and cache hit false for first read.
	ogletest.ExpectEq(false, structuredReadLogs[0].Chunks[randomReadChunkCount-1].IsSequential)
	ogletest.ExpectEq(false, structuredReadLogs[0].Chunks[randomReadChunkCount-1].CacheHit)
	// Validate last chunk was considered sequential and cache hit true for second read.
	ogletest.ExpectEq(true, structuredReadLogs[1].Chunks[randomReadChunkCount-1].IsSequential)
	ogletest.ExpectEq(true, structuredReadLogs[1].Chunks[randomReadChunkCount-1].CacheHit)

	validateFileIsNotCached(testFileNames[0], t)
	validateFileInCacheDirectory(testFileNames[1], fileSizeForRangeRead, s.ctx, s.storageClient, t)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestCacheFileForRangeReadFalseTest(t *testing.T) {
	ts := &cacheFileForRangeReadFalseTest{ctx: context.Background()}
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
		"--config-file="+createConfigFile(cacheCapacityForRangeReadTestInMiB, false, configFileName))
	appendFlags(&flagSet, "--o=ro", "")

	// Run tests.
	for _, flags := range flagSet {
		ts.flags = flags
		log.Printf("Running tests with flags: %s", ts.flags)
		test_setup.RunTests(t, ts)
	}
}
