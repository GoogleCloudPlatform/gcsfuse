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

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/dynamic_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/only_dir_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

const (
	testDirName                        = "ReadCacheTest"
	onlyDirMounted                     = "OnlyDirMountReadCache"
	cacheSubDirectoryName              = "gcsfuse-file-cache"
	smallContentSize                   = 128 * util.KiB
	chunkSizeToRead                    = 128 * util.KiB
	fileSize                           = 3 * util.MiB
	fileSizeSameAsCacheCapacity        = cacheCapacityForRangeReadTestInMiB * util.MiB
	fileSizeForRangeRead               = 8 * util.MiB
	chunksRead                         = fileSize / chunkSizeToRead
	testFileName                       = "foo"
	cacheCapacityInMB                  = 9
	NumberOfFilesWithinCacheLimit      = (cacheCapacityInMB * util.MiB) / fileSize
	NumberOfFilesMoreThanCacheLimit    = (cacheCapacityInMB*util.MiB)/fileSize + 1
	largeFileSize                      = 15 * util.MiB
	largeFileCacheCapacity             = 15
	largeFileChunksRead                = largeFileSize / chunkSizeToRead
	chunksReadAfterUpdate              = 1
	metadataCacheTTlInSec              = 10
	testFileNameSuffixLength           = 4
	zeroOffset                         = 0
	randomReadOffset                   = 9 * util.MiB
	offset5000                         = 5000
	offset1000                         = 1000
	offsetForRangeReadWithin8MB        = 4 * util.MiB
	offset10MiB                        = 10 * util.MiB
	cacheCapacityForRangeReadTestInMiB = 50
	GKETempDir                         = "/gcsfuse-tmp"
)

var (
	cacheDirName string
	mountFunc    func(*test_suite.TestConfig, []string) error
	// mount directory is where our tests run.
	mountDir string
	// root directory is the directory to be unmounted.
	rootDir string
)

type env struct {
	storageClient *storage.Client
	ctx           context.Context
	testDirPath   string
	cfg           *test_suite.TestConfig
	bucketType    string
	cacheDirPath  string
}

var testEnv env

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func setupLogFileAndCacheDir(testName string) {
	var logFilePath string
	testEnv.cacheDirPath = path.Join(setup.TestDir(), GKETempDir, testName)
	logFilePath = path.Join(setup.TestDir(), GKETempDir, testName) + ".log"
	if testEnv.cfg.GKEMountedDirectory != "" {
		testEnv.cacheDirPath = path.Join(GKETempDir, testName)
		mountDir = testEnv.cfg.GKEMountedDirectory
		logFilePath = path.Join(GKETempDir, testName) + ".log"
		if setup.ConfigFile() == "" {
			// TODO: clean this up when GKE test migration completes.
			logFilePath = "/tmp/gcsfuse_read_cache_test_logs/log.json"
			if testEnv.bucketType == setup.FlatBucket {
				testEnv.cacheDirPath = "/tmp/cache-dir-read-cache-hns-false"
			} else {
				testEnv.cacheDirPath = "/tmp/cache-dir-read-cache-hns-true"
			}
		}
	}
	testEnv.cfg.LogFile = logFilePath
	setup.SetLogFile(logFilePath)
}

func mountGCSFuseAndSetupTestDir(flags []string, ctx context.Context, storageClient *storage.Client) {
	setup.MountGCSFuseWithGivenMountWithConfigFunc(testEnv.cfg, flags, mountFunc)
	setup.SetMntDir(mountDir)
	testEnv.testDirPath = client.SetupTestDirectory(ctx, storageClient, testDirName)
}

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// 1. Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.ReadCache) == 0 {
		log.Println("No configuration found for read_cache tests in config. Using flags instead.")
		// Populate the config manually.
		cfg.ReadCache = make([]test_suite.TestConfig, 1)
		cfg.ReadCache[0].TestBucket = setup.TestBucket()
		cfg.ReadCache[0].GKEMountedDirectory = setup.MountedDirectory()
		cfg.ReadCache[0].LogFile = setup.LogFile()
		// Initialize the slice to hold 15 specific test configurations
		cfg.ReadCache[0].Configs = make([]test_suite.ConfigItem, 20)
		cfg.ReadCache[0].Configs[0].Flags = []string{
			"--metadata-cache-ttl-secs=10 --file-cache-max-size-mb=9 --file-cache-enable-parallel-downloads=false --cache-dir=/gcsfuse-tmp/TestSmallCacheTTLTest --log-file=/gcsfuse-tmp/TestSmallCacheTTLTest.log --log-severity=TRACE --implicit-dirs --enable-kernel-reader=false",
			"--metadata-cache-ttl-secs=10 --file-cache-max-size-mb=9 --file-cache-enable-parallel-downloads=true -cache-dir=/gcsfuse-tmp/TestSmallCacheTTLTest --log-file=/gcsfuse-tmp/TestSmallCacheTTLTest.log --log-severity=TRACE --enable-kernel-reader=false",
			"--metadata-cache-ttl-secs=10 --file-cache-max-size-mb=9 --file-cache-enable-parallel-downloads=false --cache-dir=/gcsfuse-tmp/TestSmallCacheTTLTest --log-file=/gcsfuse-tmp/TestSmallCacheTTLTest.log --log-severity=TRACE --client-protocol=grpc --enable-kernel-reader=false",
			"--metadata-cache-ttl-secs=10 --file-cache-max-size-mb=9 --file-cache-enable-parallel-downloads=true -cache-dir=/gcsfuse-tmp/TestSmallCacheTTLTest --log-file=/gcsfuse-tmp/TestSmallCacheTTLTest.log --log-severity=TRACE -client-protocol=grpc --implicit-dirs --enable-kernel-reader=false",
		}
		cfg.ReadCache[0].Configs[0].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.ReadCache[0].Configs[0].Run = "TestSmallCacheTTLTest"

		cfg.ReadCache[0].Configs[1].Flags = []string{
			"--file-cache-max-size-mb=9 --file-cache-cache-file-for-range-read=true --cache-dir=/gcsfuse-tmp/TestReadOnlyTest --log-file=/gcsfuse-tmp/TestReadOnlyTest.log --log-severity=TRACE --file-cache-enable-parallel-downloads=false -implicit-dirs --enable-kernel-reader=false",
			"--file-cache-max-size-mb=9 --file-cache-enable-parallel-downloads=true --cache-dir=/gcsfuse-tmp/TestReadOnlyTest --log-file=/gcsfuse-tmp/TestReadOnlyTest.log --log-severity=TRACE --enable-kernel-reader=false",
			"--file-cache-max-size-mb=9 --file-cache-cache-file-for-range-read=true --cache-dir=/gcsfuse-tmp/TestReadOnlyTest --log-file=/gcsfuse-tmp/TestReadOnlyTest.log --log-severity=TRACE --file-cache-enable-parallel-downloads=false --implicit-dirs --o=ro --enable-kernel-reader=false",
			"--file-cache-max-size-mb=9 --file-cache-enable-parallel-downloads=true --cache-dir=/gcsfuse-tmp/TestReadOnlyTest --log-file=/gcsfuse-tmp/TestReadOnlyTest.log --log-severity=TRACE --o=ro --enable-kernel-reader=false",
			"--file-cache-max-size-mb=9 --file-cache-cache-file-for-range-read=true --cache-dir=/gcsfuse-tmp/TestReadOnlyTest --log-file=/gcsfuse-tmp/TestReadOnlyTest.log --log-severity=TRACE --file-cache-enable-parallel-downloads=false -implicit-dirs --client-protocol=grpc --enable-kernel-reader=false",
			"--file-cache-max-size-mb=9 --file-cache-enable-parallel-downloads=true --cache-dir=/gcsfuse-tmp/TestReadOnlyTest --log-file=/gcsfuse-tmp/TestReadOnlyTest.log --log-severity=TRACE --client-protocol=grpc --enable-kernel-reader=false",
			"--file-cache-max-size-mb=9 --file-cache-cache-file-for-range-read=true --cache-dir=/gcsfuse-tmp/TestReadOnlyTest --log-file=/gcsfuse-tmp/TestReadOnlyTest.log --log-severity=TRACE --file-cache-enable-parallel-downloads=false --implicit-dirs --o=ro --client-protocol=grpc --enable-kernel-reader=false",
			"--file-cache-max-size-mb=9 --file-cache-enable-parallel-downloads=true --cache-dir=/gcsfuse-tmp/TestReadOnlyTest --log-file=/gcsfuse-tmp/TestReadOnlyTest.log --log-severity=TRACE --o=ro --client-protocol=grpc --enable-kernel-reader=false",
		}
		cfg.ReadCache[0].Configs[1].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.ReadCache[0].Configs[1].Run = "TestReadOnlyTest"

		cfg.ReadCache[0].Configs[2].Flags = []string{
			"--implicit-dirs --file-cache-max-size-mb=15 --file-cache-enable-parallel-downloads=false --cache-dir=/gcsfuse-tmp/TestRangeReadTest --log-file=/gcsfuse-tmp/TestRangeReadTest.log --log-severity=TRACE --enable-kernel-reader=false",
			"--implicit-dirs --file-cache-max-size-mb=15 --file-cache-enable-parallel-downloads=false --cache-dir=/gcsfuse-tmp/TestRangeReadTest --log-file=/gcsfuse-tmp/TestRangeReadTest.log --log-severity=TRACE --client-protocol=grpc --enable-kernel-reader=false",
		}
		cfg.ReadCache[0].Configs[2].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.ReadCache[0].Configs[2].Run = "TestRangeReadTest"

		cfg.ReadCache[0].Configs[3].Flags = []string{
			"--implicit-dirs --file-cache-max-size-mb=15 --file-cache-enable-parallel-downloads=true --cache-dir=/gcsfuse-tmp/TestRangeReadWithParallelDownloadsTest --log-file=/gcsfuse-tmp/TestRangeReadWithParallelDownloadsTest.log --log-severity=TRACE --enable-kernel-reader=false",
			"--implicit-dirs --file-cache-max-size-mb=15 --file-cache-enable-parallel-downloads=true --cache-dir=/gcsfuse-tmp/TestRangeReadWithParallelDownloadsTest --log-file=/gcsfuse-tmp/TestRangeReadWithParallelDownloadsTest.log --log-severity=TRACE --client-protocol=grpc --enable-kernel-reader=false",
		}
		cfg.ReadCache[0].Configs[3].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.ReadCache[0].Configs[3].Run = "TestRangeReadWithParallelDownloadsTest"

		cfg.ReadCache[0].Configs[4].Flags = []string{
			"--file-cache-max-size-mb=9 --file-cache-enable-parallel-downloads=false --cache-dir=/gcsfuse-tmp/TestLocalModificationTest --log-file=/gcsfuse-tmp/TestLocalModificationTest.log --log-severity=TRACE --implicit-dirs --enable-kernel-reader=false",
			"--file-cache-max-size-mb=9 --file-cache-enable-parallel-downloads=true --cache-dir=/gcsfuse-tmp/TestLocalModificationTest --log-file=/gcsfuse-tmp/TestLocalModificationTest.log --log-severity=TRACE --enable-kernel-reader=false",
			"--file-cache-max-size-mb=9 --file-cache-enable-parallel-downloads=false --cache-dir=/gcsfuse-tmp/TestLocalModificationTest --log-file=/gcsfuse-tmp/TestLocalModificationTest.log --log-severity=TRACE --implicit-dirs --client-protocol=grpc --enable-kernel-reader=false",
			"--file-cache-max-size-mb=9 --file-cache-enable-parallel-downloads=true --cache-dir=/gcsfuse-tmp/TestLocalModificationTest --log-file=/gcsfuse-tmp/TestLocalModificationTest.log --log-severity=TRACE --client-protocol=grpc --enable-kernel-reader=false",
		}
		cfg.ReadCache[0].Configs[4].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.ReadCache[0].Configs[4].Run = "TestLocalModificationTest"

		cfg.ReadCache[0].Configs[5].Flags = []string{
			"--stat-cache-ttl=0s --file-cache-max-size-mb=9 --file-cache-enable-parallel-downloads=false --cache-dir=/gcsfuse-tmp/TestDisabledCacheTTLTest --log-file=/gcsfuse-tmp/TestDisabledCacheTTLTest.log --log-severity=TRACE --implicit-dirs --enable-kernel-reader=false",
			"--stat-cache-ttl=0s --file-cache-max-size-mb=9 --file-cache-enable-parallel-downloads=true --cache-dir=/gcsfuse-tmp/TestDisabledCacheTTLTest --log-file=/gcsfuse-tmp/TestDisabledCacheTTLTest.log --log-severity=TRACE --enable-kernel-reader=false",
			"--stat-cache-ttl=0s --file-cache-max-size-mb=9 --file-cache-enable-parallel-downloads=false --cache-dir=/gcsfuse-tmp/TestDisabledCacheTTLTest --log-file=/gcsfuse-tmp/TestDisabledCacheTTLTest.log --log-severity=TRACE --implicit-dirs --client-protocol=grpc --enable-kernel-reader=false",
			"--stat-cache-ttl=0s --file-cache-max-size-mb=9 --file-cache-enable-parallel-downloads=true --cache-dir=/gcsfuse-tmp/TestDisabledCacheTTLTest --log-file=/gcsfuse-tmp/TestDisabledCacheTTLTest.log --log-severity=TRACE --client-protocol=grpc --enable-kernel-reader=false",
		}
		cfg.ReadCache[0].Configs[5].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.ReadCache[0].Configs[5].Run = "TestDisabledCacheTTLTest"

		cfg.ReadCache[0].Configs[6].Flags = []string{
			"--file-cache-max-size-mb=50 --file-cache-cache-file-for-range-read=true --file-cache-enable-parallel-downloads=false --cache-dir=/gcsfuse-tmp/TestCacheFileForRangeReadTrueTest --log-file=/gcsfuse-tmp/TestCacheFileForRangeReadTrueTest.log --log-severity=TRACE --implicit-dirs --enable-kernel-reader=false",
			"--file-cache-max-size-mb=50 --file-cache-cache-file-for-range-read=true --file-cache-enable-parallel-downloads=true --cache-dir=/gcsfuse-tmp/TestCacheFileForRangeReadTrueTest --log-file=/gcsfuse-tmp/TestCacheFileForRangeReadTrueTest.log --log-severity=TRACE --enable-kernel-reader=false",
			"--file-cache-max-size-mb=50 --file-cache-cache-file-for-range-read=true --file-cache-enable-parallel-downloads=true --cache-dir=/gcsfuse-tmp/TestCacheFileForRangeReadTrueTest --log-file=/gcsfuse-tmp/TestCacheFileForRangeReadTrueTest.log --log-severity=TRACE --file-cache-enable-o-direct=true --enable-kernel-reader=false",
			"--file-cache-max-size-mb=50 --file-cache-cache-file-for-range-read=true --file-cache-enable-parallel-downloads=false --cache-dir=/gcsfuse-tmp/TestCacheFileForRangeReadTrueTest --log-file=/gcsfuse-tmp/TestCacheFileForRangeReadTrueTest.log --log-severity=TRACE --implicit-dirs --client-protocol=grpc --enable-kernel-reader=false",
			"--file-cache-max-size-mb=50 --file-cache-cache-file-for-range-read=true --file-cache-enable-parallel-downloads=true --cache-dir=/gcsfuse-tmp/TestCacheFileForRangeReadTrueTest --log-file=/gcsfuse-tmp/TestCacheFileForRangeReadTrueTest.log --log-severity=TRACE --client-protocol=grpc --enable-kernel-reader=false",
			"--file-cache-max-size-mb=50 --file-cache-cache-file-for-range-read=true --file-cache-enable-parallel-downloads=true --cache-dir=/gcsfuse-tmp/TestCacheFileForRangeReadTrueTest --log-file=/gcsfuse-tmp/TestCacheFileForRangeReadTrueTest.log --log-severity=TRACE --file-cache-enable-o-direct=true --client-protocol=grpc --enable-kernel-reader=false",
		}
		cfg.ReadCache[0].Configs[6].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.ReadCache[0].Configs[6].Run = "TestCacheFileForRangeReadTrueTest"

		//cfg.ReadCache[0].Configs[7].Flags = []string{
		//	"--file-cache-max-size-mb=50 --file-cache-cache-file-for-range-read=true --file-cache-enable-parallel-downloads=true --cache-dir=/dev/shm/TestCacheFileForRangeReadTrueWithRamCache --log-file=/gcsfuse-tmp/TestCacheFileForRangeReadTrueWithRamCache.log --log-severity=TRACE",
		//	"--file-cache-max-size-mb=50 --file-cache-cache-file-for-range-read=true --file-cache-enable-parallel-downloads=true --cache-dir=/dev/shm/TestCacheFileForRangeReadTrueWithRamCache --log-file=/gcsfuse-tmp/TestCacheFileForRangeReadTrueWithRamCache.log --log-severity=TRACE --file-cache-enable-o-direct=true",
		//	"--file-cache-max-size-mb=50 --file-cache-cache-file-for-range-read=true --file-cache-enable-parallel-downloads=false --cache-dir=/dev/shm/TestCacheFileForRangeReadTrueWithRamCache --log-file=/gcsfuse-tmp/TestCacheFileForRangeReadTrueWithRamCache.log --log-severity=TRACE",
		//	"--file-cache-max-size-mb=50 --file-cache-cache-file-for-range-read=true --file-cache-enable-parallel-downloads=true --cache-dir=/dev/shm/TestCacheFileForRangeReadTrueWithRamCache --log-file=/gcsfuse-tmp/TestCacheFileForRangeReadTrueWithRamCache.log --log-severity=TRACE --client-protocol=grpc",
		//	"--file-cache-max-size-mb=50 --file-cache-cache-file-for-range-read=true --file-cache-enable-parallel-downloads=true --cache-dir=/dev/shm/TestCacheFileForRangeReadTrueWithRamCache --log-file=/gcsfuse-tmp/TestCacheFileForRangeReadTrueWithRamCache.log --log-severity=TRACE --file-cache-enable-o-direct=true --client-protocol=grpc",
		//	"--file-cache-max-size-mb=50 --file-cache-cache-file-for-range-read=true --file-cache-enable-parallel-downloads=false --cache-dir=/dev/shm/TestCacheFileForRangeReadTrueWithRamCache --log-file=/gcsfuse-tmp/TestCacheFileForRangeReadTrueWithRamCache.log --log-severity=TRACE --client-protocol=grpc",
		//}
		//cfg.ReadCache[0].Configs[7].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		//cfg.ReadCache[0].Configs[7].Run = "TestCacheFileForRangeReadTrueWithRamCache"

		cfg.ReadCache[0].Configs[8].Flags = []string{
			"--implicit-dirs --file-cache-max-size-mb=50 --file-cache-enable-parallel-downloads=false --cache-dir=/gcsfuse-tmp/TestCacheFileForRangeReadFalseTest --log-file=/gcsfuse-tmp/TestCacheFileForRangeReadFalseTest.log --log-severity=TRACE --enable-kernel-reader=false",
			"--implicit-dirs --file-cache-max-size-mb=50 --file-cache-enable-parallel-downloads=false --cache-dir=/gcsfuse-tmp/TestCacheFileForRangeReadFalseTest --log-file=/gcsfuse-tmp/TestCacheFileForRangeReadFalseTest.log --log-severity=TRACE --client-protocol=grpc --enable-kernel-reader=false",
		}
		cfg.ReadCache[0].Configs[8].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.ReadCache[0].Configs[8].Run = "TestCacheFileForRangeReadFalseTest"

		//cfg.ReadCache[0].Configs[9].Flags = []string{
		//	"--file-cache-max-size-mb=50 --file-cache-enable-parallel-downloads=false --cache-dir=/dev/shm/TestCacheFileForRangeReadFalseWithRamCache --log-file=/gcsfuse-tmp/TestCacheFileForRangeReadFalseWithRamCache.log --log-severity=TRACE",
		//	"--file-cache-max-size-mb=50 --file-cache-enable-parallel-downloads=false --cache-dir=/dev/shm/TestCacheFileForRangeReadFalseWithRamCache --log-file=/gcsfuse-tmp/TestCacheFileForRangeReadFalseWithRamCache.log --log-severity=TRACE --client-protocol=grpc",
		//}
		//cfg.ReadCache[0].Configs[9].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		//cfg.ReadCache[0].Configs[9].Run = "TestCacheFileForRangeReadFalseWithRamCache"

		cfg.ReadCache[0].Configs[10].Flags = []string{
			"--file-cache-max-size-mb=50 --file-cache-enable-parallel-downloads=true --cache-dir=/gcsfuse-tmp/TestCacheFileForRangeReadFalseWithParallelDownloads --log-file=/gcsfuse-tmp/TestCacheFileForRangeReadFalseWithParallelDownloads.log --log-severity=TRACE --enable-kernel-reader=false",
			"--file-cache-max-size-mb=50 --file-cache-enable-parallel-downloads=true --cache-dir=/gcsfuse-tmp/TestCacheFileForRangeReadFalseWithParallelDownloads --log-file=/gcsfuse-tmp/TestCacheFileForRangeReadFalseWithParallelDownloads.log --log-severity=TRACE --file-cache-enable-o-direct=true --enable-kernel-reader=false",
			"--file-cache-max-size-mb=50 --file-cache-enable-parallel-downloads=true --cache-dir=/gcsfuse-tmp/TestCacheFileForRangeReadFalseWithParallelDownloads --log-file=/gcsfuse-tmp/TestCacheFileForRangeReadFalseWithParallelDownloads.log --log-severity=TRACE --client-protocol=grpc --enable-kernel-reader=false",
			"--file-cache-max-size-mb=50 --file-cache-enable-parallel-downloads=true --cache-dir=/gcsfuse-tmp/TestCacheFileForRangeReadFalseWithParallelDownloads --log-file=/gcsfuse-tmp/TestCacheFileForRangeReadFalseWithParallelDownloads.log --log-severity=TRACE --file-cache-enable-o-direct=true --client-protocol=grpc --enable-kernel-reader=false",
		}
		cfg.ReadCache[0].Configs[10].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.ReadCache[0].Configs[10].Run = "TestCacheFileForRangeReadFalseWithParallelDownloads"

		//cfg.ReadCache[0].Configs[11].Flags = []string{
		//	"--file-cache-max-size-mb=50 --file-cache-enable-parallel-downloads=true --cache-dir=/dev/shm/TestCacheFileForRangeReadFalseWithParallelDownloadsAndRamCache --log-file=/gcsfuse-tmp/TestCacheFileForRangeReadFalseWithParallelDownloadsAndRamCache.log --log-severity=TRACE",
		//	"--file-cache-max-size-mb=50 --file-cache-enable-parallel-downloads=true --cache-dir=/dev/shm/TestCacheFileForRangeReadFalseWithParallelDownloadsAndRamCache --log-file=/gcsfuse-tmp/TestCacheFileForRangeReadFalseWithParallelDownloadsAndRamCache.log --log-severity=TRACE --file-cache-enable-o-direct=true",
		//	"--file-cache-max-size-mb=50 --file-cache-enable-parallel-downloads=true --cache-dir=/dev/shm/TestCacheFileForRangeReadFalseWithParallelDownloadsAndRamCache --log-file=/gcsfuse-tmp/TestCacheFileForRangeReadFalseWithParallelDownloadsAndRamCache.log --log-severity=TRACE --client-protocol=grpc",
		//	"--file-cache-max-size-mb=50 --file-cache-enable-parallel-downloads=true --cache-dir=/dev/shm/TestCacheFileForRangeReadFalseWithParallelDownloadsAndRamCache --log-file=/gcsfuse-tmp/TestCacheFileForRangeReadFalseWithParallelDownloadsAndRamCache.log --log-severity=TRACE --file-cache-enable-o-direct=true --client-protocol=grpc",
		//}
		//cfg.ReadCache[0].Configs[11].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		//cfg.ReadCache[0].Configs[11].Run = "TestCacheFileForRangeReadFalseWithParallelDownloadsAndRamCache"

		cfg.ReadCache[0].Configs[12].Flags = []string{
			"--file-cache-max-size-mb=48 --file-cache-cache-file-for-range-read=true --file-cache-enable-parallel-downloads=false --cache-dir=/gcsfuse-tmp/TestJobChunkTest --log-file=/gcsfuse-tmp/TestJobChunkTest.log --log-severity=TRACE --enable-kernel-reader=false",
			"--file-cache-max-size-mb=48 --file-cache-cache-file-for-range-read=true --file-cache-enable-parallel-downloads=false --cache-dir=/gcsfuse-tmp/TestJobChunkTest --log-file=/gcsfuse-tmp/TestJobChunkTest.log --log-severity=TRACE --client-protocol=grpc --enable-kernel-reader=false",
		}
		cfg.ReadCache[0].Configs[12].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.ReadCache[0].Configs[12].Run = "TestJobChunkTest"

		cfg.ReadCache[0].Configs[13].Flags = []string{
			//with unlimited max parallel downloads.
			"--file-cache-max-size-mb=48 --file-cache-enable-parallel-downloads=true --file-cache-parallel-downloads-per-file=4 --file-cache-max-parallel-downloads=-1 --file-cache-download-chunk-size-mb=4 --file-cache-enable-crc=true --cache-dir=/gcsfuse-tmp/TestJobChunkTestWithParallelDownloads --log-file=/gcsfuse-tmp/TestJobChunkTestWithParallelDownloads.log --log-severity=TRACE --enable-kernel-reader=false",
			"--file-cache-max-size-mb=48 --file-cache-enable-parallel-downloads=true --file-cache-parallel-downloads-per-file=4 --file-cache-max-parallel-downloads=-1 --file-cache-download-chunk-size-mb=4 --file-cache-enable-crc=true --cache-dir=/gcsfuse-tmp/TestJobChunkTestWithParallelDownloads --log-file=/gcsfuse-tmp/TestJobChunkTestWithParallelDownloads.log --log-severity=TRACE --client-protocol=grpc --enable-kernel-reader=false",
			//with go-routines not limited by max parallel downloads.
			//maxParallelDownloads > parallelDownloadsPerFile * number of files being accessed concurrently.
			"--file-cache-max-size-mb=48 --file-cache-enable-parallel-downloads=true --file-cache-parallel-downloads-per-file=4 --file-cache-max-parallel-downloads=9 --file-cache-download-chunk-size-mb=4 --file-cache-enable-crc=true --cache-dir=/gcsfuse-tmp/TestJobChunkTestWithParallelDownloads --log-file=/gcsfuse-tmp/TestJobChunkTestWithParallelDownloads.log --log-severity=TRACE --enable-kernel-reader=false",
			"--file-cache-max-size-mb=48 --file-cache-enable-parallel-downloads=true --file-cache-parallel-downloads-per-file=4 --file-cache-max-parallel-downloads=9 --file-cache-download-chunk-size-mb=4 --file-cache-enable-crc=true --cache-dir=/gcsfuse-tmp/TestJobChunkTestWithParallelDownloads --log-file=/gcsfuse-tmp/TestJobChunkTestWithParallelDownloads.log --log-severity=TRACE --client-protocol=grpc --enable-kernel-reader=false",
			//with go-routines limited by max parallel downloads.
			"--file-cache-max-size-mb=48 --file-cache-enable-parallel-downloads=true --file-cache-parallel-downloads-per-file=4 --file-cache-max-parallel-downloads=2 --file-cache-download-chunk-size-mb=4 --file-cache-enable-crc=true --cache-dir=/gcsfuse-tmp/TestJobChunkTestWithParallelDownloads --log-file=/gcsfuse-tmp/TestJobChunkTestWithParallelDownloads.log --log-severity=TRACE --enable-kernel-reader=false",
			"--file-cache-max-size-mb=48 --file-cache-enable-parallel-downloads=true --file-cache-parallel-downloads-per-file=4 --file-cache-max-parallel-downloads=2 --file-cache-download-chunk-size-mb=4 --file-cache-enable-crc=true --cache-dir=/gcsfuse-tmp/TestJobChunkTestWithParallelDownloads --log-file=/gcsfuse-tmp/TestJobChunkTestWithParallelDownloads.log --log-severity=TRACE --client-protocol=grpc --enable-kernel-reader=false",
		}
		cfg.ReadCache[0].Configs[13].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.ReadCache[0].Configs[13].Run = "TestJobChunkTestWithParallelDownloads"

		cfg.ReadCache[0].Configs[13].Flags = []string{
			//with unlimited max parallel downloads.
			"--file-cache-max-size-mb=48 --file-cache-enable-parallel-downloads=true --file-cache-parallel-downloads-per-file=4 --file-cache-max-parallel-downloads=-1 --file-cache-download-chunk-size-mb=4 --file-cache-enable-crc=true --cache-dir=/gcsfuse-tmp/TestJobChunkTestWithParallelDownloads --log-file=/gcsfuse-tmp/TestJobChunkTestWithParallelDownloads.log --log-severity=TRACE --enable-kernel-reader=false",
			"--file-cache-max-size-mb=48 --file-cache-enable-parallel-downloads=true --file-cache-parallel-downloads-per-file=4 --file-cache-max-parallel-downloads=-1 --file-cache-download-chunk-size-mb=4 --file-cache-enable-crc=true --cache-dir=/gcsfuse-tmp/TestJobChunkTestWithParallelDownloads --log-file=/gcsfuse-tmp/TestJobChunkTestWithParallelDownloads.log --log-severity=TRACE --client-protocol=grpc --enable-kernel-reader=false",
			//with go-routines not limited by max parallel downloads.
			//maxParallelDownloads > parallelDownloadsPerFile * number of files being accessed concurrently.
			"--file-cache-max-size-mb=48 --file-cache-enable-parallel-downloads=true --file-cache-parallel-downloads-per-file=4 --file-cache-max-parallel-downloads=9 --file-cache-download-chunk-size-mb=4 --file-cache-enable-crc=true --cache-dir=/gcsfuse-tmp/TestJobChunkTestWithParallelDownloads --log-file=/gcsfuse-tmp/TestJobChunkTestWithParallelDownloads.log --log-severity=TRACE --enable-kernel-reader=false",
			"--file-cache-max-size-mb=48 --file-cache-enable-parallel-downloads=true --file-cache-parallel-downloads-per-file=4 --file-cache-max-parallel-downloads=9 --file-cache-download-chunk-size-mb=4 --file-cache-enable-crc=true --cache-dir=/gcsfuse-tmp/TestJobChunkTestWithParallelDownloads --log-file=/gcsfuse-tmp/TestJobChunkTestWithParallelDownloads.log --log-severity=TRACE --client-protocol=grpc --enable-kernel-reader=false",
			//with go-routines limited by max parallel downloads.
			"--file-cache-max-size-mb=48 --file-cache-enable-parallel-downloads=true --file-cache-parallel-downloads-per-file=4 --file-cache-max-parallel-downloads=2 --file-cache-download-chunk-size-mb=4 --file-cache-enable-crc=true --cache-dir=/gcsfuse-tmp/TestJobChunkTestWithParallelDownloads --log-file=/gcsfuse-tmp/TestJobChunkTestWithParallelDownloads.log --log-severity=TRACE --enable-kernel-reader=false",
			"--file-cache-max-size-mb=48 --file-cache-enable-parallel-downloads=true --file-cache-parallel-downloads-per-file=4 --file-cache-max-parallel-downloads=2 --file-cache-download-chunk-size-mb=4 --file-cache-enable-crc=true --cache-dir=/gcsfuse-tmp/TestJobChunkTestWithParallelDownloads --log-file=/gcsfuse-tmp/TestJobChunkTestWithParallelDownloads.log --log-severity=TRACE --client-protocol=grpc --enable-kernel-reader=false",
		}
		cfg.ReadCache[0].Configs[13].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.ReadCache[0].Configs[13].Run = "TestJobChunkTestWithParallelDownloads"

		cfg.ReadCache[0].Configs[14].Flags = []string{
			"--file-cache-exclude-regex=. --file-cache-max-size-mb=50 --file-cache-enable-parallel-downloads=false --cache-dir=/gcsfuse-tmp/TestCacheFileForExcludeRegexTest --log-file=/gcsfuse-tmp/TestCacheFileForExcludeRegexTest.log --log-severity=TRACE --enable-kernel-reader=false",
			"--file-cache-exclude-regex=. --file-cache-max-size-mb=50 --file-cache-cache-file-for-range-read=true --file-cache-enable-parallel-downloads=false --cache-dir=/gcsfuse-tmp/TestCacheFileForExcludeRegexTest --log-file=/gcsfuse-tmp/TestCacheFileForExcludeRegexTest.log --log-severity=TRACE --enable-kernel-reader=false",
			"--file-cache-exclude-regex=^" + setup.TestBucket() + "/ --file-cache-max-size-mb=50 --file-cache-cache-file-for-range-read=true --file-cache-enable-parallel-downloads=false --cache-dir=/gcsfuse-tmp/TestCacheFileForExcludeRegexTest --log-file=/gcsfuse-tmp/TestCacheFileForExcludeRegexTest.log --log-severity=TRACE --enable-kernel-reader=false",
			"--file-cache-exclude-regex=. --file-cache-max-size-mb=50 --file-cache-enable-parallel-downloads=false --cache-dir=/gcsfuse-tmp/TestCacheFileForExcludeRegexTest --log-file=/gcsfuse-tmp/TestCacheFileForExcludeRegexTest.log --log-severity=TRACE --client-protocol=grpc --enable-kernel-reader=false",
			"--file-cache-exclude-regex=. --file-cache-max-size-mb=50 --file-cache-cache-file-for-range-read=true --file-cache-enable-parallel-downloads=false --cache-dir=/gcsfuse-tmp/TestCacheFileForExcludeRegexTest --log-file=/gcsfuse-tmp/TestCacheFileForExcludeRegexTest.log --log-severity=TRACE --client-protocol=grpc --enable-kernel-reader=false",
			"--file-cache-exclude-regex=^" + setup.TestBucket() + "/ --file-cache-max-size-mb=50 --file-cache-cache-file-for-range-read=true --file-cache-enable-parallel-downloads=false --cache-dir=/gcsfuse-tmp/TestCacheFileForExcludeRegexTest --log-file=/gcsfuse-tmp/TestCacheFileForExcludeRegexTest.log --log-severity=TRACE --client-protocol=grpc --enable-kernel-reader=false",
		}
		cfg.ReadCache[0].Configs[14].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.ReadCache[0].Configs[14].Run = "TestCacheFileForExcludeRegexTest"

		cfg.ReadCache[0].Configs[15].Flags = []string{
			"--implicit-dirs --file-cache-max-size-mb=9 --file-cache-enable-parallel-downloads=false --cache-dir=/gcsfuse-tmp/TestRemountTest --log-file=/gcsfuse-tmp/TestRemountTest.log --log-severity=TRACE --enable-kernel-reader=false",
			"--implicit-dirs --file-cache-max-size-mb=9 --file-cache-enable-parallel-downloads=true --cache-dir=/gcsfuse-tmp/TestRemountTest --log-file=/gcsfuse-tmp/TestRemountTest.log --log-severity=TRACE --enable-kernel-reader=false",
			"--implicit-dirs --file-cache-max-size-mb=9 --file-cache-enable-parallel-downloads=false --cache-dir=/gcsfuse-tmp/TestRemountTest --log-file=/gcsfuse-tmp/TestRemountTest.log --log-severity=TRACE --client-protocol=grpc --enable-kernel-reader=false",
			"--implicit-dirs --file-cache-max-size-mb=9 --file-cache-enable-parallel-downloads=true --cache-dir=/gcsfuse-tmp/TestRemountTest --log-file=/gcsfuse-tmp/TestRemountTest.log --log-severity=TRACE --client-protocol=grpc --enable-kernel-reader=false",
		}
		cfg.ReadCache[0].Configs[15].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.ReadCache[0].Configs[15].Run = "TestRemountTest"

		cfg.ReadCache[0].Configs[16].Flags = []string{
			"--file-cache-include-regex=^" + setup.TestBucket() + "/.*ReadCacheTest/foo* --file-cache-exclude-regex= --file-cache-max-size-mb=9 --cache-dir=/gcsfuse-tmp/TestCacheFileForIncludeRegexTest --log-file=/gcsfuse-tmp/TestCacheFileForIncludeRegexTest.log --log-severity=TRACE --enable-kernel-reader=false",
			"--file-cache-include-regex=^" + setup.TestBucket() + "/.*ReadCacheTest/foo* --file-cache-exclude-regex= --file-cache-max-size-mb=9 --cache-dir=/gcsfuse-tmp/TestCacheFileForIncludeRegexTest --log-file=/gcsfuse-tmp/TestCacheFileForIncludeRegexTest.log --log-severity=TRACE --client-protocol=grpc --enable-kernel-reader=false",
			"--file-cache-include-regex=^" + setup.TestBucket() + "/.*ReadCacheTest/foo* --file-cache-exclude-regex=invalid --file-cache-max-size-mb=9 --cache-dir=/gcsfuse-tmp/TestCacheFileForIncludeRegexTest --log-file=/gcsfuse-tmp/TestCacheFileForIncludeRegexTest.log --log-severity=TRACE --enable-kernel-reader=false",
			"--file-cache-include-regex=^" + setup.TestBucket() + "/.*ReadCacheTest/foo* --file-cache-exclude-regex=invalid --file-cache-max-size-mb=9 --cache-dir=/gcsfuse-tmp/TestCacheFileForIncludeRegexTest --log-file=/gcsfuse-tmp/TestCacheFileForIncludeRegexTest.log --log-severity=TRACE --client-protocol=grpc --enable-kernel-reader=false",
		}
		cfg.ReadCache[0].Configs[16].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.ReadCache[0].Configs[16].Run = "TestCacheFileForIncludeRegexTest"

		cfg.ReadCache[0].Configs[17].Flags = []string{
			"--file-cache-experimental-enable-chunk-cache=true --file-cache-download-chunk-size-mb=10 --cache-dir=/gcsfuse-tmp/TestChunkCacheTest --log-file=/gcsfuse-tmp/TestChunkCacheTest.log --log-severity=TRACE --enable-kernel-reader=false",
		}
		cfg.ReadCache[0].Configs[17].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.ReadCache[0].Configs[17].Run = "TestChunkCacheTest"

		cfg.ReadCache[0].Configs[18].Flags = []string{
			"--file-cache-experimental-enable-chunk-cache=false --cache-dir=/gcsfuse-tmp/TestChunkCacheDisabledTest --log-file=/gcsfuse-tmp/TestChunkCacheDisabledTest.log --log-severity=TRACE --enable-kernel-reader=false",
		}
		cfg.ReadCache[0].Configs[18].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.ReadCache[0].Configs[18].Run = "TestChunkCacheDisabledTest"

		cfg.ReadCache[0].Configs[19].Flags = []string{
			"--file-cache-experimental-enable-chunk-cache=true --file-cache-download-chunk-size-mb=10 --file-cache-max-size-mb=15 --cache-dir=/gcsfuse-tmp/TestChunkCacheEviction --log-file=/gcsfuse-tmp/TestChunkCacheEviction.log --log-severity=TRACE --enable-kernel-reader=false",
		}
		cfg.ReadCache[0].Configs[19].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.ReadCache[0].Configs[19].Run = "TestChunkCacheEviction"
	}

	testEnv.ctx = context.Background()
	testEnv.bucketType = setup.TestEnvironment(testEnv.ctx, &cfg.ReadCache[0])
	testEnv.cfg = &cfg.ReadCache[0]

	// 2. Create storage client before running tests.
	var err error
	testEnv.storageClient, err = client.CreateStorageClient(testEnv.ctx)
	if err != nil {
		log.Printf("Error creating storage client: %v\n", err)
		os.Exit(1)
	}
	defer testEnv.storageClient.Close()

	// 3. To run mountedDirectory tests, we need both testBucket and mountedDirectory
	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		os.Exit(setup.RunTestsForMountedDirectory(testEnv.cfg.GKEMountedDirectory, m))
	}

	// Run tests for testBucket
	// Set up test directory.
	setup.SetUpTestDirForTestBucket(testEnv.cfg)
	// Override GKE specific paths with GCSFuse paths if running in GCE environment.
	overrideFilePathsInFlagSet(testEnv.cfg, setup.TestDir())

	// Save mount and root directory variables.
	mountDir, rootDir = setup.MntDir(), setup.MntDir()

	log.Println("Running static mounting tests...")
	mountFunc = static_mounting.MountGcsfuseWithStaticMountingWithConfigFile
	successCode := m.Run()

	if successCode == 0 {
		log.Println("Running dynamic mounting tests...")
		// Save mount directory variable to have path of bucket to run tests.
		mountDir = path.Join(setup.MntDir(), setup.TestBucket())
		mountFunc = dynamic_mounting.MountGcsfuseWithDynamicMountingWithConfig
		successCode = m.Run()
	}

	if successCode == 0 {
		log.Println("Running only dir mounting tests...")
		setup.SetOnlyDirMounted(onlyDirMounted + "/")
		mountDir = rootDir
		mountFunc = only_dir_mounting.MountGcsfuseWithOnlyDirWithConfigFile
		successCode = m.Run()
		setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(setup.TestBucket(), setup.OnlyDirMounted(), testDirName))
	}

	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)
}

func overrideFilePathsInFlagSet(t *test_suite.TestConfig, GCSFuseTempDirPath string) {
	for _, flags := range t.Configs {
		for i := range flags.Flags {
			// Iterate over the indices of the flags slice
			flags.Flags[i] = strings.ReplaceAll(flags.Flags[i], "/gcsfuse-tmp", path.Join(GCSFuseTempDirPath, "gcsfuse-tmp"))
		}
	}
}
