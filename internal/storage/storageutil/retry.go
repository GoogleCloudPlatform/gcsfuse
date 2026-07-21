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
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"golang.org/x/oauth2"
)

type retryTokenSource struct {
	base        oauth2.TokenSource
	retryConfig *RetryConfig
}

func (ts *retryTokenSource) Token() (*oauth2.Token, error) {
	apiCall := func(attemptCtx context.Context) (*oauth2.Token, error) {
		tok, err := ts.base.Token()
		if err == nil && tok != nil {
			tokCopy := *tok
			// Force the token to expire in 30 seconds for testing purposes.
			tokCopy.Expiry = time.Now().Add(30 * time.Second)
			return &tokCopy, nil
		}
		return tok, err
	}
	return ExecuteWithCustomShouldRetry(context.Background(), ts.retryConfig, "TokenSource.Token", "token", uuid.NewString(), apiCall, ShouldRetryWithoutLogging)
}

// WrapTokenSource wraps an oauth2.TokenSource with retry logic using ExecuteWithCustomShouldRetry.
func WrapTokenSource(config *StorageClientConfig, base oauth2.TokenSource) oauth2.TokenSource {
	if base == nil {
		return nil
	}
	retryConfig := NewRetryConfig(config)
	return &retryTokenSource{
		base:        base,
		retryConfig: retryConfig,
	}
}

const (
	// Default retry parameters.
	DefaultRetryDeadline    = 30 * time.Second
	DefaultTotalRetryBudget = 100 * time.Hour
	DefaultInitialBackoff   = 1 * time.Second
)

// exponentialBackoffConfig is config parameters
// needed to create an exponentialBackoff.
type exponentialBackoffConfig struct {
	// initial duration for next backoff.
	initial time.Duration
	// max duration for next backoff.
	max time.Duration
	// The rate at which the backoff duration should grow
	// over subsequent calls to next().
	multiplier float64
}

// exponentialBackoff holds the duration parameters for exponential backoff.
type exponentialBackoff struct {
	// config used to create this backoff object.
	config exponentialBackoffConfig
	// Duration for next backoff. Capped at max. Returned by next().
	next time.Duration
}

// newExponentialBackoff returns a new exponentialBackoff given
// the config for it.
func newExponentialBackoff(config *exponentialBackoffConfig) *exponentialBackoff {
	return &exponentialBackoff{
		config: *config,
		next:   config.initial,
	}
}

// nextDuration returns the next backoff duration.
func (b *exponentialBackoff) nextDuration() time.Duration {
	next := b.next
	b.next = min(b.config.max, time.Duration(float64(b.next)*b.config.multiplier))
	return next
}

// waitWithJitter waits for the next backoff duration with added jitter.
// The jitter adds randomness to the backoff duration to prevent the thundering herd problem.
// This is similar to how gax-retries backoff after each failed retry.
func (b *exponentialBackoff) waitWithJitter(ctx context.Context) error {
	// If the context is already cancelled, return immediately.
	if err := ctx.Err(); err != nil {
		return err
	}

	nextDuration := b.nextDuration()
	jitteryBackoffDuration := time.Duration(1 + rand.Int63n(max(1, int64(nextDuration))))
	timer := time.NewTimer(jitteryBackoffDuration)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// RetryConfig holds configuration for retrying an operation.
type RetryConfig struct {
	// Time-limit on every individual retry attempt.
	RetryDeadline time.Duration
	// Total duration allowed across all the attempts.
	TotalRetryBudget time.Duration
	// Max attempts to make for the operation.
	MaxAttempts int
	// Config for managing backoff durations in-between retry attempts.
	BackoffConfig exponentialBackoffConfig
}

// NewCustomRetryConfig creates a new RetryConfig with custom retry timeout parameters.
func NewCustomRetryConfig(clientConfig *StorageClientConfig, retryDeadline, totalRetryBudget time.Duration) *RetryConfig {
	return &RetryConfig{
		RetryDeadline:    retryDeadline,
		TotalRetryBudget: totalRetryBudget,
		MaxAttempts:      clientConfig.MaxRetryAttempts,
		BackoffConfig: exponentialBackoffConfig{
			initial:    DefaultInitialBackoff,
			max:        clientConfig.MaxRetrySleep,
			multiplier: clientConfig.RetryMultiplier,
		},
	}
}

// NewRetryConfig creates a new RetryConfig using the standard defaults for
// deadline and total budget, combined with the user-provided clientConfig.
func NewRetryConfig(clientConfig *StorageClientConfig) *RetryConfig {
	return NewCustomRetryConfig(clientConfig, DefaultRetryDeadline, DefaultTotalRetryBudget)
}

// NewRetryConfigForTesting creates a RetryConfig with custom parameters for testing.
// It is intended for use in tests of other packages (like package storage) to avoid slow tests.
func NewRetryConfigForTesting(retryDeadline, totalRetryBudget, initialBackoff, maxRetrySleep time.Duration, retryMultiplier float64, maxAttempts int) *RetryConfig {
	return &RetryConfig{
		RetryDeadline:    retryDeadline,
		TotalRetryBudget: totalRetryBudget,
		MaxAttempts:      maxAttempts,
		BackoffConfig: exponentialBackoffConfig{
			initial:    initialBackoff,
			max:        maxRetrySleep,
			multiplier: retryMultiplier,
		},
	}
}

// ExecuteWithCustomShouldRetryAtLogLevel encapsulates the retry logic over a given operation.
// It performs time-bound, exponential backoff retries for a given API call.
// It is expected that the given apiCall returns a structure, and not an HTTP response,
// so that it does not leave behind any trace of a pending operation on server.
// It also has an option to control the log level of the initial attempt log,
// while subsequent retries are always logged at Warning level.
// It also accepts a custom shouldRetry predicate function.
func ExecuteWithCustomShouldRetryAtLogLevel[T any](
	ctx context.Context,
	config *RetryConfig,
	operationName string,
	reqDescription string,
	requestID string,
	apiCall func(attemptCtx context.Context) (T, error),
	shouldRetry func(err error) bool,
	logLevel slog.Level, // Used to log the initial attempt at the supplied log level. Subsequent retries are logged at Warning level.
) (T, error) {
	var zero T
	// If the context is already cancelled, return immediately.
	if err := ctx.Err(); err != nil {
		return zero, err
	}

	parentCtx, cancel := context.WithTimeout(ctx, config.TotalRetryBudget)
	defer cancel()

	// Create a new backoff controller specific to this api call.
	backoff := newExponentialBackoff(&config.BackoffConfig)
	var lastErr error
	for attemptNum := 1; ; attemptNum++ {
		attemptCtx, attemptCancel := context.WithTimeout(parentCtx, config.RetryDeadline)
		if attemptNum == 1 {
			logger.GetLogFHandler(logLevel)("Calling %s for %q: InvocationID: %s, Attempt: %d, with deadline=%v", operationName, reqDescription, requestID, attemptNum, config.RetryDeadline)
		} else {
			logger.GetLogFHandler(logger.LevelWarn)("Retrying %s for %q: InvocationID: %s, Attempt: %d, due to error: %v", operationName, reqDescription, requestID, attemptNum, lastErr)
		}

		result, err := apiCall(attemptCtx)
		lastErr = err
		// Cancel attemptCtx after it is no longer needed, to free up its resources.
		attemptCancel()

		if err == nil {
			return result, nil
		}

		if config.MaxAttempts > 0 && attemptNum >= config.MaxAttempts {
			return zero, fmt.Errorf("%s for %q failed: InvocationID: %s, Attempt: %d, MaxAttempts: %d, with error: %w", operationName, reqDescription, requestID, attemptNum, config.MaxAttempts, err)
		}

		// If the error is not retryable, return it immediately.
		if !shouldRetry(err) {
			return zero, fmt.Errorf("%s for %q failed: InvocationID: %s, Attempt: %d, with error: %w", operationName, reqDescription, requestID, attemptNum, err)
		}

		// If the parent context is cancelled/timed-out, we should stop retrying.
		if parentCtx.Err() != nil {
			return zero, fmt.Errorf("%s for %q failed: InvocationID: %s, Attempt: %d, (last server/client error = %v), with error: %w", operationName, reqDescription, requestID, attemptNum, err, parentCtx.Err())
		}

		// Do a jittery backoff after each retry.
		parentCtxErr := backoff.waitWithJitter(parentCtx)
		if parentCtxErr != nil {
			return zero, fmt.Errorf("%s for %q failed: InvocationID: %s, Attempt: %d, (last server/client error = %v), with error: %w", operationName, reqDescription, requestID, attemptNum, err, parentCtxErr)
		}
	}
}

// ExecuteWithCustomShouldRetry retries a given operation using a custom shouldRetry predicate, logging the initial attempt at trace level.
func ExecuteWithCustomShouldRetry[T any](
	ctx context.Context,
	config *RetryConfig,
	operationName string,
	reqDescription string,
	requestID string,
	apiCall func(attemptCtx context.Context) (T, error),
	shouldRetry func(err error) bool,
) (T, error) {
	return ExecuteWithCustomShouldRetryAtLogLevel(ctx, config, operationName, reqDescription, requestID, apiCall, shouldRetry, logger.LevelTrace)
}

// ExecuteWithRetryAtLogLevel encapsulates the retry logic over a given operation.
// It performs time-bound, exponential backoff retries for a given API call.
// It is expected that the given apiCall returns a structure, and not an HTTP response,
// so that it does not leave behind any trace of a pending operation on server.
// It also has an option to control the log level of the initial attempt log,
// while subsequent retries are always logged at Warning level.
func ExecuteWithRetryAtLogLevel[T any](
	ctx context.Context,
	config *RetryConfig,
	operationName string,
	reqDescription string,
	requestID string,
	apiCall func(attemptCtx context.Context) (T, error),
	logLevel slog.Level, // Used to log the initial attempt at the supplied log level. Subsequent retries are logged at Warning level.
) (T, error) {
	return ExecuteWithCustomShouldRetryAtLogLevel(ctx, config, operationName, reqDescription, requestID, apiCall, ShouldRetryWithoutLogging, logLevel)
}

// ExecuteWithRetry retries a given operation, logging the initial attempt at trace level.
func ExecuteWithRetry[T any](
	ctx context.Context,
	config *RetryConfig,
	operationName string,
	reqDescription string,
	requestID string,
	apiCall func(attemptCtx context.Context) (T, error),
) (T, error) {
	return ExecuteWithCustomShouldRetry(ctx, config, operationName, reqDescription, requestID, apiCall, ShouldRetryWithoutLogging)
}
