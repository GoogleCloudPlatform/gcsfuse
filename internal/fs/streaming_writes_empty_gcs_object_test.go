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

// Streaming write tests for synced empty object.

package fs_test

import (
	"os"
	"path"
	"strings"
	"syscall"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type StreamingWritesEmptyGCSObjectTest struct {
	StreamingWritesCommonTest
}

func (t *StreamingWritesEmptyGCSObjectTest) SetupSuite() {
	t.serverCfg.NewConfig = &cfg.Config{
		Write: cfg.WriteConfig{
			BlockSizeMb:           10,
			CreateEmptyFile:       false,
			EnableStreamingWrites: true,
			GlobalMaxBlocks:       20,
			MaxBlocksPerFile:      10,
		},
	}
	t.fsTest.SetUpTestSuite()
}

func (t *StreamingWritesEmptyGCSObjectTest) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
}
func (t *StreamingWritesEmptyGCSObjectTest) SetupTest() {
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

	// Validate that file exists on GCS.
	_, err = storageutil.ReadObject(ctx, bucket, fileName)
	assert.NoError(t.T(), err)
}

func (t *StreamingWritesEmptyGCSObjectTest) TestRenameFileWithPendingWrites() {
	_, err := t.f1.Write([]byte("tacos"))
	assert.NoError(t.T(), err)
	newFilePath := path.Join(mntDir, "test.txt")
	// Check that new file doesn't exist.
	_, err = os.Stat(newFilePath)
	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))

	err = os.Rename(t.f1.Name(), newFilePath)

	assert.NoError(t.T(), err)
	_, err = os.Stat(t.f1.Name())
	assert.Error(t.T(), err)
	assert.True(t.T(), strings.Contains(err.Error(), "no such file or directory"))
	content, err := os.ReadFile(newFilePath)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), "tacos", string(content))
}

func (t *StreamingWritesEmptyGCSObjectTest) TearDownTest() {
	t.fsTest.TearDown()
}

func TestStreamingWritesEmptyObjectTest(t *testing.T) {
	suite.Run(t, new(StreamingWritesEmptyGCSObjectTest))
}
