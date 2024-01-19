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

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type remountTest struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
}

func (s *remountTest) Setup(t *testing.T) {
	mountGCSFuse(s.flags)
	setup.SetMntDir(mountDir)
	testDirPath = client.SetupTestDirectory(s.ctx, s.storageClient, testDirName)
}

func (s *remountTest) Teardown(t *testing.T) {
	// unmount gcsfuse
	setup.SetMntDir(rootDir)
	unmountGCSFuseAndDeleteLogFile()
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *remountTest) TestCacheClearsOnRemount(t *testing.T) {
	testFileName := testFileName + setup.GenerateRandomString(testFileNameSuffixLength)
	client.SetupFileInTestDirectory(s.ctx, s.storageClient, testDirName, testFileName, fileSize, t)

	// Run read operations on GCSFuse mount.
	expectedOutcome1 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, fileSize, t)
	expectedOutcome2 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, fileSize, t)
	structuredReadLogsMount1 := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), t)
	// Re-mount GCSFuse and validate cache deleted.
	remountGCSFuseAndValidateCacheDeleted(s.flags, t)
	// Run read operations again on GCSFuse mount.
	expectedOutcome3 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, fileSize, t)
	expectedOutcome4 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, fileSize, t)
	structuredReadLogsMount2 := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), t)

	validate(expectedOutcome1, structuredReadLogsMount1[0], true, false, chunksRead, t)
	validate(expectedOutcome2, structuredReadLogsMount1[1], true, true, chunksRead, t)
	validate(expectedOutcome3, structuredReadLogsMount2[0], true, false, chunksRead, t)
	validate(expectedOutcome4, structuredReadLogsMount2[1], true, true, chunksRead, t)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestRemountTest(t *testing.T) {
	if setup.MountedDirectory() != "" {
		t.Log("Not running remount tests for GKE environment...")
		t.SkipNow()
	}
	// Define flag set to run the tests.
	flagSet := [][]string{
		{"--implicit-dirs=true"},
		{"--implicit-dirs=false"},
	}
	appendFlags(&flagSet, "--config-file="+createConfigFile(cacheCapacityInMB, false, configFileName))
	appendFlags(&flagSet, "--o=ro", "")

	// Create storage client before running tests.
	ts := &remountTest{ctx: context.Background()}
	closeStorageClient := createStorageClient(t, &ts.ctx, &ts.storageClient)
	defer closeStorageClient()

	// Run tests.
	for _, flags := range flagSet {
		ts.flags = flags
		t.Logf("Running tests with flags: %s", ts.flags)
		test_setup.RunTests(t, ts)
	}
}
