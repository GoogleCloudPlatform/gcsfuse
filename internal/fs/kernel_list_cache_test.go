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

// A collection of tests for a file system where parallel dirops are allowed.
// Dirops refers to readdir and lookup operations.

package fs_test

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type KernelListCacheTest struct {
	suite.Suite
	fsTest
}

func (t *KernelListCacheTest) SetupSuite() {
	t.serverCfg.ImplicitDirectories = true
	t.serverCfg.MountConfig = &config.MountConfig{
		FileSystemConfig: config.FileSystemConfig{
			DisableParallelDirops:     false,
			KernelListCacheTtlSeconds: 100,
		}}
	t.serverCfg.RenameDirLimit = 10
	t.fsTest.SetUpTestSuite()
}

func (t *KernelListCacheTest) SetupTest() {
	t.createFilesAndDirStructureInBucket()
}

func (t *KernelListCacheTest) TearDownTest() {
	t.fsTest.TearDown()
}

func (t *KernelListCacheTest) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
}

func TestKernelListCacheTestSuite(t *testing.T) {
	suite.Run(t, new(KernelListCacheTest))
}

// createFilesAndDirStructureInBucket creates the following files and directory
// structure.
// bucket
//
//	file1.txt
//	file2.txt
//	explicitDir1/
//	explicitDir1/file1.txt
//	explicitDir1/file2.txt
//	implicitDir1/file1.txt
func (t *KernelListCacheTest) createFilesAndDirStructureInBucket() {
	assert.Nil(
		t.T(),
		t.createObjects(
			map[string]string{
				"file1.txt":              "abcdef",
				"file2.txt":              "xyz",
				"explicitDir1/":          "",
				"explicitDir1/file1.txt": "12345",
				"explicitDir1/file2.txt": "6789101112",
				"implicitDir1/file1.txt": "-1234556789",
			}))
}

func (t *KernelListCacheTest) TestKernelListCache_SimpleWorkingCase() {
	// First read, kernel will cache the dir response.
	f, err := os.Open(path.Join(mntDir, "explicitDir1"))
	assert.Nil(t.T(), err)
	defer f.Close()

	names1, err := f.Readdirnames(-1)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 2, len(names1))

	err = f.Close()
	assert.Nil(t.T(), err)

	// Create another object
	assert.Nil(
		t.T(),
		t.createObjects(
			map[string]string{
				"explicitDir1/file3.txt": "123456",
			}))

	// First read, kernel will cache the dir response.
	f, err = os.Open(path.Join(mntDir, "explicitDir1"))
	assert.Nil(t.T(), err)
	names2, err := f.Readdirnames(-1)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 2, len(names2))

	err = f.Close()
	assert.Nil(t.T(), err)
}

func (t *KernelListCacheTest) TestKernelListCache_SimpleWorkingCase_False() {
	// First read, kernel will cache the dir response.
	f, err := os.Open(path.Join(mntDir, "explicitDir1"))
	assert.Nil(t.T(), err)
	defer f.Close()

	names1, err := f.Readdirnames(-1)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 2, len(names1))

	err = f.Close()
	assert.Nil(t.T(), err)

	// Create another object
	assert.Nil(
		t.T(),
		t.createObjects(
			map[string]string{
				"explicitDir1/file3.txt": "123456",
			}))

	// Advance the time more than ttl, to invalidate the cache.
	cacheClock.AdvanceTime(110 * time.Second)

	// First read, kernel will cache the dir response.
	f, err = os.Open(path.Join(mntDir, "explicitDir1"))
	assert.Nil(t.T(), err)
	names2, err := f.Readdirnames(-1)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), 3, len(names2))

	err = f.Close()
	assert.Nil(t.T(), err)
}
