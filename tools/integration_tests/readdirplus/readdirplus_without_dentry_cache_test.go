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

// Provides integration tests for long listing directory with Readdirplus
package readdirplus

import (
	"log"
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/test_setup"
	"github.com/jacobsa/fuse/fusetesting"
	"github.com/stretchr/testify/require"
)

type readdirplusWithoutDentryCacheTest struct {
	flags []string
}

func (s *readdirplusWithoutDentryCacheTest) Setup(t *testing.T) {
	mountGCSFuseAndSetupTestDir(t, s.flags, testDirName)
}

func (s *readdirplusWithoutDentryCacheTest) Teardown(t *testing.T) {
	if setup.MountedDirectory() == "" { // Only unmount if not using a pre-mounted directory
		setup.CleanupDirectoryOnGCS(ctx, storageClient, path.Join(setup.TestBucket(), testDirName))
		setup.UnmountGCSFuseAndDeleteLogFile(rootDir)
	}
}

func (s *readdirplusWithoutDentryCacheTest) TestReaddirplusWithoutDentryCache(t *testing.T) {
	targetDir := path.Join(testDirPath, targetDirName)
	expected := createDirectoryStructure(t)

	// Call Readdirplus to list the directory.
	startTime := time.Now()
	entries, err := fusetesting.ReadDirPlusPicky(targetDir)
	endTime := time.Now()

	require.NoError(t, err, "ReadDirPlusPicky failed")
	// Verify the entries.
	validateEntries(entries, expected, t)
	// Validate logs to check that ReadDirPlus was called and ReadDir was not.
	// Dentry cache is not enabled, so LookUpInode should be called for
	// parent directory as well as for all the entries.
	validateLogsForReaddirplus(t, setup.LogFile(), false, startTime, endTime)
}

func TestReaddirplusWithoutDentryCacheTest(t *testing.T) {
	ts := &readdirplusWithoutDentryCacheTest{}

	// Run tests for mounted directory if the flag is set.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		test_setup.RunTests(t, ts)
		return
	}

	// Run tests.
	ts.flags = []string{"--implicit-dirs", "--experimental-enable-readdirplus"}
	log.Printf("Running tests with flags: %s", ts.flags)
	test_setup.RunTests(t, ts)
}
