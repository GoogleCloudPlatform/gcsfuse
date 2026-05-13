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

package gcs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBucketType_IsRapid(t *testing.T) {
	testCases := []struct {
		name     string
		zonal    bool
		pirlo    PirloState
		expected bool
	}{
		{
			name:     "Neither Zonal nor Pirlo",
			zonal:    false,
			pirlo:    PirloStateNone,
			expected: false,
		},
		{
			name:     "Only Zonal is true",
			zonal:    true,
			pirlo:    PirloStateNone,
			expected: true,
		},
		{
			name:     "Pirlo Rapid Enabled",
			zonal:    false,
			pirlo:    PirloStateRapidWritesEnabled,
			expected: true,
		},
		{
			name:     "Pirlo Rapid Disabled",
			zonal:    false,
			pirlo:    PirloStateRapidWritesDisabled,
			expected: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bt := BucketType{
				Zonal: tc.zonal,
				Pirlo: tc.pirlo,
			}

			assert.Equal(t, tc.expected, bt.IsRapid())
		})
	}
}

func TestBucketType_RapidWritesEnabled(t *testing.T) {
	testCases := []struct {
		name     string
		zonal    bool
		pirlo    PirloState
		expected bool
	}{
		{
			name:     "Neither Zonal nor Pirlo",
			zonal:    false,
			pirlo:    PirloStateNone,
			expected: false,
		},
		{
			name:     "Only Zonal is true",
			zonal:    true,
			pirlo:    PirloStateNone,
			expected: true,
		},
		{
			name:     "Pirlo Rapid Enabled",
			zonal:    false,
			pirlo:    PirloStateRapidWritesEnabled,
			expected: true,
		},
		{
			name:     "Pirlo Rapid Disabled",
			zonal:    false,
			pirlo:    PirloStateRapidWritesDisabled,
			expected: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bt := BucketType{
				Zonal: tc.zonal,
				Pirlo: tc.pirlo,
			}

			assert.Equal(t, tc.expected, bt.RapidWritesEnabled())
		})
	}
}
