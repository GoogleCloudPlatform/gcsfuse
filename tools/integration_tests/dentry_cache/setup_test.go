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

// Provides integration tests for enabling dentry cache.
package dentry_cache

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
	testDirName        = "testDirForDentryCache"
	testFileName       = "testFile"
	initialContentSize = 5
	updatedContentSize = 10
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

func mountGCSFuseAndSetupTestDir(flags []string, testDirName string) {
	setup.MountGCSFuseWithGivenMountFunc(flags, mountFunc)
	setup.SetMntDir(mountDir)
	testDirPath = client.SetupTestDirectory(ctx, storageClient, testDirName)
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

	if setup.MountedDirectory() != "" {
		mountDir = setup.MountedDirectory()
		// Run tests for mounted directory if the flag is set.
		os.Exit(m.Run())
	}
	// Else run tests for testBucket.
	// Set up test directory.
	setup.SetUpTestDirForTestBucketFlag()

	// Save mount and root directory variables.
	mountDir, rootDir = setup.MntDir(), setup.MntDir()

	log.Println("Running static mounting tests...")
	mountFunc = static_mounting.MountGcsfuseWithStaticMounting
	successCode := m.Run()

	os.Exit(successCode)
}
