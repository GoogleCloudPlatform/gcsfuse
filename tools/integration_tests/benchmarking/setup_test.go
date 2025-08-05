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
//
// Note that the expected latency thresholds for the various operations has
// been set to 4 times the observed latency. Any failure of the benchmark tests
// is a direct indicator of anomaly.

package benchmarking

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

const (
	testDirName = "benchmarking"
)

var (
	storageClient *storage.Client
	ctx           context.Context
	testDirPath   string
	mountFunc     func([]string) error
	// mount directory is where our tests run.
	mountDir string
	// root directory is the directory to be unmounted.
	rootDir string
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func mountGCSFuseAndSetupTestDir(flags []string, testDirName string) {
	// When tests are running in GKE environment, use the mounted directory provided as test flag.
	if setup.MountedDirectory() != "" {
		mountDir = setup.MountedDirectory()
	}
	setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)
	setup.SetMntDir(mountDir)
	testDirPath = setup.SetupTestDirectory(testDirName)
}

// createFiles creates the below objects in the bucket.
// benchmarking/a{i}.txt where i is a counter based on the benchtime value.
func createFiles(b *testing.B) {
	for i := range b.N {
		operations.CreateFileOfSize(1, path.Join(testDirPath, fmt.Sprintf("a%d.txt", i)), b)
	}
}

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()
	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	// Create common storage client to be used in test.
	ctx = context.Background()
	closeStorageClient := client.CreateStorageClientWithCancel(&ctx, &storageClient)
	defer func() {
		if err := closeStorageClient(); err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	// If Mounted Directory flag is set, run tests for mounted directory.
	setup.RunTestsForMountedDirectoryFlag(m)

	// Else run tests for testBucket.
	// Set up test directory.
	setup.SetUpTestDirForTestBucketFlag()

	// Save mount and root directory variables.
	mountDir, rootDir = setup.MntDir(), setup.MntDir()

	log.Println("Running static mounting tests...")
	mountFunc = static_mounting.MountGcsfuseWithStaticMounting
	successCode := m.Run()
	setup.SaveLogFileInCaseOfFailure(successCode)

	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(ctx, storageClient, path.Join(setup.TestBucket(), testDirName))
	os.Exit(successCode)
}
