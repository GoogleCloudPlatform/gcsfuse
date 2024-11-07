package bufferedwrites

import (
	"context"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/block"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/semaphore"
)

const (
	blockSize = 1024
)

type UploadHandlerTest struct {
	uh         *UploadHandler
	blockPool  *block.BlockPool
	mockBucket *storage.TestifyMockBucket
	suite.Suite
}

func TestUploadHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(UploadHandlerTest))
}

func (t *UploadHandlerTest) SetupTest() {
	t.mockBucket = new(storage.TestifyMockBucket)
	var err error
	t.blockPool, err = block.NewBlockPool(blockSize, 2, semaphore.NewWeighted(2))
	assert.NoError(t.T(), err)
	t.uh = newUploadHandler("testObject", t.mockBucket, t.blockPool.FreeBlocksChannel(), blockSize)
}

func (t *UploadHandlerTest) TestStartUpload() {
	ctx := context.Background()
	assert.Equal(t.T(), NotStarted, t.uh.status)

	// Upload first block.
	writer := NewMockWriter("mockObject")
	t.mockBucket.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(writer, nil)
	block1, err := t.blockPool.Get()
	assert.NoError(t.T(), err)
	err = t.uh.Upload(ctx, block1)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), Uploading, t.uh.status)
}

func (t *UploadHandlerTest) TestUpload2Blocks() {
	ctx := context.Background()
	assert.Equal(t.T(), NotStarted, t.uh.status)

	// Upload first block.
	writer := NewMockWriter("mockObject")
	t.mockBucket.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(writer, nil)
	block1, err := t.blockPool.Get()
	assert.NoError(t.T(), err)
	err = t.uh.Upload(ctx, block1)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), Uploading, t.uh.status)
	assert.Equal(t.T(), 0, t.uh.blocksLength()) // First block does not wait for upload.

	// Upload second block.
	block2, err := t.blockPool.Get()
	assert.NoError(t.T(), err)
	err = t.uh.Upload(ctx, block2)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), Uploading, t.uh.status)
	assert.Equal(t.T(), 1, t.uh.blocksLength()) // Second block is added to queue.

	// Simulate first block upload completion.
	t.uh.statusNotifier(blockSize)
	select {
	case <-t.uh.freeBlocksCh:
	default:
		t.T().Error("Block not put back on freeBlocksCh after upload completion.")
	}
	assert.Equal(t.T(), 0, t.uh.blocksLength()) // Second block is picked next for upload.
	assert.Equal(t.T(), Uploading, t.uh.status)

	// Simulate second block upload completion.
	t.uh.statusNotifier(2 * blockSize)
	select {
	case <-t.uh.freeBlocksCh:
	default:
		t.T().Error("Block not put back on freeBlocksCh after upload completion.")
	}
	assert.Equal(t.T(), ReadyForNextBlock, t.uh.status)

	// Finalize the upload.
	err = t.uh.Finalize()
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), Finalized, t.uh.status)
}

func (t *UploadHandlerTest) TestUploadBlockWhenStatusReadyForNextBlock() {
	ctx := context.Background()
	assert.Equal(t.T(), NotStarted, t.uh.status)

	// Upload first block.
	writer := NewMockWriter("mockObject")
	t.mockBucket.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(writer, nil)
	block1, err := t.blockPool.Get()
	assert.NoError(t.T(), err)
	err = t.uh.Upload(ctx, block1)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), Uploading, t.uh.status)
	assert.Equal(t.T(), 0, t.uh.blocksLength()) // First block immediately picked from queue for upload.

	// Simulate first block upload completion.
	t.uh.statusNotifier(blockSize)
	select {
	case <-t.uh.freeBlocksCh:
	default:
		t.T().Error("Block not put back on freeBlocksCh after upload completion.")
	}
	assert.Equal(t.T(), ReadyForNextBlock, t.uh.status)

	// Upload second block.
	block2, err := t.blockPool.Get()
	assert.NoError(t.T(), err)
	err = t.uh.Upload(ctx, block2)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), Uploading, t.uh.status)
	assert.Equal(t.T(), 0, t.uh.blocksLength()) // Block immediately picked from queue for upload.

	// Simulate second block upload completion.
	t.uh.statusNotifier(2 * blockSize)
	select {
	case <-t.uh.freeBlocksCh:
	default:
		t.T().Error("Block not put back on freeBlocksCh after upload completion.")
	}
	assert.Equal(t.T(), ReadyForNextBlock, t.uh.status)

	// Finalize the upload.
	err = t.uh.Finalize()
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), Finalized, t.uh.status)
}

func (t *UploadHandlerTest) TestUploadPartialBlock() {
	// Note: Partial block will be sent for upload only with finalize call.
	ctx := context.Background()
	assert.Equal(t.T(), NotStarted, t.uh.status)

	// Upload partial block. There will be no callback.
	writer := NewMockWriter("mockObject")
	t.mockBucket.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(writer, nil)
	block1, err := t.blockPool.Get()
	assert.NoError(t.T(), err)
	err = t.uh.Upload(ctx, block1)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), Uploading, t.uh.status)
	assert.Equal(t.T(), 0, t.uh.blocksLength())
	time.Sleep(1 * time.Second)

	// Finalize the upload.
	err = t.uh.Finalize()
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), Finalized, t.uh.status)
}

//func (t *UploadHandlerTest) TestUploadError() {
//	ctx := context.Background()
//
//	// Simulate error in uploadBlock.
//	t.uh.status = Uploading
//	t.uh.bucket.SetError(errors.New("fake upload error"), 1)
//
//	block1 := newFakeBlock("block1")
//	err := uh.Upload(ctx, block1)
//	if err == nil {
//		t.Error("Expected an error, got nil.")
//	}
//	if uh.status != Failed {
//		t.Errorf("Unexpected status: %v, expected: %v", uh.status, Failed)
//	}
//}

//func TestFinalizeError(t *testing.T) {
//	ctx := context.Background()
//	bucket := fake.NewFakeBucket(timeutil.RealClock(), "FakeBucketName", gcs.NonHierarchical)
//	uh, _ := testUploadHandler(t, bucket)
//
//	// Simulate error in Finalize.
//	bucket.SetError(errors.New("fake finalize error"), 1)
//
//	block1 := newFakeBlock("block1")
//	err := uh.Upload(ctx, block1)
//	if err != nil {
//		t.Errorf("Unexpected error: %v", err)
//	}
//
//	err = uh.Finalize()
//	if err == nil {
//		t.Error("Expected an error, got nil.")
//	}
//}
//
//func TestUploadAfterFinalize(t *testing.T) {
//	ctx := context.Background()
//	bucket := fake.NewFakeBucket(timeutil.RealClock(), "FakeBucketName", gcs.NonHierarchical)
//	uh, _ := testUploadHandler(t, bucket)
//
//	block1 := newFakeBlock("block1")
//	err := uh.Upload(ctx, block1)
//	if err != nil {
//		t.Errorf("Unexpected error: %v", err)
//	}
//
//	err = uh.Finalize()
//	if err != nil {
//		t.Errorf("Unexpected error: %v", err)
//	}
//
//	err = uh.Upload(ctx, block1)
//	if err == nil {
//		t.Error("Expected an error, got nil.")
//	}
//	if !errors.Is(err, ErrUploadFinalized) {
//		t.Errorf("Unexpected error: %v, expected: %v", err, ErrUploadFinalized)
//	}
//}
//
//func TestConcurrentUploads(t *testing.T) {
//	ctx := context.Background()
//	bucket := fake.NewFakeBucket(timeutil.RealClock(), "FakeBucketName", gcs.NonHierarchical)
//	uh, _ := testUploadHandler(t, bucket)
//
//	var wg sync.WaitGroup
//	for i := 0; i < 10; i++ {
//		wg.Add(1)
//		go func(i int) {
//			defer wg.Done()
//			block := newFakeBlock(fmt.Sprintf("block%d", i))
//			err := uh.Upload(ctx, block)
//			if err != nil {
//				t.Errorf("Unexpected error: %v", err)
//			}
//		}(i)
//	}
//	wg.Wait()
//
//	// Simulate all block upload completions.
//	for i := 0; i < 10; i++ {
//		uh.statusNotifier(int64(i+1) * blockSize)
//	}
//
//	err := uh.Finalize()
//	if err != nil {
//		t.Errorf("Unexpected error: %v", err)
//	}
//}
