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
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const (
	fileName = "foo"
)

type StreamingWritesLocalFileTest struct {
	suite.Suite
	fsTest
}

func (t *StreamingWritesLocalFileTest) SetupSuite() {
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

func (t *StreamingWritesLocalFileTest) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
}
func (t *StreamingWritesLocalFileTest) SetupTest() {
	// Create a local file.
	_, t.f1 = operations.CreateLocalFile(ctx, t.T(), mntDir, bucket, fileName)
}

func (t *StreamingWritesLocalFileTest) TearDownTest() {
	t.fsTest.TearDown()
}

func TestStreamingWritesTestSuite(t *testing.T) {
	suite.Run(t, new(StreamingWritesLocalFileTest))
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *StreamingWritesLocalFileTest) TestUnlinkWhenWritesAreNotInProgress() {
	// unlink the local file.
	err := os.Remove(t.f1.Name())
	assert.NoError(t.T(), err)

	// Stat the local file and validate file is deleted.
	operations.ValidateNoFileOrDirError(t.T(), t.f1.Name())
	// Close the file and validate that file is not created on GCS.
	err = operations.CloseLocalFile(t.T(), &t.f1)
	assert.NoError(nil, err)
	operations.ValidateObjectNotFoundErr(ctx, t.T(), bucket, fileName)
}

func (t *StreamingWritesLocalFileTest) TestUnlinkWhenWritesAreInProgress() {
	_, err := t.f1.Write([]byte("tacos"))
	assert.Nil(t.T(), err)

	// unlink the local file.
	err = os.Remove(t.f1.Name())
	assert.NoError(t.T(), err)

	// Stat the local file and validate file is deleted.
	operations.ValidateNoFileOrDirError(t.T(), t.f1.Name())
	// Close the file and validate that file is not created on GCS.
	err = operations.CloseLocalFile(t.T(), &t.f1)
	assert.NoError(nil, err)
	operations.ValidateObjectNotFoundErr(ctx, t.T(), bucket, fileName)
}

func (t *StreamingWritesEmptyObjectTest) TestRemoveDirectoryContainingLocalAndEmptyObject() {
	// Create explicit directory with one synced and one local file.
	explicitDirName := "explicit"
	emptyFileName := "emptyFile"
	nonEmptyFileName := "nonEmptyFile"
	assert.Equal(t.T(),
		nil,
		t.createObjects(
			map[string]string{
				// File
				explicitDirName + "/":                        "",
				path.Join(explicitDirName, emptyFileName):    "",
				path.Join(explicitDirName, nonEmptyFileName): "taco",
			}))
	// Write content to local and empty gcs file.
	_, t.f1 = operations.CreateLocalFile(ctx, t.T(), mntDir, bucket, path.Join(explicitDirName, fileName))
	_, err := t.f1.WriteString(FileContents)
	assert.NoError(t.T(), err)
	t.f2, err = os.OpenFile(path.Join(mntDir, explicitDirName, emptyFileName), os.O_RDWR|syscall.O_DIRECT, filePerms)
	assert.Equal(t.T(), nil, err)
	_, err = t.f2.WriteString(FileContents)
	assert.NoError(t.T(), err)

	// Attempt to remove explicit directory.
	err = os.RemoveAll(path.Join(mntDir, explicitDirName))

	// Verify rmDir operation succeeds.
	assert.NoError(t.T(), err)
	operations.ValidateNoFileOrDirError(t.T(), path.Join(explicitDirName, emptyFileName))
	operations.ValidateNoFileOrDirError(t.T(), path.Join(explicitDirName, nonEmptyFileName))
	operations.ValidateNoFileOrDirError(t.T(), path.Join(explicitDirName, fileName))
	operations.ValidateNoFileOrDirError(t.T(), explicitDirName)
	// Validate flush file throws IO error and does not create object on GCS
	// Close the file and validate that file is not created on GCS.
	err = operations.CloseLocalFile(t.T(), &t.f1)
	assert.NoError(nil, err)
	operations.ValidateObjectNotFoundErr(ctx, t.T(), bucket, path.Join(explicitDirName, emptyFileName))
	operations.ValidateObjectNotFoundErr(ctx, t.T(), bucket, path.Join(explicitDirName, nonEmptyFileName))
	operations.ValidateObjectNotFoundErr(ctx, t.T(), bucket, path.Join(explicitDirName, fileName))
	operations.ValidateObjectNotFoundErr(ctx, t.T(), bucket, explicitDirName)
}
