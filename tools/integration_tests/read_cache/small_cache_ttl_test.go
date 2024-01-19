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
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/test_setup"
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
	mountGCSFuse(s.flags)
	setup.SetMntDir(mountDir)
	testDirPath = client.SetupTestDirectory(s.ctx, s.storageClient, testDirName)
}

func (s *smallCacheTTLTest) Teardown(t *testing.T) {
	// unmount gcsfuse
	setup.SetMntDir(rootDir)
	unmountGCSFuseAndDeleteLogFile()
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *smallCacheTTLTest) TestReadAfterUpdateAndCacheExpiryIsCacheMiss(t *testing.T) {
	testFileName := testFileName + setup.GenerateRandomString(testFileNameSuffixLength)
	client.SetupFileInTestDirectory(s.ctx, s.storageClient, testDirName, testFileName, fileSize, t)

	// Read file 1st time.
	expectedOutcome1 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, fileSize, t)
	// Modify the file.
	modifyFile(s.ctx, s.storageClient, testFileName, t)
	// Read same file again immediately.
	expectedOutcome2 := readFileAndGetExpectedOutcome(testDirPath, testFileName, t)
	validateFileSizeInCacheDirectory(testFileName, fileSize, t)
	// Validate that stale data is served from cache in this case.
	if strings.Compare(expectedOutcome1.content, expectedOutcome2.content) != 0 {
		t.Errorf("content mismatch. Expected old data to be served again.")
	}
	// Wait for metadata cache expiry and read the file again.
	time.Sleep(metadataCacheTTlInSec * time.Second)
	expectedOutcome3 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, smallContentSize, t)

	// Parse the log file and validate cache hit or miss from the structured logs.
	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), t)
	validate(expectedOutcome1, structuredReadLogs[0], true, false, chunksRead, t)
	validate(expectedOutcome2, structuredReadLogs[1], true, true, chunksRead, t)
	validate(expectedOutcome3, structuredReadLogs[2], true, false, chunksReadAfterUpdate, t)
}

func (s *smallCacheTTLTest) TestReadForLowMetaDataCacheTTLIsCacheHit(t *testing.T) {
	testFileName := testFileName + setup.GenerateRandomString(testFileNameSuffixLength)
	client.SetupFileInTestDirectory(s.ctx, s.storageClient, testDirName, testFileName, fileSize, t)

	// Read file 1st time.
	expectedOutcome1 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, fileSize, t)
	// Wait for metadata cache expiry and read the file again.
	time.Sleep(metadataCacheTTlInSec * time.Second)
	expectedOutcome2 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, fileSize, t)
	// Read same file again immediately.
	expectedOutcome3 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, fileSize, t)

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
	// Define flag set to run the tests.
	mountConfigFilePath := createConfigFile(9)
	var flagSet = [][]string{
		{"--implicit-dirs=true", "--config-file=" + mountConfigFilePath, fmt.Sprintf("--stat-cache-ttl=%ds", metadataCacheTTlInSec)},
		{"--implicit-dirs=false", "--config-file=" + mountConfigFilePath, fmt.Sprintf("--stat-cache-ttl=%ds", metadataCacheTTlInSec)},
	} // Create storage client before running tests.
	ts := &smallCacheTTLTest{ctx: context.Background()}
	closeStorageClient := createStorageClient(t, &ts.ctx, &ts.storageClient)
	defer closeStorageClient()

	// Run tests.
	for _, flags := range flagSet {
		// Run tests without ro flag.
		ts.flags = flags
		test_setup.RunTests(t, ts)
		// Run tests with ro flag.
		ts.flags = append(flags, "--o=ro")
		test_setup.RunTests(t, ts)
	}
}
