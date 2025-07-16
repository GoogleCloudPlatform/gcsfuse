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

// CreateTokenSourceFromTokenUrl creates a token-source from the provided token-url.
// It returns nil if the token-url is empty.
func createTokenSourceFromTokenUrl(storageClientConfig *StorageClientConfig) (oauth2.TokenSource, error) {
	if storageClientConfig.TokenUrl == "" {
		return nil, nil
	}

	return auth2.GetTokenSourceFromTokenUrl(context.Background(), storageClientConfig.TokenUrl, storageClientConfig.ReuseTokenFromUrl)
}

// CreateCredentials creates credentials from the provided key-file or using ADC.
func createCredentials(storageClientConfig *StorageClientConfig) (*auth.Credentials, error) {
	return auth2.GetCredentials(storageClientConfig.KeyFile)
}

func CreateCredentialForClient(storageClientConfig *StorageClientConfig, clientOpts []option.ClientOption) (oauth2.TokenSource, error) {
	var tokenSrc oauth2.TokenSource
	var err error
	tokenSrc, err = createTokenSourceFromTokenUrl(storageClientConfig)
	if err != nil {
		return nil, fmt.Errorf("while fetching tokenSource: %w", err)
	}
	if tokenSrc != nil {
		clientOpts = append(clientOpts, option.WithTokenSource(tokenSrc))
	} else {
		var cred *auth.Credentials
		cred, err = createCredentials(storageClientConfig)
		if err != nil {
			return nil, fmt.Errorf("while fetching credentials: %w", err)
		}
		tokenSrc = oauth2adapt.TokenSourceFromTokenProvider(cred.TokenProvider)
		var domain string
		domain, err = cred.UniverseDomain(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to get UniverseDomain: %v", err)
		}
		clientOpts = append(clientOpts, option.WithUniverseDomain(domain), option.WithAuthCredentials(cred))
	}

	return tokenSrc, nil
}
