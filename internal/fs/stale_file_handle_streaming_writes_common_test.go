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

package fs_test

import (
	"math"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

type staleFileHandleStreamingWritesCommon struct {
	// fsTest has f1 *osFile and f2 *osFile which we will reuse here.
	fsTest
	suite.Suite
}

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////

func (t *staleFileHandleStreamingWritesCommon) SetupSuite() {
	serverCfg := commonServerConfig()
	serverCfg.Write.EnableStreamingWrites = true
	serverCfg.Write.BlockSizeMb = 1
	serverCfg.Write.MaxBlocksPerFile = 1
	serverCfg.Write.GlobalMaxBlocks = math.MaxInt

	t.serverCfg.NewConfig = serverCfg
	t.mountCfg.DisableWritebackCaching = true

	t.fsTest.SetUpTestSuite()
}

func (t *staleFileHandleStreamingWritesCommon) TearDownTest() {
	// fsTest Cleanups to clean up mntDir and close t.f1 and t.f2.
	t.fsTest.TearDown()
}

func (t *staleFileHandleStreamingWritesCommon) TearDownSuite() {
	t.fsTest.TearDownTestSuite()
}

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////

func (t *staleFileHandleStreamingWritesCommon) TestWriteFileSyncFileClobberedFlushThrowsStaleFileHandleError() {
	// Dirty the file by giving it some contents.
	data, err := operations.GenerateRandomData(operations.MiB * 4)
	assert.NoError(t.T(), err)
	_, err = t.f1.WriteAt(data, 0)
	assert.NoError(t.T(), err)
	err = t.f1.Sync()
	assert.NoError(t.T(), err)
	// Replace the underlying object with a new generation.
	clobberFile(t.T(), "foobar")

	err = t.f1.Close()

	operations.ValidateESTALEError(t.T(), err)
	// Make f1 nil, so that another attempt is not taken in TearDown to close the
	// file.
	t.f1 = nil
	// Validate that object is not updated with un-synced content.
	contents, err := storageutil.ReadObject(ctx, bucket, "foo")
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), "foobar", string(contents))
}
