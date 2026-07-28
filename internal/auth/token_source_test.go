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
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func tokenHandler(accessToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := oauth2.Token{
			AccessToken: accessToken,
			TokenType:   "Bearer",
		}
		_ = json.NewEncoder(w).Encode(token)
	}
}

func startFakeTCPTokenServer(accessToken string) *httptest.Server {
	return httptest.NewServer(tokenHandler(accessToken))
}

func Test_NewTokenSourceFromURL_Success(t *testing.T) {
	accessToken := "test-access-token"
	server := startFakeTCPTokenServer(accessToken)
	defer server.Close()

	ts, err := NewTokenSourceFromURL(context.Background(), server.URL, false)

	assert.NoError(t, err)
	assert.NotNil(t, ts)
	// Fetch token
	token, err := ts.Token()
	assert.NoError(t, err)
	assert.Equal(t, accessToken, token.AccessToken)
}

func Test_NewTokenSourceFromURL_InvalidURL(t *testing.T) {
	ts, err := NewTokenSourceFromURL(context.Background(), ":", false) // invalid URL

	assert.Error(t, err)
	assert.Nil(t, ts)
}

func TestProxyTokenSource_TokenFetch_ServerError(t *testing.T) {
	// Simulate HTTP 500 error.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer server.Close()
	ts, err := NewTokenSourceFromURL(context.Background(), server.URL, false)
	require.NoError(t, err)

	token, err := ts.Token()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server error")
	assert.Nil(t, token)
}

func TestProxyTokenSource_TokenFetch_InvalidJSON(t *testing.T) {
	// Simulate invalid JSON.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("not-json"))
		require.NoError(t, err)
	}))
	defer server.Close()
	ts, err := NewTokenSourceFromURL(context.Background(), server.URL, false)
	require.NoError(t, err)

	token, err := ts.Token()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode body")
	assert.Nil(t, token)
}

func startFakeUDSTokenServer(t *testing.T, listener net.Listener, expectedEscapedPath, expectedQuery, accessToken string) *http.Server {
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, expectedEscapedPath, r.URL.EscapedPath())
			assert.Equal(t, expectedQuery, r.URL.RawQuery)
			assert.Equal(t, "unix", r.Host)
			tokenHandler(accessToken)(w, r)
		}),
	}
	go func() {
		_ = server.Serve(listener)
	}()
	return server
}

func Test_NewTokenSourceFromURL_UnixSocket_WithFragment_Success(t *testing.T) {
	// Create a temp file for the socket.
	tmpFile, err := os.CreateTemp("", "gcsfuse-uds-test-*.sock")
	require.NoError(t, err)
	socketPath := tmpFile.Name()
	require.NoError(t, tmpFile.Close())
	require.NoError(t, os.Remove(socketPath)) // remove it so net.Listen can create it
	defer func() { _ = os.Remove(socketPath) }()

	listener, err := net.Listen("unix", socketPath)
	require.NoError(t, err)
	defer func() { _ = listener.Close() }()

	expectedEscapedPath := "/computeMetadata/v1/instance/service-accounts/default/token"
	expectedQuery := "foo=bar&baz=qux"
	accessToken := "uds-access-token"

	server := startFakeUDSTokenServer(t, listener, expectedEscapedPath, expectedQuery, accessToken)
	defer func() { _ = server.Close() }()

	// unix:///path/to/socket#/http_path?query
	tokenURL := fmt.Sprintf("unix://%s#%s?%s", socketPath, expectedEscapedPath, expectedQuery)
	ts, err := NewTokenSourceFromURL(context.Background(), tokenURL, false)
	require.NoError(t, err)
	require.NotNil(t, ts)

	token, err := ts.Token()
	assert.NoError(t, err)
	assert.Equal(t, accessToken, token.AccessToken)
}

func Test_NewTokenSourceFromURL_UnixSocket_WithFragment_EscapedChars_Success(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "gcsfuse-uds-test-*.sock")
	require.NoError(t, err)
	socketPath := tmpFile.Name()
	require.NoError(t, tmpFile.Close())
	require.NoError(t, os.Remove(socketPath))
	defer func() { _ = os.Remove(socketPath) }()

	listener, err := net.Listen("unix", socketPath)
	require.NoError(t, err)
	defer func() { _ = listener.Close() }()

	// Path contains escaped slash (%2F) and query contains escaped space (%20)
	expectedEscapedPath := "/computeMetadata%2Fv1%2Finstance%2Fservice-accounts%2Fdefault%2Ftoken"
	expectedQuery := "foo=bar%20baz"
	accessToken := "uds-access-token-escaped"

	server := startFakeUDSTokenServer(t, listener, expectedEscapedPath, expectedQuery, accessToken)
	defer func() { _ = server.Close() }()

	tokenURL := fmt.Sprintf("unix://%s#%s?%s", socketPath, expectedEscapedPath, expectedQuery)
	ts, err := NewTokenSourceFromURL(context.Background(), tokenURL, false)
	require.NoError(t, err)
	require.NotNil(t, ts)

	token, err := ts.Token()
	assert.NoError(t, err)
	assert.Equal(t, accessToken, token.AccessToken)
}

func Test_NewTokenSourceFromURL_UnixSocket_BackwardCompatibility_Success(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "gcsfuse-uds-test-*.sock")
	require.NoError(t, err)
	socketPath := tmpFile.Name()
	require.NoError(t, tmpFile.Close())
	require.NoError(t, os.Remove(socketPath))
	defer func() { _ = os.Remove(socketPath) }()

	listener, err := net.Listen("unix", socketPath)
	require.NoError(t, err)
	defer func() { _ = listener.Close() }()

	expectedEscapedPath := "/"
	expectedQuery := "foo=bar"
	accessToken := "uds-access-token-compat"

	server := startFakeUDSTokenServer(t, listener, expectedEscapedPath, expectedQuery, accessToken)
	defer func() { _ = server.Close() }()

	// unix:///path/to/socket?query (old way, but with query)
	tokenURL := fmt.Sprintf("unix://%s?%s", socketPath, expectedQuery)
	ts, err := NewTokenSourceFromURL(context.Background(), tokenURL, false)
	require.NoError(t, err)
	require.NotNil(t, ts)

	token, err := ts.Token()
	assert.NoError(t, err)
	assert.Equal(t, accessToken, token.AccessToken)
}
