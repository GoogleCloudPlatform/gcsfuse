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

	"cloud.google.com/go/auth"
	"github.com/google/uuid"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
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

	return ExecuteWithCustomShouldRetryAtLogLevel(
		ctx,
		r.retryConfig,
		"TokenProvider.Token",
		"token-refresh",
		uuid.NewString(),
		apiCall,
		isTransientMDSError,
		logger.LevelInfo,
	)
}

// retryingTokenSource wraps an oauth2.TokenSource to retry transient MDS/OAuth errors.
type retryingTokenSource struct {
	ctx         context.Context
	base        oauth2.TokenSource
	retryConfig *RetryConfig
}

func (r *retryingTokenSource) Token() (*oauth2.Token, error) {
	apiCall := func(_ context.Context) (*oauth2.Token, error) {
		return r.base.Token()
	}

	ctx := r.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	return ExecuteWithCustomShouldRetryAtLogLevel(
		ctx,
		r.retryConfig,
		"Token",
		"token-refresh",
		uuid.NewString(),
		apiCall,
		isTransientMDSError,
		logger.LevelInfo,
	)
}
