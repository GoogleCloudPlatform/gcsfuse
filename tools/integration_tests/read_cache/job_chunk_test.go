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
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type jobChunkTest struct {
	flags         []string
	storageClient *storage.Client
	ctx           context.Context
	chunkSize     int64
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

func (s *jobChunkTest) TestJobChunkSize(t *testing.T) {
	var fileSize int64 = 24 * util.MiB
	chunkCount := fileSize / s.chunkSize
	testFileName := setupFileInTestDir(s.ctx, s.storageClient, testDirName, fileSize, t)

	expectedOutcome := readFileAndValidateCacheWithGCS(s.ctx, s.storageClient, testFileName, fileSize, false, t)

	// Parse the log file and validate cache hit or miss from the structured logs.
	structuredJobLogs := read_logs.GetJobLogsSortedByTimestamp(setup.LogFile(), t)
	assert.Equal(t, expectedOutcome.BucketName, structuredJobLogs[0].BucketName)
	assert.Equal(t, expectedOutcome.ObjectName, structuredJobLogs[0].ObjectName)
	assert.EqualValues(t, chunkCount, len(structuredJobLogs[0].JobEntries))
	for i := 0; int64(i) < chunkCount; i++ {
		offset := s.chunkSize * int64(i+1)
		assert.Equal(t, offset, structuredJobLogs[0].JobEntries[i].Offset)
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

	var cacheSizeMB int64 = 24
	var chunkSizeForReadCache int64 = 8
	ts.flags = []string{"--config-file=" + createConfigFile(cacheSizeMB, true, configFileName, false)}
	ts.chunkSize = chunkSizeForReadCache * util.MiB
	log.Printf("Running tests with flags: %s", ts.flags)
	test_setup.RunTests(t, ts)

	ts.flags = []string{"--config-file=" +
		createConfigFileForJobChunkTest(cacheSizeMB, false, "unlimitedMaxParallelDownloads", parallelDownloadsPerFile, maxParallelDownloads, downloadChunkSizeMB)}
	ts.chunkSize = parallelDownloadsPerFile * downloadChunkSizeMB * util.MiB
	log.Printf("Running tests with flags: %s", ts.flags)
	test_setup.RunTests(t, ts)

	parallelDownloadsPerFile := 4
	maxParallelDownloads := 1
	downloadChunkSizeMB := 3
	ts.flags = []string{"--config-file=" +
		createConfigFileForJobChunkTest(cacheSizeMB, false, "limitedMaxParallelDownloads", parallelDownloadsPerFile, maxParallelDownloads, downloadChunkSizeMB)}
	ts.chunkSize = int64(maxParallelDownloads+1) * int64(downloadChunkSizeMB) * util.MiB
	log.Printf("Running tests with flags: %s", ts.flags)
	test_setup.RunTests(t, ts)

}
