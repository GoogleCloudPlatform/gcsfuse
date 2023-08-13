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

	"github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

func TestClientHelper(t *testing.T) { RunTests(t) }

type clientHelperTest struct {
}

func init() { RegisterTestSuite(&clientHelperTest{}) }

func (t *clientHelperTest) TestIsGCSProdHostnameWithCustomName() {
	url, err := url.Parse(CustomEndpoint)
	AssertEq(nil, err)

	res := IsProdEndpoint(url)

	ExpectFalse(res)
}

func (t *clientHelperTest) TestIsGCSProdHostnameEndpoint() {
	// GCSFuse assumes prod, if we specify endpoint as nil.
	var url *url.URL

	res := IsProdEndpoint(url)

	ExpectTrue(res)
}

func (t *clientHelperTest) TestCreateTokenSrcWithCustomEndpoint() {
	url, err := url.Parse(CustomEndpoint)
	AssertEq(nil, err)
	sc := GetDefaultStorageClientConfig()
	sc.Endpoint = url

	tokenSrc, err := createTokenSource(&sc)

	ExpectEq(nil, err)
	ExpectNe(nil, &tokenSrc)
}

func (t *clientHelperTest) TestCreateTokenSrcWithProdEndpoint() {
	sc := GetDefaultStorageClientConfig()
	sc.Endpoint = nil

	// It will try to create the actual auth token and fail since key-file doesn't exist.
	tokenSrc, err := createTokenSource(&sc)

	ExpectNe(nil, err)
	ExpectThat(err, oglematchers.Error(oglematchers.HasSubstr("no such file or directory")))
	ExpectEq(nil, tokenSrc)
}

func (t *clientHelperTest) TestCreateHttpClientObjWithHttp1() {
	sc := GetDefaultStorageClientConfig() // By default http1 enabled

	// Act: this method add tokenSource and clientOptions.
	httpClient, err := CreateHttpClientObj(&sc)

	ExpectEq(nil, err)
	ExpectNe(nil, httpClient)
	ExpectNe(nil, httpClient.Transport)
	ExpectEq(sc.HttpClientTimeout, httpClient.Timeout)
}

func (t *clientHelperTest) TestCreateHttpClientObjWithHttp2() {
	sc := GetDefaultStorageClientConfig()

	// Act: this method add tokenSource and clientOptions.
	httpClient, err := CreateHttpClientObj(&sc)

	ExpectEq(nil, err)
	ExpectNe(nil, httpClient)
	ExpectNe(nil, httpClient.Transport)
	ExpectEq(sc.HttpClientTimeout, httpClient.Timeout)
}
