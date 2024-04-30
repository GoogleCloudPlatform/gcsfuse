// Copyright 2023 Google Inc. All Rights Reserved.
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
	"net/http"
	"testing"

	"github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"golang.org/x/oauth2"
)

func TestClient(t *testing.T) { RunTests(t) }

type clientTest struct {
}

func init() { RegisterTestSuite(&clientTest{}) }

// Helpers

func (t *clientTest) validateProxyInTransport(httpClient *http.Client) {
	userAgentRT, ok := httpClient.Transport.(*userAgentRoundTripper)
	AssertEq(true, ok)
	oauthTransport, ok := userAgentRT.wrapped.(*oauth2.Transport)
	AssertEq(true, ok)
	transport, ok := oauthTransport.Base.(*http.Transport)
	AssertEq(true, ok)
	if ok {
		ExpectEq(http.ProxyFromEnvironment, transport.Proxy)
	}
}

// Tests

func (t *clientTest) TestCreateHttpClientWithHttp1() {
	sc := GetDefaultStorageClientConfig() // By default http1 enabled

	httpClient, err := CreateHttpClient(&sc)

	ExpectEq(nil, err)
	ExpectNe(nil, httpClient)
	ExpectEq(sc.HttpClientTimeout, httpClient.Timeout)
}

func (t *clientTest) TestCreateHttpClientWithHttp2() {
	sc := GetDefaultStorageClientConfig()

	httpClient, err := CreateHttpClient(&sc)

	ExpectEq(nil, err)
	ExpectNe(nil, httpClient)
	ExpectEq(sc.HttpClientTimeout, httpClient.Timeout)
}

func (t *clientTest) TestCreateHttpClientWithHttp1AndAuthEnabled() {
	sc := GetDefaultStorageClientConfig() // By default http1 enabled
	sc.AnonymousAccess = false

	// Act: this method add tokenSource and clientOptions.
	httpClient, err := CreateHttpClient(&sc)

	AssertNe(nil, err)
	ExpectThat(err, oglematchers.Error(oglematchers.HasSubstr("no such file or directory")))
	AssertEq(nil, httpClient)
}

func (t *clientTest) TestCreateHttpClientWithHttp2AndAuthEnabled() {
	sc := GetDefaultStorageClientConfig()
	sc.AnonymousAccess = false
	// Act: this method add tokenSource and clientOptions.
	httpClient, err := CreateHttpClient(&sc)

	AssertNe(nil, err)
	ExpectThat(err, oglematchers.Error(oglematchers.HasSubstr("no such file or directory")))
	AssertEq(nil, httpClient)
}

func (t *clientTest) TestCreateTokenSrc() {
	sc := GetDefaultStorageClientConfig()

	tokenSrc, err := CreateTokenSource(&sc)

	AssertNe(nil, err)
	ExpectThat(err, oglematchers.Error(oglematchers.HasSubstr("no such file or directory")))
	ExpectNe(nil, &tokenSrc)
}

func (t *clientTest) TestStripScheme() {
	for _, tc := range []struct {
		input          string
		expectedOutput string
	}{
		{
			input:          "",
			expectedOutput: "",
		},
		{
			input:          "localhost:8080",
			expectedOutput: "localhost:8080",
		},
		{
			input:          "http://localhost:8888",
			expectedOutput: "localhost:8888",
		},
		{
			input:          "cp://localhost:8888",
			expectedOutput: "localhost:8888",
		},
		{
			input:          "bad://http://localhost:888://",
			expectedOutput: "http://localhost:888://",
		},
	} {
		output := StripScheme(tc.input)

		AssertEq(tc.expectedOutput, output)
	}
}
