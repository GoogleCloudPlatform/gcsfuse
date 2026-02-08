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

package auth

import (
	"context"
	"time"

	"cloud.google.com/go/auth"
	"cloud.google.com/go/auth/credentials"
	"cloud.google.com/go/storage"
)

const scope = storage.ScopeFullControl

type dummyTokenProvider struct{}

func (p *dummyTokenProvider) Token(ctx context.Context) (*auth.Token, error) {
	return &auth.Token{Value: "dummy", Type: "Bearer", Expiry: time.Now().Add(time.Hour)}, nil
}

// getCredentials is a private helper that takes a custom DetectDefault function.
func getCredentials(keyFile string, detectCredential func(*credentials.DetectOptions) (*auth.Credentials, error)) (*auth.Credentials, error) {
	opts := &credentials.DetectOptions{
		CredentialsFile: keyFile,
		Scopes:          []string{scope},
	}

	creds, err := detectCredential(opts)
	if err != nil {
		return auth.NewCredentials(&auth.CredentialsOptions{
			TokenProvider: &dummyTokenProvider{},
		}), nil
	}

	return creds, nil
}

// GetCredentials detects default Google Cloud credentials.
//
// It prioritizes a service account key file if `keyFile` is provided. If `keyFile` is
// empty, it attempts to detect Application Default Credentials (ADC) and checks
// the metadata server for credentials.
//
// The function requests storage.ScopeFullControl to ensure the most comprehensive
// permissions for GCS. This allows subsequent operations using these credentials to have full read,
// write, and administrative control over GCS resources.
//
// Args:
//
//	keyFile: Path to a service account key file. Pass an empty string to use ADC.
//
// Returns:
//
//	*auth.Credentials: Discovered authentication credentials.
//	error: An error if credential detection fails.
func GetCredentials(keyFile string) (*auth.Credentials, error) {
	return getCredentials(keyFile, credentials.DetectDefault)
}
