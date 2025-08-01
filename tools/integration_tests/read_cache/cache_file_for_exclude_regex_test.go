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
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_setup"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type cacheFileForExcludeRegexTest struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
}

func (s *cacheFileForExcludeRegexTest) Setup(t *testing.T) {
	setupForMountedDirectoryTests()
	// Clean up the cache directory path as gcsfuse don't clean up on mounting.
	operations.RemoveDir(cacheDirPath)
	mountGCSFuseAndSetupTestDir(s.flags, s.ctx, s.storageClient)
}

func (s *cacheFileForExcludeRegexTest) Teardown(t *testing.T) {
	setup.SaveGCSFuseLogFileInCaseOfFailure(t)
	setup.UnmountGCSFuseAndDeleteLogFile(rootDir)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *cacheFileForExcludeRegexTest) TestReadsForExcludedFile(t *testing.T) {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, fileSizeForRangeRead, t)

	expectedOutcome1 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, zeroOffset, t)
	expectedOutcome2 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offset1000, t)

	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), t)
	validate(expectedOutcome1, structuredReadLogs[0], true, false, 1, t)
	validate(expectedOutcome2, structuredReadLogs[1], false, false, 1, t)
	validateFileIsNotCached(testFileName, t)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestCacheFileForExcludeRegexTest(t *testing.T) {
	ts := &cacheFileForExcludeRegexTest{ctx: context.Background()}
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
		test_setup.RunTests(t, ts)
		return
	}

	// Run with cache directory pointing to RAM based dir
	ramCacheDir := path.Join("/dev/shm", cacheDirName)

	tests := []struct {
		flags       gcsfuseTestFlags
		onlyDirTest bool
	}{
		{
			flags: gcsfuseTestFlags{
				cliFlags:                []string{"--implicit-dirs", "--file-cache-experimental-exclude-regex=."},
				cacheSize:               cacheCapacityForRangeReadTestInMiB,
				cacheFileForRangeRead:   false,
				fileName:                configFileName,
				enableParallelDownloads: false,
				enableODirect:           false,
				cacheDirPath:            getDefaultCacheDirPathForTests(),
			},
		},
		{
			flags: gcsfuseTestFlags{
				cliFlags:                []string{"--file-cache-experimental-exclude-regex=."},
				cacheSize:               cacheCapacityForRangeReadTestInMiB,
				cacheFileForRangeRead:   false,
				fileName:                configFileName,
				enableParallelDownloads: false,
				enableODirect:           false,
				cacheDirPath:            ramCacheDir,
			},
		},
		{
			flags: gcsfuseTestFlags{
				cliFlags:                []string{"--file-cache-experimental-exclude-regex=."},
				cacheSize:               cacheCapacityForRangeReadTestInMiB,
				cacheFileForRangeRead:   true,
				fileName:                configFileName,
				enableParallelDownloads: false,
				enableODirect:           false,
				cacheDirPath:            ramCacheDir,
			},
		},
		{
			flags: gcsfuseTestFlags{
				// Exclude regex is set to bucket name as the prefix of the string, so should exclude all objects.
				cliFlags:                []string{fmt.Sprintf("--file-cache-experimental-exclude-regex=^%s/", setup.TestBucket())},
				cacheSize:               cacheCapacityForRangeReadTestInMiB,
				cacheFileForRangeRead:   true,
				fileName:                configFileName,
				enableParallelDownloads: false,
				enableODirect:           false,
				cacheDirPath:            ramCacheDir,
			},
		},
		{
			flags: gcsfuseTestFlags{
				// Exclude regex is set to the only-dir value which is not present in local paths, but should be present in all cloud paths.
				cliFlags:                []string{fmt.Sprintf("--file-cache-experimental-exclude-regex=^%s/%s/", setup.TestBucket(), onlyDirMounted)},
				cacheSize:               cacheCapacityForRangeReadTestInMiB,
				cacheFileForRangeRead:   true,
				fileName:                configFileName,
				enableParallelDownloads: false,
				enableODirect:           false,
				cacheDirPath:            ramCacheDir,
			},
			onlyDirTest: true,
		},
	}
	for _, test := range tests {
		test.flags = appendClientProtocolConfigToFlagSet([]gcsfuseTestFlags{test.flags})[0]
		if test.onlyDirTest && setup.OnlyDirMounted() == "" {
			continue
		}
		configFilePath := createConfigFile(&test.flags)
		ts.flags = []string{"--config-file=" + configFilePath}
		if test.flags.cliFlags != nil {
			ts.flags = append(ts.flags, test.flags.cliFlags...)
		}
		log.Printf("Running tests with flags: %s", ts.flags)
		test_setup.RunTests(t, ts)
	}
}
