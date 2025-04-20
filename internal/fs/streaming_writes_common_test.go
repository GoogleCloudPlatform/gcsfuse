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

// Streaming write tests which are common for both local file and synced empty
// object.

package fs_test

import (
	"os"
	"path"
	"strings"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type StreamingWritesCommonTest struct {
	suite.Suite
	fsTest
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *StreamingWritesCommonTest) TestUnlinkBeforeWrite() {
	// unlink the file.
	err := os.Remove(t.f1.Name())
	assert.NoError(t.T(), err)

	// Stat the file and validate file is deleted.
	operations.ValidateNoFileOrDirError(t.T(), t.f1.Name())
	// Close the file and validate that file is deleted from GCS.
	err = t.f1.Close()
	assert.NoError(nil, err)
	t.f1 = nil
	operations.ValidateObjectNotFoundErr(ctx, t.T(), bucket, fileName)
}

func (t *StreamingWritesCommonTest) TestUnlinkAfterWrite() {
	// Write content to file.
	_, err := t.f1.Write([]byte("tacos"))
	assert.NoError(t.T(), err)

	t.TestUnlinkBeforeWrite()
}

func (t *StreamingWritesCommonTest) TestRenameFileWithPendingWrites() {
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
