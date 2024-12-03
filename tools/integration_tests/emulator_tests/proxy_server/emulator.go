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

type emulatorTest struct {
	host *url.URL // set the path when using; path is not guaranteed between calls
}

type RetryTestClient interface {
	GetRetryID(instructions map[string][]string, transport string) string
}

// Create creates a retry test resource in the emulator.
func (et *emulatorTest) GetRetryID(instructions map[string][]string, transport string) string {
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

	et.host.Path = ""
	return testRes.TestID
}

// CreateRetryTest creates a retry test using the provided instructions.
func CreateRetryTest(host string, instructions map[string][]string) string {
	if len(instructions) == 0 {
		return ""
	}

	endpoint, err := url.Parse(host)
	if err != nil {
		log.Printf("Failed to parse host env: %v\n", err)
		os.Exit(0)
	}

	et := &emulatorTest{host: endpoint}
	return et.GetRetryID(instructions, "http")
}

func AddRetryID(req *http.Request, r RequestTypeAndInstruction) {
	plantOp := gOpManager.retrieveOperation(r.RequestType)
	if plantOp != "" {
		testID := CreateRetryTest(gConfig.TargetHost, map[string][]string{r.Instruction: {plantOp}})
		req.Header.Set("x-retry-test-id", testID)
	}
}
