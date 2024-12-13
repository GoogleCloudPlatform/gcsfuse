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

// A collection of tests which tests the kernel-list-cache feature, in which
// directory listing 2nd time is served from kernel page-cache unless not invalidated.
// Base of all the tests: how to detect if directory listing is served from page-cache
// or from GCSFuse?
// (a) GCSFuse file-system ensures different content, when listing happens on the same directory.
// (b) If two consecutive directory listing for the same directory are same, that means
//     2nd listing is served from kernel-page-cache.
// (c) If not then, both 1st and 2nd listing are served from GCSFuse filesystem.

package fs_test

import (
	"os"
	"path"
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type StreamingWritesEmptyObjectTest struct {
	suite.Suite
	fsTest
}

func (t *StreamingWritesEmptyObjectTest) SetupSuite() {
	t.serverCfg.NewConfig = &cfg.Config{
		Write: cfg.WriteConfig{
			BlockSizeMb:                       10,
			CreateEmptyFile:                   false,
			ExperimentalEnableStreamingWrites: true,
			GlobalMaxBlocks:                   20,
			MaxBlocksPerFile:                  10,
		},
	}
	t.fsTest.SetUpTestSuite()
}

func (t *StreamingWritesEmptyObjectTest) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
}
func (t *StreamingWritesEmptyObjectTest) SetupTest() {
	// Create an object on bucket.
	_, err := storageutil.CreateObject(
		ctx,
		bucket,
		fileName,
		[]byte("bar"))
	assert.Equal(t.T(), nil, err)
	// Open file handle to read or write.
	t.f1, err = os.OpenFile(path.Join(mntDir, fileName), os.O_RDWR|syscall.O_DIRECT, filePerms)
	assert.Equal(t.T(), nil, err)
}

func (t *StreamingWritesEmptyObjectTest) TearDownTest() {
	t.fsTest.TearDown()
}

func TestStreamingWritesEmptyObjectTest(t *testing.T) {
	suite.Run(t, new(StreamingWritesEmptyObjectTest))
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *StreamingWritesEmptyObjectTest) TestUnlinkWritesAreNotInProgress() {
	// unlink the synced file.
	err := os.Remove(t.f1.Name())
	assert.NoError(t.T(), err)

	// Stat the file and validate file is deleted.
	operations.ValidateNoFileOrDirError(t.T(), t.f1.Name())
	// Close the file and validate that file is not created on GCS.
	err = t.f1.Close()
	assert.NoError(nil, err)
	t.f1 = nil
	operations.ValidateObjectNotFoundErr(ctx, t.T(), bucket, fileName)
}

func (t *StreamingWritesEmptyObjectTest) TestUnlinkWhenWritesAreInProgress() {
	_, err := t.f1.Write([]byte("tacos"))
	assert.Nil(t.T(), err)

	// unlink the file.
	err = os.Remove(t.f1.Name())
	assert.NoError(t.T(), err)

	// Stat the file and validate file is deleted.
	operations.ValidateNoFileOrDirError(t.T(), t.f1.Name())
	// Close the file and validate that file is not created on GCS.
	err = t.f1.Close()
	assert.NoError(nil, err)
	t.f1 = nil
	operations.ValidateObjectNotFoundErr(ctx, t.T(), bucket, fileName)
}
