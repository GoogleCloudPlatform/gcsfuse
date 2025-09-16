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
	"fmt"

	"cloud.google.com/go/auth/oauth2adapt"
	auth2 "github.com/googlecloudplatform/gcsfuse/v3/internal/auth"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

// GetClientAuthOptionsAndToken returns client options and a token source using either a token URL or fallback to key file/ADC.
func GetClientAuthOptionsAndToken(ctx context.Context, config *StorageClientConfig) ([]option.ClientOption, oauth2.TokenSource, error) {
	// If Token URL is provided, attempt to fetch token source directly.
	if config.TokenUrl != "" {
		tokenSrc, err := auth2.NewTokenSourceFromURL(ctx, config.TokenUrl, config.ReuseTokenFromUrl)
		if err != nil {
			return nil, nil, fmt.Errorf("while fetching token source: %w", err)
		}

		clientOpts := []option.ClientOption{option.WithTokenSource(tokenSrc)}
		return clientOpts, tokenSrc, nil
	}

	// Fallback: Use key file credentials.
	cred, err := auth2.GetCredentials(config.KeyFile)
	if err != nil {
		return nil, nil, fmt.Errorf("while fetching credentials: %w", err)
	}

	tokenSrc := oauth2adapt.TokenSourceFromTokenProvider(cred.TokenProvider)

	retryConfig := NewRetryConfig(config, DefaultRetryDeadline, DefaultTotalRetryBudget, DefaultInitialBackoff)

	apiCall := func(attemptCtx context.Context) (string, error) {
		return cred.UniverseDomain(attemptCtx)
	}

	domain, err := ExecuteWithRetry(ctx, retryConfig, "cred.UniverseDomain", "credentials", apiCall)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get UniverseDomain: %w", err)
	}

	clientOpts := []option.ClientOption{option.WithUniverseDomain(domain), option.WithAuthCredentials(cred)}

	return clientOpts, tokenSrc, nil
}
