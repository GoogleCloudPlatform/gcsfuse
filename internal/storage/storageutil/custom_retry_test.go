// Copyright 2023 Google LLC
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
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/url"
	"os"
	"sync"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/googleapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestShouldRetryReturnsTrueWithGoogleApiError(t *testing.T) {
	// 401
	var err401 = googleapi.Error{
		Code: 401,
		Body: "Invalid Credential",
	}
	// 50x error
	var err502 = googleapi.Error{
		Code: 502,
	}
	// 429 - rate limiting error
	var err429 = googleapi.Error{
		Code: 429,
		Body: "API rate limit exceeded",
	}

	assert.Equal(t, true, ShouldRetry(context.Background(), &err401))
	assert.Equal(t, true, ShouldRetry(context.Background(), &err502))
	assert.Equal(t, true, ShouldRetry(context.Background(), &err429))
}

func TestShouldRetryReturnsFalseWithGoogleApiError400(t *testing.T) {
	// 400 - bad request
	var err400 = googleapi.Error{
		Code: 400,
	}

	assert.Equal(t, false, ShouldRetry(context.Background(), &err400))
}

func TestShouldRetryReturnsTrueWithUnexpectedEOFError(t *testing.T) {
	assert.Equal(t, true, ShouldRetry(context.Background(), io.ErrUnexpectedEOF))
}

func TestShouldRetryReturnsTrueWithNetworkError(t *testing.T) {
	assert.Equal(t, true, ShouldRetry(context.Background(), net.ErrClosed))
}

func TestShouldRetryReturnsTrueForConnectionRefusedAndResetErrors(t *testing.T) {
	testCases := []struct {
		name           string
		err            error
		expectedResult bool
	}{
		{
			name:           "URL Error - Connection Refused",
			err:            &url.Error{Err: errors.New("connection refused")},
			expectedResult: true,
		},
		{
			name:           "URL Error - Connection Reset",
			err:            &url.Error{Err: errors.New("connection reset")},
			expectedResult: true,
		},
		{
			name:           "URL Error - connection reset by peer",
			err:            &url.Error{Err: errors.New("connection reset by peer")},
			expectedResult: true,
		},
		{
			name:           "URL Error - connection refused by peer",
			err:            &url.Error{Err: errors.New("connection refused by peer")},
			expectedResult: true,
		},
		{
			name:           "Op Error - Connection Refused",
			err:            &net.OpError{Err: errors.New("connection refused")},
			expectedResult: true,
		},
		{
			name:           "Op Error - Connection Reset",
			err:            &net.OpError{Err: errors.New("connection reset")},
			expectedResult: true,
		},
		{
			name:           "Op Error - connection reset by peer",
			err:            &net.OpError{Err: errors.New("connection reset by peer")},
			expectedResult: true,
		},
		{
			name:           "Op Error - connection refused by peer",
			err:            &net.OpError{Err: errors.New("connection refused by peer")},
			expectedResult: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualResult := ShouldRetry(context.Background(), tc.err)
			assert.Equal(t, tc.expectedResult, actualResult)
		})
	}
}

func TestShouldRetryReturnsTrueForUnauthenticatedGrpcErrors(t *testing.T) {
	testCases := []struct {
		name           string
		err            error
		expectedResult bool
	}{
		{
			name:           "UNAUTHENTICATED",
			err:            status.Error(codes.Unauthenticated, "Request had invalid authentication credentials. Expected OAuth 2 access token, login cookie or other valid authentication credential. See https://developers.google.com/identity/sign-in/web/devconsole-project."),
			expectedResult: true,
		},
		{
			name:           "PERMISSION_DENIED",
			err:            status.Error(codes.PermissionDenied, "unauthorized"),
			expectedResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualResult := ShouldRetry(context.Background(), tc.err)
			assert.Equal(t, tc.expectedResult, actualResult)
		})
	}
}

func TestShouldRetryWithoutLogging(t *testing.T) {
	testCases := []struct {
		name           string
		err            error
		expectedResult bool
	}{
		{
			name: "401 error - retryable",
			err: &googleapi.Error{
				Code: 401,
				Body: "Invalid Credential",
			},
			expectedResult: true,
		},
		{
			name:           "Unauthenticated error - retryable",
			err:            status.Error(codes.Unauthenticated, "unauthenticated"),
			expectedResult: true,
		},
		{
			name: "400 error - non-retryable",
			err: &googleapi.Error{
				Code: 400,
			},
			expectedResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf logBuffer
			logger.SetOutput(&buf)
			defer logger.SetOutput(os.Stdout)

			// Act
			actualResult := ShouldRetryWithoutLogging(tc.err)

			// Assert
			assert.Equal(t, tc.expectedResult, actualResult)
			assert.Empty(t, buf.String())
		})
	}
}

func TestDetermineRetryAction(t *testing.T) {
	// Arrange
	testCases := []struct {
		name     string
		err      error
		expected retryAction
	}{
		{
			name:     "NilError",
			err:      nil,
			expected: noRetry,
		},
		{
			name:     "GoogleApiError400",
			err:      &googleapi.Error{Code: 400},
			expected: noRetry,
		},
		{
			name:     "GoogleApiError401",
			err:      &googleapi.Error{Code: 401},
			expected: retry401,
		},
		{
			name:     "GoogleApiError429",
			err:      &googleapi.Error{Code: 429},
			expected: retryTransient,
		},
		{
			name:     "UnauthenticatedGrpcError",
			err:      status.Error(codes.Unauthenticated, "unauthenticated"),
			expected: retryUnauthenticated,
		},
		{
			name:     "PermissionDeniedGrpcError",
			err:      status.Error(codes.PermissionDenied, "permission denied"),
			expected: noRetry,
		},
		{
			name:     "UnexpectedEOF",
			err:      io.ErrUnexpectedEOF,
			expected: retryTransient,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			actual := determineRetryAction(tc.err)

			// Assert
			assert.Equal(t, tc.expected, actual)
		})
	}
}

// logBuffer is a thread-safe buffer for capturing logs in tests.
type logBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *logBuffer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *logBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestShouldRetryLogsWarning(t *testing.T) {
	// Arrange
	var buf logBuffer
	logger.SetOutput(&buf)
	defer logger.SetOutput(os.Stdout)
	var err401 = &googleapi.Error{
		Code: 401,
		Body: "Invalid Credential",
	}

	// Act
	retry := ShouldRetry(context.Background(), err401)

	// Assert
	assert.True(t, retry)
	assert.Contains(t, buf.String(), "WARNING")
	assert.Contains(t, buf.String(), "Retrying for error-code 401")
}

func TestShouldRetryLogsWarningWithSDKRetryContext(t *testing.T) {
	// Arrange
	var buf logBuffer
	logger.SetOutput(&buf)
	defer logger.SetOutput(os.Stdout)
	var err401 = &googleapi.Error{
		Code:    401,
		Message: "Invalid Credential",
	}

	// Create SDK RetryContext
	retryCtx := &storage.RetryContext{
		Attempt:      3,
		InvocationID: "mock-invocation-id-123",
		Operation:    "GetObject",
		Bucket:       "my-test-bucket",
		Object:       "some/file.txt",
	}

	// Act
	retry := ShouldRetryWithContext(context.Background(), err401, retryCtx)

	// Assert
	assert.True(t, retry)
	logMsg := buf.String()
	t.Logf("Captured Log Message:\n%s", logMsg)
	assert.Contains(t, logMsg, "WARNING")
	assert.Contains(t, logMsg, "Retrying for error-code 401")
	assert.Contains(t, logMsg, "[HTTP Code: 401, Message: ")
	assert.Contains(t, logMsg, "Invalid Credential")
	assert.Contains(t, logMsg, "[Op: GetObject, Bucket: ")
	assert.Contains(t, logMsg, "my-test-bucket")
	assert.Contains(t, logMsg, "some/file.txt")
	assert.Contains(t, logMsg, "Attempt: 3")
	assert.Contains(t, logMsg, "InvocationID: mock-invocation-id-123")
}

type fakeMetricHandle struct {
	metrics.MetricHandle

	gcsRetryCountCalled   bool
	gcsRetryCountInc      int64
	gcsRetryErrorCategory string
}

func (m *fakeMetricHandle) GcsRetryCount(inc int64, val metrics.RetryErrorCategory) {
	m.gcsRetryCountCalled = true
	m.gcsRetryCountInc = inc
	m.gcsRetryErrorCategory = string(val)
}

func TestShouldRetryWithMonitoringForNonRetryableErrors(t *testing.T) {
	testCases := []struct {
		name string
		err  error
	}{
		{
			name: "nil error",
			err:  nil,
		},
		{
			name: "non-retryable error",
			err:  &googleapi.Error{Code: 400},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fakeMetrics := &fakeMetricHandle{
				MetricHandle: metrics.NewNoopMetrics(),
			}

			shouldRetry := ShouldRetryWithMonitoring(context.Background(), tc.err, fakeMetrics)

			assert.False(t, shouldRetry)
			assert.False(t, fakeMetrics.gcsRetryCountCalled)
		})
	}
}

func TestShouldRetryWithMonitoringForRetryableErrors(t *testing.T) {
	retryableErr := &googleapi.Error{Code: 429}

	testCases := []struct {
		name                   string
		err                    error
		expectedMetricCategory string
	}{
		{
			name:                   "retryable error, DeadlineExceeded",
			err:                    context.DeadlineExceeded,
			expectedMetricCategory: "STALLED_READ_REQUEST",
		},
		{
			name:                   "retryable error, not DeadlineExceeded",
			err:                    retryableErr,
			expectedMetricCategory: "OTHER_ERRORS",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fakeMetrics := &fakeMetricHandle{
				MetricHandle: metrics.NewNoopMetrics(),
			}

			shouldRetry := ShouldRetryWithMonitoring(context.Background(), tc.err, fakeMetrics)

			assert.True(t, shouldRetry)
			assert.True(t, fakeMetrics.gcsRetryCountCalled)
			assert.Equal(t, int64(1), fakeMetrics.gcsRetryCountInc)
			assert.Equal(t, tc.expectedMetricCategory, fakeMetrics.gcsRetryErrorCategory)
		})
	}
}
