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

func (t *ExponentialBackoffTestSuite) TestNewBackoff() {
	initial := 1 * time.Second
	max := 10 * time.Second
	multiplier := 2.0

	b := NewExponentialBackoff(&ExponentialBackoffConfig{
		Initial:    initial,
		Max:        max,
		Multiplier: multiplier,
	})

	assert.NotNil(t.T(), b)
	assert.Equal(t.T(), initial, b.next)
	assert.Equal(t.T(), initial, b.config.Initial)
	assert.Equal(t.T(), max, b.config.Max)
	assert.Equal(t.T(), multiplier, b.config.Multiplier)
}

func (t *ExponentialBackoffTestSuite) TestNext() {
	initial := 1 * time.Second
	max := 3 * time.Second
	multiplier := 2.0
	b := NewExponentialBackoff(&ExponentialBackoffConfig{
		Initial:    initial,
		Max:        max,
		Multiplier: multiplier,
	})

	// First call to next() should return Initial, and update current.
	assert.Equal(t.T(), 1*time.Second, b.nextDuration())

	// Second call.
	assert.Equal(t.T(), 2*time.Second, b.nextDuration())

	// Third call - capped at Max.
	assert.Equal(t.T(), 3*time.Second, b.nextDuration())

	// Should stay capped at Max.
	assert.Equal(t.T(), 3*time.Second, b.nextDuration())
}

func (t *ExponentialBackoffTestSuite) TestWaitWithJitter_ContextCancelled() {
	initial := 100 * time.Microsecond // A long duration to ensure cancellation happens first.
	max := 5 * initial
	b := NewExponentialBackoff(&ExponentialBackoffConfig{
		Initial:    initial,
		Max:        max,
		Multiplier: 2.0,
	})
	ctx, cancel := context.WithCancel(context.Background())
	// Cancel the context immediately.
	cancel()

	start := time.Now()
	err := b.WaitWithJitter(ctx)
	elapsed := time.Since(start)

	assert.ErrorIs(t.T(), err, context.Canceled)
	// The function should return almost immediately.
	assert.Less(t.T(), elapsed, initial, "waitWithJitter should return quickly when context is cancelled")
}

func (t *ExponentialBackoffTestSuite) TestWaitWithJitter_NoContextCancelled() {
	initial := time.Millisecond // A short duration to ensure it waits. Making it any shorter can cause random failures
	// because context cancel itself takes about a millisecond.
	max := 5 * initial
	b := NewExponentialBackoff(&ExponentialBackoffConfig{
		Initial:    initial,
		Max:        max,
		Multiplier: 2.0,
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	start := time.Now()
	err := b.WaitWithJitter(ctx)
	elapsed := time.Since(start)

	assert.NoError(t.T(), err)
	// The function should wait for a duration close to Initial.
	assert.LessOrEqual(t.T(), elapsed, initial*2, "waitWithJitter should not wait excessively long")
}

func (t *ExponentialBackoffTestSuite) TestNewRetryConfig() {
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
	assert.Equal(t.T(), initialBackoff, retryConfig.BackoffConfig.Initial)
	assert.Equal(t.T(), clientConfig.MaxRetrySleep, retryConfig.BackoffConfig.Max)
	assert.Equal(t.T(), clientConfig.RetryMultiplier, retryConfig.BackoffConfig.Multiplier)
}

type ExecuteWithRetryTestSuite struct {
	suite.Suite
	retryConfig *RetryConfig
}

func (t *ExecuteWithRetryTestSuite) SetupTest() {
	t.retryConfig = &RetryConfig{
		RetryDeadline:    50 * time.Millisecond,
		TotalRetryBudget: 200 * time.Millisecond,
		BackoffConfig: ExponentialBackoffConfig{
			Initial:    5 * time.Millisecond,
			Max:        20 * time.Millisecond,
			Multiplier: 2,
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
	assert.Error(t.T(), err)
	assert.Empty(t.T(), result)
	assert.Contains(t.T(), err.Error(), "failed with a non-retryable error")
	assert.ErrorIs(t.T(), err, nonRetryableErr)
	assert.Equal(t.T(), 1, callCount)
}

func (t *ExecuteWithRetryTestSuite) TestExecuteWithRetry_Timeout() {
	// Arrange
	stallDuration := t.retryConfig.RetryDeadline + 5*time.Millisecond
	var callCount int
	apiCall := func(ctx context.Context) (string, error) {
		callCount++
		// Simulate a call that always takes longer than the per-attempt deadline.
		time.Sleep(stallDuration)
		return "", ctx.Err()
	}

	// Act
	result, err := ExecuteWithRetry(context.Background(), t.retryConfig, "testOp", "testReq", apiCall)

	// Assert
	assert.Error(t.T(), err)
	assert.Empty(t.T(), result)
	assert.ErrorIs(t.T(), err, context.DeadlineExceeded)
	assert.Contains(t.T(), err.Error(), "failed after multiple retries")
}

func (t *ExecuteWithRetryTestSuite) TestExecuteWithRetry_ParentContextTimeout() {
	// Arrange
	var callCount int
	// Set a parent context timeout that is shorter than the total retry budget.
	parentCtx, cancel := context.WithTimeout(context.Background(), t.retryConfig.TotalRetryBudget-100*time.Millisecond)
	defer cancel()
	apiCall := func(ctx context.Context) (string, error) {
		callCount++
		// This will always fail with a retryable error.
		return "", status.Error(codes.Unavailable, "server unavailable")
	}

	// Act
	result, err := ExecuteWithRetry(parentCtx, t.retryConfig, "testOp", "testReq", apiCall)

	// Assert
	assert.Error(t.T(), err)
	assert.Empty(t.T(), result)
	assert.ErrorIs(t.T(), err, context.DeadlineExceeded, "The error should be from the parent context's timeout")
}
