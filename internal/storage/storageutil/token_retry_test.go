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
	"net/http"
	"testing"
	"time"

	"cloud.google.com/go/auth"
	"cloud.google.com/go/compute/metadata"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

type mockTokenProvider struct {
	tokens []*auth.Token
	errs   []error
	calls  int
}

func (m *mockTokenProvider) Token(ctx context.Context) (*auth.Token, error) {
	idx := m.calls
	m.calls++
	if idx < len(m.errs) && m.errs[idx] != nil {
		return nil, m.errs[idx]
	}
	if idx < len(m.tokens) && m.tokens[idx] != nil {
		return m.tokens[idx], nil
	}
	return &auth.Token{Value: "default-token"}, nil
}

type mockTokenSource struct {
	tokens []*oauth2.Token
	errs   []error
	calls  int
}

func (m *mockTokenSource) Token() (*oauth2.Token, error) {
	idx := m.calls
	m.calls++
	if idx < len(m.errs) && m.errs[idx] != nil {
		return nil, m.errs[idx]
	}
	if idx < len(m.tokens) && m.tokens[idx] != nil {
		return m.tokens[idx], nil
	}
	return &oauth2.Token{AccessToken: "default-token"}, nil
}

func fastRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:   3,
		RetryDeadline: 1 * time.Second,
		BackoffConfig: exponentialBackoffConfig{
			initial:    1 * time.Millisecond,
			max:        5 * time.Millisecond,
			multiplier: 1.5,
		},
	}
}

func TestRetryingTokenProvider(t *testing.T) {
	transientErr := &oauth2.RetrieveError{
		Response: &http.Response{StatusCode: http.StatusServiceUnavailable},
	}
	metadata500 := &metadata.Error{Code: 500}
	permanentErr := errors.New("permanent auth failure")

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	testCases := []struct {
		name          string
		ctx           context.Context
		mockTokens    []*auth.Token
		mockErrors    []error
		expectError   bool
		checkCancel   bool
		expectedToken string
		expectedCalls int
	}{
		{
			name:          "SuccessOnFirstAttempt",
			ctx:           context.Background(),
			mockTokens:    []*auth.Token{{Value: "valid-token"}},
			expectedToken: "valid-token",
			expectedCalls: 1,
		},
		{
			name:          "RecoveryAfterTransientMDSErrors",
			ctx:           context.Background(),
			mockErrors:    []error{transientErr, transientErr, nil},
			mockTokens:    []*auth.Token{nil, nil, {Value: "recovered-token"}},
			expectedToken: "recovered-token",
			expectedCalls: 3,
		},
		{
			name:          "FastFailOnNonTransientError",
			ctx:           context.Background(),
			mockErrors:    []error{permanentErr},
			expectError:   true,
			expectedCalls: 1,
		},
		{
			name:          "ExhaustedRetryAttempts",
			ctx:           context.Background(),
			mockErrors:    []error{metadata500, metadata500, metadata500, metadata500},
			expectError:   true,
			expectedCalls: 3,
		},
		{
			name:          "FastFailOnCancelledContext",
			ctx:           cancelledCtx,
			mockTokens:    []*auth.Token{{Value: "valid-token"}},
			expectError:   true,
			checkCancel:   true,
			expectedCalls: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockTokenProvider{
				tokens: tc.mockTokens,
				errs:   tc.mockErrors,
			}
			provider := &retryingTokenProvider{
				base:        mock,
				retryConfig: fastRetryConfig(),
			}

			token, err := provider.Token(tc.ctx)

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, token)
				if tc.checkCancel {
					assert.True(t, errors.Is(err, context.Canceled))
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, token)
				assert.Equal(t, tc.expectedToken, token.Value)
			}
			assert.Equal(t, tc.expectedCalls, mock.calls)
		})
	}
}

func TestRetryingTokenSource(t *testing.T) {
	transientErr := &oauth2.RetrieveError{
		Response: &http.Response{StatusCode: http.StatusTooManyRequests},
	}
	metadata503 := &metadata.Error{Code: 503}
	permanentErr := errors.New("permanent auth failure")

	testCases := []struct {
		name          string
		mockTokens    []*oauth2.Token
		mockErrors    []error
		expectError   bool
		expectedToken string
		expectedCalls int
	}{
		{
			name:          "SuccessOnFirstAttempt",
			mockTokens:    []*oauth2.Token{{AccessToken: "valid-token"}},
			expectedToken: "valid-token",
			expectedCalls: 1,
		},
		{
			name:          "RecoveryAfterTransientMDSError",
			mockErrors:    []error{metadata503, nil},
			mockTokens:    []*oauth2.Token{nil, {AccessToken: "recovered-token"}},
			expectedToken: "recovered-token",
			expectedCalls: 2,
		},
		{
			name:          "FastFailOnNonTransientError",
			mockErrors:    []error{permanentErr},
			expectError:   true,
			expectedCalls: 1,
		},
		{
			name:          "ExhaustedRetryAttempts",
			mockErrors:    []error{transientErr, transientErr, transientErr, transientErr},
			expectError:   true,
			expectedCalls: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockTokenSource{
				tokens: tc.mockTokens,
				errs:   tc.mockErrors,
			}
			ts := &retryingTokenSource{
				base:        mock,
				retryConfig: fastRetryConfig(),
			}

			token, err := ts.Token()

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, token)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, token)
				assert.Equal(t, tc.expectedToken, token.AccessToken)
			}
			assert.Equal(t, tc.expectedCalls, mock.calls)
		})
	}
}
