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
)

const (
	prefixFileInDirectoryWithTwelveThousandFiles           = "fileInDirectoryWithTwelveThousandFiles"
	prefixExplicitDirInLargeDirListTest                    = "explicitDirInLargeDirListTest"
	prefixImplicitDirInLargeDirListTest                    = "implicitDirInLargeDirListTest"
	numberOfFilesInDirectoryWithTwelveThousandFiles        = 12000
	numberOfImplicitDirsInDirectoryWithTwelveThousandFiles = 100
	numberOfExplicitDirsInDirectoryWithTwelveThousandFiles = 100
)

var (
	directoryWithTwelveThousandFiles = "directoryWithTwelveThousandFiles" + setup.GenerateRandomString(5)
	mountFunc                        func(*test_suite.TestConfig, []string) error
)

type env struct {
	storageClient *storage.Client
	ctx           context.Context
	bucketType    string
	cfg           *test_suite.TestConfig
}

var testEnv env

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// 1. Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.ListLargeDir) == 0 {
		log.Fatal("No configuration found for ListLargeDir in config file.")
	}

	// 2. Create storage client before running tests.
	testEnv.ctx = context.Background()
	testEnv.cfg = &cfg.ListLargeDir[0]
	testEnv.bucketType = setup.TestEnvironment(testEnv.ctx, &cfg.ListLargeDir[0])
	closeStorageClient := client.CreateStorageClientWithCancel(&testEnv.ctx, &testEnv.storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	// 3. To run mountedDirectory tests, we need both testBucket and mountedDirectory
	// flags to be set, as ListLargeDir tests validates content from the bucket.
	if cfg.ListLargeDir[0].GKEMountedDirectory != "" && cfg.ListLargeDir[0].TestBucket != "" {
		os.Exit(setup.RunTestsForMountedDirectory(cfg.ListLargeDir[0].GKEMountedDirectory, m))
	}

	// Run tests for testBucket
	// 4. Build the flag sets dynamically from the config.
	setup.SetUpTestDirForTestBucket(&cfg.ListLargeDir[0])
	log.Println("Running static mounting tests...")
	mountFunc = static_mounting.MountGcsfuseWithStaticMountingWithConfigFile
	successCode := m.Run()

	os.Exit(successCode)
}
