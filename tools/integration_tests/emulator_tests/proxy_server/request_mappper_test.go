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
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeduceRequestTypeAndInstruction(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		path        string
		expectedReq RequestType
		expectedIns string
	}{
		// JSON API Tests
		{"JsonStat GET", http.MethodGet, "/storage/v1/bucket/object", JsonStat, "storage.objects.get"},
		{"JsonCreate POST", http.MethodPost, "/storage/v1/bucket/object", JsonCreate, "storage.objects.create"},
		{"JsonDelete DELETE", http.MethodDelete, "/storage/v1/bucket/object", JsonDelete, "storage.objects.delete"},
		{"JsonUpdate PUT", http.MethodPut, "/storage/v1/bucket/object", JsonUpdate, "storage.objects.update"},
		{"JsonUnknown PATCH", http.MethodPatch, "/storage/v1/bucket/object", Unknown, ""},

		// XML API Tests
		{"XmlRead GET", http.MethodGet, "/bucket/object", XmlRead, "storage.objects.get"},
		{"XmlUnknown POST", http.MethodPost, "/bucket/object", Unknown, ""},

		// Other Edge Cases
		{"UnknownPath GET", http.MethodGet, "/unknown/path", XmlRead, "storage.objects.get"},
		{"EmptyPath GET", http.MethodPost, "", Unknown, ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := &http.Request{
				Method: test.method,
				URL:    &url.URL{Path: test.path},
			}

			result := deduceRequestTypeAndInstruction(req)
			assert.Equal(t, test.expectedReq, result.RequestType)
			assert.Equal(t, test.expectedIns, result.Instruction)
		})
	}
}
