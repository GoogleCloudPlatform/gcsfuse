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

package readonly_creds

import (
	"context"
	"log"
	"os"
	"path"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/creds_tests"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

const (
	testDirName           = "ReadonlyCredsTest"
	testFileName          = "fileName.txt"
	content               = "write content."
	permissionDeniedError = "permission denied"
)

var (
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
	if len(cfg.ReadonlyCreds) == 0 {
		log.Println("No configuration found for readonly_creds tests in config. Using flags instead.")
		cfg.ReadonlyCreds = make([]test_suite.TestConfig, 1)
		cfg.ReadonlyCreds[0].TestBucket = setup.TestBucket()
		cfg.ReadonlyCreds[0].GKEMountedDirectory = setup.MountedDirectory()
		cfg.ReadonlyCreds[0].Configs = make([]test_suite.ConfigItem, 1)
		cfg.ReadonlyCreds[0].Configs[0].Flags = []string{"--implicit-dirs=true", "--implicit-dirs=false"}
		cfg.ReadonlyCreds[0].Configs[0].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
	}

	testEnv.ctx = context.Background()
	testEnv.bucketType = setup.TestEnvironment(testEnv.ctx, &cfg.ReadonlyCreds[0])
	testEnv.cfg = &cfg.ReadonlyCreds[0]

	// 2. Create storage client before running tests.
	closeStorageClient := client.CreateStorageClientWithCancel(&testEnv.ctx, &testEnv.storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Printf("closeStorageClient failed: %v\n", err)
		}
	}()

	// 3. Skip for GKE or mounted directory tests.
	if cfg.ReadonlyCreds[0].GKEMountedDirectory != "" {
		log.Print("These tests will not run for mountedDirectory flag.")
		os.Exit(0)
	}

	// Create test directory on GCS before dropping privileges.
	client.SetupTestDirectory(testEnv.ctx, testEnv.storageClient, testDirName)

	// Run tests for testBucket
	// Set up test directory.
	setup.SetUpTestDirForTestBucket(testEnv.cfg)
	// Override GKE specific paths with GCSFuse paths if running in GCE environment.
	setup.OverrideFilePathsInFlagSet(testEnv.cfg, setup.TestDir())

	// Save mount and root directory variables.
	mountDir, rootDir = testEnv.cfg.GCSFuseMountedDirectory, testEnv.cfg.GCSFuseMountedDirectory

	flags := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, "")
	// Test for viewer permission on test bucket.
	successCode := creds_tests.RunTestsForDifferentAuthMethods(testEnv.ctx, testEnv.cfg, testEnv.storageClient, flags, "objectViewer", m)

	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(testEnv.cfg.TestBucket, testDirName))
	os.Exit(successCode)
}
