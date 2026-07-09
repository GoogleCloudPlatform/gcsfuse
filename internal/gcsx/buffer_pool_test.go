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

package gcsx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFixedSizeBufferPool_Get(t *testing.T) {
	// Arrange
	bp := NewFixedSizeBufferPool()

	// Act
	buf := bp.Get()

	// Assert
	assert.Equal(t, readPoolBufferSize, len(buf))
	assert.Equal(t, readPoolBufferSize, cap(buf))
}

func TestFixedSizeBufferPool_Put(t *testing.T) {
	testCases := []struct {
		name string
		buf  []byte
	}{
		{
			name: "ValidBuffer",
			buf:  make([]byte, readPoolBufferSize),
		},
		{
			name: "CapacityLessThanPoolSize",
			buf:  make([]byte, 10),
		},
		{
			name: "CapacityGreaterThanPoolSize",
			buf:  make([]byte, readPoolBufferSize+10),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			bp := NewFixedSizeBufferPool()

			// Act & Assert
			assert.NotPanics(t, func() {
				bp.Put(tc.buf)
			})
		})
	}
}
