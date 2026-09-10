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

// fastFileTokenSource returns a fileTokenSource whose transient-failure
// retries complete quickly.
func fastFileTokenSource(tokenFile string) fileTokenSource {
	return fileTokenSource{path: tokenFile, retryDelay: time.Millisecond}
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

func Test_NewTokenSourceFromTokenFile_ExpiresInAnchoredToModTime(t *testing.T) {
	tokenFile := writeTokenFile(t, `{"access_token":"test-access-token","expires_in":3600,"token_type":"Bearer"}`)
	// Simulate a token that was written 30 minutes ago and only read now.
	writtenAt := time.Now().Add(-30 * time.Minute).Truncate(time.Second)
	require.NoError(t, os.Chtimes(tokenFile, writtenAt, writtenAt))

	ts, err := NewTokenSourceFromTokenFile(tokenFile)

	require.NoError(t, err)
	token, err := ts.Token()
	require.NoError(t, err)
	// The remaining lifetime is 3600s measured from when the file was written,
	// not from when it was read.
	assert.WithinDuration(t, writtenAt.Add(time.Hour), token.Expiry, time.Second)
}

func Test_NewTokenSourceFromTokenFile_ExpiresInElapsedSinceModTime(t *testing.T) {
	tokenFile := writeTokenFile(t, `{"access_token":"test-access-token","expires_in":60,"token_type":"Bearer"}`)
	// The token had 60s of life when written two hours ago, so it is expired
	// even though a read-time anchor would consider it fresh.
	writtenAt := time.Now().Add(-2 * time.Hour)
	require.NoError(t, os.Chtimes(tokenFile, writtenAt, writtenAt))

	ts, err := NewTokenSourceFromTokenFile(tokenFile)

	assert.Error(t, err)
	assert.ErrorContains(t, err, "invalid or expired token")
	assert.Nil(t, ts)
}

func Test_NewTokenSourceFromTokenFile_AbsoluteExpiryTakesPrecedence(t *testing.T) {
	expiry := time.Now().Add(10 * time.Minute).Truncate(time.Second)
	tokenFile := writeTokenFile(t, fmt.Sprintf(`{"access_token":"test-access-token","expires_in":3600,"token_type":"Bearer","expiry":%q}`, expiry.Format(time.RFC3339)))

	ts, err := NewTokenSourceFromTokenFile(tokenFile)

	require.NoError(t, err)
	token, err := ts.Token()
	require.NoError(t, err)
	assert.True(t, expiry.Equal(token.Expiry), "expected %v, got %v", expiry, token.Expiry)
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

func Test_NewTokenSourceFromTokenFile_EmptyFile(t *testing.T) {
	tokenFile := writeTokenFile(t, "")

	ts, err := NewTokenSourceFromTokenFile(tokenFile)

	assert.Error(t, err)
	assert.ErrorContains(t, err, "is empty")
	assert.Nil(t, ts)
}

func Test_NewTokenSourceFromTokenFile_FileNotFound(t *testing.T) {
	ts, err := NewTokenSourceFromTokenFile(path.Join(t.TempDir(), "missing.json"))

	assert.Error(t, err)
	assert.ErrorContains(t, err, "cannot read token file")
	assert.Nil(t, ts)
}

func Test_NewTokenSourceFromTokenFile_WidelyWritableFileStillMounts(t *testing.T) {
	tokenFile := writeTokenFile(t, `{"access_token":"test-access-token","token_type":"Bearer"}`)
	require.NoError(t, os.Chmod(tokenFile, 0o666))

	ts, err := NewTokenSourceFromTokenFile(tokenFile)

	// Loose permissions produce a warning, not a failure.
	require.NoError(t, err)
	assert.NotNil(t, ts)
}

func TestFileTokenSource_RereadsFileOnEachCall(t *testing.T) {
	tokenFile := writeTokenFile(t, `{"access_token":"token-1","token_type":"Bearer"}`)
	ts := fastFileTokenSource(tokenFile)
	token, err := ts.Token()
	require.NoError(t, err)
	require.Equal(t, "token-1", token.AccessToken)

	// An external process may refresh the credential by rewriting the file.
	require.NoError(t, os.WriteFile(tokenFile, []byte(`{"access_token":"token-2","token_type":"Bearer"}`), 0o600))

	token, err = ts.Token()
	require.NoError(t, err)
	assert.Equal(t, "token-2", token.AccessToken)
}

func TestFileTokenSource_RetriesWhileFileIsBeingRewritten(t *testing.T) {
	// Simulate a non-atomic writer that has truncated the file but not yet
	// written the new contents.
	tokenFile := writeTokenFile(t, "")
	ts := fastFileTokenSource(tokenFile)
	done := make(chan struct{})
	go func() {
		defer close(done)
		time.Sleep(5 * time.Millisecond)
		_ = os.WriteFile(tokenFile, []byte(`{"access_token":"refreshed","token_type":"Bearer"}`), 0o600)
	}()

	token, err := ts.Token()

	<-done
	require.NoError(t, err)
	assert.Equal(t, "refreshed", token.AccessToken)
}

func TestFileTokenSource_RetriesWhileFileIsBeingReplaced(t *testing.T) {
	// Simulate a writer that removes the file and recreates it, leaving a
	// brief window where it does not exist.
	dir := t.TempDir()
	tokenFile := path.Join(dir, "token.json")
	ts := fastFileTokenSource(tokenFile)
	done := make(chan struct{})
	go func() {
		defer close(done)
		time.Sleep(5 * time.Millisecond)
		_ = os.WriteFile(tokenFile, []byte(`{"access_token":"recreated","token_type":"Bearer"}`), 0o600)
	}()

	token, err := ts.Token()

	<-done
	require.NoError(t, err)
	assert.Equal(t, "recreated", token.AccessToken)
}

func TestFileTokenSource_DoesNotRetryExpiredToken(t *testing.T) {
	expiry := time.Now().Add(-time.Hour).Format(time.RFC3339)
	tokenFile := writeTokenFile(t, fmt.Sprintf(`{"access_token":"test-access-token","token_type":"Bearer","expiry":%q}`, expiry))
	ts := fileTokenSource{path: tokenFile, retryDelay: time.Second}

	start := time.Now()
	_, err := ts.Token()

	// An expired token is a definitive failure, so it must not burn through
	// the retry budget.
	assert.Error(t, err)
	assert.Less(t, time.Since(start), time.Second)
}
