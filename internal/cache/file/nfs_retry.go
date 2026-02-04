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

package file

import (
	"errors"
	"math/rand"
	"syscall"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
)

// RetryConfig defines configuration for retry behavior on NFS operations
type RetryConfig struct {
	MaxRetries        int           // Maximum number of retry attempts
	InitialBackoff    time.Duration // Initial backoff duration
	MaxBackoff        time.Duration // Maximum backoff duration
	BackoffMultiplier float64       // Multiplier for exponential backoff
}

// DefaultRetryConfig returns default retry configuration optimized for NFS
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        1 * time.Second,
		BackoffMultiplier: 2.0,
	}
}

// isRetriableNFSError checks if an error should trigger a retry
func isRetriableNFSError(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific syscall errors that are retriable
	var errno syscall.Errno
	if errors.As(err, &errno) {
		switch errno {
		case syscall.ESTALE: // Stale NFS file handle
			return true
		case syscall.EIO: // I/O error (network issues)
			return true
		case syscall.EAGAIN: // Resource temporarily unavailable (EWOULDBLOCK is same on Linux)
			return true
		case syscall.ETIMEDOUT: // Operation timed out
			return true
		case syscall.EBUSY: // Resource busy
			return true
		case syscall.ENOLCK: // No locks available
			return true
		}
	}

	return false
}

// retryWithBackoff is the internal implementation for retry logic with exponential backoff
func retryWithBackoff(operation string, fn func() error, config RetryConfig, checkErrorType bool) error {
	var lastErr error
	backoff := config.InitialBackoff

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		lastErr = fn()

		// Success - no retry needed
		if lastErr == nil {
			if attempt > 0 {
				logger.Infof("Operation '%s' succeeded after %d retries", operation, attempt)
			}
			return nil
		}

		// Check if error is retriable (only if checkErrorType is true)
		if checkErrorType && !isRetriableNFSError(lastErr) {
			// Non-retriable error - fail immediately
			return lastErr
		}

		// Last attempt exhausted
		if attempt == config.MaxRetries {
			logger.Warnf("Operation '%s' failed after %d retries: %v", operation, config.MaxRetries, lastErr)
			return lastErr
		}

		// Log retry attempt
		logger.Tracef("Operation '%s' failed (attempt %d/%d), retrying after %v: %v",
			operation, attempt+1, config.MaxRetries, backoff, lastErr)

		// Sleep with jitter (±25%)
		jitter := time.Duration(float64(backoff) * (0.75 + 0.5*rand.Float64()))
		time.Sleep(jitter)

		// Exponential backoff with cap
		backoff = min(time.Duration(float64(backoff)*config.BackoffMultiplier), config.MaxBackoff)
	}

	return lastErr
}

// RetryOnNFSError executes a function with exponential backoff retry on NFS-specific errors
func RetryOnNFSError(operation string, fn func() error, config RetryConfig) error {
	return retryWithBackoff(operation, fn, config, true)
}
