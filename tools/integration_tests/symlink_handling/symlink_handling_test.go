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

package symlink_handling_test

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
	TestDirName                = "SymlinkHandlingTest"
	SymlinkMetadataKey         = "gcsfuse_symlink_target"
	StandardSymlinkMetadataKey = "goog-reserved-file-is-symlink"
)

var (
	testEnv env
)

type env struct {
	storageClient *storage.Client
	ctx           context.Context
	cfg           *test_suite.TestConfig
}

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// 1. Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.SymlinkHandling) == 0 {
		log.Println("No configuration found for symlink handling tests in config. Using flags instead.")
		// Populate the config manually.
		cfg.SymlinkHandling = make([]test_suite.TestConfig, 1)
		cfg.SymlinkHandling[0].TestBucket = setup.TestBucket()
		cfg.SymlinkHandling[0].GKEMountedDirectory = setup.MountedDirectory()
		cfg.SymlinkHandling[0].LogFile = setup.LogFile()
		cfg.SymlinkHandling[0].Configs = make([]test_suite.ConfigItem, 2)

		// 1. TestStandardSymlinksTestSuite
		cfg.SymlinkHandling[0].Configs[0].Flags = []string{"--experimental-enable-standard-symlinks=true"}
		cfg.SymlinkHandling[0].Configs[0].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.SymlinkHandling[0].Configs[0].Run = "TestStandardSymlinksTestSuite"

		// 2. TestLegacySymlinksTestSuite
		cfg.SymlinkHandling[0].Configs[1].Flags = []string{"--experimental-enable-standard-symlinks=false"}
		cfg.SymlinkHandling[0].Configs[1].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.SymlinkHandling[0].Configs[1].Run = "TestLegacySymlinksTestSuite"
	}

	testEnv.ctx = context.Background()
	testEnv.cfg = &cfg.SymlinkHandling[0]

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
		// If using config, GKEMountedDirectorySecondary should be set.
		testEnv.cfg.GCSFuseMountedDirectory = testEnv.cfg.GKEMountedDirectory
		os.Exit(m.Run())
	}

	// For GCE environment
	setup.SetUpTestDirForTestBucket(testEnv.cfg)
	// Override GKE specific paths with GCSFuse paths if running in GCE environment.
	setup.OverrideFilePathsInFlagSet(testEnv.cfg, setup.TestDir())

	log.Println("Running static mounting tests for symlink handling...")
	successCode := m.Run()

	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(setup.TestBucket(), TestDirName))
	os.Exit(successCode)
}
