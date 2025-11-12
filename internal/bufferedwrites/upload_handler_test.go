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

	"github.com/googlecloudplatform/gcsfuse/v3/internal/block"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	storagemock "github.com/googlecloudplatform/gcsfuse/v3/internal/storage/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/semaphore"
)

const (
	blockSize         = 1024
	maxBlocks  int64  = 5
	objectName        = "testObject"
	objectSize uint64 = 1024
)

var finalized = time.Date(2025, time.June, 18, 23, 30, 0, 0, time.UTC)

type UploadHandlerTest struct {
	uh         *UploadHandler
	blockPool  *block.GenBlockPool[block.Block]
	mockBucket *storagemock.TestifyMockBucket
	suite.Suite
}

func TestUploadHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(UploadHandlerTest))
}

func (t *UploadHandlerTest) SetupTest() {
	t.mockBucket = new(storagemock.TestifyMockBucket)
	var err error
	t.blockPool, err = block.NewBlockPool(blockSize, maxBlocks, 1, semaphore.NewWeighted(maxBlocks))
	require.NoError(t.T(), err)
	t.uh = newUploadHandler(&CreateUploadHandlerRequest{
		Object:                   nil,
		ObjectName:               "testObject",
		Bucket:                   t.mockBucket,
		BlockPool:                t.blockPool,
		MaxBlocksPerFile:         maxBlocks,
		BlockSize:                blockSize,
		ChunkTransferTimeoutSecs: chunkTransferTimeoutSecs,
	})
}

func (t *UploadHandlerTest) SetupSubTest() {
	t.SetupTest()
}

func (t *UploadHandlerTest) createUploadHandlerWithObjectOfGivenSize(size uint64, finalized time.Time) {
	t.uh = newUploadHandler(&CreateUploadHandlerRequest{
		Object: &gcs.Object{
			Name:      objectName,
			Size:      size,
			Finalized: finalized,
		},
		ObjectName:               "testObject",
		Bucket:                   t.mockBucket,
		BlockPool:                t.blockPool,
		MaxBlocksPerFile:         maxBlocks,
		BlockSize:                blockSize,
		ChunkTransferTimeoutSecs: chunkTransferTimeoutSecs,
	})
}

func (t *UploadHandlerTest) TestCreateObjectWriter_CreateAppendableObjectWriterCalled() {
	t.createUploadHandlerWithObjectOfGivenSize(objectSize, time.Time{})
	t.mockBucket.On("BucketType").Return(gcs.BucketType{Zonal: true})
	t.mockBucket.On("CreateAppendableObjectWriter", mock.Anything, mock.Anything).Return(&storagemock.Writer{}, nil)

	_ = t.uh.createObjectWriter()

	t.mockBucket.AssertCalled(t.T(), "CreateAppendableObjectWriter", mock.Anything, mock.Anything)
}

func (t *UploadHandlerTest) TestCreateObjectWriter_CreateObjectChunkWriterCalled() {
	t.createUploadHandlerWithObjectOfGivenSize(0, finalized)
	t.mockBucket.On("BucketType").Return(gcs.BucketType{})
	t.mockBucket.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything).Return(&storagemock.Writer{}, nil)

	_ = t.uh.createObjectWriter()

	t.mockBucket.AssertCalled(t.T(), "CreateObjectChunkWriter", mock.Anything, mock.Anything)
}

func (t *UploadHandlerTest) TestCreateObjectWriter_CreateObjectChunkWriterCalledForLocalFile() {
	t.mockBucket.On("BucketType").Return(gcs.BucketType{})
	t.mockBucket.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything).Return(&storagemock.Writer{}, nil)

	_ = t.uh.createObjectWriter()

	t.mockBucket.AssertCalled(t.T(), "CreateObjectChunkWriter", mock.Anything, mock.Anything)
}

func (t *UploadHandlerTest) TestEnsureWriter_CreateAppendableWriterIsSuccessful() {
	t.createUploadHandlerWithObjectOfGivenSize(objectSize, time.Time{})
	t.mockBucket.On("BucketType").Return(gcs.BucketType{Zonal: true})
	writer := &storagemock.Writer{}
	t.mockBucket.On("CreateAppendableObjectWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(writer, nil)

	err := t.uh.createObjectWriter()

	assert.Nil(t.T(), err)
	assert.NotNil(t.T(), t.uh.writer)
}
func (t *UploadHandlerTest) TestEnsureWriter_CreateAppendableWriterReturnsError() {
	t.createUploadHandlerWithObjectOfGivenSize(objectSize, time.Time{})
	t.mockBucket.On("BucketType").Return(gcs.BucketType{Zonal: true})
	expectedErr := fmt.Errorf("createAppendableObjectWriter failed")
	t.mockBucket.On("CreateAppendableObjectWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, expectedErr)

	err := t.uh.ensureWriter()

	assert.NotNil(t.T(), err)
	assert.Nil(t.T(), t.uh.writer)
}

func (t *UploadHandlerTest) TestEnsureWriter_CreateObjectChunkWriterIsSuccessful() {
	t.createUploadHandlerWithObjectOfGivenSize(0, finalized)
	t.mockBucket.On("BucketType").Return(gcs.BucketType{})
	writer := &storagemock.Writer{}
	t.mockBucket.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(writer, nil)

	err := t.uh.ensureWriter()

	assert.Nil(t.T(), err)
	assert.NotNil(t.T(), t.uh.writer)
}
func (t *UploadHandlerTest) TestEnsureWriter_CreateObjectChunkWriterReturnsError() {
	t.createUploadHandlerWithObjectOfGivenSize(0, finalized)
	t.mockBucket.On("BucketType").Return(gcs.BucketType{})
	expectedErr := fmt.Errorf("createObjectChunkWriter failed")
	t.mockBucket.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, expectedErr)

	err := t.uh.ensureWriter()

	assert.NotNil(t.T(), err)
	assert.Nil(t.T(), t.uh.writer)
}

func (t *UploadHandlerTest) TestMultipleBlockUpload() {
	// CreateObjectChunkWriter -- should be called once.
	writer := &storagemock.Writer{}
	mockObj := &gcs.MinObject{}
	t.mockBucket.On("BucketType").Return(gcs.BucketType{})
	t.mockBucket.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(writer, nil)
	t.mockBucket.On("FinalizeUpload", mock.Anything, writer).Return(mockObj, nil)

	// Upload the blocks.
	blocks := t.createBlocks(5)
	for _, b := range blocks {
		err := t.uh.Upload(b)
		require.NoError(t.T(), err)
	}

	// Finalize.
	obj, err := t.uh.Finalize()
	require.NoError(t.T(), err)
	require.NotNil(t.T(), obj)
	assert.Equal(t.T(), mockObj, obj)
	// All the 5 blocks should be available on the free channel for reuse.
	assert.Equal(t.T(), 5, t.uh.blockPool.TotalFreeBlocks())
	for _, expect := range blocks {
		got, err := t.uh.blockPool.Get()
		require.NoError(t.T(), err)
		assert.Equal(t.T(), expect, got)
	}
	assert.Equal(t.T(), 0, t.uh.blockPool.TotalFreeBlocks())
	assertAllBlocksProcessed(t.T(), t.uh)
}

func (t *UploadHandlerTest) TestUploadWhenCreateObjectWriterFails() {
	// Create a block.
	b, err := t.blockPool.Get()
	require.NoError(t.T(), err)
	// CreateObjectChunkWriter -- should be called once.
	t.mockBucket.On("BucketType").Return(gcs.BucketType{})
	t.mockBucket.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("taco"))

	// Upload the block.
	err = t.uh.Upload(b)

	require.Error(t.T(), err)
	assert.ErrorContains(t.T(), err, "createObjectWriter")
	assert.ErrorContains(t.T(), err, "taco")
}

func (t *UploadHandlerTest) TestFinalizeWithWriterAlreadyPresent() {
	writer := &storagemock.Writer{}
	mockObj := &gcs.MinObject{}
	t.mockBucket.On("FinalizeUpload", mock.Anything, writer).Return(mockObj, nil)
	t.uh.writer = writer

	obj, err := t.uh.Finalize()

	require.NoError(t.T(), err)
	require.NotNil(t.T(), obj)
	assert.Equal(t.T(), mockObj, obj)
}

func (t *UploadHandlerTest) TestFinalizeWithNoWriter() {
	writer := &storagemock.Writer{}
	t.mockBucket.On("BucketType").Return(gcs.BucketType{})
	t.mockBucket.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(writer, nil)
	assert.Nil(t.T(), t.uh.writer)
	mockObj := &gcs.MinObject{}
	t.mockBucket.On("FinalizeUpload", mock.Anything, writer).Return(mockObj, nil)

	obj, err := t.uh.Finalize()

	require.NoError(t.T(), err)
	require.NotNil(t.T(), obj)
	assert.Equal(t.T(), mockObj, obj)
}

func (t *UploadHandlerTest) TestFinalizeWithNoWriterWhenCreateObjectWriterFails() {
	t.mockBucket.On("BucketType").Return(gcs.BucketType{})
	t.mockBucket.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("taco"))
	assert.Nil(t.T(), t.uh.writer)

	obj, err := t.uh.Finalize()

	require.Error(t.T(), err)
	assert.ErrorContains(t.T(), err, "taco")
	assert.ErrorContains(t.T(), err, "createObjectWriter")
	assert.Nil(t.T(), obj)
}

func (t *UploadHandlerTest) TestFinalizeWhenFinalizeUploadFails() {
	writer := &storagemock.Writer{}
	t.mockBucket.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(writer, nil)
	assert.Nil(t.T(), t.uh.writer)
	mockObj := &gcs.MinObject{}
	t.mockBucket.On("BucketType").Return(gcs.BucketType{})
	t.mockBucket.On("FinalizeUpload", mock.Anything, writer).Return(mockObj, fmt.Errorf("taco"))

	obj, err := t.uh.Finalize()

	require.Error(t.T(), err)
	assert.Nil(t.T(), obj)
	assert.ErrorContains(t.T(), err, "taco")
	assertUploadFailureError(t.T(), t.uh)
}

func (t *UploadHandlerTest) TestFlushWithWriterAlreadyPresent() {
	writer := &storagemock.Writer{}
	mockObject := &gcs.MinObject{Size: 100}
	t.mockBucket.On("FlushPendingWrites", mock.Anything, writer).Return(mockObject, nil)
	t.uh.writer = writer

	o, err := t.uh.FlushPendingWrites()

	require.NoError(t.T(), err)
	assert.Equal(t.T(), mockObject, o)
}

func (t *UploadHandlerTest) TestFlushWithNoWriter() {
	writer := &storagemock.Writer{}
	t.mockBucket.On("BucketType").Return(gcs.BucketType{})
	t.mockBucket.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(writer, nil)
	assert.Nil(t.T(), t.uh.writer)
	mockObject := &gcs.MinObject{Size: 10}
	t.mockBucket.On("FlushPendingWrites", mock.Anything, writer).Return(mockObject, nil)

	o, err := t.uh.FlushPendingWrites()

	require.NoError(t.T(), err)
	assert.Equal(t.T(), mockObject, o)
}

func (t *UploadHandlerTest) TestFlushWithNoWriterWhenCreateObjectWriterFails() {
	t.mockBucket.On("BucketType").Return(gcs.BucketType{})
	t.mockBucket.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("taco"))
	assert.Nil(t.T(), t.uh.writer)

	o, err := t.uh.FlushPendingWrites()

	require.Error(t.T(), err)
	assert.ErrorContains(t.T(), err, "taco")
	assert.ErrorContains(t.T(), err, "createObjectWriter")
	assert.Nil(t.T(), o)
}

func (t *UploadHandlerTest) TestFlushWhenFlushPendingWritesFails() {
	writer := &storagemock.Writer{}
	t.mockBucket.On("BucketType").Return(gcs.BucketType{})
	t.mockBucket.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(writer, nil)
	assert.Nil(t.T(), t.uh.writer)
	var minObj *gcs.MinObject = nil
	t.mockBucket.On("FlushPendingWrites", mock.Anything, writer).Return(minObj, fmt.Errorf("taco"))

	o, err := t.uh.FlushPendingWrites()

	require.Error(t.T(), err)
	assert.Nil(t.T(), nil, o)
	assert.ErrorContains(t.T(), err, "taco")
	assertUploadFailureError(t.T(), t.uh)
}

func (t *UploadHandlerTest) TestUploadSingleBlockThrowsErrorInCopy() {
	// Create a block with test data.
	b, err := t.blockPool.Get()
	require.NoError(t.T(), err)
	_, err = b.Write([]byte("test data"))
	require.NoError(t.T(), err)
	// CreateObjectChunkWriter -- should be called once.
	writer := &storagemock.Writer{}
	t.mockBucket.On("BucketType").Return(gcs.BucketType{})
	t.mockBucket.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(writer, nil)
	// First write will be an error and Close will be successful.
	writer.On("Write", mock.Anything).Return(0, fmt.Errorf("taco")).Once()

	// Upload the block.
	err = t.uh.Upload(b)

	require.NoError(t.T(), err)
	// Expect an error on upload due to error while copying content to GCS writer.
	assertUploadFailureError(t.T(), t.uh)
	assertAllBlocksProcessed(t.T(), t.uh)
	assert.Equal(t.T(), 1, t.uh.blockPool.TotalFreeBlocks())
}

func (t *UploadHandlerTest) TestUploadMultipleBlocksThrowsErrorInCopy() {
	// Create some blocks.
	blocks := t.createBlocks(4)
	for i := range 4 {
		n, err := blocks[i].Write([]byte("testdata" + strconv.Itoa(i) + " "))
		require.Equal(t.T(), 10, n)
		require.NoError(t.T(), err)
	}
	// CreateObjectChunkWriter -- should be called once.
	writer := &storagemock.Writer{}
	t.mockBucket.On("BucketType").Return(gcs.BucketType{})
	t.mockBucket.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(writer, nil)
	// Second write will be an error and rest of the operations will be successful.
	writer.
		On("Write", mock.Anything).Return(10, nil).Once().
		On("Write", mock.Anything).Return(0, fmt.Errorf("taco"))

	// Upload the blocks.
	for _, b := range blocks {
		err := t.uh.Upload(b)
		require.NoError(t.T(), err)
	}

	assertUploadFailureError(t.T(), t.uh)
	assertAllBlocksProcessed(t.T(), t.uh)
	assert.Equal(t.T(), 4, t.uh.blockPool.TotalFreeBlocks())
}

func assertUploadFailureError(t *testing.T, handler *UploadHandler) {
	t.Helper()
	for {
		select {
		case <-time.After(200 * time.Millisecond):
			t.Error("Expected an error in uploader")
		default:
			if handler.UploadError() != nil {
				return
			}
		}
	}
}

func assertAllBlocksProcessed(t *testing.T, handler *UploadHandler) {
	t.Helper()

	// All blocks for upload should have been processed.
	done := make(chan struct{})
	go func() {
		handler.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for WaitGroup")
	}
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

func (t *UploadHandlerTest) TestMultipleBlockAwaitBlocksUpload() {
	// CreateObjectChunkWriter -- should be called once.
	writer := &storagemock.Writer{}
	t.mockBucket.On("BucketType").Return(gcs.BucketType{})
	t.mockBucket.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(writer, nil)
	// Upload the blocks.
	for _, b := range t.createBlocks(5) {
		err := t.uh.Upload(b)
		require.NoError(t.T(), err)
	}

	// AwaitBlocksUpload.
	t.uh.AwaitBlocksUpload()

	assert.Equal(t.T(), 5, t.uh.blockPool.TotalFreeBlocks())
	assert.Equal(t.T(), 0, len(t.uh.uploadCh))
	assertAllBlocksProcessed(t.T(), t.uh)
}

func (t *UploadHandlerTest) TestUploadHandlerCancelUpload() {
	cancelCalled := false
	t.uh.cancelFunc = func() { cancelCalled = true }

	t.uh.CancelUpload()

	assert.True(t.T(), cancelCalled)
}

func (t *UploadHandlerTest) TestCreateObjectChunkWriterIsCalledWithCorrectRequestParametersForEmptyGCSObject() {
	t.uh.obj = &gcs.Object{
		Name:            t.uh.objectName,
		ContentType:     "image/png",
		Size:            0,
		ContentEncoding: "gzip",
		Generation:      10,
		MetaGeneration:  20,
		Acl:             nil,
		Finalized:       finalized,
	}

	// CreateObjectChunkWriter -- should be called once with correct request parameters.
	writer := &storagemock.Writer{}
	mockObj := &gcs.Object{}
	t.mockBucket.On("BucketType").Return(gcs.BucketType{})
	t.mockBucket.On("CreateObjectChunkWriter",
		mock.Anything,
		mock.MatchedBy(func(req *gcs.CreateObjectRequest) bool {
			return req.Name == t.uh.objectName &&
				*req.GenerationPrecondition == t.uh.obj.Generation &&
				*req.MetaGenerationPrecondition == t.uh.obj.MetaGeneration &&
				req.ContentEncoding == t.uh.obj.ContentEncoding &&
				req.ContentType == t.uh.obj.ContentType &&
				req.ChunkTransferTimeoutSecs == chunkTransferTimeoutSecs
		}),
		mock.Anything,
		mock.Anything).Return(writer, nil)
	t.mockBucket.On("FinalizeUpload", mock.Anything, writer).Return(mockObj, nil)

	// Create a block.
	b, err := t.blockPool.Get()
	require.NoError(t.T(), err)
	// Upload the block.
	err = t.uh.Upload(b)
	require.NoError(t.T(), err)
}

func (t *UploadHandlerTest) TestCreateObjectChunkWriterIsCalledWithCorrectRequestParametersForLocalInode() {
	assert.Nil(t.T(), t.uh.obj)

	// CreateObjectChunkWriter -- should be called once with correct request parameters.
	writer := &storagemock.Writer{}
	mockObj := &gcs.Object{}
	t.mockBucket.On("BucketType").Return(gcs.BucketType{})
	t.mockBucket.On("CreateObjectChunkWriter",
		mock.Anything,
		mock.MatchedBy(func(req *gcs.CreateObjectRequest) bool {
			return req.Name == t.uh.objectName &&
				*req.GenerationPrecondition == 0 &&
				req.MetaGenerationPrecondition == nil &&
				req.ChunkTransferTimeoutSecs == chunkTransferTimeoutSecs
		}),
		mock.Anything,
		mock.Anything).Return(writer, nil)
	t.mockBucket.On("FinalizeUpload", mock.Anything, writer).Return(mockObj, nil)

	// Create a block.
	b, err := t.blockPool.Get()
	require.NoError(t.T(), err)
	// Upload the block.
	err = t.uh.Upload(b)
	require.NoError(t.T(), err)
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

			assertAllBlocksProcessed(t.T(), t.uh)
			assert.Equal(t.T(), 5, t.uh.blockPool.TotalFreeBlocks())
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

func (t *UploadHandlerTest) createBlocks(count int) []block.Block {
	var blocks []block.Block
	for range count {
		b, err := t.blockPool.Get()
		require.NoError(t.T(), err)
		blocks = append(blocks, b)
	}

	return blocks
}
