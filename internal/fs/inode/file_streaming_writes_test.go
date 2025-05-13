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
	"bytes"
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
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
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

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type FileStreamingWritesCommon struct {
	suite.Suite
	ctx        context.Context
	bucket     gcs.Bucket
	clock      timeutil.SimulatedClock
	backingObj *gcs.MinObject
	in         *FileInode
}
type FileStreamingWritesTest struct {
	FileStreamingWritesCommon
}

type FileStreamingWritesZonalBucketTest struct {
	FileStreamingWritesCommon
}

// //////////////////////////////////////////////////////////////////////
// Helper
// //////////////////////////////////////////////////////////////////////

func (t *FileStreamingWritesCommon) createBufferedWriteHandler() {
	// Initialize BWH for local inode created above.
	initialized, err := t.in.InitBufferedWriteHandlerIfEligible(t.ctx)
	require.NoError(t.T(), err)
	assert.True(t.T(), initialized)
	assert.NotNil(t.T(), t.in.bwh)
}

func (t *FileStreamingWritesCommon) setupTest() {
	// Enabling invariant check for all tests.
	syncutil.EnableInvariantChecking()
	t.ctx = context.Background()
	// Create the inode.
	t.createInode(fileName, localFile)
}

func (t *FileStreamingWritesZonalBucketTest) SetupTest() {
	t.clock.SetTime(time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local))
	t.bucket = fake.NewFakeBucket(&t.clock, "some_bucket", gcs.BucketType{Zonal: true})
	t.setupTest()
}

func (t *FileStreamingWritesTest) SetupTest() {
	t.clock.SetTime(time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local))
	t.bucket = fake.NewFakeBucket(&t.clock, "some_bucket", gcs.BucketType{Zonal: false})
	t.setupTest()
}

func (t *FileStreamingWritesCommon) TearDownTest() {
	t.in.Unlock()
}

func (t *FileStreamingWritesTest) SetupSubTest() {
	t.SetupTest()
}

func (t *FileStreamingWritesCommon) createInode(fileName string, fileType string) {
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

	t.in.Lock()
}

////////////////////////////////////////////////////////////////////////
// Common Tests
////////////////////////////////////////////////////////////////////////

func (t *FileStreamingWritesCommon) TestflushUsingBufferedWriteHandlerOnZeroSizeRecreatesBwhOnInitAgain() {
	t.createBufferedWriteHandler()
	err := t.in.flushUsingBufferedWriteHandler()
	require.NoError(t.T(), err)
	assert.Nil(t.T(), t.in.bwh)

	initialized, err := t.in.InitBufferedWriteHandlerIfEligible(t.ctx)

	require.NoError(t.T(), err)
	assert.True(t.T(), initialized)
	assert.NotNil(t.T(), t.in.bwh)
}

func (t *FileStreamingWritesCommon) TestflushUsingBufferedWriteHandlerOnNonZeroSizeDoesNotRecreatesBwhOnInitAgain() {
	t.createBufferedWriteHandler()
	assert.NoError(t.T(), t.in.Write(t.ctx, []byte("foobar"), 0))
	err := t.in.flushUsingBufferedWriteHandler()
	require.NoError(t.T(), err)
	assert.Nil(t.T(), t.in.bwh)

	initialized, err := t.in.InitBufferedWriteHandlerIfEligible(t.ctx)

	require.NoError(t.T(), err)
	assert.False(t.T(), initialized)
	assert.Nil(t.T(), t.in.bwh)
}

////////////////////////////////////////////////////////////////////////
// Tests (Zonal Bucket)
////////////////////////////////////////////////////////////////////////

func TestFileStreamingWritesWithZonalBucketTestSuite(t *testing.T) {
	suite.Run(t, new(FileStreamingWritesZonalBucketTest))
}

func (t *FileStreamingWritesZonalBucketTest) TestSourceGenerationIsAuthoritativeReturnsTrueForZonalBuckets() {
	assert.True(t.T(), t.in.SourceGenerationIsAuthoritative())
}

func (t *FileStreamingWritesZonalBucketTest) TestSourceGenerationIsAuthoritativeReturnsFalseAfterWriteForZonalBuckets() {
	t.createBufferedWriteHandler()
	assert.NoError(t.T(), t.in.Write(t.ctx, []byte("taco"), 0))

	assert.False(t.T(), t.in.SourceGenerationIsAuthoritative())
}

func (t *FileStreamingWritesZonalBucketTest) TestSyncPendingBufferedWritesForZonalBucketsPromotesInodeToNonLocal() {
	t.createBufferedWriteHandler()
	assert.NoError(t.T(), t.in.Write(t.ctx, []byte("pizza"), 0))

	gcsSynced, err := t.in.SyncPendingBufferedWrites()

	require.NoError(t.T(), err)
	assert.True(t.T(), gcsSynced)
	assert.False(t.T(), t.in.IsLocal())
	content, err := storageutil.ReadObject(t.ctx, t.bucket, t.in.Name().GcsObjectName())
	require.NoError(t.T(), err)
	assert.Equal(t.T(), "pizza", string(content))
}

func (t *FileStreamingWritesZonalBucketTest) TestSyncPendingBufferedWritesForZonalBucketsUpdatesSrcSize() {
	t.createBufferedWriteHandler()
	assert.NoError(t.T(), t.in.Write(t.ctx, []byte("foobar"), 0))
	assert.Equal(t.T(), uint64(0), t.in.src.Size)

	gcsSynced, err := t.in.SyncPendingBufferedWrites()

	require.NoError(t.T(), err)
	assert.True(t.T(), gcsSynced)
	assert.Equal(t.T(), uint64(6), t.in.src.Size)
}

// //////////////////////////////////////////////////////////////////////
// Tests (Non Zonal Bucket)
// //////////////////////////////////////////////////////////////////////

func TestFileStreamingWritesTestSuite(t *testing.T) {
	suite.Run(t, new(FileStreamingWritesTest))
}

func (t *FileStreamingWritesTest) TestSourceGenerationIsAuthoritativeReturnsTrueForNonZonalBuckets() {
	assert.True(t.T(), t.in.SourceGenerationIsAuthoritative())
}

func (t *FileStreamingWritesTest) TestSourceGenerationIsAuthoritativeReturnsFalseAfterWriteForNonZonalBuckets() {
	t.createBufferedWriteHandler()
	assert.NoError(t.T(), t.in.Write(t.ctx, []byte("taco"), 0))

	assert.False(t.T(), t.in.SourceGenerationIsAuthoritative())
}

func (t *FileStreamingWritesTest) TestSyncPendingBufferedWritesForNonZonalBucketsDoesNotPromoteInodeToNonLocal() {
	t.createBufferedWriteHandler()
	assert.NoError(t.T(), t.in.Write(t.ctx, []byte("taco"), 0))

	gcsSynced, err := t.in.SyncPendingBufferedWrites()

	require.NoError(t.T(), err)
	assert.False(t.T(), gcsSynced)
	assert.True(t.T(), t.in.IsLocal())
	operations.ValidateObjectNotFoundErr(t.ctx, t.T(), t.bucket, t.in.Name().GcsObjectName())
}

func (t *FileStreamingWritesTest) TestSyncPendingBufferedWritesForNonZonalBucketsDoesNotUpdateSrcSize() {
	t.createBufferedWriteHandler()
	assert.NoError(t.T(), t.in.Write(t.ctx, []byte("foobar"), 0))
	assert.Equal(t.T(), uint64(0), t.in.src.Size)

	gcsSynced, err := t.in.SyncPendingBufferedWrites()

	require.NoError(t.T(), err)
	assert.False(t.T(), gcsSynced)
	assert.Equal(t.T(), uint64(0), t.in.src.Size)
}

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
			t.createBufferedWriteHandler()
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

func (t *FileStreamingWritesTest) TestFallBackToTempFileDueToBlockConstraints() {
	tests := []struct {
		name                string
		initialTruncateSize int64
	}{
		{
			name:                "NoTruncate",
			initialTruncateSize: 0, // No initial truncate
		},
		{
			name:                "Truncate",
			initialTruncateSize: 1000,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func() {
			t.in.globalMaxWriteBlocksSem = semaphore.NewWeighted(0)
			t.createBufferedWriteHandler()
			assert.True(t.T(), t.in.IsLocal())
			// Ensure bwh exists and temp file is not there.
			assert.Nil(t.T(), t.in.content)
			assert.NotNil(t.T(), t.in.bwh)
			t.clock.AdvanceTime(15 * time.Minute)
			mtime := t.clock.Now()

			if tc.initialTruncateSize != 0 {
				err := t.in.Truncate(t.ctx, tc.initialTruncateSize)
				require.NoError(t.T(), err)
			}

			// Sequential Write at truncated offset
			err := t.in.Write(t.ctx, []byte("taco"), tc.initialTruncateSize)
			require.NoError(t.T(), err)
			// Ensure bwh cleared and temp file created.
			assert.Nil(t.T(), t.in.bwh)
			assert.NotNil(t.T(), t.in.content)
			// validate attributes.
			attrs, err := t.in.Attributes(t.ctx)
			require.Nil(t.T(), err)
			assert.WithinDuration(t.T(), attrs.Mtime, mtime, 0)
			assert.Equal(t.T(), uint64(tc.initialTruncateSize+4), attrs.Size)
			// sync file and validate content
			gcsSynced, err := t.in.Sync(t.ctx)
			require.Nil(t.T(), err)
			assert.True(t.T(), gcsSynced)
			// Read the object's contents.
			contents, err := storageutil.ReadObject(t.ctx, t.bucket, t.in.Name().GcsObjectName())
			assert.NoError(t.T(), err)
			nullBytes := bytes.Repeat([]byte{0}, int(tc.initialTruncateSize))
			assert.Equal(t.T(), string(nullBytes)+"taco", string(contents))
		})
	}
}

func (t *FileStreamingWritesTest) TestOutOfOrderWriteFollowedByOrderedWrite() {
	t.createBufferedWriteHandler()
	assert.True(t.T(), t.in.IsLocal())
	createTime := t.in.mtimeClock.Now()
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
	require.NoError(t.T(), err)
	assert.Equal(t.T(), "hello\x00taco", string(contents))
}

func (t *FileStreamingWritesTest) TestOutOfOrderWritesOnClobberedFileThrowsError() {
	t.createBufferedWriteHandler()
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
	require.NoError(t.T(), err)
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
	t.createBufferedWriteHandler()
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
	t.createBufferedWriteHandler()
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
			t.createBufferedWriteHandler()
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
			require.NoError(t.T(), err)
			assert.Equal(t.T(), uint64(len("tacos")), attrs.Size)
			assert.Equal(t.T(), t.clock.Now().UTC(), attrs.Mtime.UTC())
			// Validate Object on GCS.
			statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
			m, _, err := t.bucket.StatObject(t.ctx, statReq)
			require.NoError(t.T(), err)
			assert.NotNil(t.T(), m)
			assert.Equal(t.T(), t.in.SourceGeneration().Object, m.Generation)
			assert.Equal(t.T(), t.in.SourceGeneration().Metadata, m.MetaGeneration)
			assert.Equal(t.T(), uint64(len("tacos")), m.Size)
			// Mtime metadata is not written for buffered writes.
			assert.Equal(t.T(), "", m.Metadata["gcsfuse_mtime"])
			contents, err := storageutil.ReadObject(t.ctx, t.bucket, t.in.Name().GcsObjectName())
			require.NoError(t.T(), err)
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
			t.createBufferedWriteHandler()

			err := t.in.Flush(t.ctx)

			require.Nil(t.T(), err)
			// Ensure bwh cleared.
			assert.Nil(t.T(), t.in.bwh)
			// Verify that fileInode is no more local
			assert.False(t.T(), t.in.IsLocal())
			// Check attributes.
			attrs, err := t.in.Attributes(t.ctx)
			require.NoError(t.T(), err)
			assert.Equal(t.T(), uint64(0), attrs.Size)
			// For synced file, mtime is updated by SetInodeAttributes call.
			if tc.isLocal {
				assert.Equal(t.T(), t.clock.Now().UTC(), attrs.Mtime.UTC())
			}
			// Validate Object on GCS.
			statReq := &gcs.StatObjectRequest{Name: t.in.Name().GcsObjectName()}
			m, _, err := t.bucket.StatObject(t.ctx, statReq)
			require.NoError(t.T(), err)
			assert.NotNil(t.T(), m)
			assert.Equal(t.T(), t.in.SourceGeneration().Object, m.Generation)
			assert.Equal(t.T(), t.in.SourceGeneration().Metadata, m.MetaGeneration)
			assert.Equal(t.T(), uint64(0), m.Size)
			// Mtime metadata is not written for buffered writes.
			assert.Equal(t.T(), "", m.Metadata["gcsfuse_mtime"])
			contents, err := storageutil.ReadObject(t.ctx, t.bucket, t.in.Name().GcsObjectName())
			require.NoError(t.T(), err)
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
			t.createBufferedWriteHandler()
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
			t.createBufferedWriteHandler()
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
				require.NoError(t.T(), err)
				assert.NotNil(t.T(), m)
				assert.Equal(t.T(), uint64(0), m.Size)
			}
		})
	}
}

func (t *FileStreamingWritesTest) TestSourceGenerationSizeForLocalFileIsReflected() {
	t.createBufferedWriteHandler()
	assert.True(t.T(), t.in.IsLocal())
	err := t.in.Write(context.Background(), []byte(setup.GenerateRandomString(5)), 0)
	require.NoError(t.T(), err)

	sg := t.in.SourceGeneration()
	assert.Nil(t.T(), t.backingObj)
	assert.EqualValues(t.T(), 0, sg.Object)
	assert.EqualValues(t.T(), 0, sg.Metadata)
	assert.EqualValues(t.T(), 5, sg.Size)
}

func (t *FileStreamingWritesTest) TestSourceGenerationSizeForSyncedFileIsReflected() {
	t.createInode(fileName, emptyGCSFile)
	assert.False(t.T(), t.in.IsLocal())
	t.createBufferedWriteHandler()
	err := t.in.Write(context.Background(), []byte(setup.GenerateRandomString(5)), 0)
	require.NoError(t.T(), err)

	sg := t.in.SourceGeneration()
	assert.EqualValues(t.T(), t.backingObj.Generation, sg.Object)
	assert.EqualValues(t.T(), t.backingObj.MetaGeneration, sg.Metadata)
	assert.EqualValues(t.T(), 5, sg.Size)
}

func (t *FileStreamingWritesTest) TestTruncateOnFileUsingTempFileDoesNotRecreatesBWH() {
	t.createBufferedWriteHandler()
	assert.True(t.T(), t.in.IsLocal())
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
	require.NoError(t.T(), err)
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
			t.createBufferedWriteHandler()

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

// FakeBufferedWriteHandler is a test double for BufferedWriteHandler.
type FakeBufferedWriteHandler struct {
	WriteFunc func(data []byte, offset int64) error
	FlushFunc func() (*gcs.MinObject, error)
}

func (t *FakeBufferedWriteHandler) Write(data []byte, offset int64) error {
	if t.WriteFunc != nil {
		return t.WriteFunc(data, offset)
	}
	return nil
}

func (t *FakeBufferedWriteHandler) Flush() (*gcs.MinObject, error) {
	if t.FlushFunc != nil {
		return t.FlushFunc()
	}
	return nil, nil
}

func (t *FakeBufferedWriteHandler) WriteFileInfo() bufferedwrites.WriteFileInfo {
	return bufferedwrites.WriteFileInfo{
		TotalSize: 0,
		Mtime:     time.Time{},
	}
}

func (t *FakeBufferedWriteHandler) Sync() (*gcs.MinObject, error) { return nil, nil }
func (t *FakeBufferedWriteHandler) SetMtime(_ time.Time)          {}
func (t *FakeBufferedWriteHandler) Truncate(_ int64) error        { return nil }
func (t *FakeBufferedWriteHandler) Destroy() error                { return nil }
func (t *FakeBufferedWriteHandler) Unlink()                       {}

func (t *FileStreamingWritesTest) TestWriteUsingBufferedWritesFails() {
	t.createBufferedWriteHandler()
	assert.True(t.T(), t.in.IsLocal())
	writeErr := errors.New("write error")
	t.in.bwh = &FakeBufferedWriteHandler{
		WriteFunc: func(data []byte, offset int64) error {
			return writeErr
		},
	}

	err := t.in.Write(context.Background(), []byte("hello"), 0)

	require.Error(t.T(), err)
	assert.Regexp(t.T(), writeErr.Error(), err.Error())
}
