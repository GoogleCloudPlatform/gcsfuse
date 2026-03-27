// Copyright 2025 Google LLC
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

package gcsx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMinInt64(t *testing.T) {
	tests := []struct {
		name     string
		a        int64
		b        int64
		expected int64
	}{
		{
			name:     "a is less than b",
			a:        1,
			b:        5,
			expected: 1,
		},
		{
			name:     "a is greater than b",
			a:        5,
			b:        1,
			expected: 1,
		},
		{
			name:     "a is equal to b",
			a:        5,
			b:        5,
			expected: 5,
		},
		{
			name:     "a is negative and b is positive",
			a:        -5,
			b:        1,
			expected: -5,
		},
		{
			name:     "a is positive and b is negative",
			a:        1,
			b:        -5,
			expected: -5,
		},
		{
			name:     "a and b are both negative, a is less",
			a:        -10,
			b:        -5,
			expected: -10,
		},
		{
			name:     "a and b are both negative, a is greater",
			a:        -5,
			b:        -10,
			expected: -10,
		},
		{
			name:     "a is zero and b is positive",
			a:        0,
			b:        5,
			expected: 0,
		},
		{
			name:     "a is zero and b is negative",
			a:        0,
			b:        -5,
			expected: -5,
		},
		{
			name:     "a and b are both zero",
			a:        0,
			b:        0,
			expected: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, minInt64(tc.a, tc.b))
		})
	}
}
