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
	"log"

	"cloud.google.com/go/auth"
	"cloud.google.com/go/auth/credentials"
	"golang.org/x/oauth2"
	storagev1 "google.golang.org/api/storage/v1"
)

// GetTokenSource generates the token-source for GCS endpoint by following oauth2.0 authentication
// for key-file and default-credential flow.
// It also supports generating the self-signed JWT tokenSource for key-file authentication which can be
// used by custom-endpoint(e.g. TPC).
func GetCredentials(
	ctx context.Context,
	keyFile string,
) (*auth.Credentials, error) {
	// Create the oauth2 token source.
	const scope = storagev1.DevstorageFullControlScope

	opts := &credentials.DetectOptions{
		CredentialsFile: keyFile,
		Scopes:          []string{scope},
	}

	// Detect credentials using the specified options
	creds, err := credentials.DetectDefault(opts)
	if err != nil {
		log.Fatalf("failed to detect credentials: %v", err)
	}

	return creds, err
}

func GetTokenSourceFromTokenUrl(ctx context.Context, tokenUrl string, reuseTokenFromUrl bool) (tokenSrc oauth2.TokenSource, err error) {
	return newProxyTokenSource(ctx, tokenUrl, reuseTokenFromUrl)
}
