// Copyright 2023 Google LLC
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

// Provide test for listing large directory
package list_large_dir

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
	"gopkg.in/yaml.v3"
)

const prefixFileInDirectoryWithTwelveThousandFiles = "fileInDirectoryWithTwelveThousandFiles"
const prefixExplicitDirInLargeDirListTest = "explicitDirInLargeDirListTest"
const prefixImplicitDirInLargeDirListTest = "implicitDirInLargeDirListTest"
const numberOfFilesInDirectoryWithTwelveThousandFiles = 12000
const numberOfImplicitDirsInDirectoryWithTwelveThousandFiles = 100
const numberOfExplicitDirsInDirectoryWithTwelveThousandFiles = 100

var (
	directoryWithTwelveThousandFiles = "directoryWithTwelveThousandFiles" + setup.GenerateRandomString(5)
	storageClient                    *storage.Client
	ctx                              context.Context
)

// Config holds all test configurations parsed from the YAML file.
type Config struct {
	ListLargeDir []test_suite.TestConfig `yaml:"list_large_dir"`
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
	if len(cfg.ListLargeDir) == 0 {
		log.Println("No configuration found for list large dir tests in config. Using flags instead.")
		// Populate the config manually.
		cfg.ListLargeDir = make([]test_suite.TestConfig, 1)
		cfg.ListLargeDir[0].TestBucket = setup.TestBucket()
		cfg.ListLargeDir[0].MountedDirectory = setup.MountedDirectory()
		cfg.ListLargeDir[0].Configs = make([]test_suite.ConfigItem, 2)
		cfg.ListLargeDir[0].Configs[0].Flags = []string{"--implicit-dirs=true --stat-cache-ttl=0 --kernel-list-cache-ttl-secs=-1"}
		cfg.ListLargeDir[0].Configs[0].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.ListLargeDir[0].Configs[1].Flags = []string{
			"--client-protocol=grpc --implicit-dirs=true --stat-cache-ttl=0 --kernel-list-cache-ttl-secs=-1",
		}
		cfg.ListLargeDir[0].Configs[1].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": false}
	}

	// 2. Create storage client before running tests.
	setup.SetTestBucket(cfg.ListLargeDir[0].TestBucket)
	ctx = context.Background()
	closeStorageClient := client.CreateStorageClientWithCancel(&ctx, &storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	// 3. To run mountedDirectory tests, we need both testBucket and mountedDirectory
	// flags to be set, as ListLargeDir tests validates content from the bucket.
	if cfg.ListLargeDir[0].MountedDirectory != "" && cfg.ListLargeDir[0].TestBucket != "" {
		os.Exit(setup.RunTestsForMountedDirectory(cfg.ListLargeDir[0].MountedDirectory, m))
	}

	// Run tests for testBucket
	// 4. Build the flag sets dynamically from the config.
	bucketType, err := setup.BucketType(ctx, cfg.ListLargeDir[0].TestBucket)
	if err != nil {
		log.Fatalf("BucketType failed: %v", err)
	}
	if bucketType == setup.ZonalBucket {
		setup.SetIsZonalBucketRun(true)
	}
	flags := setup.BuildFlagSets(cfg.ListLargeDir[0], bucketType)

	setup.SetUpTestDirForTestBucket(cfg.ListLargeDir[0].TestBucket)

	successCode := static_mounting.RunTestsWithConfigFile(&cfg.ListLargeDir[0], flags, m)

	os.Exit(successCode)
}
