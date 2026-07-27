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

	"github.com/google/uuid"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"golang.org/x/oauth2"
)

type retryingTokenSource struct {
	ctx         context.Context
	wrapped     oauth2.TokenSource
	retryConfig *RetryConfig
}

// NewRetryingTokenSource returns an oauth2.TokenSource wrapper that retries
// transient OAuth2/STS token acquisition and Metadata Server readiness errors
// using exponential backoff driven by ShouldRetryOnOAuthOrMDSError predicate.
func NewRetryingTokenSource(wrapped oauth2.TokenSource, retryConfig *RetryConfig) oauth2.TokenSource {
	if wrapped == nil {
		return nil
	}
	if retryConfig == nil {
		return wrapped
	}
	return &retryingTokenSource{
		ctx:         context.Background(),
		wrapped:     wrapped,
		retryConfig: retryConfig,
	}
}

func (r *retryingTokenSource) Token() (*oauth2.Token, error) {
	apiCall := func(ctx context.Context) (*oauth2.Token, error) {
		return r.wrapped.Token()
	}
	return ExecuteWithCustomShouldRetryAtLogLevel(
		r.ctx,
		r.retryConfig,
		"TokenSource.Token",
		"auth token fetch",
		uuid.NewString(),
		apiCall,
		ShouldRetryOnOAuthOrMDSError,
		logger.LevelDebug,
	)
}
