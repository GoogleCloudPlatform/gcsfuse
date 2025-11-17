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
	"log"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/client"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/setup"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type infiniteKernelListCacheTest struct {
	flags []string
	suite.Suite
}

func (s *infiniteKernelListCacheTest) SetupTest() {
	mountGCSFuseAndSetupTestDir(s.flags, testDirName)
}

func (s *infiniteKernelListCacheTest) TearDownTest() {
	setup.UnmountGCSFuse(rootDir)
}

////////////////////////////////////////////////////////////////////////
// Test scenarios
////////////////////////////////////////////////////////////////////////

func (s *infiniteKernelListCacheTest) TestKernelListCache_AlwaysCacheHit() {
	targetDir := path.Join(testDirPath, "explicit_dir")
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
	require.NoError(s.T(), err)
	require.Equal(s.T(), 2, len(names1))
	require.Equal(s.T(), "file1.txt", names1[0])
	require.Equal(s.T(), "file2.txt", names1[1])
	err = f.Close()
	require.NoError(s.T(), err)

	// Adding one object to make sure to change the ReadDir() response.
	client.CreateObjectInGCSTestDir(ctx, storageClient, testDirName, path.Join("explicit_dir", "file3.txt"), "", s.T())
	// Waiting for 5 seconds to see if the kernel cache expires.
	time.Sleep(5 * time.Second)

	// Kernel cache will not invalidate since infinite ttl.
	f, err = os.Open(targetDir)
	assert.NoError(s.T(), err)
	names2, err := f.Readdirnames(-1)
	assert.NoError(s.T(), err)

	require.Equal(s.T(), 2, len(names2))
	assert.Equal(s.T(), "file1.txt", names2[0])
	assert.Equal(s.T(), "file2.txt", names2[1])
}

// (a) First ReadDir() will be served from GCSFuse filesystem.
// (b) Second ReadDir() will also be served from GCSFuse filesystem, because of
// addition of new file.
func (s *infiniteKernelListCacheTest) TestKernelListCache_CacheMissOnAdditionOfFile() {
	targetDir := path.Join(testDirPath, "explicit_dir")
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
		assert.NoError(s.T(), f.Close())
	}()
	names1, err := f.Readdirnames(-1)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 2, len(names1))
	require.Equal(s.T(), "file1.txt", names1[0])
	require.Equal(s.T(), "file2.txt", names1[1])
	err = f.Close()
	require.NoError(s.T(), err)

	// Adding one object to make sure to change the ReadDir() response.
	client.CreateObjectInGCSTestDir(ctx, storageClient, testDirName, path.Join("explicit_dir", "file3.txt"), "", s.T())

	// Ideally no invalidation since infinite ttl, but creation of a new file inside
	// directory evicts the list cache for that directory.
	fNew, err := os.Create(path.Join(targetDir, "file4.txt"))
	require.NoError(s.T(), err)
	assert.NotNil(s.T(), fNew)
	defer func() {
		assert.NoError(s.T(), fNew.Close())
		assert.NoError(s.T(), os.Remove(path.Join(targetDir, "file4.txt")))
	}()

	f, err = os.Open(path.Join(testDirPath, "explicit_dir"))
	assert.NoError(s.T(), err)
	names2, err := f.Readdirnames(-1)

	assert.NoError(s.T(), err)
	require.Equal(s.T(), 4, len(names2))
	assert.Equal(s.T(), "file1.txt", names2[0])
	assert.Equal(s.T(), "file2.txt", names2[1])
	assert.Equal(s.T(), "file3.txt", names2[2])
	assert.Equal(s.T(), "file4.txt", names2[3])
}

// (a) First ReadDir() will be served from GCSFuse filesystem.
// (b) Second ReadDir() will also be served from GCSFuse filesystem, because of
// deletion of new file.
func (s *infiniteKernelListCacheTest) TestKernelListCache_CacheMissOnDeletionOfFile() {
	targetDir := path.Join(testDirPath, "explicit_dir")
	operations.CreateDirectory(targetDir, s.T())
	// Create test data
	f1 := operations.CreateFile(path.Join(targetDir, "file1.txt"), setup.FilePermission_0600, s.T())
	operations.CloseFileShouldNotThrowError(s.T(), f1)
	f2 := operations.CreateFile(path.Join(targetDir, "file2.txt"), setup.FilePermission_0600, s.T())
	operations.CloseFileShouldNotThrowError(s.T(), f2)

	// First read, kernel will cache the dir response.
	f, err := os.Open(targetDir)
	assert.NoError(s.T(), err)
	defer func() {
		assert.NoError(s.T(), f.Close())
	}()
	names1, err := f.Readdirnames(-1)
	assert.NoError(s.T(), err)
	require.Equal(s.T(), 2, len(names1))
	require.Equal(s.T(), "file1.txt", names1[0])
	require.Equal(s.T(), "file2.txt", names1[1])
	err = f.Close()
	assert.NoError(s.T(), err)

	// Adding one object to make sure to change the ReadDir() response.
	client.CreateObjectInGCSTestDir(ctx, storageClient, testDirName, path.Join("explicit_dir", "file3.txt"), "", s.T())

	// Ideally no invalidation since infinite ttl, but deletion of file inside
	// directory evicts the list cache for that directory.
	err = os.Remove(path.Join(targetDir, "file2.txt"))
	require.NoError(s.T(), err)

	f, err = os.Open(targetDir)
	assert.NoError(s.T(), err)
	names2, err := f.Readdirnames(-1)

	assert.NoError(s.T(), err)
	require.Equal(s.T(), 2, len(names2))
	assert.Equal(s.T(), "file1.txt", names2[0])
	assert.Equal(s.T(), "file3.txt", names2[1])
}

// (a) First ReadDir() will be served from GCSFuse filesystem.
// (b) Second ReadDir() will also be served from GCSFuse filesystem, because of
// file rename.
func (s *infiniteKernelListCacheTest) TestKernelListCache_CacheMissOnFileRename() {
	targetDir := path.Join(testDirPath, "explicit_dir")
	operations.CreateDirectory(targetDir, s.T())
	// Create test data
	f1 := operations.CreateFile(path.Join(targetDir, "file1.txt"), setup.FilePermission_0600, s.T())
	operations.CloseFileShouldNotThrowError(s.T(), f1)
	f2 := operations.CreateFile(path.Join(targetDir, "file2.txt"), setup.FilePermission_0600, s.T())
	operations.CloseFileShouldNotThrowError(s.T(), f2)

	// First read, kernel will cache the dir response.
	f, err := os.Open(targetDir)
	assert.NoError(s.T(), err)
	defer func() {
		assert.NoError(s.T(), f.Close())
	}()
	names1, err := f.Readdirnames(-1)
	assert.NoError(s.T(), err)
	require.Equal(s.T(), 2, len(names1))
	require.Equal(s.T(), "file1.txt", names1[0])
	require.Equal(s.T(), "file2.txt", names1[1])
	err = f.Close()
	assert.NoError(s.T(), err)

	// Adding one object to make sure to change the ReadDir() response.
	client.CreateObjectInGCSTestDir(ctx, storageClient, testDirName, path.Join("explicit_dir", "file3.txt"), "", s.T())

	// Ideally no invalidation since infinite ttl, but rename of a file inside
	// directory evicts the list cache for that directory.
	err = os.Rename(path.Join(targetDir, "file2.txt"), path.Join(targetDir, "renamed_file2.txt"))
	require.NoError(s.T(), err)

	f, err = os.Open(targetDir)
	require.NoError(s.T(), err)
	names2, err := f.Readdirnames(-1)

	assert.NoError(s.T(), err)
	require.Equal(s.T(), 3, len(names2))
	assert.Equal(s.T(), "file1.txt", names2[0])
	assert.Equal(s.T(), "file3.txt", names2[1])
	assert.Equal(s.T(), "renamed_file2.txt", names2[2])
}

// explicit_dir/file1.txt
// explicit_dir/sub_dir/file2.txt
// explicit_dir/sub_dir/file3.txt
//
// ls explicit_dir
// file1.txt, sub_dir
//
// ls explicit_dir/sub_dir
// file2.txt, file3.txt
//
// Create file4.txt  in explicit_dir/sub_dir with gcsfuse
// Create file5.txt  in explicit_dir/sub_dir outside of  gcsfuse
// Create file6.txt  in explicit_dir with gcsfuse
//
// Since file4 was created through the kernel, the subdirectory cache was invalidated,
// but the parent cache remained persistent.
//
// ls explicit_dir
// file1.txt, sub_dir
//
// ls explicit_dir/sub_dir
// file2.txt, file3.txt, file4.txt, file5.txt
func (s *infiniteKernelListCacheTest) TestKernelListCache_EvictCacheEntryOfOnlyDirectParent() {
	targetDir := path.Join(testDirPath, "explicit_dir")
	operations.CreateDirectory(targetDir, s.T())
	subDir := path.Join(targetDir, "sub_dir")
	operations.CreateDirectory(subDir, s.T())
	// Create test files
	f1 := operations.CreateFile(path.Join(targetDir, "file1.txt"), setup.FilePermission_0600, s.T())
	operations.CloseFileShouldNotThrowError(s.T(), f1)
	f2 := operations.CreateFile(path.Join(subDir, "file2.txt"), setup.FilePermission_0600, s.T())
	operations.CloseFileShouldNotThrowError(s.T(), f2)
	f3 := operations.CreateFile(path.Join(subDir, "file3.txt"), setup.FilePermission_0600, s.T())
	operations.CloseFileShouldNotThrowError(s.T(), f3)
	// Initial read of parent directory (caches results)
	f, err := os.Open(targetDir)
	require.NoError(s.T(), err)
	names1, err := f.Readdirnames(-1) // Read all filenames
	require.NoError(s.T(), err)
	require.NoError(s.T(), f.Close())
	require.Equal(s.T(), 2, len(names1))
	require.Equal(s.T(), "file1.txt", names1[0])
	require.Equal(s.T(), "sub_dir", names1[1])
	// Initial read of subdirectory (caches results)
	f, err = os.Open(subDir)
	require.NoError(s.T(), err)
	names2, err := f.Readdirnames(-1)
	require.NoError(s.T(), err)
	require.NoError(s.T(), f.Close())
	require.Equal(s.T(), 2, len(names2))
	assert.Equal(s.T(), "file2.txt", names2[0])
	assert.Equal(s.T(), "file3.txt", names2[1])
	// Add a new file to the subdirectory to trigger a cache invalidation scenario
	fNew, err := os.Create(path.Join(subDir, "file4.txt"))
	require.NoError(s.T(), err)
	require.NoError(s.T(), fNew.Close())
	client.CreateObjectInGCSTestDir(ctx, storageClient, testDirName,
		path.Join("explicit_dir", "sub_dir", "file5.txt"), "", s.T())
	// Add a new file to the parent directory through the client to verify that the
	// cache is not invalidated in the case of the parent.
	client.CreateObjectInGCSTestDir(ctx, storageClient, testDirName,
		path.Join("explicit_dir", "file6.txt"), "", s.T())

	// Re-read parent directory (should still use the cache and NOT show the change in the sub-dir)
	f1, err = os.Open(targetDir)
	require.NoError(s.T(), err)
	names1, err = f1.Readdirnames(-1)
	require.NoError(s.T(), err)
	require.NoError(s.T(), f1.Close())
	// Re-read subdirectory (cache should be invalidated and show the new file)
	f2, err = os.Open(subDir)
	require.NoError(s.T(), err)
	names2, err = f2.Readdirnames(-1)
	require.NoError(s.T(), f2.Close())
	require.NoError(s.T(), err)

	// This is expected to be 2 as it is reading from cache for parent directory
	require.Equal(s.T(), 2, len(names1))
	assert.Equal(s.T(), "file1.txt", names1[0])
	assert.Equal(s.T(), "sub_dir", names1[1])
	// Cache invalidated, expect 4 items now as call went to GCS
	require.Equal(s.T(), 4, len(names2))
	assert.Equal(s.T(), "file2.txt", names2[0])
	assert.Equal(s.T(), "file3.txt", names2[1])
	assert.Equal(s.T(), "file4.txt", names2[2])
	assert.Equal(s.T(), "file5.txt", names2[3])
}

// (a) First ReadDir() will be served from GCSFuse filesystem.
// (b) Second ReadDir() will also be served from GCSFuse filesystem, because of
// addition of new directory.
func (s *infiniteKernelListCacheTest) TestKernelListCache_CacheMissOnAdditionOfDirectory() {
	targetDir := path.Join(testDirPath, "explicit_dir")
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
		assert.NoError(s.T(), f.Close())
	}()
	names1, err := f.Readdirnames(-1)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 2, len(names1))
	require.Equal(s.T(), "file1.txt", names1[0])
	require.Equal(s.T(), "file2.txt", names1[1])
	err = f.Close()
	require.NoError(s.T(), err)
	// Adding one object to make sure to change the ReadDir() response.
	client.CreateObjectInGCSTestDir(ctx, storageClient, testDirName, path.Join("explicit_dir", "file3.txt"), "", s.T())
	// Ideally no invalidation since infinite ttl, but creation of a new directory inside
	// directory evicts the list cache for that directory.
	err = os.Mkdir(path.Join(targetDir, "sub_dir"), setup.DirPermission_0755)
	require.NoError(s.T(), err)

	f, err = os.Open(targetDir)
	assert.Nil(s.T(), err)
	names2, err := f.Readdirnames(-1)

	assert.Nil(s.T(), err)
	require.Equal(s.T(), 4, len(names2))
	assert.Equal(s.T(), "file1.txt", names2[0])
	assert.Equal(s.T(), "file2.txt", names2[1])
	assert.Equal(s.T(), "file3.txt", names2[2])
	assert.Equal(s.T(), "sub_dir", names2[3])
}

// (a) First ReadDir() will be served from GCSFuse filesystem.
// (b) Second ReadDir() will also be served from GCSFuse filesystem, because of
// deletion of directory.
func (s *infiniteKernelListCacheTest) TestKernelListCache_CacheMissOnDeletionOfDirectory() {
	targetDir := path.Join(testDirPath, "explicit_dir")
	operations.CreateDirectory(targetDir, s.T())
	// Create test data
	f1 := operations.CreateFile(path.Join(targetDir, "file1.txt"), setup.FilePermission_0600, s.T())
	operations.CloseFileShouldNotThrowError(s.T(), f1)
	f2 := operations.CreateFile(path.Join(targetDir, "file2.txt"), setup.FilePermission_0600, s.T())
	operations.CloseFileShouldNotThrowError(s.T(), f2)
	err := os.Mkdir(path.Join(targetDir, "sub_dir"), setup.DirPermission_0755)
	require.NoError(s.T(), err)
	// First read, kernel will cache the dir response.
	f, err := os.Open(targetDir)
	require.NoError(s.T(), err)
	defer func() {
		assert.NoError(s.T(), f.Close())
	}()
	names1, err := f.Readdirnames(-1)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 3, len(names1))
	require.Equal(s.T(), "file1.txt", names1[0])
	require.Equal(s.T(), "file2.txt", names1[1])
	require.Equal(s.T(), "sub_dir", names1[2])
	err = f.Close()
	require.NoError(s.T(), err)
	// Adding one object to make sure to change the ReadDir() response.
	client.CreateObjectInGCSTestDir(ctx, storageClient, testDirName, path.Join("explicit_dir", "file3.txt"), "", s.T())
	// Ideally no invalidation since infinite ttl, but creation of a new file inside
	// directory evicts the list cache for that directory.
	err = os.Remove(path.Join(targetDir, "sub_dir"))
	require.Nil(s.T(), err)

	f, err = os.Open(targetDir)
	assert.Nil(s.T(), err)
	names2, err := f.Readdirnames(-1)

	assert.Nil(s.T(), err)
	require.Equal(s.T(), 3, len(names2))
	assert.Equal(s.T(), "file1.txt", names2[0])
	assert.Equal(s.T(), "file2.txt", names2[1])
	assert.Equal(s.T(), "file3.txt", names2[2])
}

// (a) First ReadDir() will be served from GCSFuse filesystem.
// (b) Second ReadDir() will also be served from GCSFuse filesystem, because of
// directory rename.
func (s *infiniteKernelListCacheTest) TestKernelListCache_CacheMissOnDirectoryRename() {
	targetDir := path.Join(testDirPath, "explicit_dir")
	operations.CreateDirectory(targetDir, s.T())
	// Create test data
	f1 := operations.CreateFile(path.Join(targetDir, "file1.txt"), setup.FilePermission_0600, s.T())
	operations.CloseFileShouldNotThrowError(s.T(), f1)
	f2 := operations.CreateFile(path.Join(targetDir, "file2.txt"), setup.FilePermission_0600, s.T())
	operations.CloseFileShouldNotThrowError(s.T(), f2)
	err := os.Mkdir(path.Join(targetDir, "sub_dir"), setup.DirPermission_0755)
	require.NoError(s.T(), err)
	// First read, kernel will cache the dir response.
	f, err := os.Open(targetDir)
	require.NoError(s.T(), err)
	defer func() {
		assert.NoError(s.T(), f.Close())
	}()
	names1, err := f.Readdirnames(-1)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 3, len(names1))
	require.Equal(s.T(), "file1.txt", names1[0])
	require.Equal(s.T(), "file2.txt", names1[1])
	require.Equal(s.T(), "sub_dir", names1[2])
	err = f.Close()
	require.NoError(s.T(), err)
	// Adding one object to make sure to change the ReadDir() response.
	client.CreateObjectInGCSTestDir(ctx, storageClient, testDirName, path.Join("explicit_dir", "file3.txt"), "", s.T())
	// Ideally no invalidation since infinite ttl, but creation of a new file inside
	// directory evicts the list cache for that directory.
	err = os.Rename(path.Join(targetDir, "sub_dir"), path.Join(targetDir, "renamed_sub_dir"))
	require.Nil(s.T(), err)
	defer func() {
		assert.Nil(s.T(), os.Remove(path.Join(targetDir, "renamed_sub_dir")))
	}()

	f, err = os.Open(targetDir)
	assert.Nil(s.T(), err)
	names2, err := f.Readdirnames(-1)

	assert.Nil(s.T(), err)
	require.Equal(s.T(), 4, len(names2))
	assert.Equal(s.T(), "file1.txt", names2[0])
	assert.Equal(s.T(), "file2.txt", names2[1])
	assert.Equal(s.T(), "file3.txt", names2[2])
	assert.Equal(s.T(), "renamed_sub_dir", names2[3])
}

////////////////////////////////////////////////////////////////////////
// Test Function (Runs once before all tests)
////////////////////////////////////////////////////////////////////////

func TestInfiniteKernelListCacheTest(t *testing.T) {
	ts := &infiniteKernelListCacheTest{}

	// Run tests for mounted directory if the flag is set.
	if setup.AreBothMountedDirectoryAndTestBucketFlagsSet() {
		suite.Run(t, ts)
		return
	}

	flagsSet := [][]string{
		{"--kernel-list-cache-ttl-secs=-1"},
	}

	// Run tests.
	for _, flags := range flagsSet {
		ts.flags = flags
		log.Printf("Running tests with flags: %s", ts.flags)

		suite.Run(t, ts)
	}
}
