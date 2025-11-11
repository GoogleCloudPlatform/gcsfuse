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

package stale_handle

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
	testDirName = "StaleHandleTest"
)

var (
	storageClient *storage.Client
	ctx           context.Context
	rootDir       string
	mountFunc     func([]string) error
	flagsSet      [][]string
)

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

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

	// To run mountedDirectory tests, we need both testBucket and mountedDirectory
	// flags to be set, as stale handle tests validates content from the bucket.
	// Note: These tests by default can only be run for non streaming mounts.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		rootDir = setup.MountedDirectory()
		setup.RunTestsForMountedDirectoryFlag(m)
		return
	}

	// Set up test directory.
	setup.SetUpTestDirForTestBucketFlag()
	rootDir = setup.MntDir()

	flagsSet = [][]string{
		{"--metadata-cache-ttl-secs=0", "--enable-streaming-writes=false"},
		{"--metadata-cache-ttl-secs=0", "--write-block-size-mb=1", "--write-max-blocks-per-file=1"},
	}
	// Run all tests with GRPC.
	setup.AppendFlagsToAllFlagsInTheFlagsSet(&flagsSet, "--client-protocol=grpc", "")

	log.Println("Running static mounting tests...")
	mountFunc = static_mounting.MountGcsfuseWithStaticMounting
	successCode := m.Run()
	os.Exit(successCode + 1)
}
