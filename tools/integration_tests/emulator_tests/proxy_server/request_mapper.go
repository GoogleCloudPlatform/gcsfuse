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
	"net/http"
	"strings"
)

type RequestType string

const (
	XmlRead     RequestType = "XmlRead"
	JsonStat    RequestType = "JsonStat"
	JsonDelete  RequestType = "JsonDelete"
	JsonUpdate  RequestType = "JsonUpdate"
	JsonCreate  RequestType = "JsonCreate"
	JsonCopy    RequestType = "JsonCopy"
	JsonList    RequestType = "JsonList"
	JsonCompose RequestType = "JsonCompose"
	Unknown     RequestType = "Unknown"
)

func deduceRequestType(r *http.Request) RequestType {
	path := r.URL.Path
	method := r.Method

	// Generic JSON API format:
	// https://storage.googleapis.com/storage/v1/b/)<bucket-name>/o/<object-name>
	if strings.Contains(path, "/storage/v1") {
		switch {
		case method == http.MethodGet:
			return JsonStat
		case method == http.MethodPost:
			return JsonCreate
		case method == http.MethodDelete:
			return JsonDelete
		case method == http.MethodPut:
			return JsonUpdate
		default:
			return Unknown
		}
	} else { // Assuming XML to start.
		switch {
		case method == http.MethodGet:
			return XmlRead
		default:
			return Unknown
		}
	}
}
