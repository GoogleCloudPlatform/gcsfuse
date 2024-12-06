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

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
	"github.com/stretchr/testify/assert"
)

// TestGetRetryID tests the GetRetryID function
func TestGetRetryID(t *testing.T) {
	setup.IgnoreTestIfIntegrationTestFlagIsSet(t)

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/retry_test", r.URL.Path, "Unexpected URL path")
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(map[string]string{"id": "test-id-123"})
		assert.NoError(t, err)
	}))
	defer mockServer.Close()
	hostURL, _ := url.Parse(mockServer.URL)
	et := &emulatorTest{host: hostURL}
	instructions := map[string][]string{"retry": {"retry-instruction"}}

	testID, err := et.GetRetryID(instructions, "http")

	assert.NoError(t, err)
	assert.Equal(t, "test-id-123", testID, "Unexpected test ID returned")
}

// TestCreateRetryTest tests the CreateRetryTest function
func TestCreateRetryTest(t *testing.T) {
	setup.IgnoreTestIfIntegrationTestFlagIsSet(t)

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/retry_test", r.URL.Path, "Unexpected URL path")
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(map[string]string{"id": "test-id-123"})
		assert.NoError(t, err)
	}))
	defer mockServer.Close()

	instructions := map[string][]string{"retry": {"retry-instruction"}}
	testID, err := CreateRetryTest(mockServer.URL, instructions)
	assert.NoError(t, err)
	assert.Equal(t, "test-id-123", testID, "Unexpected test ID returned")

	// Test with empty instructions
	testID, err = CreateRetryTest(mockServer.URL, map[string][]string{})
	assert.NoError(t, err)
	assert.Equal(t, "", testID, "Expected empty test ID for empty instructions")
}
