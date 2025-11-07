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
package unsupported_path

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

const DirForUnsupportedPathTests = "dirForUnsupportedPathTests"

var (
	storageClient *storage.Client
	ctx           context.Context
)

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// 1. Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.UnsupportedPath) == 0 {
		log.Println("No configuration found for unsupported path tests in config. Using flags instead.")
		// Populate the config manually.
		cfg.UnsupportedPath = []test_suite.TestConfig{
			{
				TestBucket:          setup.TestBucket(),
				GKEMountedDirectory: setup.MountedDirectory(),
				Configs: []test_suite.ConfigItem{
					{
						Flags: []string{
							"--implicit-dirs --client-protocol=grpc --enable-unsupported-path-support=true --rename-dir-limit=200",
							"--implicit-dirs --enable-unsupported-path-support=true --rename-dir-limit=200",
						},
						Compatible: map[string]bool{"flat": true, "hns": true, "zonal": false},
					},
					{
						Flags: []string{
							"--implicit-dirs --enable-unsupported-path-support=true --rename-dir-limit=200",
						},
						Compatible: map[string]bool{"flat": false, "hns": false, "zonal": true},
					},
				},
			},
		}
	}

	ctx = context.Background()
	bucketType := setup.TestEnvironment(ctx, &cfg.UnsupportedPath[0])

	// 2. Create storage client before running tests.
	var err error
	storageClient, err = client.CreateStorageClient(ctx)
	if err != nil {
		log.Printf("Error creating storage client: %v\n", err)
		os.Exit(1)
	}
	defer storageClient.Close()

	// 3. To run mountedDirectory tests, we need both testBucket and mountedDirectory
	if cfg.UnsupportedPath[0].GKEMountedDirectory != "" && cfg.UnsupportedPath[0].TestBucket != "" {
		os.Exit(setup.RunTestsForMountedDirectory(cfg.UnsupportedPath[0].GKEMountedDirectory, m))
	}

	// Run tests for testBucket
	// 4. Build the flag sets dynamically from the config.
	flags := setup.BuildFlagSets(cfg.UnsupportedPath[0], bucketType, "")
	setup.SetUpTestDirForTestBucket(&cfg.UnsupportedPath[0])

	successCode := static_mounting.RunTestsWithConfigFile(&cfg.UnsupportedPath[0], flags, m)

	os.Exit(successCode)
}
