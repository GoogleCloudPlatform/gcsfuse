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

// Provide tests for cases where bucket with requester-pays feature is
// mounted and used through gcsfuse.
package requester_pays_bucket

import (
	"context"
	"log"
	"os"
	"path"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_suite"
)

const (
	testDirName = "RequesterPaysBucketTests"
)

// To prevent global variable pollution, enhance code clarity,
// and avoid inadvertent errors. We strongly suggest that, all new package-level
// variables (which would otherwise be declared with `var` at the package root) should
// be added as fields to this 'env' struct instead.
type env struct {
	testDirPath string
	mountFunc   func([]string) error
	// mount directory is where our tests run.
	mountDir string
	// root directory is the directory to be mounted/unmounted.
	rootDir       string
	storageClient *storage.Client
	ctx           context.Context
	bucketName    string
}

var (
	logFileNameForMountedDirectoryTests = path.Join(os.TempDir(), "gcsfuse_requester_pays_bucket_test_logs", "log.json")
	testEnv                             env
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

	if setup.IsZonalBucketRun() {
		log.Fatal("Test not supported for zonal bucket as they don't support requester-pays feature")
	}

	testEnv.ctx = context.Background()
	// Temporarily enable --requester-pays metadata flag for the test bucket.
	if setup.TestBucket() == "" {
		log.Fatal("testBucket not passed")
	}
	testEnv.bucketName = strings.Split(setup.TestBucket(), "/")[0]
	client.EnableRequesterPays(testEnv.ctx, testEnv.bucketName)
	defer client.DisableRequesterPays(testEnv.ctx, testEnv.bucketName)
	// Set up storage-client.
	closeStorageClient := client.CreateStorageClientWithCancel(&testEnv.ctx, &testEnv.storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

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
