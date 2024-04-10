// Copyright 2024 Google Inc. All Rights Reserved.
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
	"os"
	"testing"

	. "github.com/jacobsa/ogletest"
	storagev1 "google.golang.org/api/storage/v1"
)

const tpcUniverseDomain = "apis-tpclp.goog"

func TestAuth(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type AuthTest struct {
}

func init() {
	RegisterTestSuite(&AuthTest{})
}

func (t *AuthTest) SetUp(ti *TestInfo) {
}

////////////////////////////////////////////////////////////////////////
// Tests for AuthTest
////////////////////////////////////////////////////////////////////////

func (t *AuthTest) TestGetUniverseDomainForGoogle() {
	path := "testdata/google_creds.json"
	contents, err := os.ReadFile(path)
	AssertEq(nil, err)
	ctx := context.Background()

	domain, err := getUniverseDomain(ctx, contents, storagev1.DevstorageFullControlScope)

	ExpectEq(nil, err)
	ExpectEq(universeDomainDefault, domain)
}

func (t *AuthTest) TestGetUniverseDomainForTPC() {
	path := "testdata/tpc_creds.json"
	contents, err := os.ReadFile(path)
	AssertEq(nil, err)
	ctx := context.Background()

	domain, err := getUniverseDomain(ctx, contents, storagev1.DevstorageFullControlScope)

	ExpectEq(nil, err)
	ExpectEq(tpcUniverseDomain, domain)
}

func (t *AuthTest) TestGetUniverseDomainForEmptyCreds() {
	path := "testdata/empty_creds.json"
	contents, err := os.ReadFile(path)
	AssertEq(nil, err)
	ctx := context.Background()

	_, err = getUniverseDomain(ctx, contents, storagev1.DevstorageFullControlScope)

	ExpectNe(nil, err)
}
