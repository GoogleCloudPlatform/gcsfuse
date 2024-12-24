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

package inode

import (
	"context"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/contentcache"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/fs/gcsfuse_errors"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/syncutil"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const localFile = "local"
const emptyGCSFile = "emptyGCS"

type FileStreamingWritesTest struct {
	suite.Suite
	ctx        context.Context
	bucket     gcs.Bucket
	clock      timeutil.SimulatedClock
	backingObj *gcs.MinObject
	in         *FileInode
}

func TestFileStreamingWritesTestSuite(t *testing.T) {
	suite.Run(t, new(FileStreamingWritesTest))
}

func (t *FileStreamingWritesTest) SetupTest() {
	// Enabling invariant check for all tests.
	syncutil.EnableInvariantChecking()
	t.ctx = context.Background()
	t.clock.SetTime(time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local))
	t.bucket = fake.NewFakeBucket(&t.clock, "some_bucket", gcs.NonHierarchical)

	// Create the inode.
	t.createInode(fileName, localFile)
}

func (t *FileStreamingWritesTest) TearDownTest() {
	t.in.Unlock()
}

func (t *FileStreamingWritesTest) createInode(fileName string, fileType string) {
	if fileType != emptyGCSFile && fileType != localFile {
		t.T().Errorf("fileType should be either local or empty")
	}

	name := NewFileName(
		NewRootName(""),
		fileName,
	)
	syncerBucket := gcsx.NewSyncerBucket(
		1, // Append threshold
		ChunkTransferTimeoutSecs,
		".gcsfuse_tmp/",
		t.bucket)

	isLocal := false
	if fileType == localFile {
		t.backingObj = nil
		isLocal = true
	}

	if fileType == emptyGCSFile {
		object, err := storageutil.CreateObject(
			t.ctx,
			t.bucket,
			fileName,
			[]byte{})
		t.backingObj = storageutil.ConvertObjToMinObject(object)

		assert.Nil(t.T(), err)
	}

	t.in = NewFileInode(
		fileInodeID,
		name,
		t.backingObj,
		fuseops.InodeAttributes{
			Uid:  uid,
			Gid:  gid,
			Mode: fileMode,
		},
		&syncerBucket,
		false, // localFileCache
		contentcache.New("", &t.clock),
		&t.clock,
		isLocal,
		&cfg.Config{})

	// Set buffered write config for created inode.
	t.in.config = &cfg.Config{Write: cfg.WriteConfig{
		MaxBlocksPerFile:                  5,
		BlockSizeMb:                       1,
		ExperimentalEnableStreamingWrites: true,
		GlobalMaxBlocks:                   10,
	}}

	// Create write handler for the local inode created above.
	err := t.in.CreateBufferedOrTempWriter(t.ctx)
	assert.Nil(t.T(), err)

	t.in.Lock()
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *FileStreamingWritesTest) TestOutOfOrderWritesToLocalFileFallBackToTempFile() {
	testCases := []struct {
		name            string
		offset          int64
		expectedContent string
	}{
		{
			name:            "ahead_of_current_offset",
			offset:          5,
			expectedContent: "taco\x00hello",
		},
		{
			name:            "zero_offset",
			offset:          0,
			expectedContent: "hello",
		},
		{
			name:            "before_current_offset",
			offset:          2,
			expectedContent: "tahello",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			assert.True(t.T(), t.in.IsLocal())
			createTime := t.in.mtimeClock.Now()
			err := t.in.Write(t.ctx, []byte("taco"), 0)
			require.Nil(t.T(), err)
			require.NotNil(t.T(), t.in.bwh)
			assert.Equal(t.T(), int64(4), t.in.bwh.WriteFileInfo().TotalSize)

			err = t.in.Write(t.ctx, []byte("hello"), tc.offset)
			require.Nil(t.T(), err)

			// Ensure bwh cleared and temp file created.
			assert.Nil(t.T(), t.in.bwh)
			assert.NotNil(t.T(), t.in.content)
			// The inode should agree about the new mtime and size.
			attrs, err := t.in.Attributes(t.ctx)
			require.Nil(t.T(), err)
			assert.Equal(t.T(), uint64(len(tc.expectedContent)), attrs.Size)
			assert.WithinDuration(t.T(), attrs.Mtime, createTime, Delta)
			// sync file and validate content
			err = t.in.Sync(t.ctx)
			require.Nil(t.T(), err)
			// Read the object's contents.
			contents, err := storageutil.ReadObject(t.ctx, t.bucket, t.in.Name().GcsObjectName())
			assert.Nil(t.T(), err)
			assert.Equal(t.T(), tc.expectedContent, string(contents))
			t.TearDownTest()
			t.SetupTest()
		})
	}
}

func (t *FileStreamingWritesTest) TestOutOfOrderWritesOnClobberedFileThrowsError() {
	err := t.in.Write(t.ctx, []byte("hi"), 0)
	require.Nil(t.T(), err)
	require.NotNil(t.T(), t.in.bwh)
	assert.Equal(t.T(), int64(2), t.in.bwh.WriteFileInfo().TotalSize)
	// Clobber the file.
	objWritten, err := storageutil.CreateObject(t.ctx, t.bucket, fileName, []byte("taco"))
	require.Nil(t.T(), err)

	err = t.in.Write(t.ctx, []byte("hello"), 10)

	require.Error(t.T(), err)
	var fileClobberedError *gcsfuse_errors.FileClobberedError
	assert.ErrorAs(t.T(), err, &fileClobberedError)
	// Validate Object on GCS not updated.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
	objGot, _, err := t.bucket.StatObject(t.ctx, statReq)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), storageutil.ConvertObjToMinObject(objWritten), objGot)
}

func (t *FileStreamingWritesTest) TestUnlinkLocalFileBeforeWrite() {
	assert.True(t.T(), t.in.IsLocal())

	// Unlink.
	t.in.Unlink()

	assert.True(t.T(), t.in.unlinked)
	assert.Nil(t.T(), t.in.bwh)
	// Data shouldn't be updated to GCS.
	operations.ValidateObjectNotFoundErr(t.ctx, t.T(), t.bucket, t.in.Name().GcsObjectName())
}

func (t *FileStreamingWritesTest) TestUnlinkLocalFileAfterWrite() {
	assert.True(t.T(), t.in.IsLocal())
	// Write some content.
	err := t.in.Write(t.ctx, []byte("tacos"), 0)
	assert.Nil(t.T(), err)
	assert.NotNil(t.T(), t.in.bwh)

	// Unlink.
	t.in.Unlink()

	assert.True(t.T(), t.in.IsUnlinked())
	assert.Nil(t.T(), t.in.bwh)
	// Data shouldn't be updated to GCS.
	operations.ValidateObjectNotFoundErr(t.ctx, t.T(), t.bucket, t.in.Name().GcsObjectName())
}

func (t *FileStreamingWritesTest) TestUnlinkEmptySyncedFile() {
	t.createInode(fileName, emptyGCSFile)
	assert.False(t.T(), t.in.IsLocal())
	// Write some content to temp file.
	err := t.in.Write(t.ctx, []byte("tacos"), 0)
	assert.Nil(t.T(), err)
	assert.NotNil(t.T(), t.in.bwh)

	// Unlink.
	t.in.Unlink()

	assert.True(t.T(), t.in.unlinked)
	assert.Nil(t.T(), t.in.bwh)
}
