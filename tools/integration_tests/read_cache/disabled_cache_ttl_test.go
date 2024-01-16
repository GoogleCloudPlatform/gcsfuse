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

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/test_setup"
)

const ()

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type disabledCacheTTLTest struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
}

func (s *disabledCacheTTLTest) Setup(t *testing.T) {
	mountGCSFuse(s.flags)
	setup.SetMntDir(mountDir)
	testDirPath = client.SetupTestDirectory(s.ctx, s.storageClient, testDirName)
	client.SetupFileInTestDirectory(s.ctx, s.storageClient, testDirName, testFileName, fileSize, t)
}

func (s *disabledCacheTTLTest) Teardown(t *testing.T) {
	// unmount gcsfuse
	setup.SetMntDir(rootDir)
	unmountGCSFuseAndDeleteLogFile()
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *disabledCacheTTLTest) TestReadAfterUpdateIsCacheMiss(t *testing.T) {
	// Read file 1st time.
	expectedOutcome1 := readFileAndGetExpectedOutcome(testDirPath, testFileName, t)
	validateFileInCacheDirectory(fileSize, s.ctx, s.storageClient, t)
	client.ValidateObjectContentsFromGCS(s.ctx, s.storageClient, testDirName, testFileName,
		expectedOutcome1.content, t)

	// Modify the file.
	err := client.WriteToObject(s.ctx, s.storageClient, objectName, smallContent, storage.Conditions{})
	if err != nil {
		t.Errorf("Could not append to file: %v", err)
	}

	// Read same file again immediately. New content should be served as cache ttl is 0.
	expectedOutcome2 := readFileAndGetExpectedOutcome(testDirPath, testFileName, t)
	validateFileInCacheDirectory(smallContentSize, s.ctx, s.storageClient, t)
	client.ValidateObjectContentsFromGCS(s.ctx, s.storageClient, testDirName, testFileName,
		expectedOutcome2.content, t)

	// Read the same file again. The data should be served from cache.
	expectedOutcome3 := readFileAndGetExpectedOutcome(testDirPath, testFileName, t)
	validateFileInCacheDirectory(smallContentSize, s.ctx, s.storageClient, t)
	client.ValidateObjectContentsFromGCS(s.ctx, s.storageClient, testDirName, testFileName,
		expectedOutcome3.content, t)

	// Parse the log file and validate cache hit or miss from the structured logs.
	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), t)
	validate(expectedOutcome1, structuredReadLogs[0], true, false, chunksRead, t)
	validate(expectedOutcome2, structuredReadLogs[1], true, false, chunksReadAfterUpdate, t)
	validate(expectedOutcome3, structuredReadLogs[2], true, true, chunksReadAfterUpdate, t)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestDisabledCacheTTLTest(t *testing.T) {
	// Define flag set to run the tests.
	mountConfigFilePath := createConfigFile(9)
	var flagSet = [][]string{
		{"--implicit-dirs=true", "--config-file=" + mountConfigFilePath, "--stat-cache-ttl=0s"},
		{"--implicit-dirs=false", "--config-file=" + mountConfigFilePath, "--stat-cache-ttl=0s"},
	} // Create storage client before running tests.
	ts := &disabledCacheTTLTest{ctx: context.Background()}
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
