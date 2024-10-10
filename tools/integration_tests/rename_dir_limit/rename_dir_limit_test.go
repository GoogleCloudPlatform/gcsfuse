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
const SrcDirectory = "srcDirectory"
const EmptyDestDirectory = "emptyDestDirectory"
const PrefixTempFile = "temp"
const onlyDirMounted = "OnlyDirMountRenameDirLimit"

var (
	storageClient *storage.Client
	ctx           context.Context
)

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	var err error

	ctx = context.Background()
	storageClient, err = client.CreateStorageClient(ctx)
	if err != nil {
		log.Printf("Error creating storage client: %v\n", err)
		os.Exit(1)
	}
	defer storageClient.Close()

	flags := [][]string{{"--rename-dir-limit=3", "--implicit-dirs"}, {"--rename-dir-limit=3"}}
	if hnsFlagSet, err := setup.AddHNSFlagForHierarchicalBucket(ctx, storageClient); err == nil {
		flags = [][]string{hnsFlagSet}
	}
	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	if setup.TestBucket() == "" && setup.MountedDirectory() != "" {
		log.Print("Please pass the name of bucket mounted at mountedDirectory to --testBucket flag.")
		os.Exit(1)
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
