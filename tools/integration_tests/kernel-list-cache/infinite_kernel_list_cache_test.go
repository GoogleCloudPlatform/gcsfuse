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
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/test_setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type infiniteKernelListCacheTest struct {
	flags []string
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

func (s *infiniteKernelListCacheTest) TestKernelListCache_AlwaysCacheHit(t *testing.T) {
	operations.CreateDirectory(path.Join(testDirPath, "explicit_dir"), t)
	// Create test data
	f1 := operations.CreateFile(path.Join(testDirPath, "explicit_dir", "file1.txt"), setup.FilePermission_0600, t)
	defer operations.CloseFile(f1)
	f2 := operations.CreateFile(path.Join(testDirPath, "explicit_dir", "file2.txt"), setup.FilePermission_0600, t)
	defer operations.CloseFile(f2)

	// First read, kernel will cache the dir response.
	f, err := os.Open(path.Join(testDirPath, "explicit_dir"))
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
	f3 := operations.CreateFile(path.Join(testDirPath, "explicit_dir", "file3.txt"), setup.FilePermission_0600, t)
	defer operations.CloseFile(f3)
	// Advancing time by 5 years (157800000 seconds).
	time.Sleep(157800000 * time.Second)

	// No invalidation since infinite ttl.
	f, err = os.Open(path.Join(testDirPath, "explicit_dir"))
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
		{"--kernel-list-cache-ttl-secs=-1"},
	}

	// Run tests.
	for _, flags := range flagsSet {
		ts.flags = flags
		log.Printf("Running tests with flags: %s", ts.flags)
		test_setup.RunTests(t, ts)
	}
}
