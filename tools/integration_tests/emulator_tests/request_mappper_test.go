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
	"net/http/httptest"
	"os"
	"testing"
)

func TestDeduceRequestType(t *testing.T) {
	testCases := []struct {
		name         string
		method       string
		path         string
		expectedType RequestType
	}{
		{
			name:         "XML Read",
			method:       http.MethodGet,
			path:         "/buckets/my-bucket/objects/my-object",
			expectedType: XmlRead,
		},
		{
			name:         "JSON Stat",
			method:       http.MethodGet,
			path:         "/storage/v1/b/my-bucket/o/my-object",
			expectedType: JsonStat,
		},
		{
			name:         "JSON Create",
			method:       http.MethodPost,
			path:         "/storage/v1/b/my-bucket/o/my-object",
			expectedType: JsonCreate,
		},
		{
			name:         "JSON Update",
			method:       http.MethodPut,
			path:         "/storage/v1/b/my-bucket/o/my-object",
			expectedType: JsonUpdate,
		},
		{
			name:         "Delete Method",
			method:       http.MethodDelete, // Any method other than GET, POST, PUT for JSON API
			path:         "/storage/v1/b/my-bucket/o/my-object",
			expectedType: JsonDelete,
		},
		{
			name:         "Unknown Path Get method",
			method:       http.MethodGet,
			path:         "/unknown/path",
			expectedType: XmlRead,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.path, nil)
			requestType := deduceRequestType(r)
			if requestType != tc.expectedType {
				t.Errorf("Expected request type %v, but got %v", tc.expectedType, requestType)
			}
		})
	}
}

func TestAddRetryID(t *testing.T) {
	var err error
	gConfig, err = parseConfigFile("./testdata/config.yaml")
	log.Printf("%+v\n", gConfig)
	if err != nil {
		log.Printf("Parsing error: %v\n", err)
		os.Exit(1)
	}
	path := "/storage/v1/b/my-bucket/o/my-object"

	gOpManager = NewOperationManager(*gConfig)

	testCases := []struct {
		name        string
		requestType RequestType
		method      string
	}{
		{
			name:        "JSON Create",
			requestType: JsonCreate,
			method:      http.MethodPost,
		},
		{
			name:        "JSON Stat",
			requestType: JsonStat,
			method:      http.MethodGet,
		},
		{
			name:        "JSON Delete",
			requestType: JsonDelete,
			method:      http.MethodDelete,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, path, nil)
			err = handleRequest(r, tc.requestType)
			if err != nil {
				t.Fatalf("addRetryID failed: %v", err)
			}
			if r.Header.Get("x-retry-test-id") == "" {
				t.Errorf("Expected retry header")
			}
		})
	}
}
