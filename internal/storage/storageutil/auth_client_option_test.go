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
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"cloud.google.com/go/auth"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func resetInjectedFunctions() {
	createTokenSourceFromTokenUrlFn = createTokenSourceFromTokenUrl
	createCredentialsFn = createCredentials
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func TestCreateCredentials(t *testing.T) {
	cred, err := createCredentials("testdata/key.json")

	assert.NoError(t, err)
	assert.NotNil(t, cred)
}

func Test_CreateCredentialForClient_TokenUrlPreferredSuccess(t *testing.T) {
	var clientOpts []option.ClientOption
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"access_token":"dummy-token","token_type":"Bearer"}`)
	}))
	defer server.Close()
	config := &StorageClientConfig{
		TokenUrl:          server.URL,
		ReuseTokenFromUrl: false,
		KeyFile:           "/path/to/keyfile.json",
	}

	tokenSrc, err := ConfigureClientAuth(context.TODO(), config, &clientOpts)

	assert.NoError(t, err)
	assert.NotNil(t, tokenSrc)
	assert.Len(t, clientOpts, 1) // Only tokenSource option attached
}

func Test_ConfigureClientAuth_TokenUrlPreferredError(t *testing.T) {
	createTokenSourceFromTokenUrlFn = func(ctx context.Context, tokenUrl string, reuse bool) (oauth2.TokenSource, error) {
		return nil, errors.New("simulated token source error")
	}
	defer resetInjectedFunctions()
	config := &StorageClientConfig{TokenUrl: "fake-url"}
	var clientOpts []option.ClientOption

	_, err := ConfigureClientAuth(context.TODO(), config, &clientOpts)

	assert.Error(t, err)
	assert.ErrorContains(t, err, "simulated token source error")
	assert.Empty(t, clientOpts)
}

func Test_ConfigureClientAuth_NilClientOption(t *testing.T) {
	config := &StorageClientConfig{TokenUrl: "fake-url"}

	_, err := ConfigureClientAuth(context.TODO(), config, nil)

	assert.Error(t, err)
	assert.ErrorContains(t, err, "clientOpts cannot be nil")
}

func Test_CreateCredentialForClient_FallbackToKeyFileSuccess(t *testing.T) {
	var clientOpts []option.ClientOption
	config := &StorageClientConfig{
		TokenUrl: "", // triggers fallback
		KeyFile:  "testdata/key.json",
	}

	tokenSrc, err := ConfigureClientAuth(context.TODO(), config, &clientOpts)

	assert.NoError(t, err)
	assert.NotNil(t, tokenSrc)
	assert.Len(t, clientOpts, 2) // UniverseDomain + AuthCredentials
}

func Test_ConfigureClientAuth_FallbackToKeyFileError(t *testing.T) {
	createTokenSourceFromTokenUrlFn = func(ctx context.Context, tokenUrl string, reuse bool) (oauth2.TokenSource, error) {
		return nil, nil
	}
	createCredentialsFn = func(keyFile string) (*auth.Credentials, error) {
		return nil, errors.New("error in getting credentials")
	}
	defer resetInjectedFunctions()
	config := &StorageClientConfig{TokenUrl: "", KeyFile: "fake-key"}
	var clientOpts []option.ClientOption

	_, err := ConfigureClientAuth(context.TODO(), config, &clientOpts)

	assert.Error(t, err)
	assert.Empty(t, clientOpts)
}
