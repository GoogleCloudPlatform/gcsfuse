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

// GetCredentials detects default Google Cloud credentials.
//
// It prioritizes a service account key file if `keyFile` is provided. If `keyFile` is
// empty, it automatically attempts to detect Application Default Credentials (ADC)
// and checks the metadata server for credentials.
//
// This method requests `storagev1.DevstorageFullControlScope` for Google Cloud Storage access.
//
// Args:
// keyFile: Path to a service account key file. Pass an empty string if not used.
//
// Returns:
//
//	*auth.Credentials: Discovered authentication credentials.
//	error: An error if credential detection fails.
func GetCredentials(keyFile string) (*auth.Credentials, error) {
	const scope = storage.ScopeFullControl
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
