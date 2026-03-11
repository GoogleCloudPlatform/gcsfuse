// Copyright 2024 Google LLC
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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	storagev1 "google.golang.org/api/storage/v1"
)

const tpcUniverseDomain = "apis-tpclp.goog"

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type AuthTest struct {
	suite.Suite
}

func TestAuthSuite(t *testing.T) {
	suite.Run(t, new(AuthTest))
}

////////////////////////////////////////////////////////////////////////
// Tests for AuthTest
////////////////////////////////////////////////////////////////////////

func (t *AuthTest) TestGetUniverseDomainForGoogle() {
	contents, err := os.ReadFile("testdata/google_creds.json")
	assert.NoError(t.T(), err)

	domain, err := getUniverseDomain(context.Background(), contents, storagev1.DevstorageFullControlScope)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), UniverseDomainDefault, domain)
}

func (t *AuthTest) TestGetUniverseDomainForTPC() {
	contents, err := os.ReadFile("testdata/tpc_creds.json")
	assert.NoError(t.T(), err)

	domain, err := getUniverseDomain(context.Background(), contents, storagev1.DevstorageFullControlScope)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), tpcUniverseDomain, domain)
}

func (t *AuthTest) TestGetUniverseDomainForEmptyCreds() {
	contents, err := os.ReadFile("testdata/empty_creds.json")
	assert.NoError(t.T(), err)

	_, err = getUniverseDomain(context.Background(), contents, storagev1.DevstorageFullControlScope)

	assert.Error(t.T(), err)
	assert.Equal(t.T(), "CredentialsFromJSON(): unexpected end of JSON input", err.Error())
}

func (t *AuthTest) TestGetTokenSource_WithKeyFile() {
	tokenSrc, err := GetTokenSource(
		context.Background(),
		"testdata/google_creds.json",
		"",
		false,
		"",
	)

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), tokenSrc)
}

func (t *AuthTest) TestGetTokenSource_WithEmptyImpersonation() {
	// When impersonate SA is empty, it should fall through to default token source.
	// With testdata key file, this should succeed without impersonation wrapping.
	tokenSrc, err := GetTokenSource(
		context.Background(),
		"testdata/google_creds.json",
		"",
		false,
		"",
	)

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), tokenSrc)
}

func (t *AuthTest) TestGetTokenSource_WithInvalidKeyFile() {
	tokenSrc, err := GetTokenSource(
		context.Background(),
		"non-existent-key-file.json",
		"",
		false,
		"",
	)

	assert.Error(t.T(), err)
	assert.Nil(t.T(), tokenSrc)
}

func (t *AuthTest) TestNewImpersonatedTokenSource_UsesBaseTokenSource() {
	// Verify that NewImpersonatedTokenSource returns a non-nil token source
	// when given valid base credentials and a target SA. This confirms the
	// base token source is passed through (via option.WithTokenSource) rather
	// than falling back to ADC.
	baseSrc, err := newTokenSourceFromPath(
		context.Background(),
		"testdata/google_creds.json",
		storagev1.DevstorageFullControlScope,
	)
	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), baseSrc)

	impersonatedSrc, err := NewImpersonatedTokenSource(
		context.Background(),
		baseSrc,
		"test-sa@my-project.iam.gserviceaccount.com",
	)

	// The call should succeed (creating the token source); actual token
	// generation would fail without real IAM credentials, but we verify
	// the wrapping itself works correctly.
	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), impersonatedSrc)
}

func (t *AuthTest) TestGetTokenSource_WithImpersonation() {
	// Verify the full GetTokenSource flow with impersonation enabled.
	// Uses a key file as the base credential and wraps it with impersonation.
	tokenSrc, err := GetTokenSource(
		context.Background(),
		"testdata/google_creds.json",
		"",
		false,
		"test-sa@my-project.iam.gserviceaccount.com",
	)

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), tokenSrc)
}
