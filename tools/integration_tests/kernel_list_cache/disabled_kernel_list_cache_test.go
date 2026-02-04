// Copyright 2024 Google LLC
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

package kernel_list_cache

import (
	"fmt"
	"log"
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type disabledKernelListCacheTest struct {
	flags []string
	suite.Suite
}

func (s *disabledKernelListCacheTest) SetupSuite() {
	setup.MountGCSFuseWithGivenMountWithConfigFunc(testEnv.cfg, s.flags, mountFunc)
	setup.SetMntDir(mountDir)
}

func (s *disabledKernelListCacheTest) TearDownSuite() {
	setup.UnmountGCSFuseWithConfig(testEnv.cfg)
}

func (s *disabledKernelListCacheTest) SetupTest() {
	var testDirName = fmt.Sprintf("%s-%d-%s", testDirNamePrefix, os.Getpid(), setup.GenerateRandomString(5))
	testEnv.testDirPath = setup.SetupTestDirectory(testDirName)
}

func (s *disabledKernelListCacheTest) TearDownTest() {
	setup.SaveGCSFuseLogFileInCaseOfFailure(s.T())
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *disabledKernelListCacheTest) TestKernelListCache_AlwaysCacheMiss() {
	targetDir := path.Join(testEnv.testDirPath, "explicit_dir")
	operations.CreateDirectory(targetDir, s.T())
	// Create test data
	f1 := operations.CreateFile(path.Join(targetDir, "file1.txt"), setup.FilePermission_0600, s.T())
	operations.CloseFileShouldNotThrowError(s.T(), f1)
	f2 := operations.CreateFile(path.Join(targetDir, "file2.txt"), setup.FilePermission_0600, s.T())
	operations.CloseFileShouldNotThrowError(s.T(), f2)

	// First read, kernel will cache the dir response.
	f, err := os.Open(targetDir)
	require.NoError(s.T(), err)
	defer func() {
		assert.Nil(s.T(), f.Close())
	}()
	names1, err := f.Readdirnames(-1)
	assert.Nil(s.T(), err)
	require.Equal(s.T(), 2, len(names1))
	require.Equal(s.T(), "file1.txt", names1[0])
	require.Equal(s.T(), "file2.txt", names1[1])
	err = f.Close()
	require.NoError(s.T(), err)
	// Adding one object to make sure to change the ReadDir() response.
	client.CreateObjectInGCSTestDir(testEnv.ctx, testEnv.storageClient, testDirName, path.Join("explicit_dir", "file3.txt"), "", s.T())

	// Zero ttl, means readdir will always be served from gcsfuse.
	f, err = os.Open(targetDir)
	assert.NoError(s.T(), err)
	names2, err := f.Readdirnames(-1)
	assert.NoError(s.T(), err)

	require.Equal(s.T(), 3, len(names2))
	assert.Equal(s.T(), "file1.txt", names2[0])
	assert.Equal(s.T(), "file2.txt", names2[1])
	assert.Equal(s.T(), "file3.txt", names2[2])
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestDisabledKernelListCacheTest(t *testing.T) {
	ts := &disabledKernelListCacheTest{}

	// Run tests for mounted directory if the flag is set. This assumes that run flag is properly passed by GKE team as per the config.
	if testEnv.cfg.GKEMountedDirectory != "" && testEnv.cfg.TestBucket != "" {
		suite.Run(t, ts)
		return
	}

	// Run tests for GCE environment otherwise.
	flagsSet := setup.BuildFlagSets(*testEnv.cfg, testEnv.bucketType, t.Name())
	for _, ts.flags = range flagsSet {
		log.Printf("Running tests with flags: %s", ts.flags)
		suite.Run(t, ts)
	}
}
