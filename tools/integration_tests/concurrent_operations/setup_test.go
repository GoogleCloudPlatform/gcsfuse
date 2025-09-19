// Copyright 2024 Google LLC
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

package concurrent_operations

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
	testDirName    = "ConcurrentOperationsTest"
	onlyDirMounted = "OnlyDirConcurrentOperationsTest"
)

var (
	storageClient *storage.Client
	ctx           context.Context
	testDirPath   string

	// root directory is the directory to be unmounted.
	rootDir string
)

////////////////////////////////////////////////////////////////////////
// Helper functions
////////////////////////////////////////////////////////////////////////

func mountGCSFuseAndSetupTestDir(flags []string, testDirName string, t *testing.T) {
	// When tests are running in GKE environment, use the mounted directory provided as test flag.
	if setup.MountedDirectory() != "" {
		testDirPathForRead = setup.MountedDirectory()
	} else {
		config := &test_suite.TestConfig{
			TestBucket:              setup.TestBucket(),
			GCSFuseMountedDirectory: setup.MntDir(),
			GKEMountedDirectory:     setup.MountedDirectory(),
			LogFile:                 setup.LogFile(),
		}
		if err := static_mounting.MountGcsfuseWithStaticMountingWithConfigFile(config, flags); err != nil {
			t.Fatalf("Failed to mount GCS FUSE: %v", err)
			return
		}
		testDirPathForRead = setup.MntDir()
	}
	setup.SetMntDir(testDirPathForRead)
	testDirPath := setup.SetupTestDirectory(testDirName)
	testDirPathForRead = testDirPath
}

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()
	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	// Create common storage client to be used in test.
	ctx = context.Background()
	closeStorageClient := client.CreateStorageClientWithCancel(&ctx, &storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	// If Mounted Directory flag is set, run tests for mounted directory.
	setup.RunTestsForMountedDirectoryFlag(m)
	// Else run tests for testBucket.
	// Set up test directory.
	setup.SetUpTestDirForTestBucketFlag()

	// Save root directory variables.
	rootDir = setup.MntDir()

	log.Println("Running static mounting tests...")
	successCode := m.Run()

	// If test failed, save the gcsfuse log files for debugging.
	setup.SaveLogFileInCaseOfFailure(successCode)

	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(ctx, storageClient, path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)
}
