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
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/log_parser/json_parser/read_logs"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/client"
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
	mountGCSFuse(s.flags)
	setup.SetMntDir(mountDir)
	testDirPath = client.SetupTestDirectory(s.ctx, s.storageClient, testDirName)
}

func (s *cacheFileForRangeReadTrueTest) Teardown(t *testing.T) {
	// unmount gcsfuse
	setup.SetMntDir(rootDir)
	unmountGCSFuseAndDeleteLogFile()
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *cacheFileForRangeReadTrueTest) TestRangeReadsWithCacheHit(t *testing.T) {
	testFileName := testFileName + setup.GenerateRandomString(testFileNameSuffixLength)
	client.SetupFileInTestDirectory(s.ctx, s.storageClient, testDirName, testFileName, fileSizeForRangeRead, t)

	// Do a random read on file.
	expectedOutcome1 := readFileAndGetExpectedOutcome(testDirPath, testFileName, false, chunkSizeForRangeRead, offsetForFirstRangeRead, t)
	// Validate content read via gcsfuse with gcs.
	client.ValidateObjectChunkFromGCS(s.ctx, s.storageClient, testDirName, testFileName, offsetForFirstRangeRead, chunkSizeForRangeRead,
		expectedOutcome1.content, t)
	// Wait for the cache to propagate the updates before proceeding to get cache hit.
	time.Sleep(3 * time.Second)
	// Read file again from offset 1.
	expectedOutcome2 := readFileAndGetExpectedOutcome(testDirPath, testFileName, false, chunkSizeForRangeRead, offsetForSecondRangeRead, t)
	// Validate content read via gcsfuse with gcs.
	client.ValidateObjectChunkFromGCS(s.ctx, s.storageClient, testDirName, testFileName, offsetForSecondRangeRead, chunkSizeForRangeRead,
		expectedOutcome2.content, t)

	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), t)
	validate(expectedOutcome1, structuredReadLogs[0], false, false, 1, t)
	validate(expectedOutcome2, structuredReadLogs[1], false, true, 1, t)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestCacheFileForRangeReadTrueTest(t *testing.T) {
	// Define flag set to run the tests.
	flagSet := [][]string{
		{"--implicit-dirs=true"},
		{"--implicit-dirs=false"},
	}
	appendFlags(&flagSet,
		"--config-file="+createConfigFile(cacheCapacityInMB, true, configFileName+"2"))
	appendFlags(&flagSet, "--o=ro", "")

	// Create storage client before running tests.
	ts := &cacheFileForRangeReadTrueTest{ctx: context.Background()}
	closeStorageClient := createStorageClient(t, &ts.ctx, &ts.storageClient)
	defer closeStorageClient()

	// Run tests.
	for _, flags := range flagSet {
		ts.flags = flags
		t.Logf("Running tests with flags: %s", ts.flags)
		test_setup.RunTests(t, ts)
	}
}
