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

	"cloud.google.com/go/auth"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

// retryingTokenProvider wraps an auth.TokenProvider to retry transient MDS/OAuth errors.
type retryingTokenProvider struct {
	base        auth.TokenProvider
	retryConfig *RetryConfig
}

func (r *retryingTokenProvider) Token(ctx context.Context) (*auth.Token, error) {
	apiCall := func(attemptCtx context.Context) (*auth.Token, error) {
		return r.base.Token(attemptCtx)
	}

	return ExecuteWithCustomShouldRetry(
		ctx,
		r.retryConfig,
		"TokenProvider.Token",
		"token-refresh",
		uuid.NewString(),
		apiCall,
		ShouldRetryOnMount,
	)
}

// retryingTokenSource wraps an oauth2.TokenSource to retry transient MDS/OAuth errors.
type retryingTokenSource struct {
	base        oauth2.TokenSource
	retryConfig *RetryConfig
}

func (r *retryingTokenSource) Token() (*oauth2.Token, error) {
	apiCall := func(_ context.Context) (*oauth2.Token, error) {
		return r.base.Token()
	}

	return ExecuteWithCustomShouldRetry(
		context.Background(),
		r.retryConfig,
		"TokenSource.Token",
		"token-refresh",
		uuid.NewString(),
		apiCall,
		ShouldRetryOnMount,
	)
}
