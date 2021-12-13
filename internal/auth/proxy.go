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
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"golang.org/x/oauth2"
)

// newProxyTokenSource returns a TokenSource that calls an external
// endpoint for authentication and access tokens.
func newProxyTokenSource(
	ctx context.Context,
	endpoint string,
) oauth2.TokenSource {
	ts := proxyTokenSource{
		ctx:      ctx,
		endpoint: endpoint,
		client:   &http.Client{},
	}
	return oauth2.ReuseTokenSource(nil, ts)
}

type proxyTokenSource struct {
	ctx      context.Context
	endpoint string
	client   *http.Client
}

func (ts proxyTokenSource) Token() (token *oauth2.Token, err error) {
	resp, err := ts.client.Get(ts.endpoint)
	if err != nil {
		err = fmt.Errorf("proxyTokenSource cannot fetch token: %w", err)
		return
	}

	body, err := ioutil.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		err = fmt.Errorf("proxyTokenSource cannot load body: %w", err)
		return
	}

	if c := resp.StatusCode; c < 200 || c >= 300 {
		err = &oauth2.RetrieveError{
			Response: resp,
			Body:     body,
		}
		return
	}

	token = &oauth2.Token{}
	err = json.Unmarshal(body, token)
	if err != nil {
		err = fmt.Errorf("proxyTokenSource cannot decode body: %w", err)
		return
	}

	return
}
