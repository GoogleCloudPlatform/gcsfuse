// Copyright 2025 Google LLC
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
	"path"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type cacheFileForIncludeRegexTest struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func TestCacheFileForIncludeRegex_ForIncludedFile(t *testing.T) {
	s := setupTest(t, setup.TestBucket(), "")
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, fileSizeForRangeRead, t)

	// Read the file and validate that it is cached.
	expectedOutcome1 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, zeroOffset, t)
	expectedOutcome2 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offset1000, t)

	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), t)
	validate(expectedOutcome1, structuredReadLogs[0], true, false, 1, t)
	validate(expectedOutcome2, structuredReadLogs[1], false, true, 1, t)
	validateFileIsCached(testFileName, t)
}

func TestCacheFileForIncludeRegex_ForNonIncludedFile(t *testing.T) {
	s := setupTest(t, "non-matching-regex", "")
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, fileSizeForRangeRead, t)

	// Read the file and validate that it is not cached.
	expectedOutcome1 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, zeroOffset, t)
	expectedOutcome2 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offset1000, t)

	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), t)
	validate(expectedOutcome1, structuredReadLogs[0], true, false, 1, t)
	validate(expectedOutcome2, structuredReadLogs[1], false, false, 1, t)
	validateFileIsNotCached(testFileName, t)
}

func TestCacheFileForIncludeRegex_ForIncludedAndExcludeOverlap(t *testing.T) {
	//Prioirty is given to exclude and then include when both is defined
	s := setupTest(t, setup.TestBucket(), setup.TestBucket())
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, fileSizeForRangeRead, t)
	testFileNameExclude := setupFileInTestDir(s.ctx, s.storageClient, fileSizeForRangeRead, t)

	// Read the file and validate that it is cached.
	expectedOutcome1 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, zeroOffset, t)
	expectedOutcome2 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offset1000, t)

	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), t)
	validate(expectedOutcome1, structuredReadLogs[0], true, false, 1, t)
	//Include will not hit in cache due to overlapping regex matching with exclude
	validate(expectedOutcome2, structuredReadLogs[1], false, false, 1, t)
	validateFileIsNotCached(testFileName, t)

	expectedOutcome1Exclude := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileNameExclude, zeroOffset, t)
	expectedOutcome2Exclude := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileNameExclude, offset1000, t)

	structuredReadLogsExclude := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), t)
	validate(expectedOutcome1Exclude, structuredReadLogsExclude[2], true, false, 1, t)
	validate(expectedOutcome2Exclude, structuredReadLogsExclude[3], false, false, 1, t)
	validateFileIsNotCached(testFileNameExclude, t)

}

func TestCacheFileForIncludeRegex_ForIncludedAndExcludeNoOverlap(t *testing.T) {

	s := setupTest(t, setup.TestBucket(), setup.TestBucket()+"/.*/"+testExcludeFileName+".*")
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, fileSizeForRangeRead, t)
	testFileNameExclude := setupExcludeFileInTestDir(s.ctx, s.storageClient, fileSizeForRangeRead, t)

	// Read the file and validate that it is cached.
	expectedOutcome1 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, zeroOffset, t)
	expectedOutcome2 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offset1000, t)

	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), t)
	validate(expectedOutcome1, structuredReadLogs[0], true, false, 1, t)
	validate(expectedOutcome2, structuredReadLogs[1], false, true, 1, t)
	validateFileIsCached(testFileName, t)

	expectedOutcome1Exclude := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileNameExclude, zeroOffset, t)
	expectedOutcome2Exclude := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileNameExclude, offset1000, t)

	structuredReadLogsExclude := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), t)
	validate(expectedOutcome1Exclude, structuredReadLogsExclude[2], true, false, 1, t)
	validate(expectedOutcome2Exclude, structuredReadLogsExclude[3], false, false, 1, t)
	validateFileIsNotCached(testFileNameExclude, t)

}

func setupTest(t *testing.T, includeRegex string, excludeRegex string) *cacheFileForIncludeRegexTest {

	ramCacheDir := path.Join("/dev/shm", cacheDirName)

	flagTest := gcsfuseTestFlags{
		cliFlags:              []string{},
		cacheSize:             cacheCapacityForRangeReadTestInMiB,
		cacheFileForRangeRead: true,
		fileName:              configFileName,
		cacheDirPath:          ramCacheDir,
	}
	if includeRegex != "" {
		if strings.ContainsRune(includeRegex, '/') {
			flagTest.cliFlags = append(flagTest.cliFlags, fmt.Sprintf("--file-cache-include-regex=^%s", includeRegex))
		} else {
			flagTest.cliFlags = append(flagTest.cliFlags, fmt.Sprintf("--file-cache-include-regex=^%s/", includeRegex))
		}
	}
	if excludeRegex != "" {
		if strings.ContainsRune(excludeRegex, '/') {
			flagTest.cliFlags = append(flagTest.cliFlags, fmt.Sprintf("--file-cache-exclude-regex=^%s", excludeRegex))
		} else {
			flagTest.cliFlags = append(flagTest.cliFlags, fmt.Sprintf("--file-cache-exclude-regex=^%s/", excludeRegex))
		}
	}

	ts := &cacheFileForIncludeRegexTest{
		ctx:   context.Background(),
		flags: flagTest.cliFlags,
	}

	// Create storage client before running tests.
	closeStorageClient := client.CreateStorageClientWithCancel(&ts.ctx, &ts.storageClient)

	//Add flag to mount
	flagTest = appendClientProtocolConfigToFlagSet([]gcsfuseTestFlags{flagTest})[0]
	configFilePath := createConfigFile(&flagTest)
	ts.flags = []string{"--config-file=" + configFilePath}
	if flagTest.cliFlags != nil {
		ts.flags = append(ts.flags, flagTest.cliFlags...)
	}

	t.Cleanup(func() {
		t.Logf("Tearing down %s", t.Name())
		err := closeStorageClient()
		if err != nil {
			t.Errorf("closeStorageClient failed: %v", err)
		}

		setup.SaveGCSFuseLogFileInCaseOfFailure(t)
		setup.UnmountGCSFuseAndDeleteLogFile(rootDir)
	})

	//Setup for mounted directory tests
	setupForMountedDirectoryTests()
	//Clean up the cache directory path as gcsfuse don't clean up on mounting.
	operations.RemoveDir(cacheDirPath)
	//Run with cache directory pointing to RAM based dir
	mountGCSFuseAndSetupTestDir(ts.flags, ts.ctx, ts.storageClient)

	log.Printf("Running %s with flags: %+v", t.Name(), ts.flags)

	return ts
}
