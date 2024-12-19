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
	"math"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/contentcache"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/fs/gcsfuse_errors"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/syncutil"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/semaphore"
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
		&cfg.WriteConfig{},
		semaphore.NewWeighted(math.MaxInt64))

	// Set buffered write config for created inode.
	t.in.writeConfig = &cfg.WriteConfig{
		MaxBlocksPerFile:                  10,
		BlockSizeMb:                       10,
		ExperimentalEnableStreamingWrites: true,
	}

	// Create write handler for the local inode created above.
	err := t.in.CreateBufferedOrTempWriter()
	assert.Nil(t.T(), err)

	t.in.Lock()
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *FileStreamingWritesTest) TestOutOfOrderWritesToLocalFileFallBackToTempFile() {
	assert.True(t.T(), t.in.IsLocal())
	createTime := t.in.mtimeClock.Now()
	err := t.in.Write(t.ctx, []byte("hi"), 0)
	require.Nil(t.T(), err)
	require.NotNil(t.T(), t.in.bwh)
	assert.Equal(t.T(), int64(2), t.in.bwh.WriteFileInfo().TotalSize)

	err = t.in.Write(t.ctx, []byte("hello"), 5)
	require.Nil(t.T(), err)

	// Ensure bwh cleared and temp file created.
	assert.Nil(t.T(), t.in.bwh)
	assert.NotNil(t.T(), t.in.content)
	// The inode should agree about the new mtime.
	attrs, err := t.in.Attributes(t.ctx)
	require.Nil(t.T(), err)
	assert.Equal(t.T(), uint64(10), attrs.Size)
	assert.WithinDuration(t.T(), attrs.Mtime, createTime, Delta)
	// sync file and validate content
	err = t.in.Sync(t.ctx)
	require.Nil(t.T(), err)
	// Read the object's contents.
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, t.in.Name().GcsObjectName())
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), "hi\x00\x00\x00hello", string(contents))
}

func (t *FileStreamingWritesTest) TestOutOfOrderWritesToLocalFile_FileClobbered_ThrowsError() {
	assert.True(t.T(), t.in.IsLocal())
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

func (t *FileStreamingWritesTest) TestWriteToLocalFileThenSync() {
	assert.True(t.T(), t.in.IsLocal())
	// Write some content to temp file.
	err := t.in.Write(t.ctx, []byte("tacos"), 0)
	assert.Nil(t.T(), err)
	assert.NotNil(t.T(), t.in.bwh)
	t.clock.AdvanceTime(10 * time.Second)

	// Sync.
	err = t.in.Sync(t.ctx)

	require.Nil(t.T(), err)
	// Ensure bwh cleared.
	assert.Nil(t.T(), t.in.bwh)
	// Verify that fileInode is no more local
	assert.False(t.T(), t.in.IsLocal())
	// Check attributes.
	attrs, err := t.in.Attributes(t.ctx)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), uint64(len("tacos")), attrs.Size)
	assert.Equal(t.T(), t.clock.Now().UTC(), attrs.Mtime.UTC())
	// Validate Object on GCS.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
	m, _, err := t.bucket.StatObject(t.ctx, statReq)
	assert.Nil(t.T(), err)
	assert.NotNil(t.T(), m)
	assert.Equal(t.T(), t.in.SourceGeneration().Object, m.Generation)
	assert.Equal(t.T(), t.in.SourceGeneration().Metadata, m.MetaGeneration)
	assert.Equal(t.T(), uint64(len("tacos")), m.Size)
	// Mtime metadata is not written for buffered writes.
	assert.Equal(t.T(), "", m.Metadata["gcsfuse_mtime"])
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, t.in.Name().GcsObjectName())
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), "tacos", string(contents))
}

func (t *FileStreamingWritesTest) TestSyncEmptyLocalFile() {
	assert.True(t.T(), t.in.IsLocal())
	t.clock.AdvanceTime(10 * time.Second)

	// Sync.
	err := t.in.Sync(t.ctx)

	require.Nil(t.T(), err)
	// Ensure bwh cleared.
	assert.Nil(t.T(), t.in.bwh)
	// Verify that fileInode is no more local
	assert.False(t.T(), t.in.IsLocal())
	// Check attributes.
	attrs, err := t.in.Attributes(t.ctx)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), uint64(0), attrs.Size)
	assert.Equal(t.T(), t.clock.Now().UTC(), attrs.Mtime.UTC())
	// Validate Object on GCS.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
	m, _, err := t.bucket.StatObject(t.ctx, statReq)
	assert.Nil(t.T(), err)
	assert.NotNil(t.T(), m)
	assert.Equal(t.T(), t.in.SourceGeneration().Object, m.Generation)
	assert.Equal(t.T(), t.in.SourceGeneration().Metadata, m.MetaGeneration)
	assert.Equal(t.T(), uint64(0), m.Size)
	// Mtime metadata is not written for buffered writes.
	assert.Equal(t.T(), "", m.Metadata["gcsfuse_mtime"])
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, t.in.Name().GcsObjectName())
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), "", string(contents))
}

func (t *FileStreamingWritesTest) TestSyncClobberedLocalFile() {
	assert.True(t.T(), t.in.IsLocal())
	t.clock.AdvanceTime(10 * time.Second)
	// Clobber the file.
	objWritten, err := storageutil.CreateObject(t.ctx, t.bucket, fileName, []byte("taco"))
	require.Nil(t.T(), err)

	// Sync.
	err = t.in.Sync(t.ctx)

	require.Error(t.T(), err)
	var fileClobberedError *gcsfuse_errors.FileClobberedError
	assert.ErrorAs(t.T(), err, &fileClobberedError)
	// Validate Object on GCS not updated.
	statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
	objGot, _, err := t.bucket.StatObject(t.ctx, statReq)
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), storageutil.ConvertObjToMinObject(objWritten), objGot)
}

// TODO: add tests for empty file. Changes are required to set precondition in bwh.
