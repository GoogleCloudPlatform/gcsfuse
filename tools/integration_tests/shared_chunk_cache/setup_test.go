// Copyright 2026 Google LLC
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

package shared_chunk_cache

import (
	"context"
	"log"
	"os"
	"path"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

const (
	testDirName = "SharedChunkCacheTest"
	GKETempDir  = "/gcsfuse-tmp"
)

var (
	testEnv env
)

type env struct {
	storageClient *storage.Client
	ctx           context.Context
	cfg           *test_suite.TestConfig
	bucketType    string
}

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// 1. Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.SharedChunkCache) == 0 {
		log.Println("No configuration found for shared_chunk_cache tests in config. Using flags instead.")
		// Populate the config manually.
		cfg.SharedChunkCache = make([]test_suite.TestConfig, 1)
		cfg.SharedChunkCache[0].TestBucket = setup.TestBucket()
		cfg.SharedChunkCache[0].GKEMountedDirectory = setup.MountedDirectory()
		cfg.SharedChunkCache[0].LogFile = setup.LogFile()
		cfg.SharedChunkCache[0].Configs = make([]test_suite.ConfigItem, 1)

		// TestSharedChunkCacheTestSuite - dual mount with shared cache
		cfg.SharedChunkCache[0].Configs[0].Flags = []string{
			"--enable-experimental-shared-chunk-cache --file-cache-max-size-mb=-1 --cache-dir=/gcsfuse-tmp/shared-cache",
		}
		cfg.SharedChunkCache[0].Configs[0].SecondaryFlags = []string{
			"--enable-experimental-shared-chunk-cache --file-cache-max-size-mb=-1 --cache-dir=/gcsfuse-tmp/shared-cache",
		}
		cfg.SharedChunkCache[0].Configs[0].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.SharedChunkCache[0].Configs[0].Run = "TestSharedChunkCacheTestSuite"
	}

	testEnv.ctx = context.Background()
	testEnv.cfg = &cfg.SharedChunkCache[0]
	testEnv.bucketType = setup.TestEnvironment(testEnv.ctx, testEnv.cfg)

	// 2. Create storage client before running tests.
	var err error
	testEnv.storageClient, err = client.CreateStorageClient(testEnv.ctx)
	if err != nil {
		log.Printf("Error creating storage client: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := testEnv.storageClient.Close(); err != nil {
			log.Printf("Error closing storage client: %v\n", err)
		}
	}()

	// 3. To run mountedDirectory tests, we need both testBucket and mountedDirectory
	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		// For GKE, we expect both directories to be mounted if it's a dual mount test.
		// If using config, GKEMountedDirectorySecondary should be set.
		testEnv.cfg.GCSFuseMountedDirectory = testEnv.cfg.GKEMountedDirectory
		testEnv.cfg.GCSFuseMountedDirectorySecondary = testEnv.cfg.GKEMountedDirectorySecondary
		os.Exit(m.Run())
	}

	// For GCE environment
	setup.SetUpTestDirForTestBucket(testEnv.cfg)
	// Override GKE specific paths with GCSFuse paths if running in GCE environment.
	setup.OverrideFilePathsInFlagSet(testEnv.cfg, setup.TestDir())

	// For dual mount, we create another directory.
	secondaryDir, err := os.MkdirTemp(setup.TestDir(), "gcsfuse-secondary-mount")
	if err != nil {
		log.Fatalf("Failed to create secondary mount directory: %v", err)
	}
	testEnv.cfg.GCSFuseMountedDirectorySecondary = secondaryDir

	successCode := m.Run()

	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)
}
