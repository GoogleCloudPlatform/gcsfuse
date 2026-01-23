// Copyright 2026 Google LLC
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
package kernel_reader

import (
	"context"
	"log"
	"os"
	"path"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/only_dir_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

const (
	testDirName    = "KernelReaderTests"
	onlyDirMounted = "OnlyDirMountKernelReader"
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

func mountGCSFuseAndSetupTestDir(flags []string, ctx context.Context, storageClient *storage.Client) {
	setup.MountGCSFuseWithGivenMountWithConfigFunc(&testEnv.cfg, flags, testEnv.mountFunc)
	if testEnv.cfg.GKEMountedDirectory == "" {
		setup.SetMntDir(testEnv.mountDir)
	}
	testEnv.testDirPath = client.SetupTestDirectory(ctx, storageClient, testDirName)
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
	if len(cfg.KernelReader) == 0 {
		log.Println("No configuration found for kernel_reader tests in config. Using flags instead.")
		// Populate the config manually.
		cfg.KernelReader = make([]test_suite.TestConfig, 1)
		cfg.KernelReader[0].TestBucket = setup.TestBucket()
		cfg.KernelReader[0].GKEMountedDirectory = setup.MountedDirectory()
		cfg.KernelReader[0].LogFile = setup.LogFile()
		// Initialize the slice to hold 3 specific test configurations
		cfg.KernelReader[0].Configs = make([]test_suite.ConfigItem, 3)

		cfg.KernelReader[0].Configs[0].Run = "TestFileCache_KernelReaderDisabled"
		cfg.KernelReader[0].Configs[0].Flags = []string{"--implicit-dirs --log-severity=trace --enable-kernel-reader=false --cache-dir=/gcsfuse-tmp/TestFileCache_KernelReaderDisabled"}
		cfg.KernelReader[0].Configs[0].Compatible = map[string]bool{"flat": false, "hns": false, "zonal": true}
		cfg.KernelReader[0].Configs[0].RunOnGKE = false

		cfg.KernelReader[0].Configs[1].Run = "TestKernelReader_DefaultAndPrecedence"
		cfg.KernelReader[0].Configs[1].Flags = []string{
			"--implicit-dirs --log-severity=trace",
			"--implicit-dirs --log-severity=trace --cache-dir=/gcsfuse-tmp/TestKernelReader_DefaultAndPrecedence_FileCache",
			"--implicit-dirs --log-severity=trace --enable-buffered-read=true",
			"--implicit-dirs --log-severity=trace --enable-buffered-read=true --cache-dir=/gcsfuse-tmp/TestKernelReader_DefaultAndPrecedence_Both",
		}
		cfg.KernelReader[0].Configs[1].Compatible = map[string]bool{"flat": false, "hns": false, "zonal": true}
		cfg.KernelReader[0].Configs[1].RunOnGKE = false

		cfg.KernelReader[0].Configs[2].Run = "TestBufferedReader_KernelReaderDisabled"
		cfg.KernelReader[0].Configs[2].Flags = []string{"--implicit-dirs --log-severity=trace --enable-kernel-reader=false --enable-buffered-read"}
		cfg.KernelReader[0].Configs[2].Compatible = map[string]bool{"flat": false, "hns": false, "zonal": true}
		cfg.KernelReader[0].Configs[2].RunOnGKE = false
	}

	testEnv.ctx = context.Background()
	testEnv.bucketType = setup.TestEnvironment(testEnv.ctx, &cfg.KernelReader[0])
	testEnv.cfg = cfg.KernelReader[0]

	var err error
	testEnv.storageClient, err = client.CreateStorageClient(testEnv.ctx)
	if err != nil {
		log.Fatalf("Error creating storage client: %v\n", err)
	}
	defer testEnv.storageClient.Close()

	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	// To run mountedDirectory tests, we need both testBucket and mountedDirectory
	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		os.Exit(setup.RunTestsForMountedDirectory(testEnv.cfg.GKEMountedDirectory, m))
	}

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
		log.Println("Running only dir mounting tests...")
		setup.SetOnlyDirMounted(onlyDirMounted + "/")
		testEnv.mountDir = testEnv.rootDir
		testEnv.mountFunc = only_dir_mounting.MountGcsfuseWithOnlyDirWithConfigFile
		successCode = m.Run()
		setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(setup.TestBucket(), setup.OnlyDirMounted(), testDirName))
	}

	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)
}
