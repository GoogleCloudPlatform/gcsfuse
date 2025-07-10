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

package unfinalized_appends

import (
	"context"
	"log"
	"os"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

const (
	testDirName = "AppendFileTest"
)

var (
	// Flags for the mount options for gRootDir and gOtherRootDir
	gFlags []string
	// Mount function to be used for the mounting.
	gMountFunc func([]string) error

	// Globals for primary mount.
	// Root directory which is mounted by gcsfuse.
	gRootDir string
	// Stores test directory path in the mounted path for gRootDir.
	gTestDirPath string
	// Stores log file path for the mount gRootDir.
	gLogFilePath string

	// Globals for secondary mount.
	// Other Root directory which is mounted by gcsfuse for multi-mount scenarios.
	gOtherRootDir string
	// Stores test directory path in the mounted path for gOtherRootDir.
	gOtherTestDirPath string
	// Stores log file path for the mount gOtherRootDir.
	gOtherLogFilePath string

	// Clients to create the object in GCS.
	gStorageClient *storage.Client
	gCtx           context.Context
)

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// Create storage client before running tests.
	gCtx = context.Background()
	closeStorageClient := client.CreateStorageClientWithCancel(&gCtx, &gStorageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	// Set up gOtherRootDir directory for secondary mount.
	setup.SetUpTestDirForTestBucketFlag()
	gOtherRootDir = setup.MntDir()
	gOtherTestDirPath = setup.SetupTestDirectory(testDirName)
	gOtherLogFilePath = setup.LogFile()
	// For reads to work for unfinalized object from secondary mount metadata cache ttl must be set to 0.
	// and rapid appends should be enabled.
	secondarMountFlags := []string{"--metadata-cache-ttl-secs=0", "--write-experimental-enable-rapid-appends=true"}
	err := static_mounting.MountGcsfuseWithStaticMounting(secondarMountFlags)
	if err != nil {
		log.Fatalf("Unable to mount secondary mount: %v", err)
	}
	defer func() {
		setup.UnmountGCSFuse(gOtherRootDir)
	}()

	// To run mountedDirectory tests, we need both testBucket and mountedDirectory flags.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		setup.SetMntDir(setup.MountedDirectory())
		gRootDir = setup.MntDir()
		gTestDirPath = setup.SetupTestDirectory(testDirName)
		setup.RunTestsForMountedDirectoryFlag(m)
	}

	// Set up gRootDir directory for primary mount.
	setup.SetUpTestDirForTestBucketFlag()
	gRootDir = setup.MntDir()
	gTestDirPath = setup.SetupTestDirectory(testDirName)
	gLogFilePath = setup.LogFile()

	// Define flag set to run the tests.
	flagsSet := [][]string{
		{"--metadata-cache-ttl-secs=0", "--write-experimental-enable-rapid-appends=true"},
	}

	log.Println("Running static mounting tests...")
	gMountFunc = static_mounting.MountGcsfuseWithStaticMounting

	var successCode int
	for i := range flagsSet {
		log.Printf("Running tests with flags: %v", flagsSet[i])
		gFlags = flagsSet[i]
		successCode = m.Run()
		if successCode != 0 {
			break
		}
	}
	os.Exit(successCode)
}
