// Write stretchr testify based test for prefetcher.go file
package prefetch

import (
	"context"
	"fmt"
	"io"
	"testing"

	"cloud.google.com/go/storage/control/apiv2/controlpb"
	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// Create strechr testify suite for Setupsuite and Destroy suite
type prefetchTestSuite struct {
	suite.Suite
	assert         *assert.Assertions
	threadPool     *ThreadPool
	blockPool      *BlockPool
	prefetchReader *prefetchReader

	fakeStorage storage.FakeStorage
	object      gcs.MinObject
	bucket      gcs.Bucket
}

func getMinObject(objectName string, bucket gcs.Bucket) gcs.MinObject {
	ctx := context.Background()
	minObject, _, err := bucket.StatObject(ctx, &gcs.StatObjectRequest{Name: objectName,
		ForceFetchFromGcs: true})
	if err != nil {
		panic(fmt.Errorf("error occured while statting the object: %w", err))
	}
	if minObject != nil {
		return *minObject
	}
	return gcs.MinObject{}
}

func (ps *prefetchTestSuite) SetupSuite() {
	ps.assert = assert.New(ps.T())

	// Thread pool.
	ps.threadPool = newThreadPool(4, ps.prefetchReader.download)
	ps.threadPool.Start()

	// Block pool.
	ps.blockPool = NewBlockPool(10 * _1MB, 1024*_1MB)

	// Storage, bucket and object.
	mockClient := new(storage.MockStorageControlClient)
	ps.fakeStorage = storage.NewFakeStorageWithMockClient(mockClient, cfg.HTTP2)
	storageHandle := ps.fakeStorage.CreateStorageHandle()
	mockClient.On("GetStorageLayout", mock.Anything, mock.Anything, mock.Anything).
		Return(&controlpb.StorageLayout{}, nil)
	var err error
	ps.bucket, err = storageHandle.BucketHandle(context.Background(), storage.TestBucketName, "")
	ps.assert.NoError(err)
	objects := map[string][]byte{storage.TestObjectName: make([]byte, 50*_1MB)}
	err = storageutil.CreateObjects(context.Background(), ps.bucket, objects)
	ps.object = getMinObject(storage.TestObjectName, ps.bucket)
	ps.assert.NoError(err)
}

func (ps *prefetchTestSuite) TearDownSuite() {
	ps.threadPool.Stop()
	ps.blockPool.Terminate()
	ps.fakeStorage.ShutDown()
}

func (ps *prefetchTestSuite) TestNewPrefetchReader() {
	// Create a prefetch reader
	prefetchReader := NewPrefetchReader(&ps.object, ps.bucket, getDefaultPrefetchConfig(), ps.blockPool, ps.threadPool)

	// Assert that the prefetch reader is not nil
	ps.assert.NotNil(prefetchReader)

	// Assert that the object, bucket, and prefetch config are set correctly
	ps.assert.Equal(prefetchReader.object, &ps.object)
	ps.assert.Equal(prefetchReader.bucket, ps.bucket)
	ps.assert.NotNil(prefetchReader.prefetchConfig)

	// Assert that the other fields are initialized correctly
	ps.assert.Equal(prefetchReader.lastReadOffset, int64(-1))
	ps.assert.Equal(prefetchReader.nextBlockToPrefetch, int64(0))
	ps.assert.Equal(prefetchReader.randomSeekCount, int64(0))
	ps.assert.NotNil(prefetchReader.cookedBlocks)
	ps.assert.NotNil(prefetchReader.cookingBlocks)
	ps.assert.NotNil(prefetchReader.blockIndexMap)
	ps.assert.Nil(prefetchReader.readHandle)
	ps.assert.Equal(prefetchReader.blockPool, ps.blockPool)
	ps.assert.Nil(prefetchReader.threadPool)
	ps.assert.Nil(prefetchReader.metricHandle)

	// Destroy the prefetch reader
	prefetchReader.Destroy()
}


func (ps *prefetchTestSuite) TestSequentialRead() {
	prefetchReader := NewPrefetchReader(&ps.object, ps.bucket, getDefaultPrefetchConfig(), ps.blockPool, ps.threadPool)

	buffer := make([]byte, _1MB)
	offset := int64(0)
	ctr := 0
	for {
		objectData, err := prefetchReader.ReadAt(context.Background(), buffer, offset)
		ps.assert.True(err != nil || err == io.EOF)
		ps.assert.Equal(len(buffer), int(objectData.Size))

		logger.Infof("TestSequentialRead: objectData: %v", objectData.Size)
		offset += int64(objectData.Size)
		if objectData.Size == 0 {
			break
		}
		ctr++

		if ctr == 40 {
			break
		}
	}
	ps.assert.Equal(offset, int64(50*_1MB))
}

func TestPrefetchSuite(t *testing.T) {
	suite.Run(t, new(prefetchTestSuite))
}
