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
	wrapped                          StorageControlClient
	stallDurationForGetStorageLayout *time.Duration
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

type ControlClientRetryWrapperTest struct {
	suite.Suite
	// The raw mock client for setting expectations on return values.
	mockRawClient *MockStorageControlClient
	ctx           context.Context
}

type StorageLayoutRetryWrapperTest struct {
	ControlClientRetryWrapperTest
	stallingClient *stallingStorageControlClient
	// The simulated execution time for each GetStorageLayout call made through stallingClient.
	stallDurationForGetStorageLayout time.Duration
}

func TestControlClientWrapperTestSuite(t *testing.T) {
	t.Run("StorageLayoutRetryWrapperTest", func(t *testing.T) {
		suite.Run(t, new(StorageLayoutRetryWrapperTest))
	})
}

func (t *ControlClientRetryWrapperTest) SetupTest() {
	t.mockRawClient = new(MockStorageControlClient)
	t.ctx = context.Background()
}

func (t *StorageLayoutRetryWrapperTest) SetupTest() {
	t.ControlClientRetryWrapperTest.SetupTest()
	t.stallDurationForGetStorageLayout = 0
	t.stallingClient = &stallingStorageControlClient{
		wrapped:                          t.mockRawClient,
		stallDurationForGetStorageLayout: &t.stallDurationForGetStorageLayout,
	}
}

func (t *StorageLayoutRetryWrapperTest) TestGetStorageLayout_SuccessOnFirstAttempt() {
	// Arrange
	client := newRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, 1*time.Microsecond, 10*time.Microsecond, 2)
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
	client := newRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, 1*time.Microsecond, 10*time.Microsecond, 2)
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
	client := newRetryWrapper(t.stallingClient, 100*time.Microsecond, 1000*time.Microsecond, 1*time.Microsecond, 10*time.Microsecond, 2)
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
	client := newRetryWrapper(t.stallingClient, 1000*time.Microsecond, 10000*time.Microsecond, 1*time.Microsecond, 10*time.Microsecond, 2)
	req := &controlpb.GetStorageLayoutRequest{Name: "some/bucket"}
	// Set stall time to be longer than the attempt timeout.
	t.stallDurationForGetStorageLayout = 6000 * time.Microsecond

	// Act
	_, err := client.GetStorageLayout(t.ctx, req)

	// The mock should never be called because every attempt will time out.
	assert.ErrorIs(t.T(), err, context.DeadlineExceeded)
	t.mockRawClient.AssertExpectations(t.T())
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

	b := newBackoff(initial, max, multiplier)

	assert.NotNil(t.T(), b)
	assert.Equal(t.T(), initial, b.next)
	assert.Equal(t.T(), initial, b.min)
	assert.Equal(t.T(), max, b.max)
	assert.Equal(t.T(), multiplier, b.multiplier)
}

func (t *ExponentialBackoffTest) TestNext() {
	initial := 1 * time.Second
	max := 3 * time.Second
	multiplier := 2.0
	b := newBackoff(initial, max, multiplier)

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
	b := newBackoff(initial, max, 2.0)
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
