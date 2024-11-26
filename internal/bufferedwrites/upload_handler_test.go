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
	storagemock "github.com/googlecloudplatform/gcsfuse/v2/internal/storage/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/semaphore"
)

const (
	blockSize = 1024
)

type UploadHandlerTest struct {
	uh         *UploadHandler
	blockPool  *block.BlockPool
	mockBucket *storagemock.TestifyMockBucket
	suite.Suite
}

func TestUploadHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(UploadHandlerTest))
}

func (t *UploadHandlerTest) SetupTest() {
	var maxBlocks int64 = 5
	t.mockBucket = new(storagemock.TestifyMockBucket)
	var err error
	t.blockPool, err = block.NewBlockPool(blockSize, maxBlocks, semaphore.NewWeighted(5))
	require.NoError(t.T(), err)
	t.uh = newUploadHandler("testObject", t.mockBucket, maxBlocks, t.blockPool.FreeBlocksChannel(), blockSize)
}

func (t *UploadHandlerTest) TestMultipleBlockUpload() {
	// Create some blocks.
	var blocks []block.Block
	for i := 0; i < 5; i++ {
		b, err := t.blockPool.Get()
		require.NoError(t.T(), err)
		blocks = append(blocks, b)
	}
	// CreateObjectChunkWriter -- should be called once.
	writer := &storagemock.Writer{}
	t.mockBucket.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(writer, nil)
	writer.On("Close").Return(nil)

	// Upload the blocks.
	for _, b := range blocks {
		err := t.uh.Upload(b)
		require.NoError(t.T(), err)
	}

	// Finalize.
	err := t.uh.Finalize()
	require.NoError(t.T(), err)
	// The blocks should be available on the free channel for reuse.
	for _, expect := range blocks {
		got := <-t.uh.freeBlocksCh
		assert.Equal(t.T(), expect, got)
	}
	// All goroutines for upload should have exited.
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

func (t *UploadHandlerTest) TestUpload_CreateObjectWriterFails() {
	// Create a block.
	b, err := t.blockPool.Get()
	require.NoError(t.T(), err)
	// CreateObjectChunkWriter -- should be called once.
	t.mockBucket.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("taco"))

	// Upload the block.
	err = t.uh.Upload(b)

	assert.ErrorContains(t.T(), err, "createObjectWriter")
	assert.ErrorContains(t.T(), err, "taco")
}

func (t *UploadHandlerTest) TestFinalizeWithWriterAlreadyPresent() {
	writer := &storagemock.Writer{}
	writer.On("Close").Return(nil)
	t.uh.writer = writer

	err := t.uh.Finalize()

	assert.NoError(t.T(), err)
}

func (t *UploadHandlerTest) TestFinalizeWithNoWriter() {
	writer := &storagemock.Writer{}
	t.mockBucket.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(writer, nil)
	assert.Nil(t.T(), t.uh.writer)
	writer.On("Close").Return(nil)

	err := t.uh.Finalize()

	assert.NoError(t.T(), err)
}

func (t *UploadHandlerTest) TestFinalizeWithNoWriter_CreateObjectWriterFails() {
	t.mockBucket.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("taco"))
	assert.Nil(t.T(), t.uh.writer)

	err := t.uh.Finalize()

	assert.Error(t.T(), err)
	assert.ErrorContains(t.T(), err, "taco")
	assert.ErrorContains(t.T(), err, "createObjectWriter")
}

func (t *UploadHandlerTest) TestFinalize_WriterCloseFails() {
	writer := &storagemock.Writer{}
	t.mockBucket.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(writer, nil)
	assert.Nil(t.T(), t.uh.writer)
	writer.On("Close").Return(fmt.Errorf("taco"))

	err := t.uh.Finalize()

	assert.Error(t.T(), err)
	assert.ErrorContains(t.T(), err, "writer.Close")
	assertUploadFailureSignal(t.T(), t.uh)
}

func (t *UploadHandlerTest) TestUploadHandler_singleBlock_ErrorInCopy() {
	// Create a block with test data.
	b, err := t.blockPool.Get()
	require.NoError(t.T(), err)
	err = b.Write([]byte("test data"))
	require.NoError(t.T(), err)
	// CreateObjectChunkWriter -- should be called once.
	writer := &storagemock.Writer{}
	t.mockBucket.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(writer, nil)
	// First write will be an error and Close will be successful.
	writer.
		On("Write", mock.Anything).Return(0, fmt.Errorf("taco")).Once().
		On("Close").Return(nil)

	// Upload the block.
	err = t.uh.Upload(b)

	require.NoError(t.T(), err)
	// Expect an error on the signalUploadFailure channel due to error while copying content to GCS writer.
	assertUploadFailureSignal(t.T(), t.uh)
}

func (t *UploadHandlerTest) TestUploadHandler_multipleBlocks_ErrorInCopy() {
	// Create some blocks.
	var blocks []block.Block
	for i := 0; i < 4; i++ {
		b, err := t.blockPool.Get()
		require.NoError(t.T(), err)
		err = b.Write([]byte("testdata" + strconv.Itoa(i) + " "))
		require.NoError(t.T(), err)
		blocks = append(blocks, b)
	}
	// CreateObjectChunkWriter -- should be called once.
	writer := &storagemock.Writer{}
	t.mockBucket.On("CreateObjectChunkWriter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(writer, nil)
	// Second write will be an error and rest of the operations will be successful.
	writer.
		On("Write", mock.Anything).Return(10, nil).Once().
		On("Write", mock.Anything).Return(0, fmt.Errorf("taco")).
		On("Close").Return(nil)

	// Upload the blocks.
	for _, b := range blocks {
		err := t.uh.Upload(b)
		require.NoError(t.T(), err)
	}

	assertUploadFailureSignal(t.T(), t.uh)
}

func assertUploadFailureSignal(t *testing.T, handler *UploadHandler) {
	t.Helper()

	select {
	case <-handler.signalUploadFailure:
		break
	case <-time.After(200 * time.Millisecond):
		t.Error("Expected an error on signalUploadFailure channel")
	}
}

func TestBufferedWriteHandler_SignalUploadFailure(t *testing.T) {
	mockSignalUploadFailure := make(chan error)
	uploadHandler := &UploadHandler{
		signalUploadFailure: mockSignalUploadFailure,
	}

	actualChannel := uploadHandler.SignalUploadFailure()

	assert.Equal(t, mockSignalUploadFailure, actualChannel)
}
