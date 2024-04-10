// Copyright 2020 Google Inc. All Rights Reserved.
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

const universeDomainDefault = "googleapis.com"

func getUniverseDomain(ctx context.Context, contents []byte, scope string) (string, error) {
	creds, err := google.CredentialsFromJSON(ctx, contents, scope)
	if err != nil {
		err = fmt.Errorf("Error in getting credentials.")
		return "", err
	}
	domain, err := creds.GetUniverseDomain()
	if err != nil {
		err = fmt.Errorf("Error in getting universe domain.")
		return "", err
	}

	return domain, nil
}

func getAccessToken(ctx context.Context, domain, scope string, contents []byte) (ts oauth2.TokenSource, err error) {
	if domain == universeDomainDefault {
		// By default, a standard OAuth 2.0 token source is created
		// Create a config struct based on its contents.
		jwtConfig, err := google.JWTConfigFromJSON(contents, scope)
		if err != nil {
			err = fmt.Errorf("JWTConfigFromJSON: %w", err)
		}
		// Create the token source.
		ts = jwtConfig.TokenSource(ctx)
	} else {
		// For non-GDU universe domains, token exchange is impossible and services
		// must support self-signed JWTs with scopes.

		// Create a config struct based on its contents.
		ts, err = google.JWTAccessTokenSourceWithScope(contents, scope)
		if err != nil {
			err = fmt.Errorf("JWTConfigFromJSON: %w", err)
			return
		}
	}
	return
}

// Create token source from the JSON file at the supplide path.
func newTokenSourceFromPath(
		ctx context.Context,
		path string,
		scope string,
) (ts oauth2.TokenSource, err error) {
	// Read the file.
	contents, err := os.ReadFile(path)
	if err != nil {
		err = fmt.Errorf("ReadFile(%q): %w", path, err)
		return
	}

	domain, err := getUniverseDomain(ctx, contents, scope)
	if err != nil {
		return nil, err
	}

	ts, err = getAccessToken(ctx, domain, scope, contents)
	if err != nil {
		err = fmt.Errorf("Error in getting access token:%v", err)
	}

	return
}

// GetTokenSource returns a TokenSource for GCS API given a key file, or
// with the default credentials.
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
