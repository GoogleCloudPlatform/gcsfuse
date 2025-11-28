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
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/metadata"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/caching"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type HNSBucketTests struct {
	suite.Suite
	fsTest
	RenameFileTests
	RenameDirTests
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

func (t *HNSBucketTests) SetT(testingT *testing.T) {
	t.Suite.SetT(testingT)
	t.RenameDirTests.SetT(testingT)
	t.RenameFileTests.SetT(testingT)
}

func (t *HNSBucketTests) SetupSuite() {
	t.serverCfg.ImplicitDirectories = false
	t.serverCfg.NewConfig = &cfg.Config{
		EnableHns:                true,
		EnableAtomicRenameObject: true,
	}
	t.serverCfg.MetricHandle = metrics.NewNoopMetrics()
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
	lruCache := lru.NewTrieCache(uint64(1000 * cfg.AverageSizeOfPositiveStatCacheEntry))
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
			StatCacheMaxSizeMb: 33,
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
