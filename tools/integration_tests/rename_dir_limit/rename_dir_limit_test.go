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

// Provides integration tests when --rename-dir-limit flag is set.
package rename_dir_limit_test

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/only_dir_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/persistent_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

const DirForRenameDirLimitTests = "dirForRenameDirLimitTests"
const DirectoryWithThreeFiles = "directoryWithThreeFiles"
const DirectoryWithTwoFiles = "directoryWithTwoFiles"
const DirectoryWithFourFiles = "directoryWithFourFiles"
const DirectoryWithTwoFilesOneEmptyDirectory = "directoryWithTwoFilesOneEmptyDirectory"
const DirectoryWithTwoFilesOneNonEmptyDirectory = "directoryWithTwoFilesOneNonEmptyDirectory"
const EmptySubDirectory = "emptySubDirectory"
const NonEmptySubDirectory = "nonEmptySubDirectory"
const RenamedDirectory = "renamedDirectory"
const PrefixTempFile = "temp"
const onlyDirMounted = "OnlyDirMountRenameDirLimit"

var (
	storageClient *storage.Client
	ctx           context.Context
)

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	flags := [][]string{{"--rename-dir-limit=3", "--implicit-dirs"}, {"--rename-dir-limit=3"}}

	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	// Create storage client before running tests.
	ctx = context.Background()
	closeStorageClient := client.CreateStorageClientWithTimeOut(&ctx, &storageClient, time.Minute*15)
	defer func() {
		err := closeStorageClient()
		if err != nil {
			log.Fatalf("closeStorageClient failed: %v", err)
		}
	}()

	// To run mountedDirectory tests, we need both testBucket and mountedDirectory
	// flags to be set, as bucket name require to get bucket type.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		setup.RunTestsForMountedDirectoryFlag(m)
	}

	// Don't pass --rename-dir-limit or --implicit-dirs flags for hierarchical bucket.
	if setup.IsHierarchicalBucket(ctx, storageClient) {
		flags = [][]string{{}}
	}

	// Run tests for mountedDirectory only if --mountedDirectory flag is set.
	setup.RunTestsForMountedDirectoryFlag(m)

	// Run tests for testBucket
	setup.SetUpTestDirForTestBucketFlag()

	successCode := static_mounting.RunTests(flags, m)

	if successCode == 0 {
		successCode = only_dir_mounting.RunTests(flags, onlyDirMounted, m)
	}

	if successCode == 0 {
		successCode = persistent_mounting.RunTests(flags, m)
	}

	os.Exit(successCode)
}
