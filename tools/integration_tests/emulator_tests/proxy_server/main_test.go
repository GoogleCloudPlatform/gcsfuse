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
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAddRetryID tests the AddRetryID function
func TestAddRetryID(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(map[string]string{"id": "test-id-123"})
		assert.NoError(t, err)
	}))
	defer mockServer.Close()

	gConfig = &Config{TargetHost: mockServer.URL}
	gOpManager = &OperationManager{
		retryConfigs: map[RequestType][]RetryConfig{
			"TestType": {{Method: "TestType", RetryInstruction: "retry-instruction", RetryCount: 1, SkipCount: 0}},
		},
	}

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	r := RequestTypeAndInstruction{
		RequestType: "TestType",
		Instruction: "retry",
	}

	err := AddRetryID(req, r)
	assert.NoError(t, err)
	assert.Equal(t, "test-id-123", req.Header.Get("x-retry-test-id"), "Unexpected x-retry-test-id header value")
}
