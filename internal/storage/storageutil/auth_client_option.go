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

// createTokenSourceFromTokenUrl returns a token source using tokenUrl and reuse flag.
// Returns nil if tokenUrl is empty.
func createTokenSourceFromTokenUrl(ctx context.Context, tokenUrl string, reuse bool) (oauth2.TokenSource, error) {
	if tokenUrl == "" {
		return nil, nil
	}
	return auth2.NewTokenSourceFromURL(ctx, tokenUrl, reuse)
}

// createCredentials returns credentials from the provided key file.
func createCredentials(keyFile string) (*auth.Credentials, error) {
	return auth2.GetCredentials(keyFile)
}

// GetClientAuthOptionsAndToken returns a token source using either a token URL or falling back to key file/ADC.
// It also returns client options containing the token source, universe domain, and credentials.
func GetClientAuthOptionsAndToken(ctx context.Context, config *StorageClientConfig) ([]option.ClientOption, oauth2.TokenSource, error) {
	// Attempt to create token source via token URL.
	tokenSrc, err := createTokenSourceFromTokenUrl(ctx, config.TokenUrl, config.ReuseTokenFromUrl)
	if err != nil {
		return nil, nil, fmt.Errorf("while fetching token source: %w", err)
	}

	var clientOpts []option.ClientOption

	if tokenSrc != nil {
		clientOpts = append(clientOpts, option.WithTokenSource(tokenSrc))
		return clientOpts, tokenSrc, nil
	}

	// Fallback to key file credentials.
	cred, err := createCredentials(config.KeyFile)
	if err != nil {
		return nil, nil, fmt.Errorf("while fetching credentials: %w", err)
	}

	tokenSrc = oauth2adapt.TokenSourceFromTokenProvider(cred.TokenProvider)

	domain, err := cred.UniverseDomain(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get UniverseDomain: %w", err)
	}

	clientOpts = append(clientOpts, option.WithUniverseDomain(domain), option.WithAuthCredentials(cred))

	return clientOpts, tokenSrc, nil
}
