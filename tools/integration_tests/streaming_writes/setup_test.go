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

package streaming_writes

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
	testDirName = "StreamingWritesTest"
)

type env struct {
	storageClient *storage.Client
	ctx           context.Context
	testDirPath   string
	cfg           test_suite.TestConfig
}

var testEnv env

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// 1. Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.StreamingWrites) == 0 {
		log.Fatal("No configuration found for StreamingWrites in config file.")
	}

	testEnv.ctx = context.Background()
	bucketType := setup.TestEnvironment(testEnv.ctx, &cfg.StreamingWrites[0])
	testEnv.cfg = cfg.StreamingWrites[0]

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
	// 4. Build the flag sets dynamically from the config.
	flagsSet := setup.BuildFlagSets(testEnv.cfg, bucketType, "")
	setup.SetUpTestDirForTestBucket(&testEnv.cfg)

	successCode := static_mounting.RunTestsWithConfigFile(&testEnv.cfg, flagsSet, m)
	setup.SaveLogFileInCaseOfFailure(successCode)
	os.Exit(successCode)
}
