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
	"os"
	"path"
	"syscall"
	"testing"
	"time"

	"github.com/jacobsa/fuse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/vipnydav/gcsfuse/v3/cfg"
	"github.com/vipnydav/gcsfuse/v3/common"
	"github.com/vipnydav/gcsfuse/v3/internal/storage/storageutil"
)

type NotifierTest struct {
	suite.Suite
	fsTest
}

func TestNotifierTestSuite(t *testing.T) {
	suite.Run(t, new(NotifierTest))
}

func (t *NotifierTest) SetupSuite() {
	t.serverCfg.ImplicitDirectories = true
	t.serverCfg.InodeAttributeCacheTTL = 1000 * time.Second
	t.serverCfg.Notifier = fuse.NewNotifier()
	t.serverCfg.NewConfig = &cfg.Config{
		FileSystem: cfg.FileSystemConfig{
			ExperimentalEnableDentryCache: true,
			PreconditionErrors:            true,
		},
		Write: cfg.WriteConfig{
			EnableStreamingWrites: true,
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
	filePath := path.Join(mntDir, fileName)
	// Create a file in GCS.
	_, err := storageutil.CreateObject(ctx, bucket, fileName, []byte("initial content"))
	require.NoError(t.T(), err)
	// Stat file to cache its entry.
	_, err = os.Stat(filePath)
	require.NoError(t.T(), err)
	// Clobber the file in GCS. This changes the object's generation, making
	// our file handle stale.
	_, err = storageutil.CreateObject(ctx, bucket, fileName, []byte("modified"))
	require.NoError(t.T(), err)

	// Attempt to write.
	err = common.WriteFile(filePath, "new data")

	// Should return stale file handle error.
	require.Error(t.T(), err)
	assert.Regexp(t.T(), syscall.ESTALE.Error(), err.Error())
	// Attempt to write again, the entry has now been invalidated.
	err = common.WriteFile(filePath, "new data")
	assert.NoError(t.T(), err)
}

func (t *NotifierTest) TestWriteFileWithNonRootDirParent() {
	filePath := path.Join(mntDir, "dir/foo")
	// Create a file in GCS.
	_, err := storageutil.CreateObject(ctx, bucket, "dir/foo", []byte("initial content"))
	require.NoError(t.T(), err)
	// Stat file to cache its entry.
	_, err = os.Stat(filePath)
	require.NoError(t.T(), err)
	// Clobber the file in GCS. This changes the object's generation, making
	// our file handle stale.
	_, err = storageutil.CreateObject(ctx, bucket, "dir/foo", []byte("modified"))
	require.NoError(t.T(), err)

	// Attempt to write.
	err = common.WriteFile(filePath, "new data")

	// Should return stale file handle error.
	require.Error(t.T(), err)
	assert.Regexp(t.T(), syscall.ESTALE.Error(), err.Error())
	// Attempt to write again, the entry has now been invalidated.
	err = common.WriteFile(filePath, "new data")
	assert.NoError(t.T(), err)
}

func (t *NotifierTest) TestReadFileDoNotFailPersistently() {
	filePath := path.Join(mntDir, fileName)
	// Create a file in GCS.
	_, err := storageutil.CreateObject(ctx, bucket, fileName, []byte("initial content"))
	require.NoError(t.T(), err)
	// Stat file to cache its entry.
	_, err = os.Stat(filePath)
	require.NoError(t.T(), err)
	// Clobber the file in GCS. This changes the object's generation, making
	// our file handle stale.
	_, err = storageutil.CreateObject(ctx, bucket, fileName, []byte("modified"))
	require.NoError(t.T(), err)

	// Attempt to read file.
	_, err = common.ReadFile(filePath)

	// Should return error.
	assert.NotNil(t.T(), err)
	// Attempt to read again, the entry has now been invalidated.
	_, err = common.ReadFile(filePath)
	assert.NoError(t.T(), err)
}
