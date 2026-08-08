// Copyright 2026 Google LLC
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

package storageutil

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/googleapis/gax-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockLROPoller struct {
	mock.Mock
}

func (m *mockLROPoller) Poll(ctx context.Context, opts ...gax.CallOption) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

func (m *mockLROPoller) Done() bool {
	args := m.Called()
	return args.Bool(0)
}

func TestPollLRO_ImmediateSuccess(t *testing.T) {
	ctx := context.Background()
	poller := new(mockLROPoller)
	expectedResult := "success"
	poller.On("Poll", ctx).Return(expectedResult, nil).Once()
	poller.On("Done").Return(true).Once()
	cfg := LROPollConfig{
		Min:     1 * time.Millisecond,
		Max:     10 * time.Millisecond,
		CapTime: 1 * time.Second,
	}

	result, err := PollLRO(ctx, poller, cfg)

	assert.NoError(t, err)
	assert.Equal(t, expectedResult, result)
	poller.AssertExpectations(t)
}

func TestPollLRO_ImmediateError(t *testing.T) {
	ctx := context.Background()
	poller := new(mockLROPoller)
	expectedErr := errors.New("operation failed")
	poller.On("Poll", ctx).Return("", expectedErr).Once()
	cfg := DefaultLROPollConfig()

	result, err := PollLRO(ctx, poller, cfg)

	assert.ErrorIs(t, err, expectedErr)
	assert.Equal(t, "", result)
	poller.AssertExpectations(t)
}

func TestPollLRO_SuccessAfterRetries(t *testing.T) {
	ctx := context.Background()
	poller := new(mockLROPoller)
	expectedResult := "success_after_retry"
	poller.On("Poll", ctx).Return("", nil).Once()
	poller.On("Done").Return(false).Once()
	poller.On("Poll", ctx).Return("", nil).Once()
	poller.On("Done").Return(false).Once()
	poller.On("Poll", ctx).Return(expectedResult, nil).Once()
	poller.On("Done").Return(true).Once()
	cfg := LROPollConfig{
		Min:     1 * time.Millisecond,
		Max:     10 * time.Millisecond,
		CapTime: 1 * time.Second,
	}

	result, err := PollLRO(ctx, poller, cfg)

	assert.NoError(t, err)
	assert.Equal(t, expectedResult, result)
	poller.AssertExpectations(t)
}

func TestPollLRO_NonRetryableErrorPropagated(t *testing.T) {
	ctx := context.Background()
	poller := new(mockLROPoller)
	nonRetryableErr := errors.New("some non-retryable error")
	poller.On("Poll", ctx).Return("", nil).Once()
	poller.On("Done").Return(false).Once()
	poller.On("Poll", ctx).Return("", nonRetryableErr).Once()
	cfg := LROPollConfig{
		Min:     1 * time.Millisecond,
		Max:     10 * time.Millisecond,
		CapTime: 1 * time.Second,
	}

	result, err := PollLRO(ctx, poller, cfg)

	assert.ErrorIs(t, err, nonRetryableErr)
	assert.Equal(t, "", result)
	poller.AssertExpectations(t)
}

func TestPollLRO_RetryOn429ResourceExhausted(t *testing.T) {
	ctx := context.Background()
	poller := new(mockLROPoller)
	expectedResult := "success_after_429"
	err429 := status.Error(codes.ResourceExhausted, "quota exceeded")
	// Poll 0 (immediate) returns 429
	poller.On("Poll", ctx).Return("", err429).Once()
	poller.On("Done").Return(false).Once()
	// Loop poll 1 returns 429
	poller.On("Poll", ctx).Return("", err429).Once()
	poller.On("Done").Return(false).Once()
	// Loop poll 2 succeeds
	poller.On("Poll", ctx).Return(expectedResult, nil).Once()
	poller.On("Done").Return(true).Once()
	cfg := LROPollConfig{
		Min:     1 * time.Millisecond,
		Max:     10 * time.Millisecond,
		CapTime: 1 * time.Second,
	}

	result, err := PollLRO(ctx, poller, cfg)

	assert.NoError(t, err)
	assert.Equal(t, expectedResult, result)
	poller.AssertExpectations(t)
}

func TestPollLRO_RetryOnUnauthenticated(t *testing.T) {
	ctx := context.Background()
	poller := new(mockLROPoller)
	expectedResult := "success_after_auth_refresh"
	errAuth := status.Error(codes.Unauthenticated, "token expired")
	// Poll 0 (immediate) returns auth error
	poller.On("Poll", ctx).Return("", errAuth).Once()
	poller.On("Done").Return(false).Once()
	// Loop poll 1 succeeds
	poller.On("Poll", ctx).Return(expectedResult, nil).Once()
	poller.On("Done").Return(true).Once()
	cfg := LROPollConfig{
		Min:     1 * time.Millisecond,
		Max:     10 * time.Millisecond,
		CapTime: 1 * time.Second,
	}

	result, err := PollLRO(ctx, poller, cfg)

	assert.NoError(t, err)
	assert.Equal(t, expectedResult, result)
	poller.AssertExpectations(t)
}

func TestPollLRO_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	poller := new(mockLROPoller)
	poller.On("Poll", ctx).Return("", nil).Once()
	poller.On("Done").Return(false).Once()
	cancel()
	cfg := LROPollConfig{
		Min:     1 * time.Millisecond,
		Max:     10 * time.Millisecond,
		CapTime: 1 * time.Second,
	}

	result, err := PollLRO(ctx, poller, cfg)

	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, "", result)
	poller.AssertExpectations(t)
}

func TestPollLRO_InvalidConfig(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name string
		cfg  LROPollConfig
	}{
		{
			name: "invalid_min",
			cfg: LROPollConfig{
				Min:     -1 * time.Millisecond,
				Max:     10 * time.Millisecond,
				CapTime: 1 * time.Second,
			},
		},
		{
			name: "invalid_min_greater_than_max",
			cfg: LROPollConfig{
				Min:     20 * time.Millisecond,
				Max:     10 * time.Millisecond,
				CapTime: 1 * time.Second,
			},
		},
		{
			name: "invalid_max",
			cfg: LROPollConfig{
				Min:     1 * time.Millisecond,
				Max:     0,
				CapTime: 1 * time.Second,
			},
		},
		{
			name: "invalid_cap_time",
			cfg: LROPollConfig{
				Min:     1 * time.Millisecond,
				Max:     10 * time.Millisecond,
				CapTime: 0,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			poller := new(mockLROPoller)
			// Poller should not be called if config is invalid
			_, err := PollLRO(ctx, poller, tc.cfg)

			assert.Error(t, err)
			poller.AssertNotCalled(t, "Poll")
			poller.AssertNotCalled(t, "Done")
		})
	}
}
