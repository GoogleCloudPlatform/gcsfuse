// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package stale_handle

import (
	"context"
	"log"
	"os"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

const (
	testDirName = "StaleHandleTest"
)

var (
	testEnv env
)

type env struct {
	storageClient *storage.Client
	ctx           context.Context
	testDirPath   string
	cfg           *test_suite.TestConfig
	bucketType    string
}

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// 1. Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.StaleHandle) == 0 {
		log.Println("No configuration found for stale_handle tests in config. Using flags instead.")
		// Populate the config manually.
		cfg.StaleHandle = make([]test_suite.TestConfig, 1)
		cfg.StaleHandle[0].TestBucket = setup.TestBucket()
		cfg.StaleHandle[0].GKEMountedDirectory = setup.MountedDirectory()
		cfg.StaleHandle[0].Configs = make([]test_suite.ConfigItem, 2)
		cfg.StaleHandle[0].Configs[0].Flags = []string{
			"--metadata-cache-ttl-secs=0 --enable-streaming-writes=false",
		}
		cfg.StaleHandle[0].Configs[0].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.StaleHandle[0].Configs[0].Run = "TestStaleFileHandleLocalFileTest"
		cfg.StaleHandle[0].Configs[1].Flags = []string{
			"--metadata-cache-ttl-secs=0 --write-block-size-mb=1 --write-max-blocks-per-file=1",
		}
		cfg.StaleHandle[0].Configs[1].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.StaleHandle[0].Configs[1].Run = "TestStaleFileHandleLocalFileTest"
	}

	// Run all tests with GRPC.
	for i := range cfg.StaleHandle[0].Configs {
		// Create a temporary slice of slices to satisfy the function signature.
		tempFlags := [][]string{cfg.StaleHandle[0].Configs[i].Flags}
		setup.AppendFlagsToAllFlagsInTheFlagsSet(&tempFlags, "--client-protocol=grpc", "")
		cfg.StaleHandle[0].Configs[i].Flags = tempFlags[0]
	}

	testEnv.cfg = &cfg.StaleHandle[0]
	testEnv.ctx = context.Background()
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
		os.Exit(setup.RunTestsForMountedDirectory(testEnv.cfg.GKEMountedDirectory, m))
	}

	// Run tests for testBucket
	// Set up test directory.
	setup.SetUpTestDirForTestBucket(testEnv.cfg)

	log.Println("Running static mounting tests...")
	successCode := m.Run()

	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, testDirName)
	os.Exit(successCode)
}