// Copyright 2026 Google LLC
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
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTokenFile(t *testing.T, contents string) string {
	t.Helper()
	tokenFile := path.Join(t.TempDir(), "token.json")
	require.NoError(t, os.WriteFile(tokenFile, []byte(contents), 0o600))
	return tokenFile
}

func Test_NewTokenSourceFromTokenFile_Success(t *testing.T) {
	tokenFile := writeTokenFile(t, `{"access_token":"test-access-token","expires_in":3600,"token_type":"Bearer"}`)

	ts, err := NewTokenSourceFromTokenFile(tokenFile)

	require.NoError(t, err)
	require.NotNil(t, ts)
	token, err := ts.Token()
	require.NoError(t, err)
	assert.Equal(t, "test-access-token", token.AccessToken)
	assert.Equal(t, "Bearer", token.TokenType)
	// expires_in must be anchored to an absolute expiry.
	assert.False(t, token.Expiry.IsZero())
	assert.True(t, token.Expiry.After(time.Now()))
}

func Test_NewTokenSourceFromTokenFile_NoExpiry(t *testing.T) {
	tokenFile := writeTokenFile(t, `{"access_token":"test-access-token","token_type":"Bearer"}`)

	ts, err := NewTokenSourceFromTokenFile(tokenFile)

	require.NoError(t, err)
	token, err := ts.Token()
	require.NoError(t, err)
	assert.Equal(t, "test-access-token", token.AccessToken)
	assert.True(t, token.Expiry.IsZero())
}

func Test_NewTokenSourceFromTokenFile_ExpiredToken(t *testing.T) {
	expiry := time.Now().Add(-time.Hour).Format(time.RFC3339)
	tokenFile := writeTokenFile(t, fmt.Sprintf(`{"access_token":"test-access-token","token_type":"Bearer","expiry":%q}`, expiry))

	ts, err := NewTokenSourceFromTokenFile(tokenFile)

	assert.Error(t, err)
	assert.ErrorContains(t, err, "invalid or expired token")
	assert.Nil(t, ts)
}

func Test_NewTokenSourceFromTokenFile_MissingAccessToken(t *testing.T) {
	tokenFile := writeTokenFile(t, `{"expires_in":3600,"token_type":"Bearer"}`)

	ts, err := NewTokenSourceFromTokenFile(tokenFile)

	assert.Error(t, err)
	assert.ErrorContains(t, err, "invalid or expired token")
	assert.Nil(t, ts)
}

func Test_NewTokenSourceFromTokenFile_InvalidJSON(t *testing.T) {
	tokenFile := writeTokenFile(t, "not-json")

	ts, err := NewTokenSourceFromTokenFile(tokenFile)

	assert.Error(t, err)
	assert.ErrorContains(t, err, "cannot decode token file")
	assert.Nil(t, ts)
}

func Test_NewTokenSourceFromTokenFile_FileNotFound(t *testing.T) {
	ts, err := NewTokenSourceFromTokenFile(path.Join(t.TempDir(), "missing.json"))

	assert.Error(t, err)
	assert.ErrorContains(t, err, "cannot read token file")
	assert.Nil(t, ts)
}

func TestFileTokenSource_RereadsFileOnEachCall(t *testing.T) {
	tokenFile := writeTokenFile(t, `{"access_token":"token-1","token_type":"Bearer"}`)
	ts := fileTokenSource{path: tokenFile}
	token, err := ts.Token()
	require.NoError(t, err)
	require.Equal(t, "token-1", token.AccessToken)

	// An external process may refresh the credential by rewriting the file.
	require.NoError(t, os.WriteFile(tokenFile, []byte(`{"access_token":"token-2","token_type":"Bearer"}`), 0o600))

	token, err = ts.Token()
	require.NoError(t, err)
	assert.Equal(t, "token-2", token.AccessToken)
}
