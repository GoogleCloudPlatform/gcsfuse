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

package streaming_writes

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
	testDirName = "StreamingWritesTest"
)

var (
	flags       []string
	testDirPath string
	mountFunc   func([]string) error
	// root directory is the directory to be unmounted.
	rootDir       string
	storageClient *storage.Client
	ctx           context.Context
)

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	// Create storage client before running tests.
	ctx = context.Background()
	closeStorageClient := client.CreateStorageClientWithCancel(&ctx, &storageClient)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	// To run mountedDirectory tests, we need both testBucket and mountedDirectory
	// flags to be set, as operations tests validates content from the bucket.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		rootDir = setup.MountedDirectory()
		setup.RunTestsForMountedDirectoryFlag(m)
	}

	// Set up test directory.
	setup.SetUpTestDirForTestBucketFlag()
	rootDir = setup.MntDir()

	// Define flag set to run the tests.
	flagsSet := [][]string{
		{"--rename-dir-limit=3", "--write-block-size-mb=1", "--write-max-blocks-per-file=2", "--write-global-max-blocks=0"},
		{"--rename-dir-limit=3", "--write-block-size-mb=1", "--write-max-blocks-per-file=2", "--client-protocol=grpc", "--write-global-max-blocks=-1"},
		{"--rename-dir-limit=3", "--write-block-size-mb=1", "--write-max-blocks-per-file=2", "--write-global-max-blocks=-1"},
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
