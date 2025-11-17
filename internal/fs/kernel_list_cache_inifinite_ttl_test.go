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

// Please refer kernel_list_cache_test.go for the documentation.

package fs_test

import (
	"errors"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/vipnydav/gcsfuse/v3/cfg"
	"github.com/vipnydav/gcsfuse/v3/common"
	"github.com/vipnydav/gcsfuse/v3/metrics"
)

func SkipTestForUnsupportedKernelVersion(t *testing.T) {
	// TODO: b/384648943 make this part of fsTest.SetUpTestSuite() after post fs
	// tests are fully migrated to stretchr/testify.
	t.Helper()
	unsupported, err := common.IsKLCacheEvictionUnSupported()
	assert.NoError(t, err)
	if unsupported {
		t.SkipNow()
	}
}

type KernelListCacheTestWithInfiniteTtl struct {
	suite.Suite
	fsTest
	KernelListCacheTestCommon
}

func (t *KernelListCacheTestWithInfiniteTtl) SetupSuite() {
	t.serverCfg.ImplicitDirectories = true
	t.serverCfg.NewConfig = &cfg.Config{
		FileSystem: cfg.FileSystemConfig{
			KernelListCacheTtlSecs: -1,
		},
		MetadataCache: cfg.MetadataCacheConfig{
			TtlSecs: 0,
		},
	}
	t.serverCfg.RenameDirLimit = 10
	t.serverCfg.MetricHandle = metrics.NewNoopMetrics()
	t.fsTest.SetUpTestSuite()
}

func TestKernelListCacheTestInfiniteTtlSuite(t *testing.T) {
	SkipTestForUnsupportedKernelVersion(t)
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

func (t *KernelListCacheTestWithInfiniteTtl) TestKernelListCache_RemoveDirAfterListIsCachedWorks() {
	// First read, kernel will cache the dir response.
	f, err := os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
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
	// Advancing time by 5 years (157800000 seconds).
	cacheClock.AdvanceTime(157800000 * time.Second)

	// os.RemoveDir should delete all files and invalidate cache.
	err = os.RemoveAll(path.Join(mntDir, "explicitDir"))
	assert.NoError(t.T(), err)

	_, err = os.ReadDir(path.Join(mntDir, "explicitDir"))
	var pathError *os.PathError
	assert.True(t.T(), errors.As(err, &pathError))
}

func (t *KernelListCacheTestWithInfiniteTtl) TestKernelListCache_RemoveDirAfterListCacheInvalidatesCache() {
	// First read, kernel will cache the dir response.
	f, err := os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
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
	// Advancing time by 5 years (157800000 seconds).
	cacheClock.AdvanceTime(157800000 * time.Second)
	// os.RemoveDir should delete all files and invalidate cache.
	err = os.RemoveAll(path.Join(mntDir, "explicitDir"))
	assert.NoError(t.T(), err)

	// Adding one more object to make sure to change the ReadDir() response.
	assert.Nil(t.T(), t.createObjects(map[string]string{
		"explicitDir/file4.txt": "123456",
	}))
	names2, err := os.ReadDir(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)

	assert.Nil(t.T(), err)
	require.Equal(t.T(), 1, len(names2))
	assert.Equal(t.T(), "file4.txt", names2[0].Name())
}
