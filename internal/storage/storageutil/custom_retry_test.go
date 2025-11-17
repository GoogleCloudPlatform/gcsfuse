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
	"context"
	"errors"
	"io"
	"net"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vipnydav/gcsfuse/v3/metrics"
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

	assert.Equal(t, true, ShouldRetry(&err401))
	assert.Equal(t, true, ShouldRetry(&err502))
	assert.Equal(t, true, ShouldRetry(&err429))
}

func TestShouldRetryReturnsFalseWithGoogleApiError400(t *testing.T) {
	// 400 - bad request
	var err400 = googleapi.Error{
		Code: 400,
	}

	assert.Equal(t, false, ShouldRetry(&err400))
}

func TestShouldRetryReturnsTrueWithUnexpectedEOFError(t *testing.T) {
	assert.Equal(t, true, ShouldRetry(io.ErrUnexpectedEOF))
}

func TestShouldRetryReturnsTrueWithNetworkError(t *testing.T) {
	assert.Equal(t, true, ShouldRetry(net.ErrClosed))
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
			actualResult := ShouldRetry(tc.err)
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
			actualResult := ShouldRetry(tc.err)
			assert.Equal(t, tc.expectedResult, actualResult)
		})
	}
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
