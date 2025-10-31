// Copyright 2020 Google LLC
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

package auth

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	storagev1 "google.golang.org/api/storage/v1"
)

const UniverseDomainDefault = "googleapis.com"

func getUniverseDomain(ctx context.Context, contents []byte, scope string) (string, error) {
	creds, err := google.CredentialsFromJSON(ctx, contents, scope)
	if err != nil {
		err = fmt.Errorf("CredentialsFromJSON(): %w", err)
		return "", err
	}

	domain, err := creds.GetUniverseDomain()
	if err != nil {
		err = fmt.Errorf("GetUniverseDomain(): %w", err)
		return "", err
	}

	return domain, nil
}

// Create token source from the JSON file at the supplide path.
func newTokenSourceFromPath(ctx context.Context, path string, scope string) (oauth2.TokenSource, error) {
	// Read the file.
	contents, err := os.ReadFile(path)
	if err != nil {
		err = fmt.Errorf("ReadFile(%q): %w", path, err)
		return nil, err
	}

	// By default, a standard OAuth 2.0 token source is created
	// Create a config struct based on its contents.
	jwtConfig, err := google.JWTConfigFromJSON(contents, scope)
	if err != nil {
		err = fmt.Errorf("JWTConfigFromJSON: %w", err)
		return nil, err
	}

	domain, err := getUniverseDomain(ctx, contents, scope)
	if err != nil {
		return nil, err
	}

	// By default, a standard OAuth 2.0 token source is created
	ts := jwtConfig.TokenSource(ctx)

	// For non-GDU universe domains, token exchange is impossible and services
	// must support self-signed JWTs with scopes.
	// Override the token source to use self-signed JWT.
	if domain != UniverseDomainDefault {
		// Create self signed JWT access token.
		ts, err = google.JWTAccessTokenSourceWithScope(contents, scope)
		if err != nil {
			err = fmt.Errorf("JWTAccessTokenSourceWithScope: %w", err)
			return nil, err
		}
	}
	return ts, err
}

// GetTokenSource generates the token-source for GCS endpoint by following oauth2.0 authentication
// for key-file and default-credential flow.
// It also supports generating the self-signed JWT tokenSource for key-file authentication which can be
// used by custom-endpoint(e.g. TPC).
func GetTokenSource(
	ctx context.Context,
	keyFile string,
	tokenUrl string,
	reuseTokenFromUrl bool,
) (tokenSrc oauth2.TokenSource, err error) {
	// Create the oauth2 token source.
	const scope = storagev1.DevstorageFullControlScope
	var method string

	if keyFile != "" {
		tokenSrc, err = newTokenSourceFromPath(ctx, keyFile, scope)
		method = "newTokenSourceFromPath"
	} else if tokenUrl != "" {
		tokenSrc, err = newProxyTokenSource(ctx, tokenUrl, reuseTokenFromUrl)
		method = "newProxyTokenSource"
	} else {
		tokenSrc, err = google.DefaultTokenSource(ctx, scope)
		method = "DefaultTokenSource"
	}

	if err != nil {
		err = fmt.Errorf("%s: %w", method, err)
		return
	}
	return
}
