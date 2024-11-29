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
	"log"
	"net/http"
)

// MockRetryTestClient is a mock implementation of RetryTestClient.
type MockRetryTestClient struct {
	RetryTestClient
	// Add fields to store mock data and control mock behavior
	MockTestID      string
	InstructionsArg map[string][]string
	HostArg         string
}

// CreateRetryTest is the mock implementation of the CreateRetryTest method.
func (c *MockRetryTestClient) CreateRetryTest(host string, instructions map[string][]string) string {
	c.InstructionsArg = instructions
	c.HostArg = host
	return c.MockTestID
}

func (c *MockRetryTestClient) AddRetryID(r *http.Request, requestType RequestType, instruction string) error {
	plantOp := gOpManager.retrieveOperation(requestType)
	if plantOp != "" {
		testID := c.CreateRetryTest(gConfig.TargetHost, map[string][]string{instruction: {plantOp}})
		r.Header.Set("x-retry-test-id", testID)
	}
	return nil
}

func (c *MockRetryTestClient) HandleRequest(r *http.Request, requestType RequestType) error {
	switch requestType {
	case XmlRead, JsonStat:
		return c.AddRetryID(r, requestType, "storage.objects.get")
	case JsonCreate:
		return c.AddRetryID(r, requestType, "storage.objects.insert")
	case JsonDelete:
		return c.AddRetryID(r, requestType, "storage.objects.delete")
	case JsonList:
		return c.AddRetryID(r, requestType, "storage.buckets.list")
	default:
		log.Println("No handling for unknown operation")
		return nil
	}
}
