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

package unfinalized_object

import (
	"context"
	"log"
	"os"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

const (
	testDirName = "UnfinalizedObjectTest"
)

var (
	testEnv   env
	mountFunc func(*test_suite.TestConfig, []string) error
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
	if len(cfg.UnfinalizedObject) == 0 {
		log.Println("No configuration found for unfinalized_object tests in config. Using flags instead.")
		// Populate the config manually.
		cfg.UnfinalizedObject = make([]test_suite.TestConfig, 1)
		cfg.UnfinalizedObject[0].TestBucket = setup.TestBucket()
		cfg.UnfinalizedObject[0].GKEMountedDirectory = setup.MountedDirectory()
		cfg.UnfinalizedObject[0].LogFile = setup.LogFile()
		cfg.UnfinalizedObject[0].Configs = make([]test_suite.ConfigItem, 2)
		cfg.UnfinalizedObject[0].Configs[0].Flags = []string{
			"--metadata-cache-ttl-secs=-1",
		}
		cfg.UnfinalizedObject[0].Configs[0].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.UnfinalizedObject[0].Configs[0].Run = "TestUnfinalizedObjectReadTest"
		cfg.UnfinalizedObject[0].Configs[1].Flags = []string{
			"--metadata-cache-ttl-secs=0",
		}
		cfg.UnfinalizedObject[0].Configs[1].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.UnfinalizedObject[0].Configs[1].Run = "TestUnfinalizedObjectOperationTest"
	}

	testEnv.ctx = context.Background()
	testEnv.cfg = &cfg.UnfinalizedObject[0]
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
	mountFunc = static_mounting.MountGcsfuseWithStaticMountingWithConfigFile
	successCode := m.Run()

	os.Exit(successCode)
}
