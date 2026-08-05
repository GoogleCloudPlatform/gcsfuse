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

package dummy_test_package

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
	testDirName    = "DummyIntegrationTestDir"
	onlyDirMounted = "OnlyDirMountDummyTest"
)

type env struct {
	mountFunc            func(*test_suite.TestConfig, []string) error
	mountDir             string
	rootDir              string
	storageClient        *storage.Client
	storageControlClient *control.StorageControlClient
	ctx                  context.Context
	testDirPath          string
	cfg                  *test_suite.TestConfig
	bucketType           string
}

var testEnv env

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// 1. Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.DummyTestPackage) == 0 {
		log.Fatalf("No configuration found for dummy_test_package in config path: %s", setup.ConfigFile())
	}

	testEnv.ctx = context.Background()
	testEnv.bucketType = setup.TestEnvironment(testEnv.ctx, &cfg.DummyTestPackage[0])
	testEnv.cfg = &cfg.DummyTestPackage[0]

	// 2. Build common storage clients.
	// NOTE: Most test packages only require the standard storage client.
	// Only build and initialize the storageControlClient (using client.CreateControlClientWithCancel)
	// if your test suite explicitly runs control-plane operations (e.g., Folder operations or Zonal RAPID class configurations).
	// Otherwise, omit the control client setup entirely to keep the test environment lightweight.
	closeStorageClient := client.CreateStorageClientWithCancel(&testEnv.ctx, &testEnv.storageClient)
	defer func() {
		if err := closeStorageClient(); err != nil {
			log.Printf("closeStorageClient failed: %v\n", err)
		}
	}()

	// OPTIONAL: Omit if control operations are not tested.
	closeStorageControlClient := client.CreateControlClientWithCancel(&testEnv.ctx, &testEnv.storageControlClient)
	defer func() {
		if err := closeStorageControlClient(); err != nil {
			log.Printf("closeStorageControlClient failed: %v\n", err)
		}
	}()

	// 3. Short-circuit FIRST for GKE mounted directory execution.
	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		testEnv.mountDir, testEnv.rootDir = testEnv.cfg.GKEMountedDirectory, testEnv.cfg.GKEMountedDirectory
		os.Exit(setup.RunTestsForMountedDirectory(testEnv.cfg.GKEMountedDirectory, m))
	}

	// 4. Set up local directories and targets (applying flag file path overrides).
	setup.SetUpTestDirForTestBucket(testEnv.cfg)
	setup.OverrideFilePathsInFlagSet(testEnv.cfg, setup.TestDir())
	testEnv.mountDir, testEnv.rootDir = testEnv.cfg.GCSFuseMountedDirectory, testEnv.cfg.GCSFuseMountedDirectory

	// 5. Run static mounting tests.
	log.Println("Running static mounting tests...")
	testEnv.mountFunc = static_mounting.MountGcsfuseWithStaticMountingWithConfigFile
	successCode := m.Run()

	// 6. Run dynamic mounting tests if static succeeds.
	if successCode == 0 {
		log.Println("Running dynamic mounting tests...")
		testEnv.mountDir = path.Join(testEnv.cfg.GCSFuseMountedDirectory, testEnv.cfg.TestBucket)
		testEnv.mountFunc = dynamic_mounting.MountGcsfuseWithDynamicMountingWithConfig
		successCode = m.Run()
	}

	// 7. Run only dir mounting tests if dynamic succeeds.
	if successCode == 0 {
		log.Println("Running only dir mounting tests...")
		setup.SetOnlyDirMounted(onlyDirMounted + "/")
		testEnv.mountDir = testEnv.rootDir
		testEnv.mountFunc = only_dir_mounting.MountGcsfuseWithOnlyDirWithConfigFile
		successCode = m.Run()
		setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(testEnv.cfg.TestBucket, setup.OnlyDirMounted(), testDirName))
	}

	// 8. Systematic bucket cleanup.
	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(testEnv.cfg.TestBucket, testDirName))
	os.Exit(successCode)
}
