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
	return s.wrapped.DeleteFolder(ctx, req, opts...)
}

func (s *stallingStorageControlClient) GetFolder(ctx context.Context, req *controlpb.GetFolderRequest, opts ...gax.CallOption) (*controlpb.Folder, error) {
	return s.wrapped.GetFolder(ctx, req, opts...)
}

func (s *stallingStorageControlClient) RenameFolder(ctx context.Context, req *controlpb.RenameFolderRequest, opts ...gax.CallOption) (*control.RenameFolderOperation, error) {
	return s.wrapped.RenameFolder(ctx, req, opts...)
}

func (s *stallingStorageControlClient) CreateFolder(ctx context.Context, req *controlpb.CreateFolderRequest, opts ...gax.CallOption) (*controlpb.Folder, error) {
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

func TestControlClientWrapperTestSuite(t *testing.T) {
	t.Run("StorageLayoutStallRetryWrapperTest", func(t *testing.T) {
		suite.Run(t, new(StorageLayoutStallRetryWrapperTest))
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

func (t *StorageLayoutStallRetryWrapperTest) TestGetStorageLayout_SuccessOnFirstAttempt() {
	// Arrange
	client := newRetryWrapper(t.stallingClient, 100*time.Microsecond, 500*time.Microsecond, 2, 1000*time.Microsecond, 1*time.Microsecond, 10*time.Microsecond, 2)
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
	client := newRetryWrapper(t.stallingClient, 100*time.Microsecond, 500*time.Microsecond, 2, 1000*time.Microsecond, 1*time.Microsecond, 10*time.Microsecond, 2)
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
	client := newRetryWrapper(t.stallingClient, 100*time.Microsecond, 500*time.Microsecond, 2, 1000*time.Microsecond, 1*time.Microsecond, 10*time.Microsecond, 2)
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
	client := newRetryWrapper(t.stallingClient, 100*time.Microsecond, 500*time.Microsecond, 2, 1000*time.Microsecond, 1*time.Microsecond, 10*time.Microsecond, 2)
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
	client := newRetryWrapper(t.stallingClient, 1000*time.Microsecond, 5000*time.Microsecond, 2, 10000*time.Microsecond, 1*time.Microsecond, 10*time.Microsecond, 2)
	req := &controlpb.GetStorageLayoutRequest{Name: "some/bucket"}
	// Set stall time to be longer than the max attempt timeout.
	t.stallTimeForGetStorageLayout = 6000 * time.Microsecond

	// Act
	_, err := client.GetStorageLayout(t.ctx, req)

	// The mock should never be called because every attempt will time out.
	assert.ErrorIs(t.T(), err, context.DeadlineExceeded)
	t.mockRawClient.AssertExpectations(t.T())
}
