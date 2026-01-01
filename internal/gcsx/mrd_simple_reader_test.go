package gcsx

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type MrdSimpleReaderTest struct {
	suite.Suite
	object      *gcs.MinObject
	bucket      *storage.TestifyMockBucket
	cache       *lru.Cache
	inodeID     fuseops.InodeID
	mrdConfig   cfg.MrdConfig
	mrdInstance *MrdInstance
	reader      *MrdSimpleReader
}

func TestMrdSimpleReaderTestSuite(t *testing.T) {
	suite.Run(t, new(MrdSimpleReaderTest))
}

func (t *MrdSimpleReaderTest) SetupTest() {
	t.object = &gcs.MinObject{
		Name:       "foo",
		Size:       1024 * MiB,
		Generation: 1234,
	}
	t.bucket = new(storage.TestifyMockBucket)
	t.cache = lru.NewCache(2)
	t.inodeID = 100
	t.mrdConfig = cfg.MrdConfig{PoolSize: 1}

	mi := NewMrdInstance(t.object, t.bucket, t.cache, t.inodeID, t.mrdConfig)
	t.mrdInstance = &mi
	t.reader = NewMrdSimpleReader(t.mrdInstance)
}

func (t *MrdSimpleReaderTest) TestNewMrdSimpleReader() {
	assert.NotNil(t.T(), t.reader)
	assert.Equal(t.T(), t.mrdInstance, t.reader.mrdInstance)
}

func (t *MrdSimpleReaderTest) TestReadAt_EmptyBuffer() {
	req := &ReadRequest{
		Buffer: []byte{},
		Offset: 0,
	}

	resp, err := t.reader.ReadAt(context.Background(), req)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 0, resp.Size)
}

func (t *MrdSimpleReaderTest) TestReadAt_Success() {
	data := []byte("hello world")
	fakeMRD := fake.NewFakeMultiRangeDownloader(t.object, data)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	req := &ReadRequest{
		Buffer: make([]byte, 5),
		Offset: 0,
	}

	resp, err := t.reader.ReadAt(context.Background(), req)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 5, resp.Size)
	assert.Equal(t.T(), "hello", string(req.Buffer))
}

func (t *MrdSimpleReaderTest) TestReadAt_ContextCancelled() {
	fakeMRD := fake.NewFakeMultiRangeDownloaderWithSleep(t.object, []byte("hello"), 100*time.Millisecond)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	ctx, cancel := context.WithCancel(context.Background())
	req := &ReadRequest{
		Buffer: make([]byte, 5),
		Offset: 0,
	}
	cancel()

	resp, err := t.reader.ReadAt(ctx, req)

	assert.Error(t.T(), err)
	assert.Equal(t.T(), context.Canceled, err)
	assert.Equal(t.T(), 0, resp.Size)
}

func (t *MrdSimpleReaderTest) TestReadAt_MrdError() {
	fakeMRD := fake.NewFakeMultiRangeDownloaderWithSleepAndDefaultError(t.object, []byte("hello"), 0, errors.New("read error"))
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	req := &ReadRequest{
		Buffer: make([]byte, 5),
		Offset: 0,
	}

	resp, err := t.reader.ReadAt(context.Background(), req)

	assert.Error(t.T(), err)
	assert.Contains(t.T(), err.Error(), "read error")
	assert.Equal(t.T(), 0, resp.Size)
}

func (t *MrdSimpleReaderTest) TestReadAt_MrdEOF() {
	fakeMRD := fake.NewFakeMultiRangeDownloaderWithSleepAndDefaultError(t.object, []byte("hello"), 0, io.EOF)
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD, nil).Once()
	req := &ReadRequest{
		Buffer: make([]byte, 5),
		Offset: 0,
	}

	resp, err := t.reader.ReadAt(context.Background(), req)

	assert.Error(t.T(), err)
	assert.Equal(t.T(), io.EOF, err)
	assert.Equal(t.T(), 0, resp.Size)
}

func (t *MrdSimpleReaderTest) TestReadAt_NilMrdInstance() {
	t.reader.mrdInstance = nil
	req := &ReadRequest{
		Buffer: make([]byte, 5),
		Offset: 0,
	}

	resp, err := t.reader.ReadAt(context.Background(), req)

	assert.Error(t.T(), err)
	assert.Contains(t.T(), err.Error(), "mrdInstance is nil")
	assert.Equal(t.T(), 0, resp.Size)
}

func (t *MrdSimpleReaderTest) TestReadAt_Recreation() {
	// 1. Initial creation with a broken MRD
	fakeMRD1 := fake.NewFakeMultiRangeDownloaderWithStatusError(t.object, []byte("data"), errors.New("broken"))
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD1, nil).Once()
	// Initialize pool manually to simulate existing bad state
	t.mrdInstance.EnsureMrdInstance()
	// 2. ReadAt called. getValidEntry gets entry with fakeMRD1.
	// It sees error, calls RecreateMRDEntry.
	fakeMRD2 := fake.NewFakeMultiRangeDownloader(t.object, []byte("data"))
	t.bucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fakeMRD2, nil).Once()
	req := &ReadRequest{
		Buffer: make([]byte, 4),
		Offset: 0,
	}

	resp, err := t.reader.ReadAt(context.Background(), req)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), 4, resp.Size)
	assert.Equal(t.T(), "data", string(req.Buffer))
}

func (t *MrdSimpleReaderTest) TestDestroy() {
	t.reader.Destroy()

	assert.Nil(t.T(), t.reader.mrdInstance)
	// Verify that calling Destroy again doesn't panic
	t.reader.Destroy()
}

func (t *MrdSimpleReaderTest) TestCheckInvariants() {
	// Should not panic
	t.reader.CheckInvariants()
}
