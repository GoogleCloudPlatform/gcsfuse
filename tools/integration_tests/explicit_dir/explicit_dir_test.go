// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Provide tests for explicit directory only.
package explicit_dir_test

import (
	"context"
	"log"
	"os"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup/implicit_and_explicit_dir_setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

const DirForExplicitDirTests = "dirForExplicitDirTests"

// IMPORTANT: To prevent global variable pollution, enhance code clarity,
// and avoid inadvertent errors. We strongly suggest that, all new package-level
// variables (which would otherwise be declared with `var` at the package root) should
// be added as fields to this 'env' struct instead.
type env struct {
	storageClient *storage.Client
	ctx           context.Context
}

var testEnv env

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// 1. Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.ExplicitDir) == 0 {
		log.Println("No configuration found for explicit_dir tests in config. Using flags instead.")
		// Populate the config manually.
		cfg.ExplicitDir = make([]test_suite.TestConfig, 1)
		cfg.ExplicitDir[0].TestBucket = setup.TestBucket()
		cfg.ExplicitDir[0].MountedDirectory = setup.MountedDirectory()
		cfg.ExplicitDir[0].Configs = make([]test_suite.ConfigItem, 1)
		cfg.ExplicitDir[0].Configs[0].Flags = []string{"--implicit-dirs=false", "--implicit-dirs=false --client-protocol=grpc"}
		cfg.ExplicitDir[0].Configs[0].Compatible = map[string]bool{"flat": true, "hns": false, "zonal": false}
	}

	// 2. Create storage client before running tests.
	testEnv.ctx = context.Background()
	bucketType := setup.BucketTestEnvironment(testEnv.ctx, cfg.ExplicitDir[0].TestBucket)
	closeStorageClient := client.CreateStorageClientWithCancel(&testEnv.ctx, &testEnv.storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	// 3. Build the flag sets dynamically from the config.
	flags := setup.BuildFlagSets(cfg.ExplicitDir[0], bucketType)

	// 4. Run tests with the dynamically generated flags.
	successCode := implicit_and_explicit_dir_setup.RunTestsForExplicitAndImplicitDir(&cfg.ExplicitDir[0], flags, m)
	os.Exit(successCode)
}
