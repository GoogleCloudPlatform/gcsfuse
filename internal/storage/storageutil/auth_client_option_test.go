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
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/api/option"
)

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func Test_GetClientAuthOptionsAndToken_TokenUrlSuccess(t *testing.T) {
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

	clientOpts, tokenSrc, err := GetClientAuthOptionsAndToken(context.TODO(), config)

	assert.NoError(t, err)
	assert.NotNil(t, tokenSrc)
	assert.Len(t, clientOpts, 1) // Only tokenSource option attached
}

func Test_GetClientAuthOptionsAndToken_TokenUrlError(t *testing.T) {
	config := &StorageClientConfig{TokenUrl: ":"}

	clientOpts, tokenSrc, err := GetClientAuthOptionsAndToken(context.TODO(), config)

	assert.Error(t, err)
	assert.Nil(t, tokenSrc)
	assert.Empty(t, clientOpts)
}

func Test_GetClientAuthOptionsAndToken_FallbackToKeyFileSuccess(t *testing.T) {
	config := &StorageClientConfig{
		TokenUrl: "", // triggers fallback
		KeyFile:  "testdata/key.json",
	}

	clientOpts, tokenSrc, err := GetClientAuthOptionsAndToken(context.TODO(), config)

	assert.NoError(t, err)
	assert.NotNil(t, tokenSrc)
	assert.Len(t, clientOpts, 2) // UniverseDomain + AuthCredentials
}

func Test_GetClientAuthOptionsAndToken_FallbackToKeyFileError(t *testing.T) {
	config := &StorageClientConfig{TokenUrl: "", KeyFile: "fake-key"}
	var clientOpts []option.ClientOption

	clientOpts, tokenSrc, err := GetClientAuthOptionsAndToken(context.TODO(), config)

	assert.Error(t, err)
	assert.Nil(t, tokenSrc)
	assert.Empty(t, clientOpts)
}
