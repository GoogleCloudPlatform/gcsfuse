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

package fs_test

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/caching"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type HNSBucketTests struct {
	fsTest
	HNSBucketCommonTest
}

type dirEntry struct {
	name  string
	isDir bool
}

const file1Content = "abcdef"
const file2Content = "file2"

var expectedFooDirEntries = []dirEntry{
	{name: "test", isDir: true},
	{name: "test2", isDir: true},
	{name: "file1.txt", isDir: false},
	{name: "file2.txt", isDir: false},
	{name: "implicit_dir", isDir: true},
}

func TestHNSBucketTests(t *testing.T) { suite.Run(t, new(HNSBucketTests)) }

func (t *HNSBucketTests) SetupSuite() {
	t.serverCfg.ImplicitDirectories = false
	t.serverCfg.NewConfig = &cfg.Config{
		EnableHns:                true,
		EnableAtomicRenameObject: true,
	}
	t.serverCfg.MetricHandle = common.NewNoopMetrics()
	bucketType = gcs.BucketType{Hierarchical: true}
	t.fsTest.SetUpTestSuite()
}

func (t *HNSBucketTests) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
}

func (t *HNSBucketTests) SetupTest() {
	err := t.createFolders([]string{"foo/", "bar/", "foo/test2/", "foo/test/"})
	require.NoError(t.T(), err)

	err = t.createObjects(
		map[string]string{
			"foo/file1.txt":              file1Content,
			"foo/file2.txt":              file2Content,
			"foo/test/file3.txt":         "xyz",
			"foo/implicit_dir/file3.txt": "xxw",
			"bar/file1.txt":              "-1234556789",
		})
	require.NoError(t.T(), err)
}

func (t *HNSBucketTests) TearDownTest() {
	t.fsTest.TearDown()
}

func (t *HNSBucketTests) TestReadDir() {
	dirPath := path.Join(mntDir, "foo")

	dirEntries, err := os.ReadDir(dirPath)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 5, len(dirEntries))
	actualDirEntries := []dirEntry{}
	for _, d := range dirEntries {
		actualDirEntries = append(actualDirEntries, dirEntry{
			name:  d.Name(),
			isDir: d.IsDir(),
		})
	}
	assert.ElementsMatch(t.T(), actualDirEntries, expectedFooDirEntries)
}

func (t *HNSBucketTests) TestDeleteFolder() {
	dirPath := path.Join(mntDir, "foo")

	err := os.RemoveAll(dirPath)

	assert.NoError(t.T(), err)
	_, err = os.Stat(dirPath)
	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
}

func (t *HNSBucketTests) TestDeleteImplicitDir() {
	dirPath := path.Join(mntDir, "foo", "implicit_dir")

	err := os.RemoveAll(dirPath)

	assert.NoError(t.T(), err)
	_, err = os.Stat(dirPath)
	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
}

func (t *HNSBucketTests) TestRenameFolderWithSrcDirectoryDoesNotExist() {
	oldDirPath := path.Join(mntDir, "foo_not_exist")
	newDirPath := path.Join(mntDir, "foo_rename")

	err := os.Rename(oldDirPath, newDirPath)

	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
	_, err = os.Stat(newDirPath)
	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
}

func (t *HNSBucketTests) TestRenameFolderWithDstDirectoryNotEmpty() {
	oldDirPath := path.Join(mntDir, "foo")
	_, err := os.Stat(oldDirPath)
	assert.NoError(t.T(), err)
	// In the setup phase, we created file1.txt within the bar directory.
	newDirPath := path.Join(mntDir, "bar")
	_, err = os.Stat(newDirPath)
	assert.NoError(t.T(), err)

	err = os.Rename(oldDirPath, newDirPath)

	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "file exists"))
}

func (t *HNSBucketTests) TestRenameFolderWithEmptySourceDirectory() {
	oldDirPath := path.Join(mntDir, "foo", "test2")
	_, err := os.Stat(oldDirPath)
	assert.NoError(t.T(), err)
	newDirPath := path.Join(mntDir, "foo_rename")
	_, err = os.Stat(newDirPath)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))

	err = os.Rename(oldDirPath, newDirPath)

	assert.NoError(t.T(), err)
	_, err = os.Stat(oldDirPath)
	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
	_, err = os.Stat(newDirPath)
	assert.NoError(t.T(), err)
	dirEntries, err := os.ReadDir(newDirPath)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 0, len(dirEntries))
}

func (t *HNSBucketTests) TestRenameFolderWithSourceDirectoryHaveLocalFiles() {
	oldDirPath := path.Join(mntDir, "foo", "test")
	_, err := os.Stat(oldDirPath)
	assert.NoError(t.T(), err)
	file, err := os.OpenFile(path.Join(oldDirPath, "file4.txt"), os.O_RDWR|os.O_CREATE, filePerms)
	assert.NoError(t.T(), err)
	defer file.Close()
	newDirPath := path.Join(mntDir, "bar", "foo_rename")

	err = os.Rename(oldDirPath, newDirPath)

	assert.Error(t.T(), err)
	// In the logs, we encountered the following error:
	// "Rename: operation not supported, can't rename directory 'test' with open files: operation not supported."
	// This was translated to an "operation not supported" error at the kernel level.
	assert.True(t.T(), strings.Contains(err.Error(), "operation not supported"))
}

func (t *HNSBucketTests) TestRenameFolderWithSameParent() {
	oldDirPath := path.Join(mntDir, "foo")
	_, err := os.Stat(oldDirPath)
	require.NoError(t.T(), err)
	newDirPath := path.Join(mntDir, "foo_rename")
	_, err = os.Stat(newDirPath)
	require.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))

	err = os.Rename(oldDirPath, newDirPath)

	assert.NoError(t.T(), err)
	_, err = os.Stat(oldDirPath)
	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
	_, err = os.Stat(newDirPath)
	assert.NoError(t.T(), err)
	dirEntries, err := os.ReadDir(newDirPath)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 5, len(dirEntries))
	actualDirEntries := []dirEntry{}
	for _, d := range dirEntries {
		actualDirEntries = append(actualDirEntries, dirEntry{
			name:  d.Name(),
			isDir: d.IsDir(),
		})
	}
	assert.ElementsMatch(t.T(), actualDirEntries, expectedFooDirEntries)
}

func (t *HNSBucketTests) TestRenameFolderWithExistingEmptyDestDirectory() {
	oldDirPath := path.Join(mntDir, "foo", "test")
	_, err := os.Stat(oldDirPath)
	require.NoError(t.T(), err)
	newDirPath := path.Join(mntDir, "foo", "test2")
	_, err = os.Stat(newDirPath)
	require.NoError(t.T(), err)

	// Go's Rename function does not support renaming a directory into an existing empty directory.
	// To achieve this, we call a Python rename function as a workaround.
	cmd := exec.Command("python3", "-c", fmt.Sprintf("import os; os.rename('%s', '%s')", oldDirPath, newDirPath))
	_, err = cmd.CombinedOutput()

	assert.NoError(t.T(), err)
	_, err = os.Stat(oldDirPath)
	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
	_, err = os.Stat(newDirPath)
	assert.NoError(t.T(), err)
	dirEntries, err := os.ReadDir(newDirPath)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 1, len(dirEntries))
	assert.Equal(t.T(), "file3.txt", dirEntries[0].Name())
	assert.False(t.T(), dirEntries[0].IsDir())
}

func (t *HNSBucketTests) TestRenameFolderWithDifferentParents() {
	oldDirPath := path.Join(mntDir, "foo")
	_, err := os.Stat(oldDirPath)
	assert.NoError(t.T(), err)
	newDirPath := path.Join(mntDir, "bar", "foo_rename")

	err = os.Rename(oldDirPath, newDirPath)

	assert.NoError(t.T(), err)
	_, err = os.Stat(oldDirPath)
	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
	_, err = os.Stat(newDirPath)
	assert.NoError(t.T(), err)
	dirEntries, err := os.ReadDir(newDirPath)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 5, len(dirEntries))
	actualDirEntries := []dirEntry{}
	for _, d := range dirEntries {
		actualDirEntries = append(actualDirEntries, dirEntry{
			name:  d.Name(),
			isDir: d.IsDir(),
		})
	}
	assert.ElementsMatch(t.T(), actualDirEntries, expectedFooDirEntries)
}

func (t *HNSBucketTests) TestRenameFolderWithOpenGCSFile() {
	oldDirPath := path.Join(mntDir, "bar")
	_, err := os.Stat(oldDirPath)
	assert.NoError(t.T(), err)
	newDirPath := path.Join(mntDir, "bar_rename")
	filePath := path.Join(oldDirPath, "file1.txt")
	f, err := os.Open(filePath)
	require.NoError(t.T(), err)

	err = os.Rename(oldDirPath, newDirPath)

	require.NoError(t.T(), err)
	_, err = f.WriteString("test")
	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "bad file descriptor"))
	assert.NoError(t.T(), f.Close())
	_, err = os.Stat(oldDirPath)
	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
	_, err = os.Stat(newDirPath)
	assert.NoError(t.T(), err)
	dirEntries, err := os.ReadDir(newDirPath)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 1, len(dirEntries))
	assert.Equal(t.T(), "file1.txt", dirEntries[0].Name())
	assert.False(t.T(), dirEntries[0].IsDir())
}

// Create directory foo.
// Stat the directory foo.
// Rename directory foo --> foo_rename
// Stat the old directory.
// Stat the new directory.
// Read new directory and validate.
// Create old directory again with same name - foo
// Stat the directory - foo
// Read directory again and validate it is empty.
func (t *HNSBucketTests) TestCreateDirectoryWithSameNameAfterRename() {
	oldDirPath := path.Join(mntDir, "foo")
	_, err := os.Stat(oldDirPath)
	require.NoError(t.T(), err)
	newDirPath := path.Join(mntDir, "foo_rename")
	// Rename directory foo --> foo_rename
	err = os.Rename(oldDirPath, newDirPath)
	require.NoError(t.T(), err)
	// Stat old directory.
	_, err = os.Stat(oldDirPath)
	require.Error(t.T(), err)
	require.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
	// Stat new directory.
	_, err = os.Stat(newDirPath)
	require.NoError(t.T(), err)
	// Read new directory and validate.
	dirEntries, err := os.ReadDir(newDirPath)
	require.NoError(t.T(), err)
	require.Equal(t.T(), 5, len(dirEntries))
	actualDirEntries := []dirEntry{}
	for _, d := range dirEntries {
		actualDirEntries = append(actualDirEntries, dirEntry{
			name:  d.Name(),
			isDir: d.IsDir(),
		})
	}
	require.ElementsMatch(t.T(), actualDirEntries, expectedFooDirEntries)

	// Create old directory again.
	err = os.Mkdir(oldDirPath, dirPerms)

	assert.NoError(t.T(), err)
	_, err = os.Stat(oldDirPath)
	assert.NoError(t.T(), err)
	dirEntries, err = os.ReadDir(oldDirPath)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 0, len(dirEntries))
}

// Create directory - foo/test2
// Create local file in directory - foo/test2/test.txt
// Stat the local file - foo/test2/test.txt
// Delete directory - rm -r foo/test2
// Create directory again - foo/test2
// Create local file with the same name in directory - foo/test2/test.txt
// Stat the local file - foo/test2/test.txt
func (t *HNSBucketTests) TestCreateLocalFileInSamePathAfterDeletingParentDirectory() {
	dirPath := path.Join(mntDir, "foo", "test2")
	filePath := path.Join(dirPath, "test.txt")
	// Create local file in side it.
	f1, err := os.Create(filePath)
	defer require.NoError(t.T(), f1.Close())
	require.NoError(t.T(), err)
	_, err = os.Stat(filePath)
	require.NoError(t.T(), err)
	// Delete directory rm -r foo/test2
	err = os.RemoveAll(dirPath)
	assert.NoError(t.T(), err)
	// Create directory again foo/test2
	err = os.Mkdir(dirPath, dirPerms)
	assert.NoError(t.T(), err)

	// Create local file again.
	f2, err := os.Create(filePath)
	defer require.NoError(t.T(), f2.Close())

	assert.NoError(t.T(), err)
	_, err = os.Stat(filePath)
	assert.NoError(t.T(), err)
}

// //////////////////////////////////////////////////////////////////////
// HNS bucket with caching support tests
// //////////////////////////////////////////////////////////////////////
const (
	cachedHnsBucketName string = "cachedHnsBucket"
)

var (
	uncachedHNSBucket gcs.Bucket
)

type HNSCachedBucketMountTest struct {
	suite.Suite
	fsTest
}

func TestHNSCachedBucketTests(t *testing.T) { suite.Run(t, new(HNSCachedBucketMountTest)) }

func (t *HNSCachedBucketMountTest) SetupSuite() {
	bucketType = gcs.BucketType{Hierarchical: true}
	uncachedHNSBucket = fake.NewFakeBucket(timeutil.RealClock(), cachedHnsBucketName, bucketType)
	lruCache := newLruCache(uint64(1000 * cfg.AverageSizeOfPositiveStatCacheEntry))
	statCache := metadata.NewStatCacheBucketView(lruCache, "")
	bucket = caching.NewFastStatBucket(
		ttl,
		statCache,
		&cacheClock,
		uncachedHNSBucket,
		negativeCacheTTL)

	// Enable directory type caching.
	t.serverCfg.DirTypeCacheTTL = ttl
	t.serverCfg.ImplicitDirectories = false
	t.serverCfg.NewConfig = &cfg.Config{
		EnableHns: true,
		FileCache: defaultFileCacheConfig(),
		MetadataCache: cfg.MetadataCacheConfig{
			// Setting default values.
			StatCacheMaxSizeMb: 32,
			TtlSecs:            60,
			TypeCacheMaxSizeMb: 4,
		},
	}
	// Call through.
	t.fsTest.SetUpTestSuite()
}

func (t *HNSCachedBucketMountTest) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
}

func (t *HNSCachedBucketMountTest) SetupTest() {
	err := t.createFolders([]string{"hns/", "hns/cache/"})
	require.NoError(t.T(), err)
}

func (t *HNSCachedBucketMountTest) TearDownTest() {
	t.fsTest.TearDown()
}

// ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
// --------------------- Test for delete object -------------------
// Create directory
// Create object in directory
// Stat the object
// Delete object
// Create object using other client
// Stat the object immediately, It should return not found error although object present
// Stat the object after TTL expiry and it should appear
func (t *HNSCachedBucketMountTest) TestLocalFileIsInaccessibleAfterDeleteObjectButPresentRemotely() {
	dirPath := path.Join(mntDir, "hns", "cache")
	filePath := path.Join(dirPath, "file1.txt")
	// Create local file inside it.
	ff, err := os.Create(filePath)
	require.NoError(t.T(), ff.Close())
	require.NoError(t.T(), err)
	_, err = os.Stat(filePath)
	require.NoError(t.T(), err)
	// Delete object
	err = os.Remove(filePath)
	assert.NoError(t.T(), err)
	// Create an object with the same name via GCS.
	_, err = storageutil.CreateObject(
		ctx,
		uncachedHNSBucket,
		path.Join("hns", "cache", "file1.txt"),
		[]byte("burrito"))
	assert.NoError(t.T(), err)
	// Stat the object --> It should return not found error although object present
	_, err = os.Stat(filePath)
	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
	// After the TTL elapses, we should see it reappear.
	cacheClock.AdvanceTime(ttl + time.Millisecond)
	_, err = os.Stat(filePath)
	assert.NoError(t.T(), err)
}

// --------------------- Test for delete directory -----------------
// Create directory
// stat directory
// Delete directory
// Create directory using other client
// Stat the directory immeditely, It should return not found error from cache although dir present
// Stat the directory after TTL expiry and it should appear
// ------------------------------------------------------------------
func (t *HNSCachedBucketMountTest) TestLocalDirectoryIsInaccessibleAfterDeleteDirectoryButPresentRemotely() {
	dirPath := path.Join(mntDir, "hns", "cache", "test")
	// Create directory - foo/test2
	err := os.Mkdir(dirPath, dirPerms)
	require.NoError(t.T(), err)
	// stat directory - foo/test2
	_, err = os.Stat(dirPath)
	require.NoError(t.T(), err)
	// Delete directory - rm -r foo/test2
	err = os.RemoveAll(dirPath)
	assert.NoError(t.T(), err)
	// Create a directory with the same name via GCS.
	_, err = storageutil.CreateObject(
		ctx,
		uncachedHNSBucket,
		path.Join("hns", "cache", "test")+"/",
		[]byte(""))
	assert.NoError(t.T(), err)
	// Stat the directory - foo/test2/test.txt --> It should return not found error from cache although dir present
	_, err = os.Stat(dirPath)
	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))

	// After the TTL elapses, we should see it reappear.
	cacheClock.AdvanceTime(ttl + time.Millisecond)
	_, err = os.Stat(dirPath)
	assert.NoError(t.T(), err)
}
