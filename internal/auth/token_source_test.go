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
	"sync/atomic"
	"testing"
	"time"

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

// Helper: startFakeTCPTokenServer creates an HTTP test server returning a standard token.
func startFakeTCPTokenServer(accessToken string) *httptest.Server {
	return httptest.NewServer(tokenHandler(accessToken))
}

// Helper: startFakeJSONTokenServer creates an HTTP test server returning raw JSON.
func startFakeJSONTokenServer(responseJSON string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responseJSON))
	}))
}

// Helper: fetchTokenFromJSON starts a mock server with raw JSON, creates a TokenSource, and fetches the token.
func fetchTokenFromJSON(t *testing.T, responseJSON string) *oauth2.Token {
	server := startFakeJSONTokenServer(responseJSON)
	defer server.Close()

	ts, err := NewTokenSourceFromURL(context.Background(), server.URL, false)
	require.NoError(t, err)
	require.NotNil(t, ts)

	token, err := ts.Token()
	require.NoError(t, err)
	require.NotNil(t, token)
	return token
}

// Helper: createTempUnixSocket creates a temporary Unix domain socket and registers automatic cleanup with t.Cleanup.
func createTempUnixSocket(t *testing.T) (string, net.Listener) {
	tmpFile, err := os.CreateTemp("", "gcsfuse-uds-test-*.sock")
	require.NoError(t, err)
	socketPath := tmpFile.Name()
	require.NoError(t, tmpFile.Close())
	require.NoError(t, os.Remove(socketPath))

	listener, err := net.Listen("unix", socketPath)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = listener.Close()
		_ = os.Remove(socketPath)
	})

	return socketPath, listener
}

// Helper: startUDSServer runs an HTTP server over a Unix domain socket listener with automatic cleanup.
func startUDSServer(t *testing.T, listener net.Listener, handler http.Handler) *http.Server {
	server := &http.Server{Handler: handler}
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(func() {
		_ = server.Close()
	})
	return server
}

// Helper: startFakeUDSTokenServer runs a UDS server asserting path, query, and host expectations.
func startFakeUDSTokenServer(t *testing.T, listener net.Listener, expectedEscapedPath, expectedQuery, accessToken string) *http.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, expectedEscapedPath, r.URL.EscapedPath())
		assert.Equal(t, expectedQuery, r.URL.RawQuery)
		assert.Equal(t, "unix", r.Host)
		tokenHandler(accessToken)(w, r)
	})
	return startUDSServer(t, listener, handler)
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

func Test_NewTokenSourceFromURL_UnixSocket_WithFragment_Success(t *testing.T) {
	socketPath, listener := createTempUnixSocket(t)
	expectedEscapedPath := "/computeMetadata/v1/instance/service-accounts/default/token"
	expectedQuery := "foo=bar&baz=qux"
	accessToken := "uds-access-token"

	startFakeUDSTokenServer(t, listener, expectedEscapedPath, expectedQuery, accessToken)

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
	socketPath, listener := createTempUnixSocket(t)
	// Path contains escaped slash (%2F) and query contains escaped space (%20)
	expectedEscapedPath := "/computeMetadata%2Fv1%2Finstance%2Fservice-accounts%2Fdefault%2Ftoken"
	expectedQuery := "foo=bar%20baz"
	accessToken := "uds-access-token-escaped"

	startFakeUDSTokenServer(t, listener, expectedEscapedPath, expectedQuery, accessToken)

	tokenURL := fmt.Sprintf("unix://%s#%s?%s", socketPath, expectedEscapedPath, expectedQuery)
	ts, err := NewTokenSourceFromURL(context.Background(), tokenURL, false)
	require.NoError(t, err)
	require.NotNil(t, ts)

	token, err := ts.Token()
	assert.NoError(t, err)
	assert.Equal(t, accessToken, token.AccessToken)
}

func Test_NewTokenSourceFromURL_UnixSocket_BackwardCompatibility_Success(t *testing.T) {
	socketPath, listener := createTempUnixSocket(t)
	expectedEscapedPath := "/"
	expectedQuery := "foo=bar"
	accessToken := "uds-access-token-compat"

	startFakeUDSTokenServer(t, listener, expectedEscapedPath, expectedQuery, accessToken)

	// unix:///path/to/socket?query (old way, but with query)
	tokenURL := fmt.Sprintf("unix://%s?%s", socketPath, expectedQuery)
	ts, err := NewTokenSourceFromURL(context.Background(), tokenURL, false)
	require.NoError(t, err)
	require.NotNil(t, ts)

	token, err := ts.Token()
	assert.NoError(t, err)
	assert.Equal(t, accessToken, token.AccessToken)
}

func TestProxyTokenSource_ExpiresInPopulatesExpiry_Success(t *testing.T) {
	before := time.Now()
	// Standard RFC 6749 response (expires_in only, no Go-specific expiry field)
	token := fetchTokenFromJSON(t, `{"access_token":"token-123","token_type":"Bearer","expires_in":3600}`)
	after := time.Now()

	assert.Equal(t, "token-123", token.AccessToken)
	assert.Equal(t, "Bearer", token.TokenType)
	assert.False(t, token.Expiry.IsZero(), "token.Expiry must be populated from expires_in")
	assert.True(t, token.Expiry.After(before.Add(3595*time.Second)))
	assert.True(t, token.Expiry.Before(after.Add(3605*time.Second)))
	assert.True(t, token.Valid())
}

func TestProxyTokenSource_ExplicitExpiryPreserved_Success(t *testing.T) {
	// Response with both explicit expiry timestamp and expires_in
	token := fetchTokenFromJSON(t, `{"access_token":"token-456","token_type":"Bearer","expires_in":3600,"expiry":"2035-01-01T00:00:00Z"}`)

	expectedExpiry, _ := time.Parse(time.RFC3339, "2035-01-01T00:00:00Z")
	assert.Equal(t, expectedExpiry, token.Expiry, "explicit expiry field must not be overwritten")
}

func TestProxyTokenSource_ZeroExpiresIn_ExpiryRemainsZero(t *testing.T) {
	token := fetchTokenFromJSON(t, `{"access_token":"token-no-expiry","token_type":"Bearer"}`)

	assert.True(t, token.Expiry.IsZero(), "token.Expiry should remain zero when expires_in is 0")
}

func TestProxyTokenSource_TokenRefresh_WithReuseTokenSource(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		w.Header().Set("Content-Type", "application/json")
		// oauth2.ReuseTokenSource has a 10-second safety window (expiryDelta = 10s).
		// Setting expires_in = 12 allows the token to remain valid for 2 seconds.
		resp := fmt.Sprintf(`{"access_token":"token-%d","token_type":"Bearer","expires_in":12}`, count)
		_, _ = w.Write([]byte(resp))
	}))
	defer server.Close()

	// reuseTokenFromUrl = true enables oauth2.ReuseTokenSource
	ts, err := NewTokenSourceFromURL(context.Background(), server.URL, true)
	require.NoError(t, err)
	require.NotNil(t, ts)

	// 1. Initial fetch
	token1, err := ts.Token()
	require.NoError(t, err)
	assert.Equal(t, "token-1", token1.AccessToken)
	assert.Equal(t, int32(1), atomic.LoadInt32(&requestCount))

	// 2. Fetch immediately -> Should return cached token (no new HTTP request)
	tokenCached, err := ts.Token()
	require.NoError(t, err)
	assert.Equal(t, "token-1", tokenCached.AccessToken)
	assert.Equal(t, int32(1), atomic.LoadInt32(&requestCount), "HTTP server should not be called while token is valid")

	// 3. Wait past the 2-second validity window (12s - 10s delta = 2s)
	time.Sleep(2500 * time.Millisecond)

	// 4. Fetch after expiry -> ReuseTokenSource should detect expiry and fetch fresh token
	token2, err := ts.Token()
	require.NoError(t, err)
	assert.Equal(t, "token-2", token2.AccessToken)
	assert.Equal(t, int32(2), atomic.LoadInt32(&requestCount), "HTTP server must be called when token expires")
}

func Test_NewTokenSourceFromURL_UnixSocket_ExpiresInPopulatesExpiry(t *testing.T) {
	socketPath, listener := createTempUnixSocket(t)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"uds-token-rfc","token_type":"Bearer","expires_in":1800}`))
	})
	startUDSServer(t, listener, handler)

	tokenURL := fmt.Sprintf("unix://%s#/computeMetadata/v1/instance/service-accounts/default/token", socketPath)
	ts, err := NewTokenSourceFromURL(context.Background(), tokenURL, false)
	require.NoError(t, err)
	require.NotNil(t, ts)

	token, err := ts.Token()
	assert.NoError(t, err)
	require.NotNil(t, token)
	assert.Equal(t, "uds-token-rfc", token.AccessToken)
	assert.False(t, token.Expiry.IsZero())
	assert.True(t, time.Until(token.Expiry) > 1700*time.Second)
}
