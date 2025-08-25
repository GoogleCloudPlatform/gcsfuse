// Copyright 2023 Google LLC
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

// Provide tests when implicit directory present and mounted bucket with --implicit-dir flag.
package implicit_dir_test

import (
	"context"
	"log"
	"os"
	"path"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup/implicit_and_explicit_dir_setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
	"gopkg.in/yaml.v3"
)

const ExplicitDirInImplicitDir = "explicitDirInImplicitDir"
const ExplicitDirInImplicitSubDir = "explicitDirInImplicitSubDir"
const PrefixFileInExplicitDirInImplicitDir = "fileInExplicitDirInImplicitDir"
const PrefixFileInExplicitDirInImplicitSubDir = "fileInExplicitDirInImplicitSubDir"
const NumberOfFilesInExplicitDirInImplicitSubDir = 1
const NumberOfFilesInExplicitDirInImplicitDir = 1
const DirForImplicitDirTests = "dirForImplicitDirTests"

// IMPORTANT: To prevent global variable pollution, enhance code clarity,
// and avoid inadvertent errors. We strongly suggest that, all new package-level
// variables (which would otherwise be declared with `var` at the package root) should
// be added as fields to this 'env' struct instead.
type env struct {
	storageClient *storage.Client
	ctx           context.Context
	testDirPath   string
}

var testEnv env

func setupTestDir(dirName string) string {
	dir := setup.SetupTestDirectory(DirForImplicitDirTests)
	dirPath := path.Join(dir, dirName)
	err := os.Mkdir(dirPath, setup.DirPermission_0755)
	if err != nil {
		log.Fatalf("Error while setting up directory %s for testing: %v", dirPath, err)
	}

	return dirPath
}

// Config holds all test configurations parsed from the YAML file.
type Config struct {
	ImplicitDir []test_suite.TestConfig `yaml:"implicit_dir"`
}

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// 1. Load and parse the common configuration.
	var cfg Config
	if setup.ConfigFile() != "" {
		configData, err := os.ReadFile(setup.ConfigFile())
		if err != nil {
			log.Fatalf("could not read test_config.yaml: %v", err)
		}
		expandedYaml := os.ExpandEnv(string(configData))
		if err := yaml.Unmarshal([]byte(expandedYaml), &cfg); err != nil {
			log.Fatalf("Failed to parse config YAML: %v", err)
		}
	}
	if len(cfg.ImplicitDir) == 0 {
		log.Println("No configuration found for implicit_dir tests in config. Using flags instead.")
		// Populate the config manually.
		cfg.ImplicitDir = make([]test_suite.TestConfig, 1)
		cfg.ImplicitDir[0].TestBucket = setup.TestBucket()
		cfg.ImplicitDir[0].MountedDirectory = setup.MountedDirectory()
		cfg.ImplicitDir[0].Configs = make([]test_suite.ConfigItem, 2)
		cfg.ImplicitDir[0].Configs[0].Flags = []string{"--implicit-dirs"}
		cfg.ImplicitDir[0].Configs[0].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.ImplicitDir[0].Configs[1].Flags = []string{"--implicit-dirs --client-protocol=grpc"}
		cfg.ImplicitDir[0].Configs[1].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": false}
	}

	// 2. Create storage client before running tests.
	setup.SetTestBucket(cfg.ImplicitDir[0].TestBucket)
	testEnv.ctx = context.Background()
	closeStorageClient := client.CreateStorageClientWithCancel(&testEnv.ctx, &testEnv.storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	// 4. Build the flag sets dynamically from the config.
	bucketType, err := setup.BucketType(testEnv.ctx, cfg.ImplicitDir[0].TestBucket)
	if err != nil {
		log.Fatalf("BucketType failed: %v", err)
	}
	if bucketType == setup.ZonalBucket {
		setup.SetIsZonalBucketRun(true)
	}
	flags := setup.BuildFlagSets(cfg.ImplicitDir[0], bucketType)

	// 5. Run tests with the dynamically generated flags.
	successCode := implicit_and_explicit_dir_setup.RunTestsForExplicitAndImplicitDir(&cfg.ImplicitDir[0], flags, m)
	setup.SaveLogFileInCaseOfFailure(successCode)

	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)
}
