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

// Provides integration tests for managed folders.
package empty_managed_folders_list

import (
	"github.com/googlecloudplatform/gcsfuse/internal/config"
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting/dynamic_mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting/only_dir_mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting/persistent_mounting"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting/static_mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

const (
	testDirName                     = "managedFolderTest"
	NumberOfObjectsInDirForListTest = 4
	EmptyManagedFolder1             = "emptyManagedFolder1"
	EmptyManagedFolder2             = "emptyManagedFolder2"
	SimulatedFolder                 = "simulatedFolder"
	File                            = "testFile"
)

func getMountConfigForEmptyManagedFolders() config.MountConfig {
	mountConfig := config.MountConfig{
		ListConfig: config.ListConfig{
			EnableManagedFolders: true,
		},
	}
	return mountConfig
}

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	// Run tests for mountedDirectory only if --mountedDirectory  and --testBucket flag is set.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		setup.RunTestsForMountedDirectoryFlag(m)
		return
	}

	// Else run tests for testBucket.
	// Set up test directory.
	setup.SetUpTestDirForTestBucketFlag()

	configFile := setup.YAMLConfigFile(
		getMountConfigForEmptyManagedFolders(),
		"config.yaml")
	flags := [][]string{
		{"--implicit-dirs","--config-file=" + configFile},
	}

	successCode := static_mounting.RunTests(flags, m)

	if successCode == 0 {
		successCode = only_dir_mounting.RunTests(flags, m)
	}

	if successCode == 0 {
		successCode = dynamic_mounting.RunTests(flags, m)
	}

	if successCode == 0 {
		successCode = persistent_mounting.RunTests(flags, m)
	}

	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(path.Join(setup.TestBucket(), testDirName))
	setup.RemoveBinFileCopiedForTesting()
	os.Exit(successCode)
}
