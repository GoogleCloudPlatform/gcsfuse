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

package kernel_list_cache

import (
	"log"
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/client"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/test_setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type infiniteKernelListCacheTest struct {
	flags []string
	suite.Suite
}

func (s *infiniteKernelListCacheTest) Setup(t *testing.T) {
	mountGCSFuseAndSetupTestDir(s.flags, ctx, storageClient, testDirName)
}

func (s *infiniteKernelListCacheTest) Teardown(t *testing.T) {
	setup.UnmountGCSFuse(rootDir)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

// Create initial directory structure for kernel-list-cache test.
// bucket
//
//		file1.txt
//		file2.txt
//	 explicitDir/
//		explicitDir/file1.txt
//		explicitDir/file2.txt
func createDirectoryStructure(t *testing.T) {
	// Create explicitDir structure
	explicitDir := path.Join(testDirPath, "explicitDir")
	operations.CreateDirectory(explicitDir, t)
	operations.CreateFileOfSize(5, path.Join(explicitDir, "file1.txt"), t)
	operations.CreateFileOfSize(10, path.Join(explicitDir, "file2.txt"), t)
}

// (a) First ReadDir() will be served from GCSFuse filesystem.
// (b) Second ReadDir() will also be served from GCSFuse filesystem.
func (s *infiniteKernelListCacheTest) Test_CacheHit(t *testing.T) {
	createDirectoryStructure(t)
	// First read, kernel will cache the dir response.
	f, err := os.Open(path.Join(testDirPath, "explicitDir"))
	assert.Nil(t, err)
	defer func() {
		assert.Nil(t, f.Close())
	}()
	names1, err := f.Readdirnames(-1)
	assert.Nil(t, err)
	require.Equal(t, 2, len(names1))
	assert.Equal(t, "file1.txt", names1[0])
	assert.Equal(t, "file2.txt", names1[1])
	err = f.Close()
	assert.Nil(t, err)
	// Adding one object to make sure to change the ReadDir() response.
	client.CreateObjectInGCSTestDir(ctx, storageClient, mountDir, "explicitDir/file3.txt", "123456", t)
	defer client.DeleteObjectOnGCS(ctx, storageClient, "explicitDir/file3.txt")

	// No invalidation since infinite ttl.
	f, err = os.Open(path.Join(testDirPath, "explicitDir"))
	assert.Nil(t, err)
	names2, err := f.Readdirnames(-1)

	assert.Nil(t, err)
	require.Equal(t, 2, len(names2))
	assert.Equal(t, "file1.txt", names2[0])
	assert.Equal(t, "file2.txt", names2[1])
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestInfiniteKernelListCacheTest(t *testing.T) {
	ts := &infiniteKernelListCacheTest{}

	// Run tests for mounted directory if the flag is set.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		test_setup.RunTests(t, ts)
		return
	}

	// Define flag set to run the tests.
	flagsSet := [][]string{
		{"--kernel-list-cache-ttl-secs=-1", "--implicit-dirs"},
	}

	// Run tests.
	for _, flags := range flagsSet {
		ts.flags = flags
		log.Printf("Running tests with flags: %s", ts.flags)
		test_setup.RunTests(t, ts)
	}
}
