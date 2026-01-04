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

package inode

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/contentcache"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/fs/gcsfuse_errors"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	storagemock "github.com/googlecloudplatform/gcsfuse/v3/internal/storage/mock"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/syncutil"
	"github.com/jacobsa/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/semaphore"
)

type FileMockBucketTest struct {
	suite.Suite
	ctx        context.Context
	bucket     *storagemock.TestifyMockBucket
	clock      timeutil.SimulatedClock
	backingObj *gcs.MinObject
	in         *FileInode
}

func TestFileMockBucketTestSuite(t *testing.T) {
	suite.Run(t, new(FileMockBucketTest))
}

func (t *FileMockBucketTest) SetupTest() {
	// Enabling invariant check for all tests.
	syncutil.EnableInvariantChecking()
	t.ctx = context.Background()
	t.clock.SetTime(time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local))
	t.bucket = new(storagemock.TestifyMockBucket)
	t.bucket.On("BucketType").Return(gcs.BucketType{Hierarchical: false, Zonal: false})

	// Create the inode.
	t.createLockedInode(fileName, localFile)
}

func (t *FileMockBucketTest) TearDownTest() {
	t.in.Unlock()
}

func (t *FileMockBucketTest) createLockedInode(fileName string, fileType string) {
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
		semaphore.NewWeighted(math.MaxInt64),
		nil)

	// Create empty file for local inode created above.
	err := t.in.CreateEmptyTempFile(t.ctx)
	assert.Nil(t.T(), err)

	t.in.Lock()
}

// createGCSBackedFileInode is a helper function to create and lock a FileInode for attribute tests.
// It initializes the inode with the provided backing object of non-zero size.
func (t *FileMockBucketTest) createGCSBackedFileInode(backingObj *gcs.MinObject) *FileInode {
	t.T().Helper()
	syncerBucket := gcsx.NewSyncerBucket(
		1, // Append threshold
		ChunkTransferTimeoutSecs,
		".gcsfuse_tmp/",
		t.bucket)

	f := NewFileInode(
		fileInodeID,
		NewFileName(NewRootName(""), fileName),
		backingObj,
		fuseops.InodeAttributes{
			Uid:  uid,
			Gid:  gid,
			Mode: fileMode,
		},
		&syncerBucket,
		false, // localFileCache
		contentcache.New("", &t.clock),
		&t.clock,
		false, // localFile
		&cfg.Config{},
		semaphore.NewWeighted(math.MaxInt64),
		nil)
	f.Lock()
	return f
}

func (t *FileMockBucketTest) TestFlushLocalFileDoesNotForceFetchObjectFromGCS() {
	assert.True(t.T(), t.in.IsLocal())
	// Expect only CreateObject call on bucket.
	t.bucket.On("CreateObject", t.ctx, mock.AnythingOfType("*gcs.CreateObjectRequest")).
		Return(&gcs.Object{Name: fileName}, nil)

	err := t.in.Flush(t.ctx)

	require.NoError(t.T(), err)
	t.bucket.AssertExpectations(t.T())
}

func (t *FileMockBucketTest) TestFlushSyncedFileForceFetchObjectFromGCS() {
	// Expect a CreateObject call because createLockedInode creates a synced file
	// inode (backed by GCS object).
	t.bucket.On("CreateObject", t.ctx, mock.AnythingOfType("*gcs.CreateObjectRequest")).
		Return(&gcs.Object{Name: fileName}, nil)
	t.createLockedInode(fileName, emptyGCSFile)
	assert.False(t.T(), t.in.IsLocal())
	// Expect both StatObject and CreateObject call on bucket.
	t.bucket.On("StatObject", t.ctx, mock.AnythingOfType("*gcs.StatObjectRequest")).
		Return(&gcs.MinObject{Name: fileName}, &gcs.ExtendedObjectAttributes{}, nil)
	t.bucket.On("CreateObject", t.ctx, mock.AnythingOfType("*gcs.CreateObjectRequest")).
		Return(&gcs.Object{Name: fileName}, nil)

	err := t.in.Flush(t.ctx)

	require.NoError(t.T(), err)
	t.bucket.AssertExpectations(t.T())
}

func (t *FileMockBucketTest) TestSync_RemoteAppendClobber() {
	// Expect CreateObject from createLockedInode
	t.bucket.On("CreateObject", t.ctx, mock.AnythingOfType("*gcs.CreateObjectRequest")).
		Return(&gcs.Object{Name: fileName, Generation: 1, MetaGeneration: 1}, nil)
	t.createLockedInode(fileName, emptyGCSFile)
	assert.False(t.T(), t.in.IsLocal())
	// Dirty the file.
	_, err := t.in.Write(t.ctx, []byte("data"), 0, util.NewOpenMode(util.WriteOnly, 0))
	require.NoError(t.T(), err)
	// Mock StatObject to return same generation but larger size (remote append).
	// Current size is 0 (emptyGCSFile).
	// We simulate remote append by returning size 100.
	t.bucket.On("StatObject", t.ctx, mock.AnythingOfType("*gcs.StatObjectRequest")).
		Return(&gcs.MinObject{
			Name:           fileName,
			Generation:     t.in.SourceGeneration().Object,
			MetaGeneration: t.in.SourceGeneration().Metadata,
			Size:           100, // Larger size
		}, &gcs.ExtendedObjectAttributes{}, nil)

	_, err = t.in.Sync(t.ctx)

	require.Error(t.T(), err)
	var clobberedErr *gcsfuse_errors.FileClobberedError
	require.True(t.T(), errors.As(err, &clobberedErr))
	assert.Contains(t.T(), clobberedErr.Err.Error(), "file was clobbered due to increase in size at same generation(remote appends)")
}

func (t *FileMockBucketTest) TestSync_GenerationMismatchClobber() {
	// Expect CreateObject from createLockedInode
	t.bucket.On("CreateObject", t.ctx, mock.AnythingOfType("*gcs.CreateObjectRequest")).
		Return(&gcs.Object{Name: fileName, Generation: 1, MetaGeneration: 1}, nil)
	t.createLockedInode(fileName, emptyGCSFile)
	assert.False(t.T(), t.in.IsLocal())
	// Dirty the file.
	_, err := t.in.Write(t.ctx, []byte("data"), 0, util.NewOpenMode(util.WriteOnly, 0))
	require.NoError(t.T(), err)
	// Mock StatObject to return different generation.
	t.bucket.On("StatObject", t.ctx, mock.AnythingOfType("*gcs.StatObjectRequest")).
		Return(&gcs.MinObject{
			Name:           fileName,
			Generation:     t.in.SourceGeneration().Object + 1, // Different generation
			MetaGeneration: t.in.SourceGeneration().Metadata,
			Size:           0,
		}, &gcs.ExtendedObjectAttributes{}, nil)

	_, err = t.in.Sync(t.ctx)

	require.Error(t.T(), err)
	var clobberedErr *gcsfuse_errors.FileClobberedError
	require.True(t.T(), errors.As(err, &clobberedErr))
	assert.Contains(t.T(), clobberedErr.Err.Error(), "file was clobbered due to generation/metageneration mismatch")
}

func (t *FileMockBucketTest) TestAttributes_SizeIncreasedSameGeneration() {
	initialSize := uint64(len("taco"))
	initialGeneration := int64(123)
	initialMetaGeneration := int64(456)
	// Setup the minObject.
	backingObj := &gcs.MinObject{
		Name:           fileName,
		Size:           initialSize,
		Generation:     initialGeneration,
		MetaGeneration: initialMetaGeneration,
	}
	f := t.createGCSBackedFileInode(backingObj)
	defer f.Unlock()
	statReq := &gcs.StatObjectRequest{
		Name:                           fileName,
		ForceFetchFromGcs:              false,
		ReturnExtendedObjectAttributes: false,
	}
	// 1. First call to Attributes: Mock StatObject to return the original object.
	t.bucket.On("StatObject", t.ctx, statReq).
		Return(backingObj, &gcs.ExtendedObjectAttributes{}, nil).Once()
	attrs1, err1 := f.Attributes(t.ctx, true)
	require.NoError(t.T(), err1)
	// Check that attributes match the initial object.
	assert.Equal(t.T(), initialSize, attrs1.Size)
	// 2. Second call to Attributes: Mock StatObject to return an updated object.
	newSize := initialSize + 10
	updatedMinObject := &gcs.MinObject{
		Name:           fileName,
		Generation:     initialGeneration,
		MetaGeneration: initialMetaGeneration,
		Size:           newSize,
	}
	t.bucket.On("StatObject", t.ctx, statReq).
		Return(updatedMinObject, &gcs.ExtendedObjectAttributes{}, nil).Once()

	attrs2, err2 := f.Attributes(t.ctx, true)

	require.NoError(t.T(), err2)
	// Check that attributes are updated.
	assert.Equal(t.T(), newSize, attrs2.Size)
	// Check that internal state is updated.
	assert.Equal(t.T(), newSize, f.Source().Size)
	assert.Equal(t.T(), newSize, f.attrs.Size)
	// Assert that all mock expectations were met.
	t.bucket.AssertExpectations(t.T())
}

func (t *FileMockBucketTest) TestAttributes_NoChangeInAttributes() {
	initialSize := uint64(4)
	initialGeneration := int64(123)
	initialMetaGeneration := int64(456)
	initialTime := t.clock.Now()
	// Setup the minObject
	backingObj := &gcs.MinObject{
		Name:           fileName,
		Size:           initialSize,
		Generation:     initialGeneration,
		MetaGeneration: initialMetaGeneration,
		Updated:        initialTime,
	}
	f := t.createGCSBackedFileInode(backingObj)
	defer f.Unlock()
	t.clock.AdvanceTime(time.Minute)
	// This object has a newer timestamp but the same size and generation.
	updatedMinObject := &gcs.MinObject{
		Name:           fileName,
		Generation:     initialGeneration,
		MetaGeneration: initialMetaGeneration,
		Size:           initialSize, // Size is not greater
		Updated:        t.clock.Now(),
	}
	statReq := &gcs.StatObjectRequest{Name: fileName, ForceFetchFromGcs: false, ReturnExtendedObjectAttributes: false}
	t.bucket.On("StatObject", t.ctx, statReq).Return(updatedMinObject, &gcs.ExtendedObjectAttributes{}, nil).Once()

	attrs, err := f.Attributes(t.ctx, true)

	require.NoError(t.T(), err)
	t.bucket.AssertExpectations(t.T())
	// Attributes should NOT be updated because the size hasn't increased.
	assert.Equal(t.T(), initialSize, attrs.Size)
	assert.Equal(t.T(), initialTime, attrs.Mtime)
	assert.Equal(t.T(), initialSize, f.Source().Size)
	assert.Equal(t.T(), initialTime, f.Source().Updated)
}
