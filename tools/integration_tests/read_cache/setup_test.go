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
	logFileNameForMountedDirectoryTests    = "/tmp/gcsfuse_read_cache_test_logs/log.json"
	parallelDownloadsPerFile               = 4
	maxParallelDownloads                   = -1
	downloadChunkSizeMB                    = 4
	enableCrcCheck                         = true
	http1ClientProtocol                    = "http1"
	grpcClientProtocol                     = "grpc"
	GKETempDir                             = "/gcsfuse-tmp"
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
		cfg.ReadCache[0].Configs = make([]test_suite.ConfigItem, 1)
		cfg.ReadCache[0].Configs[0].Flags = []string{
			"--implicit-dirs --metadata-cache-ttl-secs=10 --file-cache-max-size-mb=9 --file-cache-cache-file-for-range-read=false --file-cache-enable-parallel-downloads=false --file-cache-enable-o-direct=false --cache-dir=/gcsfuse-tmp/TestSmallCacheTTLTest --log-file=/gcsfuse-tmp/TestSmallCacheTTLTest.log",
			"--metadata-cache-ttl-secs=10 --file-cache-max-size-mb=9 --file-cache-cache-file-for-range-read=false --file-cache-enable-parallel-downloads=true --file-cache-enable-o-direct=false -cache-dir=/gcsfuse-tmp/TestSmallCacheTTLTest --log-file=/gcsfuse-tmp/TestSmallCacheTTLTest.log",
			"--implicit-dirs --metadata-cache-ttl-secs=10 --file-cache-max-size-mb=9 --file-cache-cache-file-for-range-read=false --file-cache-enable-parallel-downloads=false --file-cache-enable-o-direct=false --cache-dir=/gcsfuse-tmp/TestSmallCacheTTLTest --log-file=/gcsfuse-tmp/TestSmallCacheTTLTest.log --client-protocol=grpc",
			"--metadata-cache-ttl-secs=10 --file-cache-max-size-mb=9 --file-cache-cache-file-for-range-read=false --file-cache-enable-parallel-downloads=true --file-cache-enable-o-direct=false -cache-dir=/gcsfuse-tmp/TestSmallCacheTTLTest --log-file=/gcsfuse-tmp/TestSmallCacheTTLTest.log --client-protocol=grpc",
		}
		cfg.ReadCache[0].Configs[0].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.ReadCache[0].Configs[0].Run = "TestSmallCacheTTLTest"
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
