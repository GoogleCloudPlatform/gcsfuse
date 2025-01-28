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
	"errors"
	"math"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/bufferedwrites"
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
	t.bucket = fake.NewFakeBucket(&t.clock, "some_bucket", gcs.BucketType{})

	// Create the inode.
	t.createInode(fileName, localFile)
}

func (t *FileStreamingWritesTest) TearDownTest() {
	t.in.Unlock()
}

func (t *FileStreamingWritesTest) SetupSubTest() {
	t.SetupTest()
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
		&cfg.Config{},
		semaphore.NewWeighted(math.MaxInt64))

	// Set buffered write config for created inode.
	t.in.config = &cfg.Config{Write: cfg.WriteConfig{
		MaxBlocksPerFile:      5,
		BlockSizeMb:           1,
		EnableStreamingWrites: true,
		GlobalMaxBlocks:       10,
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
			createTime := t.clock.Now()
			t.clock.AdvanceTime(15 * time.Minute)
			// Sequential Write at offset 0
			err := t.in.Write(t.ctx, []byte("taco"), 0)
			require.Nil(t.T(), err)
			require.NotNil(t.T(), t.in.bwh)
			// validate attributes.
			attrs, err := t.in.Attributes(t.ctx)
			require.Nil(t.T(), err)
			assert.WithinDuration(t.T(), attrs.Mtime, createTime, 0)
			assert.Equal(t.T(), uint64(4), attrs.Size)

			// Out of order write.
			mtime := t.clock.Now()
			err = t.in.Write(t.ctx, []byte("hello"), tc.offset)
			require.Nil(t.T(), err)

			// Ensure bwh cleared and temp file created.
			assert.Nil(t.T(), t.in.bwh)
			assert.NotNil(t.T(), t.in.content)
			// The inode should agree about the new mtime and size.
			attrs, err = t.in.Attributes(t.ctx)
			require.Nil(t.T(), err)
			assert.Equal(t.T(), uint64(len(tc.expectedContent)), attrs.Size)
			assert.WithinDuration(t.T(), attrs.Mtime, mtime, 0)
			// sync file and validate content
			gcsSynced, err := t.in.Sync(t.ctx)
			require.Nil(t.T(), err)
			assert.True(t.T(), gcsSynced)
			// Read the object's contents.
			contents, err := storageutil.ReadObject(t.ctx, t.bucket, t.in.Name().GcsObjectName())
			assert.Nil(t.T(), err)
			assert.Equal(t.T(), tc.expectedContent, string(contents))
		})
	}
}

func (t *FileStreamingWritesTest) TestOutOfOrderWriteFollowedByOrderedWrite() {
	assert.True(t.T(), t.in.IsLocal())
	createTime := t.in.mtimeClock.Now()
	require.NotNil(t.T(), t.in.bwh)
	// Out of order write.
	err := t.in.Write(t.ctx, []byte("taco"), 6)
	require.Nil(t.T(), err)
	// Ensure bwh cleared and temp file created.
	assert.Nil(t.T(), t.in.bwh)
	assert.NotNil(t.T(), t.in.content)
	// validate attributes.
	attrs, err := t.in.Attributes(t.ctx)
	require.Nil(t.T(), err)
	assert.WithinDuration(t.T(), attrs.Mtime, createTime, 0)
	assert.Equal(t.T(), uint64(10), attrs.Size)

	// Ordered write.
	mtime := t.clock.Now()
	err = t.in.Write(t.ctx, []byte("hello"), 0)
	require.Nil(t.T(), err)

	// Ensure bwh not re-created.
	assert.Nil(t.T(), t.in.bwh)
	// The inode should agree about the new mtime and size.
	attrs, err = t.in.Attributes(t.ctx)
	require.Nil(t.T(), err)
	assert.Equal(t.T(), uint64(len("hello\x00taco")), attrs.Size)
	assert.WithinDuration(t.T(), attrs.Mtime, mtime, 0)
	// sync file and validate content
	gcsSynced, err := t.in.Sync(t.ctx)
	require.Nil(t.T(), err)
	assert.True(t.T(), gcsSynced)
	// Read the object's contents.
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, t.in.Name().GcsObjectName())
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), "hello\x00taco", string(contents))
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
}

func (t *FileStreamingWritesTest) TestWriteToFileAndFlush() {
	testCases := []struct {
		name    string
		isLocal bool
	}{
		{
			name:    "local_file",
			isLocal: true,
		},
		{
			name:    "synced_empty_file",
			isLocal: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			if tc.isLocal {
				assert.True(t.T(), t.in.IsLocal())
			} else {
				t.createInode(fileName, emptyGCSFile)
				assert.False(t.T(), t.in.IsLocal())
			}
			// Write some content to temp file.
			err := t.in.Write(t.ctx, []byte("tacos"), 0)
			assert.Nil(t.T(), err)
			assert.NotNil(t.T(), t.in.bwh)
			t.clock.AdvanceTime(10 * time.Second)

			err = t.in.Flush(t.ctx)

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
		})
	}
}

func (t *FileStreamingWritesTest) TestFlushEmptyFile() {
	testCases := []struct {
		name    string
		isLocal bool
	}{
		{
			name:    "local_file",
			isLocal: true,
		},
		{
			name:    "synced_empty_file",
			isLocal: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			if tc.isLocal {
				assert.True(t.T(), t.in.IsLocal())
			} else {
				t.createInode(fileName, emptyGCSFile)
				assert.False(t.T(), t.in.IsLocal())
			}
			t.clock.AdvanceTime(10 * time.Second)

			err := t.in.Flush(t.ctx)

			require.Nil(t.T(), err)
			// Ensure bwh cleared.
			assert.Nil(t.T(), t.in.bwh)
			// Verify that fileInode is no more local
			assert.False(t.T(), t.in.IsLocal())
			// Check attributes.
			attrs, err := t.in.Attributes(t.ctx)
			assert.Nil(t.T(), err)
			assert.Equal(t.T(), uint64(0), attrs.Size)
			// For synced file, mtime is updated by SetInodeAttributes call.
			if tc.isLocal {
				assert.Equal(t.T(), t.clock.Now().UTC(), attrs.Mtime.UTC())
			}
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
		})
	}
}

func (t *FileStreamingWritesTest) TestFlushClobberedFile() {
	testCases := []struct {
		name    string
		isLocal bool
	}{
		{
			name:    "local_file",
			isLocal: true,
		},
		{
			name:    "synced_empty_file",
			isLocal: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			if tc.isLocal {
				assert.True(t.T(), t.in.IsLocal())
			} else {
				t.createInode(fileName, emptyGCSFile)
				assert.False(t.T(), t.in.IsLocal())
			}
			t.clock.AdvanceTime(10 * time.Second)
			// Clobber the file.
			objWritten, err := storageutil.CreateObject(t.ctx, t.bucket, fileName, []byte("taco"))
			require.Nil(t.T(), err)

			err = t.in.Flush(t.ctx)

			require.Error(t.T(), err)
			var fileClobberedError *gcsfuse_errors.FileClobberedError
			assert.ErrorAs(t.T(), err, &fileClobberedError)
			// Validate Object on GCS not updated.
			statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
			objGot, _, err := t.bucket.StatObject(t.ctx, statReq)
			assert.Nil(t.T(), err)
			assert.Equal(t.T(), storageutil.ConvertObjToMinObject(objWritten), objGot)
		})
	}
}

func (t *FileStreamingWritesTest) TestWriteToFileAndSync() {
	testCases := []struct {
		name    string
		isLocal bool
	}{
		{
			name:    "local_file",
			isLocal: true,
		},
		{
			name:    "synced_empty_file",
			isLocal: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			if tc.isLocal {
				assert.True(t.T(), t.in.IsLocal())
			} else {
				t.createInode(fileName, emptyGCSFile)
				assert.False(t.T(), t.in.IsLocal())
			}
			// Write some content to temp file.
			err := t.in.Write(t.ctx, []byte("tacos"), 0)
			assert.Nil(t.T(), err)
			assert.NotNil(t.T(), t.in.bwh)
			t.clock.AdvanceTime(10 * time.Second)

			gcsSynced, err := t.in.Sync(t.ctx)

			require.Nil(t.T(), err)
			assert.False(t.T(), gcsSynced)
			// Ensure bwh not cleared.
			assert.NotNil(t.T(), t.in.bwh)
			// Validate Object not written on GCS.
			statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
			m, _, err := t.bucket.StatObject(t.ctx, statReq)
			if tc.isLocal {
				require.Error(t.T(), err)
				var notFoundErr *gcs.NotFoundError
				assert.ErrorAs(t.T(), err, &notFoundErr)
			} else {
				assert.NoError(t.T(), err)
				assert.NotNil(t.T(), m)
				assert.Equal(t.T(), uint64(0), m.Size)
			}
		})
	}
}

func (t *FileStreamingWritesTest) TestTruncateOnFileUsingTempFileDoesNotRecreatesBWH() {
	assert.True(t.T(), t.in.IsLocal())
	require.NotNil(t.T(), t.in.bwh)
	// Out of order write.
	err := t.in.Write(t.ctx, []byte("taco"), 2)
	require.Nil(t.T(), err)
	// Ensure bwh cleared and temp file created.
	assert.Nil(t.T(), t.in.bwh)
	assert.NotNil(t.T(), t.in.content)

	err = t.in.Truncate(t.ctx, 10)
	require.Nil(t.T(), err)

	// Ensure bwh not re-created.
	assert.Nil(t.T(), t.in.bwh)
	// The inode should agree about the new size.
	attrs, err := t.in.Attributes(t.ctx)
	require.Nil(t.T(), err)
	assert.Equal(t.T(), uint64(10), attrs.Size)
	// sync file and validate content
	gcsSynced, err := t.in.Sync(t.ctx)
	require.Nil(t.T(), err)
	assert.True(t.T(), gcsSynced)
	// Read the object's contents.
	contents, err := storageutil.ReadObject(t.ctx, t.bucket, t.in.Name().GcsObjectName())
	assert.Nil(t.T(), err)
	assert.Equal(t.T(), "\x00\x00taco\x00\x00\x00\x00", string(contents))
}

func (t *FileStreamingWritesTest) TestDeRegisterFileHandle() {
	tbl := []struct {
		name        string
		readonly    bool
		currentVal  int32
		expectedVal int32
		isBwhNil    bool
	}{
		{
			name:        "ReadOnlyHandle",
			readonly:    true,
			currentVal:  10,
			expectedVal: 10,
			isBwhNil:    false,
		},
		{
			name:        "NonZeroCurrentValueForWriteHandle",
			readonly:    false,
			currentVal:  10,
			expectedVal: 9,
			isBwhNil:    false,
		},
		{
			name:        "LastWriteHandleToDeregister",
			readonly:    false,
			currentVal:  1,
			expectedVal: 0,
			isBwhNil:    true,
		},
	}
	for _, tc := range tbl {
		t.Run(tc.name, func() {
			t.in.config = &cfg.Config{Write: *getWriteConfig()}
			t.in.writeHandleCount = tc.currentVal
			err := t.in.initBufferedWriteHandlerIfEligible(t.ctx)
			require.NoError(t.T(), err)
			require.NotNil(t.T(), t.in.bwh)

			t.in.DeRegisterFileHandle(tc.readonly)

			assert.Equal(t.T(), tc.expectedVal, t.in.writeHandleCount)
			if tc.isBwhNil {
				assert.Nil(t.T(), t.in.bwh)
			} else {
				assert.NotNil(t.T(), t.in.bwh)
			}
		})
	}
}

// TestBufferedWriteHandler is a test double for BufferedWriteHandler.
type TestBufferedWriteHandler struct {
	WriteFunc func(data []byte, offset int64) error
	FlushFunc func() (*gcs.MinObject, error)
}

func (t *TestBufferedWriteHandler) Write(data []byte, offset int64) error {
	if t.WriteFunc != nil {
		return t.WriteFunc(data, offset)
	}
	return nil
}

func (t *TestBufferedWriteHandler) Flush() (*gcs.MinObject, error) {
	if t.FlushFunc != nil {
		return t.FlushFunc()
	}
	return nil, nil
}

func (t *TestBufferedWriteHandler) WriteFileInfo() bufferedwrites.WriteFileInfo {
	return bufferedwrites.WriteFileInfo{
		TotalSize: 0,
		Mtime:     time.Time{},
	}
}

func (t *TestBufferedWriteHandler) Sync() (err error)      { return nil }
func (t *TestBufferedWriteHandler) SetMtime(_ time.Time)   {}
func (t *TestBufferedWriteHandler) Truncate(_ int64) error { return nil }
func (t *TestBufferedWriteHandler) Destroy() error         { return nil }
func (t *TestBufferedWriteHandler) Unlink()                {}

func (t *FileStreamingWritesTest) TestWriteUsingBufferedWritesErrorScenarios() {
	assert.True(t.T(), t.in.IsLocal())
	require.NotNil(t.T(), t.in.bwh)

	testCases := []struct {
		name        string
		writeErr    error
		flushErr    error
		expectedErr string
	}{
		{
			name:        "Write_error_flush_succeeds",
			writeErr:    errors.New("write error"),
			flushErr:    nil,
			expectedErr: "write error",
		},
		{
			name:        "Write_error_flush_fails",
			writeErr:    errors.New("write error"),
			flushErr:    errors.New("flush error"),
			expectedErr: "write error.*flush error", // Use regex for multiple errors
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.in.bwh = &TestBufferedWriteHandler{
				WriteFunc: func(data []byte, offset int64) error {
					return tc.writeErr
				},
				FlushFunc: func() (*gcs.MinObject, error) {
					return nil, tc.flushErr
				},
			}

			err := t.in.Write(context.Background(), []byte("hello"), 0)

			require.Error(t.T(), err)
			assert.Regexp(t.T(), tc.expectedErr, err.Error())
		})
	}
}
