// Copyright 2025 Google LLC
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

// A collection of tests to check notifier is invalidating the entry.
package fs_test

import (
	"github.com/jacobsa/fuse"
	"math"
	"os"
	"path"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type NotifierTest struct {
	suite.Suite
	fsTest
}

func TestNotifierTestSuite(t *testing.T) {
	suite.Run(t, new(NotifierTest))
}

func (t *NotifierTest) SetupSuite() {
	t.mountCfg.DisableWritebackCaching = true
	t.serverCfg.ImplicitDirectories = true
	t.serverCfg.InodeAttributeCacheTTL = 1000 * time.Second
	t.serverCfg.Notifier = fuse.NewNotifier()
	t.serverCfg.NewConfig = &cfg.Config{
		FileSystem: cfg.FileSystemConfig{
			ExperimentalEnableDentryCache: true,
			PreconditionErrors:            true,
		},
		MetadataCache: cfg.MetadataCacheConfig{
			StatCacheMaxSizeMb: 33,
			TtlSecs:            1000,
			TypeCacheMaxSizeMb: 4,
		},
		Write: cfg.WriteConfig{
			EnableStreamingWrites: true,
			BlockSizeMb:           operations.MiB,
			MaxBlocksPerFile:      1,
			GlobalMaxBlocks:       math.MaxInt,
		},
	}
	t.fsTest.SetUpTestSuite()
}

func (t *NotifierTest) TearDownTest() {
	t.fsTest.TearDown()
}

func (t *NotifierTest) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
}

func (t *NotifierTest) TestWriteFileWithRootDirParent() {
	// Create an empty object on bucket.
	t.f1 = createGCSObject(t.T(), "foo", "")
	_, err := os.Stat(path.Join(mntDir, "foo"))
	assert.Nil(t.T(), err)
	// Replace the underlying object with a new generation.
	clobberFile(t.T(), "foo", "foobar")
	// Writing to file will return Stale File Handle Error.
	data, err := operations.GenerateRandomData(operations.MiB * 4)
	assert.NoError(t.T(), err)

	_, err = t.f1.WriteAt(data, 0)

	operations.ValidateESTALEError(t.T(), err)
	// attempt to write again, the entry has now been invalidated.
	_, err = t.f1.WriteAt(data, 0)
	assert.NoError(t.T(), err)
}

//func (t *NotifierTest) TestWriteFileWithNonRootDirParent() {
//	// Create an empty object on bucket.
//	assert.Nil(t.T(), t.createObjects(map[string]string{"dir/foo": ""}))
//	_, err := os.Stat(path.Join(mntDir, "dir/foo"))
//	assert.Nil(t.T(), err)
//	// Replace the underlying object with a new generation.
//	assert.Nil(t.T(), t.createObjects(map[string]string{"dir/foo": "foobar"}))
//	// Writing to file will return Stale File Handle Error.
//	data, err := operations.GenerateRandomData(operations.MiB * 4)
//	assert.NoError(t.T(), err)
//
//	err = os.WriteFile(path.Join(mntDir, "dir/foo"), data, 0644)
//
//	operations.ValidateESTALEError(t.T(), err)
//	// attempt to write again, the entry has now been invalidated.
//	err = os.WriteFile(path.Join(mntDir, "dir/foo"), data, 0644)
//	assert.NoError(t.T(), err)
//}
//
//func (t *NotifierTest) TestReadFileDoNotFailPersistently() {
//	// Create an empty object on bucket.
//	assert.Nil(t.T(), t.createObjects(map[string]string{"foo": ""}))
//	_, err := os.Stat(path.Join(mntDir, "foo"))
//	assert.Nil(t.T(), err)
//	// Replace the underlying object with a new generation.
//	assert.Nil(t.T(), t.createObjects(map[string]string{"foo": "foobar"}))
//
//	// attempt to ReadFile, should give Stale File Handle error
//	_, err = os.ReadFile(path.Join(mntDir, "foo"))
//
//	operations.ValidateESTALEError(t.T(), err)
//	// attempt to write again, the entry has now been invalidated.
//	_, err = os.ReadFile(path.Join(mntDir, "foo"))
//	assert.NoError(t.T(), err)
//}
