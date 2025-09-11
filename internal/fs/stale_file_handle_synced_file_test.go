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
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// //////////////////////////////////////////////////////////////////////
// Boilerplate
// //////////////////////////////////////////////////////////////////////

type staleFileHandleSyncedFile struct {
	staleFileHandleCommon
}

// //////////////////////////////////////////////////////////////////////
// Helpers
// //////////////////////////////////////////////////////////////////////

func (t *staleFileHandleSyncedFile) SetupTest() {
	// Create an object on bucket.
	t.f1 = createGCSObject(t.T(), "bar")
}

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////

func (t *staleFileHandleSyncedFile) TestClobberedFileReadThrowsStaleFileHandleError() {
	// Replace the underlying object with a new generation.
	clobberFile(t.T(), "foobar")

	buffer := make([]byte, 6)
	_, err := t.f1.Read(buffer)

	operations.ValidateESTALEError(t.T(), err)
	// Validate that object is updated with new content.
	contents, err := storageutil.ReadObject(ctx, bucket, "foo")
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), "foobar", string(contents))
}

func (t *staleFileHandleSyncedFile) TestClobberedFileFirstWriteThrowsStaleFileHandleError() {
	// Replace the underlying object with a new generation.
	clobberFile(t.T(), "foobar")

	_, err := t.f1.Write([]byte("taco"))

	operations.ValidateESTALEError(t.T(), err)
	// Attempt to sync to file should not result in error as we first check if the
	// content has been dirtied before clobbered check in Sync flow.
	err = t.f1.Sync()
	assert.NoError(t.T(), err)
	// Validate that object is not updated with new content as write failed.
	contents, err := storageutil.ReadObject(ctx, bucket, "foo")
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), "foobar", string(contents))
}

func (t *staleFileHandleSyncedFile) TestRenamedFileWriteThrowsStaleFileHandleError() {
	// Dirty the file by giving it some contents.
	n, err := t.f1.Write([]byte("foobar"))
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 6, n)
	// Rename the object.
	err = os.Rename(t.f1.Name(), path.Join(mntDir, "bar"))
	assert.NoError(t.T(), err)

	// Attempt to write to file should give ESTALE error.
	_, err = t.f1.Write([]byte("taco"))
	operations.ValidateESTALEError(t.T(), err)
	// No error on sync and close because no data was written.
	err = t.f1.Sync()
	require.NoError(t.T(), err)
	err = t.f1.Close()
	require.NoError(t.T(), err)
	// Make f1 nil, so that another attempt is not taken in TearDown to close the
	// file.
	t.f1 = nil
}

func (t *staleFileHandleSyncedFile) TestFileDeletedRemotelySyncAndCloseThrowsStaleFileHandleError() {
	// Dirty the file by giving it some contents.
	n, err := t.f1.Write([]byte("foobar"))
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 6, n)
	// Unlink the file.
	err = storageutil.DeleteObject(ctx, bucket, "foo")
	assert.NoError(t.T(), err)
	// Verify unlink operation succeeds.
	assert.NoError(t.T(), err)
	operations.ValidateObjectNotFoundErr(ctx, t.T(), bucket, "foo")
	// Attempt to write to file should not give any error.
	n, err = t.f1.Write([]byte("taco"))
	assert.Equal(t.T(), 4, n)
	assert.NoError(t.T(), err)

	err = t.f1.Sync()

	operations.ValidateESTALEError(t.T(), err)
	err = t.f1.Close()
	operations.ValidateESTALEError(t.T(), err)
	// Make f1 nil, so that another attempt is not taken in TearDown to close the
	// file.
	t.f1 = nil
}

// Executes all stale handle tests for gcs synced files.
func TestStaleFileHandleSyncedFile(t *testing.T) {
	suite.Run(t, new(staleFileHandleSyncedFile))
}
