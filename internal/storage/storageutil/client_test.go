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
	"net/url"
	"testing"

	. "github.com/jacobsa/ogletest"
)

func TestClient(t *testing.T) { RunTests(t) }

type clientTest struct {
}

func init() { RegisterTestSuite(&clientTest{}) }

func (t *clientTest) TestCreateTokenSrcWithCustomEndpointWhenDisableAuthIsTrue() {
	url, err := url.Parse(CustomEndpoint)
	AssertEq(nil, err)
	sc := GetDefaultStorageClientConfig()
	sc.CustomEndpoint = url
	sc.DisableAuth = true

	tokenSrc, err := createTokenSource(&sc)

	ExpectEq(nil, err)
	ExpectNe(nil, &tokenSrc)
}

func (t *clientTest) TestCreateTokenSrcWhenCustomEndpointIsNilAndDisableAuthIsTrue() {
	sc := GetDefaultStorageClientConfig()
	sc.CustomEndpoint = nil
	sc.DisableAuth = true

	tokenSrc, err := createTokenSource(&sc)

	ExpectEq(nil, err)
	ExpectNe(nil, &tokenSrc)
}

func (t *clientTest) TestCreateHttpClientWithHttp1() {
	sc := GetDefaultStorageClientConfig() // By default http1 enabled
	sc.DisableAuth = true
	// Act: this method add tokenSource and clientOptions.
	httpClient, err := CreateHttpClient(&sc)

	ExpectEq(nil, err)
	ExpectNe(nil, httpClient)
	ExpectNe(nil, httpClient.Transport)
	ExpectEq(sc.HttpClientTimeout, httpClient.Timeout)
}

func (t *clientTest) TestCreateHttpClientWithHttp2() {
	sc := GetDefaultStorageClientConfig()
	sc.DisableAuth = true

	// Act: this method add tokenSource and clientOptions.
	httpClient, err := CreateHttpClient(&sc)

	ExpectEq(nil, err)
	ExpectNe(nil, httpClient)
	ExpectNe(nil, httpClient.Transport)
	ExpectEq(sc.HttpClientTimeout, httpClient.Timeout)
}
