// Copyright 2023 Google Inc. All Rights Reserved.
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

// Provides integration tests for file and directory operations.

package local_file_test

import (
	"context"
	"log"
	"os"
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/config"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/local_file/helpers"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting/dynamic_mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting/only_dir_mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

// testDirPath holds the path to the test subdirectory in the mounted bucket.
var testDirPath string

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	helpers.Ctx = context.Background()
	var cancel context.CancelFunc
	var err error

	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	if setup.MountedDirectory() != "" && setup.TestBucket() == "" {
		log.Print("Both --testbucket and --mountedDirectory should be specified to run mounted directory tests.")
		os.Exit(1)
	}

	// Create storage client before running tests.
	helpers.Ctx, cancel = context.WithTimeout(helpers.Ctx, time.Minute*15)
	helpers.StorageClient, err = client.CreateStorageClient(helpers.Ctx)
	if err != nil {
		log.Fatalf("client.CreateStorageClient: %v", err)
	}

	// Run tests for mountedDirectory only if --mountedDirectory flag is set.
	setup.RunTestsForMountedDirectoryFlag(m)

	// Else run tests for testBucket.
	// Set up test directory.
	setup.SetUpTestDirForTestBucketFlag()
	// Set up config file with create-empty-file: false. (default)
	mountConfig := config.MountConfig{
		WriteConfig: config.WriteConfig{
			CreateEmptyFile: false,
		},
		LogConfig: config.LogConfig{
			Severity: config.TRACE,
		},
	}
	configFile := setup.YAMLConfigFile(mountConfig)
	// Set up flags to run tests on.
	flags := [][]string{
		{"--implicit-dirs=true", "--rename-dir-limit=3", "--config-file=" + configFile},
		{"--implicit-dirs=false", "--rename-dir-limit=3", "--config-file=" + configFile}}

	successCode := static_mounting.RunTests(flags, m)

	if successCode == 0 {
		successCode = only_dir_mounting.RunTests(flags, m)
	}

	if successCode == 0 {
		successCode = dynamic_mounting.RunTests(flags, m)
	}

	// Close storage client and release resources.
	helpers.StorageClient.Close()
	cancel()
	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(path.Join(setup.TestBucket(), helpers.LocalFileTestDirInBucket))
	setup.RemoveBinFileCopiedForTesting()
	os.Exit(successCode)
}
