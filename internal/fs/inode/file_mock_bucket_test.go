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
	"math"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/contentcache"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	storagemock "github.com/googlecloudplatform/gcsfuse/v3/internal/storage/mock"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
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
		semaphore.NewWeighted(math.MaxInt64))

	// Create empty file for local inode created above.
	err := t.in.CreateEmptyTempFile(t.ctx)
	assert.Nil(t.T(), err)

	t.in.Lock()
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

func (t *FileMockBucketTest) TestAttributes_SizeIncreasedSameGeneration() {
	// Arrange
	initialContent := "taco"
	initialSize := uint64(len(initialContent))
	initialGeneration := int64(123)
	initialMetaGeneration := int64(456)
	initialTime := t.clock.Now()

	backingObj := &gcs.MinObject{
		Name:           fileName,
		Size:           initialSize,
		Generation:     initialGeneration,
		MetaGeneration: initialMetaGeneration,
		Updated:        initialTime,
	}

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
		semaphore.NewWeighted(math.MaxInt64))
	f.Lock()
	defer f.Unlock()

	newSize := initialSize + 10
	t.clock.AdvanceTime(time.Minute) // To have a different mtime
	updatedTime := t.clock.Now()
	updatedMinObject := &gcs.MinObject{
		Name:           fileName,
		Generation:     initialGeneration,
		MetaGeneration: initialMetaGeneration,
		Size:           newSize,
		Updated:        updatedTime,
	}

	// Mock StatObject to return the updated object.
	statReq := &gcs.StatObjectRequest{
		Name:                           fileName,
		ForceFetchFromGcs:              false,
		ReturnExtendedObjectAttributes: false,
	}
	t.bucket.On("StatObject", t.ctx, statReq).
		Return(updatedMinObject, &gcs.ExtendedObjectAttributes{}, nil).Once()

	// Act
	attrs, err := f.Attributes(t.ctx, true)

	// Assert
	require.NoError(t.T(), err)
	t.bucket.AssertExpectations(t.T())

	// Check that attributes are updated.
	assert.Equal(t.T(), newSize, attrs.Size)
	assert.Equal(t.T(), updatedTime, attrs.Mtime)

	// Check that internal state is updated.
	assert.Equal(t.T(), newSize, f.Source().Size)
	assert.Equal(t.T(), updatedTime, f.Source().Updated)
	assert.Equal(t.T(), newSize, f.attrs.Size)
	assert.Equal(t.T(), updatedTime, f.attrs.Mtime)
}

func (t *FileMockBucketTest) TestAttributes_NoChangeInAttributes() {
	// Arrange
	initialSize := uint64(4)
	initialGeneration := int64(123)
	initialMetaGeneration := int64(456)
	initialTime := t.clock.Now()

	f := NewFileInode(
		fileInodeID,
		NewFileName(NewRootName(""), fileName),
		&gcs.MinObject{Name: fileName, Size: initialSize, Generation: initialGeneration, MetaGeneration: initialMetaGeneration, Updated: initialTime},
		fuseops.InodeAttributes{Uid: uid, Gid: gid, Mode: fileMode},
		&gcsx.SyncerBucket{Bucket: t.bucket},
		false, contentcache.New("", &t.clock), &t.clock, false, &cfg.Config{}, semaphore.NewWeighted(math.MaxInt64))
	f.Lock()
	defer f.Unlock()

	t.clock.AdvanceTime(time.Minute)
	updatedMinObject := &gcs.MinObject{
		Name:           fileName,
		Generation:     initialGeneration,
		MetaGeneration: initialMetaGeneration,
		Size:           initialSize, // Size is not greater
		Updated:        t.clock.Now(),
	}
	statReq := &gcs.StatObjectRequest{Name: fileName, ForceFetchFromGcs: false, ReturnExtendedObjectAttributes: false}
	t.bucket.On("StatObject", t.ctx, statReq).Return(updatedMinObject, &gcs.ExtendedObjectAttributes{}, nil).Once()

	// Act
	attrs, err := f.Attributes(t.ctx, true)

	// Assert
	require.NoError(t.T(), err)
	t.bucket.AssertExpectations(t.T())
	assert.Equal(t.T(), initialSize, attrs.Size)
	assert.Equal(t.T(), initialTime, attrs.Mtime)
	assert.Equal(t.T(), initialSize, f.Source().Size)
	assert.Equal(t.T(), initialTime, f.Source().Updated)
}
