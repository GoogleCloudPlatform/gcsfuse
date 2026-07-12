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
	"fmt"
	"os"
	"time"

	"golang.org/x/oauth2"
)

// fileTokenSource is an oauth2.TokenSource that reads an OAuth2 token
// response from a JSON file on disk.
type fileTokenSource struct {
	path string
}

func (ts fileTokenSource) Token() (*oauth2.Token, error) {
	contents, err := os.ReadFile(ts.path)
	if err != nil {
		return nil, fmt.Errorf("fileTokenSource cannot read token file: %w", err)
	}

	token := &oauth2.Token{}
	if err := json.Unmarshal(contents, token); err != nil {
		return nil, fmt.Errorf("fileTokenSource cannot decode token file %q: %w", ts.path, err)
	}

	// The expires_in field is relative, so anchor it to the time the file was
	// read to get an absolute expiry. An absolute expiry field, if present in
	// the file, takes precedence.
	if token.Expiry.IsZero() && token.ExpiresIn > 0 {
		token.Expiry = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	}
	if !token.Valid() {
		return nil, fmt.Errorf("token file %q contains an invalid or expired token", ts.path)
	}

	return token, nil
}

// NewTokenSourceFromTokenFile returns a TokenSource backed by a file
// containing an OAuth2 token response in JSON format, e.g.
// {"access_token": "...", "expires_in": 3600, "token_type": "Bearer"}.
// The file is read eagerly so that an invalid or expired token fails at mount
// time, and re-read when the token expires, so an external process may
// refresh the credential by rewriting the file.
func NewTokenSourceFromTokenFile(path string) (oauth2.TokenSource, error) {
	ts := fileTokenSource{path: path}
	token, err := ts.Token()
	if err != nil {
		return nil, err
	}
	return oauth2.ReuseTokenSource(token, ts), nil
}
