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
		Initial:    1 * time.Millisecond,
		Multiplier: 1.1,
		Max:        10 * time.Millisecond,
	}

	result, err := PollLRO[string](ctx, poller, cfg)

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

	result, err := PollLRO[string](ctx, poller, cfg)

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
		Initial:    1 * time.Millisecond,
		Multiplier: 1.1,
		Max:        10 * time.Millisecond,
	}

	result, err := PollLRO[string](ctx, poller, cfg)

	assert.NoError(t, err)
	assert.Equal(t, expectedResult, result)
	poller.AssertExpectations(t)
}

func TestPollLRO_TransientRPCErrorPropagated(t *testing.T) {
	ctx := context.Background()
	poller := new(mockLROPoller)
	transientErr := errors.New("transient network error")
	poller.On("Poll", ctx).Return("", nil).Once()
	poller.On("Done").Return(false).Once()
	poller.On("Poll", ctx).Return("", transientErr).Once()
	cfg := LROPollConfig{
		Initial:    1 * time.Millisecond,
		Multiplier: 1.1,
		Max:        10 * time.Millisecond,
	}

	result, err := PollLRO[string](ctx, poller, cfg)

	assert.ErrorIs(t, err, transientErr)
	assert.Equal(t, "", result)
	poller.AssertExpectations(t)
}

func TestPollLRO_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	poller := new(mockLROPoller)
	poller.On("Poll", ctx).Return("", nil).Once()
	poller.On("Done").Return(false).Once()
	cancel()
	cfg := LROPollConfig{
		Initial:    50 * time.Millisecond,
		Multiplier: 1.1,
		Max:        100 * time.Millisecond,
	}

	result, err := PollLRO[string](ctx, poller, cfg)

	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, "", result)
	poller.AssertExpectations(t)
}

func TestPollLRO_FixedFastPhaseWindow(t *testing.T) {
	ctx := context.Background()
	poller := new(mockLROPoller)
	expectedResult := "success_fast_phase"
	poller.On("Poll", ctx).Return("", nil).Once()
	poller.On("Done").Return(false).Once()
	poller.On("Poll", ctx).Return("", nil).Once()
	poller.On("Done").Return(false).Once()
	poller.On("Poll", ctx).Return(expectedResult, nil).Once()
	poller.On("Done").Return(true).Once()
	cfg := LROPollConfig{
		Initial:         1 * time.Millisecond,
		FastPhaseWindow: 10 * time.Millisecond,
		Multiplier:      2.0,
		Max:             100 * time.Millisecond,
	}

	result, err := PollLRO[string](ctx, poller, cfg)

	assert.NoError(t, err)
	assert.Equal(t, expectedResult, result)
	poller.AssertExpectations(t)
}
