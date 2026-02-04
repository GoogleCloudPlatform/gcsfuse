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
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsRetriableNFSError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "ESTALE is retriable",
			err:      syscall.ESTALE,
			expected: true,
		},
		{
			name:     "EIO is retriable",
			err:      syscall.EIO,
			expected: true,
		},
		{
			name:     "EAGAIN is retriable",
			err:      syscall.EAGAIN,
			expected: true,
		},
		{
			name:     "ETIMEDOUT is retriable",
			err:      syscall.ETIMEDOUT,
			expected: true,
		},
		{
			name:     "EBUSY is retriable",
			err:      syscall.EBUSY,
			expected: true,
		},
		{
			name:     "ENOLCK is retriable",
			err:      syscall.ENOLCK,
			expected: true,
		},
		{
			name:     "ENOENT is not retriable",
			err:      syscall.ENOENT,
			expected: false,
		},
		{
			name:     "EPERM is not retriable",
			err:      syscall.EPERM,
			expected: false,
		},
		{
			name:     "EACCES is not retriable",
			err:      syscall.EACCES,
			expected: false,
		},
		{
			name:     "generic error is not retriable",
			err:      errors.New("generic error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetriableNFSError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Table-driven tests for both RetryOnNFSError and RetryOnAnyError
func TestRetry_SuccessFirstAttempt(t *testing.T) {
	config := RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	tests := []struct {
		name    string
		retryFn func(string, func() error, RetryConfig) error
	}{
		{"RetryOnNFSError", RetryOnNFSError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			callCount := 0

			// Act
			err := tt.retryFn("test_operation", func() error {
				callCount++
				return nil
			}, config)

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, 1, callCount, "Should succeed on first attempt")
		})
	}
}

func TestRetry_SuccessAfterRetries(t *testing.T) {
	config := RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    1 * time.Millisecond,
		MaxBackoff:        10 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	tests := []struct {
		name    string
		retryFn func(string, func() error, RetryConfig) error
		errFn   func(attempt int) error
	}{
		{
			name:    "RetryOnNFSError with ESTALE",
			retryFn: RetryOnNFSError,
			errFn: func(attempt int) error {
				if attempt < 3 {
					return syscall.ESTALE
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			callCount := 0

			// Act
			err := tt.retryFn("test_operation", func() error {
				callCount++
				return tt.errFn(callCount)
			}, config)

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, 3, callCount, "Should succeed on 3rd attempt")
		})
	}
}

func TestRetry_ExhaustsRetries(t *testing.T) {
	config := RetryConfig{
		MaxRetries:        2,
		InitialBackoff:    1 * time.Millisecond,
		MaxBackoff:        10 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	tests := []struct {
		name          string
		retryFn       func(string, func() error, RetryConfig) error
		err           error
		expectedCalls int
	}{
		{
			name:          "RetryOnNFSError exhausts retries",
			retryFn:       RetryOnNFSError,
			err:           syscall.ESTALE,
			expectedCalls: 3, // initial + 2 retries
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			callCount := 0

			// Act
			err := tt.retryFn("test_operation", func() error {
				callCount++
				return tt.err
			}, config)

			// Assert
			assert.Error(t, err)
			assert.Equal(t, tt.err, err)
			assert.Equal(t, tt.expectedCalls, callCount)
		})
	}
}

func TestRetryOnNFSError_NonRetriableError(t *testing.T) {
	// Arrange
	config := RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}
	callCount := 0
	nonRetriableErr := syscall.ENOENT

	// Act
	err := RetryOnNFSError("test_operation", func() error {
		callCount++
		return nonRetriableErr
	}, config)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, nonRetriableErr, err)
	assert.Equal(t, 1, callCount, "Should not retry non-retriable errors")
}

func TestRetry_ExponentialBackoff(t *testing.T) {
	config := RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        1 * time.Second,
		BackoffMultiplier: 2.0,
	}

	tests := []struct {
		name    string
		retryFn func(string, func() error, RetryConfig) error
		err     error
	}{
		{
			name:    "RetryOnNFSError",
			retryFn: RetryOnNFSError,
			err:     syscall.ESTALE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			timestamps := []time.Time{}

			// Act
			_ = tt.retryFn("test_operation", func() error {
				timestamps = append(timestamps, time.Now())
				return tt.err
			}, config)

			// Assert
			require.Len(t, timestamps, 4, "Should have 4 timestamps (initial + 3 retries)")

			// Verify backoff is increasing (with some tolerance for jitter ±25%)
			for i := 1; i < len(timestamps); i++ {
				interval := timestamps[i].Sub(timestamps[i-1])
				expectedBackoff := config.InitialBackoff * time.Duration(1<<uint(i-1))
				minBackoff := time.Duration(float64(expectedBackoff) * 0.70)
				maxBackoff := time.Duration(float64(expectedBackoff) * 1.50)

				assert.True(t, interval >= minBackoff,
					"Retry %d: interval %v should be >= %v", i, interval, minBackoff)
				assert.True(t, interval <= maxBackoff,
					"Retry %d: interval %v should be <= %v", i, interval, maxBackoff)
			}
		})
	}
}

func TestRetryOnNFSError_MaxBackoffCap(t *testing.T) {
	// Arrange
	config := RetryConfig{
		MaxRetries:        5,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        150 * time.Millisecond, // Cap at 150ms
		BackoffMultiplier: 2.0,
	}
	callCount := 0
	timestamps := []time.Time{}

	// Act
	_ = RetryOnNFSError("test_operation", func() error {
		timestamps = append(timestamps, time.Now())
		callCount++
		return syscall.ESTALE
	}, config)

	// Assert
	require.Len(t, timestamps, 6, "Should have 6 timestamps")

	// After a few retries, backoff should be capped at MaxBackoff
	// Retry 0: 100ms
	// Retry 1: 200ms -> capped to 150ms
	// Retry 2+: 150ms (capped)

	if len(timestamps) >= 4 {
		diff2 := timestamps[3].Sub(timestamps[2])
		diff3 := timestamps[4].Sub(timestamps[3])

		// Both should be around MaxBackoff (with jitter)
		assert.Less(t, diff2, 200*time.Millisecond, "Backoff should be capped")
		assert.Less(t, diff3, 200*time.Millisecond, "Backoff should be capped")
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	// Act
	config := DefaultRetryConfig()

	// Assert
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 10*time.Millisecond, config.InitialBackoff)
	assert.Equal(t, 1*time.Second, config.MaxBackoff)
	assert.Equal(t, 2.0, config.BackoffMultiplier)
}

func TestRetryOnNFSError_DifferentRetriableErrors(t *testing.T) {
	retriableErrors := []syscall.Errno{
		syscall.ESTALE,
		syscall.EIO,
		syscall.EAGAIN,
		syscall.ETIMEDOUT,
		syscall.EBUSY,
		syscall.ENOLCK,
	}

	config := RetryConfig{
		MaxRetries:        1,
		InitialBackoff:    1 * time.Millisecond,
		MaxBackoff:        10 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	for _, errCode := range retriableErrors {
		t.Run(errCode.Error(), func(t *testing.T) {
			callCount := 0

			err := RetryOnNFSError("test_operation", func() error {
				callCount++
				if callCount == 1 {
					return errCode
				}
				return nil
			}, config)

			assert.NoError(t, err, "Should retry and succeed for %v", errCode)
			assert.Equal(t, 2, callCount, "Should make 2 attempts")
		})
	}
}
