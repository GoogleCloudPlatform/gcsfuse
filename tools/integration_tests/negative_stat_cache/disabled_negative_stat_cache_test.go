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

func (s *disabledNegativeStatCacheTest) SetupSuite() {
	setup.MountGCSFuseWithGivenMountWithConfigFunc(testEnv.cfg, s.flags, testEnv.mountFunc)
	setup.SetMntDir(testEnv.mountDir)
}

func (s *disabledNegativeStatCacheTest) TearDownSuite() {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
}

func (s *disabledNegativeStatCacheTest) SetupTest() {
	s.testDir = testDirName + setup.GenerateRandomString(5)
	testEnv.testDirPath = setup.SetupTestDirectory(s.testDir)
}

func (s *disabledNegativeStatCacheTest) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
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

func (s *disabledNegativeStatCacheTest) TestNegativeStatCacheDisabled_ImplicitDirectory() {
	if !isImplicitDirsEnabled(s.flags) {
		s.T().Skip("Skipping implicit directory test as --implicit-dirs flag is not enabled.")
	}

	implicitDir := path.Join(testEnv.testDirPath, "implicit_dir")
	targetFile := path.Join(implicitDir, "file1.txt")

	// Stat of non-existent implicit dir should fail.
	_, err := os.Stat(implicitDir)
	assert.Error(s.T(), err)
	assert.True(s.T(), os.IsNotExist(err))

	// Open of non-existent file in implicit dir should fail.
	_, err = os.OpenFile(targetFile, os.O_RDONLY, os.FileMode(0600))
	assert.Error(s.T(), err)
	assert.True(s.T(), os.IsNotExist(err))

	// Create object in GCS directly under implicit_dir path.
	client.CreateObjectInGCSTestDir(testEnv.ctx, testEnv.storageClient, s.testDir, "implicit_dir/file1.txt", "some-content", s.T())

	// Since negative stat cache is disabled (TTL = 0), GCSFuse should not serve from negative cache.
	// Stat on implicit dir should now succeed.
	fi, err := os.Stat(implicitDir)
	assert.NoError(s.T(), err)
	assert.True(s.T(), fi.IsDir())

	// File should be found and readable.
	f, err := os.OpenFile(targetFile, os.O_RDONLY, os.FileMode(0600))
	assert.NoError(s.T(), err)
	assert.Contains(s.T(), f.Name(), "implicit_dir/file1.txt")
	assert.Nil(s.T(), f.Close())
}

func (s *disabledNegativeStatCacheTest) TestNegativeStatCacheDisabled_ImplicitDirsDisabled() {
	if isImplicitDirsEnabled(s.flags) || isHNSBucket() {
		s.T().Skip("Skipping test: requires flat bucket with implicit-dirs disabled.")
	}

	implicitDir := path.Join(testEnv.testDirPath, "implicit_dir")
	targetFile := path.Join(implicitDir, "file1.txt")

	// Stat of non-existent implicit dir should fail.
	_, err := os.Stat(implicitDir)
	assert.Error(s.T(), err)
	assert.True(s.T(), os.IsNotExist(err))

	// Open of non-existent file should fail.
	_, err = os.OpenFile(targetFile, os.O_RDONLY, os.FileMode(0600))
	assert.Error(s.T(), err)
	assert.True(s.T(), os.IsNotExist(err))

	// Create object in GCS directly under implicit_dir path.
	client.CreateObjectInGCSTestDir(testEnv.ctx, testEnv.storageClient, s.testDir, "implicit_dir/file1.txt", "some-content", s.T())

	// Since --implicit-dirs is disabled, stat on implicit dir still fails even though file exists in GCS.
	_, err = os.Stat(implicitDir)
	assert.Error(s.T(), err)
	assert.True(s.T(), os.IsNotExist(err))

	// Opening the file directly should succeed because file negative cache is disabled (TTL = 0).
	f, err := os.OpenFile(targetFile, os.O_RDONLY, os.FileMode(0600))
	assert.NoError(s.T(), err)
	assert.Contains(s.T(), f.Name(), "implicit_dir/file1.txt")
	assert.Nil(s.T(), f.Close())
}

func (s *disabledNegativeStatCacheTest) TestNegativeStatCacheDisabled_HNSFolder() {
	if !isHNSBucket() {
		s.T().Skip("Skipping test: requires HNS bucket.")
	}

	hnsDirName := "hns_dir"
	hnsDirPath := path.Join(testEnv.testDirPath, hnsDirName)
	hnsDirPathOnBucket := path.Join(s.testDir, hnsDirName)

	// Stat of non-existent HNS folder should fail.
	_, err := os.Stat(hnsDirPath)
	assert.Error(s.T(), err)
	assert.True(s.T(), os.IsNotExist(err))

	// Create folder out-of-band on HNS bucket using control client.
	_, err = client.CreateFolderInBucket(testEnv.ctx, testEnv.storageControlClient, hnsDirPathOnBucket)
	assert.NoError(s.T(), err)

	// Since negative stat cache is disabled (TTL = 0), stat on HNS folder should succeed immediately.
	fi, err := os.Stat(hnsDirPath)
	assert.NoError(s.T(), err)
	assert.True(s.T(), fi.IsDir())
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestDisabledNegativeStatCacheTest(t *testing.T) {
	ts := &disabledNegativeStatCacheTest{}

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
