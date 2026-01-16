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

	"cloud.google.com/go/auth"
	"cloud.google.com/go/auth/oauth2adapt"
	auth2 "github.com/googlecloudplatform/gcsfuse/v3/internal/auth"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
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
		d, err := cred.UniverseDomain(attemptCtx)
		return d, err
	}

	domain, err := ExecuteWithRetry(ctx, retryConfig, "cred.UniverseDomain", "credentials", apiCall)
	if err != nil {
		logger.Errorf("failed to get UniverseDomain: %v, setting default universe domain", err)
		// Setting default universe domain to googleapis.com in case we are unable to fetch the domain.
		domain = auth2.UniverseDomainDefault
	}
	logger.Tracef("Success in fetching cred.UniverseDomain")

	// Temporary Workaround: We've created a small auth object here that omits the 'quota project ID'
	// to bypass a known issue (b/442805436) in the current authentication library.
	// TODO: Remove this workaround once issue b/442805436 is resolved in the library.
	newCreds := auth.NewCredentials(&auth.CredentialsOptions{
		TokenProvider:          cred.TokenProvider,
		UniverseDomainProvider: auth.CredentialsPropertyFunc(func(_ context.Context) (string, error) { return domain, nil }),
	})
	clientOpts := []option.ClientOption{option.WithUniverseDomain(domain), option.WithAuthCredentials(newCreds)}

	return clientOpts, tokenSrc, nil
}
