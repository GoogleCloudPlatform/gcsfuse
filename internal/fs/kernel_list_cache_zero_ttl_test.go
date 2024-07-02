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

	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type KernelListCacheTestWithZeroTtl struct {
	suite.Suite
	fsTest
	KernelListCacheTestCommon
}

func (t *KernelListCacheTestWithZeroTtl) SetupSuite() {
	t.serverCfg.ImplicitDirectories = true
	t.serverCfg.MountConfig = &config.MountConfig{
		ListConfig: config.ListConfig{
			KernelListCacheTtlSeconds: 0,
		},
		MetadataCacheConfig: config.MetadataCacheConfig{
			TtlInSeconds: 0,
		},
	}
	t.serverCfg.RenameDirLimit = 10
	t.fsTest.SetUpTestSuite()
}

func TestKernelListCacheTestZeroTtlSuite(t *testing.T) {
	suite.Run(t, new(KernelListCacheTestWithZeroTtl))
}

// (a) First ReadDir() will be served from GCSFuse filesystem.
// (b) Second ReadDir() will also be served from GCSFuse filesystem.
func (t *KernelListCacheTestWithZeroTtl) TestKernelListCache_AlwaysCacheMiss() {
	// First read, kernel will cache the dir response.
	f, err := os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
	defer func() {
		assert.Nil(t.T(), f.Close())
	}()
	names1, err := f.Readdirnames(-1)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 2, len(names1))
	assert.Equal(t.T(), "file1.txt", names1[0])
	assert.Equal(t.T(), "file2.txt", names1[1])
	err = f.Close()
	assert.Nil(t.T(), err)
	// Adding one object to make sure to change the ReadDir() response.
	assert.Nil(t.T(), t.createObjects(map[string]string{
		"explicitDir/file3.txt": "123456",
	}))
	defer t.deleteObjectOrFail("explicitDir/file3.txt")

	// Zero ttl, means readdir will always be served from gcsfuse.
	f, err = os.Open(path.Join(mntDir, "explicitDir"))
	assert.Nil(t.T(), err)
	names2, err := f.Readdirnames(-1)

	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 3, len(names2))
	assert.Equal(t.T(), "file1.txt", names2[0])
	assert.Equal(t.T(), "file2.txt", names2[1])
	assert.Equal(t.T(), "file3.txt", names2[2])
}
