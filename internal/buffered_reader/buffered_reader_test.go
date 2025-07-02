// Write stretchr testify based test for prefetcher.go file
package buffered_reader

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

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
	workerPool     *WorkerPool
	blockPool      *BlockPool
	BufferedReader *BufferedReader

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
	// logger.SetLogLevel("TRACE")

	stime := time.Now()
	ps.assert = assert.New(ps.T())

	// Thread pool.
	ps.workerPool = NewWorkerPool(20, Download)
	ps.workerPool.Start()

	// Block pool.
	ps.blockPool = NewBlockPool(uint64(getDefaultBufferedReadConfig().PrefetchChunkSize), 3024*_1MB)

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

	logger.Infof("Total setup time: %v\n", time.Since(stime))

}

func (ps *prefetchTestSuite) TearDownSuite() {
	stime := time.Now()
	ps.workerPool.Stop()
	ps.blockPool.Terminate()
	ps.fakeStorage.ShutDown()
	logger.Infof("Total teardown time: %v\n", time.Since(stime))
}

func (ps *prefetchTestSuite) TestNewBufferedReader() {
	// Create a prefetch reader
	BufferedReader := NewBufferedReader(&ps.object, ps.bucket, getDefaultBufferedReadConfig(), ps.blockPool, ps.workerPool)

	// Assert that the prefetch reader is not nil
	ps.assert.NotNil(BufferedReader)

	// Assert that the object, bucket, and prefetch config are set correctly
	ps.assert.Equal(BufferedReader.object, &ps.object)
	ps.assert.Equal(BufferedReader.bucket, ps.bucket)
	ps.assert.NotNil(BufferedReader.config)

	// Assert that the other fields are initialized correctly
	ps.assert.Equal(BufferedReader.lastReadOffset, int64(-1))
	ps.assert.Equal(BufferedReader.nextBlockToPrefetch, int64(0))
	ps.assert.Equal(BufferedReader.randomSeekCount, int64(0))
	ps.assert.Nil(BufferedReader.readHandle)
	ps.assert.Equal(BufferedReader.blockPool, ps.blockPool)
	ps.assert.Equal(BufferedReader.workerPool, ps.workerPool)
	ps.assert.Nil(BufferedReader.metricHandle)

	// Destroy the prefetch reader
	BufferedReader.Destroy()
}

func (ps *prefetchTestSuite) TestSequentialRead() {
	BufferedReader := NewBufferedReader(&ps.object, ps.bucket, getDefaultBufferedReadConfig(), ps.blockPool, ps.workerPool)

	buffer := make([]byte, _1MB)
	offset := int64(0)
	timerS := time.Now()
	for offset = int64(0); offset < int64(200*_1MB); offset += int64(len(buffer)) {
		n, err := BufferedReader.ReadAt(context.Background(), buffer, offset)
		ps.assert.True(err == nil || err == io.EOF)
		ps.assert.Equal(len(buffer), int(n))
	}
	logger.Infof("Total read time: %v\n", time.Since(timerS))
	ps.assert.Equal(offset, int64(200*_1MB))
}

func (ps *prefetchTestSuite) TestSequentialReadWithHighInitialPrefetch() {
	defaultConfig := getDefaultBufferedReadConfig()
	defaultConfig.InitialPrefetchBlockCnt = 100
	BufferedReader := NewBufferedReader(&ps.object, ps.bucket, defaultConfig, ps.blockPool, ps.workerPool)

	timerS := time.Now()
	buffer := make([]byte, _1MB)
	offset := int64(0)
	for offset = int64(0); offset < int64(200*_1MB); offset += int64(len(buffer)) {
		n, err := BufferedReader.ReadAt(context.Background(), buffer, offset)
		ps.assert.True(err == nil || err == io.EOF)
		ps.assert.Equal(len(buffer), int(n))
	}
	logger.Infof("Total read time: %v\n", time.Since(timerS))
	ps.assert.Equal(offset, int64(200*_1MB))
}

func TestPrefetchSuite(t *testing.T) {
	suite.Run(t, new(prefetchTestSuite))
}
