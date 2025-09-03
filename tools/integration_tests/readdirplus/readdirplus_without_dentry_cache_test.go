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

package readdirplus

import (
	"log"
	"os"
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/jacobsa/fuse/fusetesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type readdirplusWithoutDentryCacheTest struct {
	flags []string
	suite.Suite
}

func (s *readdirplusWithoutDentryCacheTest) SetupTest() {
	mountGCSFuseAndSetupTestDir(s.flags, testDirName)
}

func (s *readdirplusWithoutDentryCacheTest) TearDownTest() {
	if setup.MountedDirectory() == "" { // Only unmount if not using a pre-mounted directory
		setup.CleanupDirectoryOnGCS(ctx, storageClient, path.Join(setup.TestBucket(), testDirName))
		setup.UnmountGCSFuseAndDeleteLogFile(rootDir)
	}
}

func (s *readdirplusWithoutDentryCacheTest) TestReaddirplusWithoutDentryCache() {
	// Create directory structure
	// testBucket/target_dir/                                                       -- Dir
	// testBucket/target_dir/file		                                            -- File
	// testBucket/target_dir/emptySubDirectory                                      -- Dir
	// testBucket/target_dir/subDirectory                                           -- Dir
	// testBucket/target_dir/subDirectory/file1                                     -- File
	targetDir := path.Join(testDirPath, targetDirName)
	operations.CreateDirectory(targetDir, s.T())
	// Create a file in the target directory.
	f1 := operations.CreateFile(path.Join(targetDir, "file"), setup.FilePermission_0600, s.T())
	operations.CloseFileShouldNotThrowError(s.T(), f1)
	// Create an empty subdirectory
	operations.CreateDirectory(path.Join(targetDir, "emptySubDirectory"), s.T())
	// Create a subdirectory with file
	operations.CreateDirectoryWithNFiles(1, path.Join(targetDir, "subDirectory"), "file", s.T())

	// Call Readdirplus to list the directory.
	startTime := time.Now()
	entries, err := fusetesting.ReadDirPlusPicky(targetDir)
	endTime := time.Now()

	require.NoError(s.T(), err, "ReadDirPlusPicky failed")
	expectedEntries := []struct {
		name  string
		isDir bool
		mode  os.FileMode
	}{
		{name: "emptySubDirectory", isDir: true, mode: os.ModeDir | 0755},
		{name: "file", isDir: false, mode: 0644},
		{name: "subDirectory", isDir: true, mode: os.ModeDir | 0755},
	}
	// Verify the entries.
	assert.Equal(s.T(), len(expectedEntries), len(entries), "Number of entries mismatch")
	for i, expected := range expectedEntries {
		entry := entries[i]
		assert.Equal(s.T(), expected.name, entry.Name(), "Name mismatch for entry %d", i)
		assert.Equal(s.T(), expected.isDir, entry.IsDir(), "IsDir mismatch for entry %s", entry.Name())
		assert.Equal(s.T(), expected.mode, entry.Mode(), "Mode mismatch for entry %s", entry.Name())
	}
	// Validate logs to check that ReadDirPlus was called and ReadDir was not.
	// Dentry cache is not enabled, so LookUpInode should be called for
	// parent directory as well as for all the entries.
	validateLogsForReaddirplus(s.T(), setup.LogFile(), false, startTime, endTime)
}

func TestReaddirplusWithoutDentryCacheTest(t *testing.T) {
	ts := &readdirplusWithoutDentryCacheTest{}

	// Run tests for mounted directory if the flag is set.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		suite.Run(t, ts)
		return
	}

	// Run tests.
	ts.flags = []string{"--implicit-dirs", "--experimental-enable-readdirplus"}
	log.Printf("Running tests with flags: %s", ts.flags)
	suite.Run(t, ts)
}
