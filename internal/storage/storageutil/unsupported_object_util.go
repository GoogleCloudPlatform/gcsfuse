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

package storageutil

import (
	"slices"
	"strings"
)

var (
	unsupportedObjectNameSubstrings = []string{"//", "..", "."}
	unsupportedObjectNamePrefixes   = []string{"/"}
	unsupportedObjectNames          = []string{""}
)

// IsUnsupportedObjectName returns true if the passed
// string is a valid GCS object name or prefix,
// which is unsupported in GCSFuse.
func IsUnsupportedObjectName(name string) bool {
	for _, substring := range unsupportedObjectNameSubstrings {
		if strings.Contains(name, substring) {
			return true
		}
	}
	for _, prefix := range unsupportedObjectNamePrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return slices.Contains(unsupportedObjectNames, name)
}
