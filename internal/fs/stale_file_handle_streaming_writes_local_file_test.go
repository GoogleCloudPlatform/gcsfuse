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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/vipnydav/gcsfuse/v3/internal/storage/storageutil"
	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

type staleFileHandleStreamingWritesLocalFile struct {
	staleFileHandleStreamingWritesCommon
}

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////

func (t *staleFileHandleStreamingWritesLocalFile) SetupTest() {
	// Create a local file.
	_, t.f1 = operations.CreateLocalFile(ctx, t.T(), mntDir, bucket, "foo")
}

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////

func (t *staleFileHandleStreamingWritesLocalFile) TestClobberedWriteFileSyncAndCloseThrowsStaleFileHandleError() {
	// Replace the underlying object with a new generation.
	clobberFile(t.T(), "foobar")
	// Writing to file will return Stale File Handle Error.
	data, err := operations.GenerateRandomData(operations.MiB * 4)
	assert.NoError(t.T(), err)

	_, err = t.f1.WriteAt(data, 0)

	operations.ValidateESTALEError(t.T(), err)
	err = t.f1.Sync()
	operations.ValidateESTALEError(t.T(), err)
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

// Executes all stale handle tests for local files with streaming writes.
func TestStaleFileHandleStreamingWritesLocalFile(t *testing.T) {
	ts := new(staleFileHandleStreamingWritesLocalFile)
	suite.Run(t, ts)
}
