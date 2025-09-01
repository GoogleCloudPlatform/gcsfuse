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

	"cloud.google.com/go/storage/control/apiv2/controlpb"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
)

// ExponentialBackoffConfig is config parameters
// needed to create an ExponentialBackoff.
type ExponentialBackoffConfig struct {
	//Initial duration for next backoff.
	Initial time.Duration
	// Max duration for next backoff.
	Max time.Duration
	// The rate at which the backoff duration should grow
	// over subsequent calls to next().
	Multiplier float64
}

// ExponentialBackoff holds the duration parameters for exponential backoff.
type ExponentialBackoff struct {
	// config used to create this backoff object.
	config ExponentialBackoffConfig
	// Duration for next backoff. Capped at max. Returned by next().
	next time.Duration
}

// NewExponentialBackoff returns a new ExponentialBackoff given
// the config for it.
func NewExponentialBackoff(config *ExponentialBackoffConfig) *ExponentialBackoff {
	return &ExponentialBackoff{
		config: *config,
		next:   config.Initial,
	}
}

// nextDuration returns the next backoff duration.
func (b *ExponentialBackoff) nextDuration() time.Duration {
	next := b.next
	b.next = min(b.config.Max, time.Duration(float64(b.next)*b.config.Multiplier))
	return next
}

// WaitWithJitter waits for the next backoff duration with added jitter.
// The jitter adds randomness to the backoff duration to prevent the thundering herd problem.
// This is similar to how gax-retries backoff after each failed retry.
func (b *ExponentialBackoff) WaitWithJitter(ctx context.Context) error {
	nextDuration := b.nextDuration()
	if nextDuration <= 0 {
		// Avoid a panic from rand.Int63n if the duration is not positive.
		// We still check for context cancellation before returning to avoid a busy-loop
		// if the context is already cancelled.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}

	jitteryBackoffDuration := time.Duration(1 + rand.Int63n(int64(nextDuration)))
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
	BackoffConfig ExponentialBackoffConfig
}

// ExecuteWithRetry encapsulates the retry logic for control client operations.
// It performs time-bound, exponential backoff retries for a given API call.
// It is expected that the given apiCall returns a structure, and not an HTTP response,
// so that it does not leave behind any trace of a pending operation on server.
func ExecuteWithRetry[T any](
	ctx context.Context,
	config RetryConfig,
	operationName string,
	reqDescription string,
	apiCall func(attemptCtx context.Context) (T, error),
) (T, error) {
	var zero T

	parentCtx, cancel := context.WithTimeout(ctx, config.TotalRetryBudget)
	defer cancel()

	// Create a new backoff controller specific to this api call.
	backoff := NewExponentialBackoff(&config.BackoffConfig)
	for {
		attemptCtx, attemptCancel := context.WithTimeout(parentCtx, config.RetryDeadline)

		logger.Tracef("Calling %s for %q with deadline=%v ...", operationName, reqDescription, config.RetryDeadline)
		result, err := apiCall(attemptCtx)
		// Cancel attemptCtx after it is no longer needed to free up its resources.
		attemptCancel()

		if err == nil {
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
		parentCtxErr := backoff.WaitWithJitter(parentCtx)
		if parentCtxErr != nil {
			return zero, fmt.Errorf("%s for %q failed after multiple retries (last server/client error = %v): %w", operationName, reqDescription, err, parentCtxErr)
		}
	}
}

func CreateFolderReqDescription(req *controlpb.CreateFolderRequest) string {
	return fmt.Sprintf("%q in %q", req.FolderId, req.Parent)
}

func RenameFolderReqDescription(req *controlpb.RenameFolderRequest) string {
	return fmt.Sprintf("%q to %q", req.Name, req.DestinationFolderId)
}
