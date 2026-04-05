// Copyright 2025 Google LLC
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

package negative_stat_cache

import (
	"context"
	"log"
	"os"
	"path"
	"testing"

	"cloud.google.com/go/storage"
	control "cloud.google.com/go/storage/control/apiv2"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/dynamic_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/only_dir_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

const (
	testDirName    = "NegativeStatCacheTest"
	onlyDirMounted = "OnlyDirMountNegativeStatCache"
)

var (
	mountFunc func(*test_suite.TestConfig, []string) error
	// mount directory is where our tests run.
	mountDir string
	// root directory is the directory to be unmounted.
	rootDir string
)

type env struct {
	storageClient        *storage.Client
	storageControlClient *control.StorageControlClient
	ctx                  context.Context
	testDirPath          string
	cfg                  *test_suite.TestConfig
	bucketType           string
}

var testEnv env

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// 1. Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.NegativeStatCache) == 0 {
		log.Println("No configuration found for negative_stat_cache tests in config. Using flags instead.")
		// Populate the config manually.
		cfg.NegativeStatCache = make([]test_suite.TestConfig, 1)
		cfg.NegativeStatCache[0].TestBucket = setup.TestBucket()
		cfg.NegativeStatCache[0].GKEMountedDirectory = setup.MountedDirectory()
		cfg.NegativeStatCache[0].LogFile = setup.LogFile()
		// Initialize the slice to hold specific test configurations
		cfg.NegativeStatCache[0].Configs = make([]test_suite.ConfigItem, 3)
		cfg.NegativeStatCache[0].Configs[0].Flags = []string{"--metadata-cache-negative-ttl-secs=0"}
		cfg.NegativeStatCache[0].Configs[0].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.NegativeStatCache[0].Configs[0].Run = "TestDisabledNegativeStatCacheTest"
		cfg.NegativeStatCache[0].Configs[1].Flags = []string{"--metadata-cache-negative-ttl-secs=5"}
		cfg.NegativeStatCache[0].Configs[1].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.NegativeStatCache[0].Configs[1].Run = "TestFiniteNegativeStatCacheTest"
		cfg.NegativeStatCache[0].Configs[2].Flags = []string{"--metadata-cache-negative-ttl-secs=-1"}
		cfg.NegativeStatCache[0].Configs[2].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.NegativeStatCache[0].Configs[2].Run = "TestInfiniteNegativeStatCacheTest"
	}

	testEnv.ctx = context.Background()
	testEnv.bucketType = setup.TestEnvironment(testEnv.ctx, &cfg.NegativeStatCache[0])
	testEnv.cfg = &cfg.NegativeStatCache[0]

	// Create common storage client to be used in test.
	closeStorageClient := client.CreateStorageClientWithCancel(&testEnv.ctx, &testEnv.storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Printf("closeStorageClient failed: %v\n", err)
		}
	}()
	closeStorageControlClient := client.CreateControlClientWithCancel(&testEnv.ctx, &testEnv.storageControlClient)
	defer func() {
		err := closeStorageControlClient()
		if err != nil {
			log.Printf("closeStorageControlClient failed: %v\n", err)
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
