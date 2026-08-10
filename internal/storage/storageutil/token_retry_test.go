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

func TestRetryingTokenProvider_SuccessOnFirstAttempt(t *testing.T) {
	mock := &mockTokenProvider{
		tokens: []*auth.Token{{Value: "valid-token"}},
	}
	provider := &retryingTokenProvider{
		base:        mock,
		retryConfig: fastRetryConfig(),
	}

	token, err := provider.Token(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, token)
	assert.Equal(t, "valid-token", token.Value)
	assert.Equal(t, 1, mock.calls)
}

func TestRetryingTokenProvider_RetryOnTransientMDSError_Success(t *testing.T) {
	transientErr := &oauth2.RetrieveError{
		Response: &http.Response{StatusCode: http.StatusServiceUnavailable},
	}
	mock := &mockTokenProvider{
		errs:   []error{transientErr, transientErr, nil},
		tokens: []*auth.Token{nil, nil, {Value: "recovered-token"}},
	}
	provider := &retryingTokenProvider{
		base:        mock,
		retryConfig: fastRetryConfig(),
	}

	token, err := provider.Token(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, token)
	assert.Equal(t, "recovered-token", token.Value)
	assert.Equal(t, 3, mock.calls)
}

func TestRetryingTokenProvider_NonTransientError_NoRetry(t *testing.T) {
	permanentErr := errors.New("permanent auth failure")
	mock := &mockTokenProvider{
		errs: []error{permanentErr},
	}
	provider := &retryingTokenProvider{
		base:        mock,
		retryConfig: fastRetryConfig(),
	}

	token, err := provider.Token(context.Background())

	assert.Error(t, err)
	assert.Nil(t, token)
	assert.Equal(t, 1, mock.calls)
}

func TestRetryingTokenProvider_ExhaustedAttempts_Fails(t *testing.T) {
	transientErr := &metadata.Error{Code: 500}
	mock := &mockTokenProvider{
		errs: []error{transientErr, transientErr, transientErr, transientErr},
	}
	provider := &retryingTokenProvider{
		base:        mock,
		retryConfig: fastRetryConfig(),
	}

	token, err := provider.Token(context.Background())

	assert.Error(t, err)
	assert.Nil(t, token)
	assert.Equal(t, 3, mock.calls)
}

func TestRetryingTokenSource_SuccessOnFirstAttempt(t *testing.T) {
	mock := &mockTokenSource{
		tokens: []*oauth2.Token{{AccessToken: "valid-token"}},
	}
	ts := &retryingTokenSource{
		base:        mock,
		retryConfig: fastRetryConfig(),
	}

	token, err := ts.Token()

	assert.NoError(t, err)
	assert.NotNil(t, token)
	assert.Equal(t, "valid-token", token.AccessToken)
	assert.Equal(t, 1, mock.calls)
}

func TestRetryingTokenSource_RetryOnTransientMDSError_Success(t *testing.T) {
	transientErr := &metadata.Error{Code: 503}
	mock := &mockTokenSource{
		errs:   []error{transientErr, nil},
		tokens: []*oauth2.Token{nil, {AccessToken: "recovered-token"}},
	}
	ts := &retryingTokenSource{
		base:        mock,
		retryConfig: fastRetryConfig(),
	}

	token, err := ts.Token()

	assert.NoError(t, err)
	assert.NotNil(t, token)
	assert.Equal(t, "recovered-token", token.AccessToken)
	assert.Equal(t, 2, mock.calls)
}

func TestRetryingTokenSource_NonTransientError_NoRetry(t *testing.T) {
	permanentErr := errors.New("permanent auth failure")
	mock := &mockTokenSource{
		errs: []error{permanentErr},
	}
	ts := &retryingTokenSource{
		base:        mock,
		retryConfig: fastRetryConfig(),
	}

	token, err := ts.Token()

	assert.Error(t, err)
	assert.Nil(t, token)
	assert.Equal(t, 1, mock.calls)
}

func TestRetryingTokenSource_ExhaustedAttempts_Fails(t *testing.T) {
	transientErr := &oauth2.RetrieveError{
		Response: &http.Response{StatusCode: http.StatusTooManyRequests},
	}
	mock := &mockTokenSource{
		errs: []error{transientErr, transientErr, transientErr, transientErr},
	}
	ts := &retryingTokenSource{
		base:        mock,
		retryConfig: fastRetryConfig(),
	}

	token, err := ts.Token()

	assert.Error(t, err)
	assert.Nil(t, token)
	assert.Equal(t, 3, mock.calls)
}

func TestRetryingTokenSource_CancelledContext_FastFail(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	mock := &mockTokenSource{
		tokens: []*oauth2.Token{{AccessToken: "valid-token"}},
	}
	ts := &retryingTokenSource{
		ctx:         ctx,
		base:        mock,
		retryConfig: fastRetryConfig(),
	}

	token, err := ts.Token()

	assert.Error(t, err)
	assert.Nil(t, token)
	assert.True(t, errors.Is(err, context.Canceled))
}
