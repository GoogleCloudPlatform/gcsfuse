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

package proxy_server

import (
	"testing"
)

func TestOperationManager(t *testing.T) {
	config := Config{
		RetryConfig: []RetryConfig{
			{Method: "XmlRead", RetryCount: 2, RetryInstruction: "retry_GET"},
			{Method: "JsonStat", SkipCount: 1, RetryCount: 1, RetryInstruction: "retry_STAT"},
		},
	}
	om := NewOperationManager(config)

	// Test GET operation
	if op := om.retrieveOperation("XmlRead"); op != "retry_GET" {
		t.Errorf("Expected 'retry_GET', got '%s'", op)
	}
	if op := om.retrieveOperation("XmlRead"); op != "retry_GET" {
		t.Errorf("Expected 'retry_GET', got '%s'", op)
	}
	if op := om.retrieveOperation("XmlRead"); op != "" {
		t.Errorf("Expected '', got '%s'", op)
	}

	// Test JsonStat operation
	if op := om.retrieveOperation("JsonStat"); op != "" {
		t.Errorf("Expected '', got '%s'", op)
	}
	if op := om.retrieveOperation("JsonStat"); op != "retry_STAT" {
		t.Errorf("Expected 'retry_POST', got '%s'", op)
	}
	if op := om.retrieveOperation("JsonStat"); op != "" {
		t.Errorf("Expected '', got '%s'", op)
	}

	// Test non-existent operation
	if op := om.retrieveOperation("JsonPut"); op != "" {
		t.Errorf("Expected '', got '%s'", op)
	}
}
