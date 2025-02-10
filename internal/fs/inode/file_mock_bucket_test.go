package inode

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/contentcache"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	storagemock "github.com/googlecloudplatform/gcsfuse/v2/internal/storage/mock"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
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
	t.createInode(fileName, localFile)
}

func (t *FileMockBucketTest) TearDownTest() {
	t.in.Unlock()
}

func (t *FileMockBucketTest) SetupSubTest() {
	t.SetupTest()
}

func (t *FileMockBucketTest) createInode(fileName string, fileType string) {
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

	// Create write handler for the local inode created above.
	err := t.in.CreateBufferedOrTempWriter(t.ctx)
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
	t.bucket.On("CreateObject", t.ctx, mock.AnythingOfType("*gcs.CreateObjectRequest")).
		Return(&gcs.Object{Name: fileName}, nil)
	t.createInode(fileName, emptyGCSFile)
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
