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

// Please refer kernel_list_cache_test.go for the documentation.

package fs_test

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type KernelListCacheTestWithInfiniteTtl struct {
	suite.Suite
	fsTest
	KernelListCacheTestCommon
}

func (t *KernelListCacheTestWithInfiniteTtl) SetupSuite() {
	t.serverCfg.ImplicitDirectories = true
	t.serverCfg.MountConfig = &config.MountConfig{
		FileSystemConfig: config.FileSystemConfig{
			KernelListCacheTtlSeconds: -1,
		},
		MetadataCacheConfig: config.MetadataCacheConfig{
			TtlInSeconds: 0,
		},
	}
	t.serverCfg.RenameDirLimit = 10
	t.fsTest.SetUpTestSuite()
}

func TestKernelListCacheTestInfiniteTtlSuite(t *testing.T) {
	suite.Run(t, new(KernelListCacheTestWithInfiniteTtl))
}

// (a) First ReadDir() will be served from GCSFuse filesystem.
// (b) Second ReadDir() will also be served from GCSFuse filesystem.
func (t *KernelListCacheTestWithInfiniteTtl) TestKernelListCache_AlwaysCacheHit() {
	// First read, kernel will cache the dir response.
	f, err := os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
	defer func() {
		assert.Nil(t.T(), f.Close())
	}()
	names1, err := f.Readdirnames(-1)
	assert.Nil(t.T(), err)
	require.Equal(t.T(), 2, len(names1))
	assert.Equal(t.T(), "file1.txt", names1[0])
	assert.Equal(t.T(), "file2.txt", names1[1])
	err = f.Close()
	assert.Nil(t.T(), err)
	// Adding one object to make sure to change the ReadDir() response.
	assert.Nil(t.T(), t.createObjects(map[string]string{
		"explicitDir/file3.txt": "123456",
	}))
	defer t.deleteObjectOrFail("explicitDir/file3.txt")
	// Advancing time by 5 years (157800000 seconds).
	cacheClock.AdvanceTime(157800000 * time.Second)

	// No invalidation since infinite ttl.
	f, err = os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
	names2, err := f.Readdirnames(-1)

	assert.Nil(t.T(), err)
	require.Equal(t.T(), 2, len(names2))
	assert.Equal(t.T(), "file1.txt", names2[0])
	assert.Equal(t.T(), "file2.txt", names2[1])
}

// (a) First ReadDir() will be served from GCSFuse filesystem.
// (b) Second ReadDir() will also be served from GCSFuse filesystem, because of
// addition of new file.
func (t *KernelListCacheTestWithInfiniteTtl) TestKernelListCache_CacheMissOnAdditionOfFile() {
	// First read, kernel will cache the dir response.
	f, err := os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
	defer func() {
		assert.Nil(t.T(), f.Close())
	}()
	names1, err := f.Readdirnames(-1)
	assert.Nil(t.T(), err)
	require.Equal(t.T(), 2, len(names1))
	assert.Equal(t.T(), "file1.txt", names1[0])
	assert.Equal(t.T(), "file2.txt", names1[1])
	err = f.Close()
	assert.Nil(t.T(), err)
	// Adding one object to make sure to change the ReadDir() response.
	assert.Nil(t.T(), t.createObjects(map[string]string{
		"explicitDir/file3.txt": "123456",
	}))
	defer t.deleteObjectOrFail("explicitDir/file3.txt")
	// Advancing time by 5 years (157800000 seconds).
	cacheClock.AdvanceTime(157800000 * time.Second)
	// Ideally no invalidation since infinite ttl, but creation of a new file inside
	// directory evicts the list cache for that directory.
	fNew, err := os.Create(path.Join(mntDir, "explicitDir/file4.txt"))
	require.Nil(t.T(), err)
	assert.NotNil(t.T(), fNew)
	defer func() {
		assert.Nil(t.T(), fNew.Close())
		assert.Nil(t.T(), os.Remove(path.Join(mntDir, "explicitDir/file4.txt")))
	}()

	f, err = os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
	names2, err := f.Readdirnames(-1)

	assert.Nil(t.T(), err)
	require.Equal(t.T(), 4, len(names2))
	assert.Equal(t.T(), "file1.txt", names2[0])
	assert.Equal(t.T(), "file2.txt", names2[1])
	assert.Equal(t.T(), "file3.txt", names2[2])
	assert.Equal(t.T(), "file4.txt", names2[3])
}

// (a) First ReadDir() will be served from GCSFuse filesystem.
// (b) Second ReadDir() will also be served from GCSFuse filesystem, because of
// deletion of new file.
func (t *KernelListCacheTestWithInfiniteTtl) TestKernelListCache_CacheMissOnDeletionOfFile() {
	// First read, kernel will cache the dir response.
	f, err := os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
	defer func() {
		assert.Nil(t.T(), f.Close())
	}()
	names1, err := f.Readdirnames(-1)
	assert.Nil(t.T(), err)
	require.Equal(t.T(), 2, len(names1))
	assert.Equal(t.T(), "file1.txt", names1[0])
	assert.Equal(t.T(), "file2.txt", names1[1])
	err = f.Close()
	assert.Nil(t.T(), err)
	// Adding one object to make sure to change the ReadDir() response.
	assert.Nil(t.T(), t.createObjects(map[string]string{
		"explicitDir/file3.txt": "123456",
	}))
	defer t.deleteObjectOrFail("explicitDir/file3.txt")
	// Advancing time by 5 years (157800000 seconds).
	cacheClock.AdvanceTime(157800000 * time.Second)
	// Ideally no invalidation since infinite ttl, but creation of a new file inside
	// directory evicts the list cache for that directory.
	err = os.Remove(path.Join(mntDir, "explicitDir/file2.txt"))
	require.Nil(t.T(), err)

	f, err = os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
	names2, err := f.Readdirnames(-1)

	assert.Nil(t.T(), err)
	require.Equal(t.T(), 2, len(names2))
	assert.Equal(t.T(), "file1.txt", names2[0])
	assert.Equal(t.T(), "file3.txt", names2[1]) // file2.txt deleted, hence file3.txt
}

// (a) First ReadDir() will be served from GCSFuse filesystem.
// (b) Second ReadDir() will also be served from GCSFuse filesystem, because of
// file rename.
func (t *KernelListCacheTestWithInfiniteTtl) TestKernelListCache_CacheMissOnFileRename() {
	// First read, kernel will cache the dir response.
	f, err := os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
	defer func() {
		assert.Nil(t.T(), f.Close())
	}()
	names1, err := f.Readdirnames(-1)
	assert.Nil(t.T(), err)
	require.Equal(t.T(), 2, len(names1))
	assert.Equal(t.T(), "file1.txt", names1[0])
	assert.Equal(t.T(), "file2.txt", names1[1])
	err = f.Close()
	assert.Nil(t.T(), err)
	// Adding one object to make sure to change the ReadDir() response.
	assert.Nil(t.T(), t.createObjects(map[string]string{
		"explicitDir/file3.txt": "123456",
	}))
	defer t.deleteObjectOrFail("explicitDir/file3.txt")
	// Advancing time by 5 years (157800000 seconds).
	cacheClock.AdvanceTime(157800000 * time.Second)
	// Ideally no invalidation since infinite ttl, but creation of a new file inside
	// directory evicts the list cache for that directory.
	err = os.Rename(path.Join(mntDir, "explicitDir/file2.txt"), path.Join(mntDir, "explicitDir/renamed_file2.txt"))
	require.Nil(t.T(), err)

	f, err = os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
	names2, err := f.Readdirnames(-1)

	assert.Nil(t.T(), err)
	require.Equal(t.T(), 3, len(names2))
	assert.Equal(t.T(), "file1.txt", names2[0])
	assert.Equal(t.T(), "file3.txt", names2[1])
	assert.Equal(t.T(), "renamed_file2.txt", names2[2])
}

// (a) First ReadDir() will be served from GCSFuse filesystem.
// (b) Second ReadDir() will also be served from GCSFuse filesystem, because of
// addition of new directory.
func (t *KernelListCacheTestWithInfiniteTtl) TestKernelListCache_CacheMissOnAdditionOfDirectory() {
	// First read, kernel will cache the dir response.
	f, err := os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
	defer func() {
		assert.Nil(t.T(), f.Close())
	}()
	names1, err := f.Readdirnames(-1)
	assert.Nil(t.T(), err)
	require.Equal(t.T(), 2, len(names1))
	assert.Equal(t.T(), "file1.txt", names1[0])
	assert.Equal(t.T(), "file2.txt", names1[1])
	err = f.Close()
	assert.Nil(t.T(), err)
	// Adding one object to make sure to change the ReadDir() response.
	assert.Nil(t.T(), t.createObjects(map[string]string{
		"explicitDir/file3.txt": "123456",
	}))
	defer t.deleteObjectOrFail("explicitDir/file3.txt")
	// Advancing time by 5 years (157800000 seconds).
	cacheClock.AdvanceTime(157800000 * time.Second)
	// Ideally no invalidation since infinite ttl, but creation of a new file inside
	// directory evicts the list cache for that directory.
	err = os.Mkdir(path.Join(mntDir, "explicitDir/sub_dir"), dirPerms)
	require.Nil(t.T(), err)
	defer func() {
		assert.Nil(t.T(), os.Remove(path.Join(mntDir, "explicitDir/sub_dir")))
	}()

	f, err = os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
	names2, err := f.Readdirnames(-1)

	assert.Nil(t.T(), err)
	require.Equal(t.T(), 4, len(names2))
	assert.Equal(t.T(), "file1.txt", names2[0])
	assert.Equal(t.T(), "file2.txt", names2[1])
	assert.Equal(t.T(), "file3.txt", names2[2])
	assert.Equal(t.T(), "sub_dir", names2[3])
}

// (a) First ReadDir() will be served from GCSFuse filesystem.
// (b) Second ReadDir() will also be served from GCSFuse filesystem, because of
// deletion of directory.
func (t *KernelListCacheTestWithInfiniteTtl) TestKernelListCache_CacheMissOnDeletionOfDirectory() {
	// Creating a directory in the start.
	err := os.Mkdir(path.Join(mntDir, "explicitDir/sub_dir"), dirPerms)
	assert.Nil(t.T(), err)
	// First read, kernel will cache the dir response.
	f, err := os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
	defer func() {
		assert.Nil(t.T(), f.Close())
	}()
	names1, err := f.Readdirnames(-1)
	assert.Nil(t.T(), err)
	require.Equal(t.T(), 3, len(names1))
	assert.Equal(t.T(), "file1.txt", names1[0])
	assert.Equal(t.T(), "file2.txt", names1[1])
	assert.Equal(t.T(), "sub_dir", names1[2])
	err = f.Close()
	assert.Nil(t.T(), err)
	// Adding one object to make sure to change the ReadDir() response.
	assert.Nil(t.T(), t.createObjects(map[string]string{
		"explicitDir/file3.txt": "123456",
	}))
	defer t.deleteObjectOrFail("explicitDir/file3.txt")
	// Advancing time by 5 years (157800000 seconds).
	cacheClock.AdvanceTime(157800000 * time.Second)
	// Ideally no invalidation since infinite ttl, but creation of a new file inside
	// directory evicts the list cache for that directory.
	err = os.Remove(path.Join(mntDir, "explicitDir/sub_dir"))
	require.Nil(t.T(), err)

	f, err = os.Open(path.Join(mntDir, "explicitDir"))
	require.Nil(t.T(), err)
	names2, err := f.Readdirnames(-1)

	assert.Nil(t.T(), err)
	require.Equal(t.T(), 3, len(names2))
	assert.Equal(t.T(), "file1.txt", names2[0])
	assert.Equal(t.T(), "file2.txt", names2[1])
	assert.Equal(t.T(), "file3.txt", names2[2])
}

// (a) First ReadDir() will be served from GCSFuse filesystem.
// (b) Second ReadDir() will also be served from GCSFuse filesystem, because of
// directory rename.
func (t *KernelListCacheTestWithInfiniteTtl) TestKernelListCache_CacheMissOnDirectoryRename() {
	// Creating a directory in the start.
	err := os.Mkdir(path.Join(mntDir, "explicitDir/sub_dir"), dirPerms)
	assert.Nil(t.T(), err)
	// First read, kernel will cache the dir response.
	f, err := os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
	defer func() {
		assert.Nil(t.T(), f.Close())
	}()
	names1, err := f.Readdirnames(-1)
	assert.Nil(t.T(), err)
	require.Equal(t.T(), 3, len(names1))
	assert.Equal(t.T(), "file1.txt", names1[0])
	assert.Equal(t.T(), "file2.txt", names1[1])
	assert.Equal(t.T(), "sub_dir", names1[2])
	err = f.Close()
	assert.Nil(t.T(), err)
	// Adding one object to make sure to change the ReadDir() response.
	assert.Nil(t.T(), t.createObjects(map[string]string{
		"explicitDir/file3.txt": "123456",
	}))
	defer t.deleteObjectOrFail("explicitDir/file3.txt")
	// Advancing time by 5 years (157800000 seconds).
	cacheClock.AdvanceTime(157800000 * time.Second)
	// Ideally no invalidation since infinite ttl, but creation of a new file inside
	// directory evicts the list cache for that directory.
	err = os.Rename(path.Join(mntDir, "explicitDir/sub_dir"), path.Join(mntDir, "explicitDir/renamed_sub_dir"))
	require.Nil(t.T(), err)
	defer func() {
		assert.Nil(t.T(), os.Remove(path.Join(mntDir, "explicitDir/renamed_sub_dir")))
	}()

	f, err = os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
	names2, err := f.Readdirnames(-1)

	assert.Nil(t.T(), err)
	require.Equal(t.T(), 4, len(names2))
	assert.Equal(t.T(), "file1.txt", names2[0])
	assert.Equal(t.T(), "file2.txt", names2[1])
	assert.Equal(t.T(), "file3.txt", names2[2])
	assert.Equal(t.T(), "renamed_sub_dir", names2[3])
}
