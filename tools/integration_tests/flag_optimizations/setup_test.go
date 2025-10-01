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

// Provide tests for cases where bucket is mounted with flag(s) --machine-type and/or --profile.
package flag_optimizations

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
	testDirName = "FlagOptimizationsTests"
)

var (
	logFileNameForMountedDirectoryTests = path.Join(os.TempDir(), "gcsfuse_flag_optimizations_logs", "log.json")
)

// IMPORTANT: To prevent global variable pollution, enhance code clarity,
// and avoid inadvertent errors. We strongly suggest that, all new package-level
// variables (which would otherwise be declared with `var` at the package root) should
// be added as fields to this 'env' struct instead.
type env struct {
	testDirPath string
	mountFunc   func([]string) error
	// mount directory is where our tests run.
	mountDir string
	// root directory is the directory to be unmounted.
	rootDir       string
	storageClient *storage.Client
	ctx           context.Context
}

var testEnv env

var (
	// Taken from gcsfuse/cfg/params.yaml .
	highEndMachines = []string{
		"a2-megagpu-16g",
		"a2-ultragpu-8g",
		"a3-edgegpu-8g",
		"a3-highgpu-8g",
		"a3-megagpu-8g",
		"a3-ultragpu-8g",
		"a4-highgpu-8g-lowmem",
		"ct5l-hightpu-8t",
		"ct5lp-hightpu-8t",
		"ct5p-hightpu-4t",
		"ct5p-hightpu-4t-tpu",
		"ct6e-standard-4t",
		"ct6e-standard-4t-tpu",
		"ct6e-standard-8t",
		"ct6e-standard-8t-tpu",
	}
	supportedAIMLProfiles = []string{
		"aiml-training",
		"aiml-checkpointing",
		"aiml-serving",
	}
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func setupForMountedDirectoryTests() {
	if setup.MountedDirectory() != "" {
		testEnv.mountDir = setup.MountedDirectory()
		setup.SetLogFile(logFileNameForMountedDirectoryTests)
	}
}

func staticMountFunc(flags []string) error {
	config := &test_suite.TestConfig{
		TestBucket:              setup.TestBucket(),
		GKEMountedDirectory:     setup.MountedDirectory(),
		GCSFuseMountedDirectory: setup.MntDir(),
		LogFile:                 setup.LogFile(),
	}
	return static_mounting.MountGcsfuseWithStaticMountingWithConfigFile(config, flags)
}

func mountGCSFuseAndSetupTestDir(flags []string, ctx context.Context, storageClient *storage.Client) {
	setup.MountGCSFuseWithGivenMountFunc(flags, testEnv.mountFunc)
	setup.SetMntDir(testEnv.mountDir)
	testEnv.testDirPath = client.SetupTestDirectory(ctx, storageClient, testDirName)
}

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	testEnv.ctx = context.Background()
	closeStorageClient := client.CreateStorageClientWithCancel(&testEnv.ctx, &testEnv.storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	setup.RunTestsForMountedDirectory(setup.MountedDirectory(), m)

	// Else run tests for testBucket.
	// Set up test directory.
	setup.SetUpTestDirForTestBucketFlag()

	// Save mount and root directory variables.
	testEnv.mountDir, testEnv.rootDir = setup.MntDir(), setup.MntDir()

	log.Println("Running static mounting tests...")
	testEnv.mountFunc = staticMountFunc
	successCode := m.Run()

	// If failed, then save the gcsfuse log file(s).
	setup.SaveLogFileInCaseOfFailure(successCode)

	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)
}
