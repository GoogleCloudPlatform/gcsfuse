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

// Provides integration tests for operations on managed folders.

package managed_folder

import (
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/creds_tests"
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

var IsPermissionObjectViewer bool = true
const (
	testDirName = "ManagedFolderTest"
	ManagedFolder = "ManagedFolder"
	TestFileInManagedFolder = "TestFileInManagedFolder"
)

func SkipTestIfNotViewerPermission(t *testing.T){
	if IsPermissionObjectViewer != true {
		t.Logf("This test will run only for bucket with view permissions..")
		t.SkipNow()
	}
}

func SkipTestIfNotAdminPermission(t *testing.T){
	if IsPermissionObjectViewer == true {
		t.Logf("This test will run only for bucket with admin permissions..")
		t.SkipNow()
	}
}

////////////////////////////////////////////////////////////////////////
// TestMain
////////////////////////////////////////////////////////////////////////

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	setup.ExitWithFailureIfBothTestBucketAndMountedDirectoryFlagsAreNotSet()

	// To run mountedDirectory tests, we need both testBucket and mountedDirectory
	// flags to be set, as local_file tests validates content from the bucket.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		setup.RunTestsForMountedDirectoryFlag(m)
	}

	// Else run tests for testBucket.
	// Set up test directory.
	setup.SetUpTestDirForTestBucketFlag()

	// Set up flags to run tests on.
	// Not setting config file explicitly with 'create-empty-file: false' as it is default.
	flags := [][]string{
		{"--implicit-dirs=true"}}


	successCode := creds_tests.RunTestsForKeyFileAndGoogleApplicationCredentialsEnvVarSet(flags, "objectViewer", m)

	IsPermissionObjectViewer = false
	successCode = creds_tests.RunTestsForKeyFileAndGoogleApplicationCredentialsEnvVarSet(flags, "objectAdmin", m)

	// Clean up test directory created.
	setup.CleanupDirectoryOnGCS(path.Join(setup.TestBucket(), testDirName))
	setup.RemoveBinFileCopiedForTesting()
	os.Exit(successCode)
}
