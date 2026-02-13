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

package negative_stat_cache

import (
	"log"
	"os"
	"path"
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type infiniteNegativeStatCacheTest struct {
	flags []string
	suite.Suite
}

func (s *infiniteNegativeStatCacheTest) SetupTest() {
	testDir := testDirName + setup.GenerateRandomString(5)
	mountGCSFuseAndSetupTestDir(s.flags, testDir)
}

func (s *infiniteNegativeStatCacheTest) TearDownTest() {
	setup.CleanUpDir(testEnv.testDirPath)
	setup.UnmountGCSFuse(testEnv.rootDir)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *infiniteNegativeStatCacheTest) TestInfiniteNegativeStatCache() {
	targetDir := path.Join(testEnv.testDirPath, "explicit_dir")
	// Create test directory
	operations.CreateDirectory(targetDir, s.T())
	targetFile := path.Join(targetDir, "file1.txt")

	// Error should be returned as file does not exist
	_, err := os.OpenFile(targetFile, os.O_RDONLY, os.FileMode(0600))

	assert.NotNil(s.T(), err)
	// Assert the underlying error is File Not Exist
	assert.ErrorContains(s.T(), err, "explicit_dir/file1.txt: no such file or directory")

	// Adding the object with same name
	testDir := path.Base(testEnv.testDirPath)
	client.CreateObjectInGCSTestDir(testEnv.ctx, testEnv.storageClient, testDir, "explicit_dir/file1.txt", "some-content", s.T())

	// Error should be returned again, as call will not be served from GCS due to infinite gcsfuse stat cache
	_, err = os.OpenFile(targetFile, os.O_RDONLY, os.FileMode(0600))

	assert.NotNil(s.T(), err)
	// Assert the underlying error is File Not Exist
	assert.ErrorContains(s.T(), err, "explicit_dir/file1.txt: no such file or directory")
}

// TestAlreadyExistFolder tests the scenario where a folder creation attempt fails
// with EEXIST. The infinite negative cache is essential because LookUpInode must
// return a "not found" error to trigger the subsequent create operation. This occurs
// when a folder is created externally after gcsfuse has cached a negative stat entry for that path.
// The negative cache prevents gcsfuse from seeing the externally created folder,
// leading to an EEXIST error when attempting to create the same folder again.
func (s *infiniteNegativeStatCacheTest) TestAlreadyExistFolder() {
	dirName := "testAlreadyExistFolder"
	dirPath := path.Join(testEnv.testDirPath, dirName)
	testDir := path.Base(testEnv.testDirPath)
	dirPathOnBucket := path.Join(testDir, dirName)
	// Stat should return an error because the directory doesn't exist yet,
	// populating the negative metadata cache.
	_, err := os.Stat(dirPath)
	require.Error(s.T(), err)
	require.True(s.T(), os.IsNotExist(err))
	// Create the directory in the bucket using a different client outside of gcsfuse.
	if setup.IsHierarchicalBucket(testEnv.ctx, testEnv.storageClient) {
		_, err = client.CreateFolderInBucket(testEnv.ctx, testEnv.storageControlClient, dirPathOnBucket)
	} else {
		err = client.CreateObjectOnGCS(testEnv.ctx, testEnv.storageClient, dirPathOnBucket+"/", "")
	}
	require.NoError(s.T(), err)

	// Attempting to create the directory again should fail with EEXIST because the
	// negative stat cache entry persists, causing LookUpInode to return a "not found" error
	// and triggering a directory creation attempt despite the directory already existing in GCS.
	err = os.Mkdir(dirPath, setup.DirPermission_0755)

	assert.ErrorIs(s.T(), err, syscall.EEXIST)
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestInfiniteNegativeStatCacheTest(t *testing.T) {
	ts := &infiniteNegativeStatCacheTest{}

	// Run tests for mounted directory if the flag is set.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		suite.Run(t, ts)
		return
	}

	// Define flag set to run the tests.
	flagsSet := []string{"--metadata-cache-negative-ttl-secs=-1"}

	// Run tests.
	ts.flags = flagsSet
	log.Printf("Running tests with flags: %s", ts.flags)
	suite.Run(t, ts)
}
