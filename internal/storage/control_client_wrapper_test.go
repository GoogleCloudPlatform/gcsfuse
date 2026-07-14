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
	"google.golang.org/grpc/metadata"
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
	client := t.newHelperRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, false)
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
	client := t.newHelperRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, false)
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
	client := t.newHelperRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, false)
	req := &controlpb.GetStorageLayoutRequest{Name: "some/bucket"}
	nonRetryableErr := status.Error(codes.NotFound, "does not exist")
	t.mockRawClient.On("GetStorageLayout", mock.Anything, req, mock.Anything).Return(nil, nonRetryableErr).Once()

	// Act
	layout, err := client.GetStorageLayout(t.ctx, req)

	// Assert
	assert.Error(t.T(), err)
	assert.Nil(t.T(), layout)
	assert.Contains(t.T(), err.Error(), "failed:")
	assert.Contains(t.T(), err.Error(), nonRetryableErr.Error())
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *StorageLayoutRetryWrapperTest) TestGetStorageLayout_AllAttemptsTimeOut() {
	// Arrange
	// This test requires different retry parameters, so we create a new client.
	client := t.newHelperRetryWrapper(t.stallingClient, 1000*time.Microsecond, 10000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, false)
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
	client := t.newHelperRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, false)
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
	client := t.newHelperRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, false)
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
	client := t.newHelperRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, false)
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
	client := t.newHelperRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, false)
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

func (t *ControlClientRetryWrapperTest) newHelperRetryWrapper(controlClient StorageControlClient, retryDeadline, totalRetryBudget, initialBackoff, maxRetrySleep time.Duration, backoffMultiplier float64, retryFolderAPIs bool) StorageControlClient {
	t.T().Helper()
	clientConfig := &storageutil.StorageClientConfig{
		MaxRetrySleep:   maxRetrySleep,
		RetryMultiplier: backoffMultiplier,
	}
	scc := newStorageControlClientWithRetry(controlClient, clientConfig)
	if retryFolderAPIs {
		scc = scc.WithRetriesOnFolderAPI()
	}

	scc.retryConfig = storageutil.NewRetryConfigForTesting(
		retryDeadline,
		totalRetryBudget,
		initialBackoff,
		maxRetrySleep,
		backoffMultiplier,
		clientConfig.MaxRetryAttempts,
	)
	return scc
}

func (t *AllApiRetryWrapperTest) TestGetStorageLayout_SuccessOnFirstAttempt() {
	// Arrange
	client := t.newHelperRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
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
	client := t.newHelperRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
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
	client := t.newHelperRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
	req := &controlpb.GetStorageLayoutRequest{Name: "some/bucket"}
	nonRetryableErr := status.Error(codes.NotFound, "does not exist")
	t.mockRawClient.On("GetStorageLayout", mock.Anything, req, mock.Anything).Return(nil, nonRetryableErr).Once()

	// Act
	layout, err := client.GetStorageLayout(t.ctx, req)

	// Assert
	assert.Error(t.T(), err)
	assert.Nil(t.T(), layout)
	assert.Contains(t.T(), err.Error(), "failed:")
	assert.Contains(t.T(), err.Error(), nonRetryableErr.Error())
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *AllApiRetryWrapperTest) TestGetStorageLayout_AllAttemptsTimeOut() {
	// Arrange
	client := t.newHelperRetryWrapper(t.stallingClient, 1000*time.Microsecond, 10000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
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
	client := t.newHelperRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
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
	client := t.newHelperRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
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
	client := t.newHelperRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
	req := &controlpb.DeleteFolderRequest{Name: "some/folder"}
	nonRetryableErr := status.Error(codes.NotFound, "does not exist")
	t.mockRawClient.On("DeleteFolder", mock.Anything, req, mock.Anything).Return(nonRetryableErr).Once()

	// Act
	err := client.DeleteFolder(t.ctx, req)

	// Assert
	assert.Error(t.T(), err)
	assert.Contains(t.T(), err.Error(), "failed:")
	assert.Contains(t.T(), err.Error(), nonRetryableErr.Error())
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *AllApiRetryWrapperTest) TestDeleteFolder_AllAttemptsTimeOut() {
	// Arrange
	client := t.newHelperRetryWrapper(t.stallingClient, 1000*time.Microsecond, 10000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
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
	client := t.newHelperRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
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
	client := t.newHelperRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
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
	client := t.newHelperRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
	req := &controlpb.GetFolderRequest{Name: "some/folder"}
	nonRetryableErr := status.Error(codes.NotFound, "does not exist")
	t.mockRawClient.On("GetFolder", mock.Anything, req, mock.Anything).Return(nil, nonRetryableErr).Once()

	// Act
	folder, err := client.GetFolder(t.ctx, req)

	// Assert
	assert.Error(t.T(), err)
	assert.Nil(t.T(), folder)
	assert.Contains(t.T(), err.Error(), "failed:")
	assert.Contains(t.T(), err.Error(), nonRetryableErr.Error())
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *AllApiRetryWrapperTest) TestGetFolder_AllAttemptsTimeOut() {
	// Arrange
	client := t.newHelperRetryWrapper(t.stallingClient, 1000*time.Microsecond, 10000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
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
	client := t.newHelperRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
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
	client := t.newHelperRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
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
	client := t.newHelperRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
	req := &controlpb.RenameFolderRequest{Name: "some/folder", DestinationFolderId: "new/folder"}
	nonRetryableErr := status.Error(codes.NotFound, "does not exist")
	t.mockRawClient.On("RenameFolder", mock.Anything, req, mock.Anything).Return(nil, nonRetryableErr).Once()

	// Act
	op, err := client.RenameFolder(t.ctx, req)

	// Assert
	assert.Error(t.T(), err)
	assert.Nil(t.T(), op)
	assert.Contains(t.T(), err.Error(), "failed:")
	assert.Contains(t.T(), err.Error(), nonRetryableErr.Error())
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *AllApiRetryWrapperTest) TestRenameFolder_AllAttemptsTimeOut() {
	// Arrange
	client := t.newHelperRetryWrapper(t.stallingClient, 1000*time.Microsecond, 10000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
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
	client := t.newHelperRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
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
	client := t.newHelperRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
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
	client := t.newHelperRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
	req := &controlpb.CreateFolderRequest{Parent: "some/", FolderId: "folder"}
	nonRetryableErr := status.Error(codes.NotFound, "does not exist")
	t.mockRawClient.On("CreateFolder", mock.Anything, req, mock.Anything).Return(nil, nonRetryableErr).Once()

	// Act
	folder, err := client.CreateFolder(t.ctx, req)

	// Assert
	assert.Error(t.T(), err)
	assert.Nil(t.T(), folder)
	assert.Contains(t.T(), err.Error(), "failed:")
	assert.Contains(t.T(), err.Error(), nonRetryableErr.Error())
	t.mockRawClient.AssertExpectations(t.T())
}

func (t *AllApiRetryWrapperTest) TestCreateFolder_AllAttemptsTimeOut() {
	// Arrange
	client := t.newHelperRetryWrapper(t.stallingClient, 1000*time.Microsecond, 10000*time.Microsecond, time.Microsecond, 10*time.Microsecond, 2, true)
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

func (testSuite *StorageLayoutRetryWrapperTest) TestWithRetry_StorageLayout_WrapsClient() {
	// Arrange
	mockClient := new(MockStorageControlClient)
	clientConfig := storageutil.GetDefaultStorageClientConfig(keyFile)

	wrappedClient := newStorageControlClientWithRetry(mockClient, &clientConfig)

	// Assert
	require.NotNil(testSuite.T(), wrappedClient)
	retryWrapper := wrappedClient
	assert.Same(testSuite.T(), mockClient, retryWrapper.raw)
	assert.False(testSuite.T(), retryWrapper.enableRetriesOnFolderAPIs, "Retries should not be enabled for folder APIs")
}

func (testSuite *StorageLayoutRetryWrapperTest) TestWithRetry_StorageLayout_UnwrapsNestedRetryClient() {
	// Arrange
	mockClient := new(MockStorageControlClient)
	clientConfig := storageutil.GetDefaultStorageClientConfig(keyFile)
	alreadyWrappedClient := newStorageControlClientWithRetry(mockClient, &clientConfig)

	// Wrap it again.
	doubleWrappedClient := newStorageControlClientWithRetry(alreadyWrappedClient, &clientConfig)

	// Assert
	require.NotNil(testSuite.T(), doubleWrappedClient)
	retryWrapper := doubleWrappedClient
	assert.Same(testSuite.T(), mockClient, retryWrapper.raw, "Should unwrap the nested retry client")
	assert.NotSame(testSuite.T(), alreadyWrappedClient, retryWrapper.raw)
	assert.False(testSuite.T(), retryWrapper.enableRetriesOnFolderAPIs, "Retries should not be enabled for folder APIs")
}

func (testSuite *AllApiRetryWrapperTest) TestWithRetry_AllAPIs_WrapsClient() {
	// Arrange
	mockClient := new(MockStorageControlClient)
	clientConfig := storageutil.GetDefaultStorageClientConfig(keyFile)

	wrappedClient := newStorageControlClientWithRetry(mockClient, &clientConfig).
		WithRetriesOnFolderAPI()

	// Assert
	require.NotNil(testSuite.T(), wrappedClient)
	retryWrapper := wrappedClient
	assert.Same(testSuite.T(), mockClient, retryWrapper.raw)
	assert.True(testSuite.T(), retryWrapper.enableRetriesOnFolderAPIs, "Retries should be enabled for folder APIs")
}

func (testSuite *AllApiRetryWrapperTest) TestWithRetry_AllAPIs_UnwrapsNestedRetryClient() {
	// Arrange
	mockClient := new(MockStorageControlClient)
	clientConfig := storageutil.GetDefaultStorageClientConfig(keyFile)
	alreadyWrappedClient := newStorageControlClientWithRetry(mockClient, &clientConfig).
		WithRetriesOnFolderAPI()

	// Wrap it again.
	doubleWrappedClient := newStorageControlClientWithRetry(alreadyWrappedClient, &clientConfig).
		WithRetriesOnFolderAPI()

	// Assert
	require.NotNil(testSuite.T(), doubleWrappedClient)
	retryWrapper := doubleWrappedClient
	assert.Same(testSuite.T(), mockClient, retryWrapper.raw, "Should unwrap the nested retry client")
	assert.NotSame(testSuite.T(), alreadyWrappedClient, retryWrapper.raw)
	assert.True(testSuite.T(), retryWrapper.enableRetriesOnFolderAPIs, "Retries should be enabled for folder APIs")
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

func (testSuite *ControlClientGaxRetryWrapperTest) TestStorageControlClientGaxRetryOptions_UnauthenticatedIsRetryable() {
	clientConfig := storageutil.GetDefaultStorageClientConfig(keyFile)
	gaxOpts := storageControlClientGaxRetryOptions(&clientConfig)
	var settings gax.CallSettings
	for _, opt := range gaxOpts {
		opt.Resolve(&settings)
	}
	require.NotNil(testSuite.T(), settings.Retry)
	retryer := settings.Retry()
	require.NotNil(testSuite.T(), retryer)

	_, shouldRetryUnauthenticated := retryer.Retry(status.Error(codes.Unauthenticated, "unauthenticated"))

	assert.True(testSuite.T(), shouldRetryUnauthenticated)
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

func (testSuite *StorageLayoutRetryWrapperTest) TestWithBillingProject_EmptyString() {
	mockClient := new(MockStorageControlClient)
	clientConfig := storageutil.GetDefaultStorageClientConfig(keyFile)

	scc := newStorageControlClientWithRetry(mockClient, &clientConfig).
		WithBillingProject("")

	require.NotNil(testSuite.T(), scc)
	// Empty string should return the *storageControlClientWithRetry without wrapping with billing project.
	_, ok := scc.(*storageControlClientWithRetry)
	assert.True(testSuite.T(), ok, "Expected *storageControlClientWithRetry type when billing project is empty")
}

func (testSuite *StorageLayoutRetryWrapperTest) TestWithBillingProject_NonEmptyString() {
	mockClient := new(MockStorageControlClient)
	clientConfig := storageutil.GetDefaultStorageClientConfig(keyFile)

	scc := newStorageControlClientWithRetry(mockClient, &clientConfig).
		WithBillingProject("my-project")

	require.NotNil(testSuite.T(), scc)
	// Non-empty string should wrap the client with billing project.
	wrappedClient, ok := scc.(*storageControlClientWithBillingProject)
	assert.True(testSuite.T(), ok, "Expected *storageControlClientWithBillingProject type")
	assert.Equal(testSuite.T(), "my-project", wrappedClient.billingProject)
}

func (testSuite *StorageLayoutRetryWrapperTest) TestWithBillingProject_InjectsHeader() {
	mockClient := new(MockStorageControlClient)
	clientConfig := storageutil.GetDefaultStorageClientConfig(keyFile)
	scc := newStorageControlClientWithRetry(mockClient, &clientConfig).
		WithBillingProject("my-project")
	req := &controlpb.GetStorageLayoutRequest{Name: "buckets/my-bucket"}
	// Verify that when GetStorageLayout is called, the context has outgoing metadata "x-goog-user-project: my-project"
	mockClient.On("GetStorageLayout", mock.MatchedBy(func(ctx context.Context) bool {
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			return false
		}
		values := md.Get("x-goog-user-project")
		return len(values) == 1 && values[0] == "my-project"
	}), req, mock.Anything).Return(&controlpb.StorageLayout{}, nil).Once()

	_, err := scc.GetStorageLayout(context.Background(), req)

	assert.NoError(testSuite.T(), err)
	mockClient.AssertExpectations(testSuite.T())
}

func (testSuite *StorageLayoutRetryWrapperTest) TestWithBillingProject_RenameFolder_NoHeader() {
	mockClient := new(MockStorageControlClient)
	clientConfig := storageutil.GetDefaultStorageClientConfig(keyFile)
	scc := newStorageControlClientWithRetry(mockClient, &clientConfig).
		WithBillingProject("my-project")
	req := &controlpb.RenameFolderRequest{Name: "buckets/my-bucket/folders/f1", DestinationFolderId: "f2"}
	// Verify that when RenameFolder is called, the context does not have outgoing metadata "x-goog-user-project"
	mockClient.On("RenameFolder", mock.MatchedBy(func(ctx context.Context) bool {
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			return true // No metadata is correct
		}
		values := md.Get("x-goog-user-project")
		return len(values) == 0
	}), req, mock.Anything).Return(&control.RenameFolderOperation{}, nil).Once()

	_, err := scc.RenameFolder(context.Background(), req)

	assert.NoError(testSuite.T(), err)
	mockClient.AssertExpectations(testSuite.T())
}

func (testSuite *StorageLayoutRetryWrapperTest) TestWithRetriesOnFolderAPI_NilReceiver() {
	var scc *storageControlClientWithRetry

	result := scc.WithRetriesOnFolderAPI()

	assert.Nil(testSuite.T(), result)
}

func (testSuite *StorageLayoutRetryWrapperTest) TestWithBillingProject_NilReceiver() {
	var scc *storageControlClientWithRetry

	result := scc.WithBillingProject("my-project")

	assert.Nil(testSuite.T(), result)
}
