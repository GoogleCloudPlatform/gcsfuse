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

type RequestTypeAndInstruction struct {
	RequestType RequestType
	Instruction string
}

// deduceRequestType determines the type of request and its corresponding instruction
func deduceRequestType(r *http.Request) RequestTypeAndInstruction {
	path := r.URL.Path
	method := r.Method

	if isJsonAPI(path) {
		return deduceJsonRequestType(method)
	}
	return deduceXmlRequestType(method)
}

// isJsonAPI checks if the request is targeting the JSON API
func isJsonAPI(path string) bool {
	return strings.Contains(path, "/storage/v1")
}

// deduceJsonRequestType determines the request type and instruction for JSON API requests
func deduceJsonRequestType(method string) RequestTypeAndInstruction {
	switch method {
	case http.MethodGet:
		return RequestTypeAndInstruction{JsonStat, "storage.objects.get"}
	case http.MethodPost:
		return RequestTypeAndInstruction{JsonCreate, "storage.objects.create"}
	case http.MethodDelete:
		return RequestTypeAndInstruction{JsonDelete, "storage.objects.delete"}
	case http.MethodPut:
		return RequestTypeAndInstruction{JsonUpdate, "storage.objects.update"}
	default:
		return RequestTypeAndInstruction{Unknown, ""}
	}
}

// deduceXmlRequestType determines the request type and instruction for XML API requests
func deduceXmlRequestType(method string) RequestTypeAndInstruction {
	switch method {
	case http.MethodGet:
		return RequestTypeAndInstruction{XmlRead, "storage.objects.get"}
	default:
		return RequestTypeAndInstruction{Unknown, ""}
	}
}
