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
	assert.Equal(t.T(), universeDomainDefault, domain)
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
}

func (t *AuthTest) Test_newTokenSourceFromPath_invalidPath() {
	ctx := context.Background()

	_, _, err := newTokenSourceFromPath(ctx, "nonexistent.json", storagev1.DevstorageFullControlScope)
	assert.Error(t.T(), err)
}

func (t *AuthTest) Test_newTokenSourceFromPath_invalidJSON() {
	ctx := context.Background()
	// Create a temp file with invalid JSON
	content := []byte(`{invalid-json}`)
	tmpFile := createTempFile(t.T(), content)
	defer cleanupTempFile(tmpFile)

	_, _, err := newTokenSourceFromPath(ctx, tmpFile, storagev1.DevstorageFullControlScope)

	assert.Error(t.T(), err)
}

func (t *AuthTest) Test_GetTokenSource_withKeyFile() {
	ctx := context.Background()

	tokenSrc, domain, err := GetTokenSource(ctx, "testdata/google_creds.json", "", false)

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), tokenSrc)
	assert.Equal(t.T(), universeDomainDefault, domain)
}

func (t *AuthTest) Test_GetTokenSource_defaultCredentials() {
	ctx := context.Background()

	tokenSrc, domain, err := GetTokenSource(ctx, "", "", false)

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), tokenSrc)
	assert.Equal(t.T(), universeDomainDefault, domain)
}

// --- Test helpers ---
func createTempFile(t *testing.T, content []byte) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "test*.json")
	assert.NoError(t, err)

	_, err = tmpFile.Write(content)
	assert.NoError(t, err)

	err = tmpFile.Close()
	assert.NoError(t, err)

	return tmpFile.Name()
}

func cleanupTempFile(path string) {
	_ = os.Remove(path)
}
