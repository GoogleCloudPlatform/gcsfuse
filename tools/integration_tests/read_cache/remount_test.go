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
	"cloud.google.com/go/compute/metadata"
	"context"
	"log"
	"path"
	"strings"
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

func (s *remountTest) TestCacheClearDynamicRemount(t *testing.T) {
	if !strings.Contains(setup.MntDir(),setup.TestBucket()){
		t.Log("This test will run only for dynamic mounting...")
		t.SkipNow()
	}

  // Created Dynamic mounting bucket.
	project_id, err := metadata.ProjectID()
	if err != nil {
		log.Printf("Error in fetching project id: %v", err)
	}
	var testBucketForDynamicMounting = "gcsfuse-dynamic-mounting-test-" + setup.GenerateRandomString(5)

	// Create bucket with name gcsfuse-dynamic-mounting-test-xxxxx
	setup.RunScriptForTestData("../util/mounting/dynamic_mounting/testdata/create_bucket.sh", testBucketForDynamicMounting, project_id)

	testFileName1 := testFileName + setup.GenerateRandomString(testFileNameSuffixLength)
	// Set up a file in test directory of size more than cache capacity.
	client.SetupFileInTestDirectory(s.ctx, s.storageClient, testDirName,
		testFileName1, fileSize, t)

	// Read file 1st time.
	expectedOutcome1 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName1, fileSize, t)

	// Changed mounted directory for dynamic mounting.
	mountDir = path.Join(setup.MntDir(),testBucketForDynamicMounting)
	setup.SetMntDir(mountDir)

	testFileName2 := testFileName + setup.GenerateRandomString(testFileNameSuffixLength)
	// Set up a file in test directory of size more than cache capacity.
	client.SetupFileInTestDirectory(s.ctx, s.storageClient, testDirName,
		testFileName2, fileSize, t)

	// Read file 1st time.
	expectedOutcome2 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName2, fileSize, t)

	// Parse the log file and validate cache hit or miss from the structured logs.
	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), t)
	validate(expectedOutcome1, structuredReadLogs[0], true, false, chunksRead, t)
	validate(expectedOutcome2, structuredReadLogs[1], true, false, chunksRead, t)

	// Deleting bucket after testing.
	defer setup.RunScriptForTestData("../util/mounting/dynamic_mounting/testdata/delete_bucket.sh", testBucketForDynamicMounting)
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
	mountConfigFilePath := createConfigFile(cacheCapacityInMB)
	flagSet := [][]string{
		{"--implicit-dirs=true", "--config-file=" + mountConfigFilePath},
		{"--implicit-dirs=false", "--config-file=" + mountConfigFilePath},
	}

	// Create storage client before running tests.
	ts := &remountTest{ctx: context.Background()}
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
