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
	"math/rand"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
)

const (
	// Default retry parameters.
	DefaultRetryDeadline    = 30 * time.Second
	DefaultTotalRetryBudget = 5 * time.Minute
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
	// Duration waited in previous backoff.
	prev time.Duration
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
	jitteryBackoffDuration := time.Duration(1 + rand.Int63n(int64(nextDuration)))
	// Ensure that the backoff duration goes up at the rate of at least the multiplier.
	jitteryBackoffDuration = max(jitteryBackoffDuration, time.Duration(float64(b.prev)*b.config.multiplier))
	b.prev = jitteryBackoffDuration
	select {
	case <-time.After(jitteryBackoffDuration):
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
	// Config for managing backoff durations in-between retry attempts.
	BackoffConfig exponentialBackoffConfig
}

// NewRetryConfig creates a new RetryConfig.
func NewRetryConfig(clientConfig *StorageClientConfig, retryDeadline, totalRetryBudget, initialBackoff time.Duration) *RetryConfig {
	// TODO: Add checks for non negative value initialization.
	return &RetryConfig{
		RetryDeadline:    retryDeadline,
		TotalRetryBudget: totalRetryBudget,
		BackoffConfig: exponentialBackoffConfig{
			initial:    initialBackoff,
			max:        clientConfig.MaxRetrySleep,
			multiplier: clientConfig.RetryMultiplier,
		},
	}
}

// ExecuteWithRetry encapsulates the retry logic over a given operation.
// It performs time-bound, exponential backoff retries for a given API call.
// It is expected that the given apiCall returns a structure, and not an HTTP response,
// so that it does not leave behind any trace of a pending operation on server.
func ExecuteWithRetry[T any](
	ctx context.Context,
	config *RetryConfig,
	operationName string,
	reqDescription string,
	apiCall func(attemptCtx context.Context) (T, error),
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
	for i := 0; ; i++ {
		attemptCtx, attemptCancel := context.WithTimeout(parentCtx, config.RetryDeadline)

		if i == 0 {
			logger.Tracef("Calling %s request for %q with deadline=%v", operationName, reqDescription, config.RetryDeadline)
		} else {
			logger.Tracef("Retrying %s for %q with deadline=%v ...", operationName, reqDescription, config.RetryDeadline)
		}

		result, err := apiCall(attemptCtx)
		// Cancel attemptCtx after it is no longer needed, to free up its resources.
		attemptCancel()

		if err == nil {
			logger.Tracef("Success %s request for %q with success", operationName, reqDescription)
			return result, nil
		}

		// If the error is not retryable, return it immediately.
		if !ShouldRetry(err) {
			return zero, fmt.Errorf("%s for %q failed with a non-retryable error: %w", operationName, reqDescription, err)
		}

		// If the parent context is cancelled/timed-out, we should stop retrying.
		if parentCtx.Err() != nil {
			return zero, fmt.Errorf("%s for %q failed after multiple retries (last server/client error = %v): %w", operationName, reqDescription, err, parentCtx.Err())
		}

		// Do a jittery backoff after each retry.
		parentCtxErr := backoff.waitWithJitter(parentCtx)
		if parentCtxErr != nil {
			return zero, fmt.Errorf("%s for %q failed after multiple retries (last server/client error = %v): %w", operationName, reqDescription, err, parentCtxErr)
		}
	}
}
