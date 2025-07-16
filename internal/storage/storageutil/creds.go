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
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

// createTokenSourceFromTokenUrl returns a token source based on the token URL in config.
// Returns nil if no token URL is provided.
func createTokenSourceFromTokenUrl(config *StorageClientConfig) (oauth2.TokenSource, error) {
	if config.TokenUrl == "" {
		return nil, nil
	}
	return auth2.GetTokenSourceFromTokenUrl(context.Background(), config.TokenUrl, config.ReuseTokenFromUrl)
}

// createCredentials returns credentials from the provided key file or ADC.
func createCredentials(config *StorageClientConfig) (*auth.Credentials, error) {
	return auth2.GetCredentials(config.KeyFile)
}

// CreateCredentialForClient returns a token source after checking token URL or fallback to key file/ADC.
// It also updates clientOpts appropriately for the generated token source or credentials.
func CreateCredentialForClient(config *StorageClientConfig, clientOpts []option.ClientOption) (oauth2.TokenSource, error) {
	// Try to create token source from token URL.
	tokenSrc, err := createTokenSourceFromTokenUrl(config)
	if err != nil {
		return nil, fmt.Errorf("while fetching token source: %w", err)
	}

	if tokenSrc != nil {
		clientOpts = append(clientOpts, option.WithTokenSource(tokenSrc))
		return tokenSrc, nil
	}

	// Fallback to credentials (key file or ADC).
	cred, err := createCredentials(config)
	if err != nil {
		return nil, fmt.Errorf("while fetching credentials: %w", err)
	}

	tokenSrc = oauth2adapt.TokenSourceFromTokenProvider(cred.TokenProvider)

	domain, err := cred.UniverseDomain(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get UniverseDomain: %w", err)
	}

	clientOpts = append(
		clientOpts,
		option.WithUniverseDomain(domain),
		option.WithAuthCredentials(cred),
	)

	return tokenSrc, nil
}
