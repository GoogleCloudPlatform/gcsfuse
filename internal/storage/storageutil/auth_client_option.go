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

var (
	createTokenSourceFromTokenUrlFn = createTokenSourceFromTokenUrl
	createCredentialsFn             = createCredentials
)

// createTokenSourceFromTokenUrl returns a token source using tokenUrl and reuse flag.
// Returns nil if tokenUrl is empty.
func createTokenSourceFromTokenUrl(tokenUrl string, reuse bool) (oauth2.TokenSource, error) {
	if tokenUrl == "" {
		return nil, nil
	}
	return auth2.NewTokenSourceFromURL(context.Background(), tokenUrl, reuse)
}

// createCredentials returns credentials from the provided key file.
func createCredentials(keyFile string) (*auth.Credentials, error) {
	return auth2.GetCredentials(keyFile)
}

// ConfigureClientAuth returns a token source using either token URL or fallback to key file/ADC.
// It also updates clientOpts via pointer, so changes are visible to the caller.
func ConfigureClientAuth(config *StorageClientConfig, clientOpts *[]option.ClientOption) (oauth2.TokenSource, error) {
	// Try token source via token URL.
	tokenSrc, err := createTokenSourceFromTokenUrlFn(config.TokenUrl, config.ReuseTokenFromUrl)
	if err != nil {
		return nil, fmt.Errorf("while fetching token source: %w", err)
	}

	if tokenSrc != nil {
		*clientOpts = append(*clientOpts, option.WithTokenSource(tokenSrc))
		return tokenSrc, nil
	}

	// Fallback to credentials.
	cred, err := createCredentialsFn(config.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("while fetching credentials: %w", err)
	}

	tokenSrc = oauth2adapt.TokenSourceFromTokenProvider(cred.TokenProvider)

	domain, err := cred.UniverseDomain(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get UniverseDomain: %w", err)
	}

	*clientOpts = append(*clientOpts, option.WithUniverseDomain(domain), option.WithAuthCredentials(cred))

	return tokenSrc, nil
}
