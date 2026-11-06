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
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func Test_GetClientAuthOptionsAndTokenSource_TokenUrlSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"access_token":"dummy-token","token_type":"Bearer"}`)
	}))
	defer server.Close()
	config := &StorageClientConfig{
		TokenUrl:          server.URL,
		ReuseTokenFromUrl: false,
		KeyFile:           "testdata/key.json",
	}

	clientOpts, tokenSrc, err := GetClientAuthOptionsAndTokenSource(context.TODO(), config)

	assert.NoError(t, err)
	assert.NotNil(t, tokenSrc)
	assert.Len(t, clientOpts, 1) // Only tokenSource option attached
}

func Test_GetClientAuthOptionsAndTokenSource_TokenUrlError(t *testing.T) {
	config := &StorageClientConfig{TokenUrl: ":"}

	clientOpts, tokenSrc, err := GetClientAuthOptionsAndTokenSource(context.TODO(), config)

	assert.Error(t, err)
	assert.Nil(t, tokenSrc)
	assert.Empty(t, clientOpts)
}

func Test_GetClientAuthOptionsAndTokenSource_TokenUrlWithMountRetries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintln(w, `{"access_token":"dummy-token","token_type":"Bearer"}`)
	}))
	defer server.Close()
	config := &StorageClientConfig{
		TokenUrl:           server.URL,
		EnableMountRetries: true,
	}

	clientOpts, tokenSrc, err := GetClientAuthOptionsAndTokenSource(context.TODO(), config)

	assert.NoError(t, err)
	assert.NotNil(t, tokenSrc)
	assert.IsType(t, &retryingTokenSource{}, tokenSrc)
	assert.Len(t, clientOpts, 1)
}

func Test_GetClientAuthOptionsAndTokenSource_FallbackToKeyFileSuccess(t *testing.T) {
	config := &StorageClientConfig{
		TokenUrl: "", // triggers fallback
		KeyFile:  "testdata/key.json",
	}

	clientOpts, tokenSrc, err := GetClientAuthOptionsAndTokenSource(context.TODO(), config)

	assert.NoError(t, err)
	assert.NotNil(t, tokenSrc)
	assert.Len(t, clientOpts, 2) // UniverseDomain + AuthCredentials
}

func Test_GetClientAuthOptionsAndTokenSource_FallbackToKeyFileWithMountRetries(t *testing.T) {
	config := &StorageClientConfig{
		KeyFile:            "testdata/key.json",
		EnableMountRetries: true,
	}

	clientOpts, tokenSrc, err := GetClientAuthOptionsAndTokenSource(context.TODO(), config)

	assert.NoError(t, err)
	assert.NotNil(t, tokenSrc)
	assert.Len(t, clientOpts, 2) // UniverseDomain + AuthCredentials
}

func Test_GetClientAuthOptionsAndTokenSource_FallbackToKeyFileError(t *testing.T) {
	config := &StorageClientConfig{TokenUrl: "", KeyFile: "fake-key"}
	var clientOpts []option.ClientOption

	clientOpts, tokenSrc, err := GetClientAuthOptionsAndTokenSource(context.TODO(), config)

	assert.Error(t, err)
	assert.Nil(t, tokenSrc)
	assert.Empty(t, clientOpts)
}

func Test_GetClientAuthOptionsAndTokenSource_InitializationCancellation(t *testing.T) {
	for _, reuse := range []bool{false, true} {
		for _, cancelBeforeFetch := range []bool{false, true} {
			t.Run(fmt.Sprintf("reuse=%t/cancel-before-fetch=%t", reuse, cancelBeforeFetch), func(t *testing.T) {
				var requests, traces atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = fmt.Fprintf(w, `{"access_token":"token-%d","expires_in":3600}`, requests.Add(1))
				}))
				defer server.Close()
				ctx := httptrace.WithClientTrace(context.Background(), &httptrace.ClientTrace{
					GetConn: func(_ string) { traces.Add(1) },
				})
				ctx, cancel := context.WithCancel(ctx)
				defer cancel()
				if cancelBeforeFetch {
					cancel()
				}
				config := &StorageClientConfig{TokenUrl: server.URL, ReuseTokenFromUrl: reuse}
				_, source, err := GetClientAuthOptionsAndTokenSource(ctx, config)
				require.NoError(t, err)
				first, err := source.Token()
				require.NoError(t, err)
				assert.Equal(t, "token-1", first.AccessToken)

				cancel()
				if reuse {
					cached, err := source.Token()
					require.NoError(t, err)
					assert.Same(t, first, cached)
					assert.Equal(t, int32(1), requests.Load())
					first.Expiry = time.Now().Add(-time.Hour)
				}
				refreshed, err := source.Token()

				require.NoError(t, err, "the cached source must outlive initialization")
				assert.Equal(t, "token-2", refreshed.AccessToken)
				assert.Equal(t, int32(2), requests.Load())
				assert.Equal(t, int32(2), traces.Load(), "initialization context values must be retained")
			})
		}
	}
}

func TestCustomTokenSourceTimeout(t *testing.T) {
	factories := map[string]func(*StorageClientConfig) (oauth2.TokenSource, error){
		"legacy": CreateTokenSource,
		"google-library": func(config *StorageClientConfig) (oauth2.TokenSource, error) {
			_, source, err := GetClientAuthOptionsAndTokenSource(context.Background(), config)
			return source, err
		},
	}
	for name, factory := range factories {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				select {
				case <-r.Context().Done():
				case <-time.After(5 * time.Second):
				}
			}))
			defer server.Close()
			source, err := factory(&StorageClientConfig{
				TokenUrl:                    server.URL,
				ExperimentalTokenUrlTimeout: 100 * time.Millisecond,
			})
			require.NoError(t, err)

			start := time.Now()
			token, err := source.Token()

			assert.ErrorIs(t, err, context.DeadlineExceeded)
			assert.Nil(t, token)
			assert.Less(t, time.Since(start), 2*time.Second)
		})
	}
}
