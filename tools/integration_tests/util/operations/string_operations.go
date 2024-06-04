// Copyright 2024 Google Inc. All Rights Reserved.
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

// Provide a helper for string operations.
package operations

import (
	"strings"
	"testing"
)

func VerifyExpectedSubstrings(t *testing.T, input string, expectedSubstrings []string) {
	for _, expectedSubstring := range expectedSubstrings {
		if !strings.Contains(input, expectedSubstring) {
			t.Errorf("input does not contain expected substring (%q)", expectedSubstring)
		}
	}
}

func VerifyUnexpectedSubstrings(t *testing.T, input string, unexpectedSubstrings []string) {
	for _, unexpectedSubstring := range unexpectedSubstrings {
		if strings.Contains(input, unexpectedSubstring) {
			t.Errorf("input contains unexpected substring (%q)", unexpectedSubstring)
		}
	}
}
