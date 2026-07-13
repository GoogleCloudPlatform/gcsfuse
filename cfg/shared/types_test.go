// Copyright 2026 Google LLC
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

package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestBucketTypeListUnmarshalYAML(t *testing.T) {
	testCases := []struct {
		name     string
		yaml     string
		expected BucketTypeList
	}{
		{
			name:     "Scalar string single",
			yaml:     `bucket-type: "zonal"`,
			expected: BucketTypeList{"zonal"},
		},
		{
			name:     "Scalar string CSV",
			yaml:     `bucket-type: "zonal, pirlo"`,
			expected: BucketTypeList{"zonal", "pirlo"},
		},
		{
			name:     "Scalar string CSV with spaces",
			yaml:     `bucket-type: " zonal , pirlo , flat "`,
			expected: BucketTypeList{"zonal", "pirlo", "flat"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var bto BucketTypeOptimization
			err := yaml.Unmarshal([]byte(tc.yaml), &bto)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, bto.BucketTypes)
		})
	}
}

func TestBucketTypeListUnmarshalYAMLErrors(t *testing.T) {
	testCases := []struct {
		name                   string
		yaml                   string
		expectedErrorSubstring string
	}{
		{
			name:                   "Sequence of strings not allowed",
			yaml:                   `bucket-type: ["zonal", "pirlo"]`,
			expectedErrorSubstring: "bucket-type must be a string",
		},
		{
			name:                   "Mapping not allowed",
			yaml:                   `bucket-type: { key: value }`,
			expectedErrorSubstring: "bucket-type must be a string",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var bto BucketTypeOptimization
			err := yaml.Unmarshal([]byte(tc.yaml), &bto)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedErrorSubstring)
		})
	}
}
