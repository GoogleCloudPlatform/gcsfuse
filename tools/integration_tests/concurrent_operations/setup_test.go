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

package concurrent_operations

import (
	"context"
	"log"
	"os"
	"path"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

const (
	testDirName    = "TestConcurrentOperations"
	GKETempDir    = "/gcsfuse-tmp"
	// // TODO: clean this up when GKE test migration completes.
	OldGKElogFilePath = "/tmp/ConcurrentOperations_logs/log.json"
)

var (
	testEnv   env
	mountFunc func(*test_suite.TestConfig, []string) error
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

func mountGCSFuseAndSetupTestDir(flags []string, ctx context.Context, storageClient *storage.Client) {
	setup.MountGCSFuseWithGivenMountWithConfigFunc(testEnv.cfg, flags, mountFunc)
	setup.SetMntDir(mountDir)
	testEnv.testDirPath = client.SetupTestDirectory(ctx, storageClient, testDirName)
}

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()
		// 1. Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.ConcurrentOperations) == 0 {
		log.Println("No configuration found for concurrent operations tests in config. Using flags instead.")
		// Populate the config manually.
		cfg.ConcurrentOperations = make([]test_suite.TestConfig, 1)
		cfg.ConcurrentOperations[0].TestBucket = setup.TestBucket()
		cfg.ConcurrentOperations[0].GKEMountedDirectory = setup.MountedDirectory()
		cfg.ConcurrentOperations[0].LogFile = setup.LogFile()
		cfg.ConcurrentOperations[0].Configs = make([]test_suite.ConfigItem, 2)
		cfg.ConcurrentOperations[0].Configs[0].Flags = []string{
			"",
			"--enable-buffered-read",
		}
		cfg.ConcurrentOperations[0].Configs[0].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.ConcurrentOperations[0].Configs[0].Run = "TestConcurrentRead"

		cfg.ConcurrentOperations[0].Configs[1].Flags = []string{
			"--kernel-list-cache-ttl-secs=0",
			"--kernel-list-cache-ttl-secs=0 --client-protocol=grpc",
			"--kernel-list-cache-ttl-secs=1",
			"--kernel-list-cache-ttl-secs=1 --client-protocol=grpc",
		}
		cfg.ConcurrentOperations[0].Configs[1].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.ConcurrentOperations[0].Configs[1].Run = "TestConcurrentListing"
	}

	testEnv.ctx = context.Background()
	testEnv.cfg = &cfg.ConcurrentOperations[0]
	testEnv.bucketType = setup.TestEnvironment(testEnv.ctx, testEnv.cfg)

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
		mountDir = testEnv.cfg.GKEMountedDirectory
		os.Exit(setup.RunTestsForMountedDirectory(mountDir, m))
	}

	// Run tests for testBucket
	// Set up test directory.
	setup.SetUpTestDirForTestBucket(testEnv.cfg)

	// Save mount and root directory variables.
	mountDir, rootDir = setup.MntDir(), setup.MntDir()

	log.Println("Running static mounting tests...")
	mountFunc = static_mounting.MountGcsfuseWithStaticMountingWithConfigFile
	successCode := m.Run()

	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(cfg.ConcurrentOperations[0].TestBucket, testDirName))
	os.Exit(successCode)
}
