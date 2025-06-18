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
	"fmt"

	"cloud.google.com/go/auth"
	"cloud.google.com/go/auth/credentials"
	"cloud.google.com/go/storage"
)

var detectCredentials = credentials.DetectDefault

const scope = storage.ScopeFullControl

// GetCredentials detects default Google Cloud credentials.
//
// It prioritizes a service account key file if `keyFile` is provided. If `keyFile` is
// empty, it attempts to detect Application Default Credentials (ADC) and checks
// the metadata server for credentials.
//
// The function requests storagev1.DevstorageFullControlScope to ensure the most comprehensive
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
	opts := &credentials.DetectOptions{
		CredentialsFile: keyFile,
		Scopes:          []string{scope},
	}

	creds, err := detectCredentials(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to detect credentials: %w", err)
	}

	return creds, nil
}
