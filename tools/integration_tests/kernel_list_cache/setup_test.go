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

package kernel_list_cache

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/dynamic_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/only_dir_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

const (
	testDirNamePrefix = "KernelListCacheTest"
	onlyDirMounted    = "OnlyDirMountKernelListCache"
	GKETempDir        = "/gcsfuse-tmp"
)

// You would need to add imports for "fmt" and "os"
var testDirName = fmt.Sprintf("%s-%d-%s", testDirNamePrefix, os.Getpid(), setup.GenerateRandomString(5))

var (
	mountFunc func(*test_suite.TestConfig, []string) error
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
}

var testEnv env

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// 1. Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.KernelListCache) == 0 {
		log.Println("No configuration found for kernel_list_cache tests in config. Using flags instead.")
		// Populate the config manually.
		cfg.KernelListCache = make([]test_suite.TestConfig, 1)
		cfg.KernelListCache[0].TestBucket = setup.TestBucket()
		cfg.KernelListCache[0].GKEMountedDirectory = setup.MountedDirectory()
		cfg.KernelListCache[0].LogFile = setup.LogFile()
		// Initialize the slice to hold 15 specific test configurations
		cfg.KernelListCache[0].Configs = make([]test_suite.ConfigItem, 4)
		cfg.KernelListCache[0].Configs[0].Flags = []string{"--kernel-list-cache-ttl-secs=-1"}
		cfg.KernelListCache[0].Configs[0].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.KernelListCache[0].Configs[0].Run = "TestInfiniteKernelListCacheTest"
		// Note: metadata cache is disabled to avoid cache consistency issue between
		// gcsfuse cache and kernel cache. As gcsfuse cache might hold the entry which
		// already became stale due to delete operation.
		cfg.KernelListCache[0].Configs[1].Flags = []string{"--kernel-list-cache-ttl-secs=-1 --metadata-cache-ttl-secs=0 --metadata-cache-negative-ttl-secs=0"}
		cfg.KernelListCache[0].Configs[1].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.KernelListCache[0].Configs[1].Run = "TestInfiniteKernelListCacheDeleteDirTest"
		cfg.KernelListCache[0].Configs[2].Flags = []string{"--kernel-list-cache-ttl-secs=5 --rename-dir-limit=10"}
		cfg.KernelListCache[0].Configs[2].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.KernelListCache[0].Configs[2].Run = "TestFiniteKernelListCacheTest"
		cfg.KernelListCache[0].Configs[3].Flags = []string{"--kernel-list-cache-ttl-secs=0 --stat-cache-ttl=0 --rename-dir-limit=10"}
		cfg.KernelListCache[0].Configs[3].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.KernelListCache[0].Configs[3].Run = "TestDisabledKernelListCacheTest"
	}

	testEnv.ctx = context.Background()
	testEnv.bucketType = setup.TestEnvironment(testEnv.ctx, &cfg.KernelListCache[0])
	testEnv.cfg = &cfg.KernelListCache[0]

	// 2. Create storage client before running tests.
	closeStorageClient := client.CreateStorageClientWithCancel(&testEnv.ctx, &testEnv.storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Printf("closeStorageClient failed: %v\n", err)
		}
	}()

	// 3. To run mountedDirectory tests, we need both testBucket and mountedDirectory
	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		// Save mount and root directory variables.
		mountDir, rootDir = testEnv.cfg.GKEMountedDirectory, testEnv.cfg.GKEMountedDirectory
		os.Exit(setup.RunTestsForMountedDirectory(testEnv.cfg.GKEMountedDirectory, m))
	}

	// Run tests for testBucket
	// Set up test directory.
	setup.SetUpTestDirForTestBucket(testEnv.cfg)
	// Override GKE specific paths with GCSFuse paths if running in GCE environment.
	overrideFilePathsInFlagSet(testEnv.cfg, setup.TestDir())

	// Save mount and root directory variables.
	mountDir, rootDir = testEnv.cfg.GCSFuseMountedDirectory, testEnv.cfg.GCSFuseMountedDirectory

	log.Println("Running static mounting tests...")
	mountFunc = static_mounting.MountGcsfuseWithStaticMountingWithConfigFile
	successCode := m.Run()

	if successCode == 0 {
		log.Println("Running dynamic mounting tests...")
		// Save mount directory variable to have path of bucket to run tests.
		mountDir = path.Join(testEnv.cfg.GCSFuseMountedDirectory, testEnv.cfg.TestBucket)
		mountFunc = dynamic_mounting.MountGcsfuseWithDynamicMountingWithConfig
		successCode = m.Run()
	}

	if successCode == 0 {
		log.Println("Running only dir mounting tests...")
		setup.SetOnlyDirMounted(onlyDirMounted + "/")
		mountDir = rootDir
		mountFunc = only_dir_mounting.MountGcsfuseWithOnlyDirWithConfigFile
		successCode = m.Run()
		setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(testEnv.cfg.TestBucket, setup.OnlyDirMounted(), testDirName))
	}

	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(testEnv.cfg.TestBucket, testDirName))
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
