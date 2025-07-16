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
	"errors"
	"testing"

	"cloud.google.com/go/auth"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

// --- Helper to reset after test ---
func resetInjectedFunctions() {
	createTokenSourceFromTokenUrlFn = createTokenSourceFromTokenUrl
	createCredentialsFn = createCredentials
}

func TestCreateCredentials(t *testing.T) {
	cred, err := createCredentials("testdata/google_creds.json")

	require.NoError(t, err)
	require.NotNil(t, cred)
}

func Test_CreateTokenSourceFromTokenUrl(t *testing.T) {
	t.Run("empty token URL returns nil", func(t *testing.T) {
		tokenSrc, err := createTokenSourceFromTokenUrl("", false)
		require.NoError(t, err)
		require.Nil(t, tokenSrc)
	})

	t.Run("valid token URL returns token source", func(t *testing.T) {
		tokenSrc, err := createTokenSourceFromTokenUrl("https://example.com/token", false)
		require.NoError(t, err)
		require.NotNil(t, tokenSrc)
	})
}

func Test_CreateCredentialForClient_TokenUrlPreferredSuccess(t *testing.T) {
	var clientOpts []option.ClientOption
	config := &StorageClientConfig{
		TokenUrl:          "https://example.com/token",
		ReuseTokenFromUrl: false,
		KeyFile:           "/path/to/keyfile.json",
	}

	tokenSrc, err := ConfigureClientAuth(config, &clientOpts)

	require.NoError(t, err)
	require.NotNil(t, tokenSrc)
	require.Len(t, clientOpts, 1) // Only tokenSource option attached
}

func Test_ConfigureClientAuth_TokenUrlPreferredError(t *testing.T) {
	createTokenSourceFromTokenUrlFn = func(tokenUrl string, reuse bool) (oauth2.TokenSource, error) {
		return nil, errors.New("simulated token source error")
	}
	defer resetInjectedFunctions()
	config := &StorageClientConfig{TokenUrl: "fake-url"}
	var clientOpts []option.ClientOption

	_, err := ConfigureClientAuth(config, &clientOpts)

	require.ErrorContains(t, err, "simulated token source error")
	require.Empty(t, clientOpts)
}

func Test_CreateCredentialForClient_FallbackToKeyFileSuccess(t *testing.T) {
	var clientOpts []option.ClientOption
	config := &StorageClientConfig{
		TokenUrl: "", // triggers fallback
		KeyFile:  "testdata/google_creds.json",
	}

	tokenSrc, err := ConfigureClientAuth(config, &clientOpts)

	require.NoError(t, err)
	require.NotNil(t, tokenSrc)
	require.Len(t, clientOpts, 2) // UniverseDomain + AuthCredentials
}

func Test_ConfigureClientAuth_FallbackToKeyFileError(t *testing.T) {
	createTokenSourceFromTokenUrlFn = func(tokenUrl string, reuse bool) (oauth2.TokenSource, error) {
		return nil, nil
	}
	createCredentialsFn = func(keyFile string) (*auth.Credentials, error) {
		return &auth.Credentials{
			TokenProvider: nil,
		}, errors.New("error in getting credentials")
	}
	defer resetInjectedFunctions()
	config := &StorageClientConfig{TokenUrl: "", KeyFile: "fake-key"}
	var clientOpts []option.ClientOption

	_, err := ConfigureClientAuth(config, &clientOpts)

	require.Error(t, err)
	require.Empty(t, clientOpts)
}
