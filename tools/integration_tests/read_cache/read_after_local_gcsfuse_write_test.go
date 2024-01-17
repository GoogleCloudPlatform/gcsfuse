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
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/test_setup"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////
type readAfterLocalWrite struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
}

func (s *readAfterLocalWrite) Setup(t *testing.T) {
	mountGCSFuse(s.flags)
	setup.SetMntDir(mountDir)
	testDirPath = setup.SetupTestDirectory(testDirName)
}

func (s *readAfterLocalWrite) Teardown(t *testing.T) {
	// unmount gcsfuse
	setup.SetMntDir(rootDir)
	unmountGCSFuseAndDeleteLogFile()
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *readAfterLocalWrite) TestReadAfterLocalGCSFuseWriteIsCacheMiss(t *testing.T) {
	testFileName := testDirName + "5"
	operations.CreateFileWithSize(fileSize, path.Join(testDirPath, testFileName), t)

	// Read file 1st time.
	expectedOutcome1 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, fileSize, t)
	// Append data in the same file to change generation.
	err := operations.WriteFileInAppendMode(path.Join(testDirPath, testFileName), smallContent)
	if err != nil {
		t.Errorf("Error in appending data in file: %v", err)
	}
	// Read file 2nd time.
	expectedOutcome2 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, fileSize+smallContentSize, t)

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
	ts := &readAfterLocalWrite{}
	closeStorageClient := createStorageClient(t, &ts.ctx, &ts.storageClient)
	defer closeStorageClient()

	// Run tests.
	for _, flags := range flagSet {
		// Run tests without ro flag.
		ts.flags = flags
		test_setup.RunTests(t, ts)
	}
}
