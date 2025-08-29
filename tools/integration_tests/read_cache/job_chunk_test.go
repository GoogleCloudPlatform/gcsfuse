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
	"sync"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/log_parser/json_parser/read_logs"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const cacheSizeMB int64 = 48

type jobChunkTest struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
	chunkSize     int64
	suite.Suite
}

func (s *jobChunkTest) SetupTest() {
	setupForMountedDirectoryTests()
	// Clean up the cache directory path as gcsfuse don't clean up on mounting.
	operations.RemoveDir(cacheDirPath)
	mountGCSFuseAndSetupTestDir(s.flags, s.ctx, s.storageClient)
}

func (s *jobChunkTest) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
	setup.UnmountGCSFuseAndDeleteLogFile(rootDir)
}

func createConfigFileForJobChunkTest(cacheFileForRangeRead bool, fileName string, parallelDownloadsPerFile, maxParallelDownloads, downloadChunkSizeMB int, clientProtocol string) string {
	cacheDirPath = path.Join(setup.TestDir(), cacheDirName)

	// Set up config file for file cache.
	mountConfig := map[string]interface{}{
		"file-cache": map[string]interface{}{
			"max-size-mb":                 cacheSizeMB,
			"cache-file-for-range-read":   cacheFileForRangeRead,
			"enable-parallel-downloads":   true,
			"parallel-downloads-per-file": parallelDownloadsPerFile,
			"max-parallel-downloads":      maxParallelDownloads,
			"download-chunk-size-mb":      downloadChunkSizeMB,
			"enable-crc":                  enableCrcCheck,
		},
		"cache-dir": cacheDirPath,
		"gcs-connection": map[string]interface{}{
			"client-protocol": clientProtocol,
		},
	}
	filePath := setup.YAMLConfigFile(mountConfig, fileName)
	return filePath
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *jobChunkTest) TestJobChunkSizeForSingleFileReads() {
	var fileSize int64 = 16 * util.MiB
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, fileSize, s.T())

	expectedOutcome := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, fileSize, false, s.T())

	// Parse the log file and validate cache hit or miss from the structured logs.
	structuredJobLogs := read_logs.GetJobLogsSortedByTimestamp(setup.LogFile(), s.T())
	assert.Equal(s.T(), expectedOutcome.BucketName, structuredJobLogs[0].BucketName)
	assert.Equal(s.T(), expectedOutcome.ObjectName, structuredJobLogs[0].ObjectName)

	// We need to check that downloadedOffset is always greater than the previous downloadedOffset
	// and is in multiples of chunkSize.
	for i := 1; i < len(structuredJobLogs[0].JobEntries); i++ {
		offsetDiff := structuredJobLogs[0].JobEntries[i].Offset - structuredJobLogs[0].JobEntries[i-1].Offset
		assert.Greater(s.T(), offsetDiff, int64(0))
		// This is true for all entries except last one.
		// Will be true for last entry only if the fileSize is multiple of chunkSize.
		assert.Equal(s.T(), int64(0), offsetDiff%s.chunkSize)
	}

	// Validate that last downloadedOffset is same as fileSize.
	assert.Equal(s.T(), fileSize, structuredJobLogs[0].JobEntries[len(structuredJobLogs[0].JobEntries)-1].Offset)
}

func (s *jobChunkTest) TestJobChunkSizeForMultipleFileReads() {
	var fileSize int64 = 16 * util.MiB
	var testFileNames [2]string
	var expectedOutcome [2]*Expected
	testFileNames[0] = setupFileInTestDir(s.ctx, s.storageClient, fileSize, s.T())
	testFileNames[1] = setupFileInTestDir(s.ctx, s.storageClient, fileSize, s.T())

	// Read 2 files in parallel.
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			expectedOutcome[i] = readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileNames[i], fileSize, false, s.T())
		}()
	}
	wg.Wait()

	// Parse the log file and validate cache hit or miss from the structured logs.
	structuredJobLogs := read_logs.GetJobLogsSortedByTimestamp(setup.LogFile(), s.T())
	require.Equal(s.T(), 2, len(structuredJobLogs))
	// Goroutine execution order isn't guaranteed.
	// If the object name in expected outcome doesn't align with the logs, swap
	// the expected outcome objects and file names at positions 0 and 1.
	if expectedOutcome[0].ObjectName != structuredJobLogs[0].ObjectName {
		expectedOutcome[0], expectedOutcome[1] = expectedOutcome[1], expectedOutcome[0]
		testFileNames[0], testFileNames[1] = testFileNames[1], testFileNames[0]
	}

	for fileIndex := 0; fileIndex < 2; fileIndex++ {
		assert.Equal(s.T(), expectedOutcome[fileIndex].BucketName, structuredJobLogs[fileIndex].BucketName)
		assert.Equal(s.T(), expectedOutcome[fileIndex].ObjectName, structuredJobLogs[fileIndex].ObjectName)

		// We need to check that downloadedOffset is always greater than the previous downloadedOffset
		// and is in multiples of chunkSize.
		entriesLen := len(structuredJobLogs[fileIndex].JobEntries)
		for entryIndex := 1; entryIndex < entriesLen; entryIndex++ {
			offsetDiff := structuredJobLogs[fileIndex].JobEntries[entryIndex].Offset - structuredJobLogs[fileIndex].JobEntries[entryIndex-1].Offset
			assert.Greater(s.T(), offsetDiff, int64(0))
			// This is true for all entries except last one.
			// Will be true for last entry only if the fileSize is multiple of chunkSize.
			assert.Equal(s.T(), int64(0), offsetDiff%s.chunkSize)
		}

		// Validate that last downloadedOffset is same as fileSize.
		assert.Equal(s.T(), fileSize, structuredJobLogs[fileIndex].JobEntries[entriesLen-1].Offset)
	}
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestJobChunkTest(t *testing.T) {
	ts := &jobChunkTest{ctx: context.Background()}
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
		suite.Run(t, ts)
		return
	}

	// Tests to validate chunk size when read cache parallel downloads are disabled.
	var chunkSizeForReadCache int64 = 8
	ts.flags = []string{"--config-file=" + createConfigFile(&gcsfuseTestFlags{cacheSize: cacheSizeMB, cacheFileForRangeRead: true, fileName: configFileName, enableParallelDownloads: false, enableODirect: false, cacheDirPath: getDefaultCacheDirPathForTests(), clientProtocol: http1ClientProtocol})}
	ts.chunkSize = chunkSizeForReadCache * util.MiB
	log.Printf("Running tests with flags: %s", ts.flags)
	suite.Run(t, ts)

	// Tests to validate chunk size when read cache parallel downloads are disabled with grpc client protocol.
	ts.flags = []string{"--config-file=" + createConfigFile(&gcsfuseTestFlags{cacheSize: cacheSizeMB, cacheFileForRangeRead: true, fileName: configFileName, enableParallelDownloads: false, enableODirect: false, cacheDirPath: getDefaultCacheDirPathForTests(), clientProtocol: grpcClientProtocol})}
	ts.chunkSize = chunkSizeForReadCache * util.MiB
	log.Printf("Running tests with flags: %s", ts.flags)
	suite.Run(t, ts)

	// Tests to validate chunk size when read cache parallel downloads are enabled
	// with unlimited max parallel downloads.
	ts.flags = []string{"--config-file=" +
		createConfigFileForJobChunkTest(false, "unlimitedMaxParallelDownloads", parallelDownloadsPerFile, maxParallelDownloads, downloadChunkSizeMB, http1ClientProtocol)}
	ts.chunkSize = downloadChunkSizeMB * util.MiB
	log.Printf("Running tests with flags: %s", ts.flags)
	suite.Run(t, ts)

	// Tests to validate chunk size when read cache parallel downloads are enabled
	// with unlimited max parallel downloads with grpc enabled.
	ts.flags = []string{"--config-file=" +
		createConfigFileForJobChunkTest(false, "unlimitedMaxParallelDownloads", parallelDownloadsPerFile, maxParallelDownloads, downloadChunkSizeMB, grpcClientProtocol)}
	ts.chunkSize = downloadChunkSizeMB * util.MiB
	log.Printf("Running tests with flags: %s", ts.flags)
	suite.Run(t, ts)

	// Tests to validate chunk size when read cache parallel downloads are enabled
	// with go-routines not limited by max parallel downloads.
	parallelDownloadsPerFile := 4
	maxParallelDownloads := 9 // maxParallelDownloads > parallelDownloadsPerFile * number of files being accessed concurrently.
	downloadChunkSizeMB := 4
	ts.flags = []string{"--config-file=" +
		createConfigFileForJobChunkTest(false, "limitedMaxParallelDownloadsNotEffectingChunkSize", parallelDownloadsPerFile, maxParallelDownloads, downloadChunkSizeMB, http1ClientProtocol)}
	ts.chunkSize = int64(downloadChunkSizeMB) * util.MiB
	log.Printf("Running tests with flags: %s", ts.flags)
	suite.Run(t, ts)

	// Tests to validate chunk size when read cache parallel downloads are enabled
	// with go-routines not limited by max parallel downloads with grpc enabled.
	ts.flags = []string{"--config-file=" +
		createConfigFileForJobChunkTest(false, "limitedMaxParallelDownloadsNotEffectingChunkSize", parallelDownloadsPerFile, maxParallelDownloads, downloadChunkSizeMB, grpcClientProtocol)}
	ts.chunkSize = int64(downloadChunkSizeMB) * util.MiB
	log.Printf("Running tests with flags: %s", ts.flags)
	suite.Run(t, ts)

	// Tests to validate chunk size when read cache parallel downloads are enabled
	// with go-routines limited by max parallel downloads.
	parallelDownloadsPerFile = 4
	maxParallelDownloads = 2
	downloadChunkSizeMB = 4
	ts.flags = []string{"--config-file=" +
		createConfigFileForJobChunkTest(false, "limitedMaxParallelDownloadsEffectingChunkSize", parallelDownloadsPerFile, maxParallelDownloads, downloadChunkSizeMB, http1ClientProtocol)}
	ts.chunkSize = int64(downloadChunkSizeMB) * util.MiB
	log.Printf("Running tests with flags: %s", ts.flags)
	suite.Run(t, ts)

	// Tests to validate chunk size when read cache parallel downloads are enabled
	// with go-routines limited by max parallel downloads with grpc enabled.
	ts.flags = []string{"--config-file=" +
		createConfigFileForJobChunkTest(false, "limitedMaxParallelDownloadsEffectingChunkSize", parallelDownloadsPerFile, maxParallelDownloads, downloadChunkSizeMB, grpcClientProtocol)}
	ts.chunkSize = int64(downloadChunkSizeMB) * util.MiB
	log.Printf("Running tests with flags: %s", ts.flags)
	suite.Run(t, ts)
}
