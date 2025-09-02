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

package storage

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/storage/control/apiv2/controlpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	control "cloud.google.com/go/storage/control/apiv2"
	"github.com/googleapis/gax-go/v2"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
)

// stallingStorageControlClient is a wrapper that introduces a controllable delay
// to every call, to simulate network latency for testing timeout-based retries.
type stallingStorageControlClient struct {
	wrapped                          StorageControlClient
	stallDurationForGetStorageLayout *time.Duration
	stallDurationForFolderAPIs       *time.Duration
}

func (s *stallingStorageControlClient) GetStorageLayout(ctx context.Context, req *controlpb.GetStorageLayoutRequest, opts ...gax.CallOption) (*controlpb.StorageLayout, error) {
	if s.stallDurationForGetStorageLayout != nil {
		select {
		case <-time.After(*s.stallDurationForGetStorageLayout):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return s.wrapped.GetStorageLayout(ctx, req, opts...)
}

func (s *stallingStorageControlClient) DeleteFolder(ctx context.Context, req *controlpb.DeleteFolderRequest, opts ...gax.CallOption) error {
	if s.stallDurationForFolderAPIs != nil {
		select {
		case <-time.After(*s.stallDurationForFolderAPIs):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return s.wrapped.DeleteFolder(ctx, req, opts...)
}

func (s *stallingStorageControlClient) GetFolder(ctx context.Context, req *controlpb.GetFolderRequest, opts ...gax.CallOption) (*controlpb.Folder, error) {
	if s.stallDurationForFolderAPIs != nil {
		select {
		case <-time.After(*s.stallDurationForFolderAPIs):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return s.wrapped.GetFolder(ctx, req, opts...)
}

func (s *stallingStorageControlClient) RenameFolder(ctx context.Context, req *controlpb.RenameFolderRequest, opts ...gax.CallOption) (*control.RenameFolderOperation, error) {
	if s.stallDurationForFolderAPIs != nil {
		select {
		case <-time.After(*s.stallDurationForFolderAPIs):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return s.wrapped.RenameFolder(ctx, req, opts...)
}

func (s *stallingStorageControlClient) CreateFolder(ctx context.Context, req *controlpb.CreateFolderRequest, opts ...gax.CallOption) (*controlpb.Folder, error) {
	if s.stallDurationForFolderAPIs != nil {
		select {
		case <-time.After(*s.stallDurationForFolderAPIs):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return s.wrapped.CreateFolder(ctx, req, opts...)
}

type ControlClientRetryWrapperTest struct {
	suite.Suite
	// The raw mock client for setting expectations on return values.
	mockRawClient  *MockStorageControlClient
	ctx            context.Context
	stallingClient *stallingStorageControlClient
	// The simulated execution time for each GetStorageLayout call made through stallingClient.
	stallDurationForGetStorageLayout time.Duration
}

type StorageLayoutRetryWrapperTest struct {
	ControlClientRetryWrapperTest
}

type AllApiRetryWrapperTest struct {
	ControlClientRetryWrapperTest
	// The execution time for each folder API call made through stallingClient. Can be adjusted
	// per test.
	stallDurationForFolderAPIs time.Duration
}

func TestControlClientWrapperTestSuite(t *testing.T) {
	t.Run("StorageLayoutRetryWrapperTest", func(t *testing.T) {
		suite.Run(t, new(StorageLayoutRetryWrapperTest))
	})
	t.Run("AllApiRetryWrapperTest", func(t *testing.T) {
		suite.Run(t, new(AllApiRetryWrapperTest))
	})
}

func (t *ControlClientRetryWrapperTest) SetupTest() {
	t.mockRawClient = new(MockStorageControlClient)
	t.ctx = context.Background()
	t.stallDurationForGetStorageLayout = 0
}

func (t *StorageLayoutRetryWrapperTest) SetupTest() {
	t.ControlClientRetryWrapperTest.SetupTest()
	t.stallingClient = &stallingStorageControlClient{
		wrapped:                          t.mockRawClient,
		stallDurationForGetStorageLayout: &t.stallDurationForGetStorageLayout,
	}
}

func (t *AllApiRetryWrapperTest) SetupTest() {
	t.ControlClientRetryWrapperTest.SetupTest()
	t.stallDurationForFolderAPIs = 0
	t.stallingClient = &stallingStorageControlClient{
		wrapped:                          t.mockRawClient,
		stallDurationForGetStorageLayout: &t.stallDurationForGetStorageLayout,
		stallDurationForFolderAPIs:       &t.stallDurationForFolderAPIs,
	}
}

func (t *StorageLayoutRetryWrapperTest) TestGetStorageLayout_SuccessOnFirstAttempt() {
	// Arrange
	client := newRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, 1*time.Microsecond, 10*time.Microsecond, 2, false)
	req := &controlpb.GetStorageLayoutRequest{Name: "some/bucket"}
	expectedLayout := &controlpb.StorageLayout{Location: "some-location"}
	t.mockRawClient.On("GetStorageLayout", mock.Anything, req, mock.Anything).Return(expectedLayout, nil).Once()

	// Act
	layout, err := client.GetStorageLayout(t.ctx, req)

	// Assert
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), expectedLayout, layout)
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *StorageLayoutRetryWrapperTest) TestGetStorageLayout_RetryableErrorThenSuccess() {
	// Arrange
	client := newRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, 1*time.Microsecond, 10*time.Microsecond, 2, false)
	req := &controlpb.GetStorageLayoutRequest{Name: "some/bucket"}
	expectedLayout := &controlpb.StorageLayout{Location: "some-location"}
	retryableErr := status.Error(codes.Unavailable, "try again")
	// First call fails, second succeeds.
	t.mockRawClient.On("GetStorageLayout", mock.Anything, req, mock.Anything).Return(nil, retryableErr).Once()
	t.mockRawClient.On("GetStorageLayout", mock.Anything, req, mock.Anything).Return(expectedLayout, nil).Once()

	// Act
	layout, err := client.GetStorageLayout(t.ctx, req)

	// Assert
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), expectedLayout, layout)
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *StorageLayoutRetryWrapperTest) TestGetStorageLayout_NonRetryableError() {
	// Arrange
	client := newRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, 1*time.Microsecond, 10*time.Microsecond, 2, false)
	req := &controlpb.GetStorageLayoutRequest{Name: "some/bucket"}
	nonRetryableErr := status.Error(codes.NotFound, "does not exist")
	t.mockRawClient.On("GetStorageLayout", mock.Anything, req, mock.Anything).Return(nil, nonRetryableErr).Once()

	// Act
	layout, err := client.GetStorageLayout(t.ctx, req)

	// Assert
	assert.Error(t.T(), err)
	assert.Nil(t.T(), layout)
	assert.Contains(t.T(), err.Error(), "failed with a non-retryable error")
	assert.Contains(t.T(), err.Error(), nonRetryableErr.Error())
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *StorageLayoutRetryWrapperTest) TestGetStorageLayout_AllAttemptsTimeOut() {
	// Arrange
	// This test requires different retry parameters, so we create a new client.
	client := newRetryWrapper(t.stallingClient, 1000*time.Microsecond, 10000*time.Microsecond, 1*time.Microsecond, 10*time.Microsecond, 2, false)
	req := &controlpb.GetStorageLayoutRequest{Name: "some/bucket"}
	// Set stall time to be longer than the attempt timeout.
	t.stallDurationForGetStorageLayout = 6000 * time.Microsecond

	// Act
	_, err := client.GetStorageLayout(t.ctx, req)

	// The mock should never be called because every attempt will time out.
	assert.ErrorIs(t.T(), err, context.DeadlineExceeded)
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *StorageLayoutRetryWrapperTest) TestGetFolder_IsNotRetried() {
	// Arrange
	client := newRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, 1*time.Microsecond, 10*time.Microsecond, 2, false)
	req := &controlpb.GetFolderRequest{Name: "some/folder"}
	retryableErr := status.Error(codes.Unavailable, "try again")
	// Mock the raw client to return a retryable error once.
	t.mockRawClient.On("GetFolder", mock.Anything, req, mock.Anything).Return(nil, retryableErr).Once()

	// Act
	folder, err := client.GetFolder(t.ctx, req)

	// Assert
	assert.Error(t.T(), err)
	assert.Nil(t.T(), folder)
	assert.Equal(t.T(), retryableErr, err)
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *StorageLayoutRetryWrapperTest) TestDeleteFolder_IsNotRetried() {
	// Arrange
	client := newRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, 1*time.Microsecond, 10*time.Microsecond, 2, false)
	req := &controlpb.DeleteFolderRequest{Name: "some/folder"}
	retryableErr := status.Error(codes.Unavailable, "try again")
	// Mock the raw client to return a retryable error once.
	t.mockRawClient.On("DeleteFolder", mock.Anything, req, mock.Anything).Return(retryableErr).Once()

	// Act
	err := client.DeleteFolder(t.ctx, req)

	// Assert
	assert.Error(t.T(), err)
	assert.Equal(t.T(), retryableErr, err)
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *StorageLayoutRetryWrapperTest) TestCreateFolder_IsNotRetried() {
	// Arrange
	client := newRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, 1*time.Microsecond, 10*time.Microsecond, 2, false)
	req := &controlpb.CreateFolderRequest{Parent: "some/", FolderId: "folder"}
	retryableErr := status.Error(codes.Unavailable, "try again")
	// Mock the raw client to return a retryable error once.
	t.mockRawClient.On("CreateFolder", mock.Anything, req, mock.Anything).Return(nil, retryableErr).Once()

	// Act
	folder, err := client.CreateFolder(t.ctx, req)

	// Assert
	assert.Error(t.T(), err)
	assert.Nil(t.T(), folder)
	assert.Equal(t.T(), retryableErr, err)
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *StorageLayoutRetryWrapperTest) TestRenameFolder_IsNotRetried() {
	// Arrange
	client := newRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, 1*time.Microsecond, 10*time.Microsecond, 2, false)
	req := &controlpb.RenameFolderRequest{Name: "some/folder", DestinationFolderId: "new/folder"}
	retryableErr := status.Error(codes.Unavailable, "try again")
	// Mock the raw client to return a retryable error once.
	t.mockRawClient.On("RenameFolder", mock.Anything, req, mock.Anything).Return(nil, retryableErr).Once()

	// Act
	op, err := client.RenameFolder(t.ctx, req)

	// Assert
	assert.Error(t.T(), err)
	assert.Nil(t.T(), op)
	assert.Equal(t.T(), retryableErr, err)
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *AllApiRetryWrapperTest) TestGetStorageLayout_SuccessOnFirstAttempt() {
	// Arrange
	client := newRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
	req := &controlpb.GetStorageLayoutRequest{Name: "some/bucket"}
	expectedLayout := &controlpb.StorageLayout{Location: "some-location"}
	t.mockRawClient.On("GetStorageLayout", mock.Anything, req, mock.Anything).Return(expectedLayout, nil).Once()

	// Act
	layout, err := client.GetStorageLayout(t.ctx, req)

	// Assert
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), expectedLayout, layout)
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *AllApiRetryWrapperTest) TestGetStorageLayout_RetryableErrorThenSuccess() {
	// Arrange
	client := newRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
	req := &controlpb.GetStorageLayoutRequest{Name: "some/bucket"}
	expectedLayout := &controlpb.StorageLayout{Location: "some-location"}
	retryableErr := status.Error(codes.Unavailable, "try again")
	// First call fails, second succeeds.
	t.mockRawClient.On("GetStorageLayout", mock.Anything, req, mock.Anything).Return(nil, retryableErr).Once()
	t.mockRawClient.On("GetStorageLayout", mock.Anything, req, mock.Anything).Return(expectedLayout, nil).Once()

	// Act
	layout, err := client.GetStorageLayout(t.ctx, req)

	// Assert
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), expectedLayout, layout)
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *AllApiRetryWrapperTest) TestGetStorageLayout_NonRetryableError() {
	// Arrange
	client := newRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
	req := &controlpb.GetStorageLayoutRequest{Name: "some/bucket"}
	nonRetryableErr := status.Error(codes.NotFound, "does not exist")
	t.mockRawClient.On("GetStorageLayout", mock.Anything, req, mock.Anything).Return(nil, nonRetryableErr).Once()

	// Act
	layout, err := client.GetStorageLayout(t.ctx, req)

	// Assert
	assert.Error(t.T(), err)
	assert.Nil(t.T(), layout)
	assert.Contains(t.T(), err.Error(), "failed with a non-retryable error")
	assert.Contains(t.T(), err.Error(), nonRetryableErr.Error())
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *AllApiRetryWrapperTest) TestGetStorageLayout_AllAttemptsTimeOut() {
	// Arrange
	client := newRetryWrapper(t.stallingClient, 1000*time.Microsecond, 10000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
	req := &controlpb.GetStorageLayoutRequest{Name: "some/bucket"}
	// Set stall time to be longer than the attempt timeout.
	t.stallDurationForGetStorageLayout = 6000 * time.Microsecond

	// Act
	_, err := client.GetStorageLayout(t.ctx, req)

	// The mock should never be called because every attempt will time out.
	assert.ErrorIs(t.T(), err, context.DeadlineExceeded)
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *AllApiRetryWrapperTest) TestDeleteFolder_SuccessOnFirstAttempt() {
	// Arrange
	client := newRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
	req := &controlpb.DeleteFolderRequest{Name: "some/folder"}
	t.mockRawClient.On("DeleteFolder", mock.Anything, req, mock.Anything).Return(nil).Once()

	// Act
	err := client.DeleteFolder(t.ctx, req)

	// Assert
	assert.NoError(t.T(), err)
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *AllApiRetryWrapperTest) TestDeleteFolder_RetryableErrorThenSuccess() {
	// Arrange
	client := newRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
	req := &controlpb.DeleteFolderRequest{Name: "some/folder"}
	retryableErr := status.Error(codes.Unavailable, "try again")
	// First call fails, second succeeds.
	t.mockRawClient.On("DeleteFolder", mock.Anything, req, mock.Anything).Return(retryableErr).Once()
	t.mockRawClient.On("DeleteFolder", mock.Anything, req, mock.Anything).Return(nil).Once()

	// Act
	err := client.DeleteFolder(t.ctx, req)

	// Assert
	assert.NoError(t.T(), err)
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *AllApiRetryWrapperTest) TestDeleteFolder_NonRetryableError() {
	// Arrange
	client := newRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
	req := &controlpb.DeleteFolderRequest{Name: "some/folder"}
	nonRetryableErr := status.Error(codes.NotFound, "does not exist")
	t.mockRawClient.On("DeleteFolder", mock.Anything, req, mock.Anything).Return(nonRetryableErr).Once()

	// Act
	err := client.DeleteFolder(t.ctx, req)

	// Assert
	assert.Error(t.T(), err)
	assert.Contains(t.T(), err.Error(), "failed with a non-retryable error")
	assert.Contains(t.T(), err.Error(), nonRetryableErr.Error())
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *AllApiRetryWrapperTest) TestDeleteFolder_AllAttemptsTimeOut() {
	// Arrange
	client := newRetryWrapper(t.stallingClient, 1000*time.Microsecond, 10000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
	req := &controlpb.DeleteFolderRequest{Name: "some/folder"}
	// Set stall time to be longer than the attempt timeout.
	t.stallDurationForFolderAPIs = 6000 * time.Microsecond

	// Act
	err := client.DeleteFolder(t.ctx, req)

	// The mock should never be called because every attempt will time out.
	assert.Error(t.T(), err)
	assert.ErrorIs(t.T(), err, context.DeadlineExceeded)
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *AllApiRetryWrapperTest) TestGetFolder_SuccessOnFirstAttempt() {
	// Arrange
	client := newRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
	req := &controlpb.GetFolderRequest{Name: "some/folder"}
	expectedFolder := &controlpb.Folder{Name: "some/folder"}
	t.mockRawClient.On("GetFolder", mock.Anything, req, mock.Anything).Return(expectedFolder, nil).Once()

	// Act
	folder, err := client.GetFolder(t.ctx, req)

	// Assert
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), expectedFolder, folder)
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *AllApiRetryWrapperTest) TestGetFolder_RetryableErrorThenSuccess() {
	// Arrange
	client := newRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
	req := &controlpb.GetFolderRequest{Name: "some/folder"}
	expectedFolder := &controlpb.Folder{Name: "some/folder"}
	retryableErr := status.Error(codes.Unavailable, "try again")
	// First call fails, second succeeds.
	t.mockRawClient.On("GetFolder", mock.Anything, req, mock.Anything).Return(nil, retryableErr).Once()
	t.mockRawClient.On("GetFolder", mock.Anything, req, mock.Anything).Return(expectedFolder, nil).Once()

	// Act
	folder, err := client.GetFolder(t.ctx, req)

	// Assert
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), expectedFolder, folder)
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *AllApiRetryWrapperTest) TestGetFolder_NonRetryableError() {
	// Arrange
	client := newRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
	req := &controlpb.GetFolderRequest{Name: "some/folder"}
	nonRetryableErr := status.Error(codes.NotFound, "does not exist")
	t.mockRawClient.On("GetFolder", mock.Anything, req, mock.Anything).Return(nil, nonRetryableErr).Once()

	// Act
	folder, err := client.GetFolder(t.ctx, req)

	// Assert
	assert.Error(t.T(), err)
	assert.Nil(t.T(), folder)
	assert.Contains(t.T(), err.Error(), "failed with a non-retryable error")
	assert.Contains(t.T(), err.Error(), nonRetryableErr.Error())
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *AllApiRetryWrapperTest) TestGetFolder_AllAttemptsTimeOut() {
	// Arrange
	client := newRetryWrapper(t.stallingClient, 1000*time.Microsecond, 10000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
	req := &controlpb.GetFolderRequest{Name: "some/folder"}
	// Set execution time to be longer than the attempt timeout.
	t.stallDurationForFolderAPIs = 6000 * time.Microsecond

	// Act
	_, err := client.GetFolder(t.ctx, req)

	// Assert: The mock should never be called because every attempt will time out.
	assert.Error(t.T(), err)
	assert.ErrorIs(t.T(), err, context.DeadlineExceeded)
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *AllApiRetryWrapperTest) TestRenameFolder_SuccessOnFirstAttempt() {
	// Arrange
	client := newRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
	req := &controlpb.RenameFolderRequest{Name: "some/folder", DestinationFolderId: "new/folder"}
	expectedOp := &control.RenameFolderOperation{}
	t.mockRawClient.On("RenameFolder", mock.Anything, req, mock.Anything).Return(expectedOp, nil).Once()

	// Act
	op, err := client.RenameFolder(t.ctx, req)

	// Assert
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), expectedOp, op)
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *AllApiRetryWrapperTest) TestRenameFolder_RetryableErrorThenSuccess() {
	// Arrange
	client := newRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
	req := &controlpb.RenameFolderRequest{Name: "some/folder", DestinationFolderId: "new/folder"}
	expectedOp := &control.RenameFolderOperation{}
	retryableErr := status.Error(codes.Unavailable, "try again")
	// First call fails, second succeeds.
	t.mockRawClient.On("RenameFolder", mock.Anything, req, mock.Anything).Return(nil, retryableErr).Once()
	t.mockRawClient.On("RenameFolder", mock.Anything, req, mock.Anything).Return(expectedOp, nil).Once()

	// Act
	op, err := client.RenameFolder(t.ctx, req)

	// Assert
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), expectedOp, op)
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *AllApiRetryWrapperTest) TestRenameFolder_NonRetryableError() {
	// Arrange
	client := newRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
	req := &controlpb.RenameFolderRequest{Name: "some/folder", DestinationFolderId: "new/folder"}
	nonRetryableErr := status.Error(codes.NotFound, "does not exist")
	t.mockRawClient.On("RenameFolder", mock.Anything, req, mock.Anything).Return(nil, nonRetryableErr).Once()

	// Act
	op, err := client.RenameFolder(t.ctx, req)

	// Assert
	assert.Error(t.T(), err)
	assert.Nil(t.T(), op)
	assert.Contains(t.T(), err.Error(), "failed with a non-retryable error")
	assert.Contains(t.T(), err.Error(), nonRetryableErr.Error())
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *AllApiRetryWrapperTest) TestRenameFolder_AllAttemptsTimeOut() {
	// Arrange
	client := newRetryWrapper(t.stallingClient, 1000*time.Microsecond, 10000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
	req := &controlpb.RenameFolderRequest{Name: "some/folder", DestinationFolderId: "new/folder"}
	// Set execution time to be longer than the attempt timeout.
	t.stallDurationForFolderAPIs = 6000 * time.Microsecond

	// Act
	_, err := client.RenameFolder(t.ctx, req)

	// Assert: The mock should never be called because every attempt will time out.
	assert.Error(t.T(), err)
	assert.ErrorIs(t.T(), err, context.DeadlineExceeded)
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *AllApiRetryWrapperTest) TestCreateFolder_SuccessOnFirstAttempt() {
	// Arrange
	client := newRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
	req := &controlpb.CreateFolderRequest{Parent: "some/", FolderId: "folder"}
	expectedFolder := &controlpb.Folder{Name: "some/folder"}
	t.mockRawClient.On("CreateFolder", mock.Anything, req, mock.Anything).Return(expectedFolder, nil).Once()

	// Act
	folder, err := client.CreateFolder(t.ctx, req)

	// Assert
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), expectedFolder, folder)
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *AllApiRetryWrapperTest) TestCreateFolder_RetryableErrorThenSuccess() {
	// Arrange
	client := newRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
	req := &controlpb.CreateFolderRequest{Parent: "some/", FolderId: "folder"}
	expectedFolder := &controlpb.Folder{Name: "some/folder"}
	retryableErr := status.Error(codes.Unavailable, "try again")
	// First call fails, second succeeds.
	t.mockRawClient.On("CreateFolder", mock.Anything, req, mock.Anything).Return(nil, retryableErr).Once()
	t.mockRawClient.On("CreateFolder", mock.Anything, req, mock.Anything).Return(expectedFolder, nil).Once()

	// Act
	folder, err := client.CreateFolder(t.ctx, req)

	// Assert
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), expectedFolder, folder)
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *AllApiRetryWrapperTest) TestCreateFolder_NonRetryableError() {
	// Arrange
	client := newRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
	req := &controlpb.CreateFolderRequest{Parent: "some/", FolderId: "folder"}
	nonRetryableErr := status.Error(codes.NotFound, "does not exist")
	t.mockRawClient.On("CreateFolder", mock.Anything, req, mock.Anything).Return(nil, nonRetryableErr).Once()

	// Act
	folder, err := client.CreateFolder(t.ctx, req)

	// Assert
	assert.Error(t.T(), err)
	assert.Nil(t.T(), folder)
	assert.Contains(t.T(), err.Error(), "failed with a non-retryable error")
	assert.Contains(t.T(), err.Error(), nonRetryableErr.Error())
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *AllApiRetryWrapperTest) TestCreateFolder_AllAttemptsTimeOut() {
	// Arrange
	client := newRetryWrapper(t.stallingClient, 1000*time.Microsecond, 10000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
	req := &controlpb.CreateFolderRequest{Parent: "some/", FolderId: "folder"}
	// Set execution time to be longer than the attempt timeout.
	t.stallDurationForFolderAPIs = 6000 * time.Microsecond

	// Act
	_, err := client.CreateFolder(t.ctx, req)

	// Assert: The mock should never be called because every attempt will time out.
	assert.Error(t.T(), err)
	assert.ErrorIs(t.T(), err, context.DeadlineExceeded)
	t.mockRawClient.AssertExpectations(t.T())
}

func (testSuite *StorageLayoutRetryWrapperTest) TestWithRetryOnStorageLayout_WrapsClient() {
	// Arrange
	mockClient := new(MockStorageControlClient)
	clientConfig := storageutil.GetDefaultStorageClientConfig(keyFile)

	// Act
	wrappedClient := withRetryOnStorageLayout(mockClient, &clientConfig)

	// Assert
	require.NotNil(testSuite.T(), wrappedClient)
	retryWrapper, ok := wrappedClient.(*storageControlClientWithRetry)
	require.True(testSuite.T(), ok, "The returned client should be of type *storageControlClientWithRetry")
	assert.Same(testSuite.T(), mockClient, retryWrapper.raw)
	assert.True(testSuite.T(), retryWrapper.enableRetriesOnStorageLayoutAPI, "Retries should be enabled for storage layout APIs")
	assert.False(testSuite.T(), retryWrapper.enableRetriesOnFolderAPIs, "Retries should not be enabled for folder APIs")
}

func (testSuite *StorageLayoutRetryWrapperTest) TestWithRetryOnStorageLayout_UnwrapsNestedRetryClient() {
	// Arrange
	mockClient := new(MockStorageControlClient)
	clientConfig := storageutil.GetDefaultStorageClientConfig(keyFile)
	// Create a client that is already wrapped.
	alreadyWrappedClient := withRetryOnStorageLayout(mockClient, &clientConfig)

	// Act
	// Wrap it again.
	doubleWrappedClient := withRetryOnStorageLayout(alreadyWrappedClient, &clientConfig)

	// Assert
	require.NotNil(testSuite.T(), doubleWrappedClient)
	retryWrapper, ok := doubleWrappedClient.(*storageControlClientWithRetry)
	require.True(testSuite.T(), ok, "The returned client should be of type *storageControlClientWithRetry")
	assert.Same(testSuite.T(), mockClient, retryWrapper.raw, "Should unwrap the nested retry client")
	assert.NotSame(testSuite.T(), alreadyWrappedClient, retryWrapper.raw)
	assert.True(testSuite.T(), retryWrapper.enableRetriesOnStorageLayoutAPI, "Retries should be enabled for storage layout APIs")
	assert.False(testSuite.T(), retryWrapper.enableRetriesOnFolderAPIs, "Retries should not be enabled for folder APIs")
}

func (testSuite *AllApiRetryWrapperTest) TestWithRetryOnAllAPIs_WrapsClient() {
	// Arrange
	mockClient := new(MockStorageControlClient)
	clientConfig := storageutil.GetDefaultStorageClientConfig(keyFile)

	// Act
	wrappedClient := withRetryOnAllAPIs(mockClient, &clientConfig)

	// Assert
	require.NotNil(testSuite.T(), wrappedClient)
	retryWrapper, ok := wrappedClient.(*storageControlClientWithRetry)
	require.True(testSuite.T(), ok, "The returned client should be of type *storageControlClientWithRetry")
	assert.Same(testSuite.T(), mockClient, retryWrapper.raw)
	assert.True(testSuite.T(), retryWrapper.enableRetriesOnStorageLayoutAPI, "Retries should be enabled for storage layout APIs")
	assert.True(testSuite.T(), retryWrapper.enableRetriesOnFolderAPIs, "Retries should be enabled for folder APIs")
}

func (testSuite *AllApiRetryWrapperTest) TestWithRetryOnAllAPIs_UnwrapsNestedRetryClient() {
	// Arrange
	mockClient := new(MockStorageControlClient)
	clientConfig := storageutil.GetDefaultStorageClientConfig(keyFile)
	// Create a client that is already wrapped.
	alreadyWrappedClient := withRetryOnAllAPIs(mockClient, &clientConfig)

	// Act
	// Wrap it again.
	doubleWrappedClient := withRetryOnAllAPIs(alreadyWrappedClient, &clientConfig)

	// Assert
	require.NotNil(testSuite.T(), doubleWrappedClient)
	retryWrapper, ok := doubleWrappedClient.(*storageControlClientWithRetry)
	require.True(testSuite.T(), ok, "The returned client should be of type *storageControlClientWithRetry")
	assert.Same(testSuite.T(), mockClient, retryWrapper.raw, "Should unwrap the nested retry client")
	assert.NotSame(testSuite.T(), alreadyWrappedClient, retryWrapper.raw)
	assert.True(testSuite.T(), retryWrapper.enableRetriesOnStorageLayoutAPI, "Retries should be enabled for storage layout APIs")
	assert.True(testSuite.T(), retryWrapper.enableRetriesOnFolderAPIs, "Retries should be enabled for folder APIs")
}

type ExponentialBackoffTest struct {
	suite.Suite
}

func TestExponentialBackoffTestSuite(t *testing.T) {
	suite.Run(t, new(ExponentialBackoffTest))
}

func (t *ExponentialBackoffTest) TestNewBackoff() {
	initial := 1 * time.Second
	max := 10 * time.Second
	multiplier := 2.0

	b := newExponentialBackoff(&exponentialBackoffConfig{
		initial:    initial,
		max:        max,
		multiplier: multiplier,
	})

	assert.NotNil(t.T(), b)
	assert.Equal(t.T(), initial, b.next)
	assert.Equal(t.T(), initial, b.config.initial)
	assert.Equal(t.T(), max, b.config.max)
	assert.Equal(t.T(), multiplier, b.config.multiplier)
}

func (t *ExponentialBackoffTest) TestNext() {
	initial := 1 * time.Second
	max := 3 * time.Second
	multiplier := 2.0
	b := newExponentialBackoff(&exponentialBackoffConfig{
		initial:    initial,
		max:        max,
		multiplier: multiplier,
	})

	// First call to next() should return initial, and update current.
	assert.Equal(t.T(), 1*time.Second, b.nextDuration())

	// Second call.
	assert.Equal(t.T(), 2*time.Second, b.nextDuration())

	// Third call - capped at max.
	assert.Equal(t.T(), 3*time.Second, b.nextDuration())

	// Should stay capped at max.
	assert.Equal(t.T(), 3*time.Second, b.nextDuration())
}

func (t *ExponentialBackoffTest) TestWaitWithJitter_ContextCancelled() {
	initial := 100 * time.Microsecond // A long duration to ensure cancellation happens first.
	max := 5 * initial
	b := newExponentialBackoff(&exponentialBackoffConfig{
		initial:    initial,
		max:        max,
		multiplier: 2.0,
	})
	ctx, cancel := context.WithCancel(context.Background())
	// Cancel the context immediately.
	cancel()

	start := time.Now()
	err := b.waitWithJitter(ctx)
	elapsed := time.Since(start)

	assert.ErrorIs(t.T(), err, context.Canceled)
	// The function should return almost immediately.
	assert.Less(t.T(), elapsed, initial, "waitWithJitter should return quickly when context is cancelled")
}

func (t *ExponentialBackoffTest) TestWaitWithJitter_NoContextCancelled() {
	initial := time.Millisecond // A short duration to ensure it waits. Making it any shorter can cause random failures
	// because context cancel itself takes about a millisecond.
	max := 5 * initial
	b := newExponentialBackoff(&exponentialBackoffConfig{
		initial:    initial,
		max:        max,
		multiplier: 2.0,
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	start := time.Now()
	err := b.waitWithJitter(ctx)
	elapsed := time.Since(start)

	assert.NoError(t.T(), err)
	// The function should wait for a duration close to initial.
	assert.LessOrEqual(t.T(), elapsed, initial*2, "waitWithJitter should not wait excessively long")
}

type ControlClientGaxRetryWrapperTest struct {
	suite.Suite
}

func TestControlClientGaxRetryWrapperTestSuite(t *testing.T) {
	suite.Run(t, new(ControlClientGaxRetryWrapperTest))
}

func (testSuite *ControlClientGaxRetryWrapperTest) TestStorageControlClientGaxRetryOptions() {
	// Arrange
	clientConfig := storageutil.GetDefaultStorageClientConfig(keyFile)

	// Act
	gaxOpts := storageControlClientGaxRetryOptions(&clientConfig)

	// Assert
	require.NotEmpty(testSuite.T(), gaxOpts)
	require.Len(testSuite.T(), gaxOpts, 2)
}

func (testSuite *ControlClientGaxRetryWrapperTest) TestAddGaxRetriesForFolderAPIs_NilRawControlClient() {
	// Arrange
	clientConfig := storageutil.GetDefaultStorageClientConfig(keyFile)

	// Act
	err := addGaxRetriesForFolderAPIs(nil, &clientConfig)

	// Assert
	require.Error(testSuite.T(), err)
}

func (testSuite *ControlClientGaxRetryWrapperTest) TestAddGaxRetriesForFolderAPIs_NilClientConfig() {
	// Arrange
	rawClient := &control.StorageControlClient{}

	// Act
	err := addGaxRetriesForFolderAPIs(rawClient, nil)

	// Assert
	require.Error(testSuite.T(), err)
}

func (testSuite *ControlClientGaxRetryWrapperTest) TestAddGaxRetriesForFolderAPIs_AppliesGaxOptions() {
	// Arrange
	rawControlClient := &control.StorageControlClient{CallOptions: &control.StorageControlCallOptions{}}
	clientConfig := storageutil.GetDefaultStorageClientConfig(keyFile)

	// Act
	err := addGaxRetriesForFolderAPIs(rawControlClient, &clientConfig)

	// Assert
	require.NoError(testSuite.T(), err)
	require.NotNil(testSuite.T(), rawControlClient.CallOptions)
	assert.Empty(testSuite.T(), rawControlClient.CallOptions.GetStorageLayout) // GetStorageLayout should not have GAX retries applied
	assert.Len(testSuite.T(), rawControlClient.CallOptions.DeleteFolder, 2)    // DeleteFolder should have GAX retries applied
	assert.Len(testSuite.T(), rawControlClient.CallOptions.GetFolder, 2)       // GetFolder should have GAX retries applied
	assert.Len(testSuite.T(), rawControlClient.CallOptions.CreateFolder, 2)    // CreateFolder should have GAX retries applied
	assert.Len(testSuite.T(), rawControlClient.CallOptions.RenameFolder, 2)    // RenameFolder should have GAX retries applied
}
