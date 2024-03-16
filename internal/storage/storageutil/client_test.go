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
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

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

func (t *clientTest) TestCreateTokenSrcWithCustomEndpoint() {
	url, err := url.Parse(CustomEndpoint)
	AssertEq(nil, err)
	sc := GetDefaultStorageClientConfig()
	sc.CustomEndpoint = url

	tokenSrc, err := createTokenSource(&sc)

	ExpectEq(nil, err)
	ExpectNe(nil, &tokenSrc)
}

func (t *clientTest) TestCreateTokenSrcWhenCustomEndpointIsNil() {
	sc := GetDefaultStorageClientConfig()
	sc.CustomEndpoint = nil

	// It will try to create the actual auth token and fail since key-file doesn't exist.
	tokenSrc, err := createTokenSource(&sc)

	ExpectNe(nil, err)
	ExpectThat(err, oglematchers.Error(oglematchers.HasSubstr("no such file or directory")))
	ExpectTrue(strings.Contains(fmt.Sprint(tokenSrc), strconv.FormatInt(int64(time.Second*10), 10)))
}

func (t *clientTest) TestCreateHttpClientWithHttp1() {
	sc := GetDefaultStorageClientConfig() // By default http1 enabled

	// Act: this method add tokenSource and clientOptions.
	httpClient, err := CreateHttpClient(&sc)

	ExpectEq(nil, err)
	ExpectNe(nil, httpClient)
	ExpectNe(nil, httpClient.Transport)
	t.validateProxyInTransport(httpClient)
	ExpectEq(sc.HttpClientTimeout, httpClient.Timeout)
}

func (t *clientTest) TestCreateHttpClientWithHttp2() {
	sc := GetDefaultStorageClientConfig()

	// Act: this method add tokenSource and clientOptions.
	httpClient, err := CreateHttpClient(&sc)

	ExpectEq(nil, err)
	ExpectNe(nil, httpClient)
	ExpectNe(nil, httpClient.Transport)
	t.validateProxyInTransport(httpClient)
	ExpectEq(sc.HttpClientTimeout, httpClient.Timeout)
}
