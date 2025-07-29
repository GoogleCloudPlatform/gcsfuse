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
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup/implicit_and_explicit_dir_setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
	"gopkg.in/yaml.v2"
)

const DirForExplicitDirTests = "dirForExplicitDirTests"

// Config holds all test configurations parsed from the YAML file.
type Config struct {
	ExplicitDir []test_suite.TestConfig `yaml:"explicit_dir"`
}

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
	configData, err := os.ReadFile(setup.ConfigFile())
	if err != nil {
		log.Fatalf("could not read test_config.yaml: %v", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(configData, &cfg); err != nil {
		log.Fatalf("Failed to parse config YAML: %v", err)
	}
	if len(cfg.ExplicitDir) == 0 {
		log.Fatal("No configuration found for explicit_dir tests in config.")
	}
	// Assuming one primary configuration for this test package.
	testCfg := cfg.ExplicitDir[0]

	// 2. Create storage client.
	testEnv.ctx = context.Background()
	closeStorageClient := client.CreateStorageClientWithCancel(&testEnv.ctx, &testEnv.storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	// These tests will not run on HNS buckets because the "--implicit-dirs=false" flag does not function similarly to how it does on FLAT buckets.
	// Note that HNS buckets do not have the concept of implicit directories.
	if setup.IsHierarchicalBucket(testEnv.ctx, testEnv.storageClient) {
		log.Println("These tests will not run on HNS buckets.")
		return
	}

	// 4. Build the flag sets dynamically from the config.
	var flagSets [][]string
	for _, flagString := range testCfg.Flags {
		flagSets = append(flagSets, strings.Fields(flagString))
	}

	if len(flagSets) == 0 {
		log.Println("No applicable test flags found for the current configuration.")
		os.Exit(0)
	}

	// 5. Run tests with the dynamically generated flags.
	successCode := implicit_and_explicit_dir_setup.RunTestsForImplicitDirAndExplicitDir(testCfg, flagSets, m)
	os.Exit(successCode)
}
