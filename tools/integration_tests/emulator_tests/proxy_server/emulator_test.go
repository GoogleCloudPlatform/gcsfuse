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
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRetryID(t *testing.T) {
	instructions := map[string][]string{
		"storage.objects.get": {"error", "success"},
	}
	expectedTestID := "test-id-1234"

	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/retry_test", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Validate the request body
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		defer r.Body.Close()

		data := struct {
			Instructions map[string][]string `json:"instructions"`
			Transport    string              `json:"transport"`
		}{}
		err = json.Unmarshal(body, &data)
		assert.NoError(t, err)
		assert.Equal(t, instructions, data.Instructions)
		assert.Equal(t, "http", data.Transport)

		// Send a mock response
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(struct {
			TestID string `json:"id"`
		}{
			TestID: expectedTestID,
		})
		assert.NoError(t, err)
	}))
	defer server.Close()

	// Test GetRetryID
	host, _ := url.Parse(server.URL)
	emulator := &emulatorTest{host: host}

	testID := emulator.GetRetryID(instructions, "http")
	assert.Equal(t, expectedTestID, testID)
}

func TestCreateRetryTest(t *testing.T) {
	instructions := map[string][]string{
		"storage.objects.get": {"error", "success"},
	}
	expectedTestID := "test-id-1234"

	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/retry_test", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Send a mock response
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(struct {
			TestID string `json:"id"`
		}{
			TestID: expectedTestID,
		})
		assert.NoError(t, err)
	}))
	defer server.Close()

	// Test CreateRetryTest
	testID := CreateRetryTest(server.URL, instructions)
	assert.Equal(t, expectedTestID, testID)
}

func TestCreateRetryTestWithEmptyInstructions(t *testing.T) {
	// Test with empty instructions
	testID := CreateRetryTest("http://localhost", map[string][]string{})
	assert.Empty(t, testID)
}
