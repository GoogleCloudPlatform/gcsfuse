// Copyright 2023 Google LLC
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

// Provide test for listing large directory
package list_large_dir

import (
	"context"
	"log"
	"os"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

const directoryForListLargeFileTests = "directoryForListLargeFileTests"
const prefixFileInDirectoryWithTwelveThousandFiles = "fileInDirectoryWithTwelveThousandFiles"
const prefixExplicitDirInLargeDirListTest = "explicitDirInLargeDirListTest"
const prefixImplicitDirInLargeDirListTest = "implicitDirInLargeDirListTest"
const numberOfFilesInDirectoryWithTwelveThousandFiles = 12000
const numberOfImplicitDirsInDirectoryWithTwelveThousandFiles = 100
const numberOfExplicitDirsInDirectoryWithTwelveThousandFiles = 100

var (
	directoryWithTwelveThousandFiles = "directoryWithTwelveThousandFiles" + setup.GenerateRandomString(5)
	storageClient                    *storage.Client
	ctx                              context.Context
)

func TestMain(m *testing.M) {
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

	flags := [][]string{{"--implicit-dirs", "--stat-cache-ttl=0", "--kernel-list-cache-ttl-secs=-1"}}
	if !testing.Short() {
		flags = append(flags, []string{"--client-protocol=grpc", "--implicit-dirs=true", "--stat-cache-ttl=0", "--kernel-list-cache-ttl-secs=-1"})
	}

	// Run tests for mountedDirectory only if --mountedDirectory flag is set.
	setup.RunTestsForMountedDirectoryFlag(m)

	setup.SetUpTestDirForTestBucketFlag()

	successCode := static_mounting.RunTests(flags, m)

	os.Exit(successCode)
}
