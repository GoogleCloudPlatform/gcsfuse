// Copyright 2020 Google Inc. All Rights Reserved.
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
	"fmt"
	"io/ioutil"

	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Create token source from the JSON file at the supplide path.
func newTokenSourceFromKeyFile(
	ctx context.Context,
	path string,
	scope string) (ts oauth2.TokenSource, err error) {
	// Read the file.
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		err = fmt.Errorf("ReadFile(%q): %v", path, err)
		return
	}

	// Create a config struct based on its contents.
	jwtConfig, err := google.JWTConfigFromJSON(contents, scope)
	if err != nil {
		err = fmt.Errorf("JWTConfigFromJSON: %v", err)
		return
	}

	// Create the token source.
	ts = jwtConfig.TokenSource(ctx)

	return
}

// GetTokenSource returns a TokenSource for GCS API
// (1) given a key file, or
// (2) using an exchange token, or
// (3) with the default credentials.
func GetTokenSource(
	keyFile string,
	exchangeToken string,
) (tokenSrc oauth2.TokenSource, err error) {
	const scope = gcs.Scope_FullControl
	var authMethod string

	ctx := context.Background()
	if keyFile != "" {
		tokenSrc, err = newTokenSourceFromKeyFile(ctx, keyFile, scope)
		authMethod = "newTokenSourceFromKeyFile"
	} else if exchangeToken != "" {
		tokenSrc, err = newExchangeTokenSource(ctx, exchangeToken)
		authMethod = "newExchangeTokenSource"
	} else {
		tokenSrc, err = google.DefaultTokenSource(ctx, scope)
		authMethod = "DefaultTokenSource"
	}
	if err != nil {
		err = fmt.Errorf("%s: %v", authMethod, err)
		return
	}
	return
}
