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

package negative_stat_cache

import (
	"context"
	"log"
	"os"
	"path"
	"testing"

	"cloud.google.com/go/storage"
	control "cloud.google.com/go/storage/control/apiv2"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/dynamic_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/only_dir_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

var (
	testDirName    = "NegativeStatCacheTest"
	onlyDirMounted = "OnlyDirMountNegativeStatCache"
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
	rootDir              string
	storageClient        *storage.Client
	storageControlClient *control.StorageControlClient
	ctx                  context.Context
}

var testEnv env

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func mountGCSFuseAndSetupTestDir(flags []string, testDirName string) {
	if setup.MountedDirectory() != "" {
		testEnv.mountDir = setup.MountedDirectory()
	}
	setup.MountGCSFuseWithGivenMountFunc(flags, testEnv.mountFunc)
	setup.SetMntDir(testEnv.mountDir)
	testEnv.testDirPath = setup.SetupTestDirectory(testDirName)
}

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()
	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	testDirName += "-" + setup.GenerateRandomString(5)
	onlyDirMounted += "-" + setup.GenerateRandomString(5)
	log.Printf("Using test directory: %s", testDirName)
	log.Printf("Using onlyDirMounted: %s", onlyDirMounted)

	// Create common storage client to be used in test.
	testEnv.ctx = context.Background()
	closeStorageClient := client.CreateStorageClientWithCancel(&testEnv.ctx, &testEnv.storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()
	closeStorageControlClient := client.CreateControlClientWithCancel(&testEnv.ctx, &testEnv.storageControlClient)
	defer func() {
		err := closeStorageControlClient()
		if err != nil {
			log.Fatalf("closeStorageControlClient failed: %v", err)
		}
	}()

	// If Mounted Directory flag is set, run tests for mounted directory.
	setup.RunTestsForMountedDirectoryFlag(m)
	// Else run tests for testBucket.
	// Set up test directory.
	setup.SetUpTestDirForTestBucketFlag()

	// Save mount and root directory variables.
	testEnv.mountDir, testEnv.rootDir = setup.MntDir(), setup.MntDir()

	log.Println("Running static mounting tests...")
	testEnv.mountFunc = static_mounting.MountGcsfuseWithStaticMounting
	successCode := m.Run()

	if successCode == 0 {
		log.Println("Running dynamic mounting tests...")
		// Save mount directory variable to have path of bucket to run tests.
		testEnv.mountDir = path.Join(setup.MntDir(), setup.TestBucket())
		testEnv.mountFunc = dynamic_mounting.MountGcsfuseWithDynamicMounting
		successCode = m.Run()
	}

	if successCode == 0 {
		log.Println("Running only dir mounting tests...")
		setup.SetOnlyDirMounted(onlyDirMounted + "/")
		testEnv.mountDir = testEnv.rootDir
		testEnv.mountFunc = only_dir_mounting.MountGcsfuseWithOnlyDir
		successCode = m.Run()
		setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(setup.TestBucket(), setup.OnlyDirMounted(), testDirName))
	}

	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)
}
