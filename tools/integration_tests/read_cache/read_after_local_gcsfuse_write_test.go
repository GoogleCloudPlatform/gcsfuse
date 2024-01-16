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
	"path"
	"strings"
	"testing"

	"cloud.google.com/go/storage"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/test_setup"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////
type readAfterLocalGCSFuseWrite struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
}

func (s *readAfterLocalGCSFuseWrite) Setup(t *testing.T) {
	mountGCSFuse(s.flags)

	setup.SetMntDir(mountDir)

	testDirPath = setup.SetupTestDirectory(testDirName)
	randomData, err := operations.GenerateRandomData(fileSize)
	randomDataString := strings.Trim(string(randomData), "\x00")
	if err != nil {
		t.Errorf("operations.GenerateRandomData: %v", err)
	}
	operations.CreateFileWithContent(path.Join(testDirPath, testFileName), setup.FilePermission_0600, randomDataString, t)
}

func (s *readAfterLocalGCSFuseWrite) Teardown(t *testing.T) {
	// unmount gcsfuse
	setup.SetMntDir(rootDir)
	unmountGCSFuseAndDeleteLogFile()
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *readAfterLocalGCSFuseWrite) TestReadAfterLocalGCSFuseWriteIsCacheMiss(t *testing.T) {
	// Read file 1st time.
	expectedOutcome1 := readFileAndGetExpectedOutcome(testDirPath, testFileName, t)
	validateFileInCacheDirectory(fileSize, s.ctx, s.storageClient, t)

	// Validate that the content read by read operation matches content on GCS.
	client.ValidateObjectContentsFromGCS(s.ctx, s.storageClient, testDirName, testFileName,
		expectedOutcome1.content, t)

	// Append data in the same file to change generation.
	err := operations.WriteFileInAppendMode(path.Join(testDirPath, testFileName), smallContent)
	if err != nil {
		t.Errorf("Error in appending data in file: %v", err)
	}

	// Read file 2nd time.
	expectedOutcome2 := readFileAndGetExpectedOutcome(testDirPath, testFileName, t)
	validateFileInCacheDirectory(fileSize+smallContentSize, s.ctx, s.storageClient, t)

	// Validate that the content read by read operation matches content on GCS.
	client.ValidateObjectContentsFromGCS(s.ctx, s.storageClient, testDirName, testFileName,
		expectedOutcome2.content, t)

	// Parse the log file and validate cache hit or miss from the structured logs.
	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), t)
	validate(expectedOutcome1, structuredReadLogs[0], true, false, chunksRead, t)
	validate(expectedOutcome2, structuredReadLogs[1], true, false, chunksRead+1, t)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestReadAfterGCSFuseLocalWrite(t *testing.T) {
	// Define flag set to run the tests.
	mountConfigFilePath := createConfigFile(9)
	flagSet := [][]string{
		{"--implicit-dirs=true", "--config-file=" + mountConfigFilePath},
		{"--implicit-dirs=false", "--config-file=" + mountConfigFilePath},
	}

	// Create storage client before running tests.
	ts := &readAfterLocalGCSFuseWrite{ctx: context.Background()}
	closeStorageClient := createStorageClient(t, &ts.ctx, &ts.storageClient)
	defer closeStorageClient()

	// Run tests.
	for _, flags := range flagSet {
		// Run tests without ro flag.
		ts.flags = flags
		test_setup.RunTests(t, ts)
	}
}
