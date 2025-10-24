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

package interrupt

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
	testDirName = "InterruptTest"
)

var (
	storageClient *storage.Client
	ctx           context.Context
)

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// 1. Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.Interrupt) == 0 {
		log.Println("No configuration found for interrupt tests in config. Using flags instead.")
		// Populate the config manually.
		cfg.Interrupt = make([]test_suite.TestConfig, 1)
		cfg.Interrupt[0].TestBucket = setup.TestBucket()
		cfg.Interrupt[0].GKEMountedDirectory = setup.MountedDirectory()
		cfg.Interrupt[0].Configs = make([]test_suite.ConfigItem, 2)
		cfg.Interrupt[0].Configs[0].Flags = []string{
			"--implicit-dirs=true --enable-streaming-writes=false",
			"--ignore-interrupts=true --enable-streaming-writes=false",
			"--ignore-interrupts=false --enable-streaming-writes=false",
		}
		cfg.Interrupt[0].Configs[0].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.Interrupt[0].Configs[1].Flags = []string{
			"--enable-streaming-writes=true",
		}
		cfg.Interrupt[0].Configs[1].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": false}
	}

	// 2. Create storage client before running tests.
	ctx = context.Background()
	bucketType := setup.TestEnvironment(ctx, &cfg.Interrupt[0])
	closeStorageClient := client.CreateStorageClientWithCancel(&ctx, &storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	// 3. To run mountedDirectory tests, we need both testBucket and mountedDirectory
	// flags to be set, as Interrupt tests validates content from the bucket.
	if cfg.Interrupt[0].GKEMountedDirectory != "" && cfg.Interrupt[0].TestBucket != "" {
		os.Exit(setup.RunTestsForMountedDirectory(cfg.Interrupt[0].GKEMountedDirectory, m))
	}

	// Run tests for testBucket
	// 4. Build the flag sets dynamically from the config.
	flags := setup.BuildFlagSets(cfg.Interrupt[0], bucketType, "")

	setup.SetUpTestDirForTestBucket(&cfg.Interrupt[0])

	successCode := static_mounting.RunTestsWithConfigFile(&cfg.Interrupt[0], flags, m)

	setup.CleanupDirectoryOnGCS(ctx, storageClient, path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)

}
