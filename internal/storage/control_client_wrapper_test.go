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
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	control "cloud.google.com/go/storage/control/apiv2"
	"github.com/googleapis/gax-go/v2"
)

// stallingStorageControlClient is a wrapper that introduces a controllable delay
// to every call, to simulate network latency for testing timeout-based retries.
type stallingStorageControlClient struct {
	wrapped                      StorageControlClient
	stallTimeForGetStorageLayout *time.Duration
	stallTimeForFolderAPIs       *time.Duration
}

func (s *stallingStorageControlClient) GetStorageLayout(ctx context.Context, req *controlpb.GetStorageLayoutRequest, opts ...gax.CallOption) (*controlpb.StorageLayout, error) {
	if s.stallTimeForGetStorageLayout != nil {
		select {
		case <-time.After(*s.stallTimeForGetStorageLayout):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return s.wrapped.GetStorageLayout(ctx, req, opts...)
}

func (s *stallingStorageControlClient) DeleteFolder(ctx context.Context, req *controlpb.DeleteFolderRequest, opts ...gax.CallOption) error {
	if s.stallTimeForFolderAPIs != nil {
		select {
		case <-time.After(*s.stallTimeForFolderAPIs):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return s.wrapped.DeleteFolder(ctx, req, opts...)
}

func (s *stallingStorageControlClient) GetFolder(ctx context.Context, req *controlpb.GetFolderRequest, opts ...gax.CallOption) (*controlpb.Folder, error) {
	if s.stallTimeForFolderAPIs != nil {
		select {
		case <-time.After(*s.stallTimeForFolderAPIs):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return s.wrapped.GetFolder(ctx, req, opts...)
}

func (s *stallingStorageControlClient) RenameFolder(ctx context.Context, req *controlpb.RenameFolderRequest, opts ...gax.CallOption) (*control.RenameFolderOperation, error) {
	if s.stallTimeForFolderAPIs != nil {
		select {
		case <-time.After(*s.stallTimeForFolderAPIs):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return s.wrapped.RenameFolder(ctx, req, opts...)
}

func (s *stallingStorageControlClient) CreateFolder(ctx context.Context, req *controlpb.CreateFolderRequest, opts ...gax.CallOption) (*controlpb.Folder, error) {
	if s.stallTimeForFolderAPIs != nil {
		select {
		case <-time.After(*s.stallTimeForFolderAPIs):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return s.wrapped.CreateFolder(ctx, req, opts...)
}

type ControlClientStallRetryWrapperTest struct {
	suite.Suite
	// The raw mock client for setting expectations on return values.
	mockRawClient  *MockStorageControlClient
	stallingClient *stallingStorageControlClient
	ctx            context.Context
	// The simulated execution time for each GetStorageLayout call made through stallingClient.
	stallTimeForGetStorageLayout time.Duration
}

type StorageLayoutStallRetryWrapperTest struct {
	ControlClientStallRetryWrapperTest
}

type AllApiStallRetryWrapperTest struct {
	ControlClientStallRetryWrapperTest
	// The execution time for each folder API call made through stallingClient. Can be adjusted
	// per test.
	stallTimeForFolderAPIs time.Duration
}

func TestControlClientWrapperTestSuite(t *testing.T) {
	t.Run("StorageLayoutStallRetryWrapperTest", func(t *testing.T) {
		suite.Run(t, new(StorageLayoutStallRetryWrapperTest))
	})
	t.Run("AllApiStallRetryWrapperTest", func(t *testing.T) {
		suite.Run(t, new(AllApiStallRetryWrapperTest))
	})
}

func (t *ControlClientStallRetryWrapperTest) SetupSuite() {
	t.mockRawClient = new(MockStorageControlClient)
	t.ctx = context.Background()
}

func (t *StorageLayoutStallRetryWrapperTest) SetupSuite() {
	t.ControlClientStallRetryWrapperTest.SetupSuite()
	t.stallingClient = &stallingStorageControlClient{
		wrapped:                      t.mockRawClient,
		stallTimeForGetStorageLayout: &t.stallTimeForGetStorageLayout,
	}
}

func (t *AllApiStallRetryWrapperTest) SetupSuite() {
	t.ControlClientStallRetryWrapperTest.SetupSuite()
	t.stallingClient = &stallingStorageControlClient{
		wrapped:                      t.mockRawClient,
		stallTimeForGetStorageLayout: &t.stallTimeForGetStorageLayout,
		stallTimeForFolderAPIs:       &t.stallTimeForFolderAPIs,
	}
}

func (t *StorageLayoutStallRetryWrapperTest) TestGetStorageLayout_SuccessOnFirstAttempt() {
	// Arrange
	client := withRetryOnStorageLayoutStall(t.stallingClient, 100*time.Microsecond, 500*time.Microsecond, 2, 1000*time.Microsecond)
	req := &controlpb.GetStorageLayoutRequest{Name: "some/bucket"}
	expectedLayout := &controlpb.StorageLayout{Location: "some-location"}
	t.stallTimeForGetStorageLayout = 0 // No stall.
	t.mockRawClient.On("GetStorageLayout", mock.Anything, req, mock.Anything).Return(expectedLayout, nil).Once()

	// Act
	layout, err := client.GetStorageLayout(t.ctx, req)

	// Assert
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), expectedLayout, layout)
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *StorageLayoutStallRetryWrapperTest) TestGetStorageLayout_RetryableErrorThenSuccess() {
	// Arrange
	client := withRetryOnStorageLayoutStall(t.stallingClient, 100*time.Microsecond, 500*time.Microsecond, 2, 1000*time.Microsecond)
	req := &controlpb.GetStorageLayoutRequest{Name: "some/bucket"}
	expectedLayout := &controlpb.StorageLayout{Location: "some-location"}
	retryableErr := status.Error(codes.Unavailable, "try again")
	t.stallTimeForGetStorageLayout = 0 // No stall.

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

func (t *StorageLayoutStallRetryWrapperTest) TestGetStorageLayout_NonRetryableError() {
	// Arrange
	client := withRetryOnStorageLayoutStall(t.stallingClient, 100*time.Microsecond, 500*time.Microsecond, 2, 1000*time.Microsecond)
	req := &controlpb.GetStorageLayoutRequest{Name: "some/bucket"}
	nonRetryableErr := status.Error(codes.NotFound, "does not exist")
	t.stallTimeForGetStorageLayout = 0 // No stall.
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

func (t *StorageLayoutStallRetryWrapperTest) TestGetStorageLayout_AttemptTimesOutAndThenSucceeds() {
	// Arrange
	// minRetryDeadline is 100us, next is 200us.
	client := withRetryOnStorageLayoutStall(t.stallingClient, 100*time.Microsecond, 500*time.Microsecond, 2, 1000*time.Microsecond)
	req := &controlpb.GetStorageLayoutRequest{Name: "some/bucket"}
	expectedLayout := &controlpb.StorageLayout{Location: "some-location"}

	// Set stall time to be longer than the first attempt's timeout (100us)
	// but shorter than the second attempt's timeout (200us).
	t.stallTimeForGetStorageLayout = 150 * time.Microsecond

	// The mock should only be called on the second attempt, which succeeds.
	t.mockRawClient.On("GetStorageLayout", mock.Anything, req, mock.Anything).Return(expectedLayout, nil).Once()

	// Act
	layout, err := client.GetStorageLayout(t.ctx, req)

	// Assert
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), expectedLayout, layout)
	t.mockRawClient.AssertExpectations(t.T())
	t.mockRawClient.AssertNumberOfCalls(t.T(), "GetStorageLayout", 1)
}

func (t *StorageLayoutStallRetryWrapperTest) TestGetStorageLayout_AllAttemptsTimeOut() {
	// Arrange
	// maxRetryDeadline is 5ms. Total budget is 10ms.
	client := withRetryOnStorageLayoutStall(t.stallingClient, 1000*time.Microsecond, 5000*time.Microsecond, 2, 10000*time.Microsecond)
	req := &controlpb.GetStorageLayoutRequest{Name: "some/bucket"}
	// Set stall time to be longer than the max attempt timeout.
	t.stallTimeForGetStorageLayout = 6000 * time.Microsecond

	// Act
	_, err := client.GetStorageLayout(t.ctx, req)

	// The mock should never be called because every attempt will time out.
	assert.ErrorIs(t.T(), err, context.DeadlineExceeded)
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *StorageLayoutStallRetryWrapperTest) TestGetFolder_IsNotRetried() {
	// Arrange
	client := withRetryOnStorageLayoutStall(t.stallingClient, 100*time.Microsecond, 500*time.Microsecond, 2, 1000*time.Microsecond)
	req := &controlpb.GetFolderRequest{Name: "some/folder"}
	retryableErr := status.Error(codes.Unavailable, "try again")
	t.stallTimeForGetStorageLayout = 0 // No stall for this test.

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

func (t *StorageLayoutStallRetryWrapperTest) TestDeleteFolder_IsNotRetried() {
	// Arrange
	client := withRetryOnStorageLayoutStall(t.stallingClient, 100*time.Microsecond, 500*time.Microsecond, 2, 1000*time.Microsecond)
	req := &controlpb.DeleteFolderRequest{Name: "some/folder"}
	retryableErr := status.Error(codes.Unavailable, "try again")
	t.stallTimeForGetStorageLayout = 0 // No stall for this test.

	// Mock the raw client to return a retryable error once.
	t.mockRawClient.On("DeleteFolder", mock.Anything, req, mock.Anything).Return(retryableErr).Once()

	// Act
	err := client.DeleteFolder(t.ctx, req)

	// Assert
	assert.Error(t.T(), err)
	assert.Equal(t.T(), retryableErr, err)
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *StorageLayoutStallRetryWrapperTest) TestCreateFolder_IsNotRetried() {
	// Arrange
	client := withRetryOnStorageLayoutStall(t.stallingClient, 100*time.Microsecond, 500*time.Microsecond, 2, 1000*time.Microsecond)
	req := &controlpb.CreateFolderRequest{Parent: "some/", FolderId: "folder"}
	retryableErr := status.Error(codes.Unavailable, "try again")
	t.stallTimeForGetStorageLayout = 0 // No stall for this test.

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

func (t *StorageLayoutStallRetryWrapperTest) TestRenameFolder_IsNotRetried() {
	// Arrange
	client := withRetryOnStorageLayoutStall(t.stallingClient, 100*time.Microsecond, 500*time.Microsecond, 2, 1000*time.Microsecond)
	req := &controlpb.RenameFolderRequest{Name: "some/folder", DestinationFolderId: "new/folder"}
	retryableErr := status.Error(codes.Unavailable, "try again")
	t.stallTimeForGetStorageLayout = 0 // No stall for this test.

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

func (t *AllApiStallRetryWrapperTest) TestGetStorageLayout_SuccessOnFirstAttempt() {
	// Arrange
	client := withRetryOnStall(t.stallingClient, 100*time.Microsecond, 500*time.Microsecond, 2, 1000*time.Microsecond)
	req := &controlpb.GetStorageLayoutRequest{Name: "some/bucket"}
	expectedLayout := &controlpb.StorageLayout{Location: "some-location"}
	t.stallTimeForGetStorageLayout = 0 // No stall.
	t.mockRawClient.On("GetStorageLayout", mock.Anything, req, mock.Anything).Return(expectedLayout, nil).Once()

	// Act
	layout, err := client.GetStorageLayout(t.ctx, req)

	// Assert
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), expectedLayout, layout)
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *AllApiStallRetryWrapperTest) TestGetStorageLayout_RetryableErrorThenSuccess() {
	// Arrange
	client := withRetryOnStall(t.stallingClient, 100*time.Microsecond, 500*time.Microsecond, 2, 1000*time.Microsecond)
	req := &controlpb.GetStorageLayoutRequest{Name: "some/bucket"}
	expectedLayout := &controlpb.StorageLayout{Location: "some-location"}
	retryableErr := status.Error(codes.Unavailable, "try again")
	t.stallTimeForGetStorageLayout = 0 // No stall.

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

func (t *AllApiStallRetryWrapperTest) TestGetStorageLayout_NonRetryableError() {
	// Arrange
	client := withRetryOnStall(t.stallingClient, 100*time.Microsecond, 500*time.Microsecond, 2, 1000*time.Microsecond)
	req := &controlpb.GetStorageLayoutRequest{Name: "some/bucket"}
	nonRetryableErr := status.Error(codes.NotFound, "does not exist")
	t.stallTimeForGetStorageLayout = 0 // No stall.
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

func (t *AllApiStallRetryWrapperTest) TestGetStorageLayout_AttemptTimesOutAndThenSucceeds() {
	// Arrange
	// minRetryDeadline is 100us, next is 200us.
	client := withRetryOnStall(t.stallingClient, 100*time.Microsecond, 500*time.Microsecond, 2, 1000*time.Microsecond)
	req := &controlpb.GetStorageLayoutRequest{Name: "some/bucket"}
	expectedLayout := &controlpb.StorageLayout{Location: "some-location"}

	// Set stall time to be longer than the first attempt's timeout (100us)
	// but shorter than the second attempt's timeout (200us).
	t.stallTimeForGetStorageLayout = 150 * time.Microsecond

	// The mock should only be called on the second attempt, which succeeds.
	t.mockRawClient.On("GetStorageLayout", mock.Anything, req, mock.Anything).Return(expectedLayout, nil).Once()

	// Act
	layout, err := client.GetStorageLayout(t.ctx, req)

	// Assert
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), expectedLayout, layout)
	t.mockRawClient.AssertExpectations(t.T())
	t.mockRawClient.AssertNumberOfCalls(t.T(), "GetStorageLayout", 1)
}

func (t *AllApiStallRetryWrapperTest) TestGetStorageLayout_AllAttemptsTimeOut() {
	// Arrange
	// maxRetryDeadline is 5ms. Total budget is 10ms.
	client := withRetryOnStall(t.stallingClient, 1000*time.Microsecond, 5000*time.Microsecond, 2, 10000*time.Microsecond)
	req := &controlpb.GetStorageLayoutRequest{Name: "some/bucket"}
	// Set stall time to be longer than the max attempt timeout.
	t.stallTimeForGetStorageLayout = 6000 * time.Microsecond

	// Act
	_, err := client.GetStorageLayout(t.ctx, req)

	// The mock should never be called because every attempt will time out.
	assert.ErrorIs(t.T(), err, context.DeadlineExceeded)
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *AllApiStallRetryWrapperTest) TestDeleteFolder_SuccessOnFirstAttempt() {
	// Arrange
	client := withRetryOnStall(t.stallingClient, 100*time.Microsecond, 500*time.Microsecond, 2, 1000*time.Microsecond)
	req := &controlpb.DeleteFolderRequest{Name: "some/folder"}
	t.mockRawClient.On("DeleteFolder", mock.Anything, req, mock.Anything).Return(nil).Once()
	t.stallTimeForFolderAPIs = 0 // No stall.

	// Act
	err := client.DeleteFolder(t.ctx, req)

	// Assert
	assert.NoError(t.T(), err)
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *AllApiStallRetryWrapperTest) TestDeleteFolder_RetryableErrorThenSuccess() {
	// Arrange
	client := withRetryOnStall(t.stallingClient, 100*time.Microsecond, 500*time.Microsecond, 2, 1000*time.Microsecond)
	req := &controlpb.DeleteFolderRequest{Name: "some/folder"}
	retryableErr := status.Error(codes.Unavailable, "try again")
	t.stallTimeForFolderAPIs = 0 // No stall

	// First call fails, second succeeds.
	t.mockRawClient.On("DeleteFolder", mock.Anything, req, mock.Anything).Return(retryableErr).Once()
	t.mockRawClient.On("DeleteFolder", mock.Anything, req, mock.Anything).Return(nil).Once()

	// Act
	err := client.DeleteFolder(t.ctx, req)

	// Assert
	assert.NoError(t.T(), err)
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *AllApiStallRetryWrapperTest) TestDeleteFolder_NonRetryableError() {
	// Arrange
	client := withRetryOnStall(t.stallingClient, 100*time.Microsecond, 500*time.Microsecond, 2, 1000*time.Microsecond)
	req := &controlpb.DeleteFolderRequest{Name: "some/folder"}
	nonRetryableErr := status.Error(codes.NotFound, "does not exist")
	t.stallTimeForFolderAPIs = 0 // No stall
	t.mockRawClient.On("DeleteFolder", mock.Anything, req, mock.Anything).Return(nonRetryableErr).Once()

	// Act
	err := client.DeleteFolder(t.ctx, req)

	// Assert
	assert.Error(t.T(), err)
	assert.Contains(t.T(), err.Error(), "failed with a non-retryable error")
	assert.Contains(t.T(), err.Error(), nonRetryableErr.Error())
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *AllApiStallRetryWrapperTest) TestDeleteFolder_AttemptTimesOutAndThenSucceeds() {
	// Arrange
	// minRetryDeadline is 100us, next is 200us.
	client := withRetryOnStall(t.stallingClient, 100*time.Microsecond, 500*time.Microsecond, 2, 1000*time.Microsecond)
	req := &controlpb.DeleteFolderRequest{Name: "some/folder"}

	// Set stall time to be longer than the first attempt's timeout (100us)
	// but shorter than the second attempt's timeout (200us).
	t.stallTimeForFolderAPIs = 150 * time.Microsecond

	// The mock should only be called on the second attempt, which succeeds.
	t.mockRawClient.On("DeleteFolder", mock.Anything, req, mock.Anything).Return(nil).Once()

	// Act
	err := client.DeleteFolder(t.ctx, req)

	// Assert
	assert.NoError(t.T(), err)
	t.mockRawClient.AssertExpectations(t.T())
	t.mockRawClient.AssertNumberOfCalls(t.T(), "DeleteFolder", 1)
}

func (t *AllApiStallRetryWrapperTest) TestDeleteFolder_AllAttemptsTimeOut() {
	// Arrange
	// maxRetryDeadline is 5ms. Total budget is 10ms.
	client := withRetryOnStall(t.stallingClient, 1000*time.Microsecond, 5000*time.Microsecond, 2, 10000*time.Microsecond)
	req := &controlpb.DeleteFolderRequest{Name: "some/folder"}
	// Set stall time to be longer than the max attempt timeout.
	t.stallTimeForFolderAPIs = 6000 * time.Microsecond

	// Act
	err := client.DeleteFolder(t.ctx, req)

	// The mock should never be called because every attempt will time out.
	assert.Error(t.T(), err)
	assert.ErrorIs(t.T(), err, context.DeadlineExceeded)
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *AllApiStallRetryWrapperTest) TestNewRetryWrapper_ParameterSanitization() {
	// Arrange
	var zeroDuration time.Duration
	var negativeMultiplier = -1.0

	// Act
	client := withRetryOnStall(t.stallingClient, zeroDuration, zeroDuration, negativeMultiplier, zeroDuration)
	wrapper, ok := client.(*storageControlClientWithRetryOnStall)
	assert.True(t.T(), ok)

	// Assert
	assert.Equal(t.T(), defaultControlClientMinRetryDeadline, wrapper.minRetryDeadline)
	assert.Equal(t.T(), defaultControlClientMaxRetryDeadline, wrapper.maxRetryDeadline)
	assert.Equal(t.T(), defaultControlClientRetryMultiplier, wrapper.retryMultiplier)
	assert.Equal(t.T(), defaultControlClientTotalRetryBudget, wrapper.totalRetryBudget)
}

func (t *AllApiStallRetryWrapperTest) TestGetFolder_SuccessOnFirstAttempt() {
	// Arrange
	client := withRetryOnStall(t.stallingClient, 100*time.Microsecond, 500*time.Microsecond, 2, 1000*time.Microsecond)
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

func (t *AllApiStallRetryWrapperTest) TestGetFolder_RetryableErrorThenSuccess() {
	// Arrange
	client := withRetryOnStall(t.stallingClient, 100*time.Microsecond, 500*time.Microsecond, 2, 1000*time.Microsecond)
	req := &controlpb.GetFolderRequest{Name: "some/folder"}
	expectedFolder := &controlpb.Folder{Name: "some/folder"}
	retryableErr := status.Error(codes.Unavailable, "try again")
	t.stallTimeForFolderAPIs = 0 // No stall

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

func (t *AllApiStallRetryWrapperTest) TestGetFolder_NonRetryableError() {
	// Arrange
	client := withRetryOnStall(t.stallingClient, 100*time.Microsecond, 500*time.Microsecond, 2, 1000*time.Microsecond)
	req := &controlpb.GetFolderRequest{Name: "some/folder"}
	nonRetryableErr := status.Error(codes.NotFound, "does not exist")
	t.stallTimeForFolderAPIs = 0 // No stall
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

func (t *AllApiStallRetryWrapperTest) TestGetFolder_AttemptTimesOutAndThenSucceeds() {
	// Arrange
	// minRetryDeadline is 100us, next is 200us.
	client := withRetryOnStall(t.stallingClient, 100*time.Microsecond, 500*time.Microsecond, 2, 1000*time.Microsecond)
	req := &controlpb.GetFolderRequest{Name: "some/folder"}
	expectedFolder := &controlpb.Folder{Name: "some/folder"}

	// Set stall time to be longer than the first attempt's timeout (100us)
	// but shorter than the second attempt's timeout (200us).
	t.stallTimeForFolderAPIs = 150 * time.Microsecond

	// The mock should only be called on the second attempt, which succeeds.
	t.mockRawClient.On("GetFolder", mock.Anything, req, mock.Anything).Return(expectedFolder, nil).Once()

	// Act
	folder, err := client.GetFolder(t.ctx, req)

	// Assert
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), expectedFolder, folder)
	t.mockRawClient.AssertExpectations(t.T())
	t.mockRawClient.AssertNumberOfCalls(t.T(), "GetFolder", 1)
}

func (t *AllApiStallRetryWrapperTest) TestGetFolder_AllAttemptsTimeOut() {
	// Arrange
	// maxRetryDeadline is 5ms. Total budget is 10ms.
	client := withRetryOnStall(t.stallingClient, 1000*time.Microsecond, 5000*time.Microsecond, 2, 10000*time.Microsecond)
	req := &controlpb.GetFolderRequest{Name: "some/folder"}
	// Set execution time to be longer than the max attempt timeout.
	t.stallTimeForFolderAPIs = 6000 * time.Microsecond

	// Act
	_, err := client.GetFolder(t.ctx, req)

	// Assert: The mock should never be called because every attempt will time out.
	assert.Error(t.T(), err)
	assert.ErrorIs(t.T(), err, context.DeadlineExceeded)
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *AllApiStallRetryWrapperTest) TestRenameFolder_SuccessOnFirstAttempt() {
	// Arrange
	client := withRetryOnStall(t.stallingClient, 100*time.Microsecond, 500*time.Microsecond, 2, 1000*time.Microsecond)
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

func (t *AllApiStallRetryWrapperTest) TestRenameFolder_RetryableErrorThenSuccess() {
	// Arrange
	client := withRetryOnStall(t.stallingClient, 100*time.Microsecond, 500*time.Microsecond, 2, 1000*time.Microsecond)
	req := &controlpb.RenameFolderRequest{Name: "some/folder", DestinationFolderId: "new/folder"}
	expectedOp := &control.RenameFolderOperation{}
	retryableErr := status.Error(codes.Unavailable, "try again")
	t.stallTimeForFolderAPIs = 0 // No stall

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

func (t *AllApiStallRetryWrapperTest) TestRenameFolder_NonRetryableError() {
	// Arrange
	client := withRetryOnStall(t.stallingClient, 100*time.Microsecond, 500*time.Microsecond, 2, 1000*time.Microsecond)
	req := &controlpb.RenameFolderRequest{Name: "some/folder", DestinationFolderId: "new/folder"}
	nonRetryableErr := status.Error(codes.NotFound, "does not exist")
	t.stallTimeForFolderAPIs = 0 // No stall
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

func (t *AllApiStallRetryWrapperTest) TestCreateFolder_SuccessOnFirstAttempt() {
	// Arrange
	client := withRetryOnStall(t.stallingClient, 100*time.Microsecond, 500*time.Microsecond, 2, 1000*time.Microsecond)
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

func (t *AllApiStallRetryWrapperTest) TestCreateFolder_RetryableErrorThenSuccess() {
	// Arrange
	client := withRetryOnStall(t.stallingClient, 100*time.Microsecond, 500*time.Microsecond, 2, 1000*time.Microsecond)
	req := &controlpb.CreateFolderRequest{Parent: "some/", FolderId: "folder"}
	expectedFolder := &controlpb.Folder{Name: "some/folder"}
	retryableErr := status.Error(codes.Unavailable, "try again")
	t.stallTimeForFolderAPIs = 0 // No stall

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

func (t *AllApiStallRetryWrapperTest) TestCreateFolder_NonRetryableError() {
	// Arrange
	client := withRetryOnStall(t.stallingClient, 100*time.Microsecond, 500*time.Microsecond, 2, 1000*time.Microsecond)
	req := &controlpb.CreateFolderRequest{Parent: "some/", FolderId: "folder"}
	nonRetryableErr := status.Error(codes.NotFound, "does not exist")
	t.stallTimeForFolderAPIs = 0 // No stall
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

func (t *StorageLayoutStallRetryWrapperTest) Test_WithRetryOnStorageLayoutStallParameterSanitization() {
	// Arrange
	var zeroDuration time.Duration
	var negativeMultiplier = -1.0

	// Act
	client := withRetryOnStorageLayoutStall(t.stallingClient, zeroDuration, zeroDuration, negativeMultiplier, zeroDuration)
	wrapper, ok := client.(*storageControlClientWithRetryOnStall)
	assert.True(t.T(), ok)

	// Assert
	assert.Equal(t.T(), defaultControlClientMinRetryDeadline, wrapper.minRetryDeadline)
	assert.Equal(t.T(), defaultControlClientMaxRetryDeadline, wrapper.maxRetryDeadline)
	assert.Equal(t.T(), defaultControlClientRetryMultiplier, wrapper.retryMultiplier)
	assert.Equal(t.T(), defaultControlClientTotalRetryBudget, wrapper.totalRetryBudget)
}

func (t *AllApiStallRetryWrapperTest) Test_WithRetryOnAllApiStallParameterSanitization() {
	// Arrange
	var zeroDuration time.Duration
	var negativeMultiplier = -1.0

	// Act
	client := withRetryOnStall(t.stallingClient, zeroDuration, zeroDuration, negativeMultiplier, zeroDuration)
	wrapper, ok := client.(*storageControlClientWithRetryOnStall)
	assert.True(t.T(), ok)

	// Assert
	assert.Equal(t.T(), defaultControlClientMinRetryDeadline, wrapper.minRetryDeadline)
	assert.Equal(t.T(), defaultControlClientMaxRetryDeadline, wrapper.maxRetryDeadline)
	assert.Equal(t.T(), defaultControlClientRetryMultiplier, wrapper.retryMultiplier)
	assert.Equal(t.T(), defaultControlClientTotalRetryBudget, wrapper.totalRetryBudget)
}
