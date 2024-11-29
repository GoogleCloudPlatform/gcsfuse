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
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
)

// RetryTestClient is an interface for creating and managing retry tests.
type RetryTestClient interface {
	CreateRetryTest(host string, instructions map[string][]string) string
	AddRetryID(r *http.Request, requestType RequestType, instruction string) error
}

// DefaultRetryTestClient is the default implementation of RetryTestClient.
type DefaultRetryTestClient struct {
	RetryTestClient
}

func (c *DefaultRetryTestClient) HandleRequest(r *http.Request, requestType RequestType) error {
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

func (c *DefaultRetryTestClient) AddRetryID(r *http.Request, requestType RequestType, instruction string) error {
	plantOp := gOpManager.retrieveOperation(requestType)
	if plantOp != "" {
		testID := c.CreateRetryTest(gConfig.TargetHost, map[string][]string{instruction: {plantOp}})
		r.Header.Set("x-retry-test-id", testID)
	}
	return nil
}

// CreateRetryTest creates a retry test using the provided instructions.
func (c *DefaultRetryTestClient) CreateRetryTest(host string, instructions map[string][]string) string {
	if len(instructions) == 0 {
		return ""
	}

	endpoint, err := url.Parse(host)
	if err != nil {
		log.Printf("Failed to parse host env: %v\n", err)
		os.Exit(0)
	}

	et := &emulatorTest{host: endpoint}
	return et.getRetryID(instructions, "http")
}

type emulatorTest struct {
	id   string   // ID to pass as a header in the test execution
	host *url.URL // set the path when using; path is not guaranteed between calls
}

// Create creates a retry test resource in the emulator.
func (et *emulatorTest) getRetryID(instructions map[string][]string, transport string) string {
	c := http.DefaultClient
	data := struct {
		Instructions map[string][]string `json:"instructions"`
		Transport    string              `json:"transport"`
	}{
		Instructions: instructions,
		Transport:    transport,
	}

	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(data); err != nil {
		log.Printf("encoding request: %v\n", err)
		os.Exit(0) // Consider returning an error instead of exiting
	}

	et.host.Path = "retry_test"
	resp, err := c.Post(et.host.String(), "application/json", buf)
	if err != nil || resp.StatusCode != 200 {
		log.Printf("creating retry test: err: %v, resp: %+v\n", err, resp)
		os.Exit(0) // Consider returning an error instead of exiting
	}
	defer func() {
		closeErr := resp.Body.Close()
		if err == nil {
			err = closeErr
		}
	}()

	testRes := struct {
		TestID string `json:"id"`
	}{}
	if err := json.NewDecoder(resp.Body).Decode(&testRes); err != nil {
		log.Printf("decoding test ID: %v\n", err)
		os.Exit(0) // Consider returning an error instead of exiting
	}

	et.id = testRes.TestID
	et.host.Path = ""
	return et.id
}
