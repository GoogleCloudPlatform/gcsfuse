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

package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
)

// createRetryTest creates a bucket in the emulator and sets up a test using the
// Retry Test API for the given instructions. This is intended for emulator tests
// of retry behavior that are not covered by conformance tests.
func CreateRetryTest(host string, instructions map[string][]string) string {
	if len(instructions) == 0 {
		return ""
	}

	endpoint, err := url.Parse(host)
	if err != nil {
		fmt.Printf("Failed to parse host env: %v\n", err)
		os.Exit(0)
	}

	et := emulatorTest{name: "test", host: endpoint}
	et.create(instructions, "http")
	return et.id
}

type emulatorTest struct {
	name string
	id   string   // ID to pass as a header in the test execution
	host *url.URL // set the path when using; path is not guaranteed between calls
}

// Creates a retry test resource in the emulator
func (et *emulatorTest) create(instructions map[string][]string, transport string) {
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
		fmt.Printf("encoding request: %v\n", err)
	}

	et.host.Path = "retry_test"
	resp, err := c.Post(et.host.String(), "application/json", buf)
	if err != nil || resp.StatusCode != 200 {
		fmt.Printf("creating retry test: err: %v, resp: %+v\n", err, resp)
		os.Exit(0)
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
		fmt.Printf("decoding test ID: %v\n", err)
	}

	et.id = testRes.TestID
	et.host.Path = ""
}
