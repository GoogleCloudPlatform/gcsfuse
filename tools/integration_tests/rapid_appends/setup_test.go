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

package rapid_appends

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
	testDirName = "RapidAppendsTest"
)

var (
	// Flags for mount options for primaryMntRootDir
	flags []string
	// Mount function to be used for the mounting.
	mountFunc func([]string) error

	// Globals for primary mount which is used to append on existing unfinalized files.
	// Root directory which is mounted by gcsfuse.
	primaryMntRootDir string
	// Stores test directory path in the mounted path for primaryMntRootDir.
	primaryMntTestDirPath string
	// Stores log file path for the mount primaryMntRootDir.
	primaryMntLogFilePath string

	// Globals for secondary mount which is used to verify reads.
	// Other Root directory which is mounted by gcsfuse for multi-mount scenarios.
	secondaryMntRootDir string
	// Stores test directory path in the mounted path for secondaryMntRootDir.
	secondaryMntTestDirPath string
	// Stores log file path for the mount secondaryMntRootDirr.
	secondaryMntLogFilePath string

	// Clients to create the object in GCS.
	storageClient *storage.Client
	ctx           context.Context
)

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()
	if !setup.IsZonalBucketRun() {
		log.Fatalf("This package is not supposed to be run with Regional Buckets.")
	}
	// TODO(b/431926259): Add support for mountedDir tests as this
	// package has multi-mount scenario tests and currently we only
	// pass single mountedDir to test package.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		log.Fatalf("This package doesn't support mountedDir tests.")
	}
	ctx = context.Background()
	closeStorageClient := client.CreateStorageClientWithCancel(&ctx, &storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	// Set up test directory for secondary mount.
	setup.SetUpTestDirForTestBucketFlag()
	secondaryMntRootDir = setup.MntDir()
	secondaryMntLogFilePath = setup.LogFile()
	// For reads to work for unfinalized object from secondary mount metadata cache ttl must be set to 0.
	// and rapid appends should be enabled.
	secondaryMountFlags := []string{"--write-experimental-enable-rapid-appends=true", "--metadata-cache-ttl-secs=0"}
	err := static_mounting.MountGcsfuseWithStaticMounting(secondaryMountFlags)
	if err != nil {
		log.Fatalf("Unable to mount secondary mount: %v", err)
	}
	// Setup Package Test Directory for secondary mount.
	secondaryMntTestDirPath = setup.SetupTestDirectory(testDirName)
	defer func() {
		setup.UnmountGCSFuse(secondaryMntRootDir)
	}()

	// Set up test directory for primary mount.
	setup.SetUpTestDirForTestBucketFlag()

	rapidAppendsCacheDir, err := os.MkdirTemp("", "rapid_appends_cache_dir_*")
	if err != nil {
		log.Fatalf("Failed to create cache dir for rapid append tests: %v", err)
	}
	defer func() {
		err := os.RemoveAll(rapidAppendsCacheDir)
		if err != nil {
			log.Fatalf("Error while cleaning up cache dir: %v", err)
		}
	}()
	// Define flag set for primary mount to run the tests.
	flagsSet := [][]string{
		{"--write-experimental-enable-rapid-appends=true"},
		{"--write-experimental-enable-rapid-appends=true", "--metadata-cache-ttl-secs=0"},
		{"--write-experimental-enable-rapid-appends=true", "--file-cache-max-size-mb=-1", "--cache-dir=" + rapidAppendsCacheDir},
		{"--write-experimental-enable-rapid-appends=true", "--metadata-cache-ttl-secs=0", "--file-cache-max-size-mb=-1", "--cache-dir=" + rapidAppendsCacheDir},
	}

	log.Println("Running static mounting tests...")
	mountFunc = static_mounting.MountGcsfuseWithStaticMounting

	var successCode int
	for i := range flagsSet {
		log.Printf("Running tests with flags: %v", flagsSet[i])
		flags = flagsSet[i]
		successCode = m.Run()
		if successCode != 0 {
			break
		}
	}
	os.Exit(successCode)
}
