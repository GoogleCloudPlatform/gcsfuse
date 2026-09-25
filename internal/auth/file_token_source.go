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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"golang.org/x/oauth2"
)

const (
	// tokenFileReadAttempts is the number of times a token file is read before
	// a transient failure (missing file, empty file, or malformed JSON) is
	// reported. This tolerates writers that replace the file non-atomically,
	// where a read landing mid-write may observe a partially written file.
	tokenFileReadAttempts = 5
	// tokenFileReadRetryDelay is the initial delay between read attempts; it
	// doubles on each subsequent attempt.
	tokenFileReadRetryDelay = 10 * time.Millisecond
)

// fileTokenSource is an oauth2.TokenSource that reads an OAuth2 token
// response from a JSON file on disk.
type fileTokenSource struct {
	path string
	// retryDelay is the initial delay between read attempts on transient
	// failures. It exists as a field so tests can shorten it.
	retryDelay time.Duration
}

// readToken reads and decodes the token file once. It reports whether a
// failure is transient, i.e. consistent with the file being mid-rewrite by an
// external process, so the caller can retry.
func (ts fileTokenSource) readToken() (token *oauth2.Token, transient bool, err error) {
	f, err := os.Open(ts.path)
	if err != nil {
		// A writer that removes and recreates the file (rather than renaming
		// over it) leaves a brief window where the file does not exist.
		return nil, errors.Is(err, os.ErrNotExist), fmt.Errorf("fileTokenSource cannot read token file: %w", err)
	}
	defer f.Close()

	// Stat the open descriptor rather than the path so the modification time
	// belongs to the same file contents we are about to read, even if the
	// file is replaced concurrently.
	info, err := f.Stat()
	if err != nil {
		return nil, false, fmt.Errorf("fileTokenSource cannot stat token file: %w", err)
	}
	contents, err := io.ReadAll(f)
	if err != nil {
		return nil, false, fmt.Errorf("fileTokenSource cannot read token file: %w", err)
	}
	if len(contents) == 0 {
		return nil, true, fmt.Errorf("token file %q is empty", ts.path)
	}

	token = &oauth2.Token{}
	if err := json.Unmarshal(contents, token); err != nil {
		return nil, true, fmt.Errorf("fileTokenSource cannot decode token file %q: %w", ts.path, err)
	}

	// The expires_in field is relative to when the token was issued, which we
	// approximate by the file's modification time, i.e. when the external
	// process wrote the refreshed credential. Anchoring to the read time
	// instead would overestimate the remaining lifetime by however long the
	// file sat on disk before being read. An absolute expiry field, if present
	// in the file, takes precedence.
	if token.Expiry.IsZero() && token.ExpiresIn > 0 {
		token.Expiry = info.ModTime().Add(time.Duration(token.ExpiresIn) * time.Second)
	}
	if !token.Valid() {
		return nil, false, fmt.Errorf("token file %q contains an invalid or expired token", ts.path)
	}

	return token, false, nil
}

func (ts fileTokenSource) Token() (*oauth2.Token, error) {
	delay := ts.retryDelay
	if delay <= 0 {
		delay = tokenFileReadRetryDelay
	}
	var err error
	for attempt := 1; ; attempt++ {
		var token *oauth2.Token
		var transient bool
		token, transient, err = ts.readToken()
		if err == nil {
			return token, nil
		}
		if !transient || attempt >= tokenFileReadAttempts {
			return nil, err
		}
		time.Sleep(delay)
		delay *= 2
	}
}

// NewTokenSourceFromTokenFile returns a TokenSource backed by a file
// containing an OAuth2 token response in JSON format, e.g.
// {"access_token": "...", "expires_in": 3600, "token_type": "Bearer"}.
//
// The file is read eagerly so that an invalid or expired token fails at mount
// time, and re-read when the token expires, so an external process may
// refresh the credential by rewriting the file. Writers should replace the
// file atomically (write to a temporary file in the same directory, then
// rename over the token file); reads that observe a partially written file
// are retried briefly, but atomic replacement is what makes refresh reliable.
//
// The token file is a credential and should be protected with filesystem
// permissions like any other credential file (e.g. --key-file). A warning is
// logged if the file is writable by group or others.
func NewTokenSourceFromTokenFile(path string) (oauth2.TokenSource, error) {
	ts := fileTokenSource{path: path}
	token, err := ts.Token()
	if err != nil {
		return nil, err
	}
	warnIfTokenFileIsWidelyWritable(path)
	return oauth2.ReuseTokenSource(token, ts), nil
}

// warnIfTokenFileIsWidelyWritable logs a warning when the token file can be
// modified by processes other than its owner, since any such process could
// substitute its own credential.
func warnIfTokenFileIsWidelyWritable(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	if perm := info.Mode().Perm(); perm&0o022 != 0 {
		logger.Warnf("token file %q has permissions %04o and is writable by group or others; restrict it to the owner (e.g. chmod 600)", path, perm)
	}
}
