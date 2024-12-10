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
	"fmt"
	"net/http"
	"net/url"
)

type emulatorTest struct {
	// host is the address of proxy server to which client interacts.
	host *url.URL
}

type RetryTestClient interface {
	GetRetryID(instructions map[string][]string, transport string) (string, error)
}

// GetRetryID creates a retry test resource in the emulator.
func (et *emulatorTest) GetRetryID(instructions map[string][]string, transport string) (string, error) {
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
		return "", fmt.Errorf("encoding request: %v\n", err)
	}

	et.host.Path = "retry_test"
	resp, err := c.Post(et.host.String(), "application/json", buf)
	if err != nil || resp.StatusCode != 200 {
		return "", fmt.Errorf("creating retry test: err: %v, resp: %+v\n", err, resp)
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
		return "", fmt.Errorf("decoding test ID: %v\n", err)
	}

	et.host.Path = ""
	return testRes.TestID, nil
}

// CreateRetryTest creates a retry test using the provided instructions.
//
// It takes emulator host (URL string) and a map of instructions. The instructions map
// specifies how the emulator should behave for specific requests.
//
// The keys of the `instructions` map are request paths, and the values are
// lists of strings representing actions to be taken for those requests.
// These actions can include things like returning specific error codes or
// simulating delays.
//
// Example `instructions` map:
//
//	{
//	  "storage.objects.list": []string{"return-503"},  // Return a 503 error for this request
//	  "storage.objects.get": []string{"stall-100ms", "return-200"}, // Delay for 100ms then return success
//	}
//
// The function returns a unique ID for the retry test, which can be used to
// identify and manage the test. It returns an error if there is a problem
// parsing the host URL or setting up the emulator test.
func CreateRetryTest(host string, instructions map[string][]string) (string, error) {
	if len(instructions) == 0 {
		return "", nil
	}

	endpoint, err := url.Parse(host)
	if err != nil {
		return "", fmt.Errorf("Failed to parse host env: %v\n", err)
	}

	et := &emulatorTest{host: endpoint}
	return et.GetRetryID(instructions, "http")
}
