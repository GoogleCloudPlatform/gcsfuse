// Copyright 2024 Google LLC
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
	"log"
	"path"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type remountTest struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
	suite.Suite
}

func (s *remountTest) SetupTest() {
	operations.RemoveDir(cacheDirPath)
	mountGCSFuseAndSetupTestDir(s.flags, s.ctx, s.storageClient)
}

func (s *remountTest) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
	setup.UnmountGCSFuseAndDeleteLogFile(rootDir)
}

////////////////////////////////////////////////////////////////////////
// Helper functions
////////////////////////////////////////////////////////////////////////

func readFileAndValidateCacheWithGCSForDynamicMount(bucketName string, ctx context.Context, storageClient *storage.Client, fileName string, checkCacheSize bool, t *testing.T) (expectedOutcome *Expected) {
	setup.SetDynamicBucketMounted(bucketName)
	defer setup.SetDynamicBucketMounted("")
	testDirPath = path.Join(rootDir, bucketName, testDirName)
	expectedOutcome = readFileAndValidateCacheWithGCS(ctx, storageClient, fileName, fileSize, checkCacheSize, t)

	return expectedOutcome
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *remountTest) TestCacheIsNotReusedOnRemount() {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, fileSize, s.T())

	// Run read operations on GCSFuse mount.
	expectedOutcome1 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, fileSize, true, s.T())
	expectedOutcome2 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, fileSize, true, s.T())
	structuredReadLogsMount1 := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), s.T())
	// Re-mount GCSFuse.
	remountGCSFuse(s.flags)
	// Run read operations again on GCSFuse mount.
	expectedOutcome3 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, fileSize, false, s.T())
	expectedOutcome4 := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, fileSize, false, s.T())
	structuredReadLogsMount2 := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), s.T())

	validate(expectedOutcome1, structuredReadLogsMount1[0], true, false, chunksRead, s.T())
	validate(expectedOutcome2, structuredReadLogsMount1[1], true, true, chunksRead, s.T())
	validate(expectedOutcome3, structuredReadLogsMount2[0], true, false, chunksRead, s.T())
	validate(expectedOutcome4, structuredReadLogsMount2[1], true, true, chunksRead, s.T())
}

func (s *remountTest) TestCacheIsNotReusedOnDynamicRemount() {
	runTestsOnlyForDynamicMount(s.T())
	testBucket1 := setup.TestBucket()
	testFileName1 := setupFileInTestDir(s.ctx, s.storageClient, fileSize, s.T())

	// 1. First read: This should result in a cache miss, and the file content will be cached.
	expectedOutcome1 := readFileAndValidateCacheWithGCSForDynamicMount(testBucket1, s.ctx, s.storageClient, testFileName1, true, s.T())
	structuredReadLogs1 := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), s.T())
	// Remount GCSFuse. This should clear any in-memory cache.
	remountGCSFuse(s.flags)
	// 2. Second read (after remount): This should also result in a cache miss as the cache should be empty.
	expectedOutcome2 := readFileAndValidateCacheWithGCSForDynamicMount(testBucket1, s.ctx, s.storageClient, testFileName1, false, s.T())
	// 3. Third read (without remount): This should result in a cache hit.
	expectedOutcome3 := readFileAndValidateCacheWithGCSForDynamicMount(testBucket1, s.ctx, s.storageClient, testFileName1, false, s.T())
	structuredReadLogs2 := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), s.T())

	// log1: First read -> cache miss
	validate(expectedOutcome1, structuredReadLogs1[0], true, false, chunksRead, s.T())
	// log2: Second read (after remount) -> cache miss
	validate(expectedOutcome2, structuredReadLogs2[0], true, false, chunksRead, s.T())
	// log3: Third read -> cache hit
	validate(expectedOutcome3, structuredReadLogs2[1], true, true, chunksRead, s.T())
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
	flagsSet := []gcsfuseTestFlags{
		{
			cliFlags:                []string{"--implicit-dirs"},
			cacheSize:               cacheCapacityInMB,
			cacheFileForRangeRead:   false,
			fileName:                configFileName,
			enableParallelDownloads: false,
			enableODirect:           false,
			cacheDirPath:            getDefaultCacheDirPathForTests(),
		},
		{
			cliFlags:                nil,
			cacheSize:               cacheCapacityInMB,
			cacheFileForRangeRead:   false,
			fileName:                configFileNameForParallelDownloadTests,
			enableParallelDownloads: true,
			enableODirect:           false,
			cacheDirPath:            getDefaultCacheDirPathForTests(),
		},
	}
	flagsSet = appendClientProtocolConfigToFlagSet(flagsSet)
	// Create storage client before running tests.
	ts := &remountTest{ctx: context.Background()}
	closeStorageClient := client.CreateStorageClientWithCancel(&ts.ctx, &ts.storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			t.Errorf("closeStorageClient failed: %v", err)
		}
	}()

	// Run tests.
	for _, flags := range flagsSet {
		configFilePath := createConfigFile(&flags)
		ts.flags = []string{"--config-file=" + configFilePath}
		if flags.cliFlags != nil {
			ts.flags = append(ts.flags, flags.cliFlags...)
		}
		log.Printf("Running tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}
