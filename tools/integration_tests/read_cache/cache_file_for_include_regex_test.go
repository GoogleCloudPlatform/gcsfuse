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

	//test_setup "github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_setup"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type cacheFileForIncludeRegexTest struct {
	flags         []string
	storageClient *storage.Client
	cacheEnable   bool
	ctx           context.Context
	suite.Suite
}

func (s *cacheFileForIncludeRegexTest) SetupTest() {
	setupForMountedDirectoryTests()
	// Clean up the cache directory path as gcsfuse don't clean up on mounting.
	operations.RemoveDir(cacheDirPath)
	mountGCSFuseAndSetupTestDir(s.flags, s.ctx, s.storageClient)
}

func (s *cacheFileForIncludeRegexTest) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
	setup.UnmountGCSFuseAndDeleteLogFile(rootDir)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *cacheFileForIncludeRegexTest) TestReadsForIncludedFile() {
	if !s.cacheEnable {
		return
	}

	testFileName := setupFileInTestDir(s.ctx, s.storageClient, fileSizeForRangeRead, s.T())

	// Read the file and validate that it is cached.
	expectedOutcome1 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, zeroOffset, s.T())
	expectedOutcome2 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offset1000, s.T())

	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), s.T())
	validate(expectedOutcome1, structuredReadLogs[0], true, false, 1, s.T())
	validate(expectedOutcome2, structuredReadLogs[1], false, true, 1, s.T())
	validateFileIsCached(testFileName, s.T())
}

func (s *cacheFileForIncludeRegexTest) TestReadsForNonIncludedFile() {
	if s.cacheEnable {
		return
	}
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, fileSizeForRangeRead, s.T())

	// Read the file and validate that it is not cached.
	expectedOutcome1 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, zeroOffset, s.T())
	expectedOutcome2 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offset1000, s.T())

	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), s.T())
	validate(expectedOutcome1, structuredReadLogs[0], true, false, 1, s.T())
	validate(expectedOutcome2, structuredReadLogs[1], false, false, 1, s.T())
	validateFileIsNotCached(testFileName, s.T())
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestCacheFileForIncludeRegexTest(t *testing.T) {
	ts := &cacheFileForIncludeRegexTest{ctx: context.Background()}
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
		ts.cacheEnable = true
		suite.Run(t, ts)
		return
	}

	// Run with cache directory pointing to RAM based dir
	ramCacheDir := path.Join("/dev/shm", cacheDirName)

	tests := []struct {
		name        string
		flags       gcsfuseTestFlags
		cacheEnable bool
	}{
		{
			name: "Test included file is cached",
			flags: gcsfuseTestFlags{
				cliFlags:              []string{fmt.Sprintf("--file-cache-include-regex=^%s/", setup.TestBucket())},
				cacheSize:             cacheCapacityForRangeReadTestInMiB,
				cacheFileForRangeRead: true,
				fileName:              configFileName,
				cacheDirPath:          ramCacheDir,
			},
			cacheEnable: true,
		},
		{
			name: "Test non-included file is not cached",
			flags: gcsfuseTestFlags{
				cliFlags:              []string{"--file-cache-include-regex=non-matching-regex"},
				cacheSize:             cacheCapacityForRangeReadTestInMiB,
				cacheFileForRangeRead: true,
				fileName:              configFileName,
				cacheDirPath:          ramCacheDir,
			},
			cacheEnable: false,
		},
	}
	for _, test := range tests {
		if setup.OnlyDirMounted() != "" {
			continue
		}
		ts.cacheEnable = test.cacheEnable
		t.Run(test.name, func(t *testing.T) {
			test.flags = appendClientProtocolConfigToFlagSet([]gcsfuseTestFlags{test.flags})[0]
			configFilePath := createConfigFile(&test.flags)
			ts.flags = []string{"--config-file=" + configFilePath}
			if test.flags.cliFlags != nil {
				ts.flags = append(ts.flags, test.flags.cliFlags...)
			}
			log.Printf("Running tests with flags: %s", ts.flags)
			suite.Run(t, ts)
		})

	}

}
