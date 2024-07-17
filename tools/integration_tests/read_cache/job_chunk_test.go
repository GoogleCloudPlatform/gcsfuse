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
	"log"
	"path"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
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

type jobChunkTest struct {
	flags                           []string
	storageClient                   *storage.Client
	ctx                             context.Context
	chunkSize                       int64
	isLimitedByMaxParallelDownloads bool
}

func (s *jobChunkTest) Setup(t *testing.T) {
	setupForMountedDirectoryTests()
	// Clean up the cache directory path as gcsfuse don't clean up on mounting.
	operations.RemoveDir(cacheDirPath)
	mountGCSFuseAndSetupTestDir(s.flags, s.ctx, s.storageClient, testDirName)
}

func (s *jobChunkTest) Teardown(t *testing.T) {
	setup.UnmountGCSFuseAndDeleteLogFile(rootDir)
}

func createConfigFileForJobChunkTest(cacheSize int64, cacheFileForRangeRead bool, fileName string, parallelDownloadsPerFile, maxParallelDownloads, downloadChunkSizeMB int) string {
	cacheDirPath = path.Join(setup.TestDir(), cacheDirName)

	// Set up config file for file cache.
	mountConfig := config.MountConfig{
		FileCacheConfig: config.FileCacheConfig{
			// Keeping the size as low because the operations are performed on small
			// files
			MaxSizeMB:                cacheSize,
			CacheFileForRangeRead:    cacheFileForRangeRead,
			EnableParallelDownloads:  true,
			ParallelDownloadsPerFile: parallelDownloadsPerFile,
			MaxParallelDownloads:     maxParallelDownloads,
			DownloadChunkSizeMB:      downloadChunkSizeMB,
			EnableCRC:                enableCrcCheck,
		},
		CacheDir: cacheDirPath,
		LogConfig: config.LogConfig{
			Severity:        config.TRACE,
			Format:          "json",
			FilePath:        setup.LogFile(),
			LogRotateConfig: config.DefaultLogRotateConfig(),
		},
	}
	filePath := setup.YAMLConfigFile(mountConfig, fileName)
	return filePath
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

//func (s *jobChunkTest) TestJobChunkSizeForSingleFileReads(t *testing.T) {
//	var fileSize int64 = 24 * util.MiB
//	chunkCount := math.Ceil(float64(fileSize) / float64(s.chunkSize))
//	testFileName := setupFileInTestDir(s.ctx, s.storageClient, testDirName, fileSize, t)
//
//	expectedOutcome := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, fileSize, false, t)
//
//	// Parse the log file and validate cache hit or miss from the structured logs.
//	structuredJobLogs := read_logs.GetJobLogsSortedByTimestamp(setup.LogFile(), t)
//	assert.Equal(t, expectedOutcome.BucketName, structuredJobLogs[0].BucketName)
//	assert.Equal(t, expectedOutcome.ObjectName, structuredJobLogs[0].ObjectName)
//	assert.EqualValues(t, chunkCount, len(structuredJobLogs[0].JobEntries))
//	for i := 0; int64(i) < int64(chunkCount); i++ {
//		offset := min(s.chunkSize*int64(i+1), fileSize)
//		assert.Equal(t, offset, structuredJobLogs[0].JobEntries[i].Offset)
//	}
//}

func (s *jobChunkTest) TestJobChunkSizeForMultipleFileReads(t *testing.T) {
	var fileSize int64 = 24 * util.MiB
	chunkCount := fileSize / s.chunkSize
	var testFileNames [2]string
	var expectedOutcome [2]*Expected
	testFileNames[0] = setupFileInTestDir(s.ctx, s.storageClient, testDirName, fileSize, t)
	testFileNames[1] = setupFileInTestDir(s.ctx, s.storageClient, testDirName, fileSize, t)

	// Read 2 files in parallel.
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			expectedOutcome[i] = readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileNames[i], fileSize, false, t)
		}()
	}
	wg.Wait()

	// Parse the log file and validate cache hit or miss from the structured logs.
	structuredJobLogs := read_logs.GetJobLogsSortedByTimestamp(setup.LogFile(), t)
	require.Equal(t, 2, len(structuredJobLogs))
	// Goroutine execution order isn't guaranteed.
	// If the object name in expected outcome doesn't align with the logs, swap
	// the expected outcome objects and file names at positions 0 and 1.
	if expectedOutcome[0].ObjectName != structuredJobLogs[0].ObjectName {
		expectedOutcome[0], expectedOutcome[1] = expectedOutcome[1], expectedOutcome[0]
		testFileNames[0], testFileNames[1] = testFileNames[1], testFileNames[0]
	}
	assert.Equal(t, expectedOutcome[0].BucketName, structuredJobLogs[0].BucketName)
	assert.Equal(t, expectedOutcome[1].BucketName, structuredJobLogs[1].BucketName)
	assert.Equal(t, expectedOutcome[0].ObjectName, structuredJobLogs[0].ObjectName)
	assert.Equal(t, expectedOutcome[1].ObjectName, structuredJobLogs[1].ObjectName)
	if s.isLimitedByMaxParallelDownloads {
		assert.LessOrEqual(t, chunkCount, int64(len(structuredJobLogs[0].JobEntries)))
		for i := 0; int64(i) < int64(len(structuredJobLogs[0].JobEntries)); i++ {
			offset := min(s.chunkSize*int64(i+1), fileSize)
			assert.GreaterOrEqual(t, offset, structuredJobLogs[0].JobEntries[i].Offset)
		}
	} else {
		assert.EqualValues(t, chunkCount, len(structuredJobLogs[0].JobEntries))
		for i := 0; int64(i) < chunkCount; i++ {
			offset := min(s.chunkSize*int64(i+1), fileSize)
			assert.Equal(t, offset, structuredJobLogs[0].JobEntries[i].Offset)
		}
	}
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestJobChunkTest(t *testing.T) {
	ts := &jobChunkTest{ctx: context.Background()}
	// Create storage client before running tests.
	closeStorageClient := client.CreateStorageClientWithTimeOut(&ts.ctx, &ts.storageClient, 15*time.Minute)
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

	var cacheSizeMB int64 = 48

	// Tests to validate chunk size when read cache parallel downloads are disabled.
	var chunkSizeForReadCache int64 = 8
	ts.flags = []string{"--config-file=" + createConfigFile(cacheSizeMB, true, configFileName, false)}
	ts.chunkSize = chunkSizeForReadCache * util.MiB
	log.Printf("Running tests with flags: %s", ts.flags)
	test_setup.RunTests(t, ts)

	// Tests to validate chunk size when read cache parallel downloads are enabled
	// with unlimited max parallel downloads.
	ts.flags = []string{"--config-file=" +
		createConfigFileForJobChunkTest(cacheSizeMB, false, "unlimitedMaxParallelDownloads", parallelDownloadsPerFile, maxParallelDownloads, downloadChunkSizeMB)}
	ts.chunkSize = parallelDownloadsPerFile * downloadChunkSizeMB * util.MiB
	log.Printf("Running tests with flags: %s", ts.flags)
	test_setup.RunTests(t, ts)

	// Tests to validate chunk size when read cache parallel downloads are enabled
	// with go-routines not limited by max parallel downloads.
	parallelDownloadsPerFile := 4
	maxParallelDownloads := 24 // maxParallelDownloads = parallelDownloadsPerFile * downloadChunkSizeMB * number of files
	downloadChunkSizeMB := 3
	ts.flags = []string{"--config-file=" +
		createConfigFileForJobChunkTest(cacheSizeMB, false, "limitedMaxParallelDownloadsNotEffectingChunkSize", parallelDownloadsPerFile, maxParallelDownloads, downloadChunkSizeMB)}
	ts.chunkSize = int64(parallelDownloadsPerFile) * int64(downloadChunkSizeMB) * util.MiB
	log.Printf("Running tests with flags: %s", ts.flags)
	test_setup.RunTests(t, ts)

	// Tests to validate chunk size when read cache parallel downloads are enabled
	// with go-routines limited by max parallel downloads.
	parallelDownloadsPerFile = 4
	maxParallelDownloads = 2
	downloadChunkSizeMB = 3
	ts.flags = []string{"--config-file=" +
		createConfigFileForJobChunkTest(cacheSizeMB, false, "limitedMaxParallelDownloadsEffectingChunkSize", parallelDownloadsPerFile, maxParallelDownloads, downloadChunkSizeMB)}
	ts.chunkSize = int64(maxParallelDownloads+1) * int64(downloadChunkSizeMB) * util.MiB
	ts.isLimitedByMaxParallelDownloads = true
	log.Printf("Running tests with flags: %s", ts.flags)
	test_setup.RunTests(t, ts)

}
