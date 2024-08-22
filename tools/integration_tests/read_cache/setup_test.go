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
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/dynamic_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/only_dir_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

const (
	testDirName                            = "ReadCacheTest"
	onlyDirMounted                         = "OnlyDirMountReadCache"
	cacheSubDirectoryName                  = "gcsfuse-file-cache"
	smallContentSize                       = 128 * util.KiB
	chunkSizeToRead                        = 128 * util.KiB
	fileSize                               = 3 * util.MiB
	fileSizeSameAsCacheCapacity            = cacheCapacityForRangeReadTestInMiB * util.MiB
	fileSizeForRangeRead                   = 8 * util.MiB
	chunksRead                             = fileSize / chunkSizeToRead
	testFileName                           = "foo"
	cacheCapacityInMB                      = 9
	NumberOfFilesWithinCacheLimit          = (cacheCapacityInMB * util.MiB) / fileSize
	NumberOfFilesMoreThanCacheLimit        = (cacheCapacityInMB*util.MiB)/fileSize + 1
	largeFileSize                          = 15 * util.MiB
	largeFileCacheCapacity                 = 15
	largeFileName                          = "15MBFile"
	largeFileChunksRead                    = largeFileSize / chunkSizeToRead
	chunksReadAfterUpdate                  = 1
	metadataCacheTTlInSec                  = 10
	testFileNameSuffixLength               = 4
	zeroOffset                             = 0
	randomReadOffset                       = 9 * util.MiB
	configFileName                         = "config"
	configFileNameForParallelDownloadTests = "configForReadCacheWithParallelDownload"
	offset5000                             = 5000
	offset1000                             = 1000
	offsetForRangeReadWithin8MB            = 4 * util.MiB
	offset10MiB                            = 10 * util.MiB
	cacheCapacityForRangeReadTestInMiB     = 50
	cacheDirName                           = "cache-dir"
	logFileNameForMountedDirectoryTests    = "/tmp/gcsfuse_read_cache_test_logs/log.json"
	parallelDownloadsPerFile               = 4
	maxParallelDownloads                   = -1
	downloadChunkSizeMB                    = 3
	enableCrcCheck                         = true
)

var (
	testDirPath  string
	cacheDirPath string
	mountFunc    func([]string) error
	// mount directory is where our tests run.
	mountDir string
	// root directory is the directory to be unmounted.
	rootDir       string
	storageClient *storage.Client
	ctx           context.Context
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func setupForMountedDirectoryTests() {
	if setup.MountedDirectory() != "" {
		cacheDirPath = path.Join(os.TempDir(), cacheDirName)
		mountDir = setup.MountedDirectory()
		setup.SetLogFile(logFileNameForMountedDirectoryTests)
	}
}

func mountGCSFuseAndSetupTestDir(flags []string, ctx context.Context, storageClient *storage.Client, testDirName string) {
	setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)
	setup.SetMntDir(mountDir)
	testDirPath = client.SetupTestDirectory(ctx, storageClient, testDirName)
}

func getDefaultCacheDirPathForTests() string {
	return path.Join(setup.TestDir(), cacheDirName)
}

func createConfigFile(cacheSize int64, cacheFileForRangeRead bool, fileName string, enableParallelDownloads bool, customCacheDirPath string) string {
	cacheDirPath = customCacheDirPath

	// Set up config file for file cache.
	mountConfig := map[string]interface{}{
		"file-cache": map[string]interface{}{
			"max-size-mb":                 cacheSize,
			"cache-file-for-range-read":   cacheFileForRangeRead,
			"enable-parallel-downloads":   enableParallelDownloads,
			"parallel-downloads-per-file": parallelDownloadsPerFile,
			"max-parallel-downloads":      maxParallelDownloads,
			"download-chunk-size-mb":      downloadChunkSizeMB,
			"enable-crc":                  enableCrcCheck,
		},
		"cache-dir": cacheDirPath,
	}
	filePath := setup.YAMLConfigFile(mountConfig, fileName)
	return filePath
}

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	ctx = context.Background()
	closeStorageClient := client.CreateStorageClientWithCancel(&ctx, &storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	setup.RunTestsForMountedDirectoryFlag(m)

	// Else run tests for testBucket.
	// Set up test directory.
	setup.SetUpTestDirForTestBucketFlag()

	// Save mount and root directory variables.
	mountDir, rootDir = setup.MntDir(), setup.MntDir()

	log.Println("Running static mounting tests...")
	mountFunc = static_mounting.MountGcsfuseWithStaticMounting
	successCode := m.Run()

	if successCode == 0 {
		log.Println("Running dynamic mounting tests...")
		// Save mount directory variable to have path of bucket to run tests.
		mountDir = path.Join(setup.MntDir(), setup.TestBucket())
		mountFunc = dynamic_mounting.MountGcsfuseWithDynamicMounting
		successCode = m.Run()
	}

	if successCode == 0 {
		log.Println("Running only dir mounting tests...")
		setup.SetOnlyDirMounted(onlyDirMounted + "/")
		mountDir = rootDir
		mountFunc = only_dir_mounting.MountGcsfuseWithOnlyDir
		successCode = m.Run()
		setup.CleanupDirectoryOnGCS(ctx, storageClient, path.Join(setup.TestBucket(), setup.OnlyDirMounted(), testDirName))
	}

	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(ctx, storageClient, path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)
}
