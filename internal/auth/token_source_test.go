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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func Test_GetTokenSourceFromTokenUrl_Success(t *testing.T) {
	// Create fake token server.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := oauth2.Token{
			AccessToken: "test-access-token",
			TokenType:   "Bearer",
		}
		require.NoError(t, json.NewEncoder(w).Encode(token))
	}))
	defer server.Close()

	ts, err := GetTokenSourceFromTokenUrl(context.Background(), server.URL, false)

	assert.NoError(t, err)
	assert.NotNil(t, ts)
	// Fetch token
	token, err := ts.Token()
	assert.NoError(t, err)
	assert.Equal(t, "test-access-token", token.AccessToken)
}

func Test_GetTokenSourceFromTokenUrl_InvalidEndpoint(t *testing.T) {
	ts, err := GetTokenSourceFromTokenUrl(context.Background(), ":", false) // invalid URL

	assert.Error(t, err)
	assert.Nil(t, ts)
}

func Test_GetTokenSourceFromTokenUrl_ServerError(t *testing.T) {
	// Simulate HTTP 500 error.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	ts, err := GetTokenSourceFromTokenUrl(context.Background(), server.URL, false)

	assert.NoError(t, err)
	token, err := ts.Token()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server error")
	assert.Nil(t, token)
}

func Test_GetTokenSourceFromTokenUrl_InvalidJSON(t *testing.T) {
	// Simulate invalid JSON.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("not-json"))
		require.NoError(t, err)
	}))
	defer server.Close()

	ts, err := GetTokenSourceFromTokenUrl(context.Background(), server.URL, false)

	assert.NoError(t, err)
	token, err := ts.Token()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode body")
	assert.Nil(t, token)
}
