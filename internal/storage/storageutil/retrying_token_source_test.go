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
	"fmt"
	"net/http"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/oauth2"
)

type mockTokenSource struct {
	attempts int32
	errs     []error
	token    *oauth2.Token
}

func (m *mockTokenSource) Token() (*oauth2.Token, error) {
	idx := atomic.AddInt32(&m.attempts, 1) - 1
	if int(idx) < len(m.errs) && m.errs[idx] != nil {
		return nil, m.errs[idx]
	}
	return m.token, nil
}

type RetryingTokenSourceTestSuite struct {
	suite.Suite
}

func TestRetryingTokenSourceTestSuite(t *testing.T) {
	suite.Run(t, new(RetryingTokenSourceTestSuite))
}

func (t *RetryingTokenSourceTestSuite) TestNewRetryingTokenSource_NilInput() {
	assert.Nil(t.T(), NewRetryingTokenSource(nil, &RetryConfig{}))

	mock := &mockTokenSource{token: &oauth2.Token{AccessToken: "test-token"}}
	assert.Equal(t.T(), mock, NewRetryingTokenSource(mock, nil))
}

func (t *RetryingTokenSourceTestSuite) TestRetryingTokenSource_SuccessFirstAttempt() {
	mock := &mockTokenSource{
		token: &oauth2.Token{AccessToken: "valid-access-token"},
	}
	retryConfig := NewRetryConfigForTesting(
		10*time.Millisecond,
		1*time.Millisecond,
		5*time.Millisecond,
		1.5,
		3,
	)

	ts := NewRetryingTokenSource(mock, retryConfig)
	require.NotNil(t.T(), ts)

	token, err := ts.Token()
	require.NoError(t.T(), err)
	assert.Equal(t.T(), "valid-access-token", token.AccessToken)
	assert.Equal(t.T(), int32(1), mock.attempts)
}

func (t *RetryingTokenSourceTestSuite) TestRetryingTokenSource_RetryOnTransientMDSError() {
	mock := &mockTokenSource{
		errs: []error{
			fmt.Errorf("dial tcp 169.254.169.254:80: connect: %w", syscall.ECONNREFUSED),
			&oauth2.RetrieveError{
				Response: &http.Response{StatusCode: http.StatusServiceUnavailable},
			},
		},
		token: &oauth2.Token{AccessToken: "recovered-token"},
	}
	retryConfig := NewRetryConfigForTesting(
		50*time.Millisecond,
		1*time.Millisecond,
		5*time.Millisecond,
		1.5,
		5,
	)

	ts := NewRetryingTokenSource(mock, retryConfig)
	token, err := ts.Token()

	require.NoError(t.T(), err)
	assert.Equal(t.T(), "recovered-token", token.AccessToken)
	assert.Equal(t.T(), int32(3), mock.attempts)
}

func (t *RetryingTokenSourceTestSuite) TestRetryingTokenSource_FailFastOnPermanentAuthError() {
	mock := &mockTokenSource{
		errs: []error{
			&oauth2.RetrieveError{
				Response: &http.Response{StatusCode: http.StatusUnauthorized},
			},
		},
	}
	retryConfig := NewRetryConfigForTesting(
		50*time.Millisecond,
		1*time.Millisecond,
		5*time.Millisecond,
		1.5,
		5,
	)

	ts := NewRetryingTokenSource(mock, retryConfig)
	token, err := ts.Token()

	require.Error(t.T(), err)
	assert.Nil(t.T(), token)
	// Should fail fast after attempt 1 without retrying
	assert.Equal(t.T(), int32(1), mock.attempts)
}

func (t *RetryingTokenSourceTestSuite) TestRetryingTokenSource_ExceedMaxAttempts() {
	mock := &mockTokenSource{
		errs: []error{
			fmt.Errorf("dial tcp 169.254.169.254:80: connect: %w", syscall.ECONNREFUSED),
			fmt.Errorf("dial tcp 169.254.169.254:80: connect: %w", syscall.ECONNREFUSED),
			fmt.Errorf("dial tcp 169.254.169.254:80: connect: %w", syscall.ECONNREFUSED),
		},
	}
	retryConfig := NewRetryConfigForTesting(
		50*time.Millisecond,
		1*time.Millisecond,
		5*time.Millisecond,
		1.5,
		2, // Max 2 attempts
	)

	ts := NewRetryingTokenSource(mock, retryConfig)
	token, err := ts.Token()

	require.Error(t.T(), err)
	assert.Nil(t.T(), token)
	assert.Equal(t.T(), int32(2), mock.attempts)
}
