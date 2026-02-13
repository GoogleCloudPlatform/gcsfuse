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
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type disabledNegativeStatCacheTest struct {
	flags   []string
	testDir string
	suite.Suite
}

func (s *disabledNegativeStatCacheTest) SetupTest() {
	s.testDir = testDirName + setup.GenerateRandomString(5)
	mountGCSFuseAndSetupTestDir(s.flags, s.testDir)
}

func (s *disabledNegativeStatCacheTest) TearDownTest() {
	setup.UnmountGCSFuse(testEnv.rootDir)
	setup.CleanupDirectoryOnGCS(testEnv.ctx, testEnv.storageClient, path.Join(setup.TestBucket(), s.testDir))
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *disabledNegativeStatCacheTest) TestNegativeStatCacheDisabled() {
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
	client.CreateObjectInGCSTestDir(testEnv.ctx, testEnv.storageClient, s.testDir, "explicit_dir/file1.txt", "some-content", s.T())

	// File should be returned, as call will be served from GCS and gcsfuse should not return from cache
	f, err := os.OpenFile(targetFile, os.O_RDONLY, os.FileMode(0600))

	//Assert File is found
	assert.NoError(s.T(), err)
	assert.Contains(s.T(), f.Name(), "explicit_dir/file1.txt")
	assert.Nil(s.T(), f.Close())
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestDisabledNegativeStatCacheTest(t *testing.T) {
	ts := &disabledNegativeStatCacheTest{}

	// Run tests for mounted directory if the flag is set.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		suite.Run(t, ts)
		return
	}

	// Define flag set to run the tests.
	flagsSet := []string{"--metadata-cache-negative-ttl-secs=0"}

	// Run tests.
	ts.flags = flagsSet
	log.Printf("Running tests with flags: %s", ts.flags)
	suite.Run(t, ts)
}
