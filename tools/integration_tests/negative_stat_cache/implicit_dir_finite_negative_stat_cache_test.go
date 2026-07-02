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
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type implicitDirFiniteNegativeStatCacheTest struct {
	flags   []string
	testDir string
	suite.Suite
}

func (s *implicitDirFiniteNegativeStatCacheTest) SetupSuite() {
	setup.MountGCSFuseWithGivenMountWithConfigFunc(testEnv.cfg, s.flags, testEnv.mountFunc)
	setup.SetMntDir(testEnv.mountDir)
}

func (s *implicitDirFiniteNegativeStatCacheTest) TearDownSuite() {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
}

func (s *implicitDirFiniteNegativeStatCacheTest) SetupTest() {
	s.testDir = testDirName + setup.GenerateRandomString(5)
	testEnv.testDirPath = setup.SetupTestDirectory(s.testDir)
}

func (s *implicitDirFiniteNegativeStatCacheTest) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *implicitDirFiniteNegativeStatCacheTest) TestImplicitDirFiniteNegativeStatCache() {
	targetDir := path.Join(testEnv.testDirPath, "implicit_dir")

	// Error should be returned as the implicit directory doesn't exist
	_, err := os.Stat(targetDir)

	assert.NotNil(s.T(), err)
	// Assert the underlying error is File Not Exist
	assert.ErrorContains(s.T(), err, "no such file or directory")

	// Adding an object inside the path to create an implicit directory
	client.CreateObjectInGCSTestDir(testEnv.ctx, testEnv.storageClient, s.testDir, path.Join("implicit_dir", "file1.txt"), "some-content", s.T())

	// Error should be returned again, as call will not be served from GCS due to finite gcsfuse stat cache
	_, err = os.Stat(targetDir)

	assert.NotNil(s.T(), err)
	// Assert the underlying error is File Not Exist
	assert.ErrorContains(s.T(), err, "no such file or directory")

	//Wait for Cache to expire
	time.Sleep(5 * time.Second)

	// Directory should be returned, as call will be served from GCS and gcsfuse should not return from cache
	fileInfo, err := os.Stat(targetDir)

	//Assert Directory is found and it is a directory
	assert.NoError(s.T(), err)
	assert.True(s.T(), fileInfo.IsDir())
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestImplicitDirFiniteNegativeStatCacheTest(t *testing.T) {
	ts := &implicitDirFiniteNegativeStatCacheTest{}

	// Run tests for mounted directory if the flag is set.
	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		suite.Run(t, ts)
		return
	}

	// Define flag set to run the tests.
	flagsSet := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())

	// Run tests.
	for _, ts.flags = range flagsSet {
		log.Printf("Running tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}
