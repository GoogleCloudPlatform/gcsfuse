// Copyright 2023 Google Inc. All Rights Reserved.
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

// Provide tests when implicit directory present and mounted bucket with --implicit-dir flag.
package implicit_dir_test

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup/implicit_and_explicit_dir_setup"
)

const ExplicitDirInImplicitDir = "explicitDirInImplicitDir"
const ExplicitDirInImplicitSubDir = "explicitDirInImplicitSubDir"
const PrefixFileInExplicitDirInImplicitDir = "fileInExplicitDirInImplicitDir"
const PrefixFileInExplicitDirInImplicitSubDir = "fileInExplicitDirInImplicitSubDir"
const NumberOfFilesInExplicitDirInImplicitSubDir = 1
const NumberOfFilesInExplicitDirInImplicitDir = 1

func TestMain(m *testing.M) {
	ctx = context.Background()
	var cancel context.CancelFunc
	var err error

	// Run tests for mountedDirectory only if --mountedDirectory and --testbucket flag is set.
	setup.RunTestsForMountedDirectoryFlag(m)

	// Run tests for testBucket only if --testbucket flag is set.
	setup.SetUpTestDirForTestBucketFlag()

	// Set testDirPath to run tests on, in the MntDir.
	testDirPath = path.Join(setup.MntDir(), testDirName)
	// Create storage client before running tests.
	ctx, cancel = context.WithTimeout(ctx, time.Minute*15)
	storageClient, err = client.CreateStorageClient(ctx)
	if err != nil {
		fmt.Printf("client.CreateStorageClient: %v", err)
		os.Exit(1)
	}

	flags := [][]string{{"--implicit-dirs"}}

	implicit_and_explicit_dir_setup.RunTestsForImplicitDirAndExplicitDir(flags, m)

	// Close storage client and release resources.
	cancel()
	storageClient.Close()
	// Clean up test directory created.
	setup.CleanupTestDirectoryOnGCS(testDirName)
}
