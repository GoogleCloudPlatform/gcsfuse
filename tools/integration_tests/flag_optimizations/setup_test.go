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
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/dynamic_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/only_dir_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

const (
	testDirName    = "FlagOptimizationsTests"
	onlyDirMounted = "OnlyDirMountFlagOptimizations"
	GKETempDir     = "/gcsfuse-tmp"
)

// To prevent global variable pollution, enhance code clarity,
// and avoid inadvertent errors. We strongly suggest that, all new package-level
// variables (which would otherwise be declared with `var` at the package root) should
// be added as fields to this 'env' struct instead.
type env struct {
	testDirPath string
	mountFunc   func(*test_suite.TestConfig, []string) error
	// mount directory is where our tests run.
	mountDir string
	// root directory is the directory to be mounted/unmounted.
	rootDir       string
	storageClient *storage.Client
	ctx           context.Context
	bucketType    string
	cfg           test_suite.TestConfig
}

var (
	testEnv env
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func mountGCSFuseAndSetupTestDir(flags []string, ctx context.Context, storageClient *storage.Client) error {
	err := setup.MayMountGCSFuseWithGivenMountWithConfigFunc(&testEnv.cfg, flags, testEnv.mountFunc)
	if err != nil {
		return err
	}
	if testEnv.cfg.GKEMountedDirectory == "" {
		setup.SetMntDir(testEnv.mountDir)
	}
	testEnv.testDirPath = client.SetupTestDirectory(ctx, storageClient, testDirName)
	return nil
}

func mustMountGCSFuseAndSetupTestDir(flags []string, ctx context.Context, storageClient *storage.Client) {
	if err := mountGCSFuseAndSetupTestDir(flags, ctx, storageClient); err != nil {
		panic(err)
	}
}

func overrideFilePathsInFlagSet(t *test_suite.TestConfig, GCSFuseTempDirPath string) {
	for _, flags := range t.Configs {
		for i := range flags.Flags {
			// Iterate over the indices of the flags slice
			flags.Flags[i] = strings.ReplaceAll(flags.Flags[i], "/gcsfuse-tmp", path.Join(GCSFuseTempDirPath, "gcsfuse-tmp"))
		}
	}
}

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// Load and parse the common configuration.
	cfg := test_suite.ReadConfigFile(setup.ConfigFile())
	if len(cfg.FlagOptimizations) == 0 {
		log.Println("No configuration found for flag_optimizations tests in config. Using flags instead.")
		// Populate the config manually.
		cfg.FlagOptimizations = make([]test_suite.TestConfig, 1)
		cfg.FlagOptimizations[0].TestBucket = setup.TestBucket()
		cfg.FlagOptimizations[0].GKEMountedDirectory = setup.MountedDirectory()
		cfg.FlagOptimizations[0].LogFile = setup.LogFile()
		// Initialize the slice to hold 7 specific test configurations
		cfg.FlagOptimizations[0].Configs = make([]test_suite.ConfigItem, 7)
		cfg.FlagOptimizations[0].Configs[0].Run = "TestMountFails"
		cfg.FlagOptimizations[0].Configs[0].Flags = []string{"--profile=unknown-profile"}
		cfg.FlagOptimizations[0].Configs[0].Compatible = map[string]bool{"flat": true, "hns": true, "zonal": true}
		cfg.FlagOptimizations[0].Configs[0].RunOnGKE = false
		cfg.FlagOptimizations[0].Configs[1].Run = "TestImplicitDirsNotEnabled"
		cfg.FlagOptimizations[0].Configs[1].Flags = []string{"--machine-type=low-end-machine"}
		cfg.FlagOptimizations[0].Configs[1].Compatible = map[string]bool{"flat": true, "hns": false, "zonal": false}
		cfg.FlagOptimizations[0].Configs[1].RunOnGKE = true
		cfg.FlagOptimizations[0].Configs[2].Run = "TestRenameDirLimitNotSet"
		cfg.FlagOptimizations[0].Configs[2].Flags = []string{"--machine-type=low-end-machine", "--profile=aiml-training", "--profile=aiml-serving"}
		cfg.FlagOptimizations[0].Configs[2].Compatible = map[string]bool{"flat": true, "hns": false, "zonal": false}
		cfg.FlagOptimizations[0].Configs[2].RunOnGKE = true
		cfg.FlagOptimizations[0].Configs[3].Run = "TestImplicitDirsEnabled"
		cfg.FlagOptimizations[0].Configs[3].Flags = []string{
			"--machine-type=a3-highgpu-8g",
			"--profile=aiml-training",
			"--profile=aiml-serving",
			"--profile=aiml-checkpointing",
			"--machine-type=low-end-machine --profile=aiml-training",
			"--machine-type=low-end-machine --profile=aiml-serving",
			"--machine-type=low-end-machine --profile=aiml-checkpointing",
		}
		cfg.FlagOptimizations[0].Configs[3].Compatible = map[string]bool{"flat": true, "hns": false, "zonal": false}
		cfg.FlagOptimizations[0].Configs[3].RunOnGKE = true
		cfg.FlagOptimizations[0].Configs[4].Run = "TestRenameDirLimitSet"
		cfg.FlagOptimizations[0].Configs[4].Flags = []string{
			"--machine-type=a3-highgpu-8g",
			"--profile=aiml-checkpointing",
			"--machine-type=low-end-machine --profile=aiml-checkpointing",
		}
		cfg.FlagOptimizations[0].Configs[4].Compatible = map[string]bool{"flat": true, "hns": false, "zonal": false}
		cfg.FlagOptimizations[0].Configs[4].RunOnGKE = true
		cfg.FlagOptimizations[0].Configs[5].Run = "TestZonalBucketOptimizations_LogVerification"
		cfg.FlagOptimizations[0].Configs[5].Flags = []string{"--log-severity=trace"}
		cfg.FlagOptimizations[0].Configs[5].Compatible = map[string]bool{"flat": false, "hns": false, "zonal": true}
		cfg.FlagOptimizations[0].Configs[5].RunOnGKE = false
		cfg.FlagOptimizations[0].Configs[6].Run = "TestZonalBucketOptimizations_KernelParamVerification"
		cfg.FlagOptimizations[0].Configs[6].Flags = []string{"--log-severity=trace"}
		cfg.FlagOptimizations[0].Configs[6].Compatible = map[string]bool{"flat": false, "hns": false, "zonal": true}
		cfg.FlagOptimizations[0].Configs[6].RunOnGKE = false
	}

	testEnv.ctx = context.Background()
	testEnv.bucketType = setup.TestEnvironment(testEnv.ctx, &cfg.FlagOptimizations[0])
	testEnv.cfg = cfg.FlagOptimizations[0]
	closeStorageClient := client.CreateStorageClientWithCancel(&testEnv.ctx, &testEnv.storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	// To run mountedDirectory tests, we need both testBucket and mountedDirectory
	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		os.Exit(setup.RunTestsForMountedDirectory(testEnv.cfg.GKEMountedDirectory, m))
	}

	// Run tests for testBucket
	// Set up test directory.
	setup.SetUpTestDirForTestBucket(&testEnv.cfg)
	// Override GKE specific paths with GCSFuse paths if running in GCE environment.
	overrideFilePathsInFlagSet(&testEnv.cfg, setup.TestDir())

	// Save mount and root directory variables.
	testEnv.mountDir, testEnv.rootDir = setup.MntDir(), setup.MntDir()

	log.Println("Running static mounting tests...")
	testEnv.mountFunc = static_mounting.MountGcsfuseWithStaticMountingWithConfigFile
	successCode := m.Run()

	if successCode == 0 {
		log.Println("Running dynamic mounting tests...")
		// Save mount directory variable to have path of bucket to run tests.
		testEnv.mountDir = path.Join(setup.MntDir(), setup.TestBucket())
		testEnv.mountFunc = dynamic_mounting.MountGcsfuseWithDynamicMountingWithConfig
		successCode = m.Run()
	}

	if successCode == 0 {
		log.Println("Running only dir mounting tests...")
		setup.SetOnlyDirMounted(onlyDirMounted + "/")
		testEnv.mountDir = testEnv.rootDir
		testEnv.mountFunc = only_dir_mounting.MountGcsfuseWithOnlyDirWithConfigFile
		successCode = m.Run()
		setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(setup.TestBucket(), setup.OnlyDirMounted(), testDirName))
	}

	// If failed, then save the gcsfuse log file(s).
	setup.SaveLogFileInCaseOfFailure(successCode)

	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)
}
