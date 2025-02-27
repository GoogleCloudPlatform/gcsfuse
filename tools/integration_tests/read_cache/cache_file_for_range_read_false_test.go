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
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/test_setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type cacheFileForRangeReadFalseTest struct {
	flags                      []string
	storageClient              *storage.Client
	ctx                        context.Context
	isParallelDownloadsEnabled bool
}

func (s *cacheFileForRangeReadFalseTest) Setup(t *testing.T) {
	setupForMountedDirectoryTests()
	// Clean up the cache directory path as gcsfuse don't clean up on mounting.
	operations.RemoveDir(cacheDirPath)
	mountGCSFuseAndSetupTestDir(s.flags, s.ctx, s.storageClient, testDirName)
}

func (s *cacheFileForRangeReadFalseTest) Teardown(t *testing.T) {
	if t.Failed() {
		setup.SaveLogFileToKOKOROArtifact("gcsfuse-failed-integration-test-logs-" + strings.Replace(t.Name(), "/", "-", -1))
	}
	setup.UnmountGCSFuseAndDeleteLogFile(rootDir)
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func readFileBetweenOffset(t *testing.T, file *os.File, startOffset, endOffSet int64) *Expected {
	t.Helper()
	expected := &Expected{
		StartTimeStampSeconds: time.Now().Unix(),
		BucketName:            setup.TestBucket(),
		ObjectName:            path.Join(testDirName, path.Base(file.Name())),
	}
	if setup.DynamicBucketMounted() != "" {
		expected.BucketName = setup.DynamicBucketMounted()
	}

	expected.content = operations.ReadFileBetweenOffset(t, file, startOffset, endOffSet, chunkSizeToRead)
	expected.EndTimeStampSeconds = time.Now().Unix()
	return expected
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *cacheFileForRangeReadFalseTest) TestRangeReadsWithCacheMiss(t *testing.T) {
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, testDirName, fileSizeForRangeRead, t)

	// Do a random read on file and validate from gcs.
	expectedOutcome1 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offset5000, t)
	// Read file again from offset 1000 and validate from gcs.
	expectedOutcome2 := readChunkAndValidateObjectContentsFromGCS(s.ctx, s.storageClient, testFileName, offset1000, t)

	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), t)
	validate(expectedOutcome1, structuredReadLogs[0], false, false, 1, t)
	validate(expectedOutcome2, structuredReadLogs[1], false, false, 1, t)
	validateFileIsNotCached(testFileName, t)
}

func (s *cacheFileForRangeReadFalseTest) TestReadIsTreatedNonSequentialAfterFileIsRemovedFromCache(t *testing.T) {
	var testFileNames [2]string
	var expectedOutcome [4]*Expected
	testFileNames[0] = setupFileInTestDir(s.ctx, s.storageClient, testDirName, fileSizeSameAsCacheCapacity, t)
	testFileNames[1] = setupFileInTestDir(s.ctx, s.storageClient, testDirName, fileSizeSameAsCacheCapacity, t)
	randomReadChunkCount := fileSizeSameAsCacheCapacity / chunkSizeToRead
	readTillChunk := randomReadChunkCount / 2
	fh1 := operations.OpenFile(path.Join(testDirPath, testFileNames[0]), t)
	defer operations.CloseFile(fh1)
	fh2 := operations.OpenFile(path.Join(testDirPath, testFileNames[1]), t)
	defer operations.CloseFile(fh2)

	// Use file handle 1 to read file 1 partially.
	expectedOutcome[0] = readFileBetweenOffset(t, fh1, 0, int64(readTillChunk*chunkSizeToRead))
	// Use file handle 2 to read file 2 partially. This will evict file 1 from
	// cache due to cache capacity constraints.
	expectedOutcome[1] = readFileBetweenOffset(t, fh2, 0, int64(readTillChunk*chunkSizeToRead))
	// Read remaining file 1. File 2 remains cached. Cache eviction happens on
	// cache handler creation, which is tied to the file handle. Since the handle
	// isn't recreated, eviction doesn't occur.
	expectedOutcome[2] = readFileBetweenOffset(t, fh1, int64(readTillChunk*chunkSizeToRead)+1, fileSizeSameAsCacheCapacity)
	// Read remaining file 2.
	expectedOutcome[3] = readFileBetweenOffset(t, fh2, int64(readTillChunk*chunkSizeToRead)+1, fileSizeSameAsCacheCapacity)

	// Merge the expected outcomes.
	expectedOutcome[0].EndTimeStampSeconds = expectedOutcome[2].EndTimeStampSeconds
	expectedOutcome[0].content = expectedOutcome[0].content + expectedOutcome[2].content
	expectedOutcome[1].EndTimeStampSeconds = expectedOutcome[3].EndTimeStampSeconds
	expectedOutcome[1].content = expectedOutcome[1].content + expectedOutcome[3].content
	// Parse the logs and validate with expected outcome.
	structuredReadLogs := read_logs.GetStructuredLogsSortedByTimestamp(setup.LogFile(), t)
	require.Equal(t, 2, len(structuredReadLogs))
	validate(expectedOutcome[0], structuredReadLogs[0], true, false, randomReadChunkCount, t)
	validate(expectedOutcome[1], structuredReadLogs[1], true, false, randomReadChunkCount, t)
	// Validate after cache eviction, read was considered non-sequential and cache
	// hit false for first file.
	// Checking for the last chunk, not readTillChunk+1, due to potential kernel
	// over-reads on some architectures.
	assert.False(t, structuredReadLogs[0].Chunks[randomReadChunkCount-1].IsSequential)
	assert.False(t, structuredReadLogs[0].Chunks[randomReadChunkCount-1].CacheHit)
	// Validate for 2nd file read was considered sequential because of no cache eviction.
	assert.True(t, structuredReadLogs[1].Chunks[randomReadChunkCount-1].IsSequential)
	if !s.isParallelDownloadsEnabled {
		// When parallel downloads are enabled, we can't concretely say that the read will be cache Hit.
		assert.True(t, structuredReadLogs[1].Chunks[randomReadChunkCount-1].CacheHit)
	}

	validateFileIsNotCached(testFileNames[0], t)
	validateFileInCacheDirectory(testFileNames[1], fileSizeSameAsCacheCapacity, s.ctx, s.storageClient, t)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestCacheFileForRangeReadFalseTest(t *testing.T) {
	ts := &cacheFileForRangeReadFalseTest{ctx: context.Background()}
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

	// Run tests with parallel downloads disabled.
	flagsSet := []gcsfuseTestFlags{
		{
			cliFlags:                []string{"--implicit-dirs"},
			cacheSize:               cacheCapacityForRangeReadTestInMiB,
			cacheFileForRangeRead:   false,
			fileName:                configFileName,
			enableParallelDownloads: false,
			enableODirect:           false,
			cacheDirPath:            getDefaultCacheDirPathForTests(),
		},
		{
			cliFlags:                nil,
			cacheSize:               cacheCapacityForRangeReadTestInMiB,
			cacheFileForRangeRead:   false,
			fileName:                configFileName,
			enableParallelDownloads: false,
			enableODirect:           false,
			cacheDirPath:            ramCacheDir,
		},
	}
	flagsSet = appendClientProtocolConfigToFlagSet(flagsSet)
	for _, flags := range flagsSet {
		configFilePath := createConfigFile(&flags)
		ts.flags = []string{"--config-file=" + configFilePath}
		if flags.cliFlags != nil {
			ts.flags = append(ts.flags, flags.cliFlags...)
		}
		log.Printf("Running tests with flags: %s", ts.flags)
		test_setup.RunTests(t, ts)
	}

	// Run tests with parallel downloads enabled.
	flagsSet = []gcsfuseTestFlags{
		{
			cliFlags:                nil,
			cacheSize:               cacheCapacityForRangeReadTestInMiB,
			cacheFileForRangeRead:   false,
			fileName:                configFileNameForParallelDownloadTests,
			enableParallelDownloads: true,
			enableODirect:           false,
			cacheDirPath:            getDefaultCacheDirPathForTests(),
		},
		{
			cliFlags:                nil,
			cacheSize:               cacheCapacityForRangeReadTestInMiB,
			cacheFileForRangeRead:   false,
			fileName:                configFileNameForParallelDownloadTests,
			enableParallelDownloads: true,
			enableODirect:           false,
			cacheDirPath:            ramCacheDir,
		},
		{
			cliFlags:                nil,
			cacheSize:               cacheCapacityForRangeReadTestInMiB,
			cacheFileForRangeRead:   false,
			fileName:                configFileNameForParallelDownloadTests,
			enableParallelDownloads: true,
			enableODirect:           true,
			cacheDirPath:            getDefaultCacheDirPathForTests(),
		},
		{
			cliFlags:                nil,
			cacheSize:               cacheCapacityForRangeReadTestInMiB,
			cacheFileForRangeRead:   false,
			fileName:                configFileNameForParallelDownloadTests,
			enableParallelDownloads: true,
			enableODirect:           true,
			cacheDirPath:            ramCacheDir,
		},
	}
	flagsSet = appendClientProtocolConfigToFlagSet(flagsSet)
	for _, flags := range flagsSet {
		configFilePath := createConfigFile(&flags)
		ts.flags = []string{"--config-file=" + configFilePath}
		if flags.cliFlags != nil {
			ts.flags = append(ts.flags, flags.cliFlags...)
		}
		ts.isParallelDownloadsEnabled = true
		log.Printf("Running tests with flags: %s", ts.flags)
		test_setup.RunTests(t, ts)
	}
}
