// Write stretchr testify based test for prefetcher.go file
package prefetch

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"cloud.google.com/go/storage/control/apiv2/controlpb"
	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
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
	PrefetchReader *PrefetchReader

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
	stime := time.Now()
	ps.assert = assert.New(ps.T())

	// Thread pool.
	ps.threadPool = NewThreadPool(4, Download)
	ps.threadPool.Start()

	// Block pool.
	ps.blockPool = NewBlockPool(10*_1MB, 1024*_1MB)

	// Storage, bucket and object.
	mockClient := new(storage.MockStorageControlClient)
	ps.fakeStorage = storage.NewFakeStorageWithMockClient(mockClient, cfg.HTTP2)
	storageHandle := ps.fakeStorage.CreateStorageHandle()
	mockClient.On("GetStorageLayout", mock.Anything, mock.Anything, mock.Anything).
		Return(&controlpb.StorageLayout{}, nil)
	var err error
	ps.bucket, err = storageHandle.BucketHandle(context.Background(), storage.TestBucketName, "")
	ps.assert.NoError(err)
	_, err = storageutil.CreateObject(context.Background(), ps.bucket, storage.TestObjectName, make([]byte, 200*_1MB))
	ps.object = getMinObject(storage.TestObjectName, ps.bucket)
	ps.assert.NoError(err)

	fmt.Printf("Total setup time: %v\n", time.Since(stime))

}

func (ps *prefetchTestSuite) TearDownSuite() {
	stime := time.Now()
	ps.threadPool.Stop()
	ps.blockPool.Terminate()
	ps.fakeStorage.ShutDown()
	fmt.Printf("Total teardown time: %v\n", time.Since(stime))
}

func (ps *prefetchTestSuite) TestNewPrefetchReader() {
	// Create a prefetch reader
	PrefetchReader := NewPrefetchReader(&ps.object, ps.bucket, getDefaultPrefetchConfig(), ps.blockPool, ps.threadPool)

	// Assert that the prefetch reader is not nil
	ps.assert.NotNil(PrefetchReader)

	// Assert that the object, bucket, and prefetch config are set correctly
	ps.assert.Equal(PrefetchReader.object, &ps.object)
	ps.assert.Equal(PrefetchReader.bucket, ps.bucket)
	ps.assert.NotNil(PrefetchReader.PrefetchConfig)

	// Assert that the other fields are initialized correctly
	ps.assert.Equal(PrefetchReader.lastReadOffset, int64(-1))
	ps.assert.Equal(PrefetchReader.nextBlockToPrefetch, int64(0))
	ps.assert.Equal(PrefetchReader.randomSeekCount, int64(0))
	ps.assert.Nil(PrefetchReader.readHandle)
	ps.assert.Equal(PrefetchReader.blockPool, ps.blockPool)
	ps.assert.Equal(PrefetchReader.threadPool, ps.threadPool)
	ps.assert.Nil(PrefetchReader.metricHandle)

	// Destroy the prefetch reader
	PrefetchReader.Destroy()
}

func (ps *prefetchTestSuite) TestSequentialRead() {
	PrefetchReader := NewPrefetchReader(&ps.object, ps.bucket, getDefaultPrefetchConfig(), ps.blockPool, ps.threadPool)

	buffer := make([]byte, _1MB)
	offset := int64(0)
	for offset = int64(0); offset < int64(100*_1MB); offset += int64(len(buffer)) {
		n, err := PrefetchReader.ReadAt(context.Background(), buffer, offset)
		ps.assert.True(err == nil || err == io.EOF)
		ps.assert.Equal(len(buffer), int(n))
	}
	ps.assert.Equal(offset, int64(100*_1MB))
}

func TestPrefetchSuite(t *testing.T) {
	suite.Run(t, new(prefetchTestSuite))
}
