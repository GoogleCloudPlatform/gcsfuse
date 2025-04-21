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

package bufferedwrites

import (
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/block"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	storagemock "github.com/googlecloudplatform/gcsfuse/v2/internal/storage/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/semaphore"
)

////////////////////////////////////////////////////////////////////////
// Constants
////////////////////////////////////////////////////////////////////////

const (
	blockSize       = 1024
	maxBlocks int64 = 5
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type UploadHandlerTest struct {
	writer     *storagemock.Writer
	uh         *UploadHandler
	blockPool  *block.BlockPool
	mockBucket *storagemock.TestifyMockBucket
	suite.Suite
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (t *UploadHandlerTest) assertUploadFailureError() {
	t.T().Helper()
	for {
		select {
		case <-time.After(200 * time.Millisecond):
			t.T().Error("Expected an error in uploader")
		default:
			if t.uh.UploadError() != nil {
				return
			}
		}
	}
}

func (t *UploadHandlerTest) assertAllBlocksProcessed() {
	t.T().Helper()

	// All blocks for upload should have been processed.
	done := make(chan struct{})
	go func() {
		t.uh.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.T().Error("Timeout waiting for WaitGroup")
	}
}

func (t *UploadHandlerTest) createBlocks(count int) []block.Block {
	var blocks []block.Block
	for i := 0; i < count; i++ {
		b, err := t.blockPool.Get()
		require.NoError(t.T(), err)
		blocks = append(blocks, b)
	}

	return blocks
}

func (t *UploadHandlerTest) SetupTest() {
	t.writer = &storagemock.Writer{}
	t.mockBucket = new(storagemock.TestifyMockBucket)
	t.mockBucket.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(t.writer, nil).Once()
	t.mockBucket.On("BucketType").Return(gcs.BucketType{Zonal: true})
	t.mockBucket.On("FlushPendingWrites", mock.Anything, t.writer).Return(0, nil).Once()
	var err error
	t.blockPool, err = block.NewBlockPool(blockSize, maxBlocks, semaphore.NewWeighted(maxBlocks))
	require.NoError(t.T(), err)
	t.uh, err = newUploadHandler(&CreateUploadHandlerRequest{
		Object:                   nil,
		ObjectName:               "testObject",
		Bucket:                   t.mockBucket,
		FreeBlocksCh:             t.blockPool.FreeBlocksChannel(),
		MaxBlocksPerFile:         maxBlocks,
		BlockSize:                blockSize,
		ChunkTransferTimeoutSecs: chunkTransferTimeoutSecs,
	})
	require.NoError(t.T(), err)
	assert.NotNil(t.T(), t.uh.writer)
}

func (t *UploadHandlerTest) SetupSubTest() {
	t.SetupTest()
}

// //////////////////////////////////////////////////////////////////////
// Tests
// //////////////////////////////////////////////////////////////////////

func (t *UploadHandlerTest) TestMultipleBlockUpload() {
	mockObj := &gcs.MinObject{}
	t.mockBucket.On("FinalizeUpload", mock.Anything, t.uh.writer).Return(mockObj, nil)

	// Upload the blocks.
	blocks := t.createBlocks(5)
	for _, b := range blocks {
		t.uh.Upload(b)
	}

	// Finalize.
	obj, err := t.uh.Finalize()
	require.NoError(t.T(), err)
	require.NotNil(t.T(), obj)
	assert.Equal(t.T(), mockObj, obj)
	// The blocks should be available on the free channel for reuse.
	for _, expect := range blocks {
		got := <-t.uh.freeBlocksCh
		assert.Equal(t.T(), expect, got)
	}
	t.assertAllBlocksProcessed()
}

func (t *UploadHandlerTest) TestFinalizeWithWriterAlreadyPresent() {
	mockObj := &gcs.MinObject{}
	t.mockBucket.On("FinalizeUpload", mock.Anything, t.uh.writer).Return(mockObj, nil)

	obj, err := t.uh.Finalize()

	require.NoError(t.T(), err)
	require.NotNil(t.T(), obj)
	assert.Equal(t.T(), mockObj, obj)
}

func (t *UploadHandlerTest) TestFinalizeWithNoWriter() {
	mockObj := &gcs.MinObject{}
	t.mockBucket.On("FinalizeUpload", mock.Anything, t.uh.writer).Return(mockObj, nil)

	obj, err := t.uh.Finalize()

	require.NoError(t.T(), err)
	require.NotNil(t.T(), obj)
	assert.Equal(t.T(), mockObj, obj)
}

func (t *UploadHandlerTest) TestFinalizeWhenFinalizeUploadFails() {
	mockObj := &gcs.MinObject{}
	t.mockBucket.On("FinalizeUpload", mock.Anything, t.writer).Return(mockObj, fmt.Errorf("taco"))

	obj, err := t.uh.Finalize()

	require.Error(t.T(), err)
	assert.Nil(t.T(), obj)
	assert.ErrorContains(t.T(), err, "taco")
	t.assertUploadFailureError()
}

func (t *UploadHandlerTest) TestFlushWithWriterAlreadyPresent() {
	mockOffset := 10
	t.mockBucket.On("FlushPendingWrites", mock.Anything, t.writer).Return(mockOffset, nil)

	offset, err := t.uh.FlushPendingWrites()

	require.NoError(t.T(), err)
	assert.EqualValues(t.T(), mockOffset, offset)
}

func (t *UploadHandlerTest) TestFlushWhenFlushPendingWritesFails() {
	t.mockBucket.On("FlushPendingWrites", mock.Anything, t.writer).Return(0, fmt.Errorf("taco"))

	offset, err := t.uh.FlushPendingWrites()

	require.Error(t.T(), err)
	assert.EqualValues(t.T(), 0, offset)
	assert.ErrorContains(t.T(), err, "taco")
	t.assertUploadFailureError()
}

func (t *UploadHandlerTest) TestUploadSingleBlockThrowsErrorInCopy() {
	// Create a block with test data.
	b, err := t.blockPool.Get()
	require.NoError(t.T(), err)
	t.writer.On("Write", mock.Anything).Return(0, fmt.Errorf("taco")).Once()
	err = b.Write([]byte("test data"))
	require.NoError(t.T(), err)

	// Upload the block.
	t.uh.Upload(b)

	// Expect an error on upload due to error while copying content to GCS writer.
	t.assertUploadFailureError()
	t.assertAllBlocksProcessed()
	assert.Equal(t.T(), 1, len(t.uh.freeBlocksCh))
}

func (t *UploadHandlerTest) TestUploadMultipleBlocksThrowsErrorInCopy() {
	// Create some blocks.
	blocks := t.createBlocks(4)
	for i := 0; i < 4; i++ {
		err := blocks[i].Write([]byte("testdata" + strconv.Itoa(i) + " "))
		require.NoError(t.T(), err)
	}
	// Second write will be an error and rest of the operations will be successful.
	t.writer.
		On("Write", mock.Anything).Return(10, nil).Once().
		On("Write", mock.Anything).Return(0, fmt.Errorf("taco"))

	// Upload the blocks.
	for _, b := range blocks {
		t.uh.Upload(b)
	}

	t.assertUploadFailureError()
	t.assertAllBlocksProcessed()
	assert.Equal(t.T(), 2, len(t.uh.freeBlocksCh))
}

func (t *UploadHandlerTest) TestMultipleBlockAwaitBlocksUpload() {
	// Upload the blocks.
	for _, b := range t.createBlocks(5) {
		t.uh.Upload(b)
	}

	// AwaitBlocksUpload.
	t.uh.AwaitBlocksUpload()

	assert.Equal(t.T(), 5, len(t.uh.freeBlocksCh))
	assert.Equal(t.T(), 0, len(t.uh.uploadCh))
	t.assertAllBlocksProcessed()
}

func (t *UploadHandlerTest) TestUploadHandlerCancelUpload() {
	cancelCalled := false
	t.uh.cancelFunc = func() { cancelCalled = true }

	t.uh.CancelUpload()

	assert.True(t.T(), cancelCalled)
}

func (t *UploadHandlerTest) TestUploadHandlerAfterCancelUploadBlocksGetProcessed() {
	// Upload the blocks.
	for _, b := range t.createBlocks(2) {
		t.uh.Upload(b)
	}
	t.uh.AwaitBlocksUpload()
	t.uh.CancelUpload()

	// Upload the blocks after context cancelled.
	for _, b := range t.createBlocks(2) {
		t.uh.Upload(b)
	}

	assert.Nil(t.T(), t.uh.UploadError())
	t.assertAllBlocksProcessed()
}

func (t *UploadHandlerTest) TestDestroy() {
	testCases := []struct {
		name           string
		uploadChClosed bool
	}{
		{
			name:           "UploadChNotClosed",
			uploadChClosed: false,
		},
		{
			name:           "UploadChClosed",
			uploadChClosed: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			// Add blocks to uploadCh.
			for _, b := range t.createBlocks(5) {
				t.uh.uploadCh <- b
				t.uh.wg.Add(1)
			}
			if tc.uploadChClosed {
				close(t.uh.uploadCh)
			}

			t.uh.Destroy()

			t.assertAllBlocksProcessed()
			assert.Equal(t.T(), 5, len(t.uh.freeBlocksCh))
			assert.Equal(t.T(), 0, len(t.uh.uploadCh))
			// Check if uploadCh is closed.
			select {
			case <-t.uh.uploadCh:
			default:
				assert.Fail(t.T(), "uploadCh not closed")
			}
		})
	}
}

func TestUploadHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(UploadHandlerTest))
}

func TestUploadErrorReturnsError(t *testing.T) {
	mockUploadError := fmt.Errorf("error")
	uploadHandler := &UploadHandler{}
	uploadHandler.uploadError.Store(&mockUploadError)

	actualUploadError := uploadHandler.UploadError()

	assert.Equal(t, mockUploadError, actualUploadError)
}

func TestUploadErrorReturnsNil(t *testing.T) {
	uploadHandler := &UploadHandler{}

	actualUploadError := uploadHandler.UploadError()

	assert.Nil(t, actualUploadError)
}

func TestCreatingUploadHandlerWhenCreateObjectWriterFails(t *testing.T) {
	mockBucket := new(storagemock.TestifyMockBucket)
	blockPool, err := block.NewBlockPool(blockSize, maxBlocks, semaphore.NewWeighted(maxBlocks))
	require.NoError(t, err)
	mockBucket.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("taco"))

	_, err = newUploadHandler(&CreateUploadHandlerRequest{
		Object:                   nil,
		ObjectName:               "testObject",
		Bucket:                   mockBucket,
		FreeBlocksCh:             blockPool.FreeBlocksChannel(),
		MaxBlocksPerFile:         maxBlocks,
		BlockSize:                blockSize,
		ChunkTransferTimeoutSecs: chunkTransferTimeoutSecs,
	})

	require.Error(t, err)
	assert.ErrorContains(t, err, "CreateObjectChunkWriter")
	assert.ErrorContains(t, err, "taco")
}

func TestCreatingUploadHandlerWhenFlushPendingWritesFails(t *testing.T) {
	writer := &storagemock.Writer{}
	mockBucket := new(storagemock.TestifyMockBucket)
	// CreateObjectChunkWriter -- should be called once.
	mockBucket.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(writer, nil)
	// Return BucketType Zonal for flushing the writer.
	mockBucket.On("BucketType").Return(gcs.BucketType{Zonal: true})
	blockPool, err := block.NewBlockPool(blockSize, maxBlocks, semaphore.NewWeighted(maxBlocks))
	require.NoError(t, err)
	mockBucket.On("FlushPendingWrites", mock.Anything, mock.Anything).Return(0, errors.New("taco"))

	_, err = newUploadHandler(&CreateUploadHandlerRequest{
		Object:                   nil,
		ObjectName:               "testObject",
		Bucket:                   mockBucket,
		FreeBlocksCh:             blockPool.FreeBlocksChannel(),
		MaxBlocksPerFile:         maxBlocks,
		BlockSize:                blockSize,
		ChunkTransferTimeoutSecs: chunkTransferTimeoutSecs,
	})

	require.Error(t, err)
	assert.ErrorContains(t, err, "FlushPendingWrites")
	assert.ErrorContains(t, err, "taco")
}

func TestCreateObjectChunkWriterIsCalledWithCorrectRequestParametersForLocalInode(t *testing.T) {
	// CreateObjectChunkWriter -- should be called once with correct request parameters.
	writer := &storagemock.Writer{}
	mockObj := &gcs.Object{}
	mockBucket := new(storage.TestifyMockBucket)
	blockPool, err := block.NewBlockPool(blockSize, maxBlocks, semaphore.NewWeighted(maxBlocks))
	require.NoError(t, err)
	mockBucket.On("CreateObjectChunkWriter",
		mock.Anything,
		mock.MatchedBy(func(req *gcs.CreateObjectRequest) bool {
			return req.Name == "testObject" &&
				*req.GenerationPrecondition == 0 &&
				req.MetaGenerationPrecondition == nil &&
				req.ChunkTransferTimeoutSecs == 0
		}),
		mock.Anything,
		mock.Anything).Return(writer, nil)
	// Return BucketType Zonal for flushing the writer.
	mockBucket.On("BucketType").Return(gcs.BucketType{Zonal: true})
	mockBucket.On("FlushPendingWrites", mock.Anything, writer).Return(int64(0), nil)
	uh, err := newUploadHandler(&CreateUploadHandlerRequest{
		Object:                   nil,
		ObjectName:               "testObject",
		Bucket:                   mockBucket,
		FreeBlocksCh:             blockPool.FreeBlocksChannel(),
		MaxBlocksPerFile:         maxBlocks,
		BlockSize:                blockSize,
		ChunkTransferTimeoutSecs: chunkTransferTimeoutSecs,
	})
	require.NoError(t, err)
	require.NotNil(t, uh.writer)
	mockBucket.On("FinalizeUpload", mock.Anything, writer).Return(mockObj, nil)

	// Create a block.
	b, err := blockPool.Get()
	require.NoError(t, err)
	// Upload the block.
	uh.Upload(b)

}

func TestCreateObjectChunkWriterIsCalledWithCorrectRequestParametersForEmptyGCSObject(t *testing.T) {
	obj := &gcs.Object{
		Name:            "emptyGCSObject",
		ContentType:     "image/png",
		Size:            0,
		ContentEncoding: "gzip",
		Generation:      10,
		MetaGeneration:  20,
		Acl:             nil,
	}
	// CreateObjectChunkWriter -- should be called once with correct request parameters.
	writer := &storagemock.Writer{}
	mockObj := &gcs.Object{}
	mockBucket := new(storage.TestifyMockBucket)
	blockPool, err := block.NewBlockPool(blockSize, maxBlocks, semaphore.NewWeighted(maxBlocks))
	require.NoError(t, err)
	mockBucket.On("CreateObjectChunkWriter",
		mock.Anything,
		mock.MatchedBy(func(req *gcs.CreateObjectRequest) bool {
			return req.Name == obj.Name &&
				*req.GenerationPrecondition == obj.Generation &&
				*req.MetaGenerationPrecondition == obj.MetaGeneration &&
				req.ContentEncoding == obj.ContentEncoding &&
				req.ContentType == obj.ContentType &&
				req.ChunkTransferTimeoutSecs == 0
		}),
		mock.Anything,
		mock.Anything).Return(writer, nil).Once()
	// Return BucketType Zonal for flushing the writer.
	mockBucket.On("BucketType").Return(gcs.BucketType{Zonal: true})
	mockBucket.On("FlushPendingWrites", mock.Anything, writer).Return(int64(0), nil)
	uh, err := newUploadHandler(&CreateUploadHandlerRequest{
		Object:                   obj,
		ObjectName:               obj.Name,
		Bucket:                   mockBucket,
		FreeBlocksCh:             blockPool.FreeBlocksChannel(),
		MaxBlocksPerFile:         maxBlocks,
		BlockSize:                blockSize,
		ChunkTransferTimeoutSecs: chunkTransferTimeoutSecs,
	})
	require.NoError(t, err)
	require.NotNil(t, uh.writer)
	mockBucket.On("FinalizeUpload", mock.Anything, writer).Return(mockObj, nil)

	// Create a block.
	b, err := blockPool.Get()
	require.NoError(t, err)
	// Upload the block.
	uh.Upload(b)
}
