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

package buffered_read

import (
	"context"
	"log"
	"os"
	"path"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

const (
	testDirName  = "BufferedReadTest"
	testFileName = "foo"
	// Global block size constant for tests
	blockSizeInBytes = int64(8 * util.MiB)
	GKETempDir       = "/gcsfuse-tmp"
)

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

	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.BufferedRead) == 0 {
		log.Fatal("No configuration found for BufferedRead in config file.")
	}

	testEnv.ctx = context.Background()
	testEnv.bucketType = setup.TestEnvironment(testEnv.ctx, &cfg.BufferedRead[0])
	testEnv.cfg = &cfg.BufferedRead[0]

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
	setup.OverrideFilePathsInFlagSet(testEnv.cfg, setup.TestDir())

	// Save mount and root directory variables.
	mountDir, rootDir = testEnv.cfg.GCSFuseMountedDirectory, testEnv.cfg.GCSFuseMountedDirectory

	log.Println("Running static mounting tests...")
	mountFunc = static_mounting.MountGcsfuseWithStaticMountingWithConfigFile
	successCode := m.Run()

	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(testEnv.cfg.TestBucket, testDirName))
	os.Exit(successCode)
}
