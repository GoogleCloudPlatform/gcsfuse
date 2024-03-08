// Copyright 2024 Google Inc. All Rights Reserved.
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

// Provides integration tests for managed folders.
package managed_folders

import (
	"log"
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting/only_dir_mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting/static_mounting"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting/dynamic_mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

const (
	onlyDirMounted = "TestManagedFolderOnlyDir"
)

var (
	mountFunc func([]string) error
	// Mount directory is where our tests run.
	mountDir string
	// Root directory is the directory to be unmounted.
	rootDir string
)

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

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

	if successCode == 0 {
		log.Println("Running only dir mounting tests...")
		setup.SetOnlyDirMounted(onlyDirMounted + "/")
		mountFunc = only_dir_mounting.MountGcsfuseWithOnlyDir
		successCode = m.Run()
		setup.SaveLogFileInCaseOfFailure(successCode)
		setup.SetOnlyDirMounted("")
	}

	if successCode == 0 {
		log.Println("Running dynamic mounting tests...")
		// Save mount directory variable to have path of bucket to run tests.
		mountDir = path.Join(setup.MntDir(), setup.TestBucket())
		mountFunc = dynamic_mounting.MountGcsfuseWithDynamicMounting
		successCode = m.Run()
		setup.SaveLogFileInCaseOfFailure(successCode)
	}

	setup.RemoveBinFileCopiedForTesting()
	os.Exit(successCode)
}
