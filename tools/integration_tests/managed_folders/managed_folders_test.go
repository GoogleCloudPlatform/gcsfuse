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

// Provides integration tests for managed folders.
package managed_folders

import (
	"context"
	"log"
	"os"
	"path"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	control "cloud.google.com/go/storage/control/apiv2"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/creds_tests"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/only_dir_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/dynamic_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

type env struct {
	ctx           context.Context
	storageClient *storage.Client
	controlClient *control.StorageControlClient
	bucketType    string
	cfg           *test_suite.TestConfig
	mountFunc     func(*test_suite.TestConfig, []string) error
	// Mount directory is where our tests run.
	mountDir string
	// Root directory is the directory to be unmounted.
	rootDir          string
	serviceAccount   string
	localKeyFilePath string
	bucket           string
	testDir          string
	testDirPath      string
}

const (
	onlyDirMounted = "TestManagedFolderOnlyDir"
)

var testEnv env

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// 1. Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.ManagedFolders) == 0 {
		log.Println("No configuration found for managed_folders tests in config. Using flags instead.")
		// Populate the config manually.
		cfg.ManagedFolders = make([]test_suite.TestConfig, 1)
		cfg.ManagedFolders[0].TestBucket = setup.TestBucket()
		cfg.ManagedFolders[0].GKEMountedDirectory = setup.MountedDirectory()
		cfg.ManagedFolders[0].LogFile = setup.LogFile()
		// Initialize the slice to hold 15 specific test configurations
		cfg.ManagedFolders[0].Configs = make([]test_suite.ConfigItem, 3)
		cfg.ManagedFolders[0].Configs[0].Flags = []string{"--implicit-dirs --key-file=${KEY_FILE} --rename-dir-limit=3"}
		cfg.ManagedFolders[0].Configs[0].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.ManagedFolders[0].Configs[0].Run = "TestManagedFolders_FolderViewPermission"
		cfg.ManagedFolders[0].Configs[1].Flags = []string{"--implicit-dirs --enable-empty-managed-folders"}
		cfg.ManagedFolders[0].Configs[1].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.ManagedFolders[0].Configs[1].Run = "TestEnableEmptyManagedFoldersTrue"
		cfg.ManagedFolders[0].Configs[2].Flags = []string{"--implicit-dirs --key-file=${KEY_FILE} --rename-dir-limit=5 --stat-cache-ttl=0"}
		cfg.ManagedFolders[0].Configs[2].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.ManagedFolders[0].Configs[2].Run = "TestManagedFolders_FolderAdminPermission"
	}

	testEnv.ctx = context.Background()
	testEnv.bucketType = setup.TestEnvironment(testEnv.ctx, &cfg.ManagedFolders[0])
	testEnv.cfg = &cfg.ManagedFolders[0]

	// 2. Create storage client before running tests.
	closeStorageClient := client.CreateStorageClientWithCancel(&testEnv.ctx, &testEnv.storageClient)
	closeControlClient := client.CreateControlClientWithCancel(&testEnv.ctx, &testEnv.controlClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Printf("closeStorageClient failed: %v\n", err)
		}
		err = closeControlClient()
		if err != nil {
			log.Printf("closeControlClient failed: %v\n", err)
		}
	}()

	// Fetch credentials and apply permission on bucket.
	testEnv.serviceAccount, testEnv.localKeyFilePath = creds_tests.CreateCredentials(testEnv.ctx)
	defer func() {
		if err := os.Remove(testEnv.localKeyFilePath); err != nil {
			log.Printf("Failed to delete temp credentials file %s: %v", testEnv.localKeyFilePath, err)
		}
	}()

	for i, testCase := range cfg.ManagedFolders[0].Configs {
		// Replace the placeholder with the actual key file path.
		cfg.ManagedFolders[0].Configs[i].Flags[0] = strings.ReplaceAll(testCase.Flags[0], "${KEY_FILE}", testEnv.localKeyFilePath)
	}

	// 3. these tests won't run with mountedDirectory.
	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		log.Printf("These tests will not run with mounted directory..")
		return
	}

	// Run tests for testBucket
	// Set up test directory.
	setup.SetUpTestDirForTestBucket(testEnv.cfg)

	// Save mount and root directory variables.
	testEnv.mountDir, testEnv.rootDir = testEnv.cfg.GCSFuseMountedDirectory, testEnv.cfg.GCSFuseMountedDirectory

	log.Println("Running static mounting tests...")
	testEnv.mountFunc = static_mounting.MountGcsfuseWithStaticMountingWithConfigFile
	successCode := m.Run()

	if successCode == 0 {
		log.Println("Running dynamic mounting tests...")
		// Save mount directory variable to have path of bucket to run tests.
		testEnv.mountDir = path.Join(testEnv.cfg.GCSFuseMountedDirectory, testEnv.cfg.TestBucket)
		testEnv.mountFunc = dynamic_mounting.MountGcsfuseWithDynamicMountingWithConfig
		successCode = m.Run()
	}

	if successCode == 0 {
		log.Println("Running only dir mounting tests...")
		setup.SetOnlyDirMounted(onlyDirMounted + "/")
		testEnv.mountDir = testEnv.rootDir
		testEnv.mountFunc = only_dir_mounting.MountGcsfuseWithOnlyDirWithConfigFile
		successCode = m.Run()
		setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(testEnv.cfg.TestBucket, setup.OnlyDirMounted(), TestDirForManagedFolderTest))
	}

	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(testEnv.cfg.TestBucket, TestDirForManagedFolderTest))
	os.Exit(successCode)
}
