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

package storageutil

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ExponentialBackoffTestSuite struct {
	suite.Suite
}

func TestExponentialBackoffTestSuite(t *testing.T) {
	suite.Run(t, new(ExponentialBackoffTestSuite))
}

func TestExecuteWithRetryTestSuite(t *testing.T) {
	suite.Run(t, new(ExecuteWithRetryTestSuite))
}

func TestRetryConfigTestSuite(t *testing.T) {
	suite.Run(t, new(RetryConfigTestSuite))
}

func (t *ExponentialBackoffTestSuite) TestNewBackoff() {
	initial := 1 * time.Second
	maxValue := 10 * time.Second
	multiplier := 2.0

	b := newExponentialBackoff(&exponentialBackoffConfig{
		initial:    initial,
		max:        maxValue,
		multiplier: multiplier,
	})

	assert.NotNil(t.T(), b)
	assert.Equal(t.T(), initial, b.next)
	assert.Equal(t.T(), initial, b.config.initial)
	assert.Equal(t.T(), maxValue, b.config.max)
	assert.Equal(t.T(), multiplier, b.config.multiplier)
}

func (t *ExponentialBackoffTestSuite) TestNext() {
	initial := 1 * time.Second
	maxValue := 3 * time.Second
	multiplier := 2.0
	b := newExponentialBackoff(&exponentialBackoffConfig{
		initial:    initial,
		max:        maxValue,
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

func (t *ExponentialBackoffTestSuite) TestWaitWithJitter_ContextCancelled() {
	initial := 100 * time.Microsecond // A long duration to ensure cancellation happens first.
	maxValue := 5 * initial
	b := newExponentialBackoff(&exponentialBackoffConfig{
		initial:    initial,
		max:        maxValue,
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

func (t *ExponentialBackoffTestSuite) TestWaitWithJitter_NoContextCancelled() {
	initial := 5 * time.Millisecond // A short duration to ensure it waits. Making it any shorter can cause random failures
	// because context cancel itself takes about a millisecond.
	maxValue := 5 * initial
	b := newExponentialBackoff(&exponentialBackoffConfig{
		initial:    initial,
		max:        maxValue,
		multiplier: 2.0,
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	start := time.Now()
	err := b.waitWithJitter(ctx)
	elapsed := time.Since(start)

	assert.NoError(t.T(), err)
	// The function should wait for a duration higher than initial, but not too high.
	// Keeping a somewhat loose limit to avoid failing because of go itself taking around 10ms sometimes
	// to return.
	assert.LessOrEqual(t.T(), elapsed, initial*3, "waitWithJitter should not wait excessively long")
}

func (t *ExponentialBackoffTestSuite) TestWaitWithJitter_BackoffGrowth() {
	initial := 2 * time.Millisecond
	maxValue := 50 * time.Millisecond
	multiplier := 2.0
	b := newExponentialBackoff(&exponentialBackoffConfig{
		initial:    initial,
		max:        maxValue,
		multiplier: multiplier,
	})
	ctx := context.Background()

	// First call to establish the initial backoff.
	err := b.waitWithJitter(ctx)
	assert.NoError(t.T(), err)
	var lastBackoff time.Duration

	// Subsequent calls should demonstrate exponential growth.
	for range 3 {
		err := b.waitWithJitter(ctx)
		require.NoError(t.T(), err)

		// The new backoff should be at least the last backoff times the multiplier.
		// Due to jitter, it can be larger, so we check for >=
		expectedMinBackoff := time.Duration(float64(lastBackoff) * multiplier)
		require.GreaterOrEqual(t.T(), b.prev, expectedMinBackoff)

		// The backoff should also be capped by the max value.
		if b.prev > maxValue {
			require.Equal(t.T(), b.prev, maxValue)
		}

		lastBackoff = b.prev
	}
}

type RetryConfigTestSuite struct {
	suite.Suite
}

func (t *RetryConfigTestSuite) TestNewRetryConfig() {
	// Arrange
	clientConfig := &StorageClientConfig{
		MaxRetrySleep:   10 * time.Second,
		RetryMultiplier: 2.5,
	}
	retryDeadline := 5 * time.Second
	totalRetryBudget := 30 * time.Second
	initialBackoff := 500 * time.Millisecond

	// Act
	retryConfig := NewRetryConfig(clientConfig, retryDeadline, totalRetryBudget, initialBackoff)

	// Assert
	assert.NotNil(t.T(), retryConfig)
	assert.Equal(t.T(), retryDeadline, retryConfig.RetryDeadline)
	assert.Equal(t.T(), totalRetryBudget, retryConfig.TotalRetryBudget)
	assert.Equal(t.T(), initialBackoff, retryConfig.BackoffConfig.initial)
	assert.Equal(t.T(), clientConfig.MaxRetrySleep, retryConfig.BackoffConfig.max)
	assert.Equal(t.T(), clientConfig.RetryMultiplier, retryConfig.BackoffConfig.multiplier)
}

type ExecuteWithRetryTestSuite struct {
	suite.Suite
	retryConfig *RetryConfig
}

func (t *ExecuteWithRetryTestSuite) SetupTest() {
	t.retryConfig = &RetryConfig{
		RetryDeadline:    500 * time.Microsecond,
		TotalRetryBudget: 2000 * time.Microsecond,
		BackoffConfig: exponentialBackoffConfig{
			initial:    1 * time.Microsecond,
			max:        100 * time.Microsecond,
			multiplier: 2,
		},
	}
}

func (t *ExecuteWithRetryTestSuite) TestExecuteWithRetry_SuccessOnFirstAttempt() {
	// Arrange
	var callCount int
	apiCall := func(ctx context.Context) (string, error) {
		callCount++
		return "success", nil
	}

	// Act
	result, err := ExecuteWithRetry(context.Background(), t.retryConfig, "testOp", "testReq", apiCall)

	// Assert
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), "success", result)
	assert.Equal(t.T(), 1, callCount)
}

func (t *ExecuteWithRetryTestSuite) TestExecuteWithRetry_SuccessAfterRetry() {
	// Arrange
	var callCount int
	retryableErr := status.Error(codes.Unavailable, "server unavailable")
	apiCall := func(ctx context.Context) (string, error) {
		callCount++
		if callCount == 1 {
			return "", retryableErr
		}
		return "success", nil
	}

	// Act
	result, err := ExecuteWithRetry(context.Background(), t.retryConfig, "testOp", "testReq", apiCall)

	// Assert
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), "success", result)
	assert.Equal(t.T(), 2, callCount)
}

func (t *ExecuteWithRetryTestSuite) TestExecuteWithRetry_FailureOnNonRetryableError() {
	// Arrange
	var callCount int
	nonRetryableErr := errors.New("non-retryable error")
	apiCall := func(ctx context.Context) (string, error) {
		callCount++
		return "", nonRetryableErr
	}

	// Act
	result, err := ExecuteWithRetry(context.Background(), t.retryConfig, "testOp", "testReq", apiCall)

	// Assert
	assert.ErrorIs(t.T(), err, nonRetryableErr)
	assert.Empty(t.T(), result)
	assert.Equal(t.T(), 1, callCount)
}

func (t *ExecuteWithRetryTestSuite) TestExecuteWithRetry_RetryableThenNonRetryableError() {
	// Arrange
	var callCount int
	retryableErr := status.Error(codes.Unavailable, "server unavailable")
	nonRetryableErr := errors.New("non-retryable error")
	apiCall := func(ctx context.Context) (string, error) {
		callCount++
		if callCount == 1 {
			return "", retryableErr
		}
		return "", nonRetryableErr
	}

	// Act
	result, err := ExecuteWithRetry(context.Background(), t.retryConfig, "testOp", "testReq", apiCall)

	// Assert
	assert.ErrorIs(t.T(), err, nonRetryableErr)
	assert.Empty(t.T(), result)
	assert.Equal(t.T(), 2, callCount)
}

func (t *ExecuteWithRetryTestSuite) TestExecuteWithRetry_Timeout() {
	// Arrange
	stallDuration := t.retryConfig.RetryDeadline + 10*time.Millisecond
	var callCount int
	apiCall := func(ctx context.Context) (string, error) {
		callCount++
		// Simulate a call that always takes longer than the per-attempt deadline.
		select {
		case <-time.After(stallDuration):
			// This case should not be hit, as the context deadline
			// is shorter by 10ms than stallDuration.
			return "", errors.New("simulated apiCall finished before context timeout")
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	// Act
	_, err := ExecuteWithRetry(context.Background(), t.retryConfig, "testOp", "testReq", apiCall)

	// Assert
	assert.ErrorIs(t.T(), err, context.DeadlineExceeded, "Expected context.DeadlineExceeded because each attempt is designed to "+
		"take longer than the per-attempt deadline.")
}

func (t *ExecuteWithRetryTestSuite) TestExecuteWithRetry_TotalRetryBudgetExceeded() {
	// Arrange
	var callCount int
	// Set a short total retry budget.
	t.retryConfig.TotalRetryBudget = 500 * time.Microsecond
	t.retryConfig.RetryDeadline = 100 * time.Microsecond
	t.retryConfig.BackoffConfig.initial = 1 * time.Microsecond // Ensure backoff pushes it over the edge.
	apiCall := func(ctx context.Context) (string, error) {
		callCount++
		return "", status.Error(codes.Unavailable, "server unavailable")
	}

	// Act
	_, err := ExecuteWithRetry(context.Background(), t.retryConfig, "testOp", "testReq", apiCall)

	// Assert
	assert.ErrorIs(t.T(), err, context.DeadlineExceeded, "The error should be from the total retry budget timeout")
	assert.Greater(t.T(), callCount, 1)
}

func (t *ExecuteWithRetryTestSuite) TestExecuteWithRetry_ParentContextTimeoutShorterThanRetryDeadline() {
	// Arrange
	var callCount int
	t.retryConfig.RetryDeadline = 1000 * time.Microsecond
	t.retryConfig.TotalRetryBudget = 10000 * time.Microsecond
	stallDuration := t.retryConfig.RetryDeadline + 1000*time.Microsecond
	// Set a parent context timeout that is shorter than the total retry budget.
	parentCtx, cancel := context.WithTimeout(context.Background(), t.retryConfig.RetryDeadline-500*time.Microsecond)
	defer cancel()
	apiCall := func(ctx context.Context) (string, error) {
		callCount++
		select {
		case <-time.After(stallDuration):
		case <-ctx.Done():
			return "", ctx.Err()
		}
		// This will always fail with a retryable error.
		return "", status.Error(codes.Unavailable, "server unavailable")
	}

	// Act
	// The parent context will be checked within ExecuteWithRetry before the first attempt,
	// but the attempt will still proceed. The attempt's context will expire
	// due to the parent's timeout.
	result, err := ExecuteWithRetry(parentCtx, t.retryConfig, "testOp", "testReq", apiCall)

	// Assert
	assert.ErrorIs(t.T(), err, context.DeadlineExceeded, "The error should be from the parent context's timeout")
	assert.Empty(t.T(), result)
	assert.Equal(t.T(), 1, callCount, "apiCall should have been called once")
}

func (t *ExecuteWithRetryTestSuite) TestExecuteWithRetry_ParentContextTimeoutBetweenDeadlines() {
	// Arrange
	var callCount int
	stallDuration := t.retryConfig.RetryDeadline + 5*time.Microsecond
	// Set a parent context timeout that is longer than one attempt but shorter than the total budget.
	parentCtx, cancel := context.WithTimeout(context.Background(), t.retryConfig.RetryDeadline+50*time.Microsecond)
	defer cancel()
	apiCall := func(ctx context.Context) (string, error) {
		callCount++
		select {
		case <-time.After(stallDuration):
		case <-ctx.Done():
			return "", ctx.Err()
		}
		// This will always fail with a retryable error.
		return "", status.Error(codes.Unavailable, "server unavailable")
	}

	// Act
	result, err := ExecuteWithRetry(parentCtx, t.retryConfig, "testOp", "testReq", apiCall)

	// Assert
	assert.ErrorIs(t.T(), err, context.DeadlineExceeded, "The error should be from the parent context's timeout")
	assert.Empty(t.T(), result)
	assert.Greater(t.T(), callCount, 0, "apiCall should have been called at least once")
}

func (t *ExecuteWithRetryTestSuite) TestExecuteWithRetry_ParentContextTimeoutLongerThanBudget() {
	// Arrange
	stallDuration := t.retryConfig.RetryDeadline + 5*time.Microsecond
	parentCtx, cancel := context.WithTimeout(context.Background(), t.retryConfig.TotalRetryBudget+100*time.Microsecond)
	defer cancel()
	apiCall := func(ctx context.Context) (string, error) {
		select {
		case <-time.After(stallDuration):
		case <-ctx.Done():
			return "", ctx.Err()
		}
		return "", status.Error(codes.Unavailable, "server unavailable")
	}

	// Act
	result, err := ExecuteWithRetry(parentCtx, t.retryConfig, "testOp", "testReq", apiCall)

	// Assert
	assert.ErrorIs(t.T(), err, context.DeadlineExceeded, "The error should be from context created in ExecuteWithRetry")
	assert.Empty(t.T(), result)
}

func (t *ExecuteWithRetryTestSuite) TestExecuteWithRetry_ParentContextAlreadyCancelled() {
	// Arrange
	var callCount int
	parentCtx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel the context immediately.
	apiCall := func(ctx context.Context) (string, error) {
		callCount++
		return "should not be called", nil
	}

	// Act
	_, err := ExecuteWithRetry(parentCtx, t.retryConfig, "testOp", "testReq", apiCall)

	// Assert
	assert.ErrorIs(t.T(), err, context.Canceled)
	assert.Equal(t.T(), 0, callCount, "apiCall should not have been executed")
}
