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

package auth2

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/auth/credentials"
	"cloud.google.com/go/auth/oauth2adapt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	storagev1 "google.golang.org/api/storage/v1"
)

// Create token source from the JSON file at the supplide path.
func newTokenSourceFromPath2(ctx context.Context, path string, scope string) (oauth2.TokenSource, string, error) {
	var opts *credentials.DetectOptions
	opts = &credentials.DetectOptions{
		CredentialsFile: path,
		Scopes:          []string{scope},
	}

	// Detect credentials using the specified options
	creds, err := credentials.DetectDefault(opts)
	if err != nil {
		log.Fatalf("failed to detect credentials: %v", err)
	}

	// Request token source for required scopes
	ts := oauth2adapt.TokenSourceFromTokenProvider(creds.TokenProvider)

	domain, err := creds.UniverseDomain(ctx)
	if err != nil {
		log.Fatalf("failed to get UniverseDomain: %v", err)
	}

	return ts, domain, err
}

// GetTokenSource generates the token-source for GCS endpoint by following oauth2.0 authentication
// for key-file and default-credential flow.
// It also supports generating the self-signed JWT tokenSource for key-file authentication which can be
// used by custom-endpoint(e.g. TPC).
func GetTokenSource2(
	ctx context.Context,
	keyFile string,
	tokenUrl string,
	reuseTokenFromUrl bool,
) (tokenSrc oauth2.TokenSource, domain string, err error) {
	// Create the oauth2 token source.
	const scope = storagev1.DevstorageFullControlScope
	var method string

	if keyFile != "" {
		tokenSrc, domain, err = newTokenSourceFromPath2(ctx, keyFile, scope)
		method = "newTokenSourceFromPath"
	} else if tokenUrl != "" {
		//tokenSrc, err = newProxyTokenSource(ctx, tokenUrl, reuseTokenFromUrl)
		method = "newProxyTokenSource"
	} else {
		var creds *google.Credentials
		creds, err = google.FindDefaultCredentials(ctx, scope)
		if err == nil {
			//	tokenSrc = creds.TokenSource
			domain, err = creds.GetUniverseDomain()
			if err != nil {
				domain = ""
				err = fmt.Errorf("error in fetching domain name: %w", err)
			}
		}
		method = "DefaultTokenSource"
	}

	if err != nil {
		err = fmt.Errorf("%s: %w", method, err)
		return
	}
	return
}
